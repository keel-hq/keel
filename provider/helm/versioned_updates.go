package helm

import (
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"github.com/keel-hq/keel/util/version"

	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"

	log "github.com/sirupsen/logrus"
)

func checkVersionedRelease(newVersion *types.Version, repo *types.Repository, namespace, name string, chart *hapi_chart.Chart, config *hapi_chart.Config) (plan *UpdatePlan, shouldUpdateRelease bool, err error) {
	plan = &UpdatePlan{
		Chart:     chart,
		Namespace: namespace,
		Name:      name,
		Values:    make(map[string]string),
	}

	eventRepoRef, err := image.Parse(repo.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"repository_name": repo.Name,
		}).Error("provider.helm: failed to parse event repository name")
		return
	}

	// getting configuration
	vals, err := values(chart, config)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("provider.helm: failed to get values.yaml for release")
		return
	}

	keelCfg, err := getKeelConfig(vals)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("provider.helm: failed to get keel configuration for release")
		// ignoring this release, no keel config found
		return plan, false, nil
	}
	// checking for impacted images
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

		if imageRef.Repository() != eventRepoRef.Repository() {
			log.WithFields(log.Fields{
				"parsed_image_name": imageRef.Remote(),
				"target_image_name": repo.Name,
			}).Info("provider.helm: images do not match, ignoring")
			continue
		}

		// checking policy and whether we should update
		if keelCfg.Policy == types.PolicyTypeForce || imageRef.Tag() == "latest" {
			path, value := getPlanValues(newVersion, imageRef, &imageDetails)
			plan.Values[path] = value
			plan.NewVersion = newVersion.String()
			plan.CurrentVersion = imageRef.Tag()
			plan.Config = keelCfg
			shouldUpdateRelease = true

			log.WithFields(log.Fields{
				"parsed_image":     imageRef.Remote(),
				"target_image":     repo.Name,
				"target_image_tag": repo.Tag,
				"policy":           keelCfg.Policy,
			}).Info("provider.helm: impacted release container found")
			continue
		}

		// checking current
		currentVersion, err := version.GetVersion(imageRef.Tag())
		if err != nil {
			log.WithFields(log.Fields{
				"error":               err,
				"container_image":     imageRef.Repository(),
				"container_image_tag": imageRef.Tag(),
				"keel_policy":         keelCfg.Policy,
			}).Error("provider.helm: failed to get image version, is it tagged as semver?")
			continue
		}

		log.WithFields(log.Fields{
			"name":            name,
			"namespace":       namespace,
			"container_image": imageRef.Repository(),
			"current_version": currentVersion.String(),
			"policy":          keelCfg.Policy,
		}).Info("provider.helm: current image version")

		shouldUpdate, err := version.ShouldUpdate(currentVersion, newVersion, keelCfg.Policy)
		if err != nil {
			log.WithFields(log.Fields{
				"error":           err,
				"new_version":     newVersion.String(),
				"current_version": currentVersion.String(),
				"keel_policy":     keelCfg.Policy,
			}).Error("provider.helm: got error while checking whether deployment should be updated")
			continue
		}

		log.WithFields(log.Fields{
			"name":            name,
			"namespace":       namespace,
			"container_image": imageRef.Repository(),
			"current_version": currentVersion.String(),
			"new_version":     newVersion.String(),
			"policy":          keelCfg.Policy,
			"should_update":   shouldUpdate,
		}).Info("provider.helm: checked version, deciding whether to update")

		if shouldUpdate {
			path, value := getPlanValues(newVersion, imageRef, &imageDetails)
			plan.Values[path] = value
			plan.NewVersion = newVersion.String()
			plan.CurrentVersion = currentVersion.String()
			plan.Config = keelCfg
			shouldUpdateRelease = true

			log.WithFields(log.Fields{
				"container_image":  imageRef.Repository(),
				"target_image":     repo.Name,
				"target_image_tag": repo.Tag,
				"policy":           keelCfg.Policy,
			}).Info("provider.helm: impacted release tags found")
		}

	}
	return plan, shouldUpdateRelease, nil
}
