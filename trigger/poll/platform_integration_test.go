package poll

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_typed_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type integrationImplementer struct {
	kubernetes.Implementer
	nodes   *core_v1.NodeList
	updated chan *k8s.GenericResource
}

func (i *integrationImplementer) Nodes() (*core_v1.NodeList, error) { return i.nodes, nil }
func (i *integrationImplementer) Pods(string, string) (*core_v1.PodList, error) {
	return &core_v1.PodList{}, nil
}
func (i *integrationImplementer) Update(resource *k8s.GenericResource) error {
	i.updated <- resource.DeepCopy()
	return nil
}
func (i *integrationImplementer) ConfigMaps(string) core_typed_v1.ConfigMapInterface {
	return nil
}

type integrationSender struct{ notification.Sender }

func (integrationSender) Send(types.EventNotification) error { return nil }

type integrationApprovals struct{ approvals.Manager }

func (integrationApprovals) Exists(string) bool { return false }

type integrationProviders struct{ provider *kubernetes.Provider }

func (p integrationProviders) Submit(event types.Event) error { return p.provider.Submit(event) }
func (p integrationProviders) TrackedImages() ([]*types.TrackedImage, error) {
	return p.provider.TrackedImages()
}
func (p integrationProviders) List() []string { return []string{p.provider.GetName()} }
func (p integrationProviders) Stop()          { p.provider.Stop() }

type registryPlatform struct {
	OS           string
	Architecture string
	Variant      string
}

func TestRegistryPollingToKubernetesPlatformSelection(t *testing.T) {
	tests := []struct {
		name          string
		currentTag    string
		tags          []string
		selector      map[string]string
		nodePlatforms []registryPlatform
		manifests     map[string][]registryPlatform
		wantTag       string
	}{
		{
			name:       "issue 834 skips newer armhf tag on amd64 workload",
			currentTag: "latest",
			tags:       []string{"latest", "10.10.7", "20240303.2-unstable-armhf"},
			selector:   map[string]string{core_v1.LabelArchStable: "amd64"},
			nodePlatforms: []registryPlatform{
				{OS: "linux", Architecture: "amd64"},
			},
			manifests: map[string][]registryPlatform{
				"10.10.7":                   {{OS: "linux", Architecture: "amd64"}},
				"20240303.2-unstable-armhf": {{OS: "linux", Architecture: "arm", Variant: "v7"}},
			},
			wantTag: "10.10.7",
		},
		{
			name:       "mixed workload rejects single platform and accepts multi arch",
			currentTag: "1.0.0",
			tags:       []string{"1.0.0", "2.0.0", "3.0.0"},
			nodePlatforms: []registryPlatform{
				{OS: "linux", Architecture: "amd64"},
				{OS: "linux", Architecture: "arm64"},
			},
			manifests: map[string][]registryPlatform{
				"3.0.0": {{OS: "linux", Architecture: "amd64"}},
				"2.0.0": {
					{OS: "linux", Architecture: "amd64"},
					{OS: "linux", Architecture: "arm64"},
				},
			},
			wantTag: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newPlatformRegistry(t, tt.tags, tt.manifests)
			defer server.Close()

			imageName := strings.TrimPrefix(server.URL, "http://") + "/jellyfin/jellyfin:" + tt.currentTag
			deployment := &apps_v1.Deployment{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "jellyfin",
					Namespace: "media",
					Labels: map[string]string{
						types.KeelPolicyLabel:  "major",
						types.KeelTriggerLabel: "poll",
					},
				},
				Spec: apps_v1.DeploymentSpec{
					Selector: &meta_v1.LabelSelector{MatchLabels: map[string]string{"app": "jellyfin"}},
					Template: core_v1.PodTemplateSpec{
						ObjectMeta: meta_v1.ObjectMeta{Labels: map[string]string{"app": "jellyfin"}},
						Spec: core_v1.PodSpec{
							NodeSelector: tt.selector,
							Containers:   []core_v1.Container{{Name: "jellyfin", Image: "http://" + imageName}},
						},
					},
				},
			}
			resource, err := k8s.NewGenericResource(deployment)
			if err != nil {
				t.Fatal(err)
			}
			cache := &k8s.GenericResourceCache{}
			cache.Add(resource)

			implementer := &integrationImplementer{updated: make(chan *k8s.GenericResource, 1)}
			for index, platform := range tt.nodePlatforms {
				implementer.nodes = appendNode(implementer.nodes, index, platform)
			}
			kubeProvider, err := kubernetes.NewProvider(implementer, integrationSender{}, integrationApprovals{}, cache)
			if err != nil {
				t.Fatal(err)
			}
			providers := integrationProviders{provider: kubeProvider}
			providerDone := make(chan error, 1)
			go func() { providerDone <- kubeProvider.Start() }()
			defer func() {
				providers.Stop()
				<-providerDone
			}()

			tracked, err := providers.TrackedImages()
			if err != nil {
				t.Fatal(err)
			}
			if len(tracked) != 1 {
				t.Fatalf("expected one tracked image, got %d", len(tracked))
			}
			job := NewWatchRepositoryTagsJob(providers, registry.New(), &watchDetails{trackedImage: tracked[0]})
			job.Run()

			select {
			case updated := <-implementer.updated:
				got := updated.GetImages(nil)
				if len(got) != 1 || !strings.HasSuffix(got[0], ":"+tt.wantTag) {
					t.Fatalf("expected selected tag %s, got %v", tt.wantTag, got)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("timed out waiting for Kubernetes update to %s", tt.wantTag)
			}
		})
	}
}

func appendNode(nodes *core_v1.NodeList, index int, platform registryPlatform) *core_v1.NodeList {
	if nodes == nil {
		nodes = &core_v1.NodeList{}
	}
	nodes.Items = append(nodes.Items, core_v1.Node{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: fmt.Sprintf("node-%d", index),
			Labels: map[string]string{
				core_v1.LabelOSStable:   platform.OS,
				core_v1.LabelArchStable: platform.Architecture,
			},
		},
		Status: core_v1.NodeStatus{NodeInfo: core_v1.NodeSystemInfo{
			OperatingSystem: platform.OS,
			Architecture:    platform.Architecture,
		}},
	})
	return nodes
}

func newPlatformRegistry(t *testing.T, tags []string, manifests map[string][]registryPlatform) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/jellyfin/jellyfin/tags/list":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"name": "jellyfin/jellyfin", "tags": tags})
		case strings.HasPrefix(r.URL.Path, "/v2/jellyfin/jellyfin/manifests/"):
			tag := strings.TrimPrefix(r.URL.Path, "/v2/jellyfin/jellyfin/manifests/")
			platforms, ok := manifests[tag]
			if !ok {
				http.NotFound(w, r)
				return
			}
			if len(platforms) == 1 {
				w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"schemaVersion": 2,
					"config": map[string]interface{}{
						"mediaType": "application/vnd.oci.image.config.v1+json",
						"digest":    configDigest(tag),
						"size":      1,
					},
				})
				return
			}
			w.Header().Set("Content-Type", "application/vnd.oci.image.index.v1+json")
			descriptors := make([]map[string]interface{}, 0, len(platforms))
			for index, platform := range platforms {
				descriptors = append(descriptors, map[string]interface{}{
					"mediaType": "application/vnd.oci.image.manifest.v1+json",
					"digest":    fmt.Sprintf("sha256:%064x", index+1),
					"size":      1,
					"platform":  platform,
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"schemaVersion": 2, "manifests": descriptors})
		case strings.HasPrefix(r.URL.Path, "/v2/jellyfin/jellyfin/blobs/"):
			digest := strings.TrimPrefix(r.URL.Path, "/v2/jellyfin/jellyfin/blobs/")
			for tag, platforms := range manifests {
				if len(platforms) == 1 && digest == configDigest(tag) {
					_ = json.NewEncoder(w).Encode(platforms[0])
					return
				}
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func configDigest(tag string) string {
	var total int
	for _, char := range tag {
		total += int(char)
	}
	return fmt.Sprintf("sha256:%064x", total)
}
