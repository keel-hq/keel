package k8s

import (
	"testing"
	"time"

	appconfig "github.com/keel-hq/keel/pkg/config"
	"github.com/sirupsen/logrus"

	core_v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiwatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func testPod(namespace, name string, labels map[string]string) *core_v1.Pod {
	return &core_v1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:          name,
			Namespace:     namespace,
			Labels:        labels,
			Annotations:   map[string]string{"large": "value"},
			ManagedFields: []meta_v1.ManagedFieldsEntry{{Manager: "kubelet"}},
		},
		Spec: core_v1.PodSpec{
			NodeName:         "node-1",
			ImagePullSecrets: []core_v1.LocalObjectReference{{Name: "registry"}},
			Containers: []core_v1.Container{{
				Name:    "app",
				Image:   "example.com/app:1.0.0",
				Command: []string{"/app"},
				Env:     []core_v1.EnvVar{{Name: "A", Value: "B"}},
			}},
		},
		Status: core_v1.PodStatus{ContainerStatuses: []core_v1.ContainerStatus{{
			Name:    "app",
			ImageID: "example.com/app@sha256:abc",
			State: core_v1.ContainerState{Waiting: &core_v1.ContainerStateWaiting{
				Reason:  "CrashLoopBackOff",
				Message: "back-off restarting failed container",
			}},
			LastTerminationState: core_v1.ContainerState{Terminated: &core_v1.ContainerStateTerminated{Message: "panic"}},
		}}},
	}
}

func runPodCache(t *testing.T, client *fake.Clientset, config appconfig.KubernetesConfig) (*PodCache, chan struct{}) {
	t.Helper()
	pc := NewPodCache(client, config, logrus.StandardLogger())
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		pc.Run(stop)
		close(done)
	}()
	t.Cleanup(func() {
		close(stop)
		<-done
	})
	return pc, done
}

func syncedPodCache(t *testing.T, config appconfig.KubernetesConfig, pods ...*core_v1.Pod) *PodCache {
	t.Helper()
	client := fake.NewSimpleClientset()
	for _, pod := range pods {
		if err := client.Tracker().Add(pod); err != nil {
			t.Fatal(err)
		}
	}
	pc, _ := runPodCache(t, client, config)
	waitFor(t, pc.HasSynced)
	return pc
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	timeout := make(chan struct{})
	timer := time.AfterFunc(5*time.Second, func() { close(timeout) })
	defer timer.Stop()
	if !cache.WaitForCacheSync(timeout, condition) {
		t.Fatal("timed out waiting for the pod cache")
	}
}

func TestPodCacheFiltersByNamespaceAndSelector(t *testing.T) {
	pc := syncedPodCache(t, appconfig.KubernetesConfig{},
		testPod("a", "match", map[string]string{"app": "web", "tier": "front"}),
		testPod("a", "other-app", map[string]string{"app": "api"}),
		testPod("b", "other-namespace", map[string]string{"app": "web"}),
	)

	list, err := pc.Pods("a", "app=web")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(list.Items) != 1 || list.Items[0].Name != "match" {
		t.Fatalf("expected only the matching pod, got %v", list.Items)
	}

	if _, err := pc.Pods("a", "app in (web"); err == nil {
		t.Fatal("expected an invalid selector to fail")
	}
}

func TestPodCacheRetainsOnlyLookupFields(t *testing.T) {
	pc := syncedPodCache(t, appconfig.KubernetesConfig{}, testPod("a", "pod", map[string]string{"app": "web"}))

	list, err := pc.Pods("a", "app=web")
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("expected one pod, got %v (%v)", list, err)
	}
	pod := list.Items[0]
	status := pod.Status.ContainerStatuses[0]
	if pod.Annotations != nil || pod.ManagedFields != nil || pod.Spec.Containers[0].Command != nil || pod.Spec.Containers[0].Env != nil ||
		status.State.Waiting.Message != "" || status.LastTerminationState.Terminated != nil {
		t.Fatalf("expected unused fields to be dropped, got %+v", pod)
	}
	if pod.Spec.NodeName != "node-1" || pod.Spec.ImagePullSecrets[0].Name != "registry" ||
		pod.Spec.Containers[0].Image != "example.com/app:1.0.0" || status.ImageID != "example.com/app@sha256:abc" || status.State.Waiting == nil {
		t.Fatalf("expected lookup fields to be retained, got %+v", pod)
	}
}

func TestPodCacheTrimsWatchedPods(t *testing.T) {
	client := fake.NewSimpleClientset()
	pc, _ := runPodCache(t, client, appconfig.KubernetesConfig{})
	waitFor(t, pc.HasSynced)

	if err := client.Tracker().Add(testPod("a", "pod", map[string]string{"app": "web"})); err != nil {
		t.Fatal(err)
	}
	var pod core_v1.Pod
	waitFor(t, func() bool {
		list, err := pc.Pods("a", "app=web")
		if err != nil || len(list.Items) != 1 {
			return false
		}
		pod = list.Items[0]
		return true
	})
	if pod.Annotations != nil || pod.Spec.Containers[0].Env != nil || pod.Spec.Containers[0].Image != "example.com/app:1.0.0" {
		t.Fatalf("expected watched pod to be trimmed, got %+v", pod)
	}
}

func TestPodCacheServesWaitsForInitialSync(t *testing.T) {
	client := fake.NewSimpleClientset(testPod("a", "pod", map[string]string{"app": "web"}))
	pc, _ := runPodCache(t, client, appconfig.KubernetesConfig{})
	waitFor(t, pc.started.Load)

	if !pc.Serves("a") {
		t.Fatal("expected Serves to wait for the initial sync")
	}
	list, err := pc.Pods("a", "app=web")
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("expected the synced pod, got %v (%v)", list, err)
	}
}

func TestPodCacheServes(t *testing.T) {
	var nilCache *PodCache
	if nilCache.Serves("a") {
		t.Fatal("nil cache must not serve lookups")
	}
	if NewPodCache(fake.NewSimpleClientset(), appconfig.KubernetesConfig{}, logrus.StandardLogger()).Serves("a") {
		t.Fatal("unsynced cache must not serve lookups")
	}
	restricted := syncedPodCache(t, appconfig.KubernetesConfig{RestrictedNamespace: "a"})
	if !restricted.Serves("a") || restricted.Serves("b") {
		t.Fatal("restricted cache must serve only its namespace")
	}
	clusterWide := syncedPodCache(t, appconfig.KubernetesConfig{})
	if !clusterWide.Serves("b") {
		t.Fatal("cluster-wide cache must serve every namespace")
	}
	if clusterWide.Serves("") {
		t.Fatal("lookups across all namespaces must use the API")
	}
}

func TestListTrimmedPodsPaginates(t *testing.T) {
	client := fake.NewSimpleClientset()
	var requests []meta_v1.ListOptions
	client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		options := action.(k8stesting.ListActionImpl).ListOptions
		requests = append(requests, options)
		page := &core_v1.PodList{ListMeta: meta_v1.ListMeta{ResourceVersion: "42"}}
		if options.Continue == "" {
			page.Continue = "next"
			page.Items = []core_v1.Pod{*testPod("a", "first", nil)}
		} else {
			page.Items = []core_v1.Pod{*testPod("a", "second", nil)}
		}
		return true, page, nil
	})

	list, err := listTrimmedPods(client, "", meta_v1.ListOptions{ResourceVersion: "0"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(requests) != 2 || requests[1].Continue != "next" {
		t.Fatalf("expected two paginated requests, got %+v", requests)
	}
	for _, request := range requests {
		if request.Limit != podListPageSize || request.ResourceVersion != "" {
			t.Fatalf("expected a paginated consistent list, got %+v", request)
		}
	}
	if list.ResourceVersion != "42" || len(list.Items) != 2 || list.Items[0].Annotations != nil {
		t.Fatalf("expected trimmed pods from every page, got %+v", list)
	}
}

func TestPodCacheDisablesItselfWhenForbidden(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependWatchReactor("pods", func(action k8stesting.Action) (bool, apiwatch.Interface, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "", nil)
	})
	pc, done := runPodCache(t, client, appconfig.KubernetesConfig{})

	waitFor(t, pc.disabled.Load)
	if pc.Serves("a") {
		t.Fatal("disabled cache must not serve lookups")
	}
	select {
	case <-done:
		t.Fatal("Run must keep blocking until stopped, otherwise the workgroup shuts down")
	case <-time.After(100 * time.Millisecond):
	}
}
