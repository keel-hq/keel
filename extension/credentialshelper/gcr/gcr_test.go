package gcr

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"golang.org/x/oauth2"
)

func mustParseRef(t *testing.T, ref string) *image.Reference {
	t.Helper()
	img, err := image.Parse(ref)
	if err != nil {
		t.Fatalf("image.Parse: %v", err)
	}
	return img
}

func tracked(img *image.Reference) *types.TrackedImage {
	return &types.TrackedImage{Image: img}
}

func TestCredentialsHelper_IsEnabled(t *testing.T) {
	h := New()
	if !h.IsEnabled() {
		t.Fatal("expected helper enabled")
	}
}

func TestCredentialsHelper_GetCredentials_errors(t *testing.T) {
	tests := []struct {
		name    string
		h       func() *CredentialsHelper
		ref     string
		wantErr string
		wantIs  error
	}{
		{
			name: "disabled",
			h: func() *CredentialsHelper {
				return &CredentialsHelper{enabled: false}
			},
			ref:     "gcr.io/project/image:1.0",
			wantErr: "not initialised",
		},
		{
			name: "unsupported registry",
			h: func() *CredentialsHelper {
				return New()
			},
			ref:    "docker.io/library/nginx:latest",
			wantIs: credentialshelper.ErrUnsupportedRegistry,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.h().GetCredentials(tracked(mustParseRef(t, tt.ref)))
			if tt.wantIs != nil {
				if !errors.Is(err, tt.wantIs) {
					t.Fatalf("got err=%v want %v", err, tt.wantIs)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("got err=%v want %q", err, tt.wantErr)
			}
		})
	}
}

func TestCredentialsHelper_GetCredentials_jsonKeyFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sa.json")
	jsonKey := `{"type":"service_account","project_id":"p"}`
	if err := os.WriteFile(path, []byte(jsonKey), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", path)

	tests := []struct {
		name string
		ref  string
	}{
		{
			name: "gcr.io",
			ref:  "gcr.io/myproject/myimage:latest",
		},
		{
			name: "artifact registry",
			ref:  "europe-west4-docker.pkg.dev/myproj/myrepo/img:3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			creds, err := h.GetCredentials(tracked(mustParseRef(t, tt.ref)))
			if err != nil {
				t.Fatalf("GetCredentials: %v", err)
			}
			if creds.Username != "_json_key" || creds.Password != jsonKey {
				t.Fatalf("unexpected creds: username=%q passwordLen=%d", creds.Username, len(creds.Password))
			}
		})
	}
}

func TestCredentialsHelper_GetCredentials_workloadIdentity(t *testing.T) {
	if err := os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS"); err != nil {
		t.Fatal(err)
	}

	origTS := tokenSourceProvider
	t.Cleanup(func() { tokenSourceProvider = origTS })

	errTokenRefresh := errors.New("token refresh failed")

	tests := []struct {
		name     string
		ref      string
		provider func(t *testing.T) func(context.Context, ...string) (oauth2.TokenSource, error)
		check    func(t *testing.T, creds *types.Credentials, err error)
	}{
		{
			name: "requests cloud platform scope and returns bearer creds",
			ref:  "europe-west4-docker.pkg.dev/prj/repo/img:3",
			provider: func(t *testing.T) func(context.Context, ...string) (oauth2.TokenSource, error) {
				var gotScopes []string
				t.Cleanup(func() {
					if len(gotScopes) != 1 || gotScopes[0] != scopeCloudPlatform {
						t.Errorf("token source scopes: got %v want single scope %q", gotScopes, scopeCloudPlatform)
					}
				})
				return func(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
					gotScopes = append([]string(nil), scopes...)
					return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "fake-wi-token"}), nil
				}
			},
			check: func(t *testing.T, creds *types.Credentials, err error) {
				if err != nil {
					t.Fatalf("GetCredentials: %v", err)
				}
				if creds.Username != oauth2TokenUsername || creds.Password != "fake-wi-token" {
					t.Fatalf("unexpected WI creds: %+v", creds)
				}
			},
		},
		{
			name: "token source error",
			ref:  "gcr.io/x/y:1",
			provider: func(t *testing.T) func(context.Context, ...string) (oauth2.TokenSource, error) {
				return func(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
					return nil, errors.New("no metadata server")
				}
			},
			check: func(t *testing.T, creds *types.Credentials, err error) {
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		{
			name: "token error from Token()",
			ref:  "gcr.io/x/y:1",
			provider: func(t *testing.T) func(context.Context, ...string) (oauth2.TokenSource, error) {
				return func(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
					return errTokenSource{err: errTokenRefresh}, nil
				}
			},
			check: func(t *testing.T, creds *types.Credentials, err error) {
				if err == nil {
					t.Fatal("expected error from Token()")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenSourceProvider = tt.provider(t)
			h := New()
			creds, err := h.GetCredentials(tracked(mustParseRef(t, tt.ref)))
			tt.check(t, creds, err)
		})
	}
}

type errTokenSource struct {
	err error
}

func (e errTokenSource) Token() (*oauth2.Token, error) {
	return nil, e.err
}
