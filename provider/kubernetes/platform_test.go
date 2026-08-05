package kubernetes

import (
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResourcePlatform(t *testing.T) {
	deployment := &apps_v1.Deployment{
		Spec: apps_v1.DeploymentSpec{Template: core_v1.PodTemplateSpec{
			Spec: core_v1.PodSpec{NodeSelector: map[string]string{
				"kubernetes.io/os":   "linux",
				"kubernetes.io/arch": "arm64",
			}},
		}},
	}
	resource, err := k8s.NewGenericResource(deployment)
	if err != nil {
		t.Fatal(err)
	}
	resolver := k8s.NewPlatformResolver(&fakeImplementer{nodeList: &core_v1.NodeList{Items: []core_v1.Node{
		{ObjectMeta: meta_v1.ObjectMeta{Name: "amd64", Labels: map[string]string{core_v1.LabelOSStable: "linux", core_v1.LabelArchStable: "amd64"}}, Status: core_v1.NodeStatus{NodeInfo: core_v1.NodeSystemInfo{OperatingSystem: "linux", Architecture: "amd64"}}},
		{ObjectMeta: meta_v1.ObjectMeta{Name: "arm64", Labels: map[string]string{core_v1.LabelOSStable: "linux", core_v1.LabelArchStable: "arm64"}}, Status: core_v1.NodeStatus{NodeInfo: core_v1.NodeSystemInfo{OperatingSystem: "linux", Architecture: "arm64"}}},
	}}})
	got, platformErr := resolver.Resolve(resource)
	if len(got) != 1 || got[0].OS != "linux" || got[0].Architecture != "arm64" {
		t.Fatalf("got platform %#v", got)
	}
	if platformErr != types.PlatformErrorNone {
		t.Fatalf("unexpected platform error %q", platformErr)
	}

	deployment.Spec.Template.Spec.NodeSelector = nil
	got, platformErr = resolver.Resolve(resource)
	if len(got) != 2 || platformErr != types.PlatformErrorNone {
		t.Fatalf("expected both node platforms, got %#v (%s)", got, platformErr)
	}
}

func TestVerifiedPollingEventIsRecheckedBeforeKubernetesUpdate(t *testing.T) {
	deployment := &apps_v1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "app",
			Namespace: "default",
			Labels:    map[string]string{types.KeelPolicyLabel: "all"},
		},
		Spec: apps_v1.DeploymentSpec{Template: core_v1.PodTemplateSpec{Spec: core_v1.PodSpec{
			NodeSelector: map[string]string{core_v1.LabelArchStable: "amd64"},
			Containers:   []core_v1.Container{{Image: "example/image:1.0.0"}},
		}}},
	}
	resource, err := k8s.NewGenericResource(deployment)
	if err != nil {
		t.Fatal(err)
	}
	cache := &k8s.GenericResourceCache{}
	cache.Add(resource)
	implementer := &fakeImplementer{nodeList: &core_v1.NodeList{Items: []core_v1.Node{
		{ObjectMeta: meta_v1.ObjectMeta{Name: "amd64", Labels: map[string]string{core_v1.LabelOSStable: "linux", core_v1.LabelArchStable: "amd64"}}, Status: core_v1.NodeStatus{NodeInfo: core_v1.NodeSystemInfo{OperatingSystem: "linux", Architecture: "amd64"}}},
	}}}
	provider, err := NewProvider(implementer, &fakeSender{}, nil, cache)
	if err != nil {
		t.Fatal(err)
	}

	repository := &types.Repository{
		Name:             "example/image",
		Tag:              "2.0.0",
		Platforms:        []types.Platform{{OS: "linux", Architecture: "arm64"}},
		PlatformVerified: true,
	}
	plans, err := provider.createUpdatePlans(repository)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Fatalf("incompatible verified polling event produced plans: %#v", plans)
	}

	repository.Platforms = []types.Platform{{OS: "linux", Architecture: "amd64"}}
	plans, err = provider.createUpdatePlans(repository)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 {
		t.Fatalf("compatible verified polling event produced %d plans", len(plans))
	}
}
