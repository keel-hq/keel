package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

var newHarborWebhooksCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "harbor_webhook_requests_total",
		Help: "How many /v1/webhooks/harbor requests processed, partitioned by image.",
	},
	[]string{"image"},
)

func init() {
	prometheus.MustRegister(newHarborWebhooksCounter)
}

// Example of Harbor trigger
// {
//     "type": "pushImage",
//     "occur_at": 1582640688,
//     "operator": "<user>",
//     "event_data": {
//         "resources": [
//             {
//                 "digest": "sha256:b4758aaed11c155a476b9857e1178f157759c99cb04c907a04993f5481eff848",
//                 "tag": "2.1.6",
//                 "resource_url": "<url>/<namespace>/<repo>:<version>"
//             }
//         ],
//         "repository": {
//             "date_created": 1582634337,
//             "name": "<repo>",
//             "namespace": "<namespace>",
//             "repo_full_name": "<namespace>/<repo>",
//             "repo_type": "private"
//         }
//     }

type harborWebhook struct {
	Type      string `json:"type"`
	OccurAt   int    `json:"occur_at"`
	Operator  string `json:"operator"`
	EventData struct {
		Resources []struct {
			Digest      string `json:"digest"`
			Tag         string `json:"tag"`
			ResourceURL string `json:"resource_url"`
		} `json:"resources"`
		Repository struct {
			DateCreated  int    `json:"date_created"`
			Name         string `json:"name"`
			Namespace    string `json:"namespace"`
			RepoFullName string `json:"repo_full_name"`
			RepoType     string `json:"repo_type"`
		} `json:"repository"`
	} `json:"event_data"`
}

func (s *TriggerServer) harborHandler(resp http.ResponseWriter, req *http.Request) {
	hn := harborWebhook{}
	if err := json.NewDecoder(req.Body).Decode(&hn); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.harborHandler: failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	log.WithFields(log.Fields{
		"event": hn,
	}).Debug("harborHandler: received event, looking for a pushImage tag")

	if hn.Type == "pushImage" || hn.Type == "PUSH_ARTIFACT" { 
		// go trough all the ressource items
		for _, e := range hn.EventData.Resources {
			imageRepo, err := image.Parse(e.ResourceURL)
			if err != nil {
				log.WithFields(log.Fields{
					"error":      err,
					"repository": e.ResourceURL,
				}).Error("trigger.harborHandler: failed to parse repository")

				resp.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(resp, "failed to parse repository %s, error: %s", e.ResourceURL, err)
				return
			}

			//create event
			event := types.Event{}
			event.CreatedAt = time.Now()
			event.TriggerName = "harbor"
			event.Repository.Name = imageRepo.Repository()
			event.Repository.Tag = imageRepo.Tag()

			log.WithFields(log.Fields{
				"action":     hn.Type,
				"tag":        imageRepo.Tag(),
				"repository": imageRepo.Repository(),
				"digest":     e.Digest,
			}).Debug("harborHandler: got registry notification, processing")

			s.trigger(event)
			newHarborWebhooksCounter.With(prometheus.Labels{"image": event.Repository.Name}).Inc()
		}
	}

	resp.WriteHeader(http.StatusOK)
}
