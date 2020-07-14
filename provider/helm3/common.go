package helm3

import (
	"errors"
	"fmt"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	"helm.sh/helm/v3/pkg/chartutil"

	log "github.com/sirupsen/logrus"
)

// ErrKeelConfigNotFound - default error when keel configuration for chart is not defined
var ErrKeelConfigNotFound = errors.New("keel configuration not found")

// getImages - get images from chart values
func getImages(vals chartutil.Values) ([]*types.TrackedImage, error) {
	var images []*types.TrackedImage

	keelCfg, err := getKeelConfig(vals)
	if err != nil {
		if err == ErrPolicyNotSpecified {
			// nothing to do
			return images, nil
		}
		log.WithFields(log.Fields{
			"error": err,
		}).Error("provider.helm3: failed to get keel configuration for release")
		// ignoring this release, no keel config found
		return nil, ErrKeelConfigNotFound
	}

	for _, imageDetails := range keelCfg.Images {
		imageRef, err := parseImage(vals, &imageDetails)
		if err != nil {
			log.WithFields(log.Fields{
				"error":           err,
				"repository_name": imageDetails.RepositoryPath,
				"repository_tag":  imageDetails.TagPath,
			}).Error("provider.helm3: failed to parse image")
			continue
		}

		trackedImage := &types.TrackedImage{
			Image:        imageRef,
			PollSchedule: keelCfg.PollSchedule,
			Trigger:      keelCfg.Trigger,
			Policy:       keelCfg.Plc,
		}

		if imageDetails.ImagePullSecret != "" {
			trackedImage.Secrets = append(trackedImage.Secrets, imageDetails.ImagePullSecret)
		}

		images = append(images, trackedImage)
	}

	return images, nil
}

func getPlanValues(newVersion *types.Version, ref *image.Reference, imageDetails *ImageDetails) (path, value string) {
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
