package provider

import (
	"github.com/rusenask/keel/types"
)

// Provider - generic provider interface
type Provider interface {
	Submit(event types.Event) error
	GetName() string
}
