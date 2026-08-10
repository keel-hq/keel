package docker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	manifestlist "github.com/distribution/distribution/v3/manifest/manifestlist"
	manifestv2 "github.com/distribution/distribution/v3/manifest/schema2"
	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestManifestDigests(t *testing.T) {
	const (
		indexDigest    = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
		amd64Digest    = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
		arm64Digest    = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
		manifestDigest = "sha256:4444444444444444444444444444444444444444444444444444444444444444"
	)

	indexBody, err := json.Marshal(oci.Index{
		MediaType: oci.MediaTypeImageIndex,
		Manifests: []oci.Descriptor{
			{Digest: digest.Digest(amd64Digest), Platform: &oci.Platform{OS: "linux", Architecture: "amd64"}},
			{Digest: digest.Digest(arm64Digest), Platform: &oci.Platform{OS: "linux", Architecture: "arm64"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestBody, err := json.Marshal(oci.Manifest{MediaType: manifestv2.MediaTypeManifest})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/example/image/manifests/multi":
			w.Header().Set("Content-Type", manifestlist.MediaTypeManifestList+"; charset=utf-8")
			w.Header().Set("Docker-Content-Digest", indexDigest)
			_, _ = w.Write(indexBody)
		case "/v2/example/image/manifests/single":
			w.Header().Set("Content-Type", manifestv2.MediaTypeManifest)
			w.Header().Set("Docker-Content-Digest", manifestDigest)
			_, _ = w.Write(manifestBody)
		case "/v2/example/image/manifests/unsigned":
			// registries are allowed to omit the digest header
			w.Header().Set("Content-Type", manifestv2.MediaTypeManifest)
			_, _ = w.Write(manifestBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	registry := New(server.URL, "", "")
	registry.Logf = func(string, ...interface{}) {}

	tests := []struct {
		reference string
		want      []string
	}{
		{reference: "multi", want: []string{indexDigest, amd64Digest, arm64Digest}},
		{reference: "single", want: []string{manifestDigest}},
		{reference: "unsigned", want: []string{digest.FromBytes(manifestBody).String()}},
	}
	for _, test := range tests {
		t.Run(test.reference, func(t *testing.T) {
			got, err := registry.ManifestDigests("example/image", test.reference)
			if err != nil {
				t.Fatalf("failed to get manifest digests: %s", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("unexpected digests: %v, want %v", got, test.want)
			}
		})
	}

	if _, err := registry.ManifestDigests("example/image", "missing"); err == nil {
		t.Error("expected an error for a missing manifest")
	}
}
