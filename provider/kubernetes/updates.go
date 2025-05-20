package kubernetes

import (
	"fmt"
	"time"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/internal/schedule"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	log "github.com/sirupsen/logrus"
)

func checkForUpdate(plc policy.Policy, repo *types.Repository, resource *k8s.GenericResource) (updatePlan *UpdatePlan, shouldUpdateDeployment bool, err error) {
	updatePlan = &UpdatePlan{}

	// Get update schedule
	schedule, err := schedule.ParseUpdateSchedule(resource.GetAnnotations())
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"name":      resource.Name,
			"namespace": resource.Namespace,
		}).Error("provider.kubernetes: failed to parse update schedule")
		return nil, false, err
	}

	// Check if update is allowed based on schedule
	if schedule != nil {
		lastUpdateStr := resource.GetAnnotations()[types.KeelUpdateTimeAnnotation]
		var lastUpdate time.Time
		if lastUpdateStr != "" {
			lastUpdate, err = time.Parse(time.RFC3339, lastUpdateStr)
			if err != nil {
				log.WithFields(log.Fields{
					"error":     err,
					"name":      resource.Name,
					"namespace": resource.Namespace,
				}).Error("provider.kubernetes: failed to parse last update time")
				return nil, false, err
			}
		}

		allowed, err := schedule.IsUpdateAllowed(lastUpdate)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"name":      resource.Name,
				"namespace": resource.Namespace,
			}).Error("provider.kubernetes: failed to check update schedule")
			return nil, false, err
		}

		if !allowed {
			log.WithFields(log.Fields{
				"name":      resource.Name,
				"namespace": resource.Namespace,
			}).Info("provider.kubernetes: update not allowed by schedule")
			return nil, false, nil
		}
	}

	eventRepoRef, err := image.Parse(repo.String())
	if err != nil {
		return
	}

	log.WithFields(log.Fields{
		"name":      resource.Name,
		"namespace": resource.Namespace,
		"kind":      resource.Kind(),
		"policy":    plc.Name(),
	}).Debug("provider.kubernetes.checkVersionedDeployment: keel policy found, checking resource...")
	shouldUpdateDeployment = false

	containerFilterFunc := GetMonitorContainersFromMeta(resource.GetAnnotations(), resource.GetLabels())

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

			if !shouldUpdateContainer {
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

		if !shouldUpdateContainer {
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
		updatePlan.Resource = resource
	}

	return updatePlan, shouldUpdateDeployment, nil
}

func setUpdateTime(resource *k8s.GenericResource) {
	specAnnotations := resource.GetSpecAnnotations()
	specAnnotations[types.KeelUpdateTimeAnnotation] = time.Now().String()
	resource.SetSpecAnnotations(specAnnotations)
}

func (p *Provider) checkForUpdate(resource k8s.GenericResource, repo *types.Repository) error {
	// Get update schedule
	schedule, err := schedule.ParseUpdateSchedule(resource.GetAnnotations())
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"name":      resource.Name,
			"namespace": resource.Namespace,
		}).Error("provider.kubernetes: failed to parse update schedule")
		return err
	}

	// Check if update is allowed based on schedule
	if schedule != nil {
		lastUpdateStr := resource.GetAnnotations()[types.KeelUpdateTimeAnnotation]
		var lastUpdate time.Time
		if lastUpdateStr != "" {
			lastUpdate, err = time.Parse(time.RFC3339, lastUpdateStr)
			if err != nil {
				log.WithFields(log.Fields{
					"error":     err,
					"name":      resource.Name,
					"namespace": resource.Namespace,
				}).Error("provider.kubernetes: failed to parse last update time")
				return err
			}
		}

		allowed, err := schedule.IsUpdateAllowed(lastUpdate)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"name":      resource.Name,
				"namespace": resource.Namespace,
			}).Error("provider.kubernetes: failed to check update schedule")
			return err
		}

		if !allowed {
			log.WithFields(log.Fields{
				"name":      resource.Name,
				"namespace": resource.Namespace,
			}).Info("provider.kubernetes: update not allowed by schedule")
			return nil
		}
	}

	// Get policy from labels/annotations
	plc := policy.GetPolicyFromLabelsOrAnnotations(resource.GetLabels(), resource.GetAnnotations())
	if plc.Type() == types.PolicyTypeNone {
		return nil
	}

	// Check for updates
	updatePlan, shouldUpdate, err := checkForUpdate(plc, repo, &resource)
	if err != nil {
		return err
	}

	if !shouldUpdate {
		return nil
	}

	// Submit update plan
	updated, err := p.updateDeployments([]*UpdatePlan{updatePlan})
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"name":      resource.Name,
			"namespace": resource.Namespace,
		}).Error("provider.kubernetes: failed to submit update plan")
		return err
	}

	if len(updated) == 0 {
		log.WithFields(log.Fields{
			"name":      resource.Name,
			"namespace": resource.Namespace,
		}).Info("provider.kubernetes: no resources were updated")
	}

	return nil
}
