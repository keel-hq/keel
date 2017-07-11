package helm

import (
	"github.com/rusenask/keel/extension/notification"
	"github.com/rusenask/keel/types"
)

// ProviderName - provider name
const ProviderName = "helm"

// Provider - kubernetes provider for auto update
type Provider struct {
	sender notification.Sender

	events chan *types.Event
	stop   chan struct{}
}

// NewProvider - create new kubernetes based provider
func NewProvider(sender notification.Sender) (*Provider, error) {
	return &Provider{
		events: make(chan *types.Event, 100),
		stop:   make(chan struct{}),
		sender: sender,
	}, nil
}

// Submit - submit event to provider
func (p *Provider) Submit(event types.Event) error {
	p.events <- &event
	return nil
}

// GetName - get provider name
func (p *Provider) GetName() string {
	return ProviderName
}
