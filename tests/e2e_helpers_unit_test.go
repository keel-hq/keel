package tests

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
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

func TestEnsureStatefulSetInitImageUnchangedPreservesSuccessfulObservation(t *testing.T) {
	const expected = "registry/app:1.0.0"
	client := fake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "app"},
		Spec: appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{InitContainers: []corev1.Container{{Name: "init", Image: expected}}},
		}},
	})
	requests := 0
	client.Fake.PrependReactor("get", "statefulsets", func(k8stesting.Action) (bool, runtime.Object, error) {
		requests++
		if requests > 1 {
			return true, nil, errors.New("client rate limiter Wait returned an error: rate: Wait(n=1) would exceed context deadline")
		}
		return false, nil, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := ensureStatefulSetInitImageUnchanged(ctx, client, "ns", "app", expected); err != nil {
		t.Fatalf("expected final client error not to replace successful observation: %v", err)
	}
	if requests < 2 {
		t.Fatalf("expected a successful observation followed by a client error, got %d request(s)", requests)
	}
}

func TestWaitForDeploymentAvailableReportsLastObservation(t *testing.T) {
	deployment := testDeployment("ns", "app", "registry/app:1.0.0")
	deployment.Generation = 2
	deployment.Status.ObservedGeneration = 1
	client := fake.NewSimpleClientset(deployment)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := waitForDeploymentAvailable(ctx, client, "ns", "app")
	if err == nil || !strings.Contains(err.Error(), "generation=2 observedGeneration=1 available=0 desired=1") {
		t.Fatalf("expected last observed readiness state in error, got %v", err)
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
