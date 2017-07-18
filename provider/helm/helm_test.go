package helm

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/rusenask/keel/types"

	"k8s.io/helm/pkg/chartutil"

	"testing"
)

// helper function to generate keel configuration
func testingConfigYaml(cfg *KeelChartConfig) (vals chartutil.Values, err error) {
	root := &Root{Keel: *cfg}
	bts, err := yaml.Marshal(root)
	if err != nil {
		return nil, err
	}

	return chartutil.ReadValues(bts)
}

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

func TestGetChartPolicy(t *testing.T) {
	imp := NewHelmImplementer("192.168.99.100:30083")

	releases, err := imp.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	policyFound := false

	for _, release := range releases.Releases {

		vals, err := values(release.Chart, release.Config)
		if err != nil {
			t.Fatalf("failed to get values: %s", err)
		}

		// policy, err := getChartPolicy(vals)
		// if err != nil {
		// 	t.Errorf("failed to parse image, error: %s", err)
		// }
		cfg, err := getKeelConfig(vals)
		if err != nil {
			t.Errorf("failed to get image paths: %s", err)
		}

		fmt.Println(cfg)

		if cfg.Policy == types.PolicyTypeAll {
			policyFound = true
		}
	}

	if !policyFound {
		t.Errorf("policy not found")
	}
}

func TestGetTriggerFromConfig(t *testing.T) {
	vals, err := testingConfigYaml(&KeelChartConfig{Trigger: "poll"})
	if err != nil {
		t.Fatalf("Failed to load testdata: %s", err)
	}

	cfg, err := getKeelConfig(vals)
	if err != nil {
		t.Errorf("failed to get image paths: %s", err)
	}

	if cfg.Trigger != "poll" {
		t.Errorf("invalid trigger: %s", cfg.Trigger)
	}
}

func TestGetPolicyFromConfig(t *testing.T) {
	vals, err := testingConfigYaml(&KeelChartConfig{Policy: types.PolicyTypeAll})
	if err != nil {
		t.Fatalf("Failed to load testdata: %s", err)
	}

	cfg, err := getKeelConfig(vals)
	if err != nil {
		t.Errorf("failed to get image paths: %s", err)
	}

	if cfg.Policy != types.PolicyTypeAll {
		t.Errorf("invalid policy: %s", cfg.Policy)
	}
}

// func TestUpdateRelease(t *testing.T) {
// 	imp := NewHelmImplementer("192.168.99.100:30083")

// 	releases, err := imp.ListReleases()
// 	if err != nil {
// 		t.Fatalf("unexpected error: %s", err)
// 	}

// 	for _, release := range releases.Releases {

// 		ref, err := parseImage(release.Chart, release.Config)
// 		if err != nil {
// 			t.Errorf("failed to parse image, error: %s", err)
// 		}

// 		fmt.Println(ref.Remote())

// 		err = updateHelmRelease(imp, release.Name, release.Chart, "image.tag=0.0.11")

// 		if err != nil {
// 			t.Errorf("failed to update release, error: %s", err)
// 		}
// 	}
// }
