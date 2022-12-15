package http

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"net/http/httptest"
	"testing"
)

var fakeJfrogWebhook = `{
   "domain": "docker",
   "event_type": "pushed",
   "data": {
     "repo_key":"docker-remote-cache",
     "event_type":"pushed",
     "path":"library/ubuntu/latest/list.manifest.json",
     "name":"list.manifest.json",
     "sha256":"35c4a2c15539c6c1e4e5fa4e554dac323ad0107d8eb5c582d6ff386b383b7dce",
     "size":1206,
     "image_name":"library/ubuntu",
     "tag":"latest",
     "platforms":[
        {
           "architecture":"amd64",
           "os":"linux"
        },
        {
           "architecture":"arm",
           "os":"linux"
        },
        {
           "architecture":"arm64",
           "os":"linux"
        },
        {
           "architecture":"ppc64le",
           "os":"linux"
        },
        {
           "architecture":"s390x",
           "os":"linux"
      }
    ]
  },
  "subscription_key": "test",
  "jpd_origin": "https://example.jfrog.io",
  "source": "jfrog/user@example.com"
}`

func TestJfrogWebhookHandler(t *testing.T) {

	fp := &fakeProvider{}
	srv, teardown := NewTestingServer(fp)
	defer teardown()

	req, err := http.NewRequest("POST", "/v1/webhooks/jfrog", bytes.NewBuffer([]byte(fakeJfrogWebhook)))
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

	expected_repo_name := "library/ubuntu"
	if pr, ok := os.LookupEnv("PRIVATE_REGISTRY"); ok {
		expected_repo_name = fmt.Sprintf("%s/%s", pr, "library/ubuntu")
	}

	if fp.submitted[0].Repository.Name != expected_repo_name {
		t.Errorf("expected %s but got %s", expected_repo_name, fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "latest" {
		t.Errorf("expected latest but got %s", fp.submitted[0].Repository.Tag)
	}
}
