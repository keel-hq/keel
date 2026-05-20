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

func TestDeploymentImageVolume(t *testing.T) {
	d := &apps_v1.Deployment{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:      "dep-1",
			Namespace: "xxxx",
		},
		apps_v1.DeploymentSpec{
			Template: core_v1.PodTemplateSpec{
				Spec: core_v1.PodSpec{
					Containers: []core_v1.Container{
						{Image: "gcr.io/v2-namespace/hello-world:1.1.1"},
					},
					Volumes: []core_v1.Volume{
						{
							Name: "config",
							VolumeSource: core_v1.VolumeSource{
								ConfigMap: &core_v1.ConfigMapVolumeSource{},
							},
						},
						{
							Name: "oci-config",
							VolumeSource: core_v1.VolumeSource{
								Image: &core_v1.ImageVolumeSource{
									Reference:  "gcr.io/v2-namespace/oci-config:1.0.0",
									PullPolicy: core_v1.PullIfNotPresent,
								},
							},
						},
						{
							Name: "oci-empty",
							VolumeSource: core_v1.VolumeSource{
								Image: &core_v1.ImageVolumeSource{},
							},
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

	refs := gr.GetImageVolumeReferences(nil)
	if len(refs) != 1 || refs[0] != "gcr.io/v2-namespace/oci-config:1.0.0" {
		t.Fatalf("unexpected image volume references: %v", refs)
	}

	gr.UpdateImageVolume(1, "gcr.io/v2-namespace/oci-config:1.1.0")

	updated, ok := gr.GetResource().(*apps_v1.Deployment)
	if !ok {
		t.Fatalf("conversion failed")
	}
	if got := updated.Spec.Template.Spec.Volumes[1].Image.Reference; got != "gcr.io/v2-namespace/oci-config:1.1.0" {
		t.Errorf("unexpected updated reference: %s", got)
	}

	filter := func(v core_v1.Volume) bool { return v.Name == "config" }
	if refs := gr.GetImageVolumeReferences(filter); len(refs) != 0 {
		t.Errorf("expected zero references when filter excludes image volume, got %v", refs)
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
