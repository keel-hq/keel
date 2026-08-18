package helm3

import (
	"sort"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

// releaseImageRunningDigests returns the digests that the Kubernetes workloads
// belonging to a release are running for a tracked image. It returns nil when
// the mapping or the runtime information is unavailable, which callers treat as
// "unknown" rather than "no drift".
func (p *Provider) releaseImageRunningDigests(namespace, releaseName string, trackedImage *types.TrackedImage) []string {
	if trackedImage == nil || trackedImage.Image == nil {
		return nil
	}
	return p.workloadRunningDigests(namespace, releaseName, trackedImage.Image.Repository(), trackedImage.Image.Tag())
}

// currentReleaseImageDigest returns the single digest the release workloads
// are running for the image identified by repository and tag, or an empty
// string when none is observable.
func (p *Provider) currentReleaseImageDigest(namespace, releaseName, repository, tag string) string {
	digests := p.workloadRunningDigests(namespace, releaseName, repository, tag)
	if len(digests) == 0 {
		return ""
	}
	return digests[0]
}

// workloadRunningDigests returns the sorted, de-duplicated digests that the
// Kubernetes workloads belonging to a release are running for an image
// identified by repository and tag. It returns nil when the runtime
// information is unavailable.
func (p *Provider) workloadRunningDigests(namespace, releaseName, repository, tag string) []string {
	if p.runningDigests == nil || p.resources == nil {
		return nil
	}

	found := make(map[string]struct{})
	for _, resource := range p.resources.Values() {
		annotations := resource.GetAnnotations()
		if resource.Namespace != namespace || annotations[helmReleaseNameAnnotation] != releaseName || annotations[helmReleaseNamespaceAnnotation] != namespace {
			continue
		}
		for img, digests := range p.runningDigests.Resolve(resource) {
			ref, err := image.Parse(img)
			if err != nil || ref.Repository() != repository || ref.Tag() != tag {
				continue
			}
			for _, digest := range digests {
				found[digest] = struct{}{}
			}
		}
	}

	if len(found) == 0 {
		return nil
	}
	result := make([]string, 0, len(found))
	for digest := range found {
		result = append(result, digest)
	}
	sort.Strings(result)
	return result
}
