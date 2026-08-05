package helm3

import (
	"fmt"
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"helm.sh/helm/v3/pkg/release"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type platformResourceCache struct {
	resources []*k8s.GenericResource
}

func (c *platformResourceCache) Values() []*k8s.GenericResource {
	return c.resources
}

type helmNodeSource struct {
	nodes *core_v1.NodeList
}

func (s *helmNodeSource) Nodes() (*core_v1.NodeList, error) {
	return s.nodes, nil
}

func TestReleaseImagePlatforms(t *testing.T) {
	amd64 := helmTestNode("amd64", "amd64")
	arm64 := helmTestNode("arm64", "arm64")
	trackedRef, err := image.Parse("example/image:1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		nodes     []core_v1.Node
		resources []*k8s.GenericResource
		want      []types.Platform
		wantErr   types.PlatformError
		multiArch bool
	}{
		{
			name:      "one owned workload",
			nodes:     []core_v1.Node{amd64, arm64},
			resources: []*k8s.GenericResource{helmWorkload(t, "release", "amd64", "example/image:1.0.0")},
			want:      []types.Platform{{OS: "linux", Architecture: "amd64"}},
			multiArch: true,
		},
		{
			name:  "mixed owned workloads",
			nodes: []core_v1.Node{amd64, arm64},
			resources: []*k8s.GenericResource{
				helmWorkload(t, "release", "amd64", "example/image:1.0.0"),
				helmWorkload(t, "release", "arm64", "example/image:1.0.0"),
			},
			want:      []types.Platform{{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "arm64"}},
			multiArch: true,
		},
		{
			name:      "missing ownership mapping",
			nodes:     []core_v1.Node{amd64},
			resources: []*k8s.GenericResource{helmWorkload(t, "another-release", "amd64", "example/image:1.0.0")},
			wantErr:   types.PlatformErrorHelmWorkloadMapping,
		},
		{
			name:      "unresolved child workload",
			nodes:     []core_v1.Node{{ObjectMeta: meta_v1.ObjectMeta{Name: "unknown"}}},
			resources: []*k8s.GenericResource{helmWorkload(t, "release", "", "example/image:1.0.0")},
			wantErr:   types.PlatformErrorNodeMetadata,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &Provider{
				platforms: k8s.NewPlatformResolver(&helmNodeSource{nodes: &core_v1.NodeList{Items: test.nodes}}),
				resources: &platformResourceCache{resources: test.resources},
			}
			got, resolutionErr := provider.releaseImagePlatforms("default", "release", &types.TrackedImage{Image: trackedRef})
			if resolutionErr != test.wantErr || fmt.Sprint(got) != fmt.Sprint(test.want) {
				t.Fatalf("got %v (%s), want %v (%s)", got, resolutionErr, test.want, test.wantErr)
			}
			if test.multiArch && !types.PlatformsSupportAll([]types.Platform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			}, got) {
				t.Fatal("multi-architecture candidate did not cover Helm workloads")
			}
		})
	}
}

func TestVerifiedPollingEventIsRecheckedBeforeHelmUpdate(t *testing.T) {
	chart, err := testingStringToChart(`
image:
  repository: example/image
  tag: 1.0.0
keel:
  policy: all
  trigger: poll
  images:
    - repository: image.repository
      tag: image.tag
`)
	if err != nil {
		t.Fatal(err)
	}
	implementer := &fakeImplementer{listReleasesResponse: []*release.Release{{
		Name:      "release",
		Namespace: "default",
		Chart:     chart,
		Config:    map[string]interface{}{},
	}}}
	resolver := k8s.NewPlatformResolver(&helmNodeSource{nodes: &core_v1.NodeList{Items: []core_v1.Node{
		helmTestNode("amd64", "amd64"),
	}}})
	resources := &platformResourceCache{resources: []*k8s.GenericResource{
		helmWorkload(t, "release", "amd64", "example/image:1.0.0"),
	}}
	provider := NewProvider(implementer, nil, nil, WithWorkloadPlatforms(resolver, resources))
	event := &types.Event{Repository: types.Repository{
		Name:             "example/image",
		Tag:              "2.0.0",
		Platforms:        []types.Platform{{OS: "linux", Architecture: "arm64"}},
		PlatformVerified: true,
	}}

	plans, err := provider.createUpdatePlans(event)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Fatalf("incompatible verified polling event produced plans: %#v", plans)
	}

	event.Repository.Platforms = []types.Platform{{OS: "linux", Architecture: "amd64"}}
	plans, err = provider.createUpdatePlans(event)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 {
		t.Fatalf("compatible verified polling event produced %d plans", len(plans))
	}
}

func helmWorkload(t *testing.T, releaseName, architecture, imageName string) *k8s.GenericResource {
	t.Helper()
	nodeSelector := map[string]string{}
	if architecture != "" {
		nodeSelector[core_v1.LabelArchStable] = architecture
	}
	deployment := &apps_v1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      releaseName + "-workload-" + architecture,
			Namespace: "default",
			Annotations: map[string]string{
				helmReleaseNameAnnotation:      releaseName,
				helmReleaseNamespaceAnnotation: "default",
			},
		},
		Spec: apps_v1.DeploymentSpec{Template: core_v1.PodTemplateSpec{Spec: core_v1.PodSpec{
			NodeSelector: nodeSelector,
			Containers:   []core_v1.Container{{Name: "app", Image: imageName}},
		}}},
	}
	resource, err := k8s.NewGenericResource(deployment)
	if err != nil {
		t.Fatal(err)
	}
	return resource
}

func helmTestNode(name, architecture string) core_v1.Node {
	return core_v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{Name: name, Labels: map[string]string{
			core_v1.LabelOSStable:   "linux",
			core_v1.LabelArchStable: architecture,
		}},
		Status: core_v1.NodeStatus{NodeInfo: core_v1.NodeSystemInfo{OperatingSystem: "linux", Architecture: architecture}},
	}
}
