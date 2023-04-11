package helm3

import (
	"os"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/chart"

	log "github.com/sirupsen/logrus"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	// "helm.sh/helm/v3/pkg/cli"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// to do:
// * update to latest chart package
// * udpate the paramateres for the function

// #595 - DefaultUpdateTimeout is in ns
// Per https://pkg.go.dev/helm.sh/helm/v3/pkg/action#Upgrade
const DefaultUpdateTimeout = 5 * time.Minute

// Implementer - generic helm implementer used to abstract actual implementation
type Implementer interface {
	// ListReleases(opts ...helm.ReleaseListOption) ([]*release.Release, error)
	ListReleases() ([]*release.Release, error)
	UpdateReleaseFromChart(rlsName string, chart *chart.Chart, vals map[string]string, namespace string, opts ...bool) (*release.Release, error)
}

// Helm3Implementer - actual helm3 implementer
type Helm3Implementer struct {
	// actionConfig *action.Configuration
	HelmDriver    string
	KubeContext   string
	KubeToken     string
	KubeAPIServer string
}

// NewHelm3Implementer - get new helm implementer
func NewHelm3Implementer() *Helm3Implementer {
	return &Helm3Implementer{
		HelmDriver:    os.Getenv("HELM_DRIVER"),
		KubeContext:   os.Getenv("HELM_KUBECONTEXT"),
		KubeToken:     os.Getenv("HELM_KUBETOKEN"),
		KubeAPIServer: os.Getenv("HELM_KUBEAPISERVER"),
	}
}

// ListReleases - list available releases
func (i *Helm3Implementer) ListReleases() ([]*release.Release, error) {
	actionConfig := i.generateConfig("")
	client := action.NewList(actionConfig)
	results, err := client.Run()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("helm3: failed to list release")
		return []*release.Release{}, err
	}
	return results, nil
}

// UpdateReleaseFromChart - update release from chart
func (i *Helm3Implementer) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, vals map[string]string, namespace string, opts ...bool) (*release.Release, error) {
	actionConfig := i.generateConfig(namespace)
	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Force = true
	client.Timeout = DefaultUpdateTimeout
	client.ReuseValues = true

	// set reuse values to false if currentRelease.config is nil (temp fix for bug in chartutil.coalesce v3.1.2)
	if len(opts) == 1 && opts[0] {
		client.ReuseValues = false
	}

	convertedVals := convertToInterface(vals)

	// returns the new release
	results, err := client.Run(rlsName, chart, convertedVals)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("helm3: failed to update release from chart")
		return nil, err
	}
	return results, err
}

func (i *Helm3Implementer) generateConfig(namespace string) *action.Configuration {
	// settings := cli.New()
	config := &genericclioptions.ConfigFlags{
		Namespace:   &namespace,
		Context:     &i.KubeContext,
		BearerToken: &i.KubeToken,
		APIServer:   &i.KubeAPIServer,
	}

	actionConfig := &action.Configuration{}

	if err := actionConfig.Init(config, namespace, i.HelmDriver, log.Printf); err != nil {
		log.Printf("%+v", err)
		os.Exit(1)
	}

	return actionConfig
}

// convert map[string]string to map[string]interface
// converts:
//     map[string]string{"image.tag": "0.1.0"}
// to:
//     map[string]interface{"image": map[string]interface{"tag": "0.1.0"}}
func convertToInterface(values map[string]string) map[string]interface{} {
	converted := make(map[string]interface{})
	for key, value := range values {
		keys := strings.SplitN(key, ".", 2)
		if len(keys) == 1 {
			converted[key] = value
		} else if len(keys) == 2 {
			converted[keys[0]] = convertToInterface(map[string]string{
				keys[1]: value,
			})
		}
	}
	return converted
}
