package kubernetes

import (
	"fmt"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"
	"github.com/rusenask/keel/util/policies"
	"github.com/rusenask/keel/util/version"

	log "github.com/Sirupsen/logrus"
)

func (p *Provider) checkDeployment(newVersion *types.Version, repo *types.Repository, deployment *v1beta1.Deployment) (updated v1beta1.Deployment, shouldUpdateDeployment bool, err error) {

	shouldUpdateDeployment = false
	updated = *deployment
	labels := deployment.GetLabels()

	policy := policies.GetPolicy(labels)
	if policy == types.PolicyTypeNone {
		return
	}

	log.WithFields(log.Fields{
		"labels":    labels,
		"name":      deployment.Name,
		"namespace": deployment.Namespace,
		"policy":    policy,
	}).Info("provider.kubernetes: keel policy found, checking deployment...")

	for idx, c := range deployment.Spec.Template.Spec.Containers {
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

		if ref.Remote() != repo.Name {
			log.WithFields(log.Fields{
				"parsed_image_name": ref.Remote(),
				"target_image_name": repo.Name,
			}).Info("provider.kubernetes: images do not match, ignoring")
			continue
		}

		currentVersion, err := version.GetVersion(ref.Tag())
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"image_name":  c.Image,
				"keel_policy": policy,
			}).Error("provider.kubernetes: failed to get image version, is it tagged as semver?")
			continue
		}

		log.WithFields(log.Fields{
			"labels":          labels,
			"name":            deployment.Name,
			"namespace":       deployment.Namespace,
			"image":           c.Image,
			"current_version": currentVersion.String(),
			"policy":          policy,
		}).Info("provider.kubernetes: current image version")

		shouldUpdateContainer, err := version.ShouldUpdate(currentVersion, newVersion, policy)
		if err != nil {
			log.WithFields(log.Fields{
				"error":           err,
				"new_version":     newVersion.String(),
				"current_version": currentVersion.String(),
				"keel_policy":     policy,
			}).Error("provider.kubernetes: got error while checking whether deployment should be updated")
			continue
		}

		log.WithFields(log.Fields{
			"labels":          labels,
			"name":            deployment.Name,
			"namespace":       deployment.Namespace,
			"image":           c.Image,
			"current_version": currentVersion.String(),
			"new_version":     newVersion.String(),
			"policy":          policy,
			"should_update":   shouldUpdateContainer,
		}).Info("provider.kubernetes: checked version, deciding whether to update")

		if shouldUpdateContainer {
			// updating image
			if ref.Registry() == image.DefaultRegistryHostname {
				c.Image = fmt.Sprintf("%s:%s", ref.ShortName(), newVersion.String())
			} else {
				c.Image = fmt.Sprintf("%s:%s", ref.Remote(), newVersion.String())
			}

			deployment.Spec.Template.Spec.Containers[idx] = c
			// marking this deployment for update
			shouldUpdateDeployment = true

			// updating digest if available
			if repo.Digest != "" {

				// labels[types.KeelDigestLabel] = hash.GetShort(repo.Digest)
			}

			log.WithFields(log.Fields{
				"parsed_image":     ref.Remote(),
				"raw_image_name":   c.Image,
				"target_image":     repo.Name,
				"target_image_tag": repo.Tag,
				"policy":           policy,
			}).Info("provider.kubernetes: impacted deployment container found")
		}
	}

	return updated, shouldUpdateDeployment, nil
}

func (p *Provider) semverPath(currentVersion, newVersion *types.Version, repo *types.Repository, container v1.Container) (updated v1.Container, shouldUpdateContainer bool, err error) {
	return
}

func (p *Provider) forceUpdatePath(repo *types.Repository, container v1.Container) (updated v1.Container, shouldUpdateContainer bool, err error) {
	return
}
