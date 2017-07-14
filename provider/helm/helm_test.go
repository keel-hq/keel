package helm

import (
	"fmt"

	"testing"
)

func TestParseImage(t *testing.T) {
	imp := NewHelmImplementer("192.168.99.100:30083")

	releases, err := imp.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	fmt.Println(releases.Count)

	for _, release := range releases.Releases {
		ref, err := parseImage(release.Chart, release.Config)
		if err != nil {
			t.Errorf("failed to parse image, error: %s", err)
		}

		fmt.Println(ref.Remote())
	}
}

func TestUpdateRelease(t *testing.T) {
	imp := NewHelmImplementer("192.168.99.100:30083")

	releases, err := imp.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for _, release := range releases.Releases {

		ref, err := parseImage(release.Chart, release.Config)
		if err != nil {
			t.Errorf("failed to parse image, error: %s", err)
		}

		fmt.Println(ref.Remote())

		err = updateHelmRelease(imp, release.Name, release.Chart, "image.tag=0.0.11")

		if err != nil {
			t.Errorf("failed to update release, error: %s", err)
		}
	}
}
