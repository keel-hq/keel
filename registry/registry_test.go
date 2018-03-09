package registry

import (
	"github.com/keel-hq/keel/constants"

	"fmt"
	"os"
	"testing"
)

func TestDigest(t *testing.T) {

	client := New()
	digest, err := client.Digest(Opts{
		Registry: "https://index.docker.io",
		Name:     "karolisr/keel",
		Tag:      "0.2.2",
	})

	if err != nil {
		t.Errorf("error while getting digest: %s", err)
	}

	if digest != "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb" {
		t.Errorf("unexpected digest: %s", digest)
	}
}

func TestGet(t *testing.T) {
	client := New()
	repo, err := client.Get(Opts{
		Registry: constants.DefaultDockerRegistry,
		Name:     "karolisr/keel",
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
