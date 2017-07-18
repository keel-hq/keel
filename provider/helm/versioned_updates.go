package helm

import (
	"github.com/rusenask/keel/types"

	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"

	log "github.com/Sirupsen/logrus"
)

func (p *Provider) checkVersionedRelease(newVersion *types.Version, namespace, name string, chart *hapi_chart.Chart, config *hapi_chart.Config) (plan *UpdatePlan, shouldUpdateRelease bool, err error) {
	plan = &UpdatePlan{
		Chart:     chart,
		Namespace: namespace,
		Name:      name,
		Values:    make(map[string]string),
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
		continue
	}

	// checking for impacted images
	for _, imageDetails := range keelCfg.Images {
		imageRef, err := parseImage(vals, &imageDetails)
		if err != nil {
			log.WithFields(log.Fields{
				"error":           err,
				"repository_name": imageDetails.Repository,
				"repository_tag":  imageDetails.Tag,
			}).Error("provider.helm: failed to parse image")
			continue
		}

	}

}
