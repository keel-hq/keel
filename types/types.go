// Package types holds most of the types used across Keel
//go:generate jsonenums -type=Notification
//go:generate jsonenums -type=Level
//go:generate jsonenums -type=PolicyType
//go:generate jsonenums -type=TriggerType
//go:generate jsonenums -type=ProviderType
package types

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// KeelDefaultPort - default port for application
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

// KeelNotificationChanAnnotation - optional notification to override
// default notification channel(-s) per deployment/chart
const KeelNotificationChanAnnotation = "keel.sh/notify"

// KeelMinimumApprovalsLabel - min approvals
const KeelMinimumApprovalsLabel = "keel.sh/approvals"

// KeelApprovalDeadlineLabel - approval deadline
const KeelApprovalDeadlineLabel = "keel.sh/approvalDeadline"

// KeelApprovalDeadlineDefault - default deadline in hours
const KeelApprovalDeadlineDefault = 24

// KeelPodTerminationGracePeriod - optional grace period during
// pod termination
const KeelPodTerminationGracePeriod = "keel.sh/gracePeriod"

// Repository - represents main docker repository fields that
// keel cares about
type Repository struct {
	Host   string `json:"host"`
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	Digest string `json:"digest"` // optional digest field
}

// Event - holds information about new event from trigger
type Event struct {
	Repository Repository `json:"repository,omitempty"`
	CreatedAt  time.Time  `json:"createdAt,omitempty"`
	// optional field to identify trigger
	TriggerName string `json:"triggerName,omitempty"`
}

// Version - version container
type Version struct {
	Major      int64
	Minor      int64
	Patch      int64
	PreRelease string
	Metadata   string

	Original string
}

func (v Version) String() string {
	if v.Original != "" {
		return v.Original
	}
	var buf bytes.Buffer

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

// ParseTrigger - parse trigger string into type
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
	// Channels is an optional variable to override
	// default channel(-s) when performing an update
	Channels []string `json:"-"`
}

// ParseEventNotificationChannels - parses deployment annotations  or chart config
// to get channel overrides
func ParseEventNotificationChannels(annotations map[string]string) []string {
	channels := []string{}
	if annotations == nil {
		return channels
	}
	chanStr, ok := annotations[KeelNotificationChanAnnotation]
	if ok {
		chans := strings.Split(chanStr, ",")
		for _, c := range chans {
			channels = append(channels, strings.TrimSpace(c))
		}
	}

	return channels
}

// ParsePodTerminationGracePeriod - parses pod termination time in seconds
// from a given map of annotations
func ParsePodTerminationGracePeriod(annotations map[string]string) int64 {
	grace := int64(5)
	if annotations == nil {
		return grace
	}
	graceStr, ok := annotations[KeelPodTerminationGracePeriod]
	if ok {

		g, err := strconv.Atoi(graceStr)
		if err != nil {
			return grace
		}

		if g > 0 && g < 600 {
			return int64(g)
		}
	}

	return grace
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

// Level - event levet
type Level int

// Available event levels
const (
	LevelDebug Level = iota
	LevelInfo
	LevelSuccess
	LevelWarn
	LevelError
	LevelFatal
)

// ParseLevel takes a string level and returns notification level constant.
func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "fatal":
		return LevelFatal, nil
	case "error":
		return LevelError, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "info":
		return LevelInfo, nil
	case "success":
		return LevelSuccess, nil
	case "debug":
		return LevelDebug, nil
	}

	var l Level
	return l, fmt.Errorf("not a valid notification Level: %q", lvl)
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelSuccess:
		return "success"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// Color - used to assign different colors for events
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

// ProviderType - provider type used to differentiate different providers
// when used with plugins
type ProviderType int

// Known provider types
const (
	ProviderTypeUnknown ProviderType = iota
	ProviderTypeKubernetes
	ProviderTypeHelm
)

func (t ProviderType) String() string {
	switch t {
	case ProviderTypeUnknown:
		return "unknown"
	case ProviderTypeKubernetes:
		return "kubernetes"
	case ProviderTypeHelm:
		return "helm"
	default:
		return ""
	}
}

// Approval used to store and track updates
type Approval struct {
	// Provider name - Kubernetes/Helm
	Provider ProviderType `json:"provider,omitempty"`

	// Identifier is used to inform user about specific
	// Helm release or k8s deployment
	// ie: k8s <namespace>/<deployment name>
	//     helm: <namespace>/<release name>
	Identifier string `json:"identifier,omitempty"`

	// Event that triggered evaluation
	Event *Event `json:"event,omitempty"`

	Message string `json:"message,omitempty"`

	CurrentVersion string `json:"currentVersion,omitempty"`
	NewVersion     string `json:"newVersion,omitempty"`

	// Requirements for the update such as number of votes
	// and deadline
	VotesRequired int `json:"votesRequired,omitempty"`
	VotesReceived int `json:"votesReceived,omitempty"`

	// Voters is a list of voter
	// IDs for audit
	Voters []string `json:"voters,omitempty"`

	// Explicitly rejected approval
	// can be set directly by user
	// so even if deadline is not reached approval
	// could be turned down
	Rejected bool `json:"rejected,omitempty"`

	// Deadline for this request
	Deadline time.Time `json:"deadline,omitempty"`

	// When this approval was created
	CreatedAt time.Time `json:"createdAt,omitempty"`
	// WHen this approval was updated
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// ApprovalStatus - approval status type used in approvals
// to determine whether it was rejected/approved or still pending
type ApprovalStatus int

// Available approval status types
const (
	ApprovalStatusUnknown ApprovalStatus = iota
	ApprovalStatusPending
	ApprovalStatusApproved
	ApprovalStatusRejected
)

func (s ApprovalStatus) String() string {
	switch s {
	case ApprovalStatusPending:
		return "pending"
	case ApprovalStatusApproved:
		return "approved"
	case ApprovalStatusRejected:
		return "rejected"
	default:
		return "unknown"
	}
}

// Status - returns current approval status
func (a *Approval) Status() ApprovalStatus {
	if a.Rejected {
		return ApprovalStatusRejected
	}

	if a.VotesReceived >= a.VotesRequired {
		return ApprovalStatusApproved
	}

	return ApprovalStatusPending
}

// Expired - checks if approval is already expired
func (a *Approval) Expired() bool {
	return a.Deadline.Before(time.Now())
}

// Delta of what's changed
// ie: webhookrelay/webhook-demo:0.15.0 -> webhookrelay/webhook-demo:0.16.0
func (a *Approval) Delta() string {
	return fmt.Sprintf("%s -> %s", a.CurrentVersion, a.NewVersion)
}
