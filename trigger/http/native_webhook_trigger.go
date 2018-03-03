package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// nativeHandler - used to trigger event directly
func (s *TriggerServer) nativeHandler(resp http.ResponseWriter, req *http.Request) {
	repo := types.Repository{}
	if err := json.NewDecoder(req.Body).Decode(&repo); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	event := types.Event{}

	if repo.Name == "" {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "repository name cannot be empty")
		return
	}

	if repo.Tag == "" {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "repository tag cannot be empty")
		return
	}

	event.Repository = repo
	event.CreatedAt = time.Now()
	event.TriggerName = "native"
	s.trigger(event)

	resp.WriteHeader(http.StatusOK)
	return
}
