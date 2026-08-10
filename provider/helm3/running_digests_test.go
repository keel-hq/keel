package helm3

import (
	"reflect"
	"strings"
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type helmPodSource struct {
	pods map[string]*core_v1.PodList
}

func (s *helmPodSource) Pods(_, selector string) (*core_v1.PodList, error) {
	if list, ok := s.pods[selector]; ok {
		return list, nil
	}
	return &core_v1.PodList{}, nil
}

func helmWorkloadWithSelector(t *testing.T, releaseName, name, imageName string) *k8s.GenericResource {
	t.Helper()
	deployment := &apps_v1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Annotations: map[string]string{
				helmReleaseNameAnnotation:      releaseName,
				helmReleaseNamespaceAnnotation: "default",
			},
		},
		Spec: apps_v1.DeploymentSpec{
			Selector: &meta_v1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: core_v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{Labels: map[string]string{"app": name}},
				Spec:       core_v1.PodSpec{Containers: []core_v1.Container{{Name: "app", Image: imageName}}},
			},
		},
	}
	resource, err := k8s.NewGenericResource(deployment)
	if err != nil {
		t.Fatal(err)
	}
	return resource
}

func helmRunningPods(imageName, imageID string) *core_v1.PodList {
	return &core_v1.PodList{Items: []core_v1.Pod{{
		ObjectMeta: meta_v1.ObjectMeta{Name: "pod", Namespace: "default"},
		Spec:       core_v1.PodSpec{Containers: []core_v1.Container{{Name: "app", Image: imageName}}},
		Status: core_v1.PodStatus{ContainerStatuses: []core_v1.ContainerStatus{{
			Name:    "app",
			ImageID: imageID,
			State:   core_v1.ContainerState{Running: &core_v1.ContainerStateRunning{}},
		}}},
	}}}
}

func TestReleaseImageRunningDigests(t *testing.T) {
	digestA := "sha256:" + strings.Repeat("a", 64)
	digestB := "sha256:" + strings.Repeat("b", 64)

	trackedRef, err := image.Parse("example/image:1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	tracked := &types.TrackedImage{Image: trackedRef}

	resources := &platformResourceCache{resources: []*k8s.GenericResource{
		helmWorkloadWithSelector(t, "release", "owned", "example/image:1.0.0"),
		helmWorkloadWithSelector(t, "release", "other-image", "example/other:1.0.0"),
		helmWorkloadWithSelector(t, "another-release", "not-owned", "example/image:1.0.0"),
	}}
	pods := &helmPodSource{pods: map[string]*core_v1.PodList{
		"app=owned":       helmRunningPods("example/image:1.0.0", "example/image@"+digestA),
		"app=other-image": helmRunningPods("example/other:1.0.0", "example/other@"+digestB),
		"app=not-owned":   helmRunningPods("example/image:1.0.0", "example/image@"+digestB),
	}}

	provider := &Provider{
		resources:      resources,
		runningDigests: k8s.NewRunningDigestResolver(pods),
	}

	got := provider.releaseImageRunningDigests("default", "release", tracked)
	if !reflect.DeepEqual(got, []string{digestA}) {
		t.Fatalf("unexpected running digests: %v", got)
	}

	// without the resolver wired in there is nothing to report
	bare := &Provider{resources: resources}
	if got := bare.releaseImageRunningDigests("default", "release", tracked); got != nil {
		t.Fatalf("expected no digests without a resolver, got %v", got)
	}
}
