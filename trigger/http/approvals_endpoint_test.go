package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rusenask/keel/approvals"
	"github.com/rusenask/keel/cache/memory"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/codecs"
)

func TestListApprovals(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers, ApprovalManager: am})
	srv.registerRoutes(srv.router)

	err := am.Create(&types.Approval{
		Identifier:     "123",
		VotesRequired:  5,
		NewVersion:     "2.0.0",
		CurrentVersion: "1.0.0",
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	// listing
	req, err := http.NewRequest("GET", "/v1/approvals", nil)
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	var approvals []*types.Approval

	err = json.Unmarshal(rec.Body.Bytes(), &approvals)
	if err != nil {
		t.Fatalf("failed to unmarshal response into approvals: %s", err)
	}

	if len(approvals) != 1 {
		t.Fatalf("expected to find 1 approval but found: %d", len(approvals))
	}

	if approvals[0].VotesRequired != 5 {
		t.Errorf("unexpected votes required")
	}
	if approvals[0].NewVersion != "2.0.0" {
		t.Errorf("unexpected new version: %s", approvals[0].NewVersion)
	}
	if approvals[0].CurrentVersion != "1.0.0" {
		t.Errorf("unexpected current version: %s", approvals[0].CurrentVersion)
	}
}
