package kubernetes

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/pkg/store/sql"
	"github.com/keel-hq/keel/types"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
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
	deployment     *apps_v1.Deployment
	deploymentList *apps_v1.DeploymentList

	podList     *v1.PodList
	deletedPods []*v1.Pod

	// stores value of an updated deployment
	updated *k8s.GenericResource

	availableSecret *v1.Secret
}

func (i *fakeImplementer) Namespaces() (*v1.NamespaceList, error) {
	return i.namespaces, nil
}

func (i *fakeImplementer) Deployment(namespace, name string) (*apps_v1.Deployment, error) {
	return i.deployment, nil
}

func (i *fakeImplementer) Deployments(namespace string) (*apps_v1.DeploymentList, error) {
	return i.deploymentList, nil
}

func (i *fakeImplementer) Update(obj *k8s.GenericResource) error {
	i.updated = obj
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

func NewTestingUtils() (*sql.SQLStore, func()) {
	dir, err := ioutil.TempDir("", "whstoretest")
	if err != nil {
		log.Fatal(err)
	}
	tmpfn := filepath.Join(dir, "gorm.db")
	// defer
	store, err := sql.New(sql.Opts{DatabaseType: "sqlite3", URI: tmpfn})
	if err != nil {
		log.Fatal(err)
	}

	teardown := func() {
		os.RemoveAll(dir) // clean up
	}

	return store, teardown
}

func approver() (*approvals.DefaultManager, func()) {
	store, teardown := NewTestingUtils()
	return approvals.New(&approvals.Opts{
		Store: store,
	}), teardown
}

func TestGetNamespaces(t *testing.T) {
	fi := &fakeImplementer{
		namespaces: &v1.NamespaceList{
			Items: []v1.Namespace{
				{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{Name: "xxxx"},
					v1.NamespaceSpec{},
					v1.NamespaceStatus{},
				},
			},
		},
	}

	grc := &k8s.GenericResourceCache{}

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fi, &fakeSender{}, approver, grc)
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

func MustParseGR(obj interface{}) *k8s.GenericResource {
	gr, err := k8s.NewGenericResource(obj)
	if err != nil {
		panic(err)
	}
	return gr
}

func MustParseGRS(objs []*apps_v1.Deployment) []*k8s.GenericResource {
	grs := make([]*k8s.GenericResource, len(objs))
	for idx, obj := range objs {
		var err error
		var gr *k8s.GenericResource
		gr, err = k8s.NewGenericResource(obj)
		if err != nil {
			panic(err)
		}
		grs[idx] = gr
	}
	return grs
}

func TestGetImpacted(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}

	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-1",
				Namespace: "xxxx",
				Labels:    map[string]string{types.KeelPolicyLabel: "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-2",
				Namespace: "xxxx",
				Labels:    map[string]string{"whatever": "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
		t.Fatalf("expected to find 1 deployment update plan but found %d", len(plans))
	}

	found := false
	for _, c := range plans[0].Resource.Containers() {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

func TestGetImpactedInit(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}

	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-1",
				Namespace:   "xxxx",
				Annotations: map[string]string{types.KeelInitContainerAnnotation: "true"},
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						InitContainers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-2",
				Namespace:   "xxxx",
				Annotations: map[string]string{types.KeelInitContainerAnnotation: "false"},
				Labels:      map[string]string{"whatever": "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						InitContainers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
		t.Fatalf("expected to find 1 deployment update plan but found %d", len(plans))
	}

	found := false
	for _, c := range plans[0].Resource.InitContainers() {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

func TestGetImpactedPolicyAnnotations(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}

	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-1",
				Namespace:   "xxxx",
				Annotations: map[string]string{types.KeelPolicyLabel: "all"},
				Labels:      map[string]string{"foo": "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-2",
				Namespace: "xxxx",
				Labels:    map[string]string{"whatever": "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
		t.Fatalf("expected to find 1 deployment update plan but found %d", len(plans))
	}

	found := false
	for _, c := range plans[0].Resource.Containers() {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}
func TestPrereleaseGetImpactedA(t *testing.T) {
	// test scenario when we have two deployments, one with pre-release tag
	// and one without. New image comes without the prerelease tag. Expected scenario
	// is to get one update plan for the second deployment. Deployment with prerelease tag
	// should be ignored

	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}

	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-1",
				Namespace: "xxxx",
				Labels:    map[string]string{types.KeelPolicyLabel: "major"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1-staging",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-2",
				Namespace: "xxxx",
				Labels:    map[string]string{types.KeelPolicyLabel: "major"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
		t.Fatalf("expected to find 1 deployment update plan but found %d", len(plans))
	}

	if plans[0].Resource.Identifier != "deployment/xxxx/dep-2" {
		t.Errorf("expected to get 'deployment/xxxx/dep-2', but got: %s", plans[0].Resource.Identifier)
	}
}

func TestPrereleaseGetImpactedB(t *testing.T) {
	// test scenario when we have two deployments, one with pre-release tag
	// and one without. New image comes without the prerelease tag. Expected scenario
	// is to get one update plan for the second deployment. Deployment with prerelease tag
	// should be ignored

	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}

	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-1",
				Namespace: "xxxx",
				Labels:    map[string]string{types.KeelPolicyLabel: "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1-staging",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-2",
				Namespace: "xxxx",
				Labels:    map[string]string{types.KeelPolicyLabel: "major"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	// creating "new version" event
	repo := &types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "1.1.2-staging",
	}

	plans, err := provider.createUpdatePlans(repo)
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected to find 1 deployment update plan but found %d", len(plans))
	}

	if plans[0].Resource.Identifier != "deployment/xxxx/dep-1" {
		t.Errorf("expected to get 'deployment/xxxx/dep-1', but got: %s", plans[0].Resource.Identifier)
	}
}

func TestProcessEvent(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "deployment-1",
				Namespace:   "ns-1",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta_v1.ObjectMeta{
						Annotations: map[string]string{
							"this": "that",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "deployment-2",
				Namespace:   "ns-2",
				Labels:      map[string]string{"whatever": "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta_v1.ObjectMeta{
						Annotations: map[string]string{
							"this": "that",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/bye-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "deployment-3",
				Namespace: "ns-3",
				Labels: map[string]string{
					"whatever": "all",
					"foo":      "bar",
				},
				Annotations: map[string]string{
					"ann": "1",
				},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta_v1.ObjectMeta{
						Annotations: map[string]string{
							"this": "that",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/bye-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)
	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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

	if fp.updated == nil {
		t.Fatalf("resource was not updated")
	}

	if fp.updated.Containers()[0].Image != repo.Name+":"+repo.Tag {
		t.Errorf("expected to find a deployment with updated image but found: %s", fp.updated.Containers()[0].Image)
	}
}

func TestProcessEventBuildNumber(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "deployment-1",
				Namespace:   "xxxx",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:10",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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

	if fp.updated != nil {
		t.Errorf("didn't expect to get updated containers, bot got: %s", fp.updated.Identifier)
	}
}

func TestEventSent(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "deployment-1",
				Namespace:   "xxxx",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:10.0.0",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	fs := &fakeSender{}
	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, fs, approver, grc)
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	repo := types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "11.0.0",
	}

	event := &types.Event{Repository: repo}
	_, err = provider.processEvent(event)
	if err != nil {
		t.Errorf("got error while processing event: %s", err)
	}

	if fp.updated.Containers()[0].Image != repo.Name+":"+repo.Tag {
		t.Errorf("expected to find a deployment with updated image but found: %s", fp.updated.Containers()[0].Image)
	}

	if fs.sentEvent.Message != "Successfully updated deployment xxxx/deployment-1 10.0.0->11.0.0 (gcr.io/v2-namespace/hello-world:11.0.0)" {
		t.Errorf("expected 'Successfully updated deployment xxxx/deployment-1 10.0.0->11.0.0 (gcr.io/v2-namespace/hello-world:11.0.0)' sent message, got: %s", fs.sentEvent.Message)
	}
}

func TestEventSentWithReleaseNotes(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "deployment-1",
				Namespace:   "xxxx",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{types.KeelReleaseNotesURL: "https://github.com/keel-hq/keel/releases"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:10.0.0",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	fs := &fakeSender{}
	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, fs, approver, grc)
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	repo := types.Repository{
		Name: "gcr.io/v2-namespace/hello-world",
		Tag:  "11.0.0",
	}

	event := &types.Event{Repository: repo}
	_, err = provider.processEvent(event)
	if err != nil {
		t.Errorf("got error while processing event: %s", err)
	}

	if fp.updated.Containers()[0].Image != repo.Name+":"+repo.Tag {
		t.Errorf("expected to find a deployment with updated image but found: %s", fp.updated.Containers()[0].Image)
	}

	if fs.sentEvent.Level != types.LevelSuccess {
		t.Errorf("expected level %s, got: %s", types.LevelSuccess, fs.sentEvent.Level)
	}

	if fs.sentEvent.Message != "Successfully updated deployment xxxx/deployment-1 10.0.0->11.0.0 (gcr.io/v2-namespace/hello-world:11.0.0). Release notes: https://github.com/keel-hq/keel/releases" {
		t.Errorf("expected 'Successfully updated deployment xxxx/deployment-1 10.0.0->11.0.0 (gcr.io/v2-namespace/hello-world:11.0.0). Release notes: https://github.com/keel-hq/keel/releases' sent message, got: %s", fs.sentEvent.Message)
	}
}

// Test to check how many deployments are "impacted" if we have sidecar container
func TestGetImpactedTwoContainersInSameDeployment(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-1",
				Namespace:   "xxxx",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
							{
								Image: "gcr.io/v2-namespace/greetings-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-2",
				Namespace: "xxxx",
				Labels:    map[string]string{"whatever": "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}
	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
	for _, c := range plans[0].Resource.Containers() {

		containerImageName := versionreg.ReplaceAllString(c.Image, "")

		if containerImageName == repo.Name {
			found = true
		}
	}

	if !found {
		t.Errorf("couldn't find expected deployment in impacted deployment list")
	}

}

// Test to check how many deployments are "impacted" if we have two init containers
func TestGetImpactedTwoInitContainersInSameDeployment(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-1",
				Namespace:   "xxxx",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{types.KeelInitContainerAnnotation: "true"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						InitContainers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
							{
								Image: "gcr.io/v2-namespace/greetings-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-2",
				Namespace: "xxxx",
				Labels:    map[string]string{"whatever": "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						InitContainers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}
	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
	for _, c := range plans[0].Resource.InitContainers() {

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
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-1",
				Namespace:   "xxxx",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-2",
				Namespace:   "xxxx",
				Labels:      map[string]string{"whatever": "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
	for _, c := range plans[0].Resource.Containers() {

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
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-1",
				Namespace:   "xxxx",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/foo-world",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-2",
				Namespace:   "xxxx",
				Annotations: map[string]string{},
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}
	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
	for _, c := range plans[0].Resource.Containers() {

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
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-1",
				Namespace:   "xxxx",
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "dep-2",
				Namespace:   "xxxx",
				Annotations: map[string]string{},
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}
	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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
		for _, c := range plan.Resource.Containers() {

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
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-1",
				Namespace: "xxxx",
				Labels:    map[string]string{types.KeelPolicyLabel: "all"},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1",
							},
						},
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "very-secret",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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

func TestTrackedImagesWithSecrets(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-1",
				Namespace: "xxxx",
				Labels: map[string]string{
					types.KeelPolicyLabel:               "all",
					types.KeelImagePullSecretAnnotation: "foo-bar",
				},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1",
							},
						},
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "very-secret",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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

	if imgs[0].Secrets[0] != "foo-bar" {
		t.Errorf("expected foo-bar, got: %s", imgs[0].Secrets[0])
	}
	if imgs[0].Secrets[1] != "very-secret" {
		t.Errorf("expected very-secret, got: %s", imgs[0].Secrets[1])
	}
}

func TestTrackedInitImagesWithSecrets(t *testing.T) {
	fp := &fakeImplementer{}
	fp.namespaces = &v1.NamespaceList{
		Items: []v1.Namespace{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "xxxx"},
				v1.NamespaceSpec{},
				v1.NamespaceStatus{},
			},
		},
	}
	deps := []*apps_v1.Deployment{
		{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-1",
				Namespace: "xxxx",
				Labels: map[string]string{
					types.KeelPolicyLabel:               "all",
					types.KeelImagePullSecretAnnotation: "foo-bar",
					types.KeelInitContainerAnnotation:   "true",
				},
			},
			apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: "very-secret",
							},
						},
						InitContainers: []v1.Container{
							{
								Image: "gcr.io/v2-namespace/hello-world:1.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		},
	}

	grs := MustParseGRS(deps)
	grc := &k8s.GenericResourceCache{}
	grc.Add(grs...)

	approver, teardown := approver()
	defer teardown()
	provider, err := NewProvider(fp, &fakeSender{}, approver, grc)
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

	if imgs[0].Secrets[0] != "foo-bar" {
		t.Errorf("expected foo-bar, got: %s", imgs[0].Secrets[0])
	}
	if imgs[0].Secrets[1] != "very-secret" {
		t.Errorf("expected very-secret, got: %s", imgs[0].Secrets[1])
	}
}
