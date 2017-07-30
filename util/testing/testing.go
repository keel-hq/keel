package testing

import (
	// meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

type FakeK8sImplementer struct {
	NamespacesList   *v1.NamespaceList
	DeploymentSingle *v1beta1.Deployment
	DeploymentList   *v1beta1.DeploymentList

	// stores value of an updated deployment
	Updated *v1beta1.Deployment

	AvailableSecret *v1.Secret

	// error to return
	Error error
}

func (i *FakeK8sImplementer) Namespaces() (*v1.NamespaceList, error) {
	return i.NamespacesList, nil
}

func (i *FakeK8sImplementer) Deployment(namespace, name string) (*v1beta1.Deployment, error) {
	return i.DeploymentSingle, nil
}

func (i *FakeK8sImplementer) Deployments(namespace string) (*v1beta1.DeploymentList, error) {
	return i.DeploymentList, nil
}

func (i *FakeK8sImplementer) Update(deployment *v1beta1.Deployment) error {
	i.Updated = deployment
	return nil
}

func (i *FakeK8sImplementer) Secret(namespace, name string) (*v1.Secret, error) {
	if i.Error != nil {
		return nil, i.Error
	}
	return i.AvailableSecret, nil
}
