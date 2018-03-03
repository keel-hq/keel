package pubsub

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/keel-hq/keel/provider"

	log "github.com/sirupsen/logrus"
)

// DefaultManager - subscription manager
type DefaultManager struct {
	providers provider.Providers

	client Subscriber
	// existing subscribers
	mu *sync.Mutex
	// a map of GCR URIs and subscribers to those URIs
	// i.e. key could be something like: gcr.io%2Fmy-project
	subscribers map[string]context.Context

	// projectID is required to correctly set GCR subscriptions
	projectID string

	// scanTick - scan interval in seconds, defaults to 60 seconds
	scanTick int

	// root context
	ctx context.Context
}

// Subscriber - subscribe is responsible to listen for repository events and
// inform providers
type Subscriber interface {
	Subscribe(ctx context.Context, topic, subscription string) error
}

// NewDefaultManager - creates new pubsub manager to create subscription for deployments
func NewDefaultManager(projectID string, providers provider.Providers, subClient Subscriber) *DefaultManager {
	return &DefaultManager{
		providers:   providers,
		client:      subClient,
		projectID:   projectID,
		subscribers: make(map[string]context.Context),
		mu:          &sync.Mutex{},
		scanTick:    60,
	}
}

// Start - start scanning deployment for changes
func (s *DefaultManager) Start(ctx context.Context) error {
	// setting root context
	s.ctx = ctx

	// initial scan
	err := s.scan(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.pubsub.manager: scan failed")
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
				}).Error("trigger.pubsub.manager: scan failed")
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
		if !isGoogleContainerRegistry(trackedImage.Image.Registry()) {
			log.Debug("registry %s is not a GCR, skipping", trackedImage.Image.Registry())
			continue
		}

		// uri
		gcrURI := containerRegistryURI(s.projectID, trackedImage.Image.Registry())
		s.ensureSubscription(gcrURI)
	}
	return nil
}

func (s *DefaultManager) subscribed(gcrURI string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.subscribers[gcrURI]
	return ok
}

func (s *DefaultManager) ensureSubscription(gcrURI string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.subscribers[gcrURI]
	if !ok {
		ctx, cancel := context.WithCancel(s.ctx)
		s.subscribers[gcrURI] = ctx
		subName := containerRegistrySubName(s.projectID, gcrURI)
		go func() {
			defer cancel()
			err := s.client.Subscribe(s.ctx, gcrURI, subName)
			if err != nil {
				log.WithFields(log.Fields{
					"error":             err,
					"gcr_uri":           gcrURI,
					"subscription_name": subName,
				}).Error("trigger.pubsub.manager: failed to create subscription")
			}

			// cleanup
			s.removeSubscription(gcrURI)

		}()
	}
}

func (s *DefaultManager) removeSubscription(gcrURI string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribers, gcrURI)
}
