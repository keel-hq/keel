package http

import (
	"bytes"
	"net/http"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/util/codecs"

	"net/http/httptest"
	"testing"
)

var fakeAzureWebhook = `{
  "id": "cb8c3971-9adc-488b-bdd8-43cbb4974ff5",
  "timestamp": "2017-11-17T16:52:01.343145347Z",
  "action": "push",
  "target": {
    "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
    "size": 524,
    "digest": "sha256:80f0d5c8786bb9e621a45ece0db56d11cdc624ad20da9fe62e9d25490f331d7d",
    "length": 524,
    "repository": "hello-world",
    "tag": "v1"
  },
  "request": {
    "id": "3cbb6949-7549-4fa1-86cd-a6d5451dffc7",
    "host": "myregistry.azurecr.io",
    "method": "PUT",
    "useragent": "docker/17.09.0-ce go/go1.8.3 git-commit/afdb6d4 kernel/4.10.0-27-generic os/linux arch/amd64 UpstreamClient(Docker-Client/17.09.0-ce \\(linux\\))"
  }
}
`

func TestAzureWebhookHandler(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache(100*time.Millisecond, 100*time.Millisecond, 10*time.Millisecond)
	am := approvals.New(mem, codecs.DefaultSerializer())
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers})
	srv.registerRoutes(srv.router)

	req, err := http.NewRequest("POST", "/v1/webhooks/azure", bytes.NewBuffer([]byte(fakeAzureWebhook)))
	if err != nil {
		t.Fatalf("failed to create req: %s", err)
	}

	//The response recorder used to record HTTP responses
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("unexpected status code: %d", rec.Code)

		t.Log(rec.Body.String())
	}

	if len(fp.submitted) != 1 {
		t.Fatalf("unexpected number of events submitted: %d", len(fp.submitted))
	}

	if fp.submitted[0].Repository.Name != "myregistry.azurecr.io/hello-world" {
		t.Errorf("myregistry.azurecr.io/hello-world but got %s", fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "v1" {
		t.Errorf("expected v1 but got %s", fp.submitted[0].Repository.Tag)
	}

	if fp.submitted[0].Repository.Digest != "sha256:80f0d5c8786bb9e621a45ece0db56d11cdc624ad20da9fe62e9d25490f331d7d" {
		t.Errorf("expected sha256:80f0d5c8786bb9e621a45ece0db56d11cdc624ad20da9fe62e9d25490f331d7d but got %s", fp.submitted[0].Repository.Digest)
	}
}
