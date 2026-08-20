package poll

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/keel-hq/keel/approvals"
	// "github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	logrustest "github.com/sirupsen/logrus/hooks/test"
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
		Policy:       policy.LegacyPolicyPopulate(ref),
	}
}

// ======== fake registry client for testing =======
type fakeRegistryClient struct {
	opts        registry.Opts // opts set if anything called Digest(opts Opts)
	digestCalls int
	getCalls    int

	digestToReturn string

	digestErrToReturn error

	// digests returned by Digests(), defaults to digestToReturn
	tagDigestsToReturn    []string
	tagDigestsErrToReturn error
	tagDigestsCalls       int

	tagsToReturn []string

	platformsToReturn map[string][]types.Platform
	platformErrors    map[string]error
	platformErr       error
	platformCalls     []string
}

func (c *fakeRegistryClient) Get(opts registry.Opts) (*registry.Repository, error) {
	c.getCalls++
	c.opts = opts
	return &registry.Repository{
		Name: opts.Name,
		Tags: c.tagsToReturn,
	}, nil
}

func (c *fakeRegistryClient) Digest(opts registry.Opts) (digest string, err error) {
	c.digestCalls++
	c.opts = opts
	return c.digestToReturn, c.digestErrToReturn
}

func (c *fakeRegistryClient) Digests(opts registry.Opts) ([]string, error) {
	c.tagDigestsCalls++
	c.opts = opts
	if c.tagDigestsErrToReturn != nil {
		return nil, c.tagDigestsErrToReturn
	}
	if c.tagDigestsToReturn != nil {
		return c.tagDigestsToReturn, nil
	}
	return []string{c.digestToReturn}, nil
}

func (c *fakeRegistryClient) Platforms(opts registry.Opts) ([]types.Platform, error) {
	c.opts = opts
	c.platformCalls = append(c.platformCalls, opts.Tag)
	if err, ok := c.platformErrors[opts.Tag]; ok {
		return nil, err
	}
	if c.platformErr != nil {
		return nil, c.platformErr
	}
	if platforms, ok := c.platformsToReturn[opts.Tag]; ok {
		return platforms, nil
	}
	return []types.Platform{{OS: "linux", Architecture: "amd64"}}, nil
}

// ======== fake provider for testing =======
type fakeProvider struct {
	submitted []types.Event
	images    []*types.TrackedImage
}

type fakeProviders struct {
	provider *fakeProvider
}

func (p *fakeProviders) Submit(event types.Event) error {
	return p.provider.Submit(event)
}

func (p *fakeProviders) TrackedImages() ([]*types.TrackedImage, error) {
	return p.provider.TrackedImages()
}

func (p *fakeProviders) List() []string { return []string{p.provider.GetName()} }
func (p *fakeProviders) Stop()          {}

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
	for _, trackedImage := range p.images {
		if len(trackedImage.Platforms) == 0 && trackedImage.PlatformErr == types.PlatformErrorNone {
			trackedImage.Platforms = []types.Platform{{OS: "linux", Architecture: "amd64"}}
		}
	}
	return p.images, nil
}

func TestWatchTagJob(t *testing.T) {

	fp := &fakeProvider{}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

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

func TestWatchTagJobForce(t *testing.T) {

	img, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			{
				Image:        img,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				Policy:       policy.NewForcePolicy(true),
			},
		},
	}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
		tagsToReturn:   []string{"1.1.2", "1.2.0"},
	}

	watcher := NewRepositoryWatcher(providers, frc)

	err := watcher.Watch(fp.images...)

	if err != nil {
		t.Errorf("expected to find watching %s", img.Remote())
	}

	if dig, ok := watcher.watched["gcr.io/v2-namespace/hello-world:1.1.1"]; ok {
		if dig.latest != "1.1.1" {
			t.Errorf("unexpected event repository tag: %s", dig.latest)
		}
	} else {
		t.Errorf("hello-world:1.1.1 watcher not found")
	}
}

func TestWatchTagJobLatest(t *testing.T) {

	fp := &fakeProvider{}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

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

	reference, _ := image.Parse("foo/bar:1.1.0")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			{
				Image:   reference,
				Trigger: types.TriggerTypePoll,
				Policy:  policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true),
			},
		},
	}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.1.2", "1.1.3", "0.9.1"},
	}

	details := &watchDetails{
		trackedImage: fp.images[0],
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

	reference, _ := image.Parse("foo/bar:latest")
	fp := &fakeProvider{
		images: []*types.TrackedImage{
			{
				Image:   reference,
				Trigger: types.TriggerTypePoll,
				Policy:  policy.NewForcePolicy(true),
			},
		},
	}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		tagsToReturn: []string{"1.1.2", "1.1.3", "0.9.1"},
	}

	details := &watchDetails{
		trackedImage: fp.images[0],
	}

	job := NewWatchRepositoryTagsJob(providers, frc, details)

	job.Run()

	// checking whether new job was submitted

	if len(fp.submitted) != 0 {
		t.Errorf("expected 0 submitted events but got something: %v", fp.submitted[0].Repository)
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

			{
				Image:        imgA,
				Trigger:      types.TriggerTypePoll,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				Policy:       policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
			},

			{
				Trigger:      types.TriggerTypePoll,
				Image:        imgB,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				Policy:       policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
			},

			{
				Trigger:      types.TriggerTypePoll,
				Image:        imgC,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				Policy:       policy.NewForcePolicy(true),
			},

			{
				Trigger:      types.TriggerTypePoll,
				Image:        imgD,
				Provider:     "fp",
				PollSchedule: types.KeelPollDefaultSchedule,
				Policy:       policy.NewForcePolicy(true),
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
		t.Errorf("master watcher not found")
		if dig.digest != "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb" {
			t.Errorf("digest not set for master")
		}
	}

	if det, ok := watcher.watched["gcr.io/v2-namespace/greetings-world"]; ok != true {
		t.Errorf("watcher not found")
		if det.latest != "5.0.0" {
			t.Errorf("expected to find a tag set for multiple tags watch job")
		}
	}
}

func TestReportedLatestMajorTagSetSkipsArmCandidate(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()

	img, _ := image.Parse("jellyfin/jellyfin:latest")
	fp := &fakeProvider{images: []*types.TrackedImage{{
		Image:   img,
		Trigger: types.TriggerTypePoll,
		Policy:  policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
		Platforms: []types.Platform{{
			OS:           "linux",
			Architecture: "amd64",
		}},
	}}}
	providers := &fakeProviders{provider: fp}
	registryClient := &fakeRegistryClient{platformsToReturn: map[string][]types.Platform{
		"20240303.2-unstable-armhf": {{OS: "linux", Architecture: "arm", Variant: "v7"}},
		"10.10.7":                   {{OS: "linux", Architecture: "amd64"}},
	}}
	job := NewWatchRepositoryTagsJob(providers, registryClient, &watchDetails{trackedImage: fp.images[0]})

	events, err := job.computeEvents([]string{"latest", "10.10.7", "20240303.2-unstable-armhf"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Repository.Tag != "10.10.7" {
		t.Fatalf("expected compatible update after skipping ARM candidate, got %#v", events)
	}
	if !events[0].Repository.PlatformVerified || !types.PlatformsSupportAll(events[0].Repository.Platforms, fp.images[0].Platforms) {
		t.Fatalf("event did not carry verified platform evidence: %#v", events[0].Repository)
	}
	assertLogMessage(t, hook, "skipping candidate because it is incompatible with a related workload platform")
}

func TestCandidateSelectionAcceptsMultiArchManifest(t *testing.T) {
	img, _ := image.Parse("example/image:1.0.0")
	fp := &fakeProvider{images: []*types.TrackedImage{{
		Image:     img,
		Trigger:   types.TriggerTypePoll,
		Policy:    policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
		Platforms: []types.Platform{{OS: "linux", Architecture: "amd64"}},
	}}}
	providers := &fakeProviders{provider: fp}
	registryClient := &fakeRegistryClient{platformsToReturn: map[string][]types.Platform{
		"2.0.0": {
			{OS: "linux", Architecture: "arm64"},
			{OS: "linux", Architecture: "amd64"},
		},
	}}
	job := NewWatchRepositoryTagsJob(providers, registryClient, &watchDetails{trackedImage: fp.images[0]})

	events, err := job.computeEvents([]string{"2.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Repository.Tag != "2.0.0" {
		t.Fatalf("expected multi-architecture update, got %#v", events)
	}
}

func TestCandidateSelectionFailsClosedWhenPlatformResolutionFails(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()

	img, _ := image.Parse("example/image:1.0.0")
	fp := &fakeProvider{images: []*types.TrackedImage{{
		Image:   img,
		Trigger: types.TriggerTypePoll,
		Policy:  policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
	}}}
	providers := &fakeProviders{provider: fp}
	job := NewWatchRepositoryTagsJob(providers, &fakeRegistryClient{platformErr: errors.New("no manifest metadata")}, &watchDetails{trackedImage: fp.images[0]})

	events, err := job.computeEvents([]string{"2.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no unsafe update, got %#v", events)
	}
	assertLogMessage(t, hook, "skipping candidate because its platform could not be established")
}

func TestCandidateSelectionSupportsEveryRelatedWorkload(t *testing.T) {
	amd64Image, _ := image.Parse("example/image:1.0.0")
	arm64Image, _ := image.Parse("example/image:1.0.0")
	majorPolicy := policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true)
	fp := &fakeProvider{images: []*types.TrackedImage{
		{
			Image:     amd64Image,
			Trigger:   types.TriggerTypePoll,
			Policy:    majorPolicy,
			Platforms: []types.Platform{{OS: "linux", Architecture: "amd64"}},
		},
		{
			Image:     arm64Image,
			Trigger:   types.TriggerTypePoll,
			Policy:    majorPolicy,
			Platforms: []types.Platform{{OS: "linux", Architecture: "arm64"}},
		},
	}}
	providers := &fakeProviders{provider: fp}
	registryClient := &fakeRegistryClient{platformsToReturn: map[string][]types.Platform{
		"3.0.0": {{OS: "linux", Architecture: "amd64"}},
		"2.0.0": {
			{OS: "linux", Architecture: "amd64"},
			{OS: "linux", Architecture: "arm64"},
		},
	}}
	job := NewWatchRepositoryTagsJob(providers, registryClient, &watchDetails{trackedImage: fp.images[0]})

	events, err := job.computeEvents([]string{"2.0.0", "3.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Repository.Tag != "2.0.0" {
		t.Fatalf("expected multi-architecture candidate for related workloads, got %#v", events)
	}
}

func TestCandidateSelectionContinuesAfterPlatformResolutionFailure(t *testing.T) {
	img, _ := image.Parse("example/image:1.0.0")
	fp := &fakeProvider{images: []*types.TrackedImage{{
		Image:     img,
		Trigger:   types.TriggerTypePoll,
		Policy:    policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
		Platforms: []types.Platform{{OS: "linux", Architecture: "amd64"}},
	}}}
	providers := &fakeProviders{provider: fp}
	registryClient := &fakeRegistryClient{
		platformErrors: map[string]error{"3.0.0": errors.New("manifest metadata unavailable")},
		platformsToReturn: map[string][]types.Platform{
			"2.0.0": {{OS: "linux", Architecture: "amd64"}},
		},
	}
	job := NewWatchRepositoryTagsJob(providers, registryClient, &watchDetails{trackedImage: fp.images[0]})

	events, err := job.computeEvents([]string{"2.0.0", "3.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Repository.Tag != "2.0.0" {
		t.Fatalf("expected compatible fallback candidate, got %#v", events)
	}
}

func TestSupportsRelatedWorkloadsRequiresEveryEligiblePlatform(t *testing.T) {
	img, _ := image.Parse("example/image:1.0.0")
	tracked := []*types.TrackedImage{{
		Image:  img,
		Policy: policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
		Platforms: []types.Platform{
			{OS: "linux", Architecture: "amd64"},
			{OS: "linux", Architecture: "arm64"},
		},
	}}
	if supportsRelatedWorkloads([]types.Platform{{OS: "linux", Architecture: "amd64"}}, "2.0.0", tracked) {
		t.Fatal("single-platform candidate unexpectedly supports a mixed workload")
	}
	if !supportsRelatedWorkloads([]types.Platform{
		{OS: "linux", Architecture: "amd64"},
		{OS: "linux", Architecture: "arm64"},
	}, "2.0.0", tracked) {
		t.Fatal("complete multi-platform candidate was rejected")
	}
	tracked[0].Platforms = nil
	tracked[0].PlatformErr = types.PlatformErrorNodeMetadata
	if supportsRelatedWorkloads([]types.Platform{{OS: "linux", Architecture: "amd64"}}, "2.0.0", tracked) {
		t.Fatal("unresolved workload unexpectedly accepted a candidate")
	}
}

func TestCandidateSelectionFailsClosedForUnresolvedWorkload(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()

	img, _ := image.Parse("example/image:1.0.0")
	fp := &fakeProvider{images: []*types.TrackedImage{{
		Image:       img,
		Trigger:     types.TriggerTypePoll,
		Policy:      policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
		PlatformErr: types.PlatformErrorNodeMetadata,
	}}}
	providers := &fakeProviders{provider: fp}
	registryClient := &fakeRegistryClient{}
	job := NewWatchRepositoryTagsJob(providers, registryClient, &watchDetails{trackedImage: fp.images[0]})

	events, err := job.computeEvents([]string{"2.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 || len(registryClient.platformCalls) != 0 {
		t.Fatalf("unresolved workload evaluated a candidate: events=%#v calls=%v", events, registryClient.platformCalls)
	}
	assertLogMessage(t, hook, "skipping workload because its eligible platforms could not be established")
}

func assertLogMessage(t *testing.T, hook *logrustest.Hook, message string) {
	t.Helper()
	messages := make([]string, 0, len(hook.AllEntries()))
	for _, entry := range hook.AllEntries() {
		messages = append(messages, entry.Message)
		if strings.Contains(entry.Message, message) {
			return
		}
	}
	t.Fatalf("expected log message %q, got %q", message, messages)
}

type fakeCredentialsHelper struct {

	// set by the caller
	getImageRequest *types.TrackedImage

	// credentials to return
	creds *types.Credentials

	// error to return
	error error
}

func (fch *fakeCredentialsHelper) GetCredentials(image *types.TrackedImage) (*types.Credentials, error) {
	fch.getImageRequest = image
	return fch.creds, fch.error
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
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

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

func TestWatchWithAuthenticationError(t *testing.T) {

	fakeHelper := &fakeCredentialsHelper{
		creds: nil,
		error: errors.New("no credentials found"),
	}

	credentialshelper.RegisterCredentialsHelper("fake", fakeHelper)
	defer credentialshelper.UnregisterCredentialsHelper("fake")

	fp := &fakeProvider{}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		digestErrToReturn: errors.New("authentication failed"),
	}

	watcher := NewRepositoryWatcher(providers, frc)

	tracked := []*types.TrackedImage{
		mustParse("private.registry.com/v2-namespace/hello-world:1.1.1", "@every 10m"),
	}

	err := watcher.Watch(tracked...)

	if err == nil {
		t.Fatalf("expected error with faild authentication, but got nil")
	}
}

func TestWatchTagJobSupportsHarborProjectPath(t *testing.T) {
	fp := &fakeProvider{}
	store, teardown := newTestingUtils()
	defer teardown()
	providers := provider.New([]provider.Provider{fp}, approvals.New(&approvals.Opts{Store: store}))

	ref, err := image.Parse("harbor.mydomain.com/library/ai-rag:latest")
	if err != nil {
		t.Fatal(err)
	}
	registryClient := &fakeRegistryClient{digestToReturn: "sha256:new"}
	job := NewWatchTagJob(providers, registryClient, &watchDetails{
		trackedImage: &types.TrackedImage{
			Image:   ref,
			Trigger: types.TriggerTypePoll,
			Policy:  policy.NewForcePolicy(true),
		},
		digest: "sha256:old",
	})

	job.Run()

	if registryClient.opts.Registry != "https://harbor.mydomain.com" || registryClient.opts.Name != "library/ai-rag" || registryClient.opts.Tag != "latest" {
		t.Fatalf("unexpected Harbor registry options: %+v", registryClient.opts)
	}
	if len(fp.submitted) != 1 {
		t.Fatalf("expected one digest update, got %d", len(fp.submitted))
	}
	event := fp.submitted[0]
	if event.Repository.Name != "harbor.mydomain.com/library/ai-rag" || event.Repository.Tag != "latest" || event.Repository.Digest != "sha256:new" {
		t.Fatalf("unexpected Harbor update event: %+v", event.Repository)
	}
}

func TestWatchTagJobLatestECR(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip()
	}

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
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{
		Store: store,
	})

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

// TestWatchTagJobForceUpdateSameTag verifies the keel.sh/force-update escape
// hatch: a poll that sees an unchanged tag/digest must still trigger a single
// rollout when a force update has been requested, and must not keep re-firing
// on every poll (which would cause a continuous rolling restart).
func TestWatchTagJobForceUpdateSameTag(t *testing.T) {
	fp := &fakeProvider{}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{Store: store})
	providers := provider.New([]provider.Provider{fp}, am)

	const digest = "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb"
	reference, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	// the registry serves the exact same digest the watcher already has, so a
	// normal poll would be a no-op
	frc := &fakeRegistryClient{digestToReturn: digest}

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image:       reference,
			Trigger:     types.TriggerTypePoll,
			Policy:      policy.NewForcePolicy(true),
			ForceUpdate: true,
		},
		digest: digest,
	}

	job := NewWatchTagJob(providers, frc, details)

	// same digest, but a force update was requested -> one rollout event
	job.Run()
	if len(fp.submitted) != 1 {
		t.Fatalf("expected 1 event for a forced same-tag update, got %d", len(fp.submitted))
	}
	if got := fp.submitted[0].Repository.Tag; got != "1.1.1" {
		t.Errorf("unexpected tag: got %s, want 1.1.1", got)
	}

	// a second poll with the same digest and still-pending request must not
	// re-submit, so a force update is one-shot until it is cleared
	job.Run()
	if len(fp.submitted) != 1 {
		t.Fatalf("force update must be one-shot, got %d events", len(fp.submitted))
	}
}

// TestWatchTagJobNoChangeNoOp makes sure the normal poll path stays a no-op
// when neither the digest nor the tag have changed and no force update is
// requested (this is what guards against continuous restarts).
func TestWatchTagJobNoChangeNoOp(t *testing.T) {
	fp := &fakeProvider{}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{Store: store})
	providers := provider.New([]provider.Provider{fp}, am)

	const digest = "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb"
	reference, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	frc := &fakeRegistryClient{digestToReturn: digest}

	details := &watchDetails{
		trackedImage: &types.TrackedImage{
			Image:   reference,
			Trigger: types.TriggerTypePoll,
			Policy:  policy.NewForcePolicy(true),
		},
		digest: digest, // unchanged and no force-update requested
	}

	job := NewWatchTagJob(providers, frc, details)

	for i := 0; i < 3; i++ {
		job.Run()
	}
	if len(fp.submitted) != 0 {
		t.Fatalf("expected no event for an unchanged digest, got %d", len(fp.submitted))
	}
}

// TestWatchTagJobForceUpdateRearm verifies that once a force-update request is
// consumed, clearing the annotation (via the provider) re-arms the watcher so a
// subsequent request fires again, even when the digest never changed.
func TestWatchTagJobForceUpdateRearm(t *testing.T) {
	fp := &fakeProvider{}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{Store: store})
	providers := provider.New([]provider.Provider{fp}, am)

	const digest = "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb"
	reference, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	frc := &fakeRegistryClient{digestToReturn: digest}

	watcher := NewRepositoryWatcher(providers, frc)

	armed := &types.TrackedImage{
		Image:        reference,
		Trigger:      types.TriggerTypePoll,
		PollSchedule: types.KeelPollDefaultSchedule,
		Policy:       policy.NewForcePolicy(true),
		ForceUpdate:  true,
	}
	notForced := &types.TrackedImage{
		Image:        reference,
		Trigger:      types.TriggerTypePoll,
		PollSchedule: types.KeelPollDefaultSchedule,
		Policy:       policy.NewForcePolicy(true),
	}

	// the annotation is present at scan time: the job is created and run once,
	// so the force update fires immediately
	if err := watcher.Watch(armed); err != nil {
		t.Fatal(err)
	}
	if len(fp.submitted) != 1 {
		t.Fatalf("expected 1 event on first force update, got %d", len(fp.submitted))
	}

	// the provider cleared the annotation; the next scan sees ForceUpdate=false
	// and resets the consumed flag (Watch does not run the job itself)
	if err := watcher.Watch(notForced); err != nil {
		t.Fatal(err)
	}
	if len(fp.submitted) != 1 {
		t.Fatalf("expected no event on a non-forced scan, got %d", len(fp.submitted))
	}

	// the user sets the annotation again; the watcher is re-armed
	if err := watcher.Watch(armed); err != nil {
		t.Fatal(err)
	}

	details := watcher.watched["gcr.io/v2-namespace/hello-world:1.1.1"]
	if details == nil {
		t.Fatal("expected watcher entry to exist")
	}
	NewWatchTagJob(providers, frc, details).Run()

	if len(fp.submitted) != 2 {
		t.Fatalf("expected 2 events after re-arm, got %d", len(fp.submitted))
	}
}
