package gcr

import (
	"errors"
	"os"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/types"
)

func init() {
	credentialshelper.RegisterCredentialsHelper("gcr", New())
}

type CredentialsHelper struct {
	enabled     bool
	credentials string
}

func New() *CredentialsHelper {
	ch := &CredentialsHelper{}

	credentialsFile, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	if !ok {
		return ch
	}

	credentials, err := os.ReadFile(credentialsFile)
	if err != nil {
		return ch
	}

	ch.enabled = true
	ch.credentials = string(credentials)
	return ch
}

func (h *CredentialsHelper) IsEnabled() bool {
	return h.enabled
}

func (h *CredentialsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	if !h.enabled {
		return nil, errors.New("not initialised")
	}

	if image.Image.Registry() != "gcr.io" {
		return nil, credentialshelper.ErrUnsupportedRegistry
	}

	return &types.Credentials{
		Username: "_json_key",
		Password: h.credentials,
	}, nil
}
