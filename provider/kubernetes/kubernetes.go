package kubernetes

import (
	"fmt"
	"regexp"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/version"

	log "github.com/Sirupsen/logrus"
)

const ProviderName = "kubernetes"

var versionreg = regexp.MustCompile(`:[^:]*$`)

// Provider - kubernetes provider for auto update
type Provider struct {
	cfg    *rest.Config
	client *kubernetes.Clientset

	events chan *types.Event
	stop   chan struct{}
}

type Opts struct {
	// if set - kube config options will be ignored
	InCluster bool

	ConfigPath string

	// Master host
	Master   string
	KeyFile  string
	CAFile   string
	CertFile string
}

// NewProvider - create new kubernetes based provider
func NewProvider(opts *Opts) (*Provider, error) {
	cfg := &rest.Config{}

	if opts.InCluster {
		var err error
		cfg, err = rest.InClusterConfig()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("provider.kubernetes: failed to get kubernetes config")
			return nil, err
		}
		log.Info("provider.kubernetes: using in-cluster configuration")
	} else if opts.ConfigPath != "" {
		var err error
		cfg, err = clientcmd.BuildConfigFromFlags("", opts.ConfigPath)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("provider.kubernetes: failed to get cmd kubernetes config")
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("kubernetes config is missing")
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("provider.kubernetes: failed to create kubernetes client")
		return nil, err
	}

	return &Provider{
		cfg:    cfg,
		client: client,
		events: make(chan *types.Event, 100),
		stop:   make(chan struct{}),
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
		_, err := p.client.Extensions().Deployments(deployment.Namespace).Update(deployment)
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
	dep := p.client.Extensions().Deployments(namespace)
	return dep.Get(name, meta_v1.GetOptions{})
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

			for idx, c := range deployment.Spec.Template.Spec.Containers {
				// Remove version if any
				containerImageName := versionreg.ReplaceAllString(c.Image, "")
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
	namespaces := p.client.Namespaces()
	return namespaces.List(meta_v1.ListOptions{})
}

// deployments - gets all deployments
func (p *Provider) deployments() ([]*v1beta1.DeploymentList, error) {
	// namespaces := p.client.Namespaces()
	deployments := []*v1beta1.DeploymentList{}

	n, err := p.namespaces()
	if err != nil {
		return nil, err
	}

	for _, n := range n.Items {
		dep := p.client.Extensions().Deployments(n.GetName())
		l, err := dep.List(meta_v1.ListOptions{})
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
