package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type GetApprovalQuery struct {
	Identifier string
	// Rejected   bool
	Archived bool
}

// Approval used to store and track updates
type Approval struct {
	ID string `json:"id" gorm:"primary_key;type:varchar(36)"`

	// Archived is set to true once approval is finally approved/rejected
	Archived bool `json:"archived"`

	// Provider name - Kubernetes/Helm
	Provider ProviderType `json:"provider"`

	// Identifier is used to inform user about specific
	// Helm release or k8s deployment
	// ie: k8s <namespace>/<deployment name>
	//     helm: <namespace>/<release name>
	Identifier string `json:"identifier"`

	// Event that triggered evaluation
	Event *Event `json:"event" gorm:"type:json"`

	Message string `json:"message"`

	CurrentVersion string `json:"currentVersion"`
	NewVersion     string `json:"newVersion"`

	// Digest is used to verify that images are the ones that got the approvals.
	// If digest doesn't match for the image, votes are reset.
	Digest string `json:"digest"`

	// Requirements for the update such as number of votes
	// and deadline
	VotesRequired int `json:"votesRequired"`
	VotesReceived int `json:"votesReceived"`

	// Voters is a list of voter
	// IDs for audit
	Voters JSONB `json:"voters" gorm:"type:json"`

	// Explicitly rejected approval
	// can be set directly by user
	// so even if deadline is not reached approval
	// could be turned down
	Rejected bool `json:"rejected"`

	// Deadline for this request
	Deadline time.Time `json:"deadline"`

	// When this approval was created
	CreatedAt time.Time `json:"createdAt"`
	// WHen this approval was updated
	UpdatedAt time.Time `json:"updatedAt"`
}

func (a *Approval) GetVoters() []string {
	// meta := make(map[string]string)
	var voters []string
	for key := range a.Voters {
		voters = append(voters, key)

	}
	return voters
}

func (a *Approval) AddVoter(voter string) {
	if a.Voters == nil {
		a.Voters = make(map[string]interface{})
	}
	a.Voters[voter] = time.Now()
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

// JSONB is stored as a JSON blob
type JSONB map[string]interface{}

func (b JSONB) Value() (driver.Value, error) {
	j, err := json.Marshal(b)
	return j, err
}

func (b *JSONB) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed.")
	}

	var i interface{}
	if err := json.Unmarshal(source, &i); err != nil {
		return err
	}

	if i == nil {
		return nil
	}

	*b, ok = i.(map[string]interface{})
	if !ok {
		return errors.New("type assertion .(map[string]interface{}) failed.")
	}

	return nil
}
