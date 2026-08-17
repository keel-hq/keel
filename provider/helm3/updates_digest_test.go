package helm3

import (
	"fmt"
	"strings"
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"

	core_v1 "k8s.io/api/core/v1"

	"helm.sh/helm/v3/pkg/release"
)

// TestReleaseUpdateNotificationShowsImageDigests covers the latest -> latest
// transition for a force policy: nothing changes in the tag, so the image
// digests are what make the update identifiable in the notification.
func TestReleaseUpdateNotificationShowsImageDigests(t *testing.T) {
	oldDigest := "sha256:" + strings.Repeat("a", 64)
	newDigest := "sha256:" + strings.Repeat("b", 64)

	chartVals := `
name: al Rashid
where:
  city: Basrah
image:
  repository: karolisr/webhook-demo
  tag: latest

keel:
  policy: force
  trigger: poll
  images:
    - repository: image.repository
      tag: image.tag
`
	myChart, err := testingStringToChart(chartVals)
	if err != nil {
		t.Errorf("chartutil.ReadValues error = %v", err)
	}

	fakeImpl := &fakeImplementer{
		listReleasesResponse: []*release.Release{
			{
				Name:      "release-1",
				Namespace: "default",
				Chart:     myChart,
				Config:    make(map[string]interface{}),
			},
		},
	}

	approver, teardown := approver()
	defer teardown()

	// the release workload is still running the previous latest build
	resources := &platformResourceCache{resources: []*k8s.GenericResource{
		helmWorkloadWithSelector(t, "release-1", "owned", "karolisr/webhook-demo:latest"),
	}}
	pods := &helmPodSource{pods: map[string]*core_v1.PodList{
		"app=owned": helmRunningPods("karolisr/webhook-demo:latest", "karolisr/webhook-demo@"+oldDigest),
	}}

	fs := &fakeSender{}
	provider := NewProvider(fakeImpl, fs, approver,
		WithWorkloadPlatforms(nil, resources),
		WithRunningDigests(k8s.NewRunningDigestResolver(pods)),
	)

	err = provider.processEvent(&types.Event{
		Repository: types.Repository{
			Name:   "karolisr/webhook-demo",
			Tag:    "latest",
			Digest: newDigest,
		},
	})
	if err != nil {
		t.Fatalf("failed to process event, error: %s", err)
	}

	if fakeImpl.updatedRlsName != "release-1" {
		t.Fatalf("unexpected release updated: %s", fakeImpl.updatedRlsName)
	}

	expectedMessage := fmt.Sprintf(
		"Successfully updated release default/release-1 latest (%s)->latest (%s) (image.tag=latest)",
		oldDigest, newDigest,
	)
	if fs.sentEvent.Message != expectedMessage {
		t.Errorf("expected message %q, got: %s", expectedMessage, fs.sentEvent.Message)
	}
	if fs.sentEvent.Level != types.LevelSuccess {
		t.Errorf("expected level %s, got: %s", types.LevelSuccess, fs.sentEvent.Level)
	}
	if got := fs.sentEvent.Metadata["newDigest"]; got != newDigest {
		t.Errorf("expected metadata newDigest %s, got: %s", newDigest, got)
	}
	if got := fs.sentEvent.Metadata["previousDigest"]; got != oldDigest {
		t.Errorf("expected metadata previousDigest %s, got: %s", oldDigest, got)
	}
}

// TestReleaseUpdateNotificationWithoutKnownDigests keeps the traditional
// message shape when neither the event nor the runtime provide digests.
func TestReleaseUpdateNotificationWithoutKnownDigests(t *testing.T) {
	chartVals := `
name: al Rashid
where:
  city: Basrah
image:
  repository: karolisr/webhook-demo
  tag: latest

keel:
  policy: force
  trigger: poll
  images:
    - repository: image.repository
      tag: image.tag
`
	myChart, err := testingStringToChart(chartVals)
	if err != nil {
		t.Errorf("chartutil.ReadValues error = %v", err)
	}

	fakeImpl := &fakeImplementer{
		listReleasesResponse: []*release.Release{
			{
				Name:      "release-1",
				Namespace: "default",
				Chart:     myChart,
				Config:    make(map[string]interface{}),
			},
		},
	}

	approver, teardown := approver()
	defer teardown()

	fs := &fakeSender{}
	provider := NewProvider(fakeImpl, fs, approver)

	err = provider.processEvent(&types.Event{
		Repository: types.Repository{
			Name: "karolisr/webhook-demo",
			Tag:  "latest",
		},
	})
	if err != nil {
		t.Fatalf("failed to process event, error: %s", err)
	}

	expectedMessage := "Successfully updated release default/release-1 latest->latest (image.tag=latest)"
	if fs.sentEvent.Message != expectedMessage {
		t.Errorf("expected message %q, got: %s", expectedMessage, fs.sentEvent.Message)
	}
}
