package helm

import (
	"testing"
)

func TestImplementerList(t *testing.T) {
	t.Skip()

	imp := NewHelmImplementer("192.168.99.100:30083")
	releases, err := imp.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if releases.Count == 0 {
		t.Errorf("why no releases? ")
	}

}
