package helm

import (
	"fmt"

	"github.com/ghodss/yaml"

	"github.com/rusenask/keel/extension/notification"
	"github.com/rusenask/keel/types"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release5 "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"

	"testing"
)

type fakeSender struct {
	sentEvent types.EventNotification
}

func (s *fakeSender) Configure(cfg *notification.Config) (bool, error) {
	return true, nil
}

func (s *fakeSender) Send(event types.EventNotification) error {
	s.sentEvent = event
	return nil
}

type fakeImplementer struct {
	listReleasesResponse *rls.ListReleasesResponse

	// updated info
	updatedRlsName string
	updatedChart   *chart.Chart
	updatedOptions []helm.UpdateOption
}

func (i *fakeImplementer) ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error) {
	return i.listReleasesResponse, nil
}

func (i *fakeImplementer) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	i.updatedRlsName = rlsName
	i.updatedChart = chart
	i.updatedOptions = opts

	return &rls.UpdateReleaseResponse{
		Release: &hapi_release5.Release{
			Version: 2,
		},
	}, nil
}

// helper function to generate keel configuration
func testingConfigYaml(cfg *KeelChartConfig) (vals chartutil.Values, err error) {
	root := &Root{Keel: *cfg}
	bts, err := yaml.Marshal(root)
	if err != nil {
		return nil, err
	}

	return chartutil.ReadValues(bts)
}

func TestGetChartPolicy(t *testing.T) {

	chartVals := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: 1.1.0

keel:  
  policy: all  
  trigger: poll  
  images:
    - repository: image.repository
      tag: image.tag

`

	fakeImpl := &fakeImplementer{
		listReleasesResponse: &rls.ListReleasesResponse{
			Releases: []*hapi_release5.Release{
				&hapi_release5.Release{
					Name: "release-1",
					Chart: &chart.Chart{
						Values: &chart.Config{Raw: chartVals},
					},
					Config: &chart.Config{Raw: ""},
				},
			},
		},
	}

	releases, err := fakeImpl.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	policyFound := false

	for _, release := range releases.Releases {

		vals, err := values(release.Chart, release.Config)
		if err != nil {
			t.Fatalf("failed to get values: %s", err)
		}

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
	vals, err := testingConfigYaml(&KeelChartConfig{Trigger: types.TriggerTypePoll})
	if err != nil {
		t.Fatalf("Failed to load testdata: %s", err)
	}

	cfg, err := getKeelConfig(vals)
	if err != nil {
		t.Errorf("failed to get image paths: %s", err)
	}

	if cfg.Trigger != types.TriggerTypePoll {
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

func TestGetImagesFromConfig(t *testing.T) {
	vals, err := testingConfigYaml(&KeelChartConfig{Policy: types.PolicyTypeAll, Images: []ImageDetails{
		ImageDetails{
			RepositoryPath: "repopath",
			TagPath:        "tagpath",
		},
	}})
	if err != nil {
		t.Fatalf("Failed to load testdata: %s", err)
	}

	cfg, err := getKeelConfig(vals)
	if err != nil {
		t.Errorf("failed to get image paths: %s", err)
	}

	if cfg.Images[0].RepositoryPath != "repopath" {
		t.Errorf("invalid repo path: %s", cfg.Images[0].RepositoryPath)
	}

	if cfg.Images[0].TagPath != "tagpath" {
		t.Errorf("invalid tag path: %s", cfg.Images[0].TagPath)
	}
}

func TestUpdateRelease(t *testing.T) {
	// imp := NewHelmImplementer("192.168.99.100:30083")

	chartVals := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: karolisr/webhook-demo
  tag: 0.0.10

keel:  
  policy: all  
  trigger: poll  
  images:
    - repository: image.repository
      tag: image.tag

`
	myChart := &chart.Chart{
		Values: &chart.Config{Raw: chartVals},
	}

	fakeImpl := &fakeImplementer{
		listReleasesResponse: &rls.ListReleasesResponse{
			Releases: []*hapi_release5.Release{
				&hapi_release5.Release{
					Name:   "release-1",
					Chart:  myChart,
					Config: &chart.Config{Raw: ""},
				},
			},
		},
	}

	provider := NewProvider(fakeImpl, &fakeSender{})

	err := provider.processEvent(&types.Event{
		Repository: types.Repository{
			Name: "karolisr/webhook-demo",
			Tag:  "0.0.11",
		},
	})
	if err != nil {
		t.Errorf("failed to process event, error: %s", err)
	}

	// checking updated release
	if fakeImpl.updatedChart != myChart {
		t.Errorf("wrong chart updated")
	}

	if fakeImpl.updatedRlsName != "release-1" {
		t.Errorf("unexpected release updated: %s", fakeImpl.updatedRlsName)
	}
}

var pollingValues = `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: 1.1.0

keel:  
  policy: all  
  trigger: poll 
  pollSchedule: "@every 12m" 
  images:
    - repository: image.repository
      tag: image.tag

`

func TestGetPollingSchedule(t *testing.T) {
	vals, _ := chartutil.ReadValues([]byte(pollingValues))

	cfg, err := getKeelConfig(vals)
	if err != nil {
		t.Errorf("failed to get config: %s", err)
	}

	if cfg.PollSchedule != "@every 12m" {
		t.Errorf("unexpected polling schedule: %s", cfg.PollSchedule)
	}
}
