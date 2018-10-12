package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/cache/memory"
	"github.com/keel-hq/keel/provider"
)

var fakeRegistryNotificationWebhook = `{
	"events": [
	   {
		  "id": "d83e8796-7ba5-46ad-b239-d88473e21b2b",
		  "timestamp": "2018-10-11T13:53:21.859222576Z",
		  "action": "push",
		  "target": {
			 "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			 "size": 2206,
			 "digest": "sha256:4afff550708506c5b8b7384ad10d401a02b29ed587cb2730cb02753095b5178d",
			 "length": 2206,
			 "repository": "foo/bar",
			 "url": "https://registry.git.erxes.io/v2/foo/bar/manifests/sha256:4afff550708506c5b8b7384ad10d401a02b29ed587cb2730cb02753095b5178d",
			 "tag": "1.6.1"
		  },
		  "request": {
			 "id": "18690582-6d1a-4e08-8825-251a0adc58ce",
			 "addr": "46.101.177.27",
			 "host": "registry.git.erxes.io",
			 "method": "PUT",
			 "useragent": "docker/18.06.1-ce go/go1.10.3 git-commit/e68fc7a kernel/4.4.0-135-generic os/linux arch/amd64 UpstreamClient(Docker-Client/18.06.1-ce \\(linux\\))"
		  },
		  "actor": {
			 "name": "foo"
		  },
		  "source": {
			 "addr": "git.erxes.io:5000",
			 "instanceID": "bde27723-d67e-4775-a9bd-55f771a2f895"
		  }
	   }
	]
 }
  `

func TestRegistryNotificationsHandler(t *testing.T) {

	fp := &fakeProvider{}
	mem := memory.NewMemoryCache()
	am := approvals.New(mem)
	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{Providers: providers})
	srv.registerRoutes(srv.router)

	req, err := http.NewRequest("POST", "/v1/webhooks/registry", bytes.NewBuffer([]byte(fakeRegistryNotificationWebhook)))
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

	if fp.submitted[0].Repository.Name != "registry.git.erxes.io/foo/bar" {
		t.Errorf("expected registry.git.erxes.io/foo/bar but got %s", fp.submitted[0].Repository.Name)
	}

	if fp.submitted[0].Repository.Tag != "1.6.1" {
		t.Errorf("expected 1.6.1 but got %s", fp.submitted[0].Repository.Tag)
	}
}
