package poll

import (
	"sort"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/internal/policy"
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

func (j *WatchRepositoryTagsJob) computeEvents(tags []string) ([]types.Event, error) {
	trackedImages, err := j.providers.TrackedImages()
	if err != nil {
		return nil, err
	}

	events := []types.Event{}

	// collapse removes all non-semver tags and only takes
	// the highest versions of each prerelease + the main version that doesn't have
	// any prereleases
	tags = collapse(tags)

	for _, trackedImage := range getRelatedTrackedImages(j.details.trackedImage, trackedImages) {
		// matches, going through tags
		for _, tag := range tags {
			update, err := trackedImage.Policy.ShouldUpdate(trackedImage.Image.Tag(), tag)
			if err != nil {
				continue
			}
			if update && !exists(tag, events) {
				event := types.Event{
					Repository: types.Repository{
						Name: j.details.trackedImage.Image.Repository(),
						Tag:  tag,
					},
					TriggerName: types.TriggerTypePoll.String(),
				}
				events = append(events, event)
			}

		}

	}

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

// collapse gets latest available tags for main version and pre-releases
// example:
// [1.0.0, 1.5.0, 1.3.0-dev, 1.4.5-dev] would become [1.5.0, 1.4.5-dev]
func collapse(tags []string) []string {
	r := map[string]string{}
	p := policy.NewSemverPolicy(policy.SemverPolicyTypeAll)
	for _, t := range tags {
		v, err := version.GetVersion(t)
		// v, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		stored, ok := r[v.PreRelease]
		if !ok {
			r[v.PreRelease] = t
			continue
		}
		higher, err := p.ShouldUpdate(stored, t)
		if err != nil {
			continue
		}
		if higher {
			r[v.PreRelease] = t
		}
	}

	result := []string{}
	for _, tag := range r {
		result = append(result, tag)
	}

	// always sort, for test purposes
	sort.Strings(result)

	return result
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
