package helm

import (
	"fmt"

	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/image"
	"github.com/rusenask/keel/util/version"

	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
	// rls "k8s.io/helm/pkg/proto/hapi/services"

	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
)

// ProviderName - helm provider name
const ProviderName = "helm"

// keel paths
const (
	policyPath = "keel.policy"
	imagesPath = "keel.images"
)

// Provider - helm provider, responsible for managing release updates
type Provider struct {
	implementer Implementer

	events chan *types.Event
	stop   chan struct{}
}

func NewProvider(implementer Implementer) *Provider {
	return &Provider{
		implementer: implementer,
		events:      make(chan *types.Event, 100),
		stop:        make(chan struct{}),
	}
}

func (p *Provider) GetName() string {
	return ProviderName
}

// Submit - submit event to provider
func (p *Provider) Submit(event types.Event) error {
	p.events <- &event
	return nil
}

// Start - starts kubernetes provider, waits for events
func (p *Provider) Start() error {
	return p.startInternal()
}

// Stop - stops kubernetes provider
func (p *Provider) Stop() {
	close(p.stop)
}

func (p *Provider) startInternal() error {
	for {
		select {
		case event := <-p.events:
			log.WithFields(log.Fields{
				"repository": event.Repository.Name,
				"tag":        event.Repository.Tag,
				"registry":   event.Repository.Host,
			}).Info("provider.helm: processing event")
			err := p.processEvent(event)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"image": event.Repository.Name,
					"tag":   event.Repository.Tag,
				}).Error("provider.helm: failed to process event")
			}
		case <-p.stop:
			log.Info("provider.helm: got shutdown signal, stopping...")
			return nil
		}
	}
}

func (p *Provider) processEvent(event *types.Event) (err error) {

	return nil
}

// UpdatePlan - release update plan
type UpdatePlan struct {
	Namespace string
	Name      string

	// chart
	Chart *hapi_chart.Chart

	// values to update path=value
	Values map[string]string
}

func (p *Provider) createUpdatePlans(event *types.Event) ([]*UpdatePlan, error) {
	var plans []*UpdatePlan

	releaseList, err := p.implementer.ListReleases()
	if err != nil {
		return nil, err
	}

	for _, release := range releaseList.Releases {

		newVersion, err := version.GetVersion(event.Repository.Tag)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("provider.helm: failed to parse version")
			continue
		}

		plan, update, err := p.checkVersionedRelease(newVersion, release.Namespace, release.Name, release.Chart, release.Config)

	}

	return plans, nil
}

// resp, err := u.client.UpdateRelease(
// 		u.release,
// 		chartPath,
// 		helm.UpdateValueOverrides(rawVals),
// 		helm.UpgradeDryRun(u.dryRun),
// 		helm.UpgradeRecreate(u.recreate),
// 		helm.UpgradeForce(u.force),
// 		helm.UpgradeDisableHooks(u.disableHooks),
// 		helm.UpgradeTimeout(u.timeout),
// 		helm.ResetValues(u.resetValues),
// 		helm.ReuseValues(u.reuseValues),
// 		helm.UpgradeWait(u.wait))
// 	if err != nil {
// 		return fmt.Errorf("UPGRADE FAILED: %v", prettyError(err))
// 	}

func updateHelmRelease(implementer Implementer, releaseName string, chart *hapi_chart.Chart, rawVals string) error {

	resp, err := implementer.UpdateReleaseFromChart(releaseName, chart,
		helm.UpdateValueOverrides([]byte(rawVals)),
		helm.UpgradeDryRun(false),
		helm.UpgradeRecreate(false),
		helm.UpgradeForce(true),
		helm.UpgradeDisableHooks(false),
		helm.UpgradeTimeout(30),
		helm.ResetValues(false),
		helm.ReuseValues(true),
		helm.UpgradeWait(true))

	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"version": resp.Release.Version,
		"release": releaseName,
	}).Info("provider.helm: release updated")
	return nil
}

// func parseImageOld(chart *hapi_chart.Chart, config *hapi_chart.Config) (*image.Reference, error) {
// 	vals, err := chartutil.ReadValues([]byte(config.Raw))
// 	if err != nil {
// 		return nil, err
// 	}

// 	log.Info(config.Raw)

// 	imageName, err := vals.PathValue("image.repository")
// 	if err != nil {
// 		return nil, err
// 	}

// 	// FIXME: need to dynamically get repositories
// 	imageTag, err := vals.PathValue("image.tag")
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get image tag: %s", err)
// 	}

// 	imageNameStr, ok := imageName.(string)
// 	if !ok {
// 		return nil, fmt.Errorf("failed to convert image name ref to string")
// 	}

// 	imageTagStr, ok := imageTag.(string)
// 	if !ok {
// 		return nil, fmt.Errorf("failed to convert image tag ref to string")
// 	}

// 	if imageTagStr != "" {
// 		return image.Parse(imageNameStr + ":" + imageTagStr)
// 	}

// 	return image.Parse(imageNameStr)
// }

func parseImage(vals chartutil.Values, details *ImageDetails) (*image.Reference, error) {
	if details.Repository == "" {
		return nil, fmt.Errorf("repository name path cannot be empty")
	}

	imageName, err := getValueAsString(vals, details.Repository)
	if err != nil {
		return nil, err
	}

	// getting image tag
	imageTag, err := getValueAsString(vals, details.Tag)
	if err != nil {
		// failed to find tag, returning anyway
		return image.Parse(imageName)
	}

	return image.Parse(imageName + ":" + imageTag)
}

func getValueAsString(vals chartutil.Values, path string) (string, error) {
	valinterface, err := vals.PathValue(path)
	if err != nil {
		return "", err
	}
	valString, ok := valinterface.(string)
	if !ok {
		return "", fmt.Errorf("failed to convert value  to string")
	}

	return valString, nil
}

func values(chart *hapi_chart.Chart, config *hapi_chart.Config) (chartutil.Values, error) {
	return chartutil.CoalesceValues(chart, config)
}

// keel:
//   # keel policy (all/major/minor/patch/force)
//   policy: all
//   # trigger type, defaults to events such as pubsub, webhooks
//   trigger: poll
//   # images to track and update
//   images:
//     - repository: image.repository
//       tag: image.tag

// Root - root element of the values yaml
type Root struct {
	Keel KeelChartConfig `json:"keel"`
}

// KeelChartConfig - keel related configuration taken from values.yaml
type KeelChartConfig struct {
	Policy  types.PolicyType `json:"policy"`
	Trigger string           `json:"trigger"`
	Images  []ImageDetails   `json:"images"`
}

// ImageDetails - image details
type ImageDetails struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

func getKeelConfig(vals chartutil.Values) (*KeelChartConfig, error) {
	yamlFull, err := vals.YAML()
	if err != nil {
		return nil, fmt.Errorf("failed to get vals config, error: %s", err)
	}

	var r Root
	err = yaml.Unmarshal([]byte(yamlFull), &r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keel config: %s", err)
	}
	return &r.Keel, nil
}
