package http

import (
	"net/http"

	"github.com/keel-hq/keel/types"
)

type dailyStats struct {
	Timestamp         int `json:"timestamp"`
	WebhooksReceived  int `json:"webhooksReceived"`
	ApprovalsApproved int `json:"approvalsApproved"`
	ApprovalsRejected int `json:"approvalsRejected"`
	Updates           int `json:"updates"`
}

func (s *TriggerServer) statsHandler(resp http.ResponseWriter, req *http.Request) {
	stats, err := s.store.AuditStatistics(&types.AuditLogStatsQuery{})
	response(stats, 200, err, resp, req)
}
