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

// statsHandler returns audit statistics.
// @Summary Get audit statistics
// @Description Returns daily webhook, approval, rejection, and update counts. This route exists only when the authenticator is enabled.
// @Tags Admin
// @ID getStats
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Success 200 {array} types.AuditLogStats
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Permission denied"
// @Failure 500 {string} string "Store query failed"
// @Router /v1/stats [get]
func (s *TriggerServer) statsHandler(resp http.ResponseWriter, req *http.Request) {
	stats, err := s.store.AuditStatistics(&types.AuditLogStatsQuery{})
	response(stats, 200, err, resp, req)
}
