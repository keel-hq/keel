package poll

import (
	"errors"
	"strings"
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

func digestOf(char string) string {
	return "sha256:" + strings.Repeat(char, 64)
}

// https://github.com/keel-hq/keel/issues/845 - the watcher baseline is
// in-memory only, so on every start it has to be seeded from what the workload
// is running instead of from what the registry currently serves.
func TestWatcherSeedsBaselineFromRunningDigest(t *testing.T) {
	registryDigest := digestOf("a")
	runningDigest := digestOf("b")

	tests := []struct {
		name           string
		runningDigests []string
		tagDigests     []string
		tagDigestsErr  error
		keepTag        bool
		expectEvent    bool
	}{
		{
			name:           "running digest is stale",
			runningDigests: []string{runningDigest},
			expectEvent:    true,
		},
		{
			name:           "running digest is current",
			runningDigests: []string{registryDigest},
			expectEvent:    false,
		},
		{
			name:           "rollout in progress, some replicas already updated",
			runningDigests: []string{registryDigest, runningDigest},
			expectEvent:    false,
		},
		{
			name:           "multi arch child manifest is not drift",
			runningDigests: []string{runningDigest},
			tagDigests:     []string{registryDigest, runningDigest},
			expectEvent:    false,
		},
		{
			name:           "unresolvable tag manifest keeps the registry baseline",
			runningDigests: []string{runningDigest},
			tagDigestsErr:  errors.New("manifest request returned 401 Unauthorized"),
			expectEvent:    false,
		},
		{
			name:           "no runtime information available",
			runningDigests: nil,
			expectEvent:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			img, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
			tracked := &types.TrackedImage{
				Image:          img,
				Trigger:        types.TriggerTypePoll,
				Provider:       "fp",
				PollSchedule:   types.KeelPollDefaultSchedule,
				Policy:         policy.NewForcePolicy(true),
				RunningDigests: test.runningDigests,
			}

			fp := &fakeProvider{images: []*types.TrackedImage{tracked}}
			store, teardown := newTestingUtils()
			defer teardown()
			am := approvals.New(&approvals.Opts{Store: store})
			providers := provider.New([]provider.Provider{fp}, am)

			frc := &fakeRegistryClient{
				digestToReturn:        registryDigest,
				tagDigestsToReturn:    test.tagDigests,
				tagDigestsErrToReturn: test.tagDigestsErr,
			}

			watcher := NewRepositoryWatcher(providers, frc)
			if err := watcher.Watch(tracked); err != nil {
				t.Fatalf("failed to watch image: %s", err)
			}

			if !test.expectEvent {
				if len(fp.submitted) != 0 {
					t.Fatalf("expected no event, got %v", fp.submitted)
				}
				return
			}

			if len(fp.submitted) != 1 {
				t.Fatalf("expected a single event, got %v", fp.submitted)
			}
			submitted := fp.submitted[0]
			if submitted.Repository.Digest != registryDigest {
				t.Errorf("unexpected event digest: %s", submitted.Repository.Digest)
			}
			if submitted.Repository.Tag != "1.1.1" {
				t.Errorf("unexpected event tag: %s", submitted.Repository.Tag)
			}

			// once the event is out, the baseline has caught up and a second
			// poll must stay quiet
			watcher.watched["gcr.io/v2-namespace/hello-world:1.1.1"].trackedImage = tracked
			NewWatchTagJob(providers, frc, watcher.watched["gcr.io/v2-namespace/hello-world:1.1.1"]).Run()
			if len(fp.submitted) != 1 {
				t.Fatalf("expected no repeated event, got %v", fp.submitted)
			}
		})
	}
}

// two workloads using the same image share a single watcher, so the baseline
// has to account for the stale one even when it is not the first seen
func TestWatcherSeedsSharedBaselineFromAnyStaleWorkload(t *testing.T) {
	registryDigest := digestOf("a")
	runningDigest := digestOf("b")

	img, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	current := &types.TrackedImage{
		Image:          img,
		Trigger:        types.TriggerTypePoll,
		Provider:       "fp",
		Namespace:      "current",
		PollSchedule:   types.KeelPollDefaultSchedule,
		Policy:         policy.NewForcePolicy(true),
		RunningDigests: []string{registryDigest},
	}
	stale := &types.TrackedImage{
		Image:          img,
		Trigger:        types.TriggerTypePoll,
		Provider:       "fp",
		Namespace:      "stale",
		PollSchedule:   types.KeelPollDefaultSchedule,
		Policy:         policy.NewForcePolicy(true),
		RunningDigests: []string{runningDigest},
	}

	fp := &fakeProvider{images: []*types.TrackedImage{current, stale}}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{Store: store})
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{digestToReturn: registryDigest}

	watcher := NewRepositoryWatcher(providers, frc)
	if err := watcher.Watch(current, stale); err != nil {
		t.Fatalf("failed to watch images: %s", err)
	}

	if len(fp.submitted) != 1 {
		t.Fatalf("expected a single event, got %v", fp.submitted)
	}
	if fp.submitted[0].Repository.Digest != registryDigest {
		t.Errorf("unexpected event digest: %s", fp.submitted[0].Repository.Digest)
	}
}

// semver style watchers do not compare digests, so they must not pay for the
// extra manifest lookup
func TestWatcherSkipsDriftCheckWithoutMatchTag(t *testing.T) {
	img, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.1")
	tracked := &types.TrackedImage{
		Image:          img,
		Trigger:        types.TriggerTypePoll,
		Provider:       "fp",
		PollSchedule:   types.KeelPollDefaultSchedule,
		Policy:         policy.NewForcePolicy(false),
		RunningDigests: []string{digestOf("b")},
	}

	fp := &fakeProvider{images: []*types.TrackedImage{tracked}}
	store, teardown := newTestingUtils()
	defer teardown()
	am := approvals.New(&approvals.Opts{Store: store})
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{digestToReturn: digestOf("a")}

	watcher := NewRepositoryWatcher(providers, frc)
	if err := watcher.Watch(tracked); err != nil {
		t.Fatalf("failed to watch image: %s", err)
	}

	if frc.tagDigestsCalls != 0 {
		t.Errorf("expected no manifest digest lookups, got %d", frc.tagDigestsCalls)
	}
}
