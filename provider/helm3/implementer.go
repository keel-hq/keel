package helm3

import (
    "os"
    "strings"

	// "k8s.io/helm/pkg/helm"
	// "k8s.io/helm/pkg/proto/hapi/chart"
    "helm.sh/helm/v3/pkg/chart"
	// rls "k8s.io/helm/pkg/proto/hapi/services"

	log "github.com/sirupsen/logrus"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
    "helm.sh/helm/v3/pkg/cli"
)

// to do:
	// * update to latest chart package
	// * udpate the paramateres for the function

const DefaultUpdateTimeout = 300

// Implementer - generic helm implementer used to abstract actual implementation
type Implementer interface {
	// ListReleases(opts ...helm.ReleaseListOption) ([]*release.Release, error)
    ListReleases() ([]*release.Release, error)
	UpdateReleaseFromChart(rlsName string, chart *chart.Chart, vals map[string]string, opts ...bool) (*release.Release, error)
}

// Helm3Implementer - actual helm3 implementer
type Helm3Implementer struct {
	actionConfig *action.Configuration
}

// NewHelm3Implementer - get new helm implementer
func NewHelm3Implementer() *Helm3Implementer {
    settings := cli.New()

    actionConfig := &action.Configuration{}
    // You can pass an empty string instead of settings.Namespace() to list
    // all namespaces
    if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
        log.Printf("%+v", err)
        os.Exit(1)
    }
    return &Helm3Implementer{
    	actionConfig: actionConfig,
    }
}

// ListReleases - list available releases
func (i *Helm3Implementer) ListReleases() ([]*release.Release, error) {
    client := action.NewList(i.actionConfig)
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
// func (i *Helm3Implementer) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, vals map[string]string) (*release.Release, error) {
func (i *Helm3Implementer) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, vals map[string]string, opts ...bool) (*release.Release, error) {
    client := action.NewUpgrade(i.actionConfig)
	client.Force = true
	client.Timeout = DefaultUpdateTimeout;
	client.ReuseValues = true

    // set reuse values to false if currentRelease.config is nil
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

    // neil
    log.WithFields(log.Fields{
        "releaseConfig": results.Config,
        "releaseValues": results.Chart.Values,
        "vals": vals,
        "convertedVals": convertedVals,
    }).Info("provider.helm3: released")
    return results, err
}

func convertToInterface(values map[string]string) (map[string]interface{}) {
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