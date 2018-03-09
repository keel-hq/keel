package kubernetes

import (
	"testing"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type fakeProvider struct {
	submitted []types.Event
	images    []*types.TrackedImage
}

func (p *fakeProvider) Submit(event types.Event) error {
	p.submitted = append(p.submitted, event)
	return nil
}

func (p *fakeProvider) TrackedImages() ([]*types.TrackedImage, error) {
	return p.images, nil
}
func (p *fakeProvider) List() []string {
	return []string{"fakeprovider"}
}
func (p *fakeProvider) Stop() {
	return
}
func (p *fakeProvider) GetName() string {
	return "fp"
}

type fakeImplementer struct {
	namespaces     *v1.NamespaceList
	deployment     *v1beta1.Deployment
	deploymentList *v1beta1.DeploymentList

	podList     *v1.PodList
	deletedPods []*v1.Pod

	// stores value of an updated deployment
	updated *v1beta1.Deployment

	availableSecret *v1.Secret
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
	i.updated = deployment
	return nil
}

func (i *fakeImplementer) Secret(namespace, name string) (*v1.Secret, error) {
	return i.availableSecret, nil
}

func (i *fakeImplementer) Pods(namespace, labelSelector string) (*v1.PodList, error) {
	return i.podList, nil
}

func (i *fakeImplementer) DeletePod(namespace, name string, opts *meta_v1.DeleteOptions) error {
	i.deletedPods = append(i.deletedPods, &v1.Pod{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		v1.PodSpec{},
		v1.PodStatus{},
	})
	return nil
}

func (i *fakeImplementer) ConfigMaps(namespace string) core_v1.ConfigMapInterface {
	return nil
}

type fakeSender struct {
	sentEvent types.EventNotification
}

func (s *fakeSender) Configure(cfg *notification.Config) (bool, error) {
	return true, nil
}

func (s *fakeSender) Send(event types.EventNotification) error {
	s.sentEvent = event
	return nil
}

func approver() *approvals.DefaultManager {
	cache := memory.NewMemoryCache(10*time.Minute, 10*time.Minute, 10*time.Minute)

	return approvals.New(cache, codecs.DefaultSerializer())
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

	provider, err := NewProvider(fi, &fakeSender{}, approver())
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

	provider, err := NewProvider(fp, &fakeSender{}, approver())
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

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	// creating "new version" event
	repo := &types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "1.1.2",
	}

	plans, err := provider.createUpdatePlans(repo)
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}

	if len(plans) != 1 {
		t.Errorf("expected to find 1 deployment update plan but found %d", len(plans))
	}

	found := false
	for _, c := range plans[0].Deployment.Spec.Template.Spec.Containers {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

func TestProcessEvent(t *testing.T) {
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
					Name:      "deployment-1",
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
					Name:      "deployment-2",
					Namespace: "xxxx",
					Labels:    map[string]string{"whatever": "all"},
				},
				v1beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "gcr.io/v2-namespace/bye-world:1.1.1",
								},
							},
						},
					},
				},
				v1beta1.DeploymentStatus{},
			},
		},
	}

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	repo := types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "1.4.5",
	}

	event := &types.Event{Repository: repo}
	_, err = provider.processEvent(event)
	if err != nil {
		t.Errorf("got error while processing event: %s", err)
	}

	if fp.updated.Spec.Template.Spec.Containers[0].Image != repo.Name+":"+repo.Tag {
		t.Errorf("expected to find a deployment with updated image but found: %s", fp.updated.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestProcessEventBuildNumber(t *testing.T) {
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
					Name:      "deployment-1",
					Namespace: "xxxx",
					Labels:    map[string]string{types.KeelPolicyLabel: "all"},
				},
				v1beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "gcr.io/v2-namespace/hello-world:10",
								},
							},
						},
					},
				},
				v1beta1.DeploymentStatus{},
			},
		},
	}

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	repo := types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "11",
	}

	event := &types.Event{Repository: repo}
	_, err = provider.processEvent(event)
	if err != nil {
		t.Errorf("got error while processing event: %s", err)
	}

	if fp.updated.Spec.Template.Spec.Containers[0].Image != repo.Name+":"+repo.Tag {
		t.Errorf("expected to find a deployment with updated image but found: %s", fp.updated.Spec.Template.Spec.Containers[0].Image)
	}
}

// Test to check how many deployments are "impacted" if we have sidecar container
func TestGetImpactedTwoContainersInSameDeployment(t *testing.T) {
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
								v1.Container{
									Image: "gcr.io/v2-namespace/greetings-world:1.1.1",
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

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	// creating "new version" event
	repo := &types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "1.1.2",
	}

	plans, err := provider.createUpdatePlans(repo)
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}

	if len(plans) != 1 {
		t.Errorf("expected to find 1 deployment but found %d", len(plans))
	}

	found := false
	for _, c := range plans[0].Deployment.Spec.Template.Spec.Containers {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

func TestGetImpactedTwoSameContainersInSameDeployment(t *testing.T) {

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

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	// creating "new version" event
	repo := &types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "1.1.2",
	}

	plans, err := provider.createUpdatePlans(repo)
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}

	if len(plans) != 1 {
		t.Errorf("expected to find 1 deployment but found %d", len(plans))
	}

	found := false
	for _, c := range plans[0].Deployment.Spec.Template.Spec.Containers {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

func TestGetImpactedUntaggedImage(t *testing.T) {
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
					Name:        "dep-1",
					Namespace:   "xxxx",
					Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					Annotations: map[string]string{},
				},
				v1beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "gcr.io/v2-namespace/foo-world",
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
					Name:        "dep-2",
					Namespace:   "xxxx",
					Annotations: map[string]string{},
					Labels:      map[string]string{types.KeelPolicyLabel: "all"},
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

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	// creating "new version" event
	repo := &types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "1.1.2",
	}

	plans, err := provider.createUpdatePlans(repo)
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}

	if len(plans) != 1 {
		t.Errorf("expected to find 1 deployment but found %d", len(plans))
	}

	found := false
	for _, c := range plans[0].Deployment.Spec.Template.Spec.Containers {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

// test to check whether we get impacted deployment when it's untagged (we should)
func TestGetImpactedUntaggedOneImage(t *testing.T) {
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
					Name:        "dep-1",
					Namespace:   "xxxx",
					Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					Annotations: map[string]string{},
				},
				v1beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								v1.Container{
									Image: "gcr.io/v2-namespace/hello-world",
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
					Name:        "dep-2",
					Namespace:   "xxxx",
					Annotations: map[string]string{},
					Labels:      map[string]string{types.KeelPolicyLabel: "all"},
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

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	// creating "new version" event
	repo := &types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "1.1.2",
	}

	plans, err := provider.createUpdatePlans(repo)
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}

	if len(plans) != 2 {
		t.Fatalf("expected to find 2 deployment but found %d", len(plans))
	}

	found := false
	for _, plan := range plans {
		for _, c := range plan.Deployment.Spec.Template.Spec.Containers {

			containerImageName := versionreg.ReplaceAllString(c.Image, "")

			if containerImageName == repo.Name {
				found = true
			}
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

func TestTrackedImages(t *testing.T) {
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
							ImagePullSecrets: []v1.LocalObjectReference{
								v1.LocalObjectReference{
									Name: "very-secret",
								},
							},
						},
					},
				},
				v1beta1.DeploymentStatus{},
			},
		},
	}

	provider, err := NewProvider(fp, &fakeSender{}, approver())
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	imgs, err := provider.TrackedImages()
	if err != nil {
		t.Errorf("failed to get image: %s", err)
	}
	if len(imgs) != 1 {
		t.Errorf("expected to find 1 image, got: %d", len(imgs))
	}

	if imgs[0].Secrets[0] != "very-secret" {
		t.Errorf("could not find image pull secret")
	}
}
