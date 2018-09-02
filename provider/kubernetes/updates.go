package kubernetes

import (
	"fmt"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	log "github.com/sirupsen/logrus"
)

func checkForUpdate(plc policy.Policy, repo *types.Repository, resource *k8s.GenericResource) (updatePlan *UpdatePlan, shouldUpdateDeployment bool, err error) {
	updatePlan = &UpdatePlan{}

	eventRepoRef, err := image.Parse(repo.String())
	if err != nil {
		return
	}

	log.WithFields(log.Fields{
		"name":      resource.Name,
		"namespace": resource.Namespace,
		"kind":      resource.Kind(),
		"policy":    plc.Name(),
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
			"kind":              resource.Kind(),
			"parsed_image_name": containerImageRef.Remote(),
			"target_image_name": repo.Name,
			"target_tag":        repo.Tag,
			"policy":            plc.Name(),
			"image":             c.Image,
		}).Info("provider.kubernetes: checking image")

		if containerImageRef.Repository() != eventRepoRef.Repository() {
			log.WithFields(log.Fields{
				"parsed_image_name": containerImageRef.Remote(),
				"target_image_name": repo.Name,
			}).Info("provider.kubernetes: images do not match, ignoring")
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

		if !shouldUpdateContainer {
			continue
		}

		// if poll trigger is used, also checking for matching versions
		// if _, ok := annotations[types.KeelPollScheduleAnnotation]; ok {
		// 	if repo.Tag != containerImageRef.Tag() {
		// 		fmt.Printf("tags different, not updating (%s != %s) \n", eventRepoRef.Tag(), containerImageRef.Tag())
		// 		continue
		// 	}
		// }

		// updating annotations
		// if matchTag, _ := labels[types.KeelForceTagMatchLabel]; matchTag == "true" {
		// 	if containerImageRef.Tag() != eventRepoRef.Tag() {
		// 		continue
		// 	}
		// }

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
		updatePlan.Resource = resource

		log.WithFields(log.Fields{
			"parsed_image":     containerImageRef.Remote(),
			"raw_image_name":   c.Image,
			"target_image":     repo.Name,
			"target_image_tag": repo.Tag,
			"policy":           plc.Name(),
		}).Info("provider.kubernetes: impacted deployment container found")

	}

	return updatePlan, shouldUpdateDeployment, nil
}
