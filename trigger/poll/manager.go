package poll

import (
	"context"
	"time"

	"github.com/keel-hq/keel/provider"

	log "github.com/sirupsen/logrus"
)

// DefaultScanInterval controls how often providers are re-enumerated. Registry
// checks keep their own per-image schedules; this interval only discovers
// changes to the set of tracked workloads.
const DefaultScanInterval = time.Minute

// DefaultManager - default manager is responsible for scanning deployments and identifying
// deployments that have market
type DefaultManager struct {
	providers provider.Providers

	// repository watcher
	watcher Watcher

	// scanInterval is the interval between provider re-enumerations.
	scanInterval time.Duration

	// root context
	ctx context.Context
}

// NewPollManager - new default poller
func NewPollManager(providers provider.Providers, watcher Watcher, intervals ...time.Duration) *DefaultManager {
	scanInterval := DefaultScanInterval
	if len(intervals) > 0 && intervals[0] > 0 {
		scanInterval = intervals[0]
	}
	return &DefaultManager{
		providers:    providers,
		watcher:      watcher,
		scanInterval: scanInterval,
	}
}

// Start - start scanning deployment for changes
func (s *DefaultManager) Start(ctx context.Context) error {
	// setting root context
	s.ctx = ctx

	log.WithField("scan_interval", s.scanInterval).Info("trigger.poll.manager: polling trigger configured")

	// initial scan
	err := s.scan(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.poll.manager: scan failed")
	}

	ticker := time.NewTicker(s.scanInterval)
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
	trackedImages, err := s.providers.TrackedImages()
	if err != nil {
		return err
	}

	err = s.watcher.Watch(trackedImages...)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.poll.manager: got error(-s) while watching images")
	}

	return nil
}
