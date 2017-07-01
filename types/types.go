package types

import (
	"bytes"
	"fmt"
	"time"
)

const KeelDefaultPort = 9300

// KeelPolicyLabel - keel update policies (version checking)
const KeelPolicyLabel = "keel.sh/policy"

// KeelTriggerLabel - trigger label is used to specify custom trigger types
// for example keel.sh/trigger=poll would signal poll trigger to start watching for repository
// changes
const KeelTriggerLabel = "keel.sh/trigger"

// KeelPollSchedule - optional variable to setup custom schedule for polling, defaults to @every 10m
const KeelPollSchedule = "keel.sh/pollSchedule"

// KeelPollDefaultSchedule - defaul polling schedule
const KeelPollDefaultSchedule = "@every 1m"

type Repository struct {
	Host string `json:"host,omitempty"`
	Name string `json:"name,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

type Event struct {
	Repository Repository `json:"repository,omitempty"`
	CreatedAt  time.Time  `json:"createdAt,omitempty"`
	// optional field to identify trigger
	TriggerName string `json:"triggerName,omitempty"`
}

type Version struct {
	Major      int64
	Minor      int64
	Patch      int64
	PreRelease string
	Metadata   string

	Prefix string // v prefix
}

func (v Version) String() string {
	var buf bytes.Buffer
	if v.Prefix != "" {
		fmt.Fprintf(&buf, v.Prefix)
	}

	fmt.Fprintf(&buf, "%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		fmt.Fprintf(&buf, "-%s", v.PreRelease)
	}
	if v.Metadata != "" {
		fmt.Fprintf(&buf, "+%s", v.Metadata)
	}

	return buf.String()
}

// TriggerType - trigger types
type TriggerType int

// Available trigger types
const (
	TriggerTypeDefault TriggerType = iota // default policy is to wait for external triggers
	TriggerTypePoll                       // poll policy sets up watchers for the affected repositories
)

func (t TriggerType) String() string {
	switch t {
	case TriggerTypeDefault:
		return "default"
	case TriggerTypePoll:
		return "poll"
	default:
		return "unknown"
	}
}

func ParseTrigger(trigger string) TriggerType {
	switch trigger {
	case "poll":
		return TriggerTypePoll
	}
	return TriggerTypeDefault
}

// PolicyType - policy type
type PolicyType int

// ParsePolicy - parse policy type
func ParsePolicy(policy string) PolicyType {
	switch policy {
	case "all":
		return PolicyTypeAll
	case "major":
		return PolicyTypeMajor
	case "minor":
		return PolicyTypeMinor
	case "patch":
		return PolicyTypePatch
	case "force":
		return PolicyTypeForce
	default:
		return PolicyTypeNone
	}
}

func (t PolicyType) String() string {
	switch t {
	case PolicyTypeNone:
		return "none"
	case PolicyTypeAll:
		return "all"
	case PolicyTypeMajor:
		return "major"
	case PolicyTypeMinor:
		return "minor"
	case PolicyTypePatch:
		return "patch"
	case PolicyTypeForce:
		return "force"
	default:
		return ""
	}
}

// available policies
const (
	PolicyTypeNone = iota
	PolicyTypeAll
	PolicyTypeMajor
	PolicyTypeMinor
	PolicyTypePatch
	PolicyTypeForce // update always when a new image is available
)
