package kubernetes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rusenask/cron"

	// "k8s.io/api/core/v1"
	"k8s.io/api/core/v1"

	"k8s.io/api/extensions/v1beta1"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"github.com/keel-hq/keel/util/policies"
	"github.com/keel-hq/keel/util/version"

	log "github.com/sirupsen/logrus"
)

// ProviderName - provider name
const ProviderName = "kubernetes"

var versionreg = regexp.MustCompile(`:[^:]*$`)

// annotation used to specify which image to force pull
const forceUpdateImageAnnotation = "keel.sh/update-image"

// UpdatePlan - deployment update plan
type UpdatePlan struct {
	// Updated deployment version
	Deployment v1beta1.Deployment
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

	events chan *types.Event
	stop   chan struct{}
}

// NewProvider - create new kubernetes based provider
func NewProvider(implementer Implementer, sender notification.Sender, approvalManager approvals.Manager) (*Provider, error) {
	return &Provider{
		implementer:     implementer,
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

// TrackedImages - get tracked images
func (p *Provider) TrackedImages() ([]*types.TrackedImage, error) {
	var trackedImages []*types.TrackedImage

	deploymentLists, err := p.deployments()
	if err != nil {
		return nil, err
	}

	for _, deploymentList := range deploymentLists {
		for _, deployment := range deploymentList.Items {
			labels := deployment.GetLabels()

			// ignoring unlabelled deployments
			policy := policies.GetPolicy(labels)
			if policy == types.PolicyTypeNone {
				continue
			}

			annotations := deployment.GetAnnotations()
			schedule, ok := annotations[types.KeelPollScheduleAnnotation]
			if ok {
				_, err := cron.Parse(schedule)
				if err != nil {
					log.WithFields(log.Fields{
						"error":      err,
						"schedule":   schedule,
						"deployment": deployment.Name,
						"namespace":  deployment.Namespace,
					}).Error("provider.kubernetes: failed to parse poll schedule, setting default schedule")
					schedule = types.KeelPollDefaultSchedule
				}
			} else {
				schedule = types.KeelPollDefaultSchedule
			}

			// trigger type, we only care for "poll" type triggers
			trigger := policies.GetTriggerPolicy(labels)

			secrets := getImagePullSecrets(&deployment)

			images := getImages(&deployment)
			for _, img := range images {
				ref, err := image.Parse(img)
				if err != nil {
					log.WithFields(log.Fields{
						"error":     err,
						"image":     img,
						"namespace": deployment.Namespace,
						"name":      deployment.Name,
					}).Error("provider.kubernetes: failed to parse image")
					continue
				}
				trackedImages = append(trackedImages, &types.TrackedImage{
					Image:        ref,
					PollSchedule: schedule,
					Trigger:      trigger,
					Provider:     ProviderName,
					Namespace:    deployment.Namespace,
					Secrets:      secrets,
				})
			}
		}
	}

	return trackedImages, nil
}

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

func (p *Provider) processEvent(event *types.Event) (updated []*v1beta1.Deployment, err error) {
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
func (p *Provider) updateDeployments(plans []*UpdatePlan) (updated []*v1beta1.Deployment, err error) {
	// for _, deployment := range plans {
	for _, plan := range plans {
		deployment := plan.Deployment
		notificationChannels := types.ParseEventNotificationChannels(deployment.Annotations)
		reset := checkForReset(deployment)

		p.sender.Send(types.EventNotification{
			Name:      "preparing to update deployment",
			Message:   fmt.Sprintf("Preparing to update deployment %s/%s %s->%s (%s)", deployment.Namespace, deployment.Name, plan.CurrentVersion, plan.NewVersion, strings.Join(getImages(&deployment), ", ")),
			CreatedAt: time.Now(),
			Type:      types.NotificationPreDeploymentUpdate,
			Level:     types.LevelDebug,
			Channels:  notificationChannels,
		})

		var err error

		if reset {
			// force update, terminating all pods
			err = p.forceUpdate(&deployment)
		} else {
			// regular update
			err = p.implementer.Update(&deployment)
		}
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"namespace":  deployment.Namespace,
				"deployment": deployment.Name,
				"update":     fmt.Sprintf("%s->%s", plan.CurrentVersion, plan.NewVersion),
			}).Error("provider.kubernetes: got error while update deployment")

			p.sender.Send(types.EventNotification{
				Name:      "update deployment",
				Message:   fmt.Sprintf("Deployment %s/%s update %s->%s failed, error: %s", deployment.Namespace, deployment.Name, plan.CurrentVersion, plan.NewVersion, err),
				CreatedAt: time.Now(),
				Type:      types.NotificationDeploymentUpdate,
				Level:     types.LevelError,
				Channels:  notificationChannels,
			})

			continue
		}

		p.sender.Send(types.EventNotification{
			Name:      "update deployment",
			Message:   fmt.Sprintf("Successfully updated deployment %s/%s %s->%s (%s)", deployment.Namespace, deployment.Name, plan.CurrentVersion, plan.NewVersion, strings.Join(getImages(&deployment), ", ")),
			CreatedAt: time.Now(),
			Type:      types.NotificationDeploymentUpdate,
			Level:     types.LevelSuccess,
			Channels:  notificationChannels,
		})

		log.WithFields(log.Fields{
			"name":      deployment.Name,
			"namespace": deployment.Namespace,
		}).Info("provider.kubernetes: deployment updated")
		updated = append(updated, &deployment)
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
func (p *Provider) getDeployment(namespace, name string) (*v1beta1.Deployment, error) {
	return p.implementer.Deployment(namespace, name)
}

// createUpdatePlans - impacted deployments by changed repository
// func (p *Provider) impactedDeployments(repo *types.Repository) ([]v1beta1.Deployment, error) {
func (p *Provider) createUpdatePlans(repo *types.Repository) ([]*UpdatePlan, error) {

	deploymentLists, err := p.deployments()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("provider.kubernetes: failed to get deployment lists")
		return nil, err
	}

	// impacted := []v1beta1.Deployment{}
	impacted := []*UpdatePlan{}

	for _, deploymentList := range deploymentLists {
		for _, deployment := range deploymentList.Items {

			labels := deployment.GetLabels()

			policy := policies.GetPolicy(labels)
			if policy == types.PolicyTypeNone {
				// skip
				continue
			}

			// annotation cleanup
			annotations := deployment.GetAnnotations()
			delete(annotations, forceUpdateImageAnnotation)
			deployment.SetAnnotations(annotations)

			newVersion, err := version.GetVersion(repo.Tag)
			if err != nil {
				// failed to get new version tag
				if policy == types.PolicyTypeForce {
					updated, shouldUpdateDeployment, err := p.checkUnversionedDeployment(policy, repo, deployment)
					if err != nil {
						log.WithFields(log.Fields{
							"error":      err,
							"deployment": deployment.Name,
							"namespace":  deployment.Namespace,
						}).Error("provider.kubernetes: got error while checking unversioned deployment")
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
					"deployment":     deployment.Name,
					"namespace":      deployment.Namespace,
					"policy":         policy,
				}).Warn("provider.kubernetes: got error while parsing repository tag")
				continue
			}

			updated, shouldUpdateDeployment, err := p.checkVersionedDeployment(newVersion, policy, repo, deployment)
			if err != nil {
				log.WithFields(log.Fields{
					"error":      err,
					"deployment": deployment.Name,
					"namespace":  deployment.Namespace,
				}).Error("provider.kubernetes: got error while checking versioned deployment")
				continue
			}

			if shouldUpdateDeployment {
				impacted = append(impacted, updated)
			}
		}
	}

	return impacted, nil
}

func (p *Provider) namespaces() (*v1.NamespaceList, error) {
	return p.implementer.Namespaces()
}

// deployments - gets all deployments
func (p *Provider) deployments() ([]*v1beta1.DeploymentList, error) {
	deployments := []*v1beta1.DeploymentList{}

	n, err := p.namespaces()
	if err != nil {
		return nil, err
	}

	for _, n := range n.Items {
		l, err := p.implementer.Deployments(n.GetName())
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"namespace": n.GetName(),
			}).Error("provider.kubernetes: failed to list deployments")
			continue
		}
		deployments = append(deployments, l)
	}

	return deployments, nil
}
