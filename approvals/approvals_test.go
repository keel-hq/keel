package approvals

import (
	"context"
	"testing"
	"time"

	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"
)

func TestCreateApproval(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       time.Now().Add(5 * time.Minute),
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	stored, err := am.Get("xxx/app-1")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if stored.CurrentVersion != "1.2.3" {
		t.Errorf("unexpected version: %s", stored.CurrentVersion)
	}
}

func TestDeleteApproval(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       time.Now().Add(5 * time.Minute),
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	err = am.Delete("xxx/app-1")
	if err != nil {
		t.Errorf("failed to delete approval: %s", err)
	}

	_, err = am.Get("xxx/app-1")
	if err == nil {
		t.Errorf("expected to get an error when retrieving deleted approval")
	}

}

func TestUpdateApproval(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		VotesRequired:  1,
		VotesReceived:  0,
		Deadline:       time.Now().Add(5 * time.Minute),
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := am.SubscribeApproved(ctx)
	if err != nil {
		t.Fatalf("failed to subscribe: %s", err)
	}

	err = am.Update(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		VotesRequired:  1,
		VotesReceived:  1,
		Deadline:       time.Now().Add(5 * time.Minute),
		Event: &types.Event{
			Repository: types.Repository{
				Name: "very/repo",
				Tag:  "1.2.5",
			},
		},
	})

	approved := <-ch

	if approved.Event.Repository.Name != "very/repo" {
		t.Errorf("unexpected repo name in re-submitted event: %s", approved.Event.Repository.Name)
	}
	if approved.Event.Repository.Tag != "1.2.5" {
		t.Errorf("unexpected repo tag in re-submitted event: %s", approved.Event.Repository.Tag)
	}
}

func TestUpdateApprovalRejected(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		VotesRequired:  1,
		VotesReceived:  0,
		Deadline:       time.Now().Add(5 * time.Minute),
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := am.SubscribeApproved(ctx)
	if err != nil {
		t.Fatalf("failed to subscribe: %s", err)
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
		Deadline:       time.Now().Add(5 * time.Minute),
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
		Deadline:       time.Now().Add(5 * time.Minute),
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

	select {
	case <-time.After(500 * time.Millisecond):
		// success
		return
	case approval := <-ch:
		t.Errorf("unexpected approval got: %s", approval.Identifier)
	}

}

func TestApprove(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1:1.2.5",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       time.Now().Add(5 * time.Minute),
		VotesRequired:  2,
		VotesReceived:  0,
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	am.Approve("xxx/app-1:1.2.5", "warda")

	stored, err := am.Get("xxx/app-1:1.2.5")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if stored.VotesReceived != 1 {
		t.Errorf("unexpected number of received votes: %d", stored.VotesReceived)
	}
}

func TestApproveTwiceSameVoter(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1:1.2.5",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       time.Now().Add(5 * time.Minute),
		VotesRequired:  2,
		VotesReceived:  0,
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	am.Approve("xxx/app-1:1.2.5", "warda")
	am.Approve("xxx/app-1:1.2.5", "warda")

	stored, err := am.Get("xxx/app-1:1.2.5")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	// should still be the same
	if stored.VotesReceived != 1 {
		t.Errorf("unexpected number of received votes: %d", stored.VotesReceived)
	}
}

func TestApproveTwoVoters(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1:1.2.5",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       time.Now().Add(5 * time.Minute),
		VotesRequired:  2,
		VotesReceived:  0,
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	am.Approve("xxx/app-1:1.2.5", "w")
	am.Approve("xxx/app-1:1.2.5", "k")

	stored, err := am.Get("xxx/app-1:1.2.5")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	// should still be the same
	if stored.VotesReceived != 2 {
		t.Errorf("unexpected number of received votes: %d", stored.VotesReceived)
	}
}

func TestReject(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       time.Now().Add(5 * time.Minute),
		VotesRequired:  2,
		VotesReceived:  0,
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	am.Reject("xxx/app-1")

	stored, err := am.Get("xxx/app-1")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if !stored.Rejected {
		t.Errorf("unexpected approval to be rejected")
	}
}

func TestExpire(t *testing.T) {
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)

	am := New(mem, codecs.DefaultSerializer())

	err := am.Create(&types.Approval{
		Provider:       types.ProviderTypeKubernetes,
		Identifier:     "xxx/app-1",
		CurrentVersion: "1.2.3",
		NewVersion:     "1.2.5",
		Deadline:       time.Now().Add(-5 * time.Minute),
		VotesRequired:  2,
		VotesReceived:  0,
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	err = am.expireEntries()
	if err != nil {
		t.Errorf("got error while expiring entries: %s", err)
	}

	_, err = am.Get("xxx/app-1")
	if err == nil {
		t.Errorf("expected approval to be deleted but didn't get an error")
	}
}
