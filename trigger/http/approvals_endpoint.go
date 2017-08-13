package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rusenask/keel/types"
)

func (s *TriggerServer) approvalsHandler(resp http.ResponseWriter, req *http.Request) {
	// unknown lists all
	approvals, err := s.approvalsManager.List(types.ProviderTypeUnknown)
	if err != nil {
		fmt.Fprintf(resp, "%s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	bts, err := json.Marshal(&approvals)
	if err != nil {
		fmt.Fprintf(resp, "%s", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Write(bts)
}
