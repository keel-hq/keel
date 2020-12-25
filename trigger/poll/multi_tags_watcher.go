package poll

import (
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

// WatchRepositoryTagsJob - watch all tags
type WatchRepositoryTagsJob struct {
	providers      provider.Providers
	registryClient registry.Client
	details        *watchDetails

	// latests map[string]string // a map of prerelease tags and their corresponding latest versions
}

// NewWatchRepositoryTagsJob - new tags watcher job
func NewWatchRepositoryTagsJob(providers provider.Providers, registryClient registry.Client, details *watchDetails) *WatchRepositoryTagsJob {
	return &WatchRepositoryTagsJob{
		providers:      providers,
		registryClient: registryClient,
		details:        details,
		// latests:        details.trackedImage.SemverPreReleaseTags,
	}
}

// Run - main function to check schedule
func (j *WatchRepositoryTagsJob) Run() {
	j.details.mu.RLock()
	defer j.details.mu.RUnlock()

	reg := j.details.trackedImage.Image.Scheme() + "://" + j.details.trackedImage.Image.Registry()
	if j.details.latest == "" {
		j.details.latest = j.details.trackedImage.Image.Tag()
	}

	registryOpts := registry.Opts{
		Registry: reg,
		Name:     j.details.trackedImage.Image.ShortName(),
		Tag:      j.details.latest,
	}

	creds, err := credentialshelper.GetCredentials(j.details.trackedImage)
	if err == nil {
		registryOpts.Username = creds.Username
		registryOpts.Password = creds.Password
	}

	repository, err := j.registryClient.Get(registryOpts)

	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"registry_url": reg,
			"image":        j.details.trackedImage.Image.String(),
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

func (j *WatchRepositoryTagsJob) computeEvents(tags []string) ([]types.Event, error) {
	trackedImages, err := j.providers.TrackedImages()
	if err != nil {
		return nil, err
	}

	events := []types.Event{}

	// Keep only semver tags, sorted desc (to optimize process)
	versions := semverSort(tags)

	for _, trackedImage := range getRelatedTrackedImages(j.details.trackedImage, trackedImages) {
		// Current version tag might not be a valid semver one
		currentVersion, invalidCurrentVersion := semver.NewVersion(trackedImage.Image.Tag())
		// matches, going through tags
		for _, version := range versions {
			if invalidCurrentVersion == nil && (currentVersion.GreaterThan(version) || currentVersion.Equal(version)) {
				// Current tag is a valid semver, and is bigger than currently tested one
				// -> we can stop now, nothing will be worth upgrading in the rest of the sorted list
				break
			}
			update, err := trackedImage.Policy.ShouldUpdate(trackedImage.Image.Tag(), version.Original())
			// log.WithFields(log.Fields{
			// 	"current_tag": j.details.trackedImage.Image.Tag(),
			// 	"image_name":  j.details.trackedImage.Image.Remote(),
			// }).Debug("trigger.poll.WatchRepositoryTagsJob: tag: ", version.Original(), "; update: ", update, "; err:", err)
			if err != nil {
				continue
			}
			if update && !exists(version.Original(), events) {
				event := types.Event{
					Repository: types.Repository{
						Name: j.details.trackedImage.Image.Repository(),
						Tag:  version.Original(),
					},
					TriggerName: types.TriggerTypePoll.String(),
				}
				events = append(events, event)
				// Only keep first match per image (should be the highest usable version)
				break
			}

		}

	}
	log.WithFields(log.Fields{
		"current_tag": j.details.trackedImage.Image.Tag(),
		"image_name":  j.details.trackedImage.Image.Remote(),
	}).Debug("trigger.poll.WatchRepositoryTagsJob: events: ", events)

	return events, nil
}

func exists(tag string, events []types.Event) bool {
	for _, e := range events {
		if tag == e.Repository.Tag {
			return true
		}
	}
	return false
}

// Filter and sort tags according to semver, desc
func semverSort(tags []string) []*semver.Version {
	var versions []*semver.Version
	for _, t := range tags {
		if len(strings.SplitN(t, ".", 3)) < 2 {
			// Keep only X.Y.Z+ semver
			continue
		}
		v, err := semver.NewVersion(t)
		// Filter out non semver tags
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}
	// Sort desc, following semver
	sort.Slice(versions, func(i, j int) bool { return versions[j].LessThan(versions[i]) })
	return versions
}

func getRelatedTrackedImages(ours *types.TrackedImage, all []*types.TrackedImage) []*types.TrackedImage {
	b := all[:0]
	for _, x := range all {
		if x.Image.Repository() == ours.Image.Repository() {
			b = append(b, x)
		}
	}

	return b
}

func (j *WatchRepositoryTagsJob) processTags(tags []string) error {

	events, err := j.computeEvents(tags)
	if err != nil {
		return err
	}
	for _, e := range events {
		err = j.providers.Submit(e)
		if err != nil {
			log.WithFields(log.Fields{
				"repository": j.details.trackedImage.Image.Repository(),
				"new_tag":    e.Repository.Tag,
				"error":      err,
			}).Error("trigger.poll.WatchRepositoryTagsJob: error while submitting an event")
		}
	}
	return nil
}
