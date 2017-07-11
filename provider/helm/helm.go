package helm

import (
	"fmt"
	"os"

	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/tlsutil"
)

var (
	tlsCaCertFile string // path to TLS CA certificate file
	tlsCertFile   string // path to TLS certificate file
	tlsKeyFile    string // path to TLS key file
	tlsVerify     bool   // enable TLS and verify remote certificates
	tlsEnable     bool   // enable TLS

	// kubeContext string
	// tillerTunnel *kube.Tunnel
	settings helm_env.EnvSettings
)

func newClient() helm.Interface {
	options := []helm.Option{helm.Host(settings.TillerHost)}

	if tlsVerify || tlsEnable {
		tlsopts := tlsutil.Options{KeyFile: tlsKeyFile, CertFile: tlsCertFile, InsecureSkipVerify: true}
		if tlsVerify {
			tlsopts.CaCertFile = tlsCaCertFile
			tlsopts.InsecureSkipVerify = false
		}
		tlscfg, err := tlsutil.ClientConfig(tlsopts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		options = append(options, helm.WithTLS(tlscfg))
	}
	return helm.NewClient(options...)
}

type HelmImplementer struct {
	client helm.Interface
}

func NewHelmImplementer() *HelmImplementer {
	client := newClient()

	return &HelmImplementer{
		client: client,
	}
}

func (i *HelmImplementer) ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error) {
	return i.client.ListReleases(opts...)
}
