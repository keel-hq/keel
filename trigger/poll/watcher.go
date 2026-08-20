package poll

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/keel-hq/keel/util/image"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
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
	mu           sync.RWMutex
	// forceUpdateConsumed tracks whether the current keel.sh/force-update
	// request has already been acted on, so an explicit request triggers a
	// rolling restart at most once until it is cleared (and re-armed) by the
	// provider.
	forceUpdateConsumed bool
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
		<-ctx.Done()
		w.cron.Stop()
	}()
}

// This identifier is used to key the watchers, so that only a watcher
// is setup per identifier
func getImageIdentifier(ref *image.Reference, keepTag bool) string {
	if keepTag == true {
		return ref.Registry() + "/" + ref.ShortName() + ":" + ref.Tag()
	}
	return ref.Registry() + "/" + ref.ShortName()
}

// Unwatch - stop watching for changes
func (w *RepositoryWatcher) Unwatch(imageIdentifier string) error {
	_, ok := w.watched[imageIdentifier]
	if ok {
		w.cron.DeleteJob(imageIdentifier)
		delete(w.watched, imageIdentifier)
	}
	return nil
}

// Watch - starts watching repository for changes, if it's already watching - ignores,
// if details changed - updates details
func (w *RepositoryWatcher) Watch(images ...*types.TrackedImage) error {

	var errs []string
	tracked := map[string]bool{}

	// a watcher can be shared by several workloads, any one of them running a
	// stale digest is drift the watcher has to report, so the digests are kept
	// grouped per workload rather than merged into one set
	running := map[string][][]string{}
	for _, image := range images {
		if image.Trigger != types.TriggerTypePoll || len(image.RunningDigests) == 0 {
			continue
		}
		key := getImageIdentifier(image.Image, image.Policy.KeepTag())
		running[key] = append(running[key], image.RunningDigests)
	}

	for _, image := range images {
		if image.Trigger != types.TriggerTypePoll {
			continue
		}
		identifier, err := w.watch(image, running[getImageIdentifier(image.Image, image.Policy.KeepTag())])
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
			}).Info("trigger.poll.RepositoryWatcher: image no longer tracked, removing watcher")
			w.cron.DeleteJob(key)
			delete(w.watched, key)
		}
	}
}

func (w *RepositoryWatcher) watch(image *types.TrackedImage, runningDigests [][]string) (string, error) {

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

	key := getImageIdentifier(image.Image, image.Policy.KeepTag())

	// checking whether it's already being watched
	details, ok := w.watched[key]
	if !ok {
		err = w.addJob(image, image.PollSchedule, runningDigests)
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
	// todo: this is not right, we are using the last seen schedule, which might not be the most frequent
	// the most frequent schedule should be used for the shared watcher
	if details.schedule != image.PollSchedule {
		err := w.cron.UpdateJob(key, image.PollSchedule)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"image": image.String(),
			}).Error("trigger.poll.RepositoryWatcher.Watch: failed to update image watch job")
		}
	}

	details.mu.Lock()
	details.trackedImage = image
	// reset the force-update latch once the keel.sh/force-update annotation has
	// been cleared, so a fresh request is picked up again
	if !image.ForceUpdate {
		details.forceUpdateConsumed = false
	}
	// setting main latest version to the lowest from the tracked
	details.latest = version.Lowest(details.trackedImage.Tags)
	details.mu.Unlock()

	// nothing to do
	return key, nil
}

// runtimeBaseline reports a digest that a workload is running when no workload
// sharing this watcher runs the digest the tag resolves to. It is deliberately
// conservative: a workload that runs the current digest on at least one replica
// is mid-rollout, not stale, and the per-platform manifests of a multi-arch
// image index never equal the index digest a registry reports.
func (w *RepositoryWatcher) runtimeBaseline(ti *types.TrackedImage, workloads [][]string, registryOpts registry.Opts, digest string) (string, bool) {
	known := map[string]bool{digest: true}
	var candidates [][]string
	for _, workload := range workloads {
		if len(workload) > 0 && !matchesAny(workload, known) {
			candidates = append(candidates, workload)
		}
	}
	if len(candidates) == 0 {
		return "", false
	}

	// the tag may resolve to an image index whose children are what nodes run
	tagDigests, err := w.registryClient.Digests(registryOpts)
	if err != nil {
		// without the full digest set a mismatch cannot be told apart from a
		// multi-arch image, so keep the registry digest as the baseline
		log.WithFields(log.Fields{
			"error": err,
			"image": ti.Image.String(),
		}).Warn("trigger.poll.RepositoryWatcher.addJob: failed to resolve tag manifest digests, cannot check for runtime drift")
		return "", false
	}
	for _, tagDigest := range tagDigests {
		known[tagDigest] = true
	}

	for _, workload := range candidates {
		if !matchesAny(workload, known) {
			return workload[0], true
		}
	}
	return "", false
}

func matchesAny(digests []string, known map[string]bool) bool {
	for _, d := range digests {
		if known[d] {
			return true
		}
	}
	return false
}

func (w *RepositoryWatcher) addJob(ti *types.TrackedImage, schedule string, runningDigests [][]string) error {
	// getting initial digest
	reg := ti.Image.Scheme() + "://" + ti.Image.Registry()

	registryOpts := registry.Opts{
		Registry: reg,
		Name:     ti.Image.ShortName(),
		Tag:      ti.Image.Tag(),
	}

	creds, err := credentialshelper.GetCredentials(ti)
	if err == nil {
		registryOpts.Username = creds.Username
		registryOpts.Password = creds.Password
	}

	digest, err := w.registryClient.Digest(registryOpts)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"image":    ti.Image.String(),
			"username": registryOpts.Username,
			"password": strings.Repeat("*", len(registryOpts.Password)),
		}).Error("trigger.poll.RepositoryWatcher.addJob: failed to get image digest")
		return err
	}

	key := getImageIdentifier(ti.Image, ti.Policy.KeepTag())

	// The baseline is what the workload is running, not what the registry
	// serves. Seeding it with the registry digest hides drift that happened
	// while Keel was not watching (the watcher state is in-memory only, so
	// every restart starts from scratch) - https://github.com/keel-hq/keel/issues/845
	baseline := digest
	if ti.Policy.KeepTag() {
		if running, ok := w.runtimeBaseline(ti, runningDigests, registryOpts, digest); ok {
			log.WithFields(log.Fields{
				"image":           ti.Image.String(),
				"registry_digest": digest,
				"running_digests": runningDigests,
			}).Info("trigger.poll.RepositoryWatcher.addJob: workload is not running the current tag digest, seeding watcher with the running digest")
			baseline = running
		}
	}

	details := &watchDetails{
		trackedImage: ti,
		digest:       baseline, // digest the workload is known to be running
		latest:       ti.Image.Tag(),
		schedule:     schedule,
	}

	// adding job to internal map
	w.watched[key] = details

	// read the docs several times, the only legit case when want a tag watcher
	// is when policy is force and keel.sh/match-tag=true.
	if ti.Policy.KeepTag() {
		// adding new job
		job := NewWatchTagJob(w.providers, w.registryClient, details)
		log.WithFields(log.Fields{
			"job_name": key,
			"image":    ti.Image.String(),
			"digest":   baseline,
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
		"image":    ti.Image.Registry() + "/" + ti.Image.ShortName(), // A watcher can be shared, so it makes little sense to specify tag depth here
		"digest":   "",                                               // A watcher can be shared, so it makes little sense to specify here a specific image digest used by one of the consumers
		"schedule": schedule,
	}).Info("trigger.poll.RepositoryWatcher: new watch repository tags job added")
	job.Run()
	return w.cron.AddJob(key, schedule, job)
}
