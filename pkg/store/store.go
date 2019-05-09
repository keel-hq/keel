package store

import (
	"errors"

	"github.com/keel-hq/keel/types"
)

type Store interface {
	CreateAuditLog(entry *types.AuditLog) (id string, err error)
	GetAuditLogs(query *types.AuditLogQuery) (logs []*types.AuditLog, err error)
	AuditLogsCount(query *types.AuditLogQuery) (int, error)
	AuditStatistics(query *types.AuditLogStatsQuery) ([]types.AuditLogStats, error)

	CreateApproval(approval *types.Approval) (*types.Approval, error)
	UpdateApproval(approval *types.Approval) error
	GetApproval(q *types.GetApprovalQuery) (*types.Approval, error)
	ListApprovals(q *types.GetApprovalQuery) ([]*types.Approval, error)
	DeleteApproval(approval *types.Approval) error

	OK() bool
	Close() error
}

// errors
var (
	ErrRecordNotFound = errors.New("record not found")
)
