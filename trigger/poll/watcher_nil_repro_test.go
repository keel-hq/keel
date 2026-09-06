package poll

import (
	"testing"

	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

// fakeRegistryClientNoop avoids any registry network calls for reproduction.
type nilReproRegistryClient struct {
	registry.Client
}

func (c *nilReproRegistryClient) Digest(opts registry.Opts) (string, error) {
	return "sha256:repro", nil
}
func (c *nilReproRegistryClient) Digests(opts registry.Opts) ([]string, error) {
	return []string{"sha256:repro"}, nil
}
func (c *nilReproRegistryClient) Get(opts registry.Opts) (*registry.Repository, error) {
	return &registry.Repository{Name: opts.Name}, nil
}
func (c *nilReproRegistryClient) Platforms(opts registry.Opts) ([]types.Platform, error) {
	return nil, nil
}

// newNilReproWatcher builds a RepositoryWatcher wired with a non-piercing
// registry client so addJob reaches the image-reference handling without
// touching the network.
func newNilReproWatcher() *RepositoryWatcher {
	w := NewRepositoryWatcher(nil, &nilReproRegistryClient{})
	return w
}

// TestAddJobNilImage panics (before the fix) when a tracked image carries a
// nil *image.Reference, mimicking a malformed/helm-provider image shape.
func TestAddJobNilImage(t *testing.T) {
	w := newNilReproWatcher()
	ti := &types.TrackedImage{
		Image:        nil,
		PollSchedule: types.KeelPollDefaultSchedule,
		Trigger:      types.TriggerTypePoll,
		Policy:       policy.NewForcePolicy(true),
	}
	err := w.addJob(ti, types.KeelPollDefaultSchedule, nil)
	if err == nil {
		t.Fatalf("expected an error for a nil image, got none")
	}
	if _, ok := w.watched["repro"]; ok {
		t.Errorf("nil image should not create a watcher entry")
	}
}

// TestAddJobUnparsedReference reproduces a reference whose underlying named is
// nil (the shape produced by a helm chart image that fails to resolve), which
// panics in Registry()/ShortName() before the fix.
func TestAddJobUnparsedReference(t *testing.T) {
	w := newNilReproWatcher()
	ti := &types.TrackedImage{
		Image:        &image.Reference{},
		PollSchedule: types.KeelPollDefaultSchedule,
		Trigger:      types.TriggerTypePoll,
		Policy:       policy.NewForcePolicy(true),
	}
	_, err := w.watch(ti, nil)
	if err == nil {
		t.Fatalf("expected an error for an unparsable image reference, got none")
	}
}

// TestWatchSkipsBadReleaseButSurvives ensures one malformed release cannot
// take down the whole poll loop: a bad image is skipped while a valid one
// still gets watched.
func TestWatchSkipsBadReleaseButSurvives(t *testing.T) {
	w := newNilReproWatcher()
	good, err := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	bad := &types.TrackedImage{
		Image:        nil,
		PollSchedule: types.KeelPollDefaultSchedule,
		Trigger:      types.TriggerTypePoll,
		Policy:       policy.NewForcePolicy(true),
	}
	goodTI := &types.TrackedImage{
		Image:        good,
		PollSchedule: types.KeelPollDefaultSchedule,
		Trigger:      types.TriggerTypePoll,
		Policy:       policy.NewForcePolicy(true),
	}

	w.Watch(bad, goodTI)

	if _, ok := w.watched["gcr.io/v2-namespace/hello-world:1.1.1"]; !ok {
		t.Fatalf("valid image should still be watched despite the bad release")
	}
}
