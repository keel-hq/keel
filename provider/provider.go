package provider

import (
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

// Provider - generic provider interface
type Provider interface {
	Submit(event types.Event) error
	GetName() string
}

// Providers - available providers
type Providers interface {
	Submit(event types.Event) error
	List() []string // list all providers
}

// New - new providers registry
func New(providers []Provider) *DefaultProviders {
	pvs := make(map[string]Provider)

	for _, p := range providers {
		pvs[p.GetName()] = p
	}

	return &DefaultProviders{
		providers: pvs,
	}
}

type DefaultProviders struct {
	providers map[string]Provider
}

func (p *DefaultProviders) Submit(event types.Event) error {
	for _, provider := range p.providers {
		err := provider.Submit(event)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"provider": provider.GetName(),
				"event":    event.Repository,
				"trigger":  event.TriggerName,
			}).Error("provider.DefaultProviders: submit event failed")
		}
	}

	return nil
}

func (p *DefaultProviders) List() []string {
	list := []string{}
	for name := range p.providers {
		list = append(list, name)
	}
	return list
}
