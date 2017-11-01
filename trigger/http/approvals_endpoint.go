package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/keel-hq/keel/types"
)

func (s *TriggerServer) approvalsHandler(resp http.ResponseWriter, req *http.Request) {
	// unknown lists all
	approvals, err := s.approvalsManager.List()
	if err != nil {
		fmt.Fprintf(resp, "%s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(approvals) == 0 {
		approvals = make([]*types.Approval, 0)
	}

	bts, err := json.Marshal(&approvals)
	if err != nil {
		fmt.Fprintf(resp, "%s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Write(bts)
}

func (s *TriggerServer) approvalDeleteHandler(resp http.ResponseWriter, req *http.Request) {
	identifier := getID(req)

	err := s.approvalsManager.Delete(identifier)
	if err != nil {
		fmt.Fprintf(resp, "%s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	fmt.Fprintf(resp, identifier)
}
