package approvals

import (
	"testing"
	"time"

	"github.com/rusenask/keel/cache/memory"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/codecs"
)

type fakeProvider struct {
	submitted []types.Event
	images    []*types.TrackedImage
}

func (p *fakeProvider) Submit(event types.Event) error {
	p.submitted = append(p.submitted, event)
	return nil
}

func (p *fakeProvider) TrackedImages() ([]*types.TrackedImage, error) {
	return p.images, nil
}
func (p *fakeProvider) List() []string {
	return []string{"fakeprovider"}
}
func (p *fakeProvider) Stop() {
	return
}
func (p *fakeProvider) GetName() string {
	return "fp"
}

func TestCreateApproval(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	fp := &fakeProvider{}

	am := New(mem, codecs.DefaultSerializer(), fp)

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       0,
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	stored, err := am.Get(types.ProviderTypeKubernetes, "xxx/app-1")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if stored.CurrentVersion != "1.2.3" {
		t.Errorf("unexpected version: %s", stored.CurrentVersion)
	}
}

func TestUpdateApproval(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	fp := &fakeProvider{}

	am := New(mem, codecs.DefaultSerializer(), fp)

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		VotesRequired:  1,
		VotesReceived:  0,
		Deadline:       0,
		Event: &types.Event{
			Repository: types.Repository{
				Name: "very/repo",
				Tag:  "1.2.5",
			},
		},
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	err = am.Update(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		VotesRequired:  1,
		VotesReceived:  1,
		Deadline:       0,
		Event: &types.Event{
			Repository: types.Repository{
				Name: "very/repo",
				Tag:  "1.2.5",
			},
		},
	})

	// checking provider
	if len(fp.submitted) != 1 {
		t.Fatalf("expected to find 1 submitted event")
	}

	if fp.submitted[0].Repository.Name != "very/repo" {
		t.Errorf("unexpected repo name in re-submitted event: %s", fp.submitted[0].Repository.Name)
	}
	if fp.submitted[0].Repository.Tag != "1.2.5" {
		t.Errorf("unexpected repo tag in re-submitted event: %s", fp.submitted[0].Repository.Tag)
	}
}

func TestUpdateApprovalRejected(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	fp := &fakeProvider{}

	am := New(mem, codecs.DefaultSerializer(), fp)

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		VotesRequired:  1,
		VotesReceived:  0,
		Deadline:       0,
		Event: &types.Event{
			Repository: types.Repository{
				Name: "very/repo",
				Tag:  "1.2.5",
			},
		},
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	// rejecting
	err = am.Update(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		VotesRequired:  1,
		VotesReceived:  0,
		Rejected:       true,
		Deadline:       0,
		Event: &types.Event{
			Repository: types.Repository{
				Name: "very/repo",
				Tag:  "1.2.5",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to update approval: %s", err)
	}

	// sending vote
	err = am.Update(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		VotesRequired:  1,
		VotesReceived:  1,
		Rejected:       true,
		Deadline:       0,
		Event: &types.Event{
			Repository: types.Repository{
				Name: "very/repo",
				Tag:  "1.2.5",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to update approval: %s", err)
	}

	// checking provider
	if len(fp.submitted) == 1 {
		t.Fatalf("expected to find 0 submitted event as it was rejected but found: %d", len(fp.submitted))
	}
}

func TestApprove(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	fp := &fakeProvider{}

	am := New(mem, codecs.DefaultSerializer(), fp)

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       0,
		VotesRequired:  2,
		VotesReceived:  0,
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	am.Approve(types.ProviderTypeKubernetes, "xxx/app-1")

	stored, err := am.Get(types.ProviderTypeKubernetes, "xxx/app-1")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if stored.VotesReceived != 1 {
		t.Errorf("unexpected number of received votes: %d", stored.VotesReceived)
	}
}

func TestReject(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	fp := &fakeProvider{}

	am := New(mem, codecs.DefaultSerializer(), fp)

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       0,
		VotesRequired:  2,
		VotesReceived:  0,
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	am.Reject(types.ProviderTypeKubernetes, "xxx/app-1")

	stored, err := am.Get(types.ProviderTypeKubernetes, "xxx/app-1")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if !stored.Rejected {
		t.Errorf("unexpected approval to be rejected")
	}
}
