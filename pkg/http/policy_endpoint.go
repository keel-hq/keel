package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/keel-hq/keel/types"
)

// ResourcePolicyUpdateRequest changes the update policy for a resource.
type ResourcePolicyUpdateRequest struct {
	Policy     string `json:"policy"`
	Identifier string `json:"identifier"`
	Provider   string `json:"provider"`
}

// policyUpdateHandler changes a resource update policy.
// @Summary Update resource policy
// @Description Updates the policy annotation for a Kubernetes resource. An empty policy removes the policy configuration. This route exists only when the authenticator is enabled.
// @Tags Admin
// @ID updateResourcePolicy
// @Accept json
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Param body body ResourcePolicyUpdateRequest true "Policy update"
// @Success 200 {object} APIResponse
// @Failure 400 {string} string "Malformed request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Permission denied"
// @Failure 404 {string} string "Resource not found"
// @Failure 500 {string} string "Update failed"
// @Router /v1/policies [put]
func (s *TriggerServer) policyUpdateHandler(resp http.ResponseWriter, req *http.Request) {
	var policyRequest ResourcePolicyUpdateRequest
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()

	err := dec.Decode(&policyRequest)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "%s", err)
		return
	}

	if policyRequest.Identifier == "" {
		http.Error(resp, "identifier cannot be empty", http.StatusBadRequest)
		return
	}

	for _, v := range s.grc.Values() {
		if v.Identifier == policyRequest.Identifier {

			labels := v.GetLabels()
			delete(labels, types.KeelPolicyLabel)
			delete(labels, "keel.observer/policy")
			v.SetLabels(labels)

			ann := v.GetAnnotations()
			delete(ann, types.KeelPolicyLabel)
			delete(ann, "keel.observer/policy")
			if policyRequest.Policy != "" {
				ann[types.KeelPolicyLabel] = policyRequest.Policy
			}

			v.SetAnnotations(ann)

			err := s.kubernetesClient.Update(v)

			response(&APIResponse{Status: "updated"}, 200, err, resp, req)
			return
		}
	}

	resp.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(resp, "resource with identifier '%s' not found", policyRequest.Identifier)
	return
}
