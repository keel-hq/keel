package types

import (
	"time"
)

const KeelDefaultPort = 9300
const KeelPolicyLabel = "keel.io/policy"

type Repository struct {
	Name string `json:"name,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

type Event struct {
	Repository Repository `json:"repository,omitempty"`
	Pusher     string     `json:"pusher,omitempty"`
	CreatedAt  time.Time  `json:"createdAt,omitempty"`
}

type Version struct {
	Major      int64
	Minor      int64
	Patch      int64
	PreRelease string
	Metadata   string
}

// PolicyType - policy type
type PolicyType int

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
)
