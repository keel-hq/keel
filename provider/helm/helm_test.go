package helm

import (
	"testing"
)

func TestImplementerList(t *testing.T) {
	imp := NewHelmImplementer()
	releases, err := imp.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if releases.Count == 0 {
		t.Errorf("why no releases? ")
	}
}
