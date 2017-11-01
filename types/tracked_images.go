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
}

func (i TrackedImage) String() string {
	return fmt.Sprintf("namespace:%s,image:%s,provider:%s,trigger:%s,sched:%s,secrets:%s", i.Namespace, i.Image.Repository(), i.Provider, i.Trigger, i.PollSchedule, i.Secrets)
}
