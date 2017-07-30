package types

import (
	"fmt"

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

func (i TrackedImage) String() string {
	return fmt.Sprintf("namespace:%s,image:%s,provider:%s,trigger:%s,sched:%s,secrets:%s", i.Namespace, i.Image.Repository(), i.Provider, i.Trigger, i.PollSchedule, i.Secrets)
}
