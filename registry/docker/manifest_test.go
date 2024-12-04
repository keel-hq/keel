package docker

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	manifestV2 "github.com/distribution/distribution/v3/manifest/schema2"
)

func TestGetDigest(t *testing.T) {

	req, err := http.NewRequest("GET", "https://registry.opensource.zalan.do/v2/teapot/external-dns/manifests/v0.4.8", nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err)
	}
	req.Header.Set("Accept", manifestV2.MediaTypeManifest)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to request: %s", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/vnd.docker.distribution.manifest.v2+json; charset=ISO-8859-1")
		io.Copy(w, resp.Body)

		// Reset body for additional calls
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}))
	defer ts.Close()

	reg := New(ts.URL, "", "")

	digest, err := reg.ManifestDigest(ts.URL, "notimportant")
	if err != nil {
		t.Errorf("failed to get digest")
	}

	if digest.String() != "sha256:7aa5175f39a7e8a4172972524302c9a8196f681e40d6ee5d2f6bf0ab7d600fee" {
		t.Errorf("unexpected digest: %s", digest.String())
	}
}
