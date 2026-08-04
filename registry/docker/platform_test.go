package docker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	manifestlist "github.com/distribution/distribution/v3/manifest/manifestlist"
	manifestv2 "github.com/distribution/distribution/v3/manifest/schema2"
	"github.com/keel-hq/keel/types"
	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestManifestPlatformsRegistryFixture(t *testing.T) {
	const (
		amd64Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		armDigest   = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	manifest := func(configDigest string) []byte {
		body, err := json.Marshal(oci.Manifest{
			MediaType: manifestv2.MediaTypeManifest,
			Config: oci.Descriptor{
				MediaType: oci.MediaTypeImageConfig,
				Digest:    digest.Digest(configDigest),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	indexBody, err := json.Marshal(oci.Index{
		MediaType: oci.MediaTypeImageIndex,
		Manifests: []oci.Descriptor{
			{Platform: &oci.Platform{OS: "linux", Architecture: "amd64"}},
			{Platform: &oci.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/example/image/manifests/multi" && !strings.Contains(r.Header.Get("Accept"), manifestlist.MediaTypeManifestList) {
			t.Error("manifest request did not advertise Docker manifest-list support")
		}
		switch r.URL.Path {
		case "/v2/example/image/manifests/amd64":
			w.Header().Set("Content-Type", manifestv2.MediaTypeManifest)
			_, _ = w.Write(manifest(amd64Digest))
		case "/v2/example/image/manifests/armhf":
			w.Header().Set("Content-Type", manifestv2.MediaTypeManifest)
			_, _ = w.Write(manifest(armDigest))
		case "/v2/example/image/manifests/multi":
			w.Header().Set("Content-Type", manifestlist.MediaTypeManifestList)
			_, _ = w.Write(indexBody)
		case "/v2/example/image/blobs/" + amd64Digest:
			_ = json.NewEncoder(w).Encode(oci.Image{Platform: oci.Platform{OS: "linux", Architecture: "amd64"}})
		case "/v2/example/image/blobs/" + armDigest:
			_ = json.NewEncoder(w).Encode(oci.Image{Platform: oci.Platform{OS: "linux", Architecture: "arm", Variant: "v7"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	registry := New(server.URL, "", "")
	tests := []struct {
		tag  string
		want []types.Platform
	}{
		{tag: "amd64", want: []types.Platform{{OS: "linux", Architecture: "amd64"}}},
		{tag: "armhf", want: []types.Platform{{OS: "linux", Architecture: "arm", Variant: "v7"}}},
		{tag: "multi", want: []types.Platform{
			{OS: "linux", Architecture: "amd64"},
			{OS: "linux", Architecture: "arm64", Variant: "v8"},
		}},
	}
	for _, test := range tests {
		t.Run(test.tag, func(t *testing.T) {
			got, err := registry.ManifestPlatforms("example/image", test.tag)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("got %v, want %v", got, test.want)
				}
			}
		})
	}
}
