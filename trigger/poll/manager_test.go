package poll

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/pkg/store/sql"

	// "github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	// "github.com/keel-hq/keel/extension/credentialshelper"
	_ "github.com/keel-hq/keel/extension/credentialshelper/aws"

	"testing"
)

type FakeSecretsGetter struct {
}

func (g *FakeSecretsGetter) Get(image *types.TrackedImage) (*types.Credentials, error) {
	return &types.Credentials{}, nil
}

func newTestingUtils() (*sql.SQLStore, func()) {
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

func TestCheckDeployment(t *testing.T) {
	// fake provider listening for events
	imgA, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	imgB, _ := image.Parse("gcr.io/v2-namespace/greetings-world:1.1.1")
	fp := &fakeProvider{
		images: []*types.TrackedImage{

			{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
			},

			{
				Trigger:      types.TriggerTypePoll,
				Image:        imgB,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
			},
		},
	}

	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	providers := provider.New([]provider.Provider{fp}, am)

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
	keyA := getImageIdentifier(ref, false)
	if watcher.watched[keyA].digest != frc.digestToReturn {
		t.Errorf("unexpected digest")
	}
	if watcher.watched[keyA].schedule != types.KeelPollDefaultSchedule {
		t.Errorf("unexpected schedule: %s", watcher.watched[keyA].schedule)
	}
	if watcher.watched[keyA].trackedImage.Image.Remote() != ref.Remote() {
		t.Errorf("unexpected remote remote: %s", watcher.watched[keyA].trackedImage.Image.Remote())
	}
	if watcher.watched[keyA].trackedImage.Image.Tag() != ref.Tag() {
		t.Errorf("unexpected tag: %s", watcher.watched[keyA].trackedImage.Image.Tag())
	}

	refB, _ := image.Parse(imageB)
	keyB := getImageIdentifier(refB, false)
	if watcher.watched[keyB].digest != frc.digestToReturn {
		t.Errorf("unexpected digest")
	}
	if watcher.watched[keyB].schedule != types.KeelPollDefaultSchedule {
		t.Errorf("unexpected schedule: %s", watcher.watched[keyB].schedule)
	}
	if watcher.watched[keyB].trackedImage.Image.Remote() != refB.Remote() {
		t.Errorf("unexpected remote remote: %s", watcher.watched[keyB].trackedImage.Image.Remote())
	}
	if watcher.watched[keyB].trackedImage.Image.Tag() != refB.Tag() {
		t.Errorf("unexpected tag: %s", watcher.watched[keyB].trackedImage.Image.Tag())
	}
}

// To run this test, set AWS env variables
// export AWS_ACCESS_KEY_ID=AKIA.........
// export AWS_ACCESS_KEY=3v..............
// export AWS_REGION=us-east-2
func TestCheckECRDeployment(t *testing.T) {

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip()
	}

	// fake provider listening for events
	imgA, _ := image.Parse("528670773427.dkr.ecr.us-east-2.amazonaws.com/webhook-demo:master")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
			},
		},
	}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	providers := provider.New([]provider.Provider{fp}, am)
	rc := registry.New()

	watcher := NewRepositoryWatcher(providers, rc)

	pm := NewPollManager(providers, watcher)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pm.scan(ctx)

	// 2 subscriptions should be added
	entries := watcher.cron.Entries()
	if len(entries) != 1 {
		t.Fatalf("unexpected list of cron entries: %d", len(entries))
	}

	keyA := getImageIdentifier(imgA, false)

	if len(watcher.watched) != 1 {
		t.Fatalf("expected to find 1 entry in watcher.watched map, found: %d", len(watcher.watched))
	}

	if watcher.watched[keyA].digest != "sha256:7712aa425c17c2e413e5f4d64e2761eda009509d05d0e45a26e389d715aebe23" {
		t.Errorf("unexpected digest: %s", watcher.watched[keyA].digest)
	}
	if watcher.watched[keyA].schedule != types.KeelPollDefaultSchedule {
		t.Errorf("unexpected schedule: %s", watcher.watched[keyA].schedule)
	}
	if watcher.watched[keyA].trackedImage.Image.Remote() != imgA.Remote() {
		t.Errorf("unexpected remote remote: %s", watcher.watched[keyA].trackedImage.Image.Remote())
	}
	if watcher.watched[keyA].trackedImage.Image.Tag() != imgA.Tag() {
		t.Errorf("unexpected tag: %s", watcher.watched[keyA].trackedImage.Image.Tag())
	}
}
