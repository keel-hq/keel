package credentialshelper

import (
	"errors"
	"sync"

	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// CredentialsHelper is a generic interface for implementing cloud vendor specific
// authorization code
type CredentialsHelper interface {
	GetCredentials(registry string) (*types.Credentials, error)
	IsEnabled() bool
}

// Common errors
var (
	ErrCredentialsNotAvailable = errors.New("no credentials available for this registry")
	ErrUnsupportedRegistry     = errors.New("unsupported registry")
)

var (
	credHelpersM sync.RWMutex
	credHelpers  = make(map[string]CredentialsHelper)
)

// RegisterCredentialsHelper - registering new credentials helper
func RegisterCredentialsHelper(name string, ch CredentialsHelper) {
	if name == "" {
		panic("credentialshelper: could not register a Credentials Helper with an empty name")
	}

	if ch == nil {
		panic("credentialshelper: could not register a nil Credentials Helper")
	}

	credHelpersM.Lock()
	defer credHelpersM.Unlock()

	if _, dup := credHelpers[name]; dup {
		panic("credentialshelper: RegisterCredentialsHelper called twice for " + name)
	}

	log.WithFields(log.Fields{
		"name": name,
	}).Info("extension.credentialshelper: helper registered")

	credHelpers[name] = ch
}

// CredentialsHelpers is a combined list of credential helpers
type CredentialsHelpers struct {
}

// New returns a combined list of credential helpers
func New() *CredentialsHelpers {
	return &CredentialsHelpers{}
}

// IsEnabled returns if cred helper is enabled
func (ch *CredentialsHelpers) IsEnabled() bool {
	return true
}

// GetCredentials - generic function for getting credentials
func (ch *CredentialsHelpers) GetCredentials(registry string) (*types.Credentials, error) {
	credHelpersM.RLock()
	defer credHelpersM.RUnlock()

	for name, credHelper := range credHelpers {
		if credHelper.IsEnabled() {
			creds, err := credHelper.GetCredentials(registry)
			if err != nil {
				log.WithFields(log.Fields{
					"helper":   name,
					"error":    err,
					"registry": registry,
				}).Error("extension.credentialshelper: credentials not found")
			} else {
				return creds, nil
			}
		}
	}

	return nil, ErrCredentialsNotAvailable
}
