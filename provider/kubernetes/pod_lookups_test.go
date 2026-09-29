package kubernetes

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/pkg/config"
	"github.com/keel-hq/keel/types"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

const podLookupDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000001"

func countPodLists(client *fake.Clientset) int {
	count := 0
	for _, action := range client.Actions() {
		if action.Matches("list", "pods") {
			count++
		}
	}
	return count
}

// TestTrackedImagesDoesNotListPodsPerWorkload guards against issue #934:
// TrackedImages runs for every poll job, so it must not issue a pod API
// request for each tracked workload.
func TestTrackedImagesDoesNotListPodsPerWorkload(t *testing.T) {
	const workloads = 50
	const img = "gcr.io/v2-namespace/hello-world:1.0.0"

	objects := []runtime.Object{&v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{Name: "node-1"},
		Status:     v1.NodeStatus{NodeInfo: v1.NodeSystemInfo{OperatingSystem: "linux", Architecture: "amd64"}},
	}}
	var deployments []*apps_v1.Deployment
	for n := range workloads {
		name := fmt.Sprintf("dep-%d", n)
		deployments = append(deployments, &apps_v1.Deployment{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:        name,
				Namespace:   "default",
				Annotations: map[string]string{types.KeelPolicyLabel: "all", types.KeelTriggerLabel: "poll"},
			},
			Spec: apps_v1.DeploymentSpec{
				Selector: &meta_v1.LabelSelector{MatchLabels: map[string]string{"app": name}},
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta_v1.ObjectMeta{Labels: map[string]string{"app": name}},
					Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "app", Image: img}}},
				},
			},
		})
		objects = append(objects, &v1.Pod{
			ObjectMeta: meta_v1.ObjectMeta{Name: name + "-pod", Namespace: "default", Labels: map[string]string{"app": name}},
			Spec:       v1.PodSpec{NodeName: "node-1", Containers: []v1.Container{{Name: "app", Image: img}}},
			Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{{
				Name:    "app",
				ImageID: "docker-pullable://gcr.io/v2-namespace/hello-world@" + podLookupDigest,
				State:   v1.ContainerState{Running: &v1.ContainerStateRunning{}},
			}}},
		})
	}
	client := fake.NewSimpleClientset(objects...)
	implementer := &KubernetesImplementer{client: client}

	pods := k8s.NewPodCache(client, config.KubernetesConfig{}, logrus.StandardLogger())
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		pods.Run(stop)
		close(done)
	}()
	defer func() {
		close(stop)
		<-done
	}()
	timeout := make(chan struct{})
	timer := time.AfterFunc(5*time.Second, func() { close(timeout) })
	defer timer.Stop()
	if !cache.WaitForCacheSync(timeout, pods.HasSynced) {
		t.Fatal("pod cache did not sync")
	}
	implementer.UsePodCache(pods)

	grc := &k8s.GenericResourceCache{}
	grc.Add(MustParseGRS(deployments)...)
	provider, err := NewProvider(implementer, &fakeSender{}, nil, grc)
	if err != nil {
		t.Fatalf("failed to create provider: %s", err)
	}

	client.ClearActions()
	start := time.Now()
	tracked, err := provider.TrackedImages()
	if err != nil {
		t.Fatalf("failed to get tracked images: %s", err)
	}
	if lists := countPodLists(client); lists != 0 {
		t.Fatalf("TrackedImages listed pods %d times for %d workloads", lists, workloads)
	}
	t.Logf("resolved %d workloads in %s", workloads, time.Since(start))

	if len(tracked) != workloads {
		t.Fatalf("expected %d tracked images, got %d", workloads, len(tracked))
	}
	for _, ti := range tracked {
		if len(ti.RunningDigests) != 1 || ti.RunningDigests[0] != podLookupDigest {
			t.Fatalf("expected running digest from cached pods, got %v", ti.RunningDigests)
		}
		if ti.PlatformErr != types.PlatformErrorNone || len(ti.Platforms) != 1 {
			t.Fatalf("expected resolved platform, got %v (%s)", ti.Platforms, ti.PlatformErr)
		}
	}
}

func TestPodsFallsBackToAPIUntilCacheSynced(t *testing.T) {
	client := fake.NewSimpleClientset()
	implementer := &KubernetesImplementer{client: client}
	implementer.UsePodCache(k8s.NewPodCache(client, config.KubernetesConfig{}, logrus.StandardLogger()))

	if _, err := implementer.Pods("default", "app=x"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if lists := countPodLists(client); lists != 1 {
		t.Fatalf("expected an API list before the cache synced, got %d", lists)
	}
}
