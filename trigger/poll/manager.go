package poll

import (
	"context"
	"sync"
	"time"

	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

var pollTriggerTrackedImages = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "poll_trigger_tracked_images",
		Help: "How many images are tracked by poll trigger",
	},
)

func init() {
	prometheus.MustRegister(pollTriggerTrackedImages)
}

// DefaultManager - default manager is responsible for scanning deployments and identifying
// deployments that have market
type DefaultManager struct {
	providers provider.Providers

	// repository watcher
	watcher Watcher

	mu *sync.Mutex

	// scanTick - scan interval in seconds, defaults to 60 seconds
	scanTick int

	// root context
	ctx context.Context
}

// NewPollManager - new default poller
func NewPollManager(providers provider.Providers, watcher Watcher) *DefaultManager {
	return &DefaultManager{
		providers: providers,
		watcher:   watcher,
		mu:        &sync.Mutex{},
		scanTick:  1,
	}
}

// Start - start scanning deployment for changes
func (s *DefaultManager) Start(ctx context.Context) error {
	// setting root context
	s.ctx = ctx

	log.Info("trigger.poll.manager: polling trigger configured")

	// initial scan
	err := s.scan(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.poll.manager: scan failed")
	}

	ticker := time.NewTicker(time.Duration(s.scanTick) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := s.scan(ctx)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("trigger.poll.manager: kubernetes scan failed")
			}
		}
	}
}

func (s *DefaultManager) scan(ctx context.Context) error {
	log.Debug("trigger.poll.manager: performing scan")
	trackedImages, err := s.providers.TrackedImages()
	if err != nil {
		return err
	}

	var tracked float64
	for _, trackedImage := range trackedImages {
		if trackedImage.Trigger != types.TriggerTypePoll {
			continue
		}
		tracked++
		err = s.watcher.Watch(trackedImage, trackedImage.PollSchedule)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"schedule": trackedImage.PollSchedule,
				"image":    trackedImage.Image.Remote(),
			}).Error("trigger.poll.manager: failed to start watching repository")
		}
	}

	pollTriggerTrackedImages.Set(tracked)

	return nil
}
