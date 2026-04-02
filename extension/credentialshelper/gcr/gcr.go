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

	log "github.com/sirupsen/logrus"
)

// tokenSourceProvider obtains an oauth2.TokenSource for workload identity or
// application default credentials. In production this is google.DefaultTokenSource;
// tests assign a stub here so the package can be exercised without calling GCP.
// Not safe for use with t.Parallel() — tests must run sequentially.
var tokenSourceProvider = google.DefaultTokenSource

// scopeCloudPlatform is the OAuth 2.0 scope we request for registry access when
// no JSON key is available. Google documents
// https://www.googleapis.com/auth/cloud-platform for the Artifact Registry API;
// the narrower Cloud Storage scope (devstorage.read_only) is not sufficient for
// Docker Registry HTTP API v2 manifest calls against *.docker.pkg.dev (e.g. HTTP
// 403 on HEAD .../manifests/...). See "OAuth 2.0 Scopes for Google APIs".
const scopeCloudPlatform = "https://www.googleapis.com/auth/cloud-platform"

// jsonKeyUsername is the Docker registry username for service account JSON key
// authentication (GOOGLE_APPLICATION_CREDENTIALS).
const jsonKeyUsername = "_json_key"

// oauth2TokenUsername is the Docker registry username for bearer access tokens
// obtained via GKE workload identity or application default credentials.
const oauth2TokenUsername = "_token"

func init() {
	credentialshelper.RegisterCredentialsHelper(credentialshelper.HelperNameGCR, New())
}

// CredentialsHelper provides Docker registry credentials for gcr.io and
// Artifact Registry (*.pkg.dev) images.
type CredentialsHelper struct {
	enabled bool
}

// New creates a new GCR/Artifact Registry credentials helper.
func New() *CredentialsHelper {
	return &CredentialsHelper{
		enabled: true,
	}
}

// IsEnabled returns whether this credentials helper is active.
func (h *CredentialsHelper) IsEnabled() bool {
	return h.enabled
}

// GetCredentials returns registry credentials for gcr.io and *.pkg.dev images.
// It tries a JSON key file (GOOGLE_APPLICATION_CREDENTIALS) first, then falls
// back to a workload-identity / ADC bearer token.
func (h *CredentialsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	if !h.enabled {
		return nil, errors.New("not initialised")
	}

	if !strings.HasPrefix(image.Image.Registry(), "gcr.io") && !strings.Contains(image.Image.Registry(), "pkg.dev") {
		return nil, credentialshelper.ErrUnsupportedRegistry
	}

	creds, err := readCredentialsFromFile()
	if err == nil {
		return creds, nil
	}

	if _, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); ok {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("extension.credentialshelper.gcr: GOOGLE_APPLICATION_CREDENTIALS set but not usable, falling back to workload identity")
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
		Username: jsonKeyUsername,
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
