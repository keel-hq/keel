package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"

	manifestlist "github.com/distribution/distribution/v3/manifest/manifestlist"
	manifestv2 "github.com/distribution/distribution/v3/manifest/schema2"
	"github.com/keel-hq/keel/types"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

// ManifestPlatforms resolves the platforms supported by a manifest or image
// index. Single-platform manifests get their platform from the image config.
func (r *Registry) ManifestPlatforms(repository, reference string) ([]types.Platform, error) {
	manifestURL := r.url("/v2/%s/manifests/%s", repository, reference)
	resp, err := r.request(http.MethodGet, manifestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("manifest request returned %s", resp.Status)
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("invalid manifest content type: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch mediaType {
	case manifestlist.MediaTypeManifestList, oci.MediaTypeImageIndex:
		var index oci.Index
		if err := json.Unmarshal(body, &index); err != nil {
			return nil, fmt.Errorf("decode image index: %w", err)
		}
		platforms := make([]types.Platform, 0, len(index.Manifests))
		for _, descriptor := range index.Manifests {
			if descriptor.Platform == nil || descriptor.Platform.OS == "" || descriptor.Platform.Architecture == "" {
				continue
			}
			platforms = append(platforms, types.Platform{
				OS:           descriptor.Platform.OS,
				Architecture: descriptor.Platform.Architecture,
				Variant:      descriptor.Platform.Variant,
			})
		}
		if len(platforms) == 0 {
			return nil, fmt.Errorf("image index contains no platform metadata")
		}
		return platforms, nil

	case manifestv2.MediaTypeManifest, oci.MediaTypeImageManifest:
		var manifest oci.Manifest
		if err := json.Unmarshal(body, &manifest); err != nil {
			return nil, fmt.Errorf("decode image manifest: %w", err)
		}
		if manifest.Config.Digest == "" {
			return nil, fmt.Errorf("image manifest has no config digest")
		}
		return r.configPlatforms(repository, manifest.Config.Digest.String())
	default:
		return nil, fmt.Errorf("unsupported manifest content type %q", mediaType)
	}
}

func (r *Registry) configPlatforms(repository, configDigest string) ([]types.Platform, error) {
	configURL := r.url("/v2/%s/blobs/%s", repository, configDigest)
	resp, err := r.request(http.MethodGet, configURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("image config request returned %s", resp.Status)
	}

	var config oci.Image
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("decode image config: %w", err)
	}
	if config.OS == "" || config.Architecture == "" {
		return nil, fmt.Errorf("image config contains no platform metadata")
	}
	return []types.Platform{{
		OS:           config.OS,
		Architecture: config.Architecture,
		Variant:      config.Variant,
	}}, nil
}
