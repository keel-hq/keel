package kubernetes

import (
	"context"
	"errors"
	"testing"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	conflictTestNamespace = "xxxx"
	conflictTestName      = "dep-1"
)

// testDeployment builds a minimal deployment with the given resourceVersion
// and container image.
func testDeployment(resourceVersion, image string) *apps_v1.Deployment {
	return &apps_v1.Deployment{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:            conflictTestName,
			Namespace:       conflictTestNamespace,
			ResourceVersion: resourceVersion,
		},
		apps_v1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{Image: image}},
				},
			},
		},
		apps_v1.DeploymentStatus{},
	}
}

// TestUpdateRetriesOnConflict verifies that a 409 Conflict on the first update
// attempt is retried: the retried attempt re-fetches the object, carries the
// fresh resourceVersion, and succeeds.
func TestUpdateRetriesOnConflict(t *testing.T) {
	const (
		staleRV  = "1"
		freshRV  = "10"
		oldImage = "gcr.io/v2-namespace/hello-world:1.0.0"
		newImage = "gcr.io/v2-namespace/hello-world:2.0.0"
	)

	// The object as it currently exists on the API server.
	latest := testDeployment(freshRV, oldImage)
	client := fake.NewSimpleClientset(latest)

	// The first update attempt is rejected with 409 Conflict; subsequent
	// attempts fall through to the default (object tracker) behavior.
	var updateRVs []string
	client.PrependReactor("update", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		dep, ok := action.(k8stesting.UpdateAction).GetObject().(*apps_v1.Deployment)
		if !ok {
			t.Fatalf("expected a deployment update action, got %T", action)
		}
		updateRVs = append(updateRVs, dep.ResourceVersion)
		if len(updateRVs) == 1 {
			return true, nil, apierrors.NewConflict(
				schema.GroupResource{Group: "apps", Resource: "deployments"},
				conflictTestName,
				errors.New("object has been modified; if applying changes, get a newer version and try again"),
			)
		}
		return false, nil, nil
	})

	// Keel holds a stale copy: same name/namespace, older resourceVersion,
	// with the new image to be applied.
	stale := testDeployment(staleRV, newImage)

	impl := &KubernetesImplementer{client: client}
	err := impl.Update(MustParseGR(stale))
	if err != nil {
		t.Fatalf("expected Update to succeed after conflict retry, got: %v", err)
	}

	if len(updateRVs) != 2 {
		t.Fatalf("expected the update to be attempted twice, got %d attempts", len(updateRVs))
	}
	for i, rv := range updateRVs {
		if rv != freshRV {
			t.Errorf("update attempt %d carried resourceVersion %q, want the fresh %q from the re-fetch", i+1, rv, freshRV)
		}
	}

	updated, err := client.AppsV1().Deployments(conflictTestNamespace).Get(context.TODO(), conflictTestName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get updated deployment: %v", err)
	}
	if got := updated.Spec.Template.Spec.Containers[0].Image; got != newImage {
		t.Errorf("expected deployment image %q, got %q", newImage, got)
	}
	if updated.ResourceVersion != freshRV {
		t.Errorf("expected updated object to carry the fresh resourceVersion %q, got %q", freshRV, updated.ResourceVersion)
	}
}

// TestUpdateDoesNotRetryOnForbidden verifies that non-conflict API errors are
// returned immediately without retry.
func TestUpdateDoesNotRetryOnForbidden(t *testing.T) {
	latest := testDeployment("10", "gcr.io/v2-namespace/hello-world:1.0.0")
	client := fake.NewSimpleClientset(latest)

	var updateCount int
	client.PrependReactor("update", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateCount++
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "apps", Resource: "deployments"},
			conflictTestName,
			errors.New("forbidden: cannot update deployment"),
		)
	})

	stale := testDeployment("1", "gcr.io/v2-namespace/hello-world:2.0.0")

	impl := &KubernetesImplementer{client: client}
	err := impl.Update(MustParseGR(stale))
	if err == nil {
		t.Fatal("expected an error from Update, got nil")
	}
	if !apierrors.IsForbidden(err) {
		t.Fatalf("expected a forbidden error, got: %v", err)
	}
	if updateCount != 1 {
		t.Fatalf("expected the update to be attempted exactly once without retry, got %d attempts", updateCount)
	}
}
