package k8s

import (
	"testing"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeployment(t *testing.T) {
	d := &apps_v1.Deployment{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:        "dep-1",
			Namespace:   "xxxx",
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		apps_v1.DeploymentSpec{
			Template: core_v1.PodTemplateSpec{
				Spec: core_v1.PodSpec{
					Containers: []core_v1.Container{
						{
							Image: "gcr.io/v2-namespace/hello-world:1.1.1",
						},
					},
				},
			},
		},
		apps_v1.DeploymentStatus{},
	}

	gr, err := NewGenericResource(d)
	if err != nil {
		t.Fatalf("failed to create generic resource: %s", err)
	}

	gr.UpdateContainer(0, "hey/there")

	updated, ok := gr.GetResource().(*apps_v1.Deployment)
	if !ok {
		t.Fatalf("conversion failed")
	}

	if updated.Spec.Template.Spec.Containers[0].Image != "hey/there" {
		t.Errorf("unexpected image: %s", updated.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestDeploymentInitContainer(t *testing.T) {
	d := &apps_v1.Deployment{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:        "dep-1",
			Namespace:   "xxxx",
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		apps_v1.DeploymentSpec{
			Template: core_v1.PodTemplateSpec{
				Spec: core_v1.PodSpec{
					Containers: []core_v1.Container{
						{
							Image: "gcr.io/v2-namespace/hello-world:1.1.1",
						},
					},
					InitContainers: []core_v1.Container{
						{
							Image: "gcr.io/v2-namespace/hello-world:1.1.1",
						},
					},
				},
			},
		},
		apps_v1.DeploymentStatus{},
	}

	gr, err := NewGenericResource(d)
	if err != nil {
		t.Fatalf("failed to create generic resource: %s", err)
	}

	gr.UpdateContainer(0, "hey/there")
	gr.UpdateInitContainer(0, "over/here")

	updated, ok := gr.GetResource().(*apps_v1.Deployment)
	if !ok {
		t.Fatalf("conversion failed")
	}

	if updated.Spec.Template.Spec.Containers[0].Image != "hey/there" {
		t.Errorf("unexpected image: %s", updated.Spec.Template.Spec.Containers[0].Image)
	}

	if updated.Spec.Template.Spec.InitContainers[0].Image != "over/here" {
		t.Errorf("unexpected image: %s", updated.Spec.Template.Spec.InitContainers[0].Image)
	}
}

func TestDeploymentMultipleContainers(t *testing.T) {
	d := &apps_v1.Deployment{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:        "dep-1",
			Namespace:   "xxxx",
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		apps_v1.DeploymentSpec{
			Template: core_v1.PodTemplateSpec{
				Spec: core_v1.PodSpec{
					Containers: []core_v1.Container{
						{
							Image: "gcr.io/v2-namespace/hi-world:1.1.1",
						},
						{
							Image: "gcr.io/v2-namespace/hello-world:1.1.1",
						},
						{
							Image: "gcr.io/v2-namespace/bye-world:1.1.1",
						},
					},
				},
			},
		},
		apps_v1.DeploymentStatus{},
	}

	gr, err := NewGenericResource(d)
	if err != nil {
		t.Fatalf("failed to create generic resource: %s", err)
	}

	gr.UpdateContainer(1, "hey/there")

	updated, ok := gr.GetResource().(*apps_v1.Deployment)
	if !ok {
		t.Fatalf("conversion failed")
	}

	if updated.Spec.Template.Spec.Containers[1].Image != "hey/there" {
		t.Errorf("unexpected image: %s", updated.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestStatefulSetMultipleContainers(t *testing.T) {
	d := &apps_v1.StatefulSet{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:        "dep-1",
			Namespace:   "xxxx",
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		apps_v1.StatefulSetSpec{
			Template: core_v1.PodTemplateSpec{
				Spec: core_v1.PodSpec{
					Containers: []core_v1.Container{
						{
							Image: "gcr.io/v2-namespace/hi-world:1.1.1",
						},
						{
							Image: "gcr.io/v2-namespace/hello-world:1.1.1",
						},
						{
							Image: "gcr.io/v2-namespace/bye-world:1.1.1",
						},
					},
				},
			},
		},
		apps_v1.StatefulSetStatus{},
	}

	gr, err := NewGenericResource(d)
	if err != nil {
		t.Fatalf("failed to create generic resource: %s", err)
	}

	gr.UpdateContainer(1, "hey/there")

	updated, ok := gr.GetResource().(*apps_v1.StatefulSet)
	if !ok {
		t.Fatalf("conversion failed")
	}

	if updated.Spec.Template.Spec.Containers[1].Image != "hey/there" {
		t.Errorf("unexpected image: %s", updated.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestDaemonsetlSetMultipleContainers(t *testing.T) {
	d := &apps_v1.DaemonSet{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:        "dep-1",
			Namespace:   "xxxx",
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		apps_v1.DaemonSetSpec{
			Template: core_v1.PodTemplateSpec{
				Spec: core_v1.PodSpec{
					Containers: []core_v1.Container{
						{
							Image: "gcr.io/v2-namespace/hi-world:1.1.1",
						},
						{
							Image: "gcr.io/v2-namespace/hello-world:1.1.1",
						},
						{
							Image: "gcr.io/v2-namespace/bye-world:1.1.1",
						},
					},
				},
			},
		},
		apps_v1.DaemonSetStatus{},
	}

	gr, err := NewGenericResource(d)
	if err != nil {
		t.Fatalf("failed to create generic resource: %s", err)
	}

	gr.UpdateContainer(1, "hey/there")

	updated, ok := gr.GetResource().(*apps_v1.DaemonSet)
	if !ok {
		t.Fatalf("conversion failed")
	}

	if updated.Spec.Template.Spec.Containers[1].Image != "hey/there" {
		t.Errorf("unexpected image: %s", updated.Spec.Template.Spec.Containers[0].Image)
	}
}
