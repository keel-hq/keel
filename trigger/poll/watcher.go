package poll

import (
	"context"
	"fmt"
	"strings"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"github.com/keel-hq/keel/util/version"
	"github.com/rusenask/cron"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

var registriesScannedCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "registries_scanned_total",
		Help: "How many registries where checked for new images, partitioned by registry and image.",
	},
	[]string{"registry", "image"},
)

var pollTriggerTrackedImages = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "poll_trigger_tracked_images",
		Help: "How many images are tracked by poll trigger",
	},
)

func init() {
	prometheus.MustRegister(registriesScannedCounter)
	prometheus.MustRegister(pollTriggerTrackedImages)
}

// Watcher - generic watcher interface
type Watcher interface {
	Watch(image ...*types.TrackedImage) error
	Unwatch(image string) error
}

type watchDetails struct {
	trackedImage *types.TrackedImage
	digest       string // image digest
	latest       string // latest tag
	schedule     string
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
		select {
		case <-ctx.Done():
			w.cron.Stop()
			return
		}
	}()
}

func getImageIdentifier(ref *image.Reference) string {
	_, err := version.GetVersion(ref.Tag())
	// if failed to parse version, will need to watch digest
	if err != nil {
		return ref.Registry() + "/" + ref.ShortName() + ":" + ref.Tag()
	}

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
func (w *RepositoryWatcher) Watch(images ...*types.TrackedImage) error {

	var errs []string
	tracked := map[string]bool{}

	for _, image := range images {
		if image.Trigger != types.TriggerTypePoll {
			continue
		}
		identifier, err := w.watch(image)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		tracked[identifier] = true
	}

	pollTriggerTrackedImages.Set(float64(len(tracked)))

	// removing registries that should not be tracked anymore
	// for example: deployment using image X was deleted so we should not query
	// registry that points to image X as nothing is using it anymore
	w.unwatch(tracked)

	if len(errs) > 0 {
		return fmt.Errorf("encountered errors while adding images: %s", strings.Join(errs, ", "))
	}

	return nil
}

func (w *RepositoryWatcher) unwatch(tracked map[string]bool) {
	for key, details := range w.watched {
		if !tracked[key] {
			log.WithFields(log.Fields{
				"job_name": key,
				"image":    details.trackedImage.String(),
				"schedule": details.schedule,
			}).Info("trigger.poll.RepositoryWatcher: image no tracked anymore, removing watcher")
			w.cron.DeleteJob(key)
			delete(w.watched, key)
		}
	}
}

func (w *RepositoryWatcher) watch(image *types.TrackedImage) (string, error) {

	if image.PollSchedule == "" {
		return "", fmt.Errorf("cron schedule cannot be empty")
	}

	_, err := cron.Parse(image.PollSchedule)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"image":    image.String(),
			"schedule": image.PollSchedule,
		}).Error("trigger.poll.RepositoryWatcher.addJob: invalid cron schedule")
		return "", fmt.Errorf("invalid cron schedule: %s", err)
	}

	key := getImageIdentifier(image.Image)

	// checking whether it's already being watched
	details, ok := w.watched[key]
	if !ok {
		// err = w.addJob(imageRef, registryUsername, registryPassword, schedule)
		err = w.addJob(image, image.PollSchedule)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"image": image.String(),
			}).Error("trigger.poll.RepositoryWatcher.Watch: failed to add image watch job")
			return "", err
		}
		return key, nil
	}

	// checking schedule
	if details.schedule != image.PollSchedule {
		w.cron.UpdateJob(key, image.PollSchedule)
	}

	// nothing to do

	return key, nil
}

func (w *RepositoryWatcher) addJob(ti *types.TrackedImage, schedule string) error {
	// getting initial digest
	reg := ti.Image.Scheme() + "://" + ti.Image.Registry()

	creds := credentialshelper.GetCredentials(ti)

	digest, err := w.registryClient.Digest(registry.Opts{
		Registry: reg,
		Name:     ti.Image.ShortName(),
		Tag:      ti.Image.Tag(),
		Username: creds.Username,
		Password: creds.Password,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": ti.Image.String(),
		}).Error("trigger.poll.RepositoryWatcher.addJob: failed to get image digest")
		return err
	}

	key := getImageIdentifier(ti.Image)
	details := &watchDetails{
		trackedImage: ti,
		digest:       digest, // current image digest
		latest:       ti.Image.Tag(),
		schedule:     schedule,
	}

	// adding job to internal map
	w.watched[key] = details

	// checking tag type, for versioned (semver) tags we setup a watch all tags job
	// and for non-semver types we create a single tag watcher which
	// checks digest
	_, err = version.GetVersion(ti.Image.Tag())
	if err != nil {
		// adding new job
		job := NewWatchTagJob(w.providers, w.registryClient, details)
		log.WithFields(log.Fields{
			"job_name": key,
			"image":    ti.Image.String(),
			"digest":   digest,
			"schedule": schedule,
		}).Info("trigger.poll.RepositoryWatcher: new watch tag digest job added")

		// running it now
		job.Run()

		return w.cron.AddJob(key, schedule, job)
	}

	// adding new job
	job := NewWatchRepositoryTagsJob(w.providers, w.registryClient, details)
	log.WithFields(log.Fields{
		"job_name": key,
		"image":    ti.Image.String(),
		"digest":   digest,
		"schedule": schedule,
	}).Info("trigger.poll.RepositoryWatcher: new watch repository tags job added")

	// running it now
	job.Run()

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
	creds := credentialshelper.GetCredentials(j.details.trackedImage)
	reg := j.details.trackedImage.Image.Scheme() + "://" + j.details.trackedImage.Image.Registry()
	currentDigest, err := j.registryClient.Digest(registry.Opts{
		Registry: reg,
		Name:     j.details.trackedImage.Image.ShortName(),
		Tag:      j.details.trackedImage.Image.Tag(),
		Username: creds.Username,
		Password: creds.Password,
	})

	registriesScannedCounter.With(prometheus.Labels{"registry": j.details.trackedImage.Image.Registry(), "image": j.details.trackedImage.Image.Repository()}).Inc()

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": j.details.trackedImage.Image.String(),
		}).Error("trigger.poll.WatchTagJob: failed to check digest")
		return
	}

	log.WithFields(log.Fields{
		"current_digest": j.details.digest,
		"new_digest":     currentDigest,
		"image":          j.details.trackedImage.Image.String(),
	}).Debug("trigger.poll.WatchTagJob: checking digest")

	// checking whether image digest has changed
	if j.details.digest != currentDigest {
		// updating digest
		j.details.digest = currentDigest

		event := types.Event{
			Repository: types.Repository{
				Name:   j.details.trackedImage.Image.Repository(),
				Tag:    j.details.trackedImage.Image.Tag(),
				Digest: currentDigest,
			},
			TriggerName: types.TriggerTypePoll.String(),
		}
		log.WithFields(log.Fields{
			"image":      j.details.trackedImage.Image.String(),
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

	latestVersion, newAvailable, err := version.NewAvailable(j.details.latest, repository.Tags)
	if err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"repository_tags": repository.Tags,
			// "image":           j.details.imageRef.Remote(),
			"image": j.details.trackedImage.Image.String(),
		}).Error("trigger.poll.WatchRepositoryTagsJob: failed to get latest version from tags")
		return
	}

	log.Debugf("new tag '%s' available", latestVersion)

	if newAvailable {
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
}
