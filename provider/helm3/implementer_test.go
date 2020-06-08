package helm3

import (
	"testing"
)

func TestImplementerList(t *testing.T) {
	t.Skip()

	imp := NewHelm3Implementer()
	releases, err := imp.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(releases) == 0 {
		t.Errorf("why no releases? ")
	}

}