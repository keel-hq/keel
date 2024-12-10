package docker

import (
	"io"
	"net/http"
	"strings"

	manifestv2 "github.com/distribution/distribution/v3/manifest/schema2"
	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

// ManifestDigest - get manifest digest
func (r *Registry) ManifestDigest(repository, reference string) (digest.Digest, error) {
	url := r.url("/v2/%s/manifests/%s", repository, reference)
	r.Logf("registry.manifest.head url=%s repository=%s reference=%s", url, repository, reference)

	// Try HEAD request first because it's free
	resp, err := r.request("HEAD", url)
	if err != nil {
		return "", err
	}

	if hdr := resp.Header.Get("Docker-Content-Digest"); hdr != "" {
		return digest.Parse(hdr)
	}

	// HEAD request didn't return a digest, attempt to fetch digest from body
	r.Logf("registry.manifest.get url=%s repository=%s reference=%s", url, repository, reference)
	resp, err = r.request("GET", url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Try to get digest from body instead, should be equal to what would be presented
	// in Docker-Content-Digest
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return digest.FromBytes(body), nil
}

// request performs a request against a url
func (r *Registry) request(method string, url string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", strings.Join([]string{manifestv2.MediaTypeManifest, oci.MediaTypeImageIndex, oci.MediaTypeImageManifest}, ","))
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
