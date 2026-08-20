package kubernetes

import (
	"errors"
	"testing"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// failingSender mimics a notification sender (e.g. a webhook) whose endpoint
// rejects every notification (e.g. returns HTTP 400). It records how many times
// it was invoked so the test can prove the update loop does not stop on the
// first failure.
type failingSender struct {
	attempts int
}

// Configure always reports the sender as enabled.
func (s *failingSender) Configure(cfg *notification.Config) (bool, error) {
	return true, nil
}

// Send always returns an error and records the attempt.
func (s *failingSender) Send(event types.EventNotification) error {
	s.attempts++
	return errors.New("simulated webhook delivery failure: HTTP 400")
}

// recordingImplementer records every resource it is asked to update so the test
// can assert that all deployments sharing an image were updated, even when the
// notification sender fails.
type recordingImplementer struct {
	fakeImplementer
	updates []*k8s.GenericResource
}

func (i *recordingImplementer) Update(obj *k8s.GenericResource) error {
	i.updates = append(i.updates, obj)
	return nil
}

// TestFailingWebhookDoesNotBlockOtherDeployments is a regression test for
// keel-hq/keel#822: a failing webhook notification must not prevent other
// deployments that share the same image from being updated in the same poll
// cycle.
func TestFailingWebhookDoesNotBlockOtherDeployments(t *testing.T) {
	const sharedImage = "gcr.io/v2-namespace/hello-world:1.0.0"

	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-1",
				Namespace:   "ns",
				Labels:      map[string]string{types.KeelPolicyLabel: "all", types.KeelTriggerLabel: "poll"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{{Name: "app", Image: sharedImage}},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-2",
				Namespace:   "ns",
				Labels:      map[string]string{types.KeelPolicyLabel: "all", types.KeelTriggerLabel: "poll"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{{Name: "app", Image: sharedImage}},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	fi := &recordingImplementer{}
	fs := &failingSender{}
	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fi, fs, approver, grc)
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	event := &types.Event{
		Repository:  types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
		TriggerName: types.TriggerTypePoll.String(),
	}

	// Even though the sender always fails, processing the event must not error
	// out and must not stop short of updating the remaining deployments.
	if _, err := provider.processEvent(event); err != nil {
		t.Fatalf("processEvent returned an error caused by a failing webhook: %s", err)
	}

	if len(fi.updates) != 2 {
		t.Fatalf("expected both deployments sharing the image to be updated, got %d updates", len(fi.updates))
	}

	for _, r := range fi.updates {
		if r.Containers()[0].Image != "gcr.io/v2-namespace/hello-world:1.1.0" {
			t.Errorf("deployment %q was not updated to the new image, got %q", r.Name, r.Containers()[0].Image)
		}
	}

	// the failing notification must have been attempted for every deployment,
	// proving the loop never aborted on a notification failure.
	if fs.attempts == 0 {
		t.Errorf("expected the failing webhook to be invoked, but it was never called")
	}
}
