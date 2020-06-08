package helm3

import (
    "os"
    "strings"

    "helm.sh/helm/v3/pkg/chart"

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

// convert map[string]string to map[string]interface
// converts:
//     map[string]string{"image.tag": "0.1.0"}
// to:
//     map[string]interface{"image": map[string]interface{"tag": "0.1.0"}}
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