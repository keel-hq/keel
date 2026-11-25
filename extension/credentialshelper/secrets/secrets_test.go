package secrets

import (
	"errors"
	"testing"

	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/types"
)

type fakeGetter struct {
	credentials *types.Credentials
	err         error
}

func (f *fakeGetter) Get(image *types.TrackedImage) (*types.Credentials, error) {
	return f.credentials, f.err
}

func TestGetCredentialsReturnsUnavailableForEmptyCredentials(t *testing.T) {
	ch := New(&fakeGetter{
		credentials: &types.Credentials{},
	})

	creds, err := ch.GetCredentials(&types.TrackedImage{})

	if creds != nil {
		t.Fatalf("expected no credentials, got %#v", creds)
	}
	if err != credentialshelper.ErrCredentialsNotAvailable {
		t.Fatalf("expected ErrCredentialsNotAvailable, got %v", err)
	}
}

func TestGetCredentialsReturnsUnavailableForNilCredentials(t *testing.T) {
	ch := New(&fakeGetter{})

	creds, err := ch.GetCredentials(&types.TrackedImage{})

	if creds != nil {
		t.Fatalf("expected no credentials, got %#v", creds)
	}
	if err != credentialshelper.ErrCredentialsNotAvailable {
		t.Fatalf("expected ErrCredentialsNotAvailable, got %v", err)
	}
}

func TestGetCredentialsReturnsNonEmptyCredentials(t *testing.T) {
	tests := []struct {
		name        string
		credentials *types.Credentials
	}{
		{
			name: "username and password",
			credentials: &types.Credentials{
				Username: "user",
				Password: "password",
			},
		},
		{
			name: "username only",
			credentials: &types.Credentials{
				Username: "user",
			},
		},
		{
			name: "password only",
			credentials: &types.Credentials{
				Password: "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := New(&fakeGetter{credentials: tt.credentials})

			creds, err := ch.GetCredentials(&types.TrackedImage{})

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if creds != tt.credentials {
				t.Fatalf("expected credentials %#v, got %#v", tt.credentials, creds)
			}
		})
	}
}

func TestGetCredentialsPropagatesGetterError(t *testing.T) {
	getterErr := errors.New("getter failed")
	ch := New(&fakeGetter{
		credentials: &types.Credentials{Username: "user", Password: "password"},
		err:         getterErr,
	})

	creds, err := ch.GetCredentials(&types.TrackedImage{})

	if creds != nil {
		t.Fatalf("expected no credentials, got %#v", creds)
	}
	if err != getterErr {
		t.Fatalf("expected getter error %v, got %v", getterErr, err)
	}
}
