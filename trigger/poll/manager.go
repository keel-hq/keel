package poll

import (
	"context"
	"sync"
	"time"

	"github.com/keel-hq/keel/provider"

	log "github.com/sirupsen/logrus"
)

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
		scanTick:  3,
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

	err = s.watcher.Watch(trackedImages...)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.poll.manager: got error(-s) while watching images")
	}

	return nil
}
