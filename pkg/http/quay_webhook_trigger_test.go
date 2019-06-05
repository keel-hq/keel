package http

import (
	"bytes"
	"net/http"

	"net/http/httptest"
	"testing"
)

var fakeQuayWebhook = `{
  "name": "repository",
  "repository": "mynamespace/repository",
  "namespace": "mynamespace",
  "docker_url": "quay.io/mynamespace/repository",
  "homepage": "https://quay.io/repository/mynamespace/repository",
  "updated_tags": [
    "1.2.3"
  ]
}
`

func TestQuayWebhookHandler(t *testing.T) {

	fp := &fakeProvider{}
	srv, teardown := NewTestingServer(fp)
	defer teardown()

	req, err := http.NewRequest("POST", "/v1/webhooks/quay", bytes.NewBuffer([]byte(fakeQuayWebhook)))
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

	if fp.submitted[0].Repository.Name != "quay.io/mynamespace/repository" {
		t.Errorf("expected quay.io/mynamespace/repository but got %s", fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "1.2.3" {
		t.Errorf("expected 1.2.3 but got %s", fp.submitted[0].Repository.Tag)
	}
}
