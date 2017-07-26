package poll

import (
	"context"
	"sync"
	"time"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
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
		scanTick:  55,
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

	for _ = range time.Tick(time.Duration(s.scanTick) * time.Second) {
		select {
		case <-ctx.Done():
			return nil
		default:
			log.Debug("performing scan")
			err := s.scan(ctx)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("trigger.poll.manager: kubernetes scan failed")
			}
		}
	}

	return nil
}

func (s *DefaultManager) scan(ctx context.Context) error {
	trackedImages, err := s.providers.TrackedImages()
	if err != nil {
		return err
	}

	for _, trackedImage := range trackedImages {
		if trackedImage.Trigger != types.TriggerTypePoll {
			continue
		}

		err := s.watcher.Watch(trackedImage.Image.Remote(), trackedImage.PollSchedule, "", "")
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"schedule": trackedImage.PollSchedule,
				"image":    trackedImage.Image.Remote(),
			}).Error("trigger.poll.manager: failed to start watching repository")
			return err
		}
	}
	return nil
}
