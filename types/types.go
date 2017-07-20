//go:generate jsonenums -type=Notification
//go:generate jsonenums -type=Level
//go:generate jsonenums -type=PolicyType
//go:generate jsonenums -type=TriggerType
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

// KeelPollScheduleAnnotation - optional variable to setup custom schedule for polling, defaults to @every 10m
const KeelPollScheduleAnnotation = "keel.sh/pollSchedule"

// KeelPollDefaultSchedule - defaul polling schedule
const KeelPollDefaultSchedule = "@every 1m"

// KeelDigestAnnotation - digest annotation
const KeelDigestAnnotation = "keel.sh/digest"

type Repository struct {
	Host   string `json:"host"`
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	Digest string `json:"digest"` // optional digest field
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
		return "default"
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
	PolicyTypeNone PolicyType = iota
	PolicyTypeAll
	PolicyTypeMajor
	PolicyTypeMinor
	PolicyTypePatch
	PolicyTypeForce // update always when a new image is available
)

// EventNotification notification used for sending
type EventNotification struct {
	Name      string       `json:"name"`
	Message   string       `json:"message"`
	CreatedAt time.Time    `json:"createdAt"`
	Type      Notification `json:"type"`
	Level     Level        `json:"level"`
}

// Notification - notification types used by notifier
type Notification int

// available notification types for hooks
const (
	PreProviderSubmitNotification Notification = iota
	PostProviderSubmitNotification

	// Kubernetes notification types
	NotificationPreDeploymentUpdate
	NotificationDeploymentUpdate

	// Helm notification types
	NotificationPreReleaseUpdate
	NotificationReleaseUpdate
)

func (n Notification) String() string {
	switch n {
	case PreProviderSubmitNotification:
		return "pre provider submit"
	case PostProviderSubmitNotification:
		return "post provider submit"
	case NotificationPreDeploymentUpdate:
		return "preparing deployment update"
	case NotificationDeploymentUpdate:
		return "deployment update"
	case NotificationPreReleaseUpdate:
		return "preparing release update"
	case NotificationReleaseUpdate:
		return "release update"
	default:
		return "unknown"
	}
}

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelSuccess
	LevelWarn
	LevelError
	LevelFatal
)

func (l Level) Color() string {
	switch l {
	case LevelError:
		return "#F44336"
	case LevelInfo:
		return "#2196F3"
	case LevelSuccess:
		return "#00C853"
	case LevelFatal:
		return "#B71C1C"
	case LevelWarn:
		return "#FF9800"
	default:
		return "#9E9E9E"
	}
}
