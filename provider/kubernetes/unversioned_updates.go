package kubernetes

import (
	"fmt"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"

	log "github.com/Sirupsen/logrus"
)

func (p *Provider) checkUnversionedDeployment(policy types.PolicyType, repo *types.Repository, deployment v1beta1.Deployment) (updated v1beta1.Deployment, shouldUpdateDeployment bool, err error) {
	labels := deployment.GetLabels()

	log.WithFields(log.Fields{
		"labels":    labels,
		"name":      deployment.Name,
		"namespace": deployment.Namespace,
		"policy":    policy,
	}).Info("provider.kubernetes.checkVersionedDeployment: keel policy found, checking deployment...")

	shouldUpdateDeployment = false

	for idx, c := range deployment.Spec.Template.Spec.Containers {
		// Remove version if any
		// containerImageName := versionreg.ReplaceAllString(c.Image, "")

		ref, err := image.Parse(c.Image)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"image_name": c.Image,
			}).Error("provider.kubernetes: failed to parse image name")
			continue
		}

		log.WithFields(log.Fields{
			"name":              deployment.Name,
			"namespace":         deployment.Namespace,
			"parsed_image_name": ref.Remote(),
			"target_image_name": repo.Name,
			"target_tag":        repo.Tag,
			"policy":            policy,
			"image":             c.Image,
		}).Info("provider.kubernetes: checking image")

		if ref.Repository() != repo.Name {
			log.WithFields(log.Fields{
				"parsed_image_name": ref.Remote(),
				"target_image_name": repo.Name,
			}).Info("provider.kubernetes: images do not match, ignoring")
			continue
		}

		// updating image
		if ref.Registry() == image.DefaultRegistryHostname {
			c.Image = fmt.Sprintf("%s:%s", ref.ShortName(), repo.Tag)
		} else {
			c.Image = fmt.Sprintf("%s:%s", ref.Repository(), repo.Tag)
		}

		deployment.Spec.Template.Spec.Containers[idx] = c
		// marking this deployment for update
		shouldUpdateDeployment = true

		// updating digest if available
		if repo.Digest != "" {
			annotations := deployment.GetAnnotations()
			annotations[types.KeelDigestLabel+"/"+ref.Remote()] = repo.Digest
			deployment.SetAnnotations(annotations)
		}

		log.WithFields(log.Fields{
			"parsed_image":     ref.Remote(),
			"raw_image_name":   c.Image,
			"target_image":     repo.Name,
			"target_image_tag": repo.Tag,
			"policy":           policy,
		}).Info("provider.kubernetes: impacted deployment container found")

	}

	return deployment, shouldUpdateDeployment, nil
}
