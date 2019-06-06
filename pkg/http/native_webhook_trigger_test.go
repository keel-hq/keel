package http

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"

	"github.com/keel-hq/keel/pkg/auth"
	"github.com/keel-hq/keel/pkg/store/sql"

	"net/http/httptest"
	"testing"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func NewTestingUtils() (*sql.SQLStore, func()) {
	dir, err := ioutil.TempDir("", "whstoretest")
	if err != nil {
		log.Fatal(err)
	}
	tmpfn := filepath.Join(dir, "gorm.db")
	// defer
	store, err := sql.New(sql.Opts{DatabaseType: "sqlite3", URI: tmpfn})
	if err != nil {
		log.Fatal(err)
	}

	teardown := func() {
		os.RemoveAll(dir) // clean up
	}

	return store, teardown
}

func NewTestingServer(fp provider.Provider) (*TriggerServer, func()) {
	// fp := &fakeProvider{}
	store, teardown := NewTestingUtils()
	// defer teardown()

	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	authenticator := auth.New(&auth.Opts{
		Username: "user-1",
		Password: "secret",
	})

	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{
		Providers:       providers,
		ApprovalManager: am,
		Authenticator:   authenticator,
		Store:           store,
	})
	srv.registerRoutes(srv.router)

	return srv, teardown
}

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
	srv, teardown := NewTestingServer(fp)
	defer teardown()

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
	srv, teardown := NewTestingServer(fp)
	defer teardown()

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
