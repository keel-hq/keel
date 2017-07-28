package types

import (
	"github.com/rusenask/keel/util/image"
)

type Credentials struct {
	Username, Password string
}

type TrackedImage struct {
	Image        *image.Reference
	Trigger      TriggerType
	PollSchedule string
	Provider     string
	Namespace    string
	Secrets      []string
}
