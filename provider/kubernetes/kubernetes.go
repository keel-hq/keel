package kubernetes

import (
	"fmt"
	"regexp"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/rusenask/keel/types"
	// "github.com/rusenask/keel/util/hash"
	// "github.com/rusenask/keel/util/image"
	// "github.com/rusenask/keel/util/policies"
	"github.com/rusenask/keel/util/version"

	log "github.com/Sirupsen/logrus"
)

// ProviderName - provider name
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

	return p.updateDeployments(impacted)
}

func (p *Provider) updateDeployments(deployments []v1beta1.Deployment) (updated []*v1beta1.Deployment, err error) {
	for _, deployment := range deployments {
		err := p.implementer.Update(&deployment)
		if err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"namespace":  deployment.Namespace,
				"deployment": deployment.Name,
			}).Error("provider.kubernetes: got error while update deployment")
			continue
		}
		log.WithFields(log.Fields{
			"name":      deployment.Name,
			"namespace": deployment.Namespace,
		}).Info("provider.kubernetes: deployment updated")
		updated = append(updated, &deployment)
	}

	return
}

// getDeployment - helper function to get specific deployment
func (p *Provider) getDeployment(namespace, name string) (*v1beta1.Deployment, error) {
	return p.implementer.Deployment(namespace, name)
}

// gets impacted deployments by changed repository
func (p *Provider) impactedDeployments(repo *types.Repository) ([]v1beta1.Deployment, error) {
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

	impacted := []v1beta1.Deployment{}

	for _, deploymentList := range deploymentLists {
		for _, deployment := range deploymentList.Items {
			// labels := deployment.GetLabels()

			// policy := policies.GetPolicy(labels)
			// if policy == types.PolicyTypeNone {
			// 	continue
			// }

			// log.WithFields(log.Fields{
			// 	"labels":    labels,
			// 	"name":      deployment.Name,
			// 	"namespace": deployment.Namespace,
			// 	"policy":    policy,
			// }).Info("provider.kubernetes: keel policy found, checking deployment...")

			// shouldUpdateDeployment := false

			// for idx, c := range deployment.Spec.Template.Spec.Containers {
			// 	// Remove version if any
			// 	// containerImageName := versionreg.ReplaceAllString(c.Image, "")

			// 	ref, err := image.Parse(c.Image)
			// 	if err != nil {
			// 		log.WithFields(log.Fields{
			// 			"error":      err,
			// 			"image_name": c.Image,
			// 		}).Error("provider.kubernetes: failed to parse image name")
			// 		continue
			// 	}

			// 	log.WithFields(log.Fields{
			// 		"name":              deployment.Name,
			// 		"namespace":         deployment.Namespace,
			// 		"parsed_image_name": ref.Remote(),
			// 		"target_image_name": repo.Name,
			// 		"target_tag":        repo.Tag,
			// 		"policy":            policy,
			// 		"image":             c.Image,
			// 	}).Info("provider.kubernetes: checking image")

			// 	if ref.Remote() != repo.Name {
			// 		log.WithFields(log.Fields{
			// 			"parsed_image_name": ref.Remote(),
			// 			"target_image_name": repo.Name,
			// 		}).Info("provider.kubernetes: images do not match, ignoring")
			// 		continue
			// 	}

			// 	currentVersion, err := version.GetVersionFromImageName(c.Image)
			// 	if err != nil {
			// 		log.WithFields(log.Fields{
			// 			"error":       err,
			// 			"image_name":  c.Image,
			// 			"keel_policy": policy,
			// 		}).Error("provider.kubernetes: failed to get image version, is it tagged as semver?")
			// 		continue
			// 	}

			// 	log.WithFields(log.Fields{
			// 		"labels":          labels,
			// 		"name":            deployment.Name,
			// 		"namespace":       deployment.Namespace,
			// 		"image":           c.Image,
			// 		"current_version": currentVersion.String(),
			// 		"policy":          policy,
			// 	}).Info("provider.kubernetes: current image version")

			// 	shouldUpdateContainer, err := version.ShouldUpdate(currentVersion, newVersion, policy)
			// 	if err != nil {
			// 		log.WithFields(log.Fields{
			// 			"error":           err,
			// 			"new_version":     newVersion.String(),
			// 			"current_version": currentVersion.String(),
			// 			"keel_policy":     policy,
			// 		}).Error("provider.kubernetes: got error while checking whether deployment should be updated")
			// 		continue
			// 	}

			// 	log.WithFields(log.Fields{
			// 		"labels":          labels,
			// 		"name":            deployment.Name,
			// 		"namespace":       deployment.Namespace,
			// 		"image":           c.Image,
			// 		"current_version": currentVersion.String(),
			// 		"new_version":     newVersion.String(),
			// 		"policy":          policy,
			// 		"should_update":   shouldUpdateContainer,
			// 	}).Info("provider.kubernetes: checked version, deciding whether to update")

			// 	if shouldUpdateContainer {
			// 		// updating image
			// 		if ref.Registry() == image.DefaultRegistryHostname {
			// 			c.Image = fmt.Sprintf("%s:%s", ref.ShortName(), newVersion.String())
			// 		} else {
			// 			c.Image = fmt.Sprintf("%s:%s", ref.Remote(), newVersion.String())
			// 		}

			// 		deployment.Spec.Template.Spec.Containers[idx] = c
			// 		// marking this deployment for update
			// 		shouldUpdateDeployment = true

			// 		// updating digest if available
			// 		if repo.Digest != "" {

			// 			labels[types.KeelDigestLabel] = hash.GetShort(repo.Digest)
			// 		}

			// 		log.WithFields(log.Fields{
			// 			"parsed_image":     ref.Remote(),
			// 			"raw_image_name":   c.Image,
			// 			"target_image":     repo.Name,
			// 			"target_image_tag": repo.Tag,
			// 			"policy":           policy,
			// 		}).Info("provider.kubernetes: impacted deployment container found")
			// 	}
			// }

			updated, shouldUpdateDeployment, err := p.checkDeployment(newVersion, repo, &deployment)
			if err != nil {
				log.WithFields(log.Fields{
					"error":      err,
					"deployment": deployment.Name,
					"namespace":  deployment.Namespace,
				}).Error("provider.kubernetes: got error while checking deployment")
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
