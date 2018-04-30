package k8s

import (
	"testing"

	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddGet(t *testing.T) {

	cc := &GenericResourceCache{}

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

	cc.Add(gr)

	// updating deployment
	stored := cc.Values()[0]
	stored.UpdateContainer(0, "gcr.io/v2-namespace/hi-world:2.2.2.")

	// getting again
	stored2 := cc.Values()[0]
	if stored2.Containers()[0].Image != "gcr.io/v2-namespace/hi-world:1.1.1" {
		t.Errorf("cached entry got modified: %s", stored2.Containers()[0].Image)
	}
}
