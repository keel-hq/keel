package kubernetes

import (
	"fmt"
	"time"

	// "k8s.io/api/core/v1"

	// "k8s.io/api/extensions/v1beta1"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	"github.com/keel-hq/keel/util/version"

	log "github.com/sirupsen/logrus"
)

// func (p *Provider) checkVersionedDeployment(newVersion *types.Version, policy types.PolicyType, repo *types.Repository, deployment v1beta1.Deployment) (updated v1beta1.Deployment, shouldUpdateDeployment bool, err error) {
func (p *Provider) checkVersionedDeployment(newVersion *types.Version, policy types.PolicyType, repo *types.Repository, resource *k8s.GenericResource) (updatePlan *UpdatePlan, shouldUpdateDeployment bool, err error) {
	updatePlan = &UpdatePlan{}

	eventRepoRef, err := image.Parse(repo.String())
	if err != nil {
		return
	}

	labels := resource.GetLabels()

	log.WithFields(log.Fields{
		"labels":    labels,
		"name":      resource.Name,
		"namespace": resource.Namespace,
		"kind":      resource.Kind(),
		"policy":    policy,
	}).Info("provider.kubernetes.checkVersionedDeployment: keel policy found, checking resource...")

	shouldUpdateDeployment = false

	for idx, c := range resource.Containers() {
		containerImageRef, err := image.Parse(c.Image)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"image_name": c.Image,
			}).Error("provider.kubernetes: failed to parse image name")
			continue
		}

		log.WithFields(log.Fields{
			"name":              resource.Name,
			"namespace":         resource.Namespace,
			"parsed_image_name": containerImageRef.Remote(),
			"kind":              resource.Kind(),
			"target_image_name": repo.Name,
			"target_tag":        repo.Tag,
			"policy":            policy,
			"image":             c.Image,
		}).Info("provider.kubernetes: checking image")

		if containerImageRef.Repository() != eventRepoRef.Repository() {
			log.WithFields(log.Fields{
				"parsed_image_name": containerImageRef.Remote(),
				"target_image_name": repo.Name,
			}).Info("provider.kubernetes: images do not match, ignoring")
			continue
		}

		// if policy is force, don't bother with version checking
		// same with `latest` images, update them to versioned ones
		if policy == types.PolicyTypeForce || containerImageRef.Tag() == "latest" {
			if matchTag, _ := labels[types.KeelForceTagMatchLabel]; matchTag == "true" {
				if containerImageRef.Tag() != eventRepoRef.Tag() {
					continue
				}
			}
			if containerImageRef.Registry() == image.DefaultRegistryHostname {
				resource.UpdateContainer(idx, fmt.Sprintf("%s:%s", containerImageRef.ShortName(), newVersion.String()))
			} else {
				resource.UpdateContainer(idx, fmt.Sprintf("%s:%s", containerImageRef.Repository(), newVersion.String()))
			}
			shouldUpdateDeployment = true
			setUpdateTime(resource)

			log.WithFields(log.Fields{
				"parsed_image":     containerImageRef.Remote(),
				"raw_image_name":   c.Image,
				"target_image":     repo.Name,
				"target_image_tag": repo.Tag,
				"policy":           policy,
			}).Info("provider.kubernetes: impacted deployment container found")

			updatePlan.CurrentVersion = containerImageRef.Tag()
			updatePlan.NewVersion = newVersion.Original
			updatePlan.Resource = resource

			// success, moving to next container
			continue
		}

		currentVersion, err := version.GetVersionFromImageName(c.Image)
		if err != nil {
			log.WithFields(log.Fields{
				"error":               err,
				"container_image":     c.Image,
				"container_image_tag": containerImageRef.Tag(),
				"keel_policy":         policy,
			}).Error("provider.kubernetes: failed to get image version, is it tagged as semver?")
			continue
		}

		log.WithFields(log.Fields{
			"labels":          labels,
			"name":            resource.Name,
			"namespace":       resource.Namespace,
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
			"name":            resource.Name,
			"namespace":       resource.Namespace,
			"image":           c.Image,
			"current_version": currentVersion.String(),
			"new_version":     newVersion.String(),
			"policy":          policy,
			"should_update":   shouldUpdateContainer,
		}).Info("provider.kubernetes: checked version, deciding whether to update")

		if shouldUpdateContainer {
			if containerImageRef.Registry() == image.DefaultRegistryHostname {
				resource.UpdateContainer(idx, fmt.Sprintf("%s:%s", containerImageRef.ShortName(), newVersion.String()))
			} else {
				resource.UpdateContainer(idx, fmt.Sprintf("%s:%s", containerImageRef.Repository(), newVersion.String()))
			}
			// marking this deployment for update
			shouldUpdateDeployment = true

			setUpdateTime(resource)

			updatePlan.CurrentVersion = currentVersion.Original
			updatePlan.NewVersion = newVersion.Original
			updatePlan.Resource = resource

			log.WithFields(log.Fields{
				"parsed_image":     containerImageRef.Remote(),
				"raw_image_name":   c.Image,
				"target_image":     repo.Name,
				"target_image_tag": repo.Tag,
				"policy":           policy,
			}).Info("provider.kubernetes: impacted deployment container found")
		}
	}

	return updatePlan, shouldUpdateDeployment, nil
}

func setUpdateTime(resource *k8s.GenericResource) {
	specAnnotations := resource.GetSpecAnnotations()
	specAnnotations[types.KeelUpdateTimeAnnotation] = time.Now().String()
	resource.SetSpecAnnotations(specAnnotations)
}
