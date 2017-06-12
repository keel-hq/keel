package kubernetes

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/types"
	// "github.com/rusenask/keel/util/version"

	"testing"
)

type fakeImplementer struct {
	namespaces     *v1.NamespaceList
	deployment     *v1beta1.Deployment
	deploymentList *v1beta1.DeploymentList
}

func (i *fakeImplementer) Namespaces() (*v1.NamespaceList, error) {
	return i.namespaces, nil
}

func (i *fakeImplementer) Deployment(namespace, name string) (*v1beta1.Deployment, error) {
	return i.deployment, nil
}

func (i *fakeImplementer) Deployments(namespace string) (*v1beta1.DeploymentList, error) {
	return i.deploymentList, nil
}

func (i *fakeImplementer) Update(deployment *v1beta1.Deployment) error {
	return nil
}

func TestGetNamespaces(t *testing.T) {
	fi := &fakeImplementer{
		namespaces: &v1.NamespaceList{
			Items: []v1.Namespace{
				v1.Namespace{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{Name: "xxxx"},
					v1.NamespaceSpec{},
					v1.NamespaceStatus{},
				},
			},
		},
	}

	provider, err := NewProvider(fi)
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	namespaces, err := provider.namespaces()
	if err != nil {
		t.Errorf("failed to get namespaces: %s", err)
	}

	if namespaces.Items[0].Name != "xxxx" {
		t.Errorf("expected xxxx but got %s", namespaces.Items[0].Name)
	}
}

func TestGetImageName(t *testing.T) {
	name := versionreg.ReplaceAllString("gcr.io/v2-namespace/hello-world:1.1", "")
	if name != "gcr.io/v2-namespace/hello-world" {
		t.Errorf("expected 'gcr.io/v2-namespace/hello-world' but got '%s'", name)
	}
}

func TestGetDeployments(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			v1.Namespace{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	fp.deploymentList = &v1beta1.DeploymentList{
		Items: []v1beta1.Deployment{
			v1beta1.Deployment{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{
					Name:      "dep-1",
					Namespace: "xxxx",
					Labels:    map[string]string{types.KeelPolicyLabel: "all"},
				},
				v1beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "gcr.io/v2-namespace/hello-world:1.1",
								},
							},
						},
					},
				},
				v1beta1.DeploymentStatus{},
			},
		},
	}

	provider, err := NewProvider(fp)
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	deps, err := provider.deployments()
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}
	if len(deps) != 1 {
		t.Errorf("expected to find 1 deployment, got: %d", len(deps))
	}

	if deps[0].Items[0].GetName() != "dep-1" {
		t.Errorf("expected name %s, got %s", "dep-1", deps[0].Items[0].GetName())
	}
}

func TestGetImpacted(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			v1.Namespace{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	fp.deploymentList = &v1beta1.DeploymentList{
		Items: []v1beta1.Deployment{
			v1beta1.Deployment{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{
					Name:      "dep-1",
					Namespace: "xxxx",
					Labels:    map[string]string{types.KeelPolicyLabel: "all"},
				},
				v1beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "gcr.io/v2-namespace/hello-world:1.1.1",
								},
							},
						},
					},
				},
				v1beta1.DeploymentStatus{},
			},
			v1beta1.Deployment{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{
					Name:      "dep-2",
					Namespace: "xxxx",
					Labels:    map[string]string{"whatever": "all"},
				},
				v1beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "gcr.io/v2-namespace/hello-world:1.1.1",
								},
							},
						},
					},
				},
				v1beta1.DeploymentStatus{},
			},
		},
	}

	provider, err := NewProvider(fp)
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	// creating "new version" event
	repo := &types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "1.1.2",
	}

	deps, err := provider.impactedDeployments(repo)
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}

	if len(deps) != 1 {
		t.Errorf("expected to find 1 deployment but found %s", len(deps))
	}

	found := false
	for _, c := range deps[0].Spec.Template.Spec.Containers {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

// func TestProcessEvent(t *testing.T) {
// 	provider, err := NewProvider(&fakeImplementer{})
// 	if err != nil {
// 		t.Fatalf("failed to get provider: %s", err)
// 	}

// 	repo := types.Repository{
// 		Name: "karolisr/webhook-demo",
// 		Tag:  newVersion,
// 	}

// 	event := &types.Event{Repository: repo}
// 	updated, err := provider.processEvent(event)
// 	if err != nil {
// 		t.Errorf("got error while processing event: %s", err)
// 	}

// 	//
// 	time.Sleep(100 * time.Millisecond)
// 	for _, upd := range updated {
// 		current, err := provider.getDeployment(upd.Namespace, upd.Name)
// 		if err != nil {
// 			t.Fatalf("failed to get deployment %s, error: %s", upd.Name, err)
// 		}
// 		currentVer, err := version.GetVersionFromImageName(current.Spec.Template.Spec.Containers[0].Image)
// 		if err != nil {
// 			t.Fatalf("failed to get version from %s, error: %s", current.Spec.Template.Spec.Containers[0].Image, err)
// 		}

// 		if currentVer.String() != newVersion {
// 			t.Errorf("deployment version wasn't updated, got: %s while expected: %s", currentVer.String(), newVersion)
// 		}
// 	}

// }
