package credentialshelper

import (
	"errors"
	"testing"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

type stubCredentialsHelper struct {
	enabled bool
	creds   *types.Credentials
	err     error
}

func (s stubCredentialsHelper) IsEnabled() bool { return s.enabled }

func (s stubCredentialsHelper) GetCredentials(*types.TrackedImage) (*types.Credentials, error) {
	return s.creds, s.err
}

func mustTrackedImage(t *testing.T, ref string) *types.TrackedImage {
	t.Helper()
	img, err := image.Parse(ref)
	if err != nil {
		t.Fatal(err)
	}
	return &types.TrackedImage{Image: img}
}

func TestGetCredentials_prefersKubernetesSecretsBeforeGCR(t *testing.T) {
	gcrCreds := &types.Credentials{Username: "gcr", Password: "gcr"}
	secretCreds := &types.Credentials{Username: "secret", Password: "secret"}

	RegisterCredentialsHelper(HelperNameGCR, stubCredentialsHelper{enabled: true, creds: gcrCreds})
	RegisterCredentialsHelper(HelperNameSecrets, stubCredentialsHelper{enabled: true, creds: secretCreds})
	t.Cleanup(func() {
		UnregisterCredentialsHelper(HelperNameGCR)
		UnregisterCredentialsHelper(HelperNameSecrets)
	})

	creds, err := GetCredentials(mustTrackedImage(t, "europe-west4-docker.pkg.dev/proj/repo/img:1"))
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if creds.Username != "secret" {
		t.Fatalf("want secret helper to win, got username %q", creds.Username)
	}
}

func TestGetCredentials_secretsSkippedOnUnsupportedRegistryFallsThroughToGCR(t *testing.T) {
	gcrCreds := &types.Credentials{Username: "gcr", Password: "tok"}

	RegisterCredentialsHelper(HelperNameSecrets, stubCredentialsHelper{enabled: true, err: ErrUnsupportedRegistry})
	RegisterCredentialsHelper(HelperNameGCR, stubCredentialsHelper{enabled: true, creds: gcrCreds})
	t.Cleanup(func() {
		UnregisterCredentialsHelper(HelperNameSecrets)
		UnregisterCredentialsHelper(HelperNameGCR)
	})

	creds, err := GetCredentials(mustTrackedImage(t, "europe-west4-docker.pkg.dev/proj/repo/img:1"))
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if creds.Username != "gcr" {
		t.Fatalf("want gcr helper after secrets unsupported, got username %q", creds.Username)
	}
}

func TestHelperCallOrder(t *testing.T) {
	helpers := map[string]CredentialsHelper{
		"zebra":           stubCredentialsHelper{enabled: true},
		HelperNameGCR:     stubCredentialsHelper{enabled: true},
		HelperNameSecrets: stubCredentialsHelper{enabled: true},
		HelperNameAWS:     stubCredentialsHelper{enabled: true},
		"auxiliary":       stubCredentialsHelper{enabled: true},
	}
	got := helperCallOrder(helpers)
	want := []string{HelperNameSecrets, HelperNameAWS, HelperNameGCR, "auxiliary", "zebra"}
	if len(got) != len(want) {
		t.Fatalf("len got %d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q want %q (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestGetCredentials_nonPriorityHelpersAreOrderedLexicographically(t *testing.T) {
	RegisterCredentialsHelper("zebra-extra", stubCredentialsHelper{
		enabled: true,
		creds:   &types.Credentials{Username: "from-zebra", Password: "z"},
	})
	RegisterCredentialsHelper("aaa-extra", stubCredentialsHelper{
		enabled: true,
		creds:   &types.Credentials{Username: "from-aaa", Password: "a"},
	})
	t.Cleanup(func() {
		UnregisterCredentialsHelper("zebra-extra")
		UnregisterCredentialsHelper("aaa-extra")
	})

	creds, err := GetCredentials(mustTrackedImage(t, "example.com/foo:1"))
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if creds.Username != "from-aaa" {
		t.Fatalf("expected aaa-extra before zebra-extra, got username %q", creds.Username)
	}
}

func TestGetCredentials_disabledHelperSkipped(t *testing.T) {
	RegisterCredentialsHelper(HelperNameSecrets, stubCredentialsHelper{
		enabled: false,
		creds:   &types.Credentials{Username: "no", Password: "no"},
	})
	RegisterCredentialsHelper(HelperNameGCR, stubCredentialsHelper{
		enabled: true,
		creds:   &types.Credentials{Username: "yes", Password: "yes"},
	})
	t.Cleanup(func() {
		UnregisterCredentialsHelper(HelperNameSecrets)
		UnregisterCredentialsHelper(HelperNameGCR)
	})

	creds, err := GetCredentials(mustTrackedImage(t, "gcr.io/p/i:1"))
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if creds.Username != "yes" {
		t.Fatalf("disabled secrets should be skipped: got %q", creds.Username)
	}
}

func TestGetCredentials_allFailReturnsErrCredentialsNotAvailable(t *testing.T) {
	RegisterCredentialsHelper(HelperNameSecrets, stubCredentialsHelper{enabled: true, err: errors.New("nope")})
	t.Cleanup(func() { UnregisterCredentialsHelper(HelperNameSecrets) })

	_, err := GetCredentials(mustTrackedImage(t, "docker.io/library/nginx:latest"))
	if !errors.Is(err, ErrCredentialsNotAvailable) {
		t.Fatalf("got err=%v want ErrCredentialsNotAvailable", err)
	}
}
