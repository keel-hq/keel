package kubernetes

import (
	"strings"
)

func addImageToPull(annotations map[string]string, image string) map[string]string {
	existing, ok := annotations[forceUpdateImageAnnotation]
	if ok {
		// check if it's already there
		if shouldPullImage(annotations, image) {
			// skipping
			return annotations
		}

		annotations[forceUpdateImageAnnotation] = existing + "," + image
		return annotations
	}
	annotations[forceUpdateImageAnnotation] = image
	return annotations
}

func shouldPullImage(annotations map[string]string, image string) bool {
	imagesStr, ok := annotations[forceUpdateImageAnnotation]
	if !ok {
		return false
	}

	images := strings.Split(imagesStr, ",")
	for _, img := range images {
		if img == image {
			return true
		}
	}
	return false
}
