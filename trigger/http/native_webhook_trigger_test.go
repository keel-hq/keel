package http

import (
	"bytes"
	"net/http"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"

	"net/http/httptest"
	"testing"
)

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

func TestNativeWebhookHandler(t *testing.T) {

	fp := &fakeProvider{}
	providers := map[string]provider.Provider{
		fp.GetName(): fp,
	}
	srv := NewTriggerServer(&Opts{Providers: providers})
	srv.registerRoutes(srv.router)

	req, err := http.NewRequest("POST", "/v1/native", bytes.NewBuffer([]byte(`{"repository": {"name": "gcr.io/v2-namespace/hello-world", "tag": "1.1.1"}}`)))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	//The response recorder used to record HTTP responses
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)
	}

	if len(fp.submitted) != 1 {
		t.Fatalf("unexpected number of events submitted: %d", len(fp.submitted))
	}

}

func TestNativeWebhookHandlerNoRepoName(t *testing.T) {

	fp := &fakeProvider{}
	providers := map[string]provider.Provider{
		fp.GetName(): fp,
	}
	srv := NewTriggerServer(&Opts{Providers: providers})
	srv.registerRoutes(srv.router)

	req, err := http.NewRequest("POST", "/v1/native", bytes.NewBuffer([]byte(`{"repository": { "tag": "1.1.1"}}`)))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	//The response recorder used to record HTTP responses
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("unexpected status code: %d", rec.Code)
	}

	if len(fp.submitted) != 0 {
		t.Fatalf("unexpected number of events submitted: %d", len(fp.submitted))
	}

}
