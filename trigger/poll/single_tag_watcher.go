package poll

import (
	"github.com/keel-hq/keel/extension/credentialshelper"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// WatchTagJob - Watch specific tag job
type WatchTagJob struct {
	providers      provider.Providers
	registryClient registry.Client
	details        *watchDetails
}

// NewWatchTagJob - new watch tag job monitors specific tag by checking digest based on specified
// cron style schedule
func NewWatchTagJob(providers provider.Providers, registryClient registry.Client, details *watchDetails) *WatchTagJob {
	return &WatchTagJob{
		providers:      providers,
		registryClient: registryClient,
		details:        details,
	}
}

// Run - main function to check schedule
func (j *WatchTagJob) Run() {
	reg := j.details.trackedImage.Image.Scheme() + "://" + j.details.trackedImage.Image.Registry()
	registryOpts := registry.Opts{
		Registry: reg,
		Name:     j.details.trackedImage.Image.ShortName(),
		Tag:      j.details.trackedImage.Image.Tag(),
	}

	creds, err := credentialshelper.GetCredentials(j.details.trackedImage)
	if err == nil {
		registryOpts.Username = creds.Username
		registryOpts.Password = creds.Password
	}

	currentDigest, err := j.registryClient.Digest(registryOpts)

	registriesScannedCounter.With(prometheus.Labels{"registry": j.details.trackedImage.Image.Registry(), "image": j.details.trackedImage.Image.Repository()}).Inc()

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"image": j.details.trackedImage.Image.String(),
		}).Error("trigger.poll.WatchTagJob: failed to check digest")
		return
	}

	// A changed image digest always triggers an update. In addition, an explicit
	// force update (keel.sh/force-update) triggers a rolling restart even when
	// the tag and digest are unchanged, so a same-tag redeploy can be requested
	// on demand (https://github.com/keel-hq/keel/issues/846).
	if !j.shouldUpdate(currentDigest) {
		return
	}

	log.WithFields(log.Fields{
		"current_digest": j.details.digest,
		"new_digest":     currentDigest,
		"registry_url":   reg,
		"image":          j.details.trackedImage.Image.String(),
		"force_update":   j.forceUpdateRequested(),
	}).Debug("trigger.poll.WatchTagJob: checking digest")

	// updating digest
	j.details.digest = currentDigest

	// consume the force-update request so it triggers at most one rollout. It is
	// reset by Watch() once the keel.sh/force-update annotation is cleared, so a
	// fresh request can fire again. This keeps the normal poll-based
	// no-change/no-op behavior intact and avoids a rolling restart on every poll
	// cycle.
	if j.details.trackedImage.ForceUpdate {
		j.details.mu.Lock()
		j.details.forceUpdateConsumed = true
		j.details.mu.Unlock()
	}

	event := types.Event{
		Repository: types.Repository{
			Name:   j.details.trackedImage.Image.Repository(),
			Tag:    j.details.trackedImage.Image.Tag(),
			Digest: currentDigest,
		},
		TriggerName: types.TriggerTypePoll.String(),
	}
	log.WithFields(log.Fields{
		"image":      j.details.trackedImage.Image.String(),
		"new_digest": currentDigest,
	}).Info("trigger.poll.WatchTagJob: digest change detected, submiting event to providers")

	// j.providers.Submit(event)
	err = j.providers.Submit(event)
	if err != nil {
		log.WithFields(log.Fields{
			"repository": j.details.trackedImage.Image.Repository(),
			"digest":     currentDigest,
			"error":      err,
		}).Error("trigger.poll.WatchRepositoryTagsJob: error while submitting an event")
	}
}

// shouldUpdate reports whether the event should be submitted. The tracked
// digest changing is a normal update; an explicit force-update request
// (keel.sh/force-update) is consumed once so a same-tag rolling restart is
// triggered exactly once per request.
func (j *WatchTagJob) shouldUpdate(currentDigest string) bool {
	j.details.mu.Lock()
	defer j.details.mu.Unlock()

	if j.details.digest != currentDigest {
		return true
	}

	if j.details.trackedImage.ForceUpdate && !j.details.forceUpdateConsumed {
		j.details.forceUpdateConsumed = true
		return true
	}

	return false
}

// forceUpdateRequested reports whether keel.sh/force-update is currently set on
// the tracked resource.
func (j *WatchTagJob) forceUpdateRequested() bool {
	j.details.mu.RLock()
	defer j.details.mu.RUnlock()
	return j.details.trackedImage.ForceUpdate
}
