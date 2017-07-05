package poll

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"
	keelTesting "github.com/rusenask/keel/util/testing"

	"testing"
)

func TestCheckDeployment(t *testing.T) {
	// fake provider listening for events
	fp := &fakeProvider{}
	providers := provider.New([]provider.Provider{fp})
	// implementer should not be called in this case
	k8sImplementer := &keelTesting.FakeK8sImplementer{}

	// returning some sha
	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
	}

	watcher := NewRepositoryWatcher(providers, frc)

	pm := NewPollManager(k8sImplementer, watcher)

	imageA := "gcr.io/v2-namespace/hello-world:1.1.1"
	imageB := "gcr.io/v2-namespace/greetings-world:1.1.1"

	dep := &v1beta1.Deployment{
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
							Image: imageA,
						},
						v1.Container{
							Image: imageB,
						},
					},
				},
			},
		},
		v1beta1.DeploymentStatus{},
	}

	err := pm.checkDeployment(dep)
	if err != nil {
		t.Errorf("deployment check failed: %s", err)
	}

	// 2 subscriptions should be added
	entries := watcher.cron.Entries()
	if len(entries) != 2 {
		t.Errorf("unexpected list of cron entries: %d", len(entries))
	}

	ref, _ := image.Parse(imageA)
	keyA := getImageIdentifier(ref)
	if watcher.watched[keyA].digest != frc.digestToReturn {
		t.Errorf("unexpected digest")
	}
	if watcher.watched[keyA].schedule != types.KeelPollDefaultSchedule {
		t.Errorf("unexpected schedule: %s", watcher.watched[keyA].schedule)
	}
	if watcher.watched[keyA].imageRef.Remote() != ref.Remote() {
		t.Errorf("unexpected remote remote: %s", watcher.watched[keyA].imageRef.Remote())
	}
	if watcher.watched[keyA].imageRef.Tag() != ref.Tag() {
		t.Errorf("unexpected tag: %s", watcher.watched[keyA].imageRef.Tag())
	}

	refB, _ := image.Parse(imageB)
	keyB := getImageIdentifier(refB)
	if watcher.watched[keyB].digest != frc.digestToReturn {
		t.Errorf("unexpected digest")
	}
	if watcher.watched[keyB].schedule != types.KeelPollDefaultSchedule {
		t.Errorf("unexpected schedule: %s", watcher.watched[keyB].schedule)
	}
	if watcher.watched[keyB].imageRef.Remote() != refB.Remote() {
		t.Errorf("unexpected remote remote: %s", watcher.watched[keyB].imageRef.Remote())
	}
	if watcher.watched[keyB].imageRef.Tag() != refB.Tag() {
		t.Errorf("unexpected tag: %s", watcher.watched[keyB].imageRef.Tag())
	}
}
