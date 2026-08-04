package kubernetes

import (
	"runtime"
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
)

func TestResourcePlatform(t *testing.T) {
	deployment := &apps_v1.Deployment{
		Spec: apps_v1.DeploymentSpec{Template: core_v1.PodTemplateSpec{
			Spec: core_v1.PodSpec{NodeSelector: map[string]string{
				"kubernetes.io/os":   "linux",
				"kubernetes.io/arch": "arm64",
			}},
		}},
	}
	resource, err := k8s.NewGenericResource(deployment)
	if err != nil {
		t.Fatal(err)
	}
	got := resourcePlatform(resource)
	if got.OS != "linux" || got.Architecture != "arm64" {
		t.Fatalf("got platform %#v", got)
	}

	deployment.Spec.Template.Spec.NodeSelector = nil
	got = resourcePlatform(resource)
	if got.OS != runtime.GOOS || got.Architecture != runtime.GOARCH {
		t.Fatalf("got fallback platform %#v", got)
	}
}
