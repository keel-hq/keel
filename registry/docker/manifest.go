package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	manifestlist "github.com/distribution/distribution/v3/manifest/manifestlist"
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

// ManifestDigests - get every digest that identifies a reference. For an image
// index that is the digest of the index itself plus the digests of the
// per-platform manifests it points at, since a node runs one of the children
// while the registry reports the index. Single-platform manifests return one
// digest.
func (r *Registry) ManifestDigests(repository, reference string) ([]string, error) {
	url := r.url("/v2/%s/manifests/%s", repository, reference)
	r.Logf("registry.manifest.digests url=%s repository=%s reference=%s", url, repository, reference)

	resp, err := r.request("GET", url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("manifest request returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	top := resp.Header.Get("Docker-Content-Digest")
	if top == "" {
		top = digest.FromBytes(body).String()
	} else if parsed, err := digest.Parse(top); err == nil {
		top = parsed.String()
	} else {
		return nil, err
	}
	digests := []string{top}

	mediaType := resp.Header.Get("Content-Type")
	if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsed
	}
	switch mediaType {
	case manifestlist.MediaTypeManifestList, oci.MediaTypeImageIndex:
		var index oci.Index
		if err := json.Unmarshal(body, &index); err != nil {
			return nil, fmt.Errorf("decode image index: %w", err)
		}
		for _, descriptor := range index.Manifests {
			if descriptor.Digest == "" {
				continue
			}
			digests = append(digests, descriptor.Digest.String())
		}
	}

	return digests, nil
}

// request performs a request against a url
func (r *Registry) request(method string, url string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", strings.Join([]string{manifestlist.MediaTypeManifestList, manifestv2.MediaTypeManifest, oci.MediaTypeImageIndex, oci.MediaTypeImageManifest}, ","))
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
