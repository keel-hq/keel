package poll

import (
	"testing"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"
	"github.com/keel-hq/keel/util/image"
)

// ======== fake registry client for testing =======
type fakeRegistryClient struct {
	opts registry.Opts // opts set if anything called Digest(opts Opts)

	digestToReturn string

	tagsToReturn []string
}

func (c *fakeRegistryClient) Get(opts registry.Opts) (*registry.Repository, error) {
	return &registry.Repository{
		Name: opts.Name,
		Tags: c.tagsToReturn,
	}, nil
}

func (c *fakeRegistryClient) Digest(opts registry.Opts) (digest string, err error) {
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
		imageRef: reference,
		digest:   "sha256:123123123",
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
		imageRef: reference,
		digest:   "sha256:123123123",
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
		imageRef: reference,
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
		imageRef: reference,
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if len(fp.submitted) != 0 {
		t.Errorf("expected 0 submitted events but got something: %s", fp.submitted[0].Repository)
	}

}
