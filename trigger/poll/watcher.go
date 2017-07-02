package poll

import (
	"context"

	"github.com/rusenask/cron"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/registry"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"

	log "github.com/Sirupsen/logrus"
)

type Watcher interface {
	Watch(imageName, registryUsername, registryPassword, schedule string) error
	Unwatch(image string) error
}

type watchDetails struct {
	imageRef         *image.Reference
	registryUsername string // "" for anonymous
	registryPassword string // "" for anonymous
	digest           string // image digest
	schedule         string
}

// RepositoryWatcher - repository watcher cron
type RepositoryWatcher struct {
	providers provider.Providers

	// registry client
	registryClient registry.Client

	// internal map of internal watches
	// map[registry/name]=image.Reference
	watched map[string]*watchDetails

	cron *cron.Cron
}

// NewRepositoryWatcher - create new repository watcher
func NewRepositoryWatcher(providers provider.Providers, registryClient registry.Client) *RepositoryWatcher {
	c := cron.New()

	return &RepositoryWatcher{
		providers:      providers,
		registryClient: registryClient,
		watched:        make(map[string]*watchDetails),
		cron:           c,
	}
}

func (w *RepositoryWatcher) Start(ctx context.Context) {
	// starting cron job
	w.cron.Start()
	go func() {
		for {
			select {
			case <-ctx.Done():
				w.cron.Stop()
			}
		}
	}()
}

func getImageIdentifier(ref *image.Reference) string {
	return ref.Registry() + "/" + ref.ShortName()
}

// Unwatch - stop watching for changes
func (w *RepositoryWatcher) Unwatch(imageName string) error {
	imageRef, err := image.Parse(imageName)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"image_name": imageName,
		}).Error("trigger.poll.RepositoryWatcher.Unwatch: failed to parse image")
		return err
	}
	key := getImageIdentifier(imageRef)
	_, ok := w.watched[key]
	if ok {
		w.cron.DeleteJob(key)
		delete(w.watched, key)
	}

	return nil
}

// Watch - starts watching repository for changes, if it's already watching - ignores,
// if details changed - updates details
func (w *RepositoryWatcher) Watch(imageName, schedule, registryUsername, registryPassword string) error {

	imageRef, err := image.Parse(imageName)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"image_name": imageName,
		}).Error("trigger.poll.RepositoryWatcher.Watch: failed to parse image")
		return err
	}

	key := getImageIdentifier(imageRef)

	// checking whether it's already being watched
	details, ok := w.watched[key]
	if !ok {
		err = w.addJob(imageRef, registryUsername, registryPassword, schedule)
		if err != nil {
			log.WithFields(log.Fields{
				"error":             err,
				"image_name":        imageName,
				"registry_username": registryUsername,
			}).Error("trigger.poll.RepositoryWatcher.Watch: failed to add image watch job")

		}
		return err
	}

	// checking schedule
	if details.schedule != schedule {
		w.cron.UpdateJob(key, schedule)
	}

	// checking auth details, if changed - need to update
	if details.registryPassword != registryPassword || details.registryUsername != registryUsername {
		// recreating job
		w.cron.DeleteJob(key)
		err = w.addJob(imageRef, registryUsername, registryPassword, schedule)
		if err != nil {
			log.WithFields(log.Fields{
				"error":             err,
				"image_name":        imageName,
				"registry_username": registryUsername,
			}).Error("trigger.poll.RepositoryWatcher.Watch: failed to add image watch job")
		}
		return err
	}

	// nothing to do

	return nil
}

func (w *RepositoryWatcher) addJob(ref *image.Reference, registryUsername, registryPassword, schedule string) error {
	// getting initial digest
	reg := ref.Scheme() + "://" + ref.Registry()

	digest, err := w.registryClient.Digest(registry.Opts{
		Registry: reg,
		Name:     ref.ShortName(),
		Tag:      ref.Tag(),
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": ref.Remote(),
		}).Error("trigger.poll.RepositoryWatcher.addJob: failed to get image digest")
		return err
	}

	key := getImageIdentifier(ref)
	details := &watchDetails{
		imageRef:         ref,
		digest:           digest, // current image digest
		registryUsername: registryUsername,
		registryPassword: registryPassword,
		schedule:         schedule,
	}

	// adding job to internal map
	w.watched[key] = details

	// adding new job
	job := NewWatchTagJob(w.providers, w.registryClient, details)
	log.WithFields(log.Fields{
		"job_name": key,
		"image":    ref.Remote(),
		"digest":   digest,
		"schedule": schedule,
	}).Info("trigger.poll.RepositoryWatcher: new job added")
	return w.cron.AddJob(key, schedule, job)

}

// WatchTagJob - Watch specific tag job
type WatchTagJob struct {
	providers      provider.Providers
	registryClient registry.Client
	details        *watchDetails
}

// NewWatchTagJob - new watch tag job monitors specific tag by checking digest based on specified
// cron style schedule
func NewWatchTagJob(providers provider.Providers, registryClient registry.Client, details *watchDetails) *WatchTagJob {
	return &WatchTagJob{
		providers:      providers,
		registryClient: registryClient,
		details:        details,
	}
}

// Run - main function to check schedule
func (j *WatchTagJob) Run() {
	reg := j.details.imageRef.Scheme() + "://" + j.details.imageRef.Registry()
	currentDigest, err := j.registryClient.Digest(registry.Opts{
		Registry: reg,
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

	log.WithFields(log.Fields{
		"current_digest": j.details.digest,
		"new_digest":     currentDigest,
		"image_name":     j.details.imageRef.Remote(),
	}).Info("trigger.poll.WatchTagJob: checking digest")

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
		log.Info("trigger.poll.WatchTagJob: digest change detected, submiting event to providers")

		j.providers.Submit(event)

	}
}
