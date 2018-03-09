package kubernetes

import (
	"fmt"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	// "k8s.io/api/core/v1"

	"k8s.io/api/core/v1"

	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/sirupsen/logrus"
)

// Implementer - thing wrapper around currently used k8s APIs
type Implementer interface {
	Namespaces() (*v1.NamespaceList, error)

	Deployment(namespace, name string) (*v1beta1.Deployment, error)
	Deployments(namespace string) (*v1beta1.DeploymentList, error)
	Update(deployment *v1beta1.Deployment) error

	Secret(namespace, name string) (*v1.Secret, error)

	Pods(namespace, labelSelector string) (*v1.PodList, error)
	DeletePod(namespace, name string, opts *meta_v1.DeleteOptions) error

	ConfigMaps(namespace string) core_v1.ConfigMapInterface
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
	namespaces := i.client.Core().Namespaces()
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
	// retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
	// 	// Retrieve the latest version of Deployment before attempting update
	// 	// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
	// 	_, updateErr := i.client.Extensions().Deployments(deployment.Namespace).Update(deployment)
	// 	return updateErr
	// })
	// return retryErr

	_, err := i.client.Extensions().Deployments(deployment.Namespace).Update(deployment)
	return err
}

// Secret - get secret
func (i *KubernetesImplementer) Secret(namespace, name string) (*v1.Secret, error) {

	return i.client.Core().Secrets(namespace).Get(name, meta_v1.GetOptions{})
}

// Pods - get pods
func (i *KubernetesImplementer) Pods(namespace, labelSelector string) (*v1.PodList, error) {
	return i.client.Core().Pods(namespace).List(meta_v1.ListOptions{LabelSelector: labelSelector})
}

// DeletePod - delete pod by name
func (i *KubernetesImplementer) DeletePod(namespace, name string, opts *meta_v1.DeleteOptions) error {
	return i.client.Core().Pods(namespace).Delete(name, opts)
}

// ConfigMaps - returns an interface to config maps for a specified namespace
func (i *KubernetesImplementer) ConfigMaps(namespace string) core_v1.ConfigMapInterface {
	return i.client.Core().ConfigMaps(namespace)
}
