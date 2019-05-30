package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/keel-hq/keel/types"
)

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

	result := auditLogsResponse{
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

type auditLogsResponse struct {
	Data   []*types.AuditLog `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}
