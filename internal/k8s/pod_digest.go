package k8s

import (
	"strings"

	core_v1 "k8s.io/api/core/v1"
)

// ExtractDigestFromImageID parses the digest from a pod's imageID field.
// The imageID typically looks like:
//   - "docker-pullable://registry/image@sha256:abc123..."
//   - "registry/image@sha256:abc123..."
//
// Returns the "sha256:abc123..." portion, or empty string if not found.
func ExtractDigestFromImageID(imageID string) string {
	// imageID may start with "docker-pullable://" or similar prefix
	idx := strings.LastIndex(imageID, "@")
	if idx == -1 {
		return ""
	}
	digest := imageID[idx+1:]
	// Basic validation: should start with a known algorithm prefix
	if strings.HasPrefix(digest, "sha256:") || strings.HasPrefix(digest, "sha512:") {
		return digest
	}
	return ""
}

// FindPodImageDigest searches through a list of pods' container statuses
// for a container running the given image (matched by repository name)
// and returns the digest extracted from its imageID.
// Returns empty string if no matching container is found or imageID is empty.
func FindPodImageDigest(pods []core_v1.Pod, imageRepo string) string {
	for _, pod := range pods {
		// Check regular container statuses
		for _, cs := range pod.Status.ContainerStatuses {
			digest := extractDigestIfMatch(cs.Image, cs.ImageID, imageRepo)
			if digest != "" {
				return digest
			}
		}
		// Check init container statuses
		for _, cs := range pod.Status.InitContainerStatuses {
			digest := extractDigestIfMatch(cs.Image, cs.ImageID, imageRepo)
			if digest != "" {
				return digest
			}
		}
	}
	return ""
}

// extractDigestIfMatch checks if the container status image matches the
// target image repository and returns the digest from imageID if so.
func extractDigestIfMatch(statusImage, imageID, targetImageRepo string) string {
	if imageID == "" {
		return ""
	}
	// Normalize: strip tag from status image for comparison
	statusRepo := statusImage
	if idx := strings.LastIndex(statusRepo, ":"); idx != -1 {
		// But be careful not to strip port numbers; only strip if after last "/"
		lastSlash := strings.LastIndex(statusRepo, "/")
		if idx > lastSlash {
			statusRepo = statusRepo[:idx]
		}
	}
	// Also strip tag from target for comparison
	targetRepo := targetImageRepo
	if idx := strings.LastIndex(targetRepo, ":"); idx != -1 {
		lastSlash := strings.LastIndex(targetRepo, "/")
		if idx > lastSlash {
			targetRepo = targetRepo[:idx]
		}
	}

	// Compare repositories - handle docker.io normalization
	if normalizeRepo(statusRepo) == normalizeRepo(targetRepo) {
		return ExtractDigestFromImageID(imageID)
	}
	return ""
}

// normalizeRepo normalizes Docker Hub image names for comparison.
// e.g., "nginx" -> "docker.io/library/nginx"
//
// "myuser/myimg" -> "docker.io/myuser/myimg"
func normalizeRepo(repo string) string {
	// Remove any scheme prefix
	if idx := strings.Index(repo, "://"); idx != -1 {
		repo = repo[idx+3:]
	}

	parts := strings.Split(repo, "/")
	switch len(parts) {
	case 1:
		// e.g., "nginx" -> "docker.io/library/nginx"
		return "docker.io/library/" + parts[0]
	case 2:
		// Could be "myuser/myimg" (Docker Hub) or "gcr.io/myimg"
		// If the first part contains a dot, it's a registry
		if strings.Contains(parts[0], ".") {
			return repo
		}
		return "docker.io/" + repo
	default:
		return repo
	}
}