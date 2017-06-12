package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

// nativeHandler - used to trigger event directly
func (s *TriggerServer) nativeHandler(resp http.ResponseWriter, req *http.Request) {
	event := types.Event{}
	if err := json.NewDecoder(req.Body).Decode(&event); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	event.CreatedAt = time.Now()

	for _, p := range s.providers {
		err := p.Submit(event)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"provider": p.GetName(),
			}).Error("trigger.webhook: got error while submitting event to provider")
		}
	}

	resp.WriteHeader(http.StatusOK)
	return
}
