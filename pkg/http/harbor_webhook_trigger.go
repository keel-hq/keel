package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"regexp"

	"github.com/keel-hq/keel/types"
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
}

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
	qw := harborWebhook{}
	if err := json.NewDecoder(req.Body).Decode(&qw); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.harborHandler: failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if qw.type == "pushImage" {
		if qw.ResourceURL == "" {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "ResourceURL cannot be empty")
			return
		}
		//Split the combined <URL>:<tag> into seperate fields
		split_regexp := regexp.MustCompile("(.*):(.*)")
		split_string := split_regexp.FindAllStringSubmatch(qw.ResourceURL,-1)
		DockerURL    := split_string[0][1]
		tag          := split_string[0][2]
		
		if len(DockerURL) == 0 {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "DockerURL cannot be empty")
			return
		}
		
		if len(tag) == 0 {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "tags cannot be empty")
			return
		}

	    //create event		
		event := types.Event{}
		event.CreatedAt = time.Now()
		event.TriggerName = "harbor"
		event.Repository.Name = DockerURL
		event.Repository.Tag = tag

		s.trigger(event)
		newHarborWebhooksCounter.With(prometheus.Labels{"image": event.Repository.Name}).Inc()
		
	}

	resp.WriteHeader(http.StatusOK)
	return
}
