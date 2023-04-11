package docker

import (
	"testing"
)

func TestGetDigestDockerHub(t *testing.T) {
	client := New("https://index.docker.io", "", "")

	tags, err := client.Tags("karolisr/keel")
	if err != nil {
		t.Errorf("failed to get tags, error: %s", err)
	}

	if len(tags) == 0 {
		t.Errorf("no tags?")
	}
}
