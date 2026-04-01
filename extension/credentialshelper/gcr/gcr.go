package gcr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/types"
	"golang.org/x/oauth2/google"
)

// tokenSourceProvider obtains an oauth2.TokenSource for workload identity or
// application default credentials. In production this is google.DefaultTokenSource;
// tests assign a stub here so the package can be exercised without calling GCP.
var tokenSourceProvider = google.DefaultTokenSource

// scopeCloudPlatform is the OAuth 2.0 scope we request for registry access when
// no JSON key is available. Google documents
// https://www.googleapis.com/auth/cloud-platform for the Artifact Registry API;
// the narrower Cloud Storage scope (devstorage.read_only) is not sufficient for
// Docker Registry HTTP API v2 manifest calls against *.docker.pkg.dev (e.g. HTTP
// 403 on HEAD .../manifests/...). See “OAuth 2.0 Scopes for Google APIs”.
const scopeCloudPlatform = "https://www.googleapis.com/auth/cloud-platform"

// oauth2TokenUsername is the registry username used with a bearer access token for
// GKE workload identity / ADC (see also Artifact Registry “access token” auth).
// This is separate from "_json_key", which is used with a service account JSON file.
const oauth2TokenUsername = "_token"

func init() {
	credentialshelper.RegisterCredentialsHelper("gcr", New())
}

type CredentialsHelper struct {
	enabled bool
}

func New() *CredentialsHelper {
	return &CredentialsHelper{
		enabled: true,
	}
}

func (h *CredentialsHelper) IsEnabled() bool {
	return h.enabled
}

func (h *CredentialsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	if !h.enabled {
		return nil, errors.New("not initialised")
	}

	if !strings.HasPrefix(image.Image.Registry(), "gcr.io") && !strings.Contains(image.Image.Registry(), "pkg.dev") {
		return nil, credentialshelper.ErrUnsupportedRegistry
	}

	if credentials, err := readCredentialsFromFile(); err == nil {
		return credentials, nil
	}

	return getWorkloadIdentityTokenCredentials()
}

func readCredentialsFromFile() (*types.Credentials, error) {
	credentialsFile, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	if !ok {
		return nil, errors.New("GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
	}

	credentials, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	return &types.Credentials{
		Username: "_json_key",
		Password: string(credentials),
	}, nil
}

func getWorkloadIdentityTokenCredentials() (*types.Credentials, error) {
	ctx := context.Background()
	tokenSource, err := tokenSourceProvider(ctx, scopeCloudPlatform)
	if err != nil {
		return nil, fmt.Errorf("failed to get default token source: %w", err)
	}
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return &types.Credentials{
		Username: oauth2TokenUsername,
		Password: token.AccessToken,
	}, nil
}
