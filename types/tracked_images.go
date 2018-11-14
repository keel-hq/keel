package types

import (
	"fmt"

	"github.com/keel-hq/keel/util/image"
)

// Credentials - registry credentials
type Credentials struct {
	Username, Password string
}

// TrackedImage - tracked image data+metadata
type TrackedImage struct {
	Image        *image.Reference
	Trigger      TriggerType
	PollSchedule string
	Provider     string
	Namespace    string
	Secrets      []string
	Meta         map[string]string // metadata supplied by providers
	// a list of pre-release tags, ie: 1.0.0-dev, 1.5.0-prod get translated into
	// dev, prod
	// SemverPreReleaseTags []string
	SemverPreReleaseTags map[string]string
	// combined semver tags
	Tags []string
}

func (i TrackedImage) String() string {
	return fmt.Sprintf("namespace:%s,image:%s:%s,provider:%s,trigger:%s,sched:%s,secrets:%s,semver:%v,tags:%v", i.Namespace, i.Image.Repository(), i.Image.Tag(), i.Provider, i.Trigger, i.PollSchedule, i.Secrets, i.SemverPreReleaseTags, i.Tags)
}
