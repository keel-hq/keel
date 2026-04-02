package credentialshelper

import (
	"errors"
	"sort"
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

// Well-known helper names used at registration and for call-order priority.
const (
	HelperNameSecrets = "secrets"
	HelperNameAWS     = "aws"
	HelperNameGCR     = "gcr"
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

// helperCallOrder returns helper names in a stable, intentional order. Map iteration
// in Go is randomised; without this, cloud ADC helpers (e.g. gcr) could run before
// the Kubernetes secrets helper and win with workload-identity tokens even when the
// tracked image references imagePullSecrets (e.g. docker-registries), causing flaky
// 403 responses against private registries.
func helperCallOrder(helpers map[string]CredentialsHelper) []string {
	priority := []string{HelperNameSecrets, HelperNameAWS, HelperNameGCR}
	seen := make(map[string]struct{}, len(helpers))
	out := make([]string, 0, len(helpers))
	for _, name := range priority {
		if _, ok := helpers[name]; ok {
			out = append(out, name)
			seen[name] = struct{}{}
		}
	}
	rest := make([]string, 0, len(helpers))
	for name := range helpers {
		if _, ok := seen[name]; ok {
			continue
		}
		rest = append(rest, name)
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// GetCredentials iterates registered helpers in priority order and returns the
// first successful credentials, or ErrCredentialsNotAvailable if none match.
func GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	credHelpersM.RLock()
	defer credHelpersM.RUnlock()

	for _, name := range helperCallOrder(credHelpers) {
		credHelper := credHelpers[name]
		if credHelper.IsEnabled() {
			foundCredentials, err := credHelper.GetCredentials(image)
			if err != nil {
				if errors.Is(err, ErrUnsupportedRegistry) {
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
