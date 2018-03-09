package kubernetes

import (
	"testing"

	"github.com/keel-hq/keel/types"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestForceUpdate(t *testing.T) {

	fp := &fakeImplementer{}

	dep := &v1beta1.Deployment{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:      "deployment-1",
			Namespace: "xx",
			Labels:    map[string]string{types.KeelPolicyLabel: "all"},
		},
		v1beta1.DeploymentSpec{},
		v1beta1.DeploymentStatus{},
	}

	fp.podList = &v1.PodList{
		Items: []v1.Pod{
			v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "1",
					Namespace: "xx",
				},
			},
			v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "2",
					Namespace: "xx",
				},
			},
		},
	}

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	err = provider.forceUpdate(dep)
	if err != nil {
		t.Fatalf("failed to force update: %s", err)
	}

	if len(fp.deletedPods) != 2 {
		t.Errorf("expected to get 2 deleted pods")
	}

	if fp.deletedPods[0].Namespace != "xx" {
		t.Errorf("wrong namespace: %s", fp.deletedPods[0].Namespace)
	}
	if fp.deletedPods[1].Namespace != "xx" {
		t.Errorf("wrong namespace: %s", fp.deletedPods[1].Namespace)
	}

	if fp.deletedPods[0].Name != "1" {
		t.Errorf("wrong name: %s", fp.deletedPods[0].Name)
	}
	if fp.deletedPods[1].Name != "2" {
		t.Errorf("wrong name: %s", fp.deletedPods[1].Name)
	}
}
