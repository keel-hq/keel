package http

import (
	"bytes"
	"net/http"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"

	"net/http/httptest"
	"testing"
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
func TestNativeWebhookHandler(t *testing.T) {

	fp := &fakeProvider{}

	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)

	srv := NewTriggerServer(&Opts{Providers: providers})
	srv.registerRoutes(srv.router)

	req, err := http.NewRequest("POST", "/v1/webhooks/native", bytes.NewBuffer([]byte(`{"name": "gcr.io/v2-namespace/hello-world", "tag": "1.1.1"}`)))
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
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers})
	srv.registerRoutes(srv.router)

	req, err := http.NewRequest("POST", "/v1/webhooks/native", bytes.NewBuffer([]byte(`{ "tag": "1.1.1"}`)))
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
