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
	GetCredentials(image *types.TrackedImage) (*types.Credentials, error)
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

// UnregisterCredentialsHelper - unregister existing credentials helper, used for testing
func UnregisterCredentialsHelper(name string) {
	if name == "" {
		panic("credentialshelper: could not unregister a Credentials Helper with an empty name")
	}

	credHelpersM.Lock()
	defer credHelpersM.Unlock()

	delete(credHelpers, name)
}

// GetCredentials - generic function for getting credentials
// func (ch *CredentialsHelpers) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
func GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	credHelpersM.RLock()
	defer credHelpersM.RUnlock()

	for name, credHelper := range credHelpers {
		if credHelper.IsEnabled() {
			foundCredentials, err := credHelper.GetCredentials(image)
			if err != nil {
				if err == ErrUnsupportedRegistry {
					log.WithFields(log.Fields{
						"helper":        name,
						"error":         err,
						"tracked_image": image,
					}).Debug("extension.credentialshelper: helper doesn't support this registry")
				} else {
					log.WithFields(log.Fields{
						"helper":        name,
						"error":         err,
						"tracked_image": image,
					}).Debug("extension.credentialshelper: credentials not found")
				}
				continue
			}

			if foundCredentials == nil {
				log.WithFields(log.Fields{
					"error":                   "credentials helper returned nil error but also nil creds",
					"credentials_helper_name": name,
				}).Warn("extension.credentialshelper: no error and no credentials")
				// try next helper
				continue
			}
			// credentials found!
			return foundCredentials, nil
		}
	}
	log.WithFields(log.Fields{
		"tracked_image": image,
	}).Debug("extension.credentialshelper: credentials helper not found")
	return nil, ErrCredentialsNotAvailable
}
