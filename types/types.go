package types

import (
	"fmt"
	"time"
)

const KeelDefaultPort = 9300
const KeelPolicyLabel = "keel.observer/policy"

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
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
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
	default:
		return PolicyTypeUnknown
	}
}

func (t PolicyType) String() string {
	switch t {
	case PolicyTypeUnknown:
		return "unknown"
	case PolicyTypeAll:
		return "all"
	case PolicyTypeMajor:
		return "major"
	case PolicyTypeMinor:
		return "minor"
	case PolicyTypePatch:
		return "patch"
	default:
		return ""
	}
}

// available policies
const (
	PolicyTypeUnknown = iota
	PolicyTypeAll
	PolicyTypeMajor
	PolicyTypeMinor
	PolicyTypePatch
)
