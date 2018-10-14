package poll

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/version"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// WatchRepositoryTagsJob - watch all tags
type WatchRepositoryTagsJob struct {
	providers      provider.Providers
	registryClient registry.Client
	details        *watchDetails
	latests        map[string]string // a map of prerelease tags and their corresponding latest versions
}

// NewWatchRepositoryTagsJob - new tags watcher job
func NewWatchRepositoryTagsJob(providers provider.Providers, registryClient registry.Client, details *watchDetails) *WatchRepositoryTagsJob {
	return &WatchRepositoryTagsJob{
		providers:      providers,
		registryClient: registryClient,
		details:        details,
		latests:        details.trackedImage.SemverPreReleaseTags,
	}
}

// Run - main function to check schedule
func (j *WatchRepositoryTagsJob) Run() {
	j.details.mu.RLock()
	defer j.details.mu.RUnlock()

	creds := credentialshelper.GetCredentials(j.details.trackedImage)

	reg := j.details.trackedImage.Image.Scheme() + "://" + j.details.trackedImage.Image.Registry()
	if j.details.latest == "" {
		j.details.latest = j.details.trackedImage.Image.Tag()
	}

	repository, err := j.registryClient.Get(registry.Opts{
		Registry: reg,
		Name:     j.details.trackedImage.Image.ShortName(),
		Tag:      j.details.latest,
		Username: creds.Username,
		Password: creds.Password,
	})

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": j.details.trackedImage.Image.String(),
		}).Error("trigger.poll.WatchRepositoryTagsJob: failed to get repository")
		return
	}

	registriesScannedCounter.With(prometheus.Labels{"registry": j.details.trackedImage.Image.Registry(), "image": j.details.trackedImage.Image.Repository()}).Inc()

	log.WithFields(log.Fields{
		"current_tag":     j.details.trackedImage.Image.Tag(),
		"repository_tags": repository.Tags,
		"image_name":      j.details.trackedImage.Image.Remote(),
	}).Debug("trigger.poll.WatchRepositoryTagsJob: checking tags")

	err = j.processTags(repository.Tags)
	if err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"repository_tags": repository.Tags,
			"image":           j.details.trackedImage.Image.String(),
		}).Error("trigger.poll.WatchRepositoryTagsJob: failed to process tags")
		return
	}
}

func (j *WatchRepositoryTagsJob) processTags(tags []string) error {
	var errors []string

	latestVersion, newAvailable, err := version.NewAvailable(j.details.latest, tags, false)
	if err != nil {
		errors = append(errors, fmt.Sprintf("new available version func returned an error: %s", err))
	}

	if newAvailable {
		log.Debugf("new tag '%s' available", latestVersion)
		// updating current latest
		j.details.latest = latestVersion
		event := types.Event{
			Repository: types.Repository{
				Name: j.details.trackedImage.Image.Repository(),
				Tag:  latestVersion,
			},
			TriggerName: types.TriggerTypePoll.String(),
		}
		log.WithFields(log.Fields{
			"repository": j.details.trackedImage.Image.Repository(),
			"new_tag":    latestVersion,
		}).Info("trigger.poll.WatchRepositoryTagsJob: submiting event to providers")
		j.providers.Submit(event)
	}

	for _, tag := range tags {
		sv, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		if sv.Prerelease() == "" {
			continue
		}
		trackedVersion, ok := j.details.trackedImage.SemverPreReleaseTags[sv.Prerelease()]
		if ok {
			latestVersion, newAvailable, err := version.NewAvailable(trackedVersion, tags, true)
			if err != nil {
				errors = append(errors, fmt.Sprintf("new available version func for tag %s returned an error: %s", trackedVersion, err))
				continue
			}

			if newAvailable {
				// log.Debugf("new tag with prerelease '%s' available", latestVersion)
				// updating current latest
				// j.details.latest = latestVersion
				j.details.trackedImage.SemverPreReleaseTags[sv.Prerelease()] = latestVersion

				event := types.Event{
					Repository: types.Repository{
						Name: j.details.trackedImage.Image.Repository(),
						Tag:  latestVersion,
					},
					TriggerName: types.TriggerTypePoll.String(),
				}
				log.WithFields(log.Fields{
					"repository": j.details.trackedImage.Image.Repository(),
					"new_tag":    latestVersion,
				}).Info("trigger.poll.WatchRepositoryTagsJob: submiting event to providers")
				j.providers.Submit(event)
			}

			if err != nil {
				errors = append(errors, fmt.Sprintf("new available version func returned an error: %s", err))
			}
		}
		// nothing to do
	}

	// for k, v := range j.details.trackedImage.SemverPreReleaseTags {

	// }

	if len(errors) > 0 {
		return fmt.Errorf("errors while processing repository tags: %s", strings.Join(errors, ","))
	}
	return nil
}
