package secrets

import (
	"fmt"

	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"

	"k8s.io/client-go/pkg/api/v1"

	testutil "github.com/rusenask/keel/util/testing"
	"testing"
)

var secretDataPayload = `{"https://index.docker.io/v1/":{"username":"user-x","password":"pass-x","email":"karolis.rusenas@gmail.com","auth":"somethinghere"}}`

func TestGetSecret(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailableSecret: &v1.Secret{
			Data: map[string][]byte{
				dockerConfigJSONKey: []byte(secretDataPayload),
			},
			Type: v1.SecretTypeDockercfg,
		},
	}

	getter := NewGetter(impl)

	trackedImage := &types.TrackedImage{
		Image:     imgRef,
		Namespace: "default",
		Secrets:   []string{"myregistrysecret"},
	}

	creds, err := getter.Get(trackedImage)
	if err != nil {
		t.Errorf("failed to get creds: %s", err)
	}

	if creds.Username != "user-x" {
		t.Errorf("unexpected username: %s", creds.Username)
	}

	if creds.Password != "pass-x" {
		t.Errorf("unexpected pass: %s", creds.Password)
	}
}

func TestGetSecretNotFound(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		Error: fmt.Errorf("some error"),
	}

	getter := NewGetter(impl)

	trackedImage := &types.TrackedImage{
		Image:     imgRef,
		Namespace: "default",
		Secrets:   []string{"myregistrysecret"},
	}

	creds, err := getter.Get(trackedImage)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if creds.Username != "" {
		t.Errorf("expected empty username")
	}

	if creds.Password != "" {
		t.Errorf("expected empty password")
	}
}

func TestLookupHelmSecret(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailablePods: &v1.PodList{
			Items: []v1.Pod{
				v1.Pod{
					Spec: v1.PodSpec{ImagePullSecrets: []v1.LocalObjectReference{
						v1.LocalObjectReference{
							Name: "very-secret",
						},
					},
					},
				},
			},
		},
		AvailableSecret: &v1.Secret{
			Data: map[string][]byte{
				dockerConfigJSONKey: []byte(secretDataPayload),
			},
			Type: v1.SecretTypeDockercfg,
		},
	}

	getter := NewGetter(impl)

	trackedImage := &types.TrackedImage{
		Image:     imgRef,
		Namespace: "default",
		Secrets:   []string{"myregistrysecret"},
	}

	creds, err := getter.Get(trackedImage)
	if err != nil {
		t.Errorf("failed to get creds: %s", err)
	}

	if creds.Username != "user-x" {
		t.Errorf("unexpected username: %s", creds.Username)
	}

	if creds.Password != "pass-x" {
		t.Errorf("unexpected pass: %s", creds.Password)
	}
}

func TestLookupHelmNoSecretsFound(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailablePods: &v1.PodList{
			Items: []v1.Pod{
				v1.Pod{
					Spec: v1.PodSpec{ImagePullSecrets: []v1.LocalObjectReference{
						v1.LocalObjectReference{
							Name: "very-secret",
						},
					},
					},
				},
			},
		},
		Error: fmt.Errorf("not found"),
	}

	getter := NewGetter(impl)

	trackedImage := &types.TrackedImage{
		Image:     imgRef,
		Namespace: "default",
		Secrets:   []string{"myregistrysecret"},
	}

	creds, err := getter.Get(trackedImage)
	if err != nil {
		t.Errorf("failed to get creds: %s", err)
	}

	// should be anonymous
	if creds.Username != "" {
		t.Errorf("unexpected username: %s", creds.Username)
	}

	if creds.Password != "" {
		t.Errorf("unexpected pass: %s", creds.Password)
	}
}
