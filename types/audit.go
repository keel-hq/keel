package types

import (
	"time"
)

const (
	AuditActionCreated = "created"
	AuditActionUpdated = "updated"
	AuditActionDeleted = "deleted"

	// Approval specific actions
	AuditActionApprovalApproved = "approved"
	AuditActionApprovalRejected = "rejected"
	AuditActionApprovalExpired  = "expired"
	AuditActionApprovalArchived = "archived"

	// audit specific resource kinds (others are set by
	// providers, ie: deployment, daemonset, helm chart)
	AuditResourceKindApproval = "approval"
	AuditResourceKindWebhook  = "webhook"
)

// AuditLog - audit logs lets users basic things happening in keel such as
// deployment updates and approval actions
type AuditLog struct {
	ID        string    `json:"id" gorm:"primary_key;type:varchar(36)"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	AccountID string `json:"accountId"`
	Username  string `json:"username"`
	Email     string `json:"email"`

	// create/delete/update
	Action       string `json:"action"`
	ResourceKind string `json:"resourceKind"` // approval/deployment/daemonset/statefulset/etc...
	Identifier   string `json:"identifier"`

	Message     string `json:"message"`
	Payload     string `json:"payload"` // can be used for bigger messages such as webhook payload
	PayloadType string `json:"payloadType"`

	Metadata JSONB `json:"metadata" gorm:"type:json"`
}

// SetMetadata - set audit log metadata (providers, namespaces)
func (l *AuditLog) SetMetadata(m map[string]string) {
	meta := make(map[string]interface{})
	for key, value := range m {
		meta[key] = value
	}

	l.Metadata = meta
}

// AuditLogQuery - struct used to query audit logs
type AuditLogQuery struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Order    string `json:"order"` // empty or "desc"
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`

	ResourceKindFilter []string `json:"resourceKindFilter"`
}

type AuditLogStatsQuery struct {
	Days int
}

type AuditLogStats struct {
	Date     string `json:"date"`
	Webhooks int    `json:"webhooks"`
	Approved int    `json:"approved"`
	Rejected int    `json:"rejected"`
	Updates  int    `json:"updates"`
}
