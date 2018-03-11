package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/keel-hq/keel/cache"
	"github.com/keel-hq/keel/types"
)

type approveRequest struct {
	Voter  string `json:"voter"`
	Action string `json:"action"` // defaults to approve
}

// available API actions
const (
	actionApprove = "approve"
	actionReject  = "reject"
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

func (s *TriggerServer) approvalApproveHandler(resp http.ResponseWriter, req *http.Request) {

	var ar approveRequest
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()

	err := dec.Decode(&ar)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "%s", err)
		return
	}

	var approval *types.Approval

	// checking action
	switch ar.Action {
	case actionReject:
		approval, err = s.approvalsManager.Reject(getID(req))
		if err != nil {
			if err == cache.ErrNotFound {
				http.Error(resp, fmt.Sprintf("approval '%s' not found", getID(req)), http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(resp, "%s", err)
			return
		}

	default:
		// "" or "approve"
		approval, err = s.approvalsManager.Approve(getID(req), ar.Voter)
		if err != nil {
			if err == cache.ErrNotFound {
				http.Error(resp, fmt.Sprintf("approval '%s' not found", getID(req)), http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(resp, "%s", err)
			return
		}
	}

	bts, err := json.Marshal(&approval)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(resp, "%s", err)
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
