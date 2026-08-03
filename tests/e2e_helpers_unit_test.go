package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWaitForDeploymentImageReportsLastObservation(t *testing.T) {
	client := fake.NewSimpleClientset(testDeployment("ns", "app", "registry/app:1.0.0"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := waitForDeploymentImage(ctx, client, "ns", "app", "registry/app:1.0.1")
	if err == nil || !strings.Contains(err.Error(), `last observed "registry/app:1.0.0"`) {
		t.Fatalf("expected last observed image in error, got %v", err)
	}
}

func TestEnsureDeploymentImageUnchangedObservesWholeWindow(t *testing.T) {
	client := fake.NewSimpleClientset(testDeployment("ns", "app", "registry/app:1.0.0"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := ensureDeploymentImageUnchanged(ctx, client, "ns", "app", "registry/app:1.0.0"); err != nil {
		t.Fatalf("expected unchanged image: %v", err)
	}
}

func testDeployment(namespace, name, image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: image}}},
		}},
	}
}
