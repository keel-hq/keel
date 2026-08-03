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

func (s *E2ESuite) createWorkload(name, image, policy string, polling bool) {
	labels := map[string]string{"app": name, types.KeelPolicyLabel: policy, e2eRunLabel: s.cfg.runID}
	annotations := map[string]string{}
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
