package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/keel-hq/keel/types"
)

type resourcePolicyUpdateRequest struct {
	Policy     string `json:"policy"`
	Identifier string `json:"identifier"`
	Provider   string `json:"provider"`
}

func getIdentifier(req *http.Request) string {
	return mux.Vars(req)["identifier"]
}

func (s *TriggerServer) policyUpdateHandler(resp http.ResponseWriter, req *http.Request) {
	var policyRequest resourcePolicyUpdateRequest
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

			ann := v.GetAnnotations()
			ann[types.KeelPolicyLabel] = policyRequest.Policy

			v.SetAnnotations(ann)

			err := s.kubernetesClient.Update(v)
			// if err != nil {
			// 	resp.WriteHeader(http.StatusInternalServerError)
			// 	fmt.Fprintf(resp, "%s", err)
			// 	return
			// }
			response(&APIResponse{Status: "updated"}, 200, err, resp, req)
			return
		}
	}

	resp.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(resp, "resource with identifier '%s' not found", policyRequest.Identifier)
	return
}
