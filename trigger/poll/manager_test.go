package poll

import (
	"context"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"

	"testing"
)

func TestCheckDeployment(t *testing.T) {
	// fake provider listening for events
	imgA, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	imgB, _ := image.Parse("gcr.io/v2-namespace/greetings-world:1.1.1")
	fp := &fakeProvider{
		images: []*types.TrackedImage{

			&types.TrackedImage{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
			},

			&types.TrackedImage{
				Trigger:      types.TriggerTypePoll,
				Image:        imgB,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
			},
		},
	}
	providers := provider.New([]provider.Provider{fp})

	// returning some sha
	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
	}

	watcher := NewRepositoryWatcher(providers, frc)

	pm := NewPollManager(providers, watcher)

	imageA := "gcr.io/v2-namespace/hello-world:1.1.1"
	imageB := "gcr.io/v2-namespace/greetings-world:1.1.1"

	pm.scan(context.Background())

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
