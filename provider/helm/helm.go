package helm

import (
	"fmt"
	"strings"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"github.com/keel-hq/keel/util/version"

	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/keel-hq/keel/extension/notification"

	log "github.com/sirupsen/logrus"
	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/strvals"
)

// Manager - high level interface into helm provider related data used by
// triggers
type Manager interface {
	Images() ([]*image.Reference, error)
}

// ProviderName - helm provider name
const ProviderName = "helm"

// DefaultUpdateTimeout - update timeout in seconds
const DefaultUpdateTimeout = 300

// UpdatePlan - release update plan
type UpdatePlan struct {
	Namespace string
	Name      string

	Config *KeelChartConfig

	// chart
	Chart *hapi_chart.Chart

	// values to update path=value
	Values map[string]string

	// Current (last seen cluster version)
	CurrentVersion string
	// New version that's already in the deployment
	NewVersion string
}

// keel:
//   # keel policy (all/major/minor/patch/force)
//   policy: all
//   # trigger type, defaults to events such as pubsub, webhooks
//   trigger: poll
//   pollSchedule: "@every 2m"
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
	Policy               types.PolicyType  `json:"policy"`
	Trigger              types.TriggerType `json:"trigger"`
	PollSchedule         string            `json:"pollSchedule"`
	Approvals            int               `json:"approvals"`        // Minimum required approvals
	ApprovalDeadline     int               `json:"approvalDeadline"` // Deadline in hours
	Images               []ImageDetails    `json:"images"`
	NotificationChannels []string          `json:"notificationChannels"` // optional notification channels
}

// ImageDetails - image details
type ImageDetails struct {
	RepositoryPath string `json:"repository"`
	TagPath        string `json:"tag"`
}

// Provider - helm provider, responsible for managing release updates
type Provider struct {
	implementer Implementer

	sender notification.Sender

	approvalManager approvals.Manager

	events chan *types.Event
	stop   chan struct{}
}

// NewProvider - create new Helm provider
func NewProvider(implementer Implementer, sender notification.Sender, approvalManager approvals.Manager) *Provider {
	return &Provider{
		implementer:     implementer,
		approvalManager: approvalManager,
		sender:          sender,
		events:          make(chan *types.Event, 100),
		stop:            make(chan struct{}),
	}
}

// GetName - get provider name
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

// TrackedImages - returns tracked images from all releases that have keel configuration
func (p *Provider) TrackedImages() ([]*types.TrackedImage, error) {
	var trackedImages []*types.TrackedImage

	releaseList, err := p.implementer.ListReleases()
	if err != nil {
		return nil, err
	}

	for _, release := range releaseList.Releases {
		// getting configuration
		vals, err := values(release.Chart, release.Config)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"release":   release.Name,
				"namespace": release.Namespace,
			}).Error("provider.helm: failed to get values.yaml for release")
			continue
		}

		cfg, err := getKeelConfig(vals)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"release":   release.Name,
				"namespace": release.Namespace,
			}).Error("provider.helm: failed to get config for release")
			continue
		}

		if cfg.PollSchedule == "" {
			cfg.PollSchedule = types.KeelPollDefaultSchedule
		}
		// used to check pod secrets
		selector := fmt.Sprintf("app=%s,release=%s", release.Chart.Metadata.Name, release.Name)

		releaseImages, err := getImages(vals)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"release":   release.Name,
				"namespace": release.Namespace,
			}).Error("provider.helm: failed to get images for release")
			continue
		}

		for _, img := range releaseImages {
			img.Meta = map[string]string{
				"selector": selector,
			}
			img.Namespace = release.Namespace
			img.Provider = ProviderName
			trackedImages = append(trackedImages, img)
		}

	}

	return trackedImages, nil
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
	plans, err := p.createUpdatePlans(event)
	if err != nil {
		return err
	}

	approved := p.checkForApprovals(event, plans)

	return p.applyPlans(approved)
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

			plan, update, errCheck := checkUnversionedRelease(&event.Repository, release.Namespace, release.Name, release.Chart, release.Config)
			if errCheck != nil {
				log.WithFields(log.Fields{
					"error":      err,
					"deployment": release.Name,
					"namespace":  release.Namespace,
				}).Error("provider.helm: got error while checking unversioned release")
				continue
			}

			if update {
				plans = append(plans, plan)
				continue
			}

			log.WithFields(log.Fields{
				"error": err,
				"tag":   event.Repository.Tag,
			}).Error("provider.helm: failed to parse version")
			continue
		}

		plan, update, err := checkVersionedRelease(newVersion, &event.Repository, release.Namespace, release.Name, release.Chart, release.Config)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"name":      release.Name,
				"namespace": release.Namespace,
			}).Error("provider.helm: failed to process versioned release")
			continue
		}
		if update {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

func (p *Provider) applyPlans(plans []*UpdatePlan) error {
	for _, plan := range plans {

		p.sender.Send(types.EventNotification{
			Name:      "update release",
			Message:   fmt.Sprintf("Preparing to update release %s/%s %s->%s (%s)", plan.Namespace, plan.Name, plan.CurrentVersion, plan.NewVersion, strings.Join(mapToSlice(plan.Values), ", ")),
			CreatedAt: time.Now(),
			Type:      types.NotificationPreReleaseUpdate,
			Level:     types.LevelDebug,
			Channels:  plan.Config.NotificationChannels,
		})

		err := updateHelmRelease(p.implementer, plan.Name, plan.Chart, plan.Values)
		if err != nil {
			log.WithFields(log.Fields{
				"error":     err,
				"name":      plan.Name,
				"namespace": plan.Namespace,
			}).Error("provider.helm: failed to apply plan")

			p.sender.Send(types.EventNotification{
				Name:      "update release",
				Message:   fmt.Sprintf("Release update failed %s/%s %s->%s (%s), error: %s", plan.Namespace, plan.Name, plan.CurrentVersion, plan.NewVersion, strings.Join(mapToSlice(plan.Values), ", "), err),
				CreatedAt: time.Now(),
				Type:      types.NotificationReleaseUpdate,
				Level:     types.LevelError,
				Channels:  plan.Config.NotificationChannels,
			})
			continue
		}

		p.sender.Send(types.EventNotification{
			Name:      "update release",
			Message:   fmt.Sprintf("Successfully updated release %s/%s %s->%s (%s)", plan.Namespace, plan.Name, plan.CurrentVersion, plan.NewVersion, strings.Join(mapToSlice(plan.Values), ", ")),
			CreatedAt: time.Now(),
			Type:      types.NotificationReleaseUpdate,
			Level:     types.LevelSuccess,
			Channels:  plan.Config.NotificationChannels,
		})

	}

	return nil
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

func updateHelmRelease(implementer Implementer, releaseName string, chart *hapi_chart.Chart, overrideValues map[string]string) error {

	overrideBts, err := convertToYaml(mapToSlice(overrideValues))
	if err != nil {
		return err
	}

	resp, err := implementer.UpdateReleaseFromChart(releaseName, chart,
		helm.UpdateValueOverrides(overrideBts),
		helm.UpgradeDryRun(false),
		helm.UpgradeRecreate(false),
		helm.UpgradeForce(true),
		helm.UpgradeDisableHooks(false),
		helm.UpgradeTimeout(DefaultUpdateTimeout),
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

func mapToSlice(values map[string]string) []string {
	converted := []string{}
	for k, v := range values {
		concat := k + "=" + v
		converted = append(converted, concat)
	}
	return converted
}

// parse
func convertToYaml(values []string) ([]byte, error) {
	base := map[string]interface{}{}
	for _, value := range values {
		if err := strvals.ParseInto(value, base); err != nil {
			return []byte{}, fmt.Errorf("failed parsing --set data: %s", err)
		}
	}

	return yaml.Marshal(base)
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
