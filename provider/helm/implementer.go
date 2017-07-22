package helm

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

var (
	TillerAddress = "tiller-deploy:44134"
)

type Implementer interface {
	ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error)
	UpdateReleaseFromChart(rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error)
}

type HelmImplementer struct {
	client helm.Interface
}

func NewHelmImplementer(address string) *HelmImplementer {
	if address == "" {
		address = TillerAddress
	}

	return &HelmImplementer{
		client: helm.NewClient(helm.Host(address)),
	}
}

func (i *HelmImplementer) ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error) {
	return i.client.ListReleases(opts...)
}

func (i *HelmImplementer) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	return i.client.UpdateReleaseFromChart(rlsName, chart, opts...)
}
