package docker

import (
	"io/ioutil"
	"net/http"
	"strings"

	manifestv2 "github.com/docker/distribution/manifest/schema2"
	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

// ManifestDigest - get manifest digest
func (r *Registry) ManifestDigest(repository, reference string) (digest.Digest, error) {
	url := r.url("/v2/%s/manifests/%s", repository, reference)
	r.Logf("registry.manifest.head url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", strings.Join([]string{manifestv2.MediaTypeManifest, oci.MediaTypeImageIndex, oci.MediaTypeImageManifest}, ","))
	resp, err := r.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if hdr := resp.Header.Get("Docker-Content-Digest"); hdr != "" {
		return digest.Parse(hdr)
	}

	// Try to get digest from body instead, should be equal to what would be presented
	// in Docker-Content-Digest
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return digest.FromBytes(body), nil
}
