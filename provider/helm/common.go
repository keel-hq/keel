package helm

import (
	"errors"
	"fmt"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	"k8s.io/helm/pkg/chartutil"

	log "github.com/sirupsen/logrus"
)

// ErrKeelConfigNotFound - default error when keel configuration for chart is not defined
var ErrKeelConfigNotFound = errors.New("keel configuration not found")

// getImages - get images from chart values
func getImages(vals chartutil.Values) ([]*types.TrackedImage, error) {
	keelCfg, err := getKeelConfig(vals)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("provider.helm: failed to get keel configuration for release")
		// ignoring this release, no keel config found
		return nil, ErrKeelConfigNotFound
	}

	var images []*types.TrackedImage

	for _, imageDetails := range keelCfg.Images {
		imageRef, err := parseImage(vals, &imageDetails)
		if err != nil {
			log.WithFields(log.Fields{
				"error":           err,
				"repository_name": imageDetails.RepositoryPath,
				"repository_tag":  imageDetails.TagPath,
			}).Error("provider.helm: failed to parse image")
			continue
		}

		trackedImage := &types.TrackedImage{
			Image:        imageRef,
			PollSchedule: keelCfg.PollSchedule,
			Trigger:      keelCfg.Trigger,
		}

		images = append(images, trackedImage)
	}

	return images, nil
}

func getPlanValues(newVersion *types.Version, ref *image.Reference, imageDetails *ImageDetails) (path, value string) {
	// vals := make(map[string]string)
	// if tag is not supplied, then user specified full image name
	if imageDetails.TagPath == "" {
		return imageDetails.RepositoryPath, getUpdatedImage(ref, newVersion.String())
	}
	return imageDetails.TagPath, newVersion.String()
}

func getUnversionedPlanValues(newTag string, ref *image.Reference, imageDetails *ImageDetails) (path, value string) {
	// if tag is not supplied, then user specified full image name
	if imageDetails.TagPath == "" {
		return imageDetails.RepositoryPath, getUpdatedImage(ref, newTag)
	}
	return imageDetails.TagPath, newTag
}

func getUpdatedImage(ref *image.Reference, version string) string {
	// updating image
	if ref.Registry() == image.DefaultRegistryHostname {
		return fmt.Sprintf("%s:%s", ref.ShortName(), version)
	}
	return fmt.Sprintf("%s:%s", ref.Repository(), version)
}

func parseImage(vals chartutil.Values, details *ImageDetails) (*image.Reference, error) {
	if details.RepositoryPath == "" {
		return nil, fmt.Errorf("repository name path cannot be empty")
	}

	imageName, err := getValueAsString(vals, details.RepositoryPath)
	if err != nil {
		return nil, err
	}

	// getting image tag
	imageTag, err := getValueAsString(vals, details.TagPath)
	if err != nil {
		// failed to find tag, returning anyway
		return image.Parse(imageName)
	}

	return image.Parse(imageName + ":" + imageTag)
}
