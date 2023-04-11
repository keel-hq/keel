package registry

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/registry/docker"
)

func TestDigest(t *testing.T) {

	client := New()
	digest, err := client.Digest(Opts{
		Registry: "https://index.docker.io",
		Name:     "keelhq/keel",
		Tag:      "0.8.0",
	})

	if err != nil {
		t.Errorf("error while getting digest: %s", err)
	}

	if digest != "sha256:671b6250a0793abdd9603d7f5c6f2fa1b4070661d6f56bcfc7ad5de86574ab48" {
		t.Errorf("unexpected digest: %s", digest)
	}
}

func TestOCIDigest(t *testing.T) {

	client := New()
	digest, err := client.Digest(Opts{
		Registry: "https://index.docker.io",
		Name:     "vaultwarden/server",
		Tag:      "1.25.1",
	})

	if err != nil {
		t.Errorf("error while getting digest: %s", err)
	}

	if digest != "sha256:dd8cf61d1997c098cc5686ef3116ca5cfef36f12192c01caa1de79a968397d4c" {
		t.Errorf("unexpected digest: %s", digest)
	}
}

func TestGet(t *testing.T) {
	client := New()
	repo, err := client.Get(Opts{
		Registry: constants.DefaultDockerRegistry,
		Name:     "keelhq/keel",
	})

	if err != nil {
		t.Errorf("error while getting repo: %s", err)
	}

	fmt.Println(repo.Name)
	fmt.Println(repo.Tags)
}

// https://registry.opensource.zalan.do/v2/teapot/external-dns
func TestGetNonDockerRegistryTags(t *testing.T) {
	client := New()

	repo, err := client.Get(Opts{
		Registry: "https://registry.opensource.zalan.do",
		Name:     "teapot/external-dns",
	})

	if err != nil {
		t.Errorf("error while getting repo: %s", err)
	}

	fmt.Println(repo.Name)
	fmt.Println(repo.Tags)
}

func TestGetNonDockerRegistryManifest(t *testing.T) {
	client := New()

	d, err := client.Digest(Opts{
		Registry: "https://registry.opensource.zalan.do",
		Name:     "teapot/external-dns",
		Tag:      "v0.4.8",
	})

	if err != nil {
		t.Errorf("error while getting repo manifest: %s", err)
	}

	if d != "sha256:7aa5175f39a7e8a4172972524302c9a8196f681e40d6ee5d2f6bf0ab7d600fee" {
		t.Errorf("unexpected sha?")
	}
}
func TestGetQuayRegistryManifest(t *testing.T) {
	client := New()

	d, err := client.Digest(Opts{
		Registry: "https://quay.io",
		Name:     "jetstack/cert-manager-controller",
		Tag:      "v0.2.3",
	})

	if err != nil {
		t.Fatalf("error while getting repo manifest: %s", err)
	}

	if d != "sha256:6bccc03f2e98e34f2b1782d29aed77763e93ea81de96f246ebeb81effd947085" {
		t.Errorf("unexpected sha? %s", d)
	}
}

var EnvArtifactoryUsername = "ARTIFACTORY_USERNAME"
var EnvArtifactoryPassword = "ARTIFACTORY_PASSWORD"

func TestGetArtifactory(t *testing.T) {

	if os.Getenv(EnvArtifactoryUsername) == "" && os.Getenv(EnvArtifactoryPassword) == "" {
		t.Skip()
	}

	client := New()
	repo, err := client.Get(Opts{
		Registry: "https://keel-docker-local.jfrog.io",
		Name:     "webhook-demo",
		Username: os.Getenv(EnvArtifactoryUsername),
		Password: os.Getenv(EnvArtifactoryPassword),
	})

	if err != nil {
		t.Errorf("error while getting repo: %s", err)
	}

	fmt.Println(repo.Name)
	fmt.Println(repo.Tags)
}

func TestInsecureRegistry(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, registryResp)
	}))
	defer ts.Close()

	url := strings.Replace(ts.URL, "http://", "https://", 1)

	os.Setenv(EnvInsecure, "true")

	client := New()
	digest, err := client.Digest(Opts{
		Registry: url,
		Name:     "keelhq/keel",
		Tag:      "0.8.0",
	})

	if err != nil {
		t.Errorf("error while getting digest: %s", err)
	}

	if digest != "sha256:6592be974faae18818dca9b75682c9911815a98e6d952bf8c3932fcbef4c62e8" {
		t.Errorf("unexpected digest: %s", digest)
	}
}

var registryResp = `{
	"schemaVersion": 1,
	"name": "jetstack/cert-manager-controller",
	"tag": "v0.2.3",
	"architecture": "amd64",
	"fsLayers": [
	  {
		"blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	  },
	  {
		"blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	  },
	  {
		"blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	  },
	  {
		"blobSum": "sha256:1f54aecbb7796f75aff2de7eb149830a9bb54404621fde8502a62c8d1af8e93c"
	  },
	  {
		"blobSum": "sha256:e7a7e4794e8def7e10d9ff617643cf2e6385bb74031376a8a1a5bf1af9ded12a"
	  },
	  {
		"blobSum": "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	  },
	  {
		"blobSum": "sha256:b56274a82c9306a621590419ef5ad609aab87f7d667d3a5b2f7ddf7e703a3b46"
	  }
	],
	"history": [
	  {
		"v1Compatibility": "{\"architecture\":\"amd64\",\"config\":{\"Hostname\":\"4014af9b39b0\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Cmd\":null,\"ArgsEscaped\":true,\"Image\":\"sha256:8247920c5cbae94a266528c46aa796a3f5bcdd6dddc8cdc95382d6210fb3ba29\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":[\"/usr/bin/cert-manager\"],\"OnBuild\":[],\"Labels\":{\"org.label-schema.license\":\"Apache-2.0\",\"org.label-schema.vcs-ref\":\"\",\"org.label-schema.vcs-url\":\"https://github.com/jetstack/cert-manager\"}},\"container\":\"bde62697049c8933c36b1cf1f59b840acba310fa7e8ba43fc02c109f79965d31\",\"container_config\":{\"Hostname\":\"4014af9b39b0\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Cmd\":[\"/bin/sh\",\"-c\",\"#(nop) \",\"LABEL org.label-schema.vcs-ref= org.label-schema.vcs-url=https://github.com/jetstack/cert-manager org.label-schema.license=Apache-2.0\"],\"ArgsEscaped\":true,\"Image\":\"sha256:8247920c5cbae94a266528c46aa796a3f5bcdd6dddc8cdc95382d6210fb3ba29\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":[\"/usr/bin/cert-manager\"],\"OnBuild\":[],\"Labels\":{\"org.label-schema.license\":\"Apache-2.0\",\"org.label-schema.vcs-ref\":\"\",\"org.label-schema.vcs-url\":\"https://github.com/jetstack/cert-manager\"}},\"created\":\"2018-01-15T20:42:14.793524683Z\",\"docker_version\":\"1.12.6\",\"id\":\"7ab5531a07d17680851e4a437ef32b14c9ee95cc69da4e870b2dcf9c5239c9f9\",\"os\":\"linux\",\"parent\":\"351080cf75ef7a1df210b17692f09a3f456538ff723aaf902b01bbdf4f44830e\",\"throwaway\":true}"
	  },
	  {
		"v1Compatibility": "{\"id\":\"351080cf75ef7a1df210b17692f09a3f456538ff723aaf902b01bbdf4f44830e\",\"parent\":\"f0707ef85bac7d34a0c9ce29ab0a7acc9ff0b33d4e2ba77c11e31070721086f8\",\"created\":\"2018-01-15T20:42:14.581745362Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop)  ARG VCS_REF\"]},\"throwaway\":true}"
	  },
	  {
		"v1Compatibility": "{\"id\":\"f0707ef85bac7d34a0c9ce29ab0a7acc9ff0b33d4e2ba77c11e31070721086f8\",\"parent\":\"6f0c230ea0ae652de3fe69324583652551ac499f385e89f7a79b008c69643db6\",\"created\":\"2018-01-15T20:42:14.252489701Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop)  ENTRYPOINT [\\\"/usr/bin/cert-manager\\\"]\"]},\"throwaway\":true}"
	  },
	  {
		"v1Compatibility": "{\"id\":\"6f0c230ea0ae652de3fe69324583652551ac499f385e89f7a79b008c69643db6\",\"parent\":\"debfa4650882d32380bb40246bb7f8794ce3c135f7ea38afa99fb6f8ec1789bb\",\"created\":\"2018-01-15T20:42:14.08861391Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop) ADD file:a99ab3da41517ef947b49c181074d5cf6663c374a587876723e7338a636a8d5a in /usr/bin/cert-manager \"]}}"
	  },
	  {
		"v1Compatibility": "{\"id\":\"debfa4650882d32380bb40246bb7f8794ce3c135f7ea38afa99fb6f8ec1789bb\",\"parent\":\"ffc102809100d1d58cd54b0567e0d4ff2f95175fa748be1b4bae7099bf99379c\",\"created\":\"2018-01-15T20:42:09.737612407Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c apk add --no-cache ca-certificates\"]}}"
	  },
	  {
		"v1Compatibility": "{\"id\":\"ffc102809100d1d58cd54b0567e0d4ff2f95175fa748be1b4bae7099bf99379c\",\"parent\":\"d19ee9d514689ff9251b35352861234b7eb2595d5835b57f7a6c414fad0f8f06\",\"created\":\"2018-01-09T21:10:38.538173323Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop)  CMD [\\\"/bin/sh\\\"]\"]},\"throwaway\":true}"
	  },
	  {
		"v1Compatibility": "{\"id\":\"d19ee9d514689ff9251b35352861234b7eb2595d5835b57f7a6c414fad0f8f06\",\"created\":\"2018-01-09T21:10:38.317079775Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop) ADD file:6edc55fb54ec9fc3658c8f5176a70e792103a516154442f94fed8e0290e4960e in / \"]}}"
	  }
	],
	"signatures": [
	  {
		"header": {
		  "jwk": {
			"crv": "P-256",
			"kid": "6JXV:GDO3:GVXU:G535:KHP5:BZM3:SKOZ:7UWX:DHLR:FPBC:CIQV:NUFC",
			"kty": "EC",
			"x": "3WVv4W2HX1S5vqKdghuNuWS38123GyTaAYEzlbhPC-c",
			"y": "Er7yRtQqYOrHYZtncoARxENx9tBqL3OTJ_T8pYACFNk"
		  },
		  "alg": "ES256"
		},
		"signature": "fHUOPKkqdXsWUmo20uR5xzO4B22M6XGATed00OoBZExUr3XiSIRoqBxZ2yDq1dGaLVUxzlZoRvsJOsUwfNGNOw",
		"protected": "eyJmb3JtYXRMZW5ndGgiOjQ5NjYsImZvcm1hdFRhaWwiOiJDbjAiLCJ0aW1lIjoiMjAxOC0wMS0xNVQyMDo0MzowNFoifQ"
	  }
	]
  }`

func TestInsecureRegistryTags(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, tagsResp)
	}))
	defer ts.Close()

	// replacing HTTP with HTTPS
	url := strings.Replace(ts.URL, "http://", "https://", 1)

	os.Setenv(EnvInsecure, "true")

	client := New()
	tags, err := client.Get(Opts{
		Registry: url,
		Name:     "jetstack/cert-manager-controller",
	})

	if err != nil {
		t.Errorf("error while getting tags: %s", err)
	}

	if tags.Tags[0] != "master-2993" {
		t.Errorf("unexpected tag: %s", tags.Tags[0])
	}
}

var tagsResp = `{
	"name": "jetstack/cert-manager-controller",
	"tags": [
	  "master-2993",
	  "v0.1.1",
	  "v0.1.0",
	  "master-2996",
	  "v0.2.0",
	  "master-3005",
	  "master-3006",
	  "master-3007",
	  "v0.2.1",
	  "master-3047",
	  "master-3062",
	  "master-3116",
	  "master-3123",
	  "master-3172",
	  "master-3177",
	  "master-3185",
	  "master-3186",
	  "v0.2.2",
	  "master-3381",
	  "v0.2.3",
	  "master-3383",
	  "master-3391",
	  "master-3407",
	  "master-3408",
	  "master-3412",
	  "master-3425",
	  "master-3428",
	  "master-3583",
	  "master-3584",
	  "master-3587",
	  "master-3594",
	  "master-3598",
	  "master-3600",
	  "master-3603",
	  "master-3605",
	  "master-3610",
	  "master-3611",
	  "master-3619",
	  "master-3629",
	  "master-3656",
	  "master-3666",
	  "master-3781",
	  "master-3827",
	  "master-3828",
	  "master-3884",
	  "master-3886",
	  "master-3889",
	  "master-3890",
	  "master-3922",
	  "master-3945"
	]
  }`

func TestGetDockerHubManyTags(t *testing.T) {
	client := docker.New("https://quay.io", "", "")
	tags, err := client.Tags("coreos/prometheus-operator")
	if err != nil {
		t.Errorf("error while getting repo: %s", err)
	}
	fmt.Println(tags)
}
