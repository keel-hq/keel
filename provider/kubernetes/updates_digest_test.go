package kubernetes

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/types"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWebhookAndPollingDigestUpdates(t *testing.T) {
	fp := &fakeImplementer{podList: latestToLatestPods("gcr.io/v2-namespace/hello-world@" + oldImageDigest)}
	dep := latestToLatestDeployment(map[string]string{types.KeelTriggerLabel: "poll"})
	cache := &k8s.GenericResourceCache{}
	cache.Add(MustParseGRS([]*apps_v1.Deployment{dep})...)
	approver, teardown := approver()
	defer teardown()
	p, err := NewProvider(fp, &fakeSender{}, approver, cache)
	if err != nil {
		t.Fatal(err)
	}
	event := &types.Event{TriggerName: "dockerhub", Repository: types.Repository{
		Name: "gcr.io/v2-namespace/hello-world", Tag: "latest", Digest: newImageDigest,
	}}
	updated, err := p.processEvent(event)
	if err != nil || len(updated) != 1 {
		t.Fatalf("webhook: updated=%d, err=%v", len(updated), err)
	}
	// Observe the successful update as the workload informer does. The old
	// pods deliberately remain, representing an unfinished rollout.
	cache.Add(updated...)
	for _, trigger := range []string{"dockerhub", "poll"} {
		event.TriggerName = trigger
		updated, err = p.processEvent(event)
		if err != nil || len(updated) != 0 {
			t.Fatalf("duplicate %s: updated=%d, err=%v", trigger, len(updated), err)
		}
	}
	event.Repository.Digest = "sha256:" + strings.Repeat("c", 64)
	updated, err = p.processEvent(event)
	if err != nil || len(updated) != 1 {
		t.Fatalf("new digest: updated=%d, err=%v", len(updated), err)
	}
	cache.Add(updated...)
	event.TriggerName = "dockerhub"
	event.Repository.Digest = ""
	updated, err = p.processEvent(event)
	if err != nil || len(updated) != 1 {
		t.Fatalf("digestless webhook: updated=%d, err=%v", len(updated), err)
	}
	if len(deployedDigests(updated[0])) != 0 {
		t.Fatal("digestless webhook retained stale deployed digests")
	}
}

func TestDeployedDigestsAreScopedToImageAndTag(t *testing.T) {
	for _, tc := range []struct {
		name, recordedImage, currentTag string
		wantUpdate                      bool
	}{
		{"same image and tag", "index.docker.io/myuser/app:latest", "latest", false},
		{"another repository", "index.docker.io/other/app:latest", "latest", true},
		{"another tag", "index.docker.io/myuser/app:stable", "latest", true},
		{"tag change with identical digest", "index.docker.io/myuser/app:latest", "stable", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(map[string]string{tc.recordedImage: newImageDigest})
			if err != nil {
				t.Fatal(err)
			}
			dep := latestToLatestDeployment(map[string]string{types.KeelDigestsAnnotation: string(encoded)})
			dep.Spec.Template.Spec.Containers[0].Image = "myuser/app:" + tc.currentTag
			cache := &k8s.GenericResourceCache{}
			cache.Add(MustParseGRS([]*apps_v1.Deployment{dep})...)
			approver, teardown := approver()
			defer teardown()
			p, err := NewProvider(&fakeImplementer{}, &fakeSender{}, approver, cache)
			if err != nil {
				t.Fatal(err)
			}
			plans, err := p.createUpdatePlans(&types.Repository{Name: "docker.io/myuser/app", Tag: "latest", Digest: newImageDigest})
			if err != nil {
				t.Fatal(err)
			}
			if (len(plans) != 0) != tc.wantUpdate {
				t.Fatalf("plans=%d, want update=%t", len(plans), tc.wantUpdate)
			}
		})
	}
}

func TestDuplicateDigestDoesNotRestartTrackedImageSources(t *testing.T) {
	for _, source := range []string{"container", "init container", "image volume"} {
		t.Run(source, func(t *testing.T) {
			dep := latestToLatestDeployment(map[string]string{
				types.KeelInitContainerAnnotation: "true",
				types.KeelImageVolumeAnnotation:   "true",
				types.KeelDigestsAnnotation:       `{"index.docker.io/myuser/app:latest":"` + newImageDigest + `"}`,
			})
			spec := &dep.Spec.Template.Spec
			container := v1.Container{Name: "target", Image: "myuser/app:latest"}
			switch source {
			case "container":
				spec.Containers = []v1.Container{container}
			case "init container":
				spec.InitContainers = []v1.Container{container}
			case "image volume":
				spec.Volumes = []v1.Volume{{Name: "target", VolumeSource: v1.VolumeSource{
					Image: &v1.ImageVolumeSource{Reference: "myuser/app:latest"},
				}}}
			}
			resource := MustParseGR(dep)
			_, updated, err := checkForUpdate(policy.NewForcePolicy(true), &types.Repository{
				Name: "myuser/app", Tag: "latest", Digest: newImageDigest,
			}, resource)
			if err != nil || updated {
				t.Fatalf("updated=%t, err=%v", updated, err)
			}
			if resource.GetSpecAnnotations()[types.KeelUpdateTimeAnnotation] != "" {
				t.Fatal("duplicate digest changed the rollout timestamp")
			}
		})
	}
}

func TestRecordDeployedDigestRetainsOtherImagesAndPrunesOldTags(t *testing.T) {
	dep := latestToLatestDeployment(map[string]string{types.KeelDigestsAnnotation: `{"gcr.io/v2-namespace/hello-world:old":"old","index.docker.io/other/app:latest":"other"}`})
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers,
		v1.Container{Name: "other", Image: "other/app:latest"})
	resource := MustParseGR(dep)
	recordDeployedDigest(resource, &UpdatePlan{Image: "gcr.io/v2-namespace/hello-world:latest", NewDigest: newImageDigest})
	digests := deployedDigests(resource)
	if len(digests) != 2 || digests["gcr.io/v2-namespace/hello-world:latest"] != newImageDigest ||
		digests["index.docker.io/other/app:latest"] != "other" {
		t.Fatalf("unexpected deployed digests: %v", digests)
	}
}

// digests used by the latest -> latest notification tests
var (
	oldImageDigest = "sha256:" + strings.Repeat("a", 64)
	newImageDigest = "sha256:" + strings.Repeat("b", 64)
)

func latestToLatestDeployment(annotations map[string]string) *apps_v1.Deployment {
	return &apps_v1.Deployment{
		meta_v1.TypeMeta{},
		meta_v1.ObjectMeta{
			Name:        "deployment-1",
			Namespace:   "xxxx",
			Labels:      map[string]string{types.KeelPolicyLabel: "force"},
			Annotations: annotations,
		},
		apps_v1.DeploymentSpec{
			Selector: &meta_v1.LabelSelector{MatchLabels: map[string]string{"app": "hello-world"}},
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{Labels: map[string]string{"app": "hello-world"}},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{Name: "app", Image: "gcr.io/v2-namespace/hello-world:latest"},
					},
				},
			},
		},
		apps_v1.DeploymentStatus{},
	}
}

func latestToLatestPods(imageID string) *v1.PodList {
	return &v1.PodList{
		Items: []v1.Pod{
			{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{Name: "deployment-1-pod", Namespace: "xxxx"},
				v1.PodSpec{
					Containers: []v1.Container{
						{Name: "app", Image: "gcr.io/v2-namespace/hello-world:latest"},
					},
				},
				v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:    "app",
							ImageID: imageID,
							State:   v1.ContainerState{Running: &v1.ContainerStateRunning{}},
						},
					},
				},
			},
		},
	}
}

func TestLatestToLatestUpdateNotificationShowsImageDigests(t *testing.T) {
	fp := &fakeImplementer{
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
		// workloads still report the digest of the previous latest push
		podList: latestToLatestPods("docker-pullable://gcr.io/v2-namespace/hello-world@" + oldImageDigest),
	}

	deps := []*apps_v1.Deployment{latestToLatestDeployment(map[string]string{})}
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

	event := &types.Event{
		Repository: types.Repository{
			Name:   "gcr.io/v2-namespace/hello-world",
			Tag:    "latest",
			Digest: newImageDigest,
		},
	}
	_, err = provider.processEvent(event)
	if err != nil {
		t.Fatalf("got error while processing event: %s", err)
	}

	expectedMessage := fmt.Sprintf(
		"Successfully updated deployment xxxx/deployment-1 latest (%s)->latest (%s) (gcr.io/v2-namespace/hello-world:latest)",
		oldImageDigest, newImageDigest,
	)
	if fs.sentEvent.Message != expectedMessage {
		t.Errorf("expected message %q, got: %s", expectedMessage, fs.sentEvent.Message)
	}
	if fs.sentEvent.Level != types.LevelSuccess {
		t.Errorf("expected level %s, got: %s", types.LevelSuccess, fs.sentEvent.Level)
	}
	if got := fs.sentEvent.Metadata["newDigest"]; got != newImageDigest {
		t.Errorf("expected metadata newDigest %s, got: %s", newImageDigest, got)
	}
	if got := fs.sentEvent.Metadata["previousDigest"]; got != oldImageDigest {
		t.Errorf("expected metadata previousDigest %s, got: %s", oldImageDigest, got)
	}

	// the digest keel deployed is recorded so the next update can report it
	if got := fp.updated.GetAnnotations()[types.KeelDigestAnnotation]; got != newImageDigest {
		t.Errorf("expected %s annotation %s, got: %s", types.KeelDigestAnnotation, newImageDigest, got)
	}
	changeCause := fp.updated.GetAnnotations()["kubernetes.io/change-cause"]
	if !strings.Contains(changeCause, fmt.Sprintf("latest (%s) -> latest (%s)", oldImageDigest, newImageDigest)) {
		t.Errorf("expected change-cause to contain the digest transition, got: %s", changeCause)
	}
}

func TestLatestToLatestUpdateNotificationFallsBackToDigestAnnotation(t *testing.T) {
	fp := &fakeImplementer{
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
		// no pods to observe, the previous digest comes from the annotation
	}

	deps := []*apps_v1.Deployment{
		latestToLatestDeployment(map[string]string{types.KeelDigestAnnotation: oldImageDigest}),
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

	event := &types.Event{
		Repository: types.Repository{
			Name:   "gcr.io/v2-namespace/hello-world",
			Tag:    "latest",
			Digest: newImageDigest,
		},
	}
	_, err = provider.processEvent(event)
	if err != nil {
		t.Fatalf("got error while processing event: %s", err)
	}

	expectedMessage := fmt.Sprintf(
		"Successfully updated deployment xxxx/deployment-1 latest (%s)->latest (%s) (gcr.io/v2-namespace/hello-world:latest)",
		oldImageDigest, newImageDigest,
	)
	if fs.sentEvent.Message != expectedMessage {
		t.Errorf("expected message %q, got: %s", expectedMessage, fs.sentEvent.Message)
	}
	if got := fp.updated.GetAnnotations()[types.KeelDigestAnnotation]; got != newImageDigest {
		t.Errorf("expected %s annotation %s, got: %s", types.KeelDigestAnnotation, newImageDigest, got)
	}
}

func TestCurrentDigestPrefersRunningDigestOverAnnotation(t *testing.T) {
	fp := &fakeImplementer{
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
		podList: latestToLatestPods("gcr.io/v2-namespace/hello-world@" + oldImageDigest),
	}

	staleAnnotationDigest := "sha256:" + strings.Repeat("c", 64)
	deps := []*apps_v1.Deployment{
		latestToLatestDeployment(map[string]string{types.KeelDigestAnnotation: staleAnnotationDigest}),
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

	repo := &types.Repository{
		Name:   "gcr.io/v2-namespace/hello-world",
		Tag:    "latest",
		Digest: newImageDigest,
	}
	plans, err := provider.createUpdatePlans(repo)
	if err != nil {
		t.Fatalf("got error while creating update plans: %s", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 update plan, got %d", len(plans))
	}
	if plans[0].CurrentDigest != oldImageDigest {
		t.Errorf("expected current digest from running pods %s, got: %s", oldImageDigest, plans[0].CurrentDigest)
	}
	if plans[0].NewDigest != newImageDigest {
		t.Errorf("expected new digest %s, got: %s", newImageDigest, plans[0].NewDigest)
	}
}

func TestLatestToLatestUpdateNotificationWithoutKnownDigests(t *testing.T) {
	fp := &fakeImplementer{
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

	deps := []*apps_v1.Deployment{latestToLatestDeployment(map[string]string{})}
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

	// triggers such as some webhooks do not provide the image digest
	event := &types.Event{
		Repository: types.Repository{
			Name: "gcr.io/v2-namespace/hello-world",
			Tag:  "latest",
		},
	}
	_, err = provider.processEvent(event)
	if err != nil {
		t.Fatalf("got error while processing event: %s", err)
	}

	// without any digest information the message keeps its traditional shape
	expectedMessage := "Successfully updated deployment xxxx/deployment-1 latest->latest (gcr.io/v2-namespace/hello-world:latest)"
	if fs.sentEvent.Message != expectedMessage {
		t.Errorf("expected message %q, got: %s", expectedMessage, fs.sentEvent.Message)
	}
	if fp.updated.GetAnnotations()[types.KeelDigestAnnotation] != "" {
		t.Errorf("expected no %s annotation to be recorded without a digest", types.KeelDigestAnnotation)
	}
}
