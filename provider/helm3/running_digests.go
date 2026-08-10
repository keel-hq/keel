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
	if p.runningDigests == nil || p.resources == nil || trackedImage == nil || trackedImage.Image == nil {
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
			if err != nil || ref.Repository() != trackedImage.Image.Repository() || ref.Tag() != trackedImage.Image.Tag() {
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
