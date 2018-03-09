package helm

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"

	log "github.com/sirupsen/logrus"
)

// TillerAddress - default tiller address
var (
	TillerAddress = "tiller-deploy:44134"
)

// Implementer - generic helm implementer used to abstract actual implementation
type Implementer interface {
	ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error)
	UpdateReleaseFromChart(rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error)
}

// HelmImplementer - actual helm implementer
type HelmImplementer struct {
	client helm.Interface
}

// NewHelmImplementer - get new helm implementer
func NewHelmImplementer(address string) *HelmImplementer {
	if address == "" {
		address = TillerAddress
	} else {
		log.Infof("provider.helm: tiller address '%s' supplied", address)
	}

	return &HelmImplementer{
		client: helm.NewClient(helm.Host(address)),
	}
}

// ListReleases - list available releases
func (i *HelmImplementer) ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error) {
	return i.client.ListReleases(opts...)
}

// UpdateReleaseFromChart - update release from chart
func (i *HelmImplementer) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	return i.client.UpdateReleaseFromChart(rlsName, chart, opts...)
}
