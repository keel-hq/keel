package poll

import (
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

func trackedImageForTrigger(t *testing.T, name string, trigger types.TriggerType, imagePolicy policy.Policy) *types.TrackedImage {
	t.Helper()
	ref, err := image.Parse(name)
	if err != nil {
		t.Fatalf("parse image %q: %v", name, err)
	}
	return &types.TrackedImage{
		Image:        ref,
		PollSchedule: "@every 10m",
		Trigger:      trigger,
		Provider:     "fakeProvider",
		Policy:       imagePolicy,
	}
}

func TestRepositoryWatcherDoesNotRegisterWebhookOnlyImage(t *testing.T) {
	regexpPolicy, err := policy.NewRegexpPolicy(`regexp:^main_[a-fA-F0-9]{40}$`)
	if err != nil {
		t.Fatal(err)
	}
	webhookImage := trackedImageForTrigger(t,
		"registry.example.com/team/app:main_0000000000000000000000000000000000000000",
		types.TriggerTypeDefault,
		regexpPolicy,
	)
	fp := &fakeProvider{images: []*types.TrackedImage{webhookImage}}
	store, teardown := newTestingUtils()
	defer teardown()
	providers := provider.New([]provider.Provider{fp}, approvals.New(&approvals.Opts{Store: store}))
	registryClient := &fakeRegistryClient{digestToReturn: "sha256:current"}
	watcher := NewRepositoryWatcher(providers, registryClient)

	for range 2 { // initial scan plus a resource resync/restart scan
		if err := watcher.Watch(fp.images...); err != nil {
			t.Fatalf("Watch() error = %v", err)
		}
	}

	if got := len(watcher.watched); got != 0 {
		t.Fatalf("registered %d polling watcher(s) for webhook-only image", got)
	}
	if registryClient.digestCalls != 0 || registryClient.getCalls != 0 {
		t.Fatalf("webhook-only image executed registry polling: digest=%d get=%d", registryClient.digestCalls, registryClient.getCalls)
	}
}

func TestRepositoryWatcherSharedRepositoryUsesOnlyPollingConsumers(t *testing.T) {
	webhookPolicy, err := policy.NewRegexpPolicy(`regexp:^main_f{40}$`)
	if err != nil {
		t.Fatal(err)
	}
	pollPolicy, err := policy.NewRegexpPolicy(`regexp:^main_1{40}$`)
	if err != nil {
		t.Fatal(err)
	}
	webhookImage := trackedImageForTrigger(t,
		"registry.example.com/team/app:main_0000000000000000000000000000000000000000",
		types.TriggerTypeDefault,
		webhookPolicy,
	)
	pollImage := trackedImageForTrigger(t,
		"registry.example.com/team/app:main_0000000000000000000000000000000000000000",
		types.TriggerTypePoll,
		pollPolicy,
	)
	fp := &fakeProvider{images: []*types.TrackedImage{webhookImage, pollImage}}
	store, teardown := newTestingUtils()
	defer teardown()
	providers := provider.New([]provider.Provider{fp}, approvals.New(&approvals.Opts{Store: store}))
	registryClient := &fakeRegistryClient{
		digestToReturn: "sha256:current",
		tagsToReturn: []string{
			"main_1111111111111111111111111111111111111111",
			"main_ffffffffffffffffffffffffffffffffffffffff",
		},
	}
	watcher := NewRepositoryWatcher(providers, registryClient)

	if err := watcher.Watch(webhookImage, pollImage); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	if got := len(watcher.watched); got != 1 {
		t.Fatalf("registered %d polling watchers, want 1", got)
	}
	if got := len(fp.submitted); got != 1 {
		t.Fatalf("submitted %d poll events, want 1", got)
	}
	if got, want := fp.submitted[0].Repository.Tag, "main_1111111111111111111111111111111111111111"; got != want {
		t.Fatalf("poll event tag = %q, want polling consumer tag %q", got, want)
	}
}
