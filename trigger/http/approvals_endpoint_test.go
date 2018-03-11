package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"
)

func TestListApprovals(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
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

func TestDeleteApproval(t *testing.T) {
	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers, ApprovalManager: am})
	srv.registerRoutes(srv.router)

	err := am.Create(&types.Approval{
		Identifier:     "dev/whd-dev:0.0.15",
		VotesRequired:  5,
		NewVersion:     "2.0.0",
		CurrentVersion: "1.0.0",
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	// listing
	req, err := http.NewRequest("POST", "/v1/approvals", bytes.NewBufferString(`{"action": "delete","identifier": "dev/whd-dev:0.0.15"}`))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	_, err = am.Get("dev/whd-dev:0.0.15")
	if err == nil {
		t.Errorf("expected approval to be deleted")
	}
}

func TestApprove(t *testing.T) {
	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers, ApprovalManager: am})
	srv.registerRoutes(srv.router)

	err := am.Create(&types.Approval{
		Identifier:     "dev/whd-dev:0.0.15",
		VotesRequired:  5,
		NewVersion:     "2.0.0",
		CurrentVersion: "1.0.0",
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	// listing
	req, err := http.NewRequest("POST", "/v1/approvals", bytes.NewBufferString(`{"voter": "foo","identifier": "dev/whd-dev:0.0.15"}`))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	approved, err := am.Get("dev/whd-dev:0.0.15")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if approved.VotesReceived != 1 {
		t.Errorf("expected to find one voter")
	}

	if approved.Voters[0] != "foo" {
		t.Errorf("unexpected voter: %s", approved.Voters[0])
	}
}

func TestApproveNotFound(t *testing.T) {
	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers, ApprovalManager: am})
	srv.registerRoutes(srv.router)

	// listing
	req, err := http.NewRequest("POST", "/v1/approvals", bytes.NewBufferString(`{"voter": "foo","identifier": "dev/whd-dev:0.0.15"}`))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}
}

func TestApproveGarbageRequest(t *testing.T) {
	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers, ApprovalManager: am})
	srv.registerRoutes(srv.router)

	// listing
	req, err := http.NewRequest("POST", "/v1/approvals", bytes.NewBufferString(`<>`))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}
}

func TestSameVoter(t *testing.T) {
	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers, ApprovalManager: am})
	srv.registerRoutes(srv.router)

	err := am.Create(&types.Approval{
		Identifier:     "dev/12345",
		VotesRequired:  5,
		NewVersion:     "2.0.0",
		CurrentVersion: "1.0.0",
		VotesReceived:  1,
		Voters:         []string{"foo"},
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	// listing
	req, err := http.NewRequest("POST", "/v1/approvals", bytes.NewBufferString(`{"voter": "foo", "identifier": "dev/12345"}`))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	approved, err := am.Get("dev/12345")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if approved.VotesReceived != 1 {
		t.Errorf("expected to find one voter")
	}

	if approved.Voters[0] != "foo" {
		t.Errorf("unexpected voter: %s", approved.Voters[0])
	}
}
func TestDifferentVoter(t *testing.T) {
	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers, ApprovalManager: am})
	srv.registerRoutes(srv.router)

	err := am.Create(&types.Approval{
		Identifier:     "dev/12345",
		VotesRequired:  5,
		NewVersion:     "2.0.0",
		CurrentVersion: "1.0.0",
		VotesReceived:  1,
		Voters:         []string{"bar"},
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	// listing
	req, err := http.NewRequest("POST", "/v1/approvals", bytes.NewBufferString(`{"voter": "foo", "identifier": "dev/12345"}`))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	approved, err := am.Get("dev/12345")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if approved.VotesReceived != 2 {
		t.Errorf("expected to find 2 voters")
	}

	if approved.Voters[0] != "bar" {
		t.Errorf("unexpected voter: %s", approved.Voters[0])
	}
	if approved.Voters[1] != "foo" {
		t.Errorf("unexpected voter: %s", approved.Voters[0])
	}
}

func TestReject(t *testing.T) {
	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Second, 100*time.Second, 10*time.Second)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers, ApprovalManager: am})
	srv.registerRoutes(srv.router)

	err := am.Create(&types.Approval{
		Identifier:     "dev/12345",
		VotesRequired:  5,
		NewVersion:     "2.0.0",
		CurrentVersion: "1.0.0",
	})

	if err != nil {
		t.Fatalf("failed to create approval: %s", err)
	}

	// listing
	req, err := http.NewRequest("POST", "/v1/approvals", bytes.NewBufferString(`{"voter": "foo", "action": "reject", "identifier":"dev/12345"}`))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	approved, err := am.Get("dev/12345")
	if err != nil {
		t.Fatalf("failed to get approval: %s", err)
	}

	if approved.Rejected != true {
		t.Errorf("expected to find approval rejected")
	}

}
