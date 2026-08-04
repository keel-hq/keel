package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApprovalSetHandlerReevaluatesPendingApprovals(t *testing.T) {
	store, teardown := NewTestingUtils()
	defer teardown()

	manager := approvals.New(&approvals.Opts{Store: store})
	resource, err := k8s.NewGenericResource(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storefront",
			Namespace: "keel-demo",
		},
	})
	if err != nil {
		t.Fatalf("create generic resource: %v", err)
	}
	cache := &k8s.GenericResourceCache{}
	cache.Add(resource)
	client := &recordingKubernetesImplementer{}
	server := NewTriggerServer(&Opts{
		GRC:              cache,
		KubernetesClient: client,
		ApprovalManager:  manager,
	})

	identifier := resource.Identifier + ":1.27.6"
	err = manager.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     identifier,
		CurrentVersion: "1.27.5",
		NewVersion:     "1.27.6",
		VotesRequired:  2,
		VotesReceived:  1,
		Deadline:       time.Now().Add(time.Hour),
		Event: &types.Event{Repository: types.Repository{
			Name: "nginx",
			Tag:  "1.27.6",
		}},
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	pendingIdentifier := resource.Identifier + ":1.27.7"
	err = manager.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     pendingIdentifier,
		CurrentVersion: "1.27.5",
		NewVersion:     "1.27.7",
		VotesRequired:  2,
		VotesReceived:  0,
		Deadline:       time.Now().Add(time.Hour),
		Event: &types.Event{Repository: types.Repository{
			Name: "nginx",
			Tag:  "1.27.7",
		}},
	})
	if err != nil {
		t.Fatalf("create second approval: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	approved, err := manager.SubscribeApproved(ctx)
	if err != nil {
		t.Fatalf("subscribe to approved events: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPut,
		"/v1/approvals",
		bytes.NewBufferString(`{"identifier":"deployment/keel-demo/storefront","provider":"kubernetes","votesRequired":1}`),
	)
	rec := httptest.NewRecorder()
	server.approvalSetHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusOK)
	}
	updated, err := manager.Get(identifier)
	if err != nil {
		t.Fatalf("get updated approval: %v", err)
	}
	if updated.VotesRequired != 1 {
		t.Fatalf("unexpected required votes: got %d, want 1", updated.VotesRequired)
	}
	pending, err := manager.Get(pendingIdentifier)
	if err != nil {
		t.Fatalf("get pending approval: %v", err)
	}
	if pending.VotesRequired != 1 || pending.Status() != types.ApprovalStatusPending {
		t.Fatalf(
			"expected second approval to remain pending at 0/1, got %d/%d",
			pending.VotesReceived,
			pending.VotesRequired,
		)
	}

	select {
	case event := <-approved:
		if event.Identifier != identifier {
			t.Fatalf("unexpected approved identifier: got %q, want %q", event.Identifier, identifier)
		}
	case <-time.After(time.Second):
		t.Fatal("expected approval to be reevaluated and published")
	}
}
