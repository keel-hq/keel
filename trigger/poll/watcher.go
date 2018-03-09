package poll

import (
	"context"
	"fmt"

	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"github.com/keel-hq/keel/util/version"
	"github.com/rusenask/cron"

	log "github.com/sirupsen/logrus"
)

// Watcher - generic watcher interface
type Watcher interface {
	Watch(imageName, registryUsername, registryPassword, schedule string) error
	Unwatch(image string) error
}

type watchDetails struct {
	imageRef         *image.Reference
	registryUsername string // "" for anonymous
	registryPassword string // "" for anonymous
	digest           string // image digest
	latest           string // latest tag
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

// Start - starts repository watcher
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

	if schedule == "" {
		return fmt.Errorf("cron schedule cannot be empty")
	}

	_, err := cron.Parse(schedule)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"image":    imageName,
			"schedule": schedule,
		}).Error("trigger.poll.RepositoryWatcher.addJob: invalid cron schedule")
		return fmt.Errorf("invalid cron schedule: %s", err)
	}

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
		Username: registryUsername,
		Password: registryPassword,
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
		latest:           ref.Tag(),
		registryUsername: registryUsername,
		registryPassword: registryPassword,
		schedule:         schedule,
	}

	// adding job to internal map
	w.watched[key] = details

	// checking tag type, for versioned (semver) tags we setup a watch all tags job
	// and for non-semver types we create a single tag watcher which
	// checks digest
	_, err = version.GetVersion(ref.Tag())
	if err != nil {
		// adding new job
		job := NewWatchTagJob(w.providers, w.registryClient, details)
		log.WithFields(log.Fields{
			"job_name": key,
			"image":    ref.Remote(),
			"digest":   digest,
			"schedule": schedule,
		}).Info("trigger.poll.RepositoryWatcher: new watch tag digest job added")
		return w.cron.AddJob(key, schedule, job)
	}

	// adding new job
	job := NewWatchRepositoryTagsJob(w.providers, w.registryClient, details)
	log.WithFields(log.Fields{
		"job_name": key,
		"image":    ref.Remote(),
		"digest":   digest,
		"schedule": schedule,
	}).Info("trigger.poll.RepositoryWatcher: new watch repository tags job added")
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
		Username: j.details.registryUsername,
		Password: j.details.registryPassword,
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
	}).Debug("trigger.poll.WatchTagJob: checking digest")

	// checking whether image digest has changed
	if j.details.digest != currentDigest {
		// updating digest
		j.details.digest = currentDigest

		event := types.Event{
			Repository: types.Repository{
				Name:   j.details.imageRef.Repository(),
				Tag:    j.details.imageRef.Tag(),
				Digest: currentDigest,
			},
			TriggerName: types.TriggerTypePoll.String(),
		}
		log.WithFields(log.Fields{
			"repository": j.details.imageRef.Repository(),
			"new_digest": currentDigest,
		}).Info("trigger.poll.WatchTagJob: digest change detected, submiting event to providers")

		j.providers.Submit(event)

	}
}

// WatchRepositoryTagsJob - watch all tags
type WatchRepositoryTagsJob struct {
	providers      provider.Providers
	registryClient registry.Client
	details        *watchDetails
}

// NewWatchRepositoryTagsJob - new tags watcher job
func NewWatchRepositoryTagsJob(providers provider.Providers, registryClient registry.Client, details *watchDetails) *WatchRepositoryTagsJob {
	return &WatchRepositoryTagsJob{
		providers:      providers,
		registryClient: registryClient,
		details:        details,
	}
}

// Run - main function to check schedule
func (j *WatchRepositoryTagsJob) Run() {
	reg := j.details.imageRef.Scheme() + "://" + j.details.imageRef.Registry()

	if j.details.latest == "" {
		j.details.latest = j.details.imageRef.Tag()
	}

	repository, err := j.registryClient.Get(registry.Opts{
		Registry: reg,
		Name:     j.details.imageRef.ShortName(),
		Tag:      j.details.latest,
		Username: j.details.registryUsername,
		Password: j.details.registryPassword,
	})

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": j.details.imageRef.Remote(),
		}).Error("trigger.poll.WatchRepositoryTagsJob: failed to get repository")
		return
	}

	log.WithFields(log.Fields{
		"current_tag":     j.details.imageRef.Tag(),
		"repository_tags": repository.Tags,
		"image_name":      j.details.imageRef.Remote(),
	}).Debug("trigger.poll.WatchRepositoryTagsJob: checking tags")

	latestVersion, newAvailable, err := version.NewAvailable(j.details.latest, repository.Tags)
	if err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"repository_tags": repository.Tags,
			"image":           j.details.imageRef.Remote(),
		}).Error("trigger.poll.WatchRepositoryTagsJob: failed to get latest version from tags")
		return
	}

	log.Debugf("new tag '%s' available", latestVersion)

	if newAvailable {
		// updating current latest
		j.details.latest = latestVersion
		event := types.Event{
			Repository: types.Repository{
				Name: j.details.imageRef.Repository(),
				Tag:  latestVersion,
			},
			TriggerName: types.TriggerTypePoll.String(),
		}
		log.WithFields(log.Fields{
			"repository": j.details.imageRef.Repository(),
			"new_tag":    latestVersion,
		}).Info("trigger.poll.WatchRepositoryTagsJob: submiting event to providers")
		j.providers.Submit(event)
	}
}
