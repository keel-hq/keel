package kubernetes

import (
	"fmt"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/Sirupsen/logrus"
)

// Implementer - thing wrapper around currently used k8s APIs
type Implementer interface {
	Namespaces() (*v1.NamespaceList, error)
	Deployment(namespace, name string) (*v1beta1.Deployment, error)
	Deployments(namespace string) (*v1beta1.DeploymentList, error)
	Update(deployment *v1beta1.Deployment) error
}

// KubernetesImplementer - default kubernetes client implementer, uses
// https://github.com/kubernetes/client-go v3.0.0-beta.0
type KubernetesImplementer struct {
	cfg    *rest.Config
	client *kubernetes.Clientset
}

// Opts - implementer options, usually for k8s deployments
// it's best to use InCluster option
type Opts struct {
	// if set - kube config options will be ignored
	InCluster  bool
	ConfigPath string
	Master     string
}

// NewKubernetesImplementer - create new k8s implementer
func NewKubernetesImplementer(opts *Opts) (*KubernetesImplementer, error) {
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

	return &KubernetesImplementer{client: client, cfg: cfg}, nil
}

// Namespaces - get all namespaces
func (i *KubernetesImplementer) Namespaces() (*v1.NamespaceList, error) {
	namespaces := i.client.Namespaces()
	return namespaces.List(meta_v1.ListOptions{})
}

// Deployment - get specific deployment for namespace/name
func (i *KubernetesImplementer) Deployment(namespace, name string) (*v1beta1.Deployment, error) {
	dep := i.client.Extensions().Deployments(namespace)
	return dep.Get(name, meta_v1.GetOptions{})
}

// Deployments - get all deployments for namespace
func (i *KubernetesImplementer) Deployments(namespace string) (*v1beta1.DeploymentList, error) {
	dep := i.client.Extensions().Deployments(namespace)
	l, err := dep.List(meta_v1.ListOptions{})
	return l, err
}

// Update - update deployment
func (i *KubernetesImplementer) Update(deployment *v1beta1.Deployment) error {
	_, err := i.client.Extensions().Deployments(deployment.Namespace).Update(deployment)
	return err
}
