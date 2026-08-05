package helm3

import (
	"sort"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

const (
	helmReleaseNameAnnotation      = "meta.helm.sh/release-name"
	helmReleaseNamespaceAnnotation = "meta.helm.sh/release-namespace"
)

func (p *Provider) releaseImagePlatforms(namespace, releaseName string, trackedImage *types.TrackedImage) ([]types.Platform, types.PlatformError) {
	if p.platforms == nil || p.resources == nil {
		return nil, types.PlatformErrorHelmWorkloadMapping
	}
	platforms := make(map[types.Platform]struct{})
	matched := false
	for _, resource := range p.resources.Values() {
		annotations := resource.GetAnnotations()
		if resource.Namespace != namespace || annotations[helmReleaseNameAnnotation] != releaseName || annotations[helmReleaseNamespaceAnnotation] != namespace {
			continue
		}
		if !resourceUsesRepository(resource.GetImages(nil), trackedImage.Image.Repository()) &&
			!resourceUsesRepository(resource.GetInitImages(nil), trackedImage.Image.Repository()) &&
			!resourceUsesRepository(resource.GetImageVolumeReferences(nil), trackedImage.Image.Repository()) {
			continue
		}
		matched = true
		resolved, resolutionErr := p.platforms.Resolve(resource)
		if resolutionErr != types.PlatformErrorNone {
			return nil, resolutionErr
		}
		for _, platform := range resolved {
			platforms[platform] = struct{}{}
		}
	}
	if !matched || len(platforms) == 0 {
		return nil, types.PlatformErrorHelmWorkloadMapping
	}
	result := make([]types.Platform, 0, len(platforms))
	for platform := range platforms {
		result = append(result, platform)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].OS != result[j].OS {
			return result[i].OS < result[j].OS
		}
		if result[i].Architecture != result[j].Architecture {
			return result[i].Architecture < result[j].Architecture
		}
		return result[i].Variant < result[j].Variant
	})
	return result, types.PlatformErrorNone
}

func resourceUsesRepository(images []string, repository string) bool {
	for _, candidate := range images {
		ref, err := image.Parse(candidate)
		if err == nil && ref.Repository() == repository {
			return true
		}
	}
	return false
}
