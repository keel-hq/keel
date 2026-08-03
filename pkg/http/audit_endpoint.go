package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/keel-hq/keel/types"
)

// adminAuditLogHandler returns a page of audit logs.
// @Summary List audit logs
// @Description Lists audit entries. Invalid limit or offset values are silently treated as zero. This route exists only when the authenticator is enabled.
// @Tags Admin
// @ID listAuditLogs
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Param limit query int false "Maximum entries"
// @Param offset query int false "Entry offset"
// @Param filter query string false "Comma-separated resource kinds"
// @Param email query string false "Account email"
// @Success 200 {object} AuditLogsResponse
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Permission denied"
// @Failure 500 {string} string "Store query failed"
// @Router /v1/audit [get]
func (s *TriggerServer) adminAuditLogHandler(resp http.ResponseWriter, req *http.Request) {

	query := &types.AuditLogQuery{}
	limitS := req.URL.Query().Get("limit")
	if limitS != "" {
		l, err := strconv.Atoi(limitS)
		if err == nil {
			query.Limit = l
		}
	}

	offsetS := req.URL.Query().Get("offset")
	if offsetS != "" {
		o, err := strconv.Atoi(offsetS)
		if err == nil {
			query.Offset = o
		}
	}

	kindFilter := req.URL.Query().Get("filter")
	if kindFilter != "" {
		kinds := strings.Split(kindFilter, ",")
		query.ResourceKindFilter = kinds
	}

	emailFilter := req.URL.Query().Get("email")
	if emailFilter != "" {
		query.Email = strings.TrimSpace(emailFilter)
	}

	entries, err := s.store.GetAuditLogs(query)
	if err != nil {
		response(nil, 500, err, resp, req)
		return
	}

	result := AuditLogsResponse{
		Data:   entries,
		Offset: query.Offset,
		Limit:  query.Limit,
	}

	count, err := s.store.AuditLogsCount(query)
	if err == nil {
		result.Total = count
	}

	response(result, http.StatusOK, err, resp, req)
}

// AuditLogsResponse is a page of audit log entries.
type AuditLogsResponse struct {
	Data   []*types.AuditLog `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}
