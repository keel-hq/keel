package types

import (
	"github.com/rusenask/keel/util/image"
)

type TrackedImage struct {
	Image        *image.Reference
	Trigger      TriggerType
	PollSchedule string
	Provider     string
}
