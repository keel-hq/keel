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
	Image        *image.Reference  `json:"image"`
	Trigger      TriggerType       `json:"trigger"`
	PollSchedule string            `json:"pollSchedule"`
	Provider     string            `json:"provider"`
	Namespace    string            `json:"namespace"`
	Secrets      []string          `json:"secrets"`
	Meta         map[string]string `json:"meta"` // metadata supplied by providers
	// a list of pre-release tags, ie: 1.0.0-dev, 1.5.0-prod get translated into
	// dev, prod
	// combined semver tags
	Tags   []string `json:"tags"`
	Policy Policy   `json:"policy"`
}

type Policy interface {
	ShouldUpdate(current, new string) (bool, error)
	Name() string
}

func (i TrackedImage) String() string {
	return fmt.Sprintf("namespace:%s,image:%s:%s,provider:%s,trigger:%s,sched:%s,secrets:%s", i.Namespace, i.Image.Repository(), i.Image.Tag(), i.Provider, i.Trigger, i.PollSchedule, i.Secrets)
}
