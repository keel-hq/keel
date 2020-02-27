package http

import (
	"bytes"
	"net/http"

	"net/http/httptest"
	"testing"
)

var fakeHarborWebhook = ` {
    "type": "pushImage",
    "occur_at": 1582640688,
    "operator": "user",
    "event_data": {
        "resources": [
            {
                "digest": "sha256:b4758aaed11c155a476b9857e1178f157759c99cb04c907a04993f5481eff848",
                "tag": "2.1.6",
                "resource_url": "quay.io/mynamespace/repository:1.2.3"
            }
        ],
        "repository": {
            "date_created": 1582634337,
            "name": "repository",
            "namespace": "mynamespace",
            "repo_full_name": "mynamespace/repository",
            "repo_type": "private"
        }
    }
}
`

func TestQuayWebhookHandler(t *testing.T) {

	fp := &fakeProvider{}
	srv, teardown := NewTestingServer(fp)
	defer teardown()

	req, err := http.NewRequest("POST", "/v1/webhooks/harbor", bytes.NewBuffer([]byte(fakeHarborWebhook)))
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
