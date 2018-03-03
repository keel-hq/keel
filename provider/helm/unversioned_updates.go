package helm

import (
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"

	log "github.com/sirupsen/logrus"
)

func checkUnversionedRelease(repo *types.Repository, namespace, name string, chart *hapi_chart.Chart, config *hapi_chart.Config) (plan *UpdatePlan, shouldUpdateRelease bool, err error) {

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

	if keelCfg.Policy != types.PolicyTypeForce {
		// policy is not force, ignoring release
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

		path, value := getUnversionedPlanValues(repo.Tag, imageRef, &imageDetails)
		plan.Values[path] = value
		plan.NewVersion = repo.Tag
		plan.CurrentVersion = imageRef.Tag()
		plan.Config = keelCfg
		shouldUpdateRelease = true
	}

	return plan, shouldUpdateRelease, nil
}
