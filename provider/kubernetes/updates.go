package kubernetes

import (
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	v1 "k8s.io/api/core/v1"

	log "github.com/sirupsen/logrus"
)

func checkForUpdate(plc policy.Policy, repo *types.Repository, resource *k8s.GenericResource) (updatePlan *UpdatePlan, shouldUpdateDeployment bool, err error) {
	updatePlan = &UpdatePlan{}

	eventRepoRef, err := image.Parse(repo.String())
	if err != nil {
		return
	}
	alreadyDeployed := repo.Digest != "" && deployedDigests(resource)[eventRepoRef.Remote()] == repo.Digest

	log.WithFields(log.Fields{
		"name":      resource.Name,
		"namespace": resource.Namespace,
		"kind":      resource.Kind(),
		"policy":    plc.Name(),
	}).Debug("provider.kubernetes.checkVersionedDeployment: keel policy found, checking resource...")
	shouldUpdateDeployment = false

	containerFilterFunc := GetMonitorContainersFromMeta(resource.GetAnnotations(), resource.GetLabels())
	volumeFilterFunc := GetMonitorVolumesFromMeta(resource.GetAnnotations(), resource.GetLabels())

	if getImageVolumeTrackingFromMeta(resource.GetLabels(), resource.GetAnnotations()) {
		for idx, vol := range resource.Volumes() {
			if vol.Image == nil || vol.Image.Reference == "" {
				continue
			}
			if !volumeFilterFunc(vol) {
				continue
			}

			volumeImageRef, err := image.Parse(vol.Image.Reference)
			if err != nil {
				log.WithFields(log.Fields{
					"error":      err,
					"image_name": vol.Image.Reference,
				}).Error("provider.kubernetes: failed to parse image volume reference")
				continue
			}

			log.WithFields(log.Fields{
				"name":              resource.Name,
				"namespace":         resource.Namespace,
				"kind":              resource.Kind(),
				"volume":            vol.Name,
				"parsed_image_name": volumeImageRef.Remote(),
				"target_image_name": repo.Name,
				"target_tag":        repo.Tag,
				"policy":            plc.Name(),
				"image":             vol.Image.Reference,
			}).Debug("provider.kubernetes: checking image volume")

			if volumeImageRef.Repository() != eventRepoRef.Repository() {
				log.WithFields(log.Fields{
					"parsed_image_name": volumeImageRef.Remote(),
					"target_image_name": repo.Name,
				}).Debug("provider.kubernetes: image volume reference does not match, ignoring")
				continue
			}

			shouldUpdateVolume, err := plc.ShouldUpdate(volumeImageRef.Tag(), eventRepoRef.Tag())
			if err != nil {
				log.WithFields(log.Fields{
					"error":             err,
					"parsed_image_name": volumeImageRef.Remote(),
					"target_image_name": repo.Name,
					"policy":            plc.Name(),
				}).Error("provider.kubernetes: failed to check whether image volume should be updated")
				continue
			}

			if !shouldUpdateVolume || (alreadyDeployed && volumeImageRef.Tag() == eventRepoRef.Tag()) {
				continue
			}

			setUpdateTime(resource)

			if volumeImageRef.Registry() == image.DefaultRegistryHostname {
				resource.UpdateImageVolume(idx, fmt.Sprintf("%s:%s", volumeImageRef.ShortName(), repo.Tag))
			} else {
				resource.UpdateImageVolume(idx, fmt.Sprintf("%s:%s", volumeImageRef.Repository(), repo.Tag))
			}

			shouldUpdateDeployment = true

			updatePlan.CurrentVersion = volumeImageRef.Tag()
			updatePlan.NewVersion = repo.Tag
			updatePlan.NewDigest = repo.Digest
			updatePlan.Resource = resource
		}
	}

	if schedule, ok := resource.GetAnnotations()[types.KeelInitContainerAnnotation]; ok && schedule == "true" {
		for idx, c := range resource.InitContainers() {
			if !containerFilterFunc(c) {
				continue
			}
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
				"kind":              resource.Kind(),
				"parsed_image_name": containerImageRef.Remote(),
				"target_image_name": repo.Name,
				"target_tag":        repo.Tag,
				"policy":            plc.Name(),
				"image":             c.Image,
			}).Debug("provider.kubernetes: checking image")

			if containerImageRef.Repository() != eventRepoRef.Repository() {
				log.WithFields(log.Fields{
					"parsed_image_name": containerImageRef.Remote(),
					"target_image_name": repo.Name,
				}).Debug("provider.kubernetes: images do not match, ignoring")
				continue
			}

			shouldUpdateContainer, err := plc.ShouldUpdate(containerImageRef.Tag(), eventRepoRef.Tag())
			if err != nil {
				log.WithFields(log.Fields{
					"error":             err,
					"parsed_image_name": containerImageRef.Remote(),
					"target_image_name": repo.Name,
					"policy":            plc.Name(),
				}).Error("provider.kubernetes: failed to check whether init container should be updated")
				continue
			}

			if !shouldUpdateContainer || (alreadyDeployed && containerImageRef.Tag() == eventRepoRef.Tag()) {
				continue
			}

			// updating spec template annotations
			setUpdateTime(resource)

			// updating image
			if containerImageRef.Registry() == image.DefaultRegistryHostname {
				resource.UpdateInitContainer(idx, fmt.Sprintf("%s:%s", containerImageRef.ShortName(), repo.Tag))
			} else {
				resource.UpdateInitContainer(idx, fmt.Sprintf("%s:%s", containerImageRef.Repository(), repo.Tag))
			}

			shouldUpdateDeployment = true

			updatePlan.CurrentVersion = containerImageRef.Tag()
			updatePlan.NewVersion = repo.Tag
			updatePlan.NewDigest = repo.Digest
			updatePlan.Resource = resource
		}
	}
	for idx, c := range resource.Containers() {
		if !containerFilterFunc(c) {
			continue
		}
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
			"kind":              resource.Kind(),
			"parsed_image_name": containerImageRef.Remote(),
			"target_image_name": repo.Name,
			"target_tag":        repo.Tag,
			"policy":            plc.Name(),
			"image":             c.Image,
		}).Debug("provider.kubernetes: checking image")

		if containerImageRef.Repository() != eventRepoRef.Repository() {
			log.WithFields(log.Fields{
				"parsed_image_name": containerImageRef.Remote(),
				"target_image_name": repo.Name,
			}).Debug("provider.kubernetes: images do not match, ignoring")
			continue
		}

		shouldUpdateContainer, err := plc.ShouldUpdate(containerImageRef.Tag(), eventRepoRef.Tag())
		if err != nil {
			log.WithFields(log.Fields{
				"error":             err,
				"parsed_image_name": containerImageRef.Remote(),
				"target_image_name": repo.Name,
				"policy":            plc.Name(),
			}).Error("provider.kubernetes: failed to check whether container should be updated")
			continue
		}

		if !shouldUpdateContainer || (alreadyDeployed && containerImageRef.Tag() == eventRepoRef.Tag()) {
			continue
		}

		// updating spec template annotations
		setUpdateTime(resource)

		// updating image
		if containerImageRef.Registry() == image.DefaultRegistryHostname {
			resource.UpdateContainer(idx, fmt.Sprintf("%s:%s", containerImageRef.ShortName(), repo.Tag))
		} else {
			resource.UpdateContainer(idx, fmt.Sprintf("%s:%s", containerImageRef.Repository(), repo.Tag))
		}

		shouldUpdateDeployment = true

		updatePlan.CurrentVersion = containerImageRef.Tag()
		updatePlan.NewVersion = repo.Tag
		updatePlan.NewDigest = repo.Digest
		updatePlan.Resource = resource
	}

	if shouldUpdateDeployment {
		updatePlan.Image = eventRepoRef.Remote()
	}
	return updatePlan, shouldUpdateDeployment, nil
}

func setUpdateTime(resource *k8s.GenericResource) {
	specAnnotations := resource.GetSpecAnnotations()
	specAnnotations[types.KeelUpdateTimeAnnotation] = time.Now().String()
	resource.SetSpecAnnotations(specAnnotations)
}

func deployedDigests(resource *k8s.GenericResource) map[string]string {
	digests := make(map[string]string)
	if err := json.Unmarshal([]byte(resource.GetAnnotations()[types.KeelDigestsAnnotation]), &digests); err != nil || digests == nil {
		return make(map[string]string)
	}
	return digests
}

func recordDeployedDigest(resource *k8s.GenericResource, plan *UpdatePlan) {
	if plan.Image == "" {
		return
	}
	digests := deployedDigests(resource)
	// Keep only references still present in the workload, so version changes
	// do not leave an ever-growing history in its annotations.
	images := resource.GetImages(func(v1.Container) bool { return true })
	images = append(images, resource.GetInitImages(func(v1.Container) bool { return true })...)
	images = append(images, resource.GetImageVolumeReferences(func(v1.Volume) bool { return true })...)
	present := make(map[string]bool, len(images))
	for _, img := range images {
		if ref, err := image.Parse(img); err == nil {
			present[ref.Remote()] = true
		}
	}
	maps.DeleteFunc(digests, func(ref, _ string) bool { return !present[ref] })
	if plan.NewDigest == "" {
		// A digestless webhook requests a fresh rollout. Its old digest must
		// not suppress the next digest-bearing event for the same image.
		delete(digests, plan.Image)
	} else {
		digests[plan.Image] = plan.NewDigest
	}
	annotations := resource.GetAnnotations()
	if len(digests) == 0 {
		delete(annotations, types.KeelDigestsAnnotation)
	} else {
		encoded, _ := json.Marshal(digests)
		annotations[types.KeelDigestsAnnotation] = string(encoded)
	}
	resource.SetAnnotations(annotations)
}
