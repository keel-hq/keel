package secrets

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	testutil "github.com/keel-hq/keel/util/testing"
	v1 "k8s.io/api/core/v1"
)

var secretDataPayload = `{"https://index.docker.io/v1/":{"username":"user-x","password":"pass-x","email":"karolis.rusenas@gmail.com","auth":"somethinghere"}}`
var secretDataPayload2 = `{"https://index.docker.io/v1/":{"username":"foo-user-x-2","password":"bar-pass-x-2","email":"k@gmail.com","auth":"somethinghere"}}`

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

var secretDockerConfigJSONPayloadWithUsernamePassword = `{"auths":{"https://index.docker.io/v1/":{"username":"login","password":"somepass","email":"email@email.com","auth":"longbase64secret"}}}`

func TestGetSecret(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailableSecret: map[string]*v1.Secret{
			"myregistrysecret": {
				Data: map[string][]byte{
					dockerConfigKey: []byte(secretDataPayload),
				},
				Type: v1.SecretTypeDockercfg,
			},
		},
	}

	getter := NewGetter(impl, nil)

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
		AvailableSecret: map[string]*v1.Secret{
			"myregistrysecret": {
				Data: map[string][]byte{
					dockerConfigJSONKey: []byte(secretDockerConfigJSONPayload),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
		},
	}

	getter := NewGetter(impl, nil)

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
func TestGetDockerConfigJSONSecretUsernmePassword(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailableSecret: map[string]*v1.Secret{
			"myregistrysecret": {
				Data: map[string][]byte{
					dockerConfigJSONKey: []byte(secretDockerConfigJSONPayloadWithUsernamePassword),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
		},
	}

	getter := NewGetter(impl, nil)

	trackedImage := &types.TrackedImage{
		Image:     imgRef,
		Namespace: "default",
		Secrets:   []string{"myregistrysecret"},
	}

	creds, err := getter.Get(trackedImage)
	if err != nil {
		t.Errorf("failed to get creds: %s", err)
	}

	if creds.Username != "login" {
		t.Errorf("unexpected username: %s", creds.Username)
	}

	if creds.Password != "somepass" {
		t.Errorf("unexpected pass: %s", creds.Password)
	}
}

func TestGetFromDefaultCredentials(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailableSecret: map[string]*v1.Secret{
			"myregistrysecret": {
				Data: map[string][]byte{
					dockerConfigJSONKey: []byte(secretDockerConfigJSONPayloadWithUsernamePassword),
				},
				Type: v1.SecretTypeDockerConfigJson,
			},
		},
	}

	getter := NewGetter(impl, DockerCfg{
		"https://index.docker.io/v1/": &Auth{
			Username: "aa",
			Password: "bb",
		},
	})

	trackedImage := &types.TrackedImage{
		Image:     imgRef,
		Namespace: "default",
		Secrets:   []string{"myregistrysecret"},
	}

	creds, err := getter.Get(trackedImage)
	if err != nil {
		t.Errorf("failed to get creds: %s", err)
	}

	if creds.Username != "aa" {
		t.Errorf("unexpected username: %s", creds.Username)
	}

	if creds.Password != "bb" {
		t.Errorf("unexpected pass: %s", creds.Password)
	}
}

func TestGetSecretNotFound(t *testing.T) {
	imgRef, _ := image.Parse("karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		Error: fmt.Errorf("some error"),
	}

	getter := NewGetter(impl, nil)

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
				{
					Spec: v1.PodSpec{ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: "very-secret",
						},
					},
					},
				},
			},
		},
		AvailableSecret: map[string]*v1.Secret{
			"myregistrysecret": {
				Data: map[string][]byte{
					dockerConfigKey: []byte(fmt.Sprintf(secretDataPayloadEncoded, mustEncode("user-y:pass-y"))),
				},
				Type: v1.SecretTypeDockercfg,
			},
		},
	}

	getter := NewGetter(impl, nil)

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
				{
					Spec: v1.PodSpec{ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: "very-secret",
						},
					},
					},
				},
			},
		},
		AvailableSecret: map[string]*v1.Secret{
			"myregistrysecret": {
				Data: map[string][]byte{
					dockerConfigKey: []byte(secretDataPayload),
				},
				Type: v1.SecretTypeDockercfg,
			},
		},
	}

	getter := NewGetter(impl, nil)

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
				{
					Spec: v1.PodSpec{ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: "very-secret",
						},
					},
					},
				},
			},
		},
		Error: fmt.Errorf("not found"),
	}

	getter := NewGetter(impl, nil)

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

var secretDataPayloadWithPort = `{"https://example.com:3456":{"username":"user-x","password":"pass-x","email":"karolis.rusenas@gmail.com","auth":"somethinghere"}}`

func TestLookupWithPortedRegistry(t *testing.T) {
	imgRef, _ := image.Parse("https://example.com:3456/karolisr/webhook-demo:0.0.11")

	impl := &testutil.FakeK8sImplementer{
		AvailablePods: &v1.PodList{
			Items: []v1.Pod{
				{
					Spec: v1.PodSpec{ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: "example.com",
						},
					},
					},
				},
			},
		},
		AvailableSecret: map[string]*v1.Secret{
			"example.com": {
				Data: map[string][]byte{
					dockerConfigKey: []byte(secretDataPayloadWithPort),
				},
				Type: v1.SecretTypeDockercfg,
			},
		},
	}

	getter := NewGetter(impl, nil)

	trackedImage := &types.TrackedImage{
		Image:     imgRef,
		Namespace: "default",
		Secrets:   []string{"example.com"},
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
		{
			name:    "withport",
			args:    args{registry: "https://example.com:3456"},
			want:    "example.com",
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
