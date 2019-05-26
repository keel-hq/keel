package sql

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/keel-hq/keel/types"
)

// CreateAuditLog - create new audit log entry
func (s *SQLStore) CreateAuditLog(entry *types.AuditLog) (id string, err error) {
	// generating ID
	entry.ID = uuid.New().String()

	tx := s.db.Begin()
	// Note the use of tx as the database handle once you are within a transaction
	if err := tx.Create(entry).Error; err != nil {
		tx.Rollback()
		return "", err
	}

	tx.Commit()

	return entry.ID, nil
}

func (s *SQLStore) GetAuditLogs(query *types.AuditLogQuery) (logs []*types.AuditLog, err error) {
	if query.Offset == 0 {
		query.Offset = -1
	}

	if query.Limit == 0 {
		query.Limit = -1
	}

	switch query.Order {
	case "created_at desc", "created_at", "account", "identifier desc":
		// ok
	default:
		query.Order = "created_at desc"
	}

	if len(query.ResourceKindFilter) == 1 && query.ResourceKindFilter[0] == "*" {
		err = s.db.Order(query.Order).Limit(query.Limit).Offset(query.Offset).Find(&logs).Error
	} else if query.Username != "" {
		err = s.db.Order(query.Order).Where("resource_kind in (?)", query.ResourceKindFilter).Limit(query.Limit).Offset(query.Offset).Where("username = ?", query.Username).Find(&logs).Error
	} else {
		err = s.db.Order(query.Order).Where("resource_kind in (?)", query.ResourceKindFilter).Limit(query.Limit).Offset(query.Offset).Find(&logs).Error
	}

	return logs, err
}

func (s *SQLStore) AuditLogsCount(query *types.AuditLogQuery) (int, error) {
	var err error
	var count int

	if len(query.ResourceKindFilter) == 1 && query.ResourceKindFilter[0] == "*" {
		err = s.db.Model(&types.AuditLog{}).Count(&count).Error
	} else if query.Username != "" {
		err = s.db.Model(&types.AuditLog{}).Where("resource_kind in (?)", query.ResourceKindFilter).Where("username = ?", query.Username).Count(&count).Error
	} else {
		err = s.db.Model(&types.AuditLog{}).Where("resource_kind in (?)", query.ResourceKindFilter).Count(&count).Error
	}
	return count, err
}

var logsWeeklyStats = `SELECT day, COALESCE(updates, 0) AS updates, COALESCE(approved, 0) as approved
FROM  (SELECT ? - d AS day FROM generate_series (0, 6) d) d  -- 6, not 7
LEFT   JOIN (
   SELECT created_at AS day, count(*) AS updates 
   FROM   audit_logs AS l
   WHERE  l.created_at >= date_trunc('day', now()) - interval '6d' AND l.action = 'deployment update'
   GROUP  BY 1
   ) e USING (day)
LEFT   JOIN (
   SELECT created_at AS day, count(*) AS approved 
   FROM   audit_logs AS l
   WHERE  l.created_at >= date_trunc('day', now()) - interval '6d' AND l.action = 'approved'
   GROUP  BY 1
   ) b USING (day);`

// var logsWeeklyStats = `SELECT day, COALESCE(updates, 0) AS updates, COALESCE(approved, 0) as approved
// FROM  (SELECT now()::date - d AS day FROM generate_series (0, 6) d) d  -- 6, not 7
// LEFT   JOIN (
//    SELECT created_at::date AS day, count(*) AS updates
//    FROM   audit_logs AS l
//    WHERE  l.created_at >= date_trunc('day', now()) - interval '6d' AND l.action = 'deployment update'
//    GROUP  BY 1
//    ) e USING (day)
// LEFT   JOIN (
//    SELECT created_at::date AS day, count(*) AS approved
//    FROM   audit_logs AS l
//    WHERE  l.created_at >= date_trunc('day', now()) - interval '6d' AND l.action = 'approved'
//    GROUP  BY 1
//    ) b USING (day);`

const auditDays = 31

func (s *SQLStore) AuditStatistics(query *types.AuditLogStatsQuery) ([]types.AuditLogStats, error) {

	var logs []*types.AuditLog
	err := s.db.Order("created_at desc").
		Where("action in (?)", []string{"approved", "rejected", "deployment update", "release update"}).
		Where("created_at > ?", time.Now().Add(time.Hour*24*auditDays*-1)).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	getTime := func(day time.Time) string {
		return fmt.Sprintf("%d-%d-%d", day.Year(), day.Month(), day.Day())
	}

	// generate X days map of YYYY-MM-DD
	days := make(map[string]types.AuditLogStats)
	for i := 0; i < auditDays; i++ {
		day := getTime(time.Now().Add(time.Duration(-i) * time.Hour * 24))
		days[day] = types.AuditLogStats{Date: day}
	}

	for _, l := range logs {
		key := getTime(l.CreatedAt)
		switch l.Action {
		case types.NotificationDeploymentUpdate.String(), types.NotificationReleaseUpdate.String():
			entry, ok := days[key]
			if !ok {
				days[key] = types.AuditLogStats{
					Updates: 1,
				}
			}
			entry.Updates = entry.Updates + 1
			days[key] = entry
		case types.AuditActionApprovalApproved:
			entry := days[key]
			entry.Approved++
			days[key] = entry
		case types.AuditActionApprovalRejected:
			entry := days[key]
			entry.Rejected++
			days[key] = entry
		}

	}

	var stats []types.AuditLogStats

	for _, v := range days {
		stats = append(stats, v)
	}

	return stats, err

}
