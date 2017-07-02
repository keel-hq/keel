package poll

import (
	"github.com/rusenask/cron"
	"github.com/rusenask/keel/image"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
)

type Watcher interface {
	Watch(image string) error
	Unwatch(image string) error
	List() ([]types.Repository, error)
}

type RepositoryWatcher struct {
	providers provider.Providers

	cron *cron.Cron
}
// Watch specific tag job
type WatchTagJob struct {
	providers      provider.Providers
	registryClient registry.Client
	details        *watchDetails
}

func NewWatchTagJob(providers provider.Providers, registryClient registry.Client, details *watchDetails) *WatchTagJob {
	return &WatchTagJob{
		providers:      providers,
		registryClient: registryClient,
		details:        details,
	}
}

func (j *WatchTagJob) Run() {
	currentDigest, err := j.registryClient.Digest(registry.Opts{
		Registry: j.details.imageRef.Registry(),
		Name:     j.details.imageRef.ShortName(),
		Tag:      j.details.imageRef.Tag(),
	})

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": j.details.imageRef.Remote(),
		}).Error("trigger.poll.WatchTagJob: failed to check digest")
		return
	}

	// checking whether image digest has changed
	if j.details.digest != currentDigest {
		// updating digest
		j.details.digest = currentDigest

		event := types.Event{
			Repository: types.Repository{
				Name:   j.details.imageRef.Remote(),
				Tag:    j.details.imageRef.Tag(),
				Digest: currentDigest,
			},
			TriggerName: types.TriggerTypePoll.String(),
		}

		j.providers.Submit(event)

	}
}
