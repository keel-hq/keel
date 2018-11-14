package provider

import (
	"context"

	"github.com/Masterminds/semver"
	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// Provider - generic provider interface
type Provider interface {
	Submit(event types.Event) error
	TrackedImages() ([]*types.TrackedImage, error)
	GetName() string
	Stop()
}

// Providers - available providers
type Providers interface {
	Submit(event types.Event) error
	TrackedImages() ([]*types.TrackedImage, error)
	List() []string // list all providers
	Stop()          // stop all providers
}

// New - new providers registry
func New(providers []Provider, approvalsManager approvals.Manager) *DefaultProviders {
	pvs := make(map[string]Provider)

	for _, p := range providers {
		pvs[p.GetName()] = p
		log.Infof("provider.defaultProviders: provider '%s' registered", p.GetName())
	}

	dp := &DefaultProviders{
		providers:        pvs,
		approvalsManager: approvalsManager,
		stopCh:           make(chan struct{}),
	}

	// subscribing to approved events
	// TODO: create Start() function for DefaultProviders
	go dp.subscribeToApproved()

	return dp
}

// DefaultProviders - default providers container
type DefaultProviders struct {
	providers        map[string]Provider
	approvalsManager approvals.Manager
	stopCh           chan struct{}
}

func (p *DefaultProviders) subscribeToApproved() {
	ctx, cancel := context.WithCancel(context.Background())

	approvedCh, err := p.approvalsManager.SubscribeApproved(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("provider.subscribeToApproved: failed to subscribe for approved reqs")
	}

	for {
		select {
		case approval := <-approvedCh:
			p.Submit(*approval.Event)
		case <-p.stopCh:
			cancel()
			return
		}
	}

}

// Submit - submit event to all providers
func (p *DefaultProviders) Submit(event types.Event) error {
	for _, provider := range p.providers {
		err := provider.Submit(event)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"provider": provider.GetName(),
				"event":    event.Repository,
				"trigger":  event.TriggerName,
			}).Error("provider.Submit: submit event failed")
		}
	}

	return nil
}

// TrackedImages - get tracked images for provider
func (p *DefaultProviders) TrackedImages() ([]*types.TrackedImage, error) {
	var trackedImages []*types.TrackedImage
	for _, provider := range p.providers {
		ti, err := provider.TrackedImages()
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"provider": provider.GetName(),
			}).Error("provider.defaultProviders: failed to get tracked images")
			continue
		}
		trackedImages = append(trackedImages, ti...)
	}

	log.WithFields(log.Fields{
		"images": trackedImages,
	}).Debug("tracked images")

	return trackedImages, nil
}

func appendIfDoesntExist(tags []string, tag string) []string {
	found := false
	for _, t := range tags {
		if t == tag {
			found = true
		}
	}
	if !found {
		return append(tags, tag)
	}
	return tags
}

func appendImage(images []*types.TrackedImage, new *types.TrackedImage) []*types.TrackedImage {

	newSemverTag, err := semver.NewVersion(new.Image.Tag())
	if err != nil {
		// not semver, just appending as a new image
		return append(images, new)
	}

	new.Tags = appendIfDoesntExist(new.Tags, new.Image.Tag())

	// looking for a semver image
	idx, ok := lookupSemverImageIdx(images, new)
	if !ok || len(images) == 0 {
		if newSemverTag.Prerelease() != "" {
			new.SemverPreReleaseTags[newSemverTag.Prerelease()] = new.Image.Tag()
			// new.SemverPreReleaseTags = append(new.SemverPreReleaseTags, newSemverTag.Prerelease())
		}
		return append(images, new)
	}

	existingSemverTag, err := semver.NewVersion(images[idx].Image.Tag())
	if err != nil {
		// existing tag not semver, just appending as new image
		if newSemverTag.Prerelease() != "" {
			new.SemverPreReleaseTags[newSemverTag.Prerelease()] = new.Image.Tag()
			// new.SemverPreReleaseTags = append(new.SemverPreReleaseTags, newSemverTag.Prerelease())
		}
		return append(images, new)
	}

	// semver, checking for prerelease tags
	if newSemverTag.Prerelease() != "" {
		_, ok := images[idx].SemverPreReleaseTags[newSemverTag.Prerelease()]
		if ok {
			// checking which is higher and setting higher
			if newSemverTag.GreaterThan(existingSemverTag) {
				images[idx].SemverPreReleaseTags[newSemverTag.Prerelease()] = new.Image.Tag()
				return images
			}
		}
		images[idx].SemverPreReleaseTags[newSemverTag.Prerelease()] = new.Image.Tag()
	}

	// if new semver tag is a higher version, updating it as well
	if newSemverTag.GreaterThan(existingSemverTag) {
		images[idx].Image = new.Image
	}

	images[idx].Tags = appendIfDoesntExist(images[idx].Tags, new.Image.Tag())

	return images
}

func lookupSemverImageIdx(images []*types.TrackedImage, new *types.TrackedImage) (int, bool) {
	_, err := semver.NewVersion(new.Image.Tag())
	if err != nil {
		// looking for a non semver
		return 0, false
	}
	for idx, existing := range images {

		if existing.Image.Repository() == new.Image.Repository() {
			_, err = semver.NewVersion(existing.Image.Tag())
			if err != nil {
				continue
			}
			return idx, true
		}
	}
	return 0, false
}

// List - list available providers
func (p *DefaultProviders) List() []string {
	list := []string{}
	for name := range p.providers {
		list = append(list, name)
	}
	return list
}

// Stop - stop all providers
func (p *DefaultProviders) Stop() {
	for _, provider := range p.providers {
		provider.Stop()
	}
}
