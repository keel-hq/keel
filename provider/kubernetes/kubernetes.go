package kubernetes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rusenask/cron"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"github.com/keel-hq/keel/util/policies"
	"github.com/keel-hq/keel/util/version"

	log "github.com/sirupsen/logrus"
)

var kubernetesVersionedUpdatesCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "kubernetes_versioned_updates_total",
		Help: "How many versioned deployments were updated, partitioned by deployment name.",
	},
	[]string{"deployment"},
)

var kubernetesUnversionedUpdatesCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "kubernetes_unversioned_updates_total",
		Help: "How many unversioned deployments were updated, partitioned by deployment name.",
	},
	[]string{"deployment"},
)

func init() {
	prometheus.MustRegister(kubernetesVersionedUpdatesCounter)
	prometheus.MustRegister(kubernetesUnversionedUpdatesCounter)
}

// ProviderName - provider name
const ProviderName = "kubernetes"

var versionreg = regexp.MustCompile(`:[^:]*$`)

// annotation used to specify which image to force pull
const forceUpdateImageAnnotation = "keel.sh/update-image"

// GenericResourceCache an interface for generic resource cache.
type GenericResourceCache interface {
	// Values returns a copy of the contents of the cache.
	// The slice and its contents should be treated as read-only.
	Values() []*k8s.GenericResource

	// Register registers ch to receive a value when Notify is called.
	Register(chan int, int)
}

// UpdatePlan - deployment update plan
type UpdatePlan struct {
	// Updated deployment version
	// Deployment v1beta1.Deployment
	Resource *k8s.GenericResource

	// Current (last seen cluster version)
	CurrentVersion string
	// New version that's already in the deployment
	NewVersion string
}

// Provider - kubernetes provider for auto update
type Provider struct {
	implementer Implementer

	sender notification.Sender

	approvalManager approvals.Manager

	cache GenericResourceCache

	events chan *types.Event
	stop   chan struct{}
}

// NewProvider - create new kubernetes based provider
func NewProvider(implementer Implementer, sender notification.Sender, approvalManager approvals.Manager, cache GenericResourceCache) (*Provider, error) {
	return &Provider{
		implementer:     implementer,
		cache:           cache,
		approvalManager: approvalManager,
		events:          make(chan *types.Event, 100),
		stop:            make(chan struct{}),
		sender:          sender,
	}, nil
}

// Submit - submit event to provider
func (p *Provider) Submit(event types.Event) error {
	p.events <- &event
	return nil
}

// GetName - get provider name
func (p *Provider) GetName() string {
	return ProviderName
}

// Start - starts kubernetes provider, waits for events
func (p *Provider) Start() error {
	return p.startInternal()
}

// Stop - stops kubernetes provider
func (p *Provider) Stop() {
	close(p.stop)
}

// TrackedImages returns a list of tracked images.
func (p *Provider) TrackedImages() ([]*types.TrackedImage, error) {
	var trackedImages []*types.TrackedImage

	for _, gr := range p.cache.Values() {
		labels := gr.GetLabels()

		// ignoring unlabelled deployments
		policy := policies.GetPolicy(labels)
		if policy == types.PolicyTypeNone {
			continue
		}

		annotations := gr.GetAnnotations()
		schedule, ok := annotations[types.KeelPollScheduleAnnotation]
		if ok {
			_, err := cron.Parse(schedule)
			if err != nil {
				log.WithFields(log.Fields{
					"error":     err,
					"schedule":  schedule,
					"name":      gr.Name,
					"namespace": gr.Namespace,
				}).Error("provider.kubernetes: failed to parse poll schedule, setting default schedule")
				schedule = types.KeelPollDefaultSchedule
			}
		} else {
			schedule = types.KeelPollDefaultSchedule
		}

		// trigger type, we only care for "poll" type triggers
		trigger := policies.GetTriggerPolicy(labels)

		// secrets := getImagePullSecrets(&deployment)

		secrets := gr.GetImagePullSecrets()

		// images := getImages(&deployment)
		images := gr.GetImages()
		for _, img := range images {
			ref, err := image.Parse(img)
			if err != nil {
				log.WithFields(log.Fields{
					"error":     err,
					"image":     img,
					"namespace": gr.Namespace,
					"name":      gr.Name,
				}).Error("provider.kubernetes: failed to parse image")
				continue
			}
			trackedImages = append(trackedImages, &types.TrackedImage{
				Image:        ref,
				PollSchedule: schedule,
				Trigger:      trigger,
				Provider:     ProviderName,
				Namespace:    gr.Namespace,
				Secrets:      secrets,
			})
		}
	}

	return trackedImages, nil
}

// TrackedImages - get tracked images
// func (p *Provider) TrackedImages() ([]*types.TrackedImage, error) {
// 	var trackedImages []*types.TrackedImage

// 	deploymentLists, err := p.deployments()
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, deploymentList := range deploymentLists {
// 		for _, deployment := range deploymentList.Items {
// 			labels := deployment.GetLabels()

// 			// ignoring unlabelled deployments
// 			policy := policies.GetPolicy(labels)
// 			if policy == types.PolicyTypeNone {
// 				continue
// 			}

// 			annotations := deployment.GetAnnotations()
// 			schedule, ok := annotations[types.KeelPollScheduleAnnotation]
// 			if ok {
// 				_, err := cron.Parse(schedule)
// 				if err != nil {
// 					log.WithFields(log.Fields{
// 						"error":      err,
// 						"schedule":   schedule,
// 						"deployment": deployment.Name,
// 						"namespace":  deployment.Namespace,
// 					}).Error("provider.kubernetes: failed to parse poll schedule, setting default schedule")
// 					schedule = types.KeelPollDefaultSchedule
// 				}
// 			} else {
// 				schedule = types.KeelPollDefaultSchedule
// 			}

// 			// trigger type, we only care for "poll" type triggers
// 			trigger := policies.GetTriggerPolicy(labels)

// 			secrets := getImagePullSecrets(&deployment)

// 			images := getImages(&deployment)
// 			for _, img := range images {
// 				ref, err := image.Parse(img)
// 				if err != nil {
// 					log.WithFields(log.Fields{
// 						"error":     err,
// 						"image":     img,
// 						"namespace": deployment.Namespace,
// 						"name":      deployment.Name,
// 					}).Error("provider.kubernetes: failed to parse image")
// 					continue
// 				}
// 				trackedImages = append(trackedImages, &types.TrackedImage{
// 					Image:        ref,
// 					PollSchedule: schedule,
// 					Trigger:      trigger,
// 					Provider:     ProviderName,
// 					Namespace:    deployment.Namespace,
// 					Secrets:      secrets,
// 				})
// 			}
// 		}
// 	}

// 	return trackedImages, nil
// }

func (p *Provider) startInternal() error {
	for {
		select {
		case event := <-p.events:
			log.WithFields(log.Fields{
				"repository": event.Repository.Name,
				"tag":        event.Repository.Tag,
				"registry":   event.Repository.Host,
			}).Info("provider.kubernetes: processing event")
			_, err := p.processEvent(event)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"image": event.Repository.Name,
					"tag":   event.Repository.Tag,
				}).Error("provider.kubernetes: failed to process event")
			}
		case <-p.stop:
			log.Info("provider.kubernetes: got shutdown signal, stopping...")
			return nil
		}
	}
}

// func (p *Provider) processEvent(event *types.Event) (updated []*v1beta1.Deployment, err error) {
func (p *Provider) processEvent(event *types.Event) (updated []*k8s.GenericResource, err error) {
	plans, err := p.createUpdatePlans(&event.Repository)
	if err != nil {
		return nil, err
	}

	if len(plans) == 0 {
		log.WithFields(log.Fields{
			"image": event.Repository.Name,
			"tag":   event.Repository.Tag,
		}).Info("provider.kubernetes: no plans for deployment updates found for this event")
		return
	}

	approvedPlans := p.checkForApprovals(event, plans)

	return p.updateDeployments(approvedPlans)
}

// func (p *Provider) updateDeployments(deployments []v1beta1.Deployment) (updated []*v1beta1.Deployment, err error) {
func (p *Provider) updateDeployments(plans []*UpdatePlan) (updated []*k8s.GenericResource, err error) {
	for _, plan := range plans {
		resource := plan.Resource

		annotations := resource.GetAnnotations()

		notificationChannels := types.ParseEventNotificationChannels(annotations)
		// reset := checkForReset(deployment)

		p.sender.Send(types.EventNotification{
			Name:      "preparing to update resource",
			Message:   fmt.Sprintf("Preparing to update %s %s/%s %s->%s (%s)", resource.Kind(), resource.Namespace, resource.Name, plan.CurrentVersion, plan.NewVersion, strings.Join(resource.GetImages(), ", ")),
			CreatedAt: time.Now(),
			Type:      types.NotificationPreDeploymentUpdate,
			Level:     types.LevelDebug,
			Channels:  notificationChannels,
		})

		var err error

		// if reset {
		// annotations["keel.sh/update-time"] = time.Now().String()
		// }
		// force update, terminating all pods
		// err = p.forceUpdate(&deployment)
		// kubernetesUnversionedUpdatesCounter.With(prometheus.Labels{"deployment": fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)}).Inc()
		// } else {
		// regular update
		// deployment.Annotations["kubernetes.io/change-cause"] = fmt.Sprintf("keel automated update, version %s -> %s", plan.CurrentVersion, plan.NewVersion)
		annotations["kubernetes.io/change-cause"] = fmt.Sprintf("keel automated update, version %s -> %s", plan.CurrentVersion, plan.NewVersion)

		resource.SetAnnotations(annotations)

		err = p.implementer.Update(resource)
		kubernetesVersionedUpdatesCounter.With(prometheus.Labels{"deployment": fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)}).Inc()
		// }
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"namespace":  resource.Namespace,
				"deployment": resource.Name,
				"update":     fmt.Sprintf("%s->%s", plan.CurrentVersion, plan.NewVersion),
			}).Error("provider.kubernetes: got error while update deployment")

			p.sender.Send(types.EventNotification{
				Name:      "update resource",
				Message:   fmt.Sprintf("%s %s/%s update %s->%s failed, error: %s", resource.Kind(), resource.Namespace, resource.Name, plan.CurrentVersion, plan.NewVersion, err),
				CreatedAt: time.Now(),
				Type:      types.NotificationDeploymentUpdate,
				Level:     types.LevelError,
				Channels:  notificationChannels,
			})

			continue
		}

		p.sender.Send(types.EventNotification{
			Name:      "update resource",
			Message:   fmt.Sprintf("Successfully updated %s %s/%s %s->%s (%s)", resource.Kind(), resource.Namespace, resource.Name, plan.CurrentVersion, plan.NewVersion, strings.Join(resource.GetImages(), ", ")),
			CreatedAt: time.Now(),
			Type:      types.NotificationDeploymentUpdate,
			Level:     types.LevelSuccess,
			Channels:  notificationChannels,
		})

		log.WithFields(log.Fields{
			"name":      resource.Name,
			"namespace": resource.Namespace,
		}).Info("provider.kubernetes: deployment updated")
		updated = append(updated, resource)
	}

	return
}

func getImages(deployment *v1beta1.Deployment) []string {
	var images []string
	for _, c := range deployment.Spec.Template.Spec.Containers {
		images = append(images, c.Image)
	}

	return images
}

func getImagePullSecrets(deployment *v1beta1.Deployment) []string {
	var secrets []string
	for _, s := range deployment.Spec.Template.Spec.ImagePullSecrets {
		secrets = append(secrets, s.Name)
	}
	return secrets
}

func getDesiredImage(delta map[string]string, currentImage string) (string, error) {
	currentRef, err := image.Parse(currentImage)
	if err != nil {
		return "", err
	}
	for repository, tag := range delta {
		if repository == currentRef.Repository() {
			ref, err := image.Parse(repository)
			if err != nil {
				return "", err
			}

			// updating image
			if ref.Registry() == image.DefaultRegistryHostname {
				return fmt.Sprintf("%s:%s", ref.ShortName(), tag), nil
			}
			return fmt.Sprintf("%s:%s", ref.Repository(), tag), nil
		}
	}
	return "", fmt.Errorf("image %s not found in deltas", currentImage)
}

// checkForReset returns delta to apply after setting image
func checkForReset(deployment v1beta1.Deployment) bool {
	reset := false
	annotations := deployment.GetAnnotations()
	for _, c := range deployment.Spec.Template.Spec.Containers {
		if shouldPullImage(annotations, c.Image) {
			reset = true
		}
	}
	return reset
}

// getDeployment - helper function to get specific deployment
// func (p *Provider) getDeployment(namespace, name string) (*v1beta1.Deployment, error) {
// 	return p.implementer.Deployment(namespace, name)
// }

// createUpdatePlans - impacted deployments by changed repository
func (p *Provider) createUpdatePlans(repo *types.Repository) ([]*UpdatePlan, error) {

	// deploymentLists, err := p.deployments()
	// if err != nil {
	// 	log.WithFields(log.Fields{
	// 		"error": err,
	// 	}).Error("provider.kubernetes: failed to get deployment lists")
	// 	return nil, err
	// }

	impacted := []*UpdatePlan{}

	// for _, deploymentList := range deploymentLists {
	for _, resource := range p.cache.Values() {

		labels := resource.GetLabels()

		policy := policies.GetPolicy(labels)
		if policy == types.PolicyTypeNone {
			// skip
			continue
		}

		// annotation cleanup
		// annotations := gr.GetAnnotations()
		// delete(annotations, forceUpdateImageAnnotation)
		// deployment.SetAnnotations(annotations)

		newVersion, err := version.GetVersion(repo.Tag)
		if err != nil {
			// failed to get new version tag
			if policy == types.PolicyTypeForce {
				updated, shouldUpdateDeployment, err := p.checkUnversionedDeployment(policy, repo, resource)
				if err != nil {
					log.WithFields(log.Fields{
						"error":     err,
						"name":      resource.Name,
						"namespace": resource.Namespace,
						"kind":      resource.Kind(),
					}).Error("provider.kubernetes: got error while checking unversioned resource")
					continue
				}

				if shouldUpdateDeployment {
					impacted = append(impacted, updated)
				}

				// success, unversioned deployment marked for update
				continue
			}

			log.WithFields(log.Fields{
				"error":          err,
				"repository_tag": repo.Tag,
				"deployment":     resource.Name,
				"namespace":      resource.Namespace,
				"kind":           resource.Kind(),
				"policy":         policy,
			}).Warn("provider.kubernetes: got error while parsing repository tag")
			continue
		}

		updated, shouldUpdateDeployment, err := p.checkVersionedDeployment(newVersion, policy, repo, resource)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"deployment": resource.Name,
				"kind":       resource.Kind(),
				"namespace":  resource.Namespace,
			}).Error("provider.kubernetes: got error while checking versioned resource")
			continue
		}

		if shouldUpdateDeployment {
			impacted = append(impacted, updated)
		}
	}
	// }

	return impacted, nil
}

func (p *Provider) namespaces() (*v1.NamespaceList, error) {
	return p.implementer.Namespaces()
}

// deployments - gets all deployments
// func (p *Provider) deployments() ([]*v1beta1.DeploymentList, error) {
// 	deployments := []*v1beta1.DeploymentList{}

// 	n, err := p.namespaces()
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, n := range n.Items {
// 		l, err := p.implementer.Deployments(n.GetName())
// 		if err != nil {
// 			log.WithFields(log.Fields{
// 				"error":     err,
// 				"namespace": n.GetName(),
// 			}).Error("provider.kubernetes: failed to list deployments")
// 			continue
// 		}
// 		deployments = append(deployments, l)
// 	}

// 	return deployments, nil
// }
