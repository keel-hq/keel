package kubernetes

import (
	"fmt"

	// "k8s.io/api/core/v1"

	"k8s.io/api/core/v1"

	// "k8s.io/api/extensions/v1beta1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	"github.com/keel-hq/keel/util/version"

	log "github.com/sirupsen/logrus"
)

// func (p *Provider) checkVersionedDeployment(newVersion *types.Version, policy types.PolicyType, repo *types.Repository, deployment v1beta1.Deployment) (updated v1beta1.Deployment, shouldUpdateDeployment bool, err error) {
func (p *Provider) checkVersionedDeployment(newVersion *types.Version, policy types.PolicyType, repo *types.Repository, deployment v1beta1.Deployment) (updatePlan *UpdatePlan, shouldUpdateDeployment bool, err error) {
	updatePlan = &UpdatePlan{}

	eventRepoRef, err := image.Parse(repo.Name)
	if err != nil {
		return
	}

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

		conatinerImageRef, err := image.Parse(c.Image)
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
			"parsed_image_name": conatinerImageRef.Remote(),
			"target_image_name": repo.Name,
			"target_tag":        repo.Tag,
			"policy":            policy,
			"image":             c.Image,
		}).Info("provider.kubernetes: checking image")

		if conatinerImageRef.Repository() != eventRepoRef.Repository() {
			log.WithFields(log.Fields{
				"parsed_image_name": conatinerImageRef.Remote(),
				"target_image_name": repo.Name,
			}).Info("provider.kubernetes: images do not match, ignoring")
			continue
		}

		// if policy is force, don't bother with version checking
		// same with `latest` images, update them to versioned ones
		if policy == types.PolicyTypeForce || conatinerImageRef.Tag() == "latest" {
			c = updateContainer(c, conatinerImageRef, newVersion.String())

			deployment.Spec.Template.Spec.Containers[idx] = c

			// marking this deployment for update
			shouldUpdateDeployment = true
			// updating digest if available
			annotations := deployment.GetAnnotations()

			if repo.Digest != "" {
				// annotations[types.KeelDigestAnnotation+"/"+conatinerImageRef.Remote()] = repo.Digest
			}
			annotations = addImageToPull(annotations, c.Image)

			deployment.SetAnnotations(annotations)
			log.WithFields(log.Fields{
				"parsed_image":     conatinerImageRef.Remote(),
				"raw_image_name":   c.Image,
				"target_image":     repo.Name,
				"target_image_tag": repo.Tag,
				"policy":           policy,
			}).Info("provider.kubernetes: impacted deployment container found")

			updatePlan.CurrentVersion = conatinerImageRef.Tag()
			updatePlan.NewVersion = newVersion.Original
			updatePlan.Deployment = deployment

			// success, moving to next container
			continue
		}

		currentVersion, err := version.GetVersionFromImageName(c.Image)
		if err != nil {
			log.WithFields(log.Fields{
				"error":               err,
				"container_image":     c.Image,
				"container_image_tag": conatinerImageRef.Tag(),
				"keel_policy":         policy,
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
			c = updateContainer(c, conatinerImageRef, newVersion.String())
			deployment.Spec.Template.Spec.Containers[idx] = c
			// marking this deployment for update
			shouldUpdateDeployment = true

			// updating annotations
			annotations := deployment.GetAnnotations()
			// updating digest if available
			if repo.Digest != "" {
				// annotations[types.KeelDigestAnnotation+"/"+conatinerImageRef.Remote()] = repo.Digest
			}
			deployment.SetAnnotations(annotations)

			updatePlan.CurrentVersion = currentVersion.Original
			updatePlan.NewVersion = newVersion.Original
			updatePlan.Deployment = deployment

			log.WithFields(log.Fields{
				"parsed_image":     conatinerImageRef.Remote(),
				"raw_image_name":   c.Image,
				"target_image":     repo.Name,
				"target_image_tag": repo.Tag,
				"policy":           policy,
			}).Info("provider.kubernetes: impacted deployment container found")
		}
	}

	return updatePlan, shouldUpdateDeployment, nil
}

func updateContainer(container v1.Container, ref *image.Reference, version string) v1.Container {
	// updating image
	if ref.Registry() == image.DefaultRegistryHostname {
		container.Image = fmt.Sprintf("%s:%s", ref.ShortName(), version)
	} else {
		container.Image = fmt.Sprintf("%s:%s", ref.Repository(), version)
	}

	return container
}
