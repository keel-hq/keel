package kubernetes

import (
	"fmt"
	"regexp"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/version"

	log "github.com/Sirupsen/logrus"
)

const ProviderName = "kubernetes"

var versionreg = regexp.MustCompile(`:[^:]*$`)

// Provider - kubernetes provider for auto update
type Provider struct {
	implementer Implementer

	events chan *types.Event
	stop   chan struct{}
}

// NewProvider - create new kubernetes based provider
func NewProvider(implementer Implementer) (*Provider, error) {
	return &Provider{
		implementer: implementer,
		events:      make(chan *types.Event, 100),
		stop:        make(chan struct{}),
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

func (p *Provider) startInternal() error {
	for {
		select {
		case event := <-p.events:
			log.WithFields(log.Fields{
				"repository": event.Repository.Name,
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
	impacted, err := p.impactedDeployments(&event.Repository)
	if err != nil {
		return nil, err
	}

	if len(impacted) == 0 {
		log.WithFields(log.Fields{
			"image": event.Repository.Name,
			"tag":   event.Repository.Tag,
		}).Info("provider.kubernetes: no impacted deployments found for this event")
		return
	}

	updated, err = p.updateDeployments(impacted)

	return
}

func (p *Provider) updateDeployments(deployments []*v1beta1.Deployment) (updated []*v1beta1.Deployment, err error) {
	for _, deployment := range deployments {
		// _, err := p.client.Extensions().Deployments(deployment.Namespace).Update(deployment)
		err := p.implementer.Update(deployment)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"namespace":  deployment.Namespace,
				"deployment": deployment.Name,
			}).Error("provider.kubernetes: got error while update deployment")
			continue
		}
		updated = append(updated, deployment)
	}

	return
}

// getDeployment - helper function to get specific deployment
func (p *Provider) getDeployment(namespace, name string) (*v1beta1.Deployment, error) {
	// dep := p.client.Extensions().Deployments(namespace)
	// return dep.Get(name, meta_v1.GetOptions{})
	return p.implementer.Deployment(namespace, name)
}

// gets impacted deployments by changed repository
func (p *Provider) impactedDeployments(repo *types.Repository) ([]*v1beta1.Deployment, error) {
	newVersion, err := version.GetVersion(repo.Tag)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version from repository tag, error: %s", err)
	}

	deploymentLists, err := p.deployments()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("provider.kubernetes: failed to get deployment lists")
		return nil, err
	}

	impacted := []*v1beta1.Deployment{}

	for _, deploymentList := range deploymentLists {
		for _, deployment := range deploymentList.Items {
			labels := deployment.GetLabels()
			policyStr, ok := labels[types.KeelPolicyLabel]
			// if no policy is set - skipping this deployment
			if !ok {
				continue
			}
			policy := types.ParsePolicy(policyStr)

			log.WithFields(log.Fields{
				"labels":    labels,
				"name":      deployment.Name,
				"namespace": deployment.Namespace,
				"policy":    policy,
			}).Info("provider.kubernetes: keel policy found, checking deployment...")

			for idx, c := range deployment.Spec.Template.Spec.Containers {
				// Remove version if any
				containerImageName := versionreg.ReplaceAllString(c.Image, "")

				log.WithFields(log.Fields{
					"name":              deployment.Name,
					"namespace":         deployment.Namespace,
					"parsed_image_name": containerImageName,
					"target_image_name": repo.Name,
					"target_tag":        repo.Tag,
					"policy":            policy,
					"image":             c.Image,
				}).Info("provider.kubernetes: checking image")

				if containerImageName != repo.Name {
					continue
				}

				currentVersion, err := version.GetVersionFromImageName(c.Image)
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

				shouldUpdate, err := version.ShouldUpdate(currentVersion, newVersion, policy)
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
					"should_update":   shouldUpdate,
				}).Info("provider.kubernetes: checked version, deciding whether to update")

				if shouldUpdate {
					// updating image
					c.Image = fmt.Sprintf("%s:%s", containerImageName, newVersion.String())
					deployment.Spec.Template.Spec.Containers[idx] = c
					impacted = append(impacted, &deployment)
					log.WithFields(log.Fields{
						"parsed_image":     containerImageName,
						"raw_image_name":   c.Image,
						"target_image":     repo.Name,
						"target_image_tag": repo.Tag,
						"policy":           policy,
					}).Info("provider.kubernetes: impacted deployment container found")
				}
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
