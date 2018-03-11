package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/keel-hq/keel/cache"
	"github.com/keel-hq/keel/types"
)

type approveRequest struct {
	Identifier string `json:"identifier"`
	Voter      string `json:"voter"`
}

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
		fmt.Fprintf(resp, "%s", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if ar.Identifier == "" {
		http.Error(resp, "identifier not supplied", http.StatusBadRequest)
		return
	}

	approval, err := s.approvalsManager.Approve(ar.Identifier, ar.Voter)
	if err != nil {
		if err == cache.ErrNotFound {
			http.Error(resp, fmt.Sprintf("approval '%s' not found", ar.Identifier), http.StatusNotFound)
			return
		}

		fmt.Fprintf(resp, "%s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	bts, err := json.Marshal(&approval)
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
