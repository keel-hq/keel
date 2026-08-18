package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/keel-hq/keel/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *E2ESuite) TestWebhookUpdatesEligibleImage() {
	s.requireRegistryTags("webhook", "1.0.0", "1.0.1")
	current := s.cfg.repositoryPrefix + "/webhook:1.0.0"
	desired := s.cfg.repositoryPrefix + "/webhook:1.0.1"
	s.createWorkload("webhook", current, "all", false)

	payload := map[string]any{"events": []map[string]any{{
		"action":  "push",
		"target":  map[string]any{"repository": s.cfg.runID + "/webhook", "tag": "1.0.1", "digest": "sha256:e2e"},
		"request": map[string]any{"host": s.cfg.registry},
	}}}
	body, err := json.Marshal(payload)
	s.Require().NoError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	s.Require().NoError(s.postWebhookUntilImageChanges(ctx, body, "webhook", desired))
}

func (s *E2ESuite) postWebhookUntilImageChanges(ctx context.Context, body []byte, deployment, desired string) error {
	var lastErr error
	for {
		response, err := http.Post(keelForwardURL+"/v1/webhooks/registry", "application/json", bytes.NewReader(body))
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode != http.StatusOK {
				err = fmt.Errorf("webhook returned HTTP %d", response.StatusCode)
			}
		}
		if err != nil {
			lastErr = err
		} else {
			attemptCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			lastErr = waitForDeploymentImage(attemptCtx, s.client, s.testNamespace, deployment, desired)
			cancel()
			if lastErr == nil {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("webhook did not update deployment after bounded retries: %w", lastErr)
		case <-time.After(time.Second):
		}
	}
}

func (s *E2ESuite) TestPollingUpdatesEligibleImage() {
	s.requireRegistryTags("polling", "1.0.0", "1.0.1")
	current := s.cfg.repositoryPrefix + "/polling:1.0.0"
	desired := s.cfg.repositoryPrefix + "/polling:1.0.1"
	s.createWorkload("polling", current, "patch", true)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	s.Require().NoError(waitForDeploymentImage(ctx, s.client, s.testNamespace, "polling", desired))
}

func (s *E2ESuite) TestPollingRejectsIneligibleImage() {
	s.requireRegistryTags("negative", "1.0.0", "1.1.0")
	current := s.cfg.repositoryPrefix + "/negative:1.0.0"
	s.createWorkload("negative", current, "patch", true)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	s.Require().NoError(ensureDeploymentImageUnchanged(ctx, s.client, s.testNamespace, "negative", current))
}

// https://github.com/keel-hq/keel/issues/823: with the force policy, the poll
// trigger must converge on the newest available tag instead of picking the
// first tag in the registry tag list.
func (s *E2ESuite) TestPollingForcePolicyUpdatesToNewestTag() {
	s.requireRegistryTags("force-latest", "1.0.0", "1.0.1", "1.0.2")
	repository := s.cfg.repositoryPrefix + "/force-latest"
	s.createWorkload("force-latest", repository+":1.0.0", "force", true)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	s.Require().NoError(waitForDeploymentImage(ctx, s.client, s.testNamespace, "force-latest", repository+":1.0.2"))
}

// https://github.com/keel-hq/keel/issues/823 (8.1.0 -> 8.0.0 report): the
// force policy must never update a workload to an older semantic version.
func (s *E2ESuite) TestPollingForcePolicyDoesNotDowngrade() {
	s.requireRegistryTags("force-nodowngrade", "1.0.0", "1.0.1")
	repository := s.cfg.repositoryPrefix + "/force-nodowngrade"
	current := repository + ":1.0.1"
	s.createWorkload("force-nodowngrade", current, "force", true)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	s.Require().NoError(ensureDeploymentImageUnchanged(ctx, s.client, s.testNamespace, "force-nodowngrade", current))
}

// https://github.com/keel-hq/keel/issues/823 (original report): a workload on
// cp-kafka:7.9.1-1-ubi8 was "updated" to cp-kafka:3.0.0 because the oldest
// tag in the registry list won. With no newer tag available the image must
// remain untouched.
func (s *E2ESuite) TestPollingForcePolicyIgnoresOlderTagsForPrereleaseCurrent() {
	s.requireRegistryTags("force-prerelease", "3.0.0", "5.0.0", "7.9.1-1-ubi8")
	repository := s.cfg.repositoryPrefix + "/force-prerelease"
	current := repository + ":7.9.1-1-ubi8"
	s.createWorkload("force-prerelease", current, "force", true)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	s.Require().NoError(ensureDeploymentImageUnchanged(ctx, s.client, s.testNamespace, "force-prerelease", current))
}

func (s *E2ESuite) TestWebhookStatefulSetInitContainerRemainsIsolatedFromPolling() {
	const (
		initialTag = "main_0000000000000000000000000000000000000000"
		webhookTag = "main_1111111111111111111111111111111111111111"
		pollTag    = "main_ffffffffffffffffffffffffffffffffffffffff"
		policy     = `regexp:^main_[a-fA-F0-9]{40}$`
	)
	s.requireRegistryTags("webhook-regression", initialTag, webhookTag, pollTag)
	repository := s.cfg.repositoryPrefix + "/webhook-regression"
	initial := repository + ":" + initialTag
	webhookSelected := repository + ":" + webhookTag
	pollSelected := repository + ":" + pollTag

	labels := map[string]string{"app": "webhook-regression", e2eRunLabel: s.cfg.runID}
	_, err := s.client.AppsV1().StatefulSets(s.testNamespace).Create(context.Background(), &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-regression",
			Namespace: s.testNamespace,
			Labels:    labels,
			Annotations: map[string]string{
				types.KeelPolicyLabel:             policy,
				types.KeelTriggerLabel:            "webhook",
				types.KeelInitContainerAnnotation: "true",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "webhook-regression",
			Replicas:    ptrTo(int32(1)),
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{
						Name:    "tracked-init",
						Image:   initial,
						Command: []string{"sh", "-c", "true"},
					}},
					Containers: []corev1.Container{{
						Name:    "fixture",
						Image:   initial,
						Command: []string{"sh", "-c", "sleep 3600"},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})
	s.Require().NoError(err)

	// A legitimate poll consumer shares the repository. Its watcher must not
	// select tags for, or apply poll events to, the webhook-only StatefulSet.
	s.createWorkload("webhook-regression-poll", initial, policy, true)

	payload := map[string]any{"events": []map[string]any{{
		"action":  "push",
		"target":  map[string]any{"repository": s.cfg.runID + "/webhook-regression", "tag": webhookTag, "digest": "sha256:e2e"},
		"request": map[string]any{"host": s.cfg.registry},
	}}}
	body, err := json.Marshal(payload)
	s.Require().NoError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	for {
		response, postErr := http.Post(keelForwardURL+"/v1/webhooks/registry", "application/json", bytes.NewReader(body))
		if postErr == nil {
			_ = response.Body.Close()
		}
		attemptCtx, attemptCancel := context.WithTimeout(ctx, 3*time.Second)
		waitErr := waitForStatefulSetInitImage(attemptCtx, s.client, s.testNamespace, "webhook-regression", webhookSelected)
		attemptCancel()
		if waitErr == nil {
			break
		}
		select {
		case <-ctx.Done():
			s.Require().NoError(waitErr)
		case <-time.After(time.Second):
		}
	}

	s.Require().NoError(waitForDeploymentImage(ctx, s.client, s.testNamespace, "webhook-regression-poll", pollSelected))
	unchangedCtx, unchangedCancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer unchangedCancel()
	s.Require().NoError(ensureStatefulSetInitImageUnchanged(unchangedCtx, s.client, s.testNamespace, "webhook-regression", webhookSelected))
	s.T().Logf("poll control advanced to %s while webhook StatefulSet init image remained %s", pollSelected, webhookSelected)
}

func ptrTo[T any](value T) *T {
	return &value
}

func (s *E2ESuite) createWorkload(name, image, policy string, polling bool) {
	labels := map[string]string{"app": name, e2eRunLabel: s.cfg.runID}
	annotations := map[string]string{types.KeelPolicyLabel: policy}
	if polling {
		labels[types.KeelTriggerLabel] = "poll"
		annotations[types.KeelPollScheduleAnnotation] = "@every 2s"
	}
	replicas := int32(1)
	_, err := s.client.AppsV1().Deployments(s.testNamespace).Create(context.Background(), &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.testNamespace, Labels: labels, Annotations: annotations},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:            "fixture",
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"sh", "-c", "sleep 3600"},
				}}},
			},
		},
	}, metav1.CreateOptions{})
	s.Require().NoError(err, "create workload %s in namespace %s", name, s.testNamespace)
}
