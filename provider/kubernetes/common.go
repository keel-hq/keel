package kubernetes

import (
	"strings"
)

func addImageToPull(annotations map[string]string, image string) map[string]string {
	existing, ok := annotations[forceUpdateImageAnnotation]
	if ok {
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
