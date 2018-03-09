package provider

import (
	"context"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// Provider - generic provider interface
type Provider interface {
	Submit(event types.Event) error
	TrackedImages() ([]*types.TrackedImage, error)
	GetName() string
	Stop()
}

// Providers - available providers
type Providers interface {
	Submit(event types.Event) error
	TrackedImages() ([]*types.TrackedImage, error)
	List() []string // list all providers
	Stop()          // stop all providers
}

// New - new providers registry
func New(providers []Provider, approvalsManager approvals.Manager) *DefaultProviders {
	pvs := make(map[string]Provider)

	for _, p := range providers {
		pvs[p.GetName()] = p
		log.Infof("provider.defaultProviders: provider '%s' registered", p.GetName())
	}

	dp := &DefaultProviders{
		providers:        pvs,
		approvalsManager: approvalsManager,
		stopCh:           make(chan struct{}),
	}

	// subscribing to approved events
	// TODO: create Start() function for DefaultProviders
	go dp.subscribeToApproved()

	return dp
}

// DefaultProviders - default providers container
type DefaultProviders struct {
	providers        map[string]Provider
	approvalsManager approvals.Manager
	stopCh           chan struct{}
}

func (p *DefaultProviders) subscribeToApproved() {
	ctx, cancel := context.WithCancel(context.Background())

	approvedCh, err := p.approvalsManager.SubscribeApproved(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("provider.subscribeToApproved: failed to subscribe for approved reqs")
	}

	for {
		select {
		case approval := <-approvedCh:
			p.Submit(*approval.Event)
		case <-p.stopCh:
			cancel()
			return
		}
	}

}

// Submit - submit event to all providers
func (p *DefaultProviders) Submit(event types.Event) error {
	for _, provider := range p.providers {
		err := provider.Submit(event)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"provider": provider.GetName(),
				"event":    event.Repository,
				"trigger":  event.TriggerName,
			}).Error("provider.Submit: submit event failed")
		}
	}

	return nil
}

// TrackedImages - get tracked images for provider
func (p *DefaultProviders) TrackedImages() ([]*types.TrackedImage, error) {
	var trackedImages []*types.TrackedImage
	for _, provider := range p.providers {
		ti, err := provider.TrackedImages()
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"provider": provider.GetName(),
			}).Error("provider.defaultProviders: failed to get tracked images")
			continue
		}
		trackedImages = append(trackedImages, ti...)
	}

	return trackedImages, nil
}

// List - list available providers
func (p *DefaultProviders) List() []string {
	list := []string{}
	for name := range p.providers {
		list = append(list, name)
	}
	return list
}

// Stop - stop all providers
func (p *DefaultProviders) Stop() {
	for _, provider := range p.providers {
		provider.Stop()
	}
	return
}
