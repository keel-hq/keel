package k8s

import (
	"fmt"
	"testing"

	"github.com/keel-hq/keel/types"
	apps_v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakePlatformSource struct {
	nodes *core_v1.NodeList
	pods  *core_v1.PodList
	err   error
	calls int
}

func (f *fakePlatformSource) Nodes() (*core_v1.NodeList, error) {
	f.calls++
	return f.nodes, f.err
}

func (f *fakePlatformSource) Pods(_, _ string) (*core_v1.PodList, error) {
	return f.pods, nil
}

func TestGenericResourceGetPodSpec(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
	}{
		{name: "deployment", obj: &apps_v1.Deployment{Spec: apps_v1.DeploymentSpec{Template: podTemplate("deployment-node")}}},
		{name: "statefulset", obj: &apps_v1.StatefulSet{Spec: apps_v1.StatefulSetSpec{Template: podTemplate("statefulset-node")}}},
		{name: "daemonset", obj: &apps_v1.DaemonSet{Spec: apps_v1.DaemonSetSpec{Template: podTemplate("daemonset-node")}}},
		{name: "cronjob", obj: &batch_v1.CronJob{Spec: batch_v1.CronJobSpec{JobTemplate: batch_v1.JobTemplateSpec{Spec: batch_v1.JobSpec{Template: podTemplate("cronjob-node")}}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource, err := NewGenericResource(test.obj)
			if err != nil {
				t.Fatal(err)
			}
			if got := resource.GetPodSpec(); got == nil || got.NodeName != test.name+"-node" {
				t.Fatalf("unexpected pod spec: %#v", got)
			}
		})
	}
}

func podTemplate(nodeName string) core_v1.PodTemplateSpec {
	return core_v1.PodTemplateSpec{Spec: core_v1.PodSpec{NodeName: nodeName}}
}

func TestPlatformResolverSchedulingMetadata(t *testing.T) {
	nodes := &core_v1.NodeList{Items: []core_v1.Node{
		testNode("amd", "linux", "amd64", map[string]string{"pool": "general", "generation": "3"}),
		testNode("arm", "linux", "arm64", map[string]string{"pool": "edge", "generation": "2"}),
		testNode("windows", "windows", "amd64", map[string]string{"pool": "general", "generation": "4"}),
	}}
	tests := []struct {
		name string
		spec core_v1.PodSpec
		want []types.Platform
	}{
		{
			name: "mixed unconstrained nodes",
			want: []types.Platform{{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "arm64"}, {OS: "windows", Architecture: "amd64"}},
		},
		{
			name: "stable architecture selector",
			spec: core_v1.PodSpec{NodeSelector: map[string]string{core_v1.LabelArchStable: "amd64", core_v1.LabelOSStable: "linux"}},
			want: []types.Platform{{OS: "linux", Architecture: "amd64"}},
		},
		{
			name: "legacy architecture selector",
			spec: core_v1.PodSpec{NodeSelector: map[string]string{"beta.kubernetes.io/arch": "arm64"}},
			want: []types.Platform{{OS: "linux", Architecture: "arm64"}},
		},
		{
			name: "node name",
			spec: core_v1.PodSpec{NodeName: "arm"},
			want: []types.Platform{{OS: "linux", Architecture: "arm64"}},
		},
		{
			name: "pod OS",
			spec: core_v1.PodSpec{OS: &core_v1.PodOS{Name: core_v1.Windows}},
			want: []types.Platform{{OS: "windows", Architecture: "amd64"}},
		},
		{
			name: "required affinity",
			spec: core_v1.PodSpec{Affinity: &core_v1.Affinity{
				NodeAffinity: &core_v1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &core_v1.NodeSelector{
						NodeSelectorTerms: []core_v1.NodeSelectorTerm{{
							MatchExpressions: []core_v1.NodeSelectorRequirement{
								{Key: "pool", Operator: core_v1.NodeSelectorOpIn, Values: []string{"general"}},
								{Key: "generation", Operator: core_v1.NodeSelectorOpGt, Values: []string{"3"}},
							},
						}},
					},
				},
			}},
			want: []types.Platform{{OS: "windows", Architecture: "amd64"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployment := &apps_v1.Deployment{Spec: apps_v1.DeploymentSpec{Template: core_v1.PodTemplateSpec{Spec: test.spec}}}
			resource, err := NewGenericResource(deployment)
			if err != nil {
				t.Fatal(err)
			}
			got, resolutionErr := NewPlatformResolver(&fakePlatformSource{nodes: nodes}).Resolve(resource)
			if resolutionErr != types.PlatformErrorNone || fmt.Sprint(got) != fmt.Sprint(test.want) {
				t.Fatalf("got %v (%s), want %v", got, resolutionErr, test.want)
			}
		})
	}
}

func TestPlatformResolverIncludesObservedPodPlatform(t *testing.T) {
	deployment := &apps_v1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{Namespace: "default"},
		Spec: apps_v1.DeploymentSpec{
			Selector: &meta_v1.LabelSelector{MatchLabels: map[string]string{"app": "example"}},
			Template: core_v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{Labels: map[string]string{"app": "example"}},
				Spec:       core_v1.PodSpec{NodeSelector: map[string]string{core_v1.LabelArchStable: "arm64"}},
			},
		},
	}
	resource, err := NewGenericResource(deployment)
	if err != nil {
		t.Fatal(err)
	}
	source := &fakePlatformSource{
		nodes: &core_v1.NodeList{Items: []core_v1.Node{
			testNode("amd", "linux", "amd64", nil),
			testNode("arm", "linux", "arm64", nil),
		}},
		pods: &core_v1.PodList{Items: []core_v1.Pod{{Spec: core_v1.PodSpec{NodeName: "amd"}}}},
	}
	got, resolutionErr := NewPlatformResolver(source).Resolve(resource)
	want := []types.Platform{{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "arm64"}}
	if resolutionErr != types.PlatformErrorNone || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v (%s), want %v", got, resolutionErr, want)
	}
}

func TestPlatformResolverFailsClosed(t *testing.T) {
	resource, err := NewGenericResource(&apps_v1.Deployment{})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		source *fakePlatformSource
		want   types.PlatformError
	}{
		{name: "node API unavailable", source: &fakePlatformSource{err: fmt.Errorf("forbidden")}, want: types.PlatformErrorNodeMetadata},
		{name: "nil node response", source: &fakePlatformSource{}, want: types.PlatformErrorNodeMetadata},
		{name: "no nodes", source: &fakePlatformSource{nodes: &core_v1.NodeList{}}, want: types.PlatformErrorNoEligibleNodes},
		{name: "missing node platform", source: &fakePlatformSource{nodes: &core_v1.NodeList{Items: []core_v1.Node{{}}}}, want: types.PlatformErrorNodeMetadata},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			platforms, resolutionErr := NewPlatformResolver(test.source).Resolve(resource)
			if len(platforms) != 0 || resolutionErr != test.want {
				t.Fatalf("got %v (%s), want no platforms and %s", platforms, resolutionErr, test.want)
			}
		})
	}
}

func TestMatchesRequirementOperators(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		operator core_v1.NodeSelectorOperator
		values   []string
		want     bool
	}{
		{name: "in", value: "edge", operator: core_v1.NodeSelectorOpIn, values: []string{"edge"}, want: true},
		{name: "not in", value: "edge", operator: core_v1.NodeSelectorOpNotIn, values: []string{"general"}, want: true},
		{name: "exists", value: "set", operator: core_v1.NodeSelectorOpExists, want: true},
		{name: "does not exist", operator: core_v1.NodeSelectorOpDoesNotExist, want: true},
		{name: "greater than", value: "3", operator: core_v1.NodeSelectorOpGt, values: []string{"2"}, want: true},
		{name: "less than", value: "2", operator: core_v1.NodeSelectorOpLt, values: []string{"3"}, want: true},
		{name: "invalid numeric is conservative", value: "unknown", operator: core_v1.NodeSelectorOpGt, values: []string{"3"}, want: true},
		{name: "unknown operator is conservative", value: "anything", operator: core_v1.NodeSelectorOperator("Future"), want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := matchesRequirement(test.value, test.operator, test.values); got != test.want {
				t.Fatalf("got %t, want %t", got, test.want)
			}
		})
	}
}

func TestPlatformResolverAffinityTermsAreORed(t *testing.T) {
	source := &fakePlatformSource{nodes: &core_v1.NodeList{Items: []core_v1.Node{
		testNode("amd", "linux", "amd64", map[string]string{"pool": "general"}),
		testNode("arm", "linux", "arm64", map[string]string{"pool": "edge"}),
	}}}
	deployment := &apps_v1.Deployment{Spec: apps_v1.DeploymentSpec{Template: core_v1.PodTemplateSpec{Spec: core_v1.PodSpec{
		Affinity: &core_v1.Affinity{NodeAffinity: &core_v1.NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution: &core_v1.NodeSelector{
			NodeSelectorTerms: []core_v1.NodeSelectorTerm{
				{MatchExpressions: []core_v1.NodeSelectorRequirement{{Key: "pool", Operator: core_v1.NodeSelectorOpIn, Values: []string{"missing"}}}},
				{MatchFields: []core_v1.NodeSelectorRequirement{{Key: "future.field", Operator: core_v1.NodeSelectorOpIn, Values: []string{"unknown"}}}},
			},
		}}},
	}}}}
	resource, err := NewGenericResource(deployment)
	if err != nil {
		t.Fatal(err)
	}
	platforms, resolutionErr := NewPlatformResolver(source).Resolve(resource)
	if resolutionErr != types.PlatformErrorNone || len(platforms) != 2 {
		t.Fatalf("expected conservative mixed set, got %v (%s)", platforms, resolutionErr)
	}
}

func TestPlatformResolverCachesNodes(t *testing.T) {
	source := &fakePlatformSource{nodes: &core_v1.NodeList{Items: []core_v1.Node{testNode("amd", "linux", "amd64", nil)}}}
	resource, err := NewGenericResource(&apps_v1.Deployment{})
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewPlatformResolver(source)
	for range 2 {
		if _, resolutionErr := resolver.Resolve(resource); resolutionErr != types.PlatformErrorNone {
			t.Fatal(resolutionErr)
		}
	}
	if source.calls != 1 {
		t.Fatalf("expected one node-list call, got %d", source.calls)
	}
}

func testNode(name, os, architecture string, extraLabels map[string]string) core_v1.Node {
	labels := map[string]string{
		core_v1.LabelOSStable:     os,
		core_v1.LabelArchStable:   architecture,
		"beta.kubernetes.io/os":   os,
		"beta.kubernetes.io/arch": architecture,
	}
	for key, value := range extraLabels {
		labels[key] = value
	}
	return core_v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{Name: name, Labels: labels},
		Status: core_v1.NodeStatus{NodeInfo: core_v1.NodeSystemInfo{
			OperatingSystem: os,
			Architecture:    architecture,
		}},
	}
}
