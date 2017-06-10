package kubernetes

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/rusenask/keel/types"
	// "github.com/rusenask/keel/util/version"

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
		cfg.Host = opts.Master
		cfg.KeyFile = opts.KeyFile
		cfg.CAFile = opts.CAFile
		cfg.CertFile = opts.CertFile
		log.Info("provider.kubernetes: using out-of-cluster configuration")
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
			err := p.processEvent(event)
			if err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"event":  event.Repository,
					"pusher": event.Pusher,
				}).Error("provider.kubernetes: failed to process event")
			}
		case <-p.stop:
			log.Info("provider.kubernetes: got shutdown signal, stopping...")
			return nil
		}
	}
}

func (p *Provider) processEvent(event *types.Event) error {
	impacted, err := p.impactedDeployments(&event.Repository)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"impacted": len(impacted),
	}).Info("processing event, got impacted deployments")

	return nil
}

// gets impacted deployments by changed repository
func (p *Provider) impactedDeployments(repo *types.Repository) ([]*v1beta1.Deployment, error) {
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
			policy, ok := labels[types.KeelPolicyLabel]
			// if no policy is set - skipping this deployment
			if !ok {
				continue
			}

			for _, c := range deployment.Spec.Template.Spec.Containers {
				// Remove version if any
				containerImageName := versionreg.ReplaceAllString(c.Image, "")
				if containerImageName != repo.Name {
					continue
				}

				// ver, err := version.GetVersion(c.Image)

				// if err != nil {
				// 	log.WithFields(log.Fields{
				// 		"error":       err,
				// 		"image_name":  c.Image,
				// 		"keel_policy": policy,
				// 	}).Error("provider.kubernetes: failed to get image version, is it tagged as semver?")
				// 	continue
				// }

				major, minor, patch := getVersion(c.Image)

				log.WithFields(log.Fields{
					"parsed_image": containerImageName,
					// "parsed_version":   fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Patch),
					"parsed_version":   fmt.Sprintf("%d.%d.%d", major, minor, patch),
					"raw_image_name":   c.Image,
					"target_image":     repo.Name,
					"target_image_tag": repo.Tag,
					// "deployment":   deployment.String(),
					"policy": policy,
				}).Info("checking container")
			}

		}
	}

	return impacted, nil
}

// decompose version string in major, minor, patch list.
func getVersion(a string) (int, int, int) {
	v := strings.Split(a, ".")
	log.Info(a)
	log.Info(len(v))
	switch len(v) {
	case 0:
		v = append(v, "0")
		fallthrough
	case 1:
		v = append(v, "0")
		fallthrough
	case 2:
		v = append(v, "0")
	}
	version := []int{}
	for _, i := range v {
		s, _ := strconv.Atoi(i)
		version = append(version, s)
	}
	return version[0], version[1], version[2]
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
