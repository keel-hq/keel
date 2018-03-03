package secrets

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	testutil "github.com/keel-hq/keel/util/testing"
	"k8s.io/api/core/v1"
)

var secretDataPayload = `{"https://index.docker.io/v1/":{"username":"user-x","password":"pass-x","email":"karolis.rusenas@gmail.com","auth":"somethinghere"}}`
var secretDockerConfigJSONPayload = `{
	"auths": {
	  "quay.io": {
		"auth": "a2VlbHVzZXIra2VlbHRlc3Q6U05NR0lIVlRHUkRLSTZQMTdPTkVWUFBDQUpON1g5Sk1XUDg2ODJLWDA1RDdUQU5SWDRXMDhIUEw5QldRTDAxSg==",
		"email": ""
	  }
	}
  }`

func mustEncode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func TestGetSecret(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailableSecret: &v1.Secret{
			Data: map[string][]byte{
				dockerConfigKey: []byte(secretDataPayload),
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

func TestGetDockerConfigJSONSecret(t *testing.T) {
	imgRef, _ := image.Parse("quay.io/karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailableSecret: &v1.Secret{
			Data: map[string][]byte{
				dockerConfigJSONKey: []byte(secretDockerConfigJSONPayload),
			},
			Type: v1.SecretTypeDockerConfigJson,
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

	if creds.Username != "keeluser+keeltest" {
		t.Errorf("unexpected username: %s", creds.Username)
	}

	if creds.Password != "SNMGIHVTGRDKI6P17ONEVPPCAJN7X9JMWP8682KX05D7TANRX4W08HPL9BWQL01J" {
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

var secretDataPayloadEncoded = `{"https://index.docker.io/v1/":{"auth": "%s"}}`

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
				dockerConfigKey: []byte(fmt.Sprintf(secretDataPayloadEncoded, mustEncode("user-y:pass-y"))),
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

	if creds.Username != "user-y" {
		t.Errorf("unexpected username: %s", creds.Username)
	}

	if creds.Password != "pass-y" {
		t.Errorf("unexpected pass: %s", creds.Password)
	}
}

func TestLookupHelmEncodedSecret(t *testing.T) {
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
				dockerConfigKey: []byte(secretDataPayload),
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

func Test_decodeBase64Secret(t *testing.T) {
	type args struct {
		authSecret string
	}
	tests := []struct {
		name         string
		args         args
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{
			name:         "hello there",
			args:         args{authSecret: "aGVsbG86dGhlcmU="},
			wantUsername: "hello",
			wantPassword: "there",
			wantErr:      false,
		},
		{
			name:         "hello there, encoded",
			args:         args{authSecret: mustEncode("hello:there")},
			wantUsername: "hello",
			wantPassword: "there",
			wantErr:      false,
		},
		{
			name:         "empty",
			args:         args{authSecret: ""},
			wantUsername: "",
			wantPassword: "",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUsername, gotPassword, err := decodeBase64Secret(tt.args.authSecret)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeBase64Secret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUsername != tt.wantUsername {
				t.Errorf("decodeBase64Secret() gotUsername = %v, want %v", gotUsername, tt.wantUsername)
			}
			if gotPassword != tt.wantPassword {
				t.Errorf("decodeBase64Secret() gotPassword = %v, want %v", gotPassword, tt.wantPassword)
			}
		})
	}
}

func Test_hostname(t *testing.T) {
	type args struct {
		registry string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "dockerhub",
			args:    args{registry: "https://index.docker.io/v1/"},
			want:    "index.docker.io",
			wantErr: false,
		},
		{
			name:    "quay",
			args:    args{registry: "quay.io"},
			want:    "quay.io",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hostname(tt.args.registry)
			if (err != nil) != tt.wantErr {
				t.Errorf("hostname() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("hostname() = %v, want %v", got, tt.want)
			}
		})
	}
}
