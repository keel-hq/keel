package helm3

import (
	"errors"
	"os"
	"sort"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/chart"

	log "github.com/sirupsen/logrus"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
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
	if err := actionConfig.KubeClient.IsReachable(); err != nil {
		return nil, err
	}
	results, err := listCurrentReleases(actionConfig.Releases)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("helm3: failed to list releases")
		return nil, err
	}
	return results, nil
}

type releaseQuerier interface {
	Query(map[string]string) ([]*release.Release, error)
}

var currentReleaseStatuses = []release.Status{
	release.StatusUnknown,
	release.StatusDeployed,
	release.StatusUninstalled,
	release.StatusFailed,
	release.StatusUninstalling,
	release.StatusPendingInstall,
	release.StatusPendingUpgrade,
	release.StatusPendingRollback,
}

// listCurrentReleases uses Helm's storage labels to avoid retrieving and
// decoding superseded revisions. It still considers every non-superseded
// status when selecting the latest revision, matching action.List semantics
// during installs, upgrades, rollbacks, and uninstalls.
func listCurrentReleases(storage releaseQuerier) ([]*release.Release, error) {
	latest := make(map[string]*release.Release)
	for _, status := range currentReleaseStatuses {
		releases, err := storage.Query(map[string]string{
			"owner":  "helm",
			"status": status.String(),
		})
		if errors.Is(err, driver.ErrReleaseNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, candidate := range releases {
			if candidate == nil {
				continue
			}
			key := candidate.Namespace + "\x00" + candidate.Name
			if existing := latest[key]; existing == nil || candidate.Version > existing.Version {
				latest[key] = candidate
			}
		}
	}

	results := make([]*release.Release, 0, len(latest))
	for _, candidate := range latest {
		if candidate.Info == nil || (candidate.Info.Status != release.StatusDeployed && candidate.Info.Status != release.StatusFailed) {
			continue
		}
		results = append(results, candidate)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}
		return results[i].Namespace < results[j].Namespace
	})
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
//
//	map[string]string{"image.tag": "0.1.0"}
//
// to:
//
//	map[string]interface{"image": map[string]interface{"tag": "0.1.0"}}
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
