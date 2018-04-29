package secrets

import (
	"github.com/keel-hq/keel/secrets"

	"github.com/keel-hq/keel/types"
)

// CredentialsHelper - credentials helper that uses kubernetes secrets to get
// username/password for registries
type CredentialsHelper struct {
	secretsGetter secrets.Getter
}

// IsEnabled returns whether credentials helper is enabled. By default
// secrets based cred helper is always enabled, no additional configuration is required
func (ch *CredentialsHelper) IsEnabled() bool { return true }

// GetCredentials looks into kubernetes secrets to find registry credentials
func (ch *CredentialsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	return ch.secretsGetter.Get(image)
}

// New creates a new instance of secrets based credentials helper
func New(sg secrets.Getter) *CredentialsHelper {
	return &CredentialsHelper{
		secretsGetter: sg,
	}
}
