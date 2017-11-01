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
