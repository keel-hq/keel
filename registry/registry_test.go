package registry

import (
	"github.com/rusenask/keel/constants"

	"testing"

	"fmt"
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

	if digest != "sha256:b976145cfe8c6ed1c9670ff3309b6029e68896adee621f047e2818fafb6ee199" {
		t.Errorf("unexpected digest: %s", digest)
	}
}

func TestGet(t *testing.T) {
	client := New()
	repo, err := client.Get(Opts{
		// Registry: "https://index.docker.io",
		Registry: constants.DefaultDockerRegistry,
		Name:     "karolisr/keel",
	})

	if err != nil {
		t.Errorf("error while getting repo: %s", err)
	}

	fmt.Println(repo.Name)
	fmt.Println(repo.Tags)
}
