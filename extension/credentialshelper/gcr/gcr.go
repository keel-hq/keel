package gcr

import (
        "context"
        "errors"
        "fmt"
        "io/ioutil"
        "os"
        "strings"
    
        "cloud.google.com/go/storage"
        "github.com/keel-hq/keel/extension/credentialshelper"
        "github.com/keel-hq/keel/types"
        "golang.org/x/oauth2/google"
)

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
    
        credentials, err := ioutil.ReadFile(credentialsFile)
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
        tokenSource, err := google.DefaultTokenSource(ctx, storage.ScopeReadOnly)
        if err != nil {
            return nil, fmt.Errorf("failed to get default token source: %w", err)
        }
        token, err := tokenSource.Token()
        if err != nil {
            return nil, fmt.Errorf("failed to get token: %w", err)
        }
    
        return &types.Credentials{
            Username: "_token",
            Password: token.AccessToken,
        }, nil
}
