package kubernetes

import (
	"reflect"
	"strings"
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTrackedImagesCarryRunningDigests(t *testing.T) {
	runningDigest := "sha256:" + strings.Repeat("a", 64)

	fp := &fakeImplementer{
		namespaces: &v1.NamespaceList{Items: []v1.Namespace{{ObjectMeta: meta_v1.ObjectMeta{Name: "xxxx"}}}},
		podList: &v1.PodList{Items: []v1.Pod{{
			ObjectMeta: meta_v1.ObjectMeta{Name: "dep-1-abc", Namespace: "xxxx"},
			Spec: v1.PodSpec{Containers: []v1.Container{
				{Name: "app", Image: "gcr.io/v2-namespace/hello-world:1.1.1"},
			}},
			Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{{
				Name:    "app",
				ImageID: "docker-pullable://gcr.io/v2-namespace/hello-world@" + runningDigest,
				State:   v1.ContainerState{Running: &v1.ContainerStateRunning{}},
			}}},
		}}},
	}

	dep := &apps_v1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:        "dep-1",
			Namespace:   "xxxx",
			Labels:      map[string]string{types.KeelPolicyLabel: "force"},
			Annotations: map[string]string{types.KeelForceTagMatchLabel: "true"},
		},
		Spec: apps_v1.DeploymentSpec{
			Selector: &meta_v1.LabelSelector{MatchLabels: map[string]string{"app": "dep-1"}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{Labels: map[string]string{"app": "dep-1"}},
				Spec: v1.PodSpec{Containers: []v1.Container{
					{Name: "app", Image: "gcr.io/v2-namespace/hello-world:1.1.1"},
				}},
			},
		},
	}

	grc := &k8s.GenericResourceCache{}
	grc.Add(MustParseGRS([]*apps_v1.Deployment{dep})...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	tracked, err := provider.TrackedImages()
	if err != nil {
		t.Fatalf("failed to get tracked images: %s", err)
	}
	if len(tracked) != 1 {
		t.Fatalf("expected a single tracked image, got %d", len(tracked))
	}
	if !reflect.DeepEqual(tracked[0].RunningDigests, []string{runningDigest}) {
		t.Fatalf("unexpected running digests: %v", tracked[0].RunningDigests)
	}
}
