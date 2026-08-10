package k8s

import (
	"errors"
	"reflect"
	"testing"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakePodSource struct {
	pods      *core_v1.PodList
	err       error
	namespace string
	selector  string
}

func (f *fakePodSource) Pods(namespace, selector string) (*core_v1.PodList, error) {
	f.namespace = namespace
	f.selector = selector
	return f.pods, f.err
}

func runningDigestDeployment() *GenericResource {
	resource, err := NewGenericResource(&apps_v1.Deployment{
		ObjectMeta: meta_v1.ObjectMeta{Name: "app", Namespace: "prod"},
		Spec: apps_v1.DeploymentSpec{
			Selector: &meta_v1.LabelSelector{MatchLabels: map[string]string{"app": "app"}},
			Template: core_v1.PodTemplateSpec{
				Spec: core_v1.PodSpec{Containers: []core_v1.Container{{Name: "app", Image: "foo/bar:latest"}}},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return resource
}

func runningPod(name string, containers []core_v1.Container, statuses []core_v1.ContainerStatus) core_v1.Pod {
	return core_v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{Name: name, Namespace: "prod"},
		Spec:       core_v1.PodSpec{Containers: containers},
		Status:     core_v1.PodStatus{ContainerStatuses: statuses},
	}
}

func running(name, imageID string) core_v1.ContainerStatus {
	return core_v1.ContainerStatus{
		Name:    name,
		ImageID: imageID,
		State:   core_v1.ContainerState{Running: &core_v1.ContainerStateRunning{}},
	}
}

func TestRunningDigestResolver(t *testing.T) {
	container := []core_v1.Container{{Name: "app", Image: "foo/bar:latest"}}

	tests := []struct {
		name string
		pods []core_v1.Pod
		want map[string][]string
	}{
		{
			name: "single pod",
			pods: []core_v1.Pod{
				runningPod("a", container, []core_v1.ContainerStatus{running("app", "docker-pullable://foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}),
			},
			want: map[string][]string{"foo/bar:latest": {"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
		},
		{
			name: "replicas mid rollout report both digests",
			pods: []core_v1.Pod{
				runningPod("a", container, []core_v1.ContainerStatus{running("app", "foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}),
				runningPod("b", container, []core_v1.ContainerStatus{running("app", "foo/bar@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")}),
				runningPod("c", container, []core_v1.ContainerStatus{running("app", "foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}),
			},
			want: map[string][]string{"foo/bar:latest": {"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		},
		{
			name: "digests are keyed per container image",
			pods: []core_v1.Pod{
				runningPod("a", []core_v1.Container{
					{Name: "app", Image: "foo/bar:latest"},
					{Name: "sidecar", Image: "foo/sidecar:latest"},
				}, []core_v1.ContainerStatus{
					running("app", "foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					running("sidecar", "foo/sidecar@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
				}),
			},
			want: map[string][]string{
				"foo/bar:latest":     {"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
				"foo/sidecar:latest": {"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
			},
		},
		{
			name: "init containers are included",
			pods: []core_v1.Pod{{
				ObjectMeta: meta_v1.ObjectMeta{Name: "a", Namespace: "prod"},
				Spec: core_v1.PodSpec{
					InitContainers: []core_v1.Container{{Name: "init", Image: "foo/init:latest"}},
					Containers:     container,
				},
				Status: core_v1.PodStatus{
					InitContainerStatuses: []core_v1.ContainerStatus{{
						Name:    "init",
						ImageID: "foo/init@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
						State:   core_v1.ContainerState{Terminated: &core_v1.ContainerStateTerminated{}},
					}},
					ContainerStatuses: []core_v1.ContainerStatus{running("app", "foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
				},
			}},
			want: map[string][]string{
				"foo/bar:latest":  {"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
				"foo/init:latest": {"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
			},
		},
		{
			name: "terminating pods are ignored",
			pods: []core_v1.Pod{
				func() core_v1.Pod {
					pod := runningPod("a", container, []core_v1.ContainerStatus{running("app", "foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")})
					now := meta_v1.Now()
					pod.DeletionTimestamp = &now
					return pod
				}(),
				runningPod("b", container, []core_v1.ContainerStatus{running("app", "foo/bar@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")}),
			},
			want: map[string][]string{"foo/bar:latest": {"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		},
		{
			name: "waiting containers are ignored",
			pods: []core_v1.Pod{
				runningPod("a", container, []core_v1.ContainerStatus{{
					Name:    "app",
					ImageID: "foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					State:   core_v1.ContainerState{Waiting: &core_v1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
				}}),
			},
			want: nil,
		},
		{
			name: "statuses without a matching container are ignored",
			pods: []core_v1.Pod{
				runningPod("a", container, []core_v1.ContainerStatus{running("gone", "foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}),
			},
			want: nil,
		},
		{
			name: "images without a digest are ignored",
			pods: []core_v1.Pod{
				runningPod("a", container, []core_v1.ContainerStatus{running("app", "")}),
			},
			want: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := &fakePodSource{pods: &core_v1.PodList{Items: test.pods}}
			got := NewRunningDigestResolver(source).Resolve(runningDigestDeployment())
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("unexpected digests: %v, want %v", got, test.want)
			}
			if source.namespace != "prod" || source.selector != "app=app" {
				t.Fatalf("unexpected pod query: %s/%s", source.namespace, source.selector)
			}
		})
	}
}

func TestRunningDigestResolverUnavailable(t *testing.T) {
	if got := NewRunningDigestResolver(&fakePodSource{err: errors.New("forbidden")}).Resolve(runningDigestDeployment()); got != nil {
		t.Fatalf("expected no digests when pods cannot be listed, got %v", got)
	}
	if got := NewRunningDigestResolver(&fakePodSource{}).Resolve(nil); got != nil {
		t.Fatalf("expected no digests for a nil resource, got %v", got)
	}
	var resolver *RunningDigestResolver
	if got := resolver.Resolve(runningDigestDeployment()); got != nil {
		t.Fatalf("expected no digests from a nil resolver, got %v", got)
	}
}

func TestParseImageID(t *testing.T) {
	tests := []struct {
		imageID string
		want    string
	}{
		{imageID: "docker-pullable://foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", want: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{imageID: "foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", want: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{imageID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", want: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{imageID: "docker://sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", want: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{imageID: "registry:5000/foo/bar@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", want: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{imageID: "", want: ""},
		{imageID: "foo/bar:latest", want: ""},
		{imageID: "foo/bar@md5:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", want: ""},
	}
	for _, test := range tests {
		if got := parseImageID(test.imageID); got != test.want {
			t.Errorf("parseImageID(%q) = %q, want %q", test.imageID, got, test.want)
		}
	}
}
