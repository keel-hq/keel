package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	providerkubernetes "github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type recordingKubernetesImplementer struct {
	providerkubernetes.Implementer
	updated *k8s.GenericResource
}

func (i *recordingKubernetesImplementer) Update(resource *k8s.GenericResource) error {
	i.updated = resource
	return nil
}

func TestPolicyUpdateHandlerClearsPolicy(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      "storefront",
		Namespace: "keel-demo",
		Labels: map[string]string{
			types.KeelPolicyLabel:  "minor",
			"keel.observer/policy": "patch",
		},
		Annotations: map[string]string{
			types.KeelPolicyLabel:  "major",
			"keel.observer/policy": "all",
		},
	}}
	resource, err := k8s.NewGenericResource(deployment)
	if err != nil {
		t.Fatalf("create generic resource: %v", err)
	}
	cache := &k8s.GenericResourceCache{}
	cache.Add(resource)
	client := &recordingKubernetesImplementer{}
	server := NewTriggerServer(&Opts{GRC: cache, KubernetesClient: client})

	req := httptest.NewRequest(
		http.MethodPut,
		"/v1/policies",
		bytes.NewBufferString(`{"identifier":"deployment/keel-demo/storefront","provider":"kubernetes","policy":""}`),
	)
	rec := httptest.NewRecorder()
	server.policyUpdateHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}
	if client.updated == nil {
		t.Fatal("expected Kubernetes resource to be updated")
	}
	for _, values := range []map[string]string{
		client.updated.GetLabels(),
		client.updated.GetAnnotations(),
	} {
		if _, ok := values[types.KeelPolicyLabel]; ok {
			t.Errorf("expected %q to be removed", types.KeelPolicyLabel)
		}
		if _, ok := values["keel.observer/policy"]; ok {
			t.Error("expected legacy policy key to be removed")
		}
	}
}
