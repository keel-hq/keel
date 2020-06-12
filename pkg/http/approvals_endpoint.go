package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/keel-hq/keel/pkg/store"
	"github.com/keel-hq/keel/types"
)

type approveRequest struct {
	ID         string `json:"id"`
	Voter      string `json:"voter"`
	Identifier string `json:"identifier"`
	Action     string `json:"action"` // defaults to approve
}

// available API actions
const (
	actionApprove = "approve"
	actionReject  = "reject"
	actionDelete  = "delete"
	actionArchive = "archive"
)

func (s *TriggerServer) approvalsHandler(resp http.ResponseWriter, req *http.Request) {

	// lists all (both archived)
	approvals, err := s.store.ListApprovals(&types.GetApprovalQuery{})
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

type resourceApprovalsUpdateRequest struct {
	Identifier    string `json:"identifier"`
	Provider      string `json:"provider"`
	VotesRequired int    `json:"votesRequired"`
}

// approvalSetHandler allows to set/remove approvals for resources
func (s *TriggerServer) approvalSetHandler(resp http.ResponseWriter, req *http.Request) {

	var approvalUpdateRequest resourceApprovalsUpdateRequest
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()

	err := dec.Decode(&approvalUpdateRequest)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "%s", err)
		return
	}

	if approvalUpdateRequest.VotesRequired < 0 || approvalUpdateRequest.VotesRequired > 100 {
		http.Error(resp, "votesRequired should be between 0 and 100", http.StatusBadRequest)
		return
	}

	switch approvalUpdateRequest.Provider {
	case types.ProviderTypeKubernetes.String():
		// ok
	default:
		http.Error(resp, "unsupported provider", http.StatusBadRequest)
		return
	}

	if approvalUpdateRequest.Identifier == "" {
		http.Error(resp, "identifier cannot be empty", http.StatusBadRequest)
		return
	}

	for _, v := range s.grc.Values() {
		if v.Identifier == approvalUpdateRequest.Identifier {

			labels := v.GetLabels()
			delete(labels, types.KeelMinimumApprovalsLabel)
			v.SetLabels(labels)

			ann := v.GetAnnotations()
			ann[types.KeelMinimumApprovalsLabel] = strconv.Itoa(approvalUpdateRequest.VotesRequired)

			v.SetAnnotations(ann)

			err := s.kubernetesClient.Update(v)

			response(&APIResponse{Status: "updated"}, 200, err, resp, req)
			return
		}
	}

	resp.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(resp, "resource with identifier '%s' not found", approvalUpdateRequest.Identifier)
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

	if ar.Identifier == "" {
		http.Error(resp, "identifier cannot be empty", http.StatusNotFound)
		return
	}

	var approval *types.Approval

	// checking action
	switch ar.Action {
	case actionReject:
		approval, err = s.approvalsManager.Reject(ar.Identifier)
		if err != nil {
			if err == store.ErrRecordNotFound {
				http.Error(resp, fmt.Sprintf("approval '%s' not found", ar.Identifier), http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(resp, "%s", err)
			return
		}
	case actionDelete:
		if ar.Identifier != "" && ar.ID == "" {
			existing, err := s.approvalsManager.Get(ar.Identifier)
			if err == nil {
				ar.ID = existing.ID
			}
		}
		// deleting it
		err := s.approvalsManager.Delete(&types.Approval{
			ID: ar.ID,
		})
		if err != nil {
			fmt.Fprintf(resp, "%s", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	case actionArchive:
		approval, err = s.approvalsManager.Get(ar.Identifier)
		if err != nil {
			if err == store.ErrRecordNotFound {
				http.Error(resp, fmt.Sprintf("approval '%s' not found", ar.Identifier), http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(resp, "%s", err)
			return
		}

		approval.Archived = true

		// deleting it
		err := s.approvalsManager.Archive(ar.Identifier)
		if err != nil {
			fmt.Fprintf(resp, "%s", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

	default:
		// "" or "approve"
		approval, err = s.approvalsManager.Approve(ar.Identifier, ar.Voter)
		if err != nil {
			if err == store.ErrRecordNotFound {
				http.Error(resp, fmt.Sprintf("approval '%s' not found", ar.Identifier), http.StatusNotFound)
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
