package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

// manifestResponse is used to extract the config digest from a v2 manifest.
type manifestResponse struct {
	Config struct {
		Digest string `json:"digest"`
	} `json:"config"`
}

// configResponse is used to extract the creation timestamp from an image config blob.
type configResponse struct {
	Created time.Time `json:"created"`
}

// ImageCreatedAt fetches the image creation timestamp by reading the config blob.
func (r *Registry) ImageCreatedAt(repository, reference string) (time.Time, error) {
	// Fetch the manifest to get the config digest
	url := r.url("/v2/%s/manifests/%s", repository, reference)
	r.Logf("registry.manifest.createdAt url=%s repository=%s reference=%s", url, repository, reference)

	resp, err := r.request("GET", url)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Time{}, err
	}

	var manifest manifestResponse
	if err := json.Unmarshal(body, &manifest); err != nil {
		return time.Time{}, err
	}

	if manifest.Config.Digest == "" {
		return time.Time{}, fmt.Errorf("no config digest in manifest for %s:%s", repository, reference)
	}

	// Fetch the config blob to get the creation date
	blobURL := r.url("/v2/%s/blobs/%s", repository, manifest.Config.Digest)
	r.Logf("registry.blob.get url=%s", blobURL)

	blobResp, err := r.Client.Get(blobURL)
	if err != nil {
		return time.Time{}, err
	}
	defer blobResp.Body.Close()

	blobBody, err := io.ReadAll(blobResp.Body)
	if err != nil {
		return time.Time{}, err
	}

	var config configResponse
	if err := json.Unmarshal(blobBody, &config); err != nil {
		return time.Time{}, err
	}

	return config.Created, nil
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
