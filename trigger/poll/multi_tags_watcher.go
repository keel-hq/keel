package poll

import (
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

	// This contains all tracked images that share the same repository path
	allRelatedTrackedImages := getRelatedTrackedImages(j.details.trackedImage, trackedImages)

	for _, trackedImage := range allRelatedTrackedImages {

		filteredTags := tags

		// The fact that they are related, does not mean they share the exact same Policy configuration, so wee need
		// to calculate the tags here for each image.
		if j.details.trackedImage.Policy != nil {
			filteredTags = j.details.trackedImage.Policy.Filter(tags)
		}

		for _, tag := range filteredTags {

			update, err := trackedImage.Policy.ShouldUpdate(trackedImage.Image.Tag(), tag)
			if err != nil {
				continue
			}
			if update == false {
				continue
			}
			if trackedImage.Image.Tag() == tag {
				break
			}
			if !exists(tag, events) {
				event := types.Event{
					Repository: types.Repository{
						Name: trackedImage.Image.Repository(),
						Tag:  tag,
					},
					TriggerName: types.TriggerTypePoll.String(),
				}
				events = append(events, event)
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
