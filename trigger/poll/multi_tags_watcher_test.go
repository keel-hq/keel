package poll

import (
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

func TestWatchMultipleTagsWithSemver(t *testing.T) {
	// fake provider listening for events
	imgA, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			&types.TrackedImage{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				SemverPreReleaseTags: map[string]string{
					"dev":  "1.0.0-dev",
					"prod": "1.5.0-prod",
				},
			},
		},
	}
	mem := memory.NewMemoryCache()
	am := approvals.New(mem)
	providers := provider.New([]provider.Provider{fp}, am)

	// returning some sha
	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
		tagsToReturn:   []string{"5.0.0"},
	}

	watcher := NewRepositoryWatcher(providers, frc)

	tracked := []*types.TrackedImage{
		mustParse("gcr.io/v2-namespace/hello-world:1.1.1", "@every 10m"),
	}

	err := watcher.Watch(tracked...)
	if err != nil {
		t.Errorf("failed to watch: %s", err)
	}

	if len(watcher.watched) != 1 {
		t.Errorf("expected to find watching 4 entries, found: %d", len(watcher.watched))
	}
	if det, ok := watcher.watched["gcr.io/v2-namespace/hello-world"]; ok != true {
		t.Errorf("alpha watcher not found")
		if det.latest != "1.5.0" {
			t.Errorf("expected to find a tag set for multiple tags watch job")
		}
	}
}

func TestWatchAllTagsJobWithSemver(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache()
	am := approvals.New(mem)
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.3.0-dev", "1.5.0", "1.8.0-alpha"},
	}

	reference, _ := image.Parse("foo/bar:1.1.0")

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image: reference,
			SemverPreReleaseTags: map[string]string{
				"dev": "1.2.0-dev",
			},
		},
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if len(fp.submitted) != 2 {
		t.Errorf("expected 2 events, got: %d", len(fp.submitted))
	}

	submitted := fp.submitted[0]

	if submitted.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "1.8.0-alpha" {
		t.Errorf("expected event repository tag 1.8.0-alpha, but got: %s", submitted.Repository.Tag)
	}

	submitted2 := fp.submitted[1]

	if submitted2.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted2.Repository.Tag != "1.3.0-dev" {
		t.Errorf("expected event repository tag 1.3.0-dev, but got: %s", submitted.Repository.Tag)
	}
}
