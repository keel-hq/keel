package poll

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"
	"github.com/keel-hq/keel/util/image"
)

func mustParse(img string, schedule string) *types.TrackedImage {
	ref, err := image.Parse(img)
	if err != nil {
		panic(err)
	}
	return &types.TrackedImage{
		Image:        ref,
		PollSchedule: schedule,
		Trigger:      types.TriggerTypePoll,
	}
}

// ======== fake registry client for testing =======
type fakeRegistryClient struct {
	opts registry.Opts // opts set if anything called Digest(opts Opts)

	digestToReturn string

	tagsToReturn []string
}

func (c *fakeRegistryClient) Get(opts registry.Opts) (*registry.Repository, error) {
	c.opts = opts
	return &registry.Repository{
		Name: opts.Name,
		Tags: c.tagsToReturn,
	}, nil
}

func (c *fakeRegistryClient) Digest(opts registry.Opts) (digest string, err error) {
	c.opts = opts
	return c.digestToReturn, nil
}

// ======== fake provider for testing =======
type fakeProvider struct {
	submitted []types.Event
	images    []*types.TrackedImage
}

func (p *fakeProvider) Submit(event types.Event) error {
	p.submitted = append(p.submitted, event)
	return nil
}

func (p *fakeProvider) GetName() string {
	return "fakeProvider"
}
func (p *fakeProvider) Stop() {
	return
}
func (p *fakeProvider) TrackedImages() ([]*types.TrackedImage, error) {
	return p.images, nil
}

func TestWatchTagJob(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
	}

	reference, _ := image.Parse("foo/bar:1.1")

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image: reference,
		},
		digest: "sha256:123123123",
	}

	job := NewWatchTagJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	submitted := fp.submitted[0]

	if submitted.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "1.1" {
		t.Errorf("unexpected event repository tag: %s", submitted.Repository.Tag)
	}

	if submitted.Repository.Digest != frc.digestToReturn {
		t.Errorf("unexpected event repository digest: %s", submitted.Repository.Digest)
	}

	// digest should be updated

	if job.details.digest != frc.digestToReturn {
		t.Errorf("job details digest wasn't updated")
	}
}

func TestWatchTagJobLatest(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
	}

	reference, _ := image.Parse("foo/bar:latest")

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image: reference,
		},
		digest: "sha256:123123123",
	}

	job := NewWatchTagJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	submitted := fp.submitted[0]

	if submitted.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "latest" {
		t.Errorf("unexpected event repository tag: %s", submitted.Repository.Tag)
	}

	if submitted.Repository.Digest != frc.digestToReturn {
		t.Errorf("unexpected event repository digest: %s", submitted.Repository.Digest)
	}

	// digest should be updated

	if job.details.digest != frc.digestToReturn {
		t.Errorf("job details digest wasn't updated")
	}
}

func TestWatchAllTagsJob(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.1.2", "1.1.3", "0.9.1"},
	}

	reference, _ := image.Parse("foo/bar:1.1.0")

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image: reference,
		},
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	submitted := fp.submitted[0]

	if submitted.Repository.Name != "index.docker.io/foo/bar" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "1.1.3" {
		t.Errorf("expected event repository tag 1.1.3, but got: %s", submitted.Repository.Tag)
	}
}

func TestWatchAllTagsJobCurrentLatest(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.1.2", "1.1.3", "0.9.1"},
	}

	reference, _ := image.Parse("foo/bar:latest")

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image: reference,
		},
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if len(fp.submitted) != 0 {
		t.Errorf("expected 0 submitted events but got something: %s", fp.submitted[0].Repository)
	}

}

func TestWatchMultipleTags(t *testing.T) {
	// fake provider listening for events
	imgA, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	imgB, _ := image.Parse("gcr.io/v2-namespace/greetings-world:1.1.1")
	imgC, _ := image.Parse("gcr.io/v2-namespace/greetings-world:alpha")
	imgD, _ := image.Parse("gcr.io/v2-namespace/greetings-world:master")
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

			&types.TrackedImage{
				Trigger:      types.TriggerTypePoll,
				Image:        imgC,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
			},

			&types.TrackedImage{
				Trigger:      types.TriggerTypePoll,
				Image:        imgD,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
			},
		},
	}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)

	// returning some sha
	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
		tagsToReturn:   []string{"5.0.0"},
	}

	watcher := NewRepositoryWatcher(providers, frc)

	tracked := []*types.TrackedImage{
		mustParse("gcr.io/v2-namespace/hello-world:1.1.1", "@every 10m"),
		mustParse("gcr.io/v2-namespace/greetings-world:1.1.1", "@every 10m"),
		mustParse("gcr.io/v2-namespace/greetings-world:alpha", "@every 10m"),
		mustParse("gcr.io/v2-namespace/greetings-world:master", "@every 10m"),
	}

	watcher.Watch(tracked...)

	if len(watcher.watched) != 4 {
		t.Errorf("expected to find watching 4 entries, found: %d", len(watcher.watched))
	}

	if dig, ok := watcher.watched["gcr.io/v2-namespace/greetings-world:alpha"]; ok != true {
		t.Errorf("alpha watcher not found")
		if dig.digest != "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb" {
			t.Errorf("digest not set for alpha")
		}
	}

	if dig, ok := watcher.watched["gcr.io/v2-namespace/greetings-world:master"]; ok != true {
		t.Errorf("alpha watcher not found")
		if dig.digest != "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb" {
			t.Errorf("digest not set for alpha")
		}
	}

	if det, ok := watcher.watched["gcr.io/v2-namespace/greetings-world"]; ok != true {
		t.Errorf("alpha watcher not found")
		if det.latest != "5.0.0" {
			t.Errorf("expected to find a tag set for multiple tags watch job")
		}
	}
}

type fakeCredentialsHelper struct {

	// set by the caller
	getImageRequest *types.TrackedImage

	// credentials to return
	creds *types.Credentials
}

func (fch *fakeCredentialsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	fch.getImageRequest = image
	return fch.creds, nil
}

func (fch *fakeCredentialsHelper) IsEnabled() bool { return true }

func TestWatchTagJobCheckCredentials(t *testing.T) {

	fakeHelper := &fakeCredentialsHelper{
		creds: &types.Credentials{
			Username: "user-xx",
			Password: "pass-xx",
		},
	}

	credentialshelper.RegisterCredentialsHelper("fake", fakeHelper)
	defer credentialshelper.UnregisterCredentialsHelper("fake")

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
	}

	reference, _ := image.Parse("foo/bar:1.1")

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image: reference,
		},
		digest: "sha256:123123123",
	}

	job := NewWatchTagJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if frc.opts.Password != "pass-xx" {
		t.Errorf("unexpected password for registry: %s", frc.opts.Password)
	}

	if frc.opts.Username != "user-xx" {
		t.Errorf("unexpected username for registry: %s", frc.opts.Username)
	}
}

func TestWatchTagJobLatestECR(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip()
	}

	imgA, _ := image.Parse("528670773427.dkr.ecr.us-east-2.amazonaws.com/webhook-demo:master")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			&types.TrackedImage{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
			},
		},
	}

	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	rc := registry.New()

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image: imgA,
		},
		digest: "sha256:123123123",
	}

	job := NewWatchTagJob(providers, rc, details)

	for i := 0; i < 5; i++ {
		job.Run()
	}

	// checking whether new job was submitted

	submitted := fp.submitted[0]

	if submitted.Repository.Name != "528670773427.dkr.ecr.us-east-2.amazonaws.com/webhook-demo" {
		t.Errorf("unexpected event repository name: %s", submitted.Repository.Name)
	}

	if submitted.Repository.Tag != "master" {
		t.Errorf("unexpected event repository tag: %s", submitted.Repository.Tag)
	}

	if submitted.Repository.Digest != "sha256:7712aa425c17c2e413e5f4d64e2761eda009509d05d0e45a26e389d715aebe23" {
		t.Errorf("unexpected event repository digest: %s", submitted.Repository.Digest)
	}

	// digest should be updated

	if job.details.digest != "sha256:7712aa425c17c2e413e5f4d64e2761eda009509d05d0e45a26e389d715aebe23" {
		t.Errorf("job details digest wasn't updated")
	}
}

func TestUnwatchAfterNotTrackedAnymore(t *testing.T) {
	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)

	// returning some sha
	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
		tagsToReturn:   []string{"5.0.0"},
	}

	watcher := NewRepositoryWatcher(providers, frc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher.Start(ctx)

	tracked := []*types.TrackedImage{
		mustParse("gcr.io/v2-namespace/hello-world:1.1.1", "@every 10m"),
		mustParse("gcr.io/v2-namespace/greetings-world:1.1.1", "@every 10m"),
		mustParse("gcr.io/v2-namespace/greetings-world:alpha", "@every 10m"),
		mustParse("gcr.io/v2-namespace/greetings-world:master", "@every 10m"),
	}

	watcher.Watch(tracked...)

	if len(watcher.watched) != 4 {
		t.Errorf("expected to find watching 4 entries, found: %d", len(watcher.watched))
	}

	if dig, ok := watcher.watched["gcr.io/v2-namespace/greetings-world:alpha"]; ok != true {
		t.Errorf("alpha watcher not found")
		if dig.digest != "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb" {
			t.Errorf("digest not set for alpha")
		}
	}

	if dig, ok := watcher.watched["gcr.io/v2-namespace/greetings-world:master"]; ok != true {
		t.Errorf("alpha watcher not found")
		if dig.digest != "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb" {
			t.Errorf("digest not set for alpha")
		}
	}

	if det, ok := watcher.watched["gcr.io/v2-namespace/greetings-world"]; ok != true {
		t.Errorf("alpha watcher not found")
		if det.latest != "5.0.0" {
			t.Errorf("expected to find a tag set for multiple tags watch job")
		}
	}

	trackedUpdated := []*types.TrackedImage{
		mustParse("gcr.io/v2-namespace/hello-world:1.1.1", "@every 10m"),
		mustParse("gcr.io/v2-namespace/greetings-world:1.1.1", "@every 10m"),
		mustParse("gcr.io/v2-namespace/greetings-world:alpha", "@every 10m"),
	}

	watcher.Watch(trackedUpdated...)

	if len(watcher.watched) != 3 {
		t.Errorf("expected to find watching 3 entries, found: %d", len(watcher.watched))
	}
}
