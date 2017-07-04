package poll

import (
	"testing"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/registry"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"
)

// ======== fake registry client for testing =======
type fakeRegistryClient struct {
	opts registry.Opts // opts set if anything called Digest(opts Opts)

	digestToReturn string
}

func (c *fakeRegistryClient) Get(opts registry.Opts) (*registry.Repository, error) {
	return nil, nil
}

func (c *fakeRegistryClient) Digest(opts registry.Opts) (digest string, err error) {
	return c.digestToReturn, nil
}

// ======== fake provider for testing =======
type fakeProvider struct {
	submitted []types.Event
}

func (p *fakeProvider) Submit(event types.Event) error {
	p.submitted = append(p.submitted, event)
	return nil
}

func (p *fakeProvider) GetName() string {
	return "fakeProvider"
}

func TestWatchTagJob(t *testing.T) {

	fp := &fakeProvider{}
	providers := provider.New([]provider.Provider{fp})

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

	if submitted.Repository.Name != "index.docker.io/foo/bar:1.1" {
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
	providers := provider.New([]provider.Provider{fp})

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

	if submitted.Repository.Name != "index.docker.io/foo/bar:latest" {
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
