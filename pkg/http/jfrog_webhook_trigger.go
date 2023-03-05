package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/keel-hq/keel/types"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

const (
	EnvPrivateRegistry = "PRIVATE_REGISTRY"
)

var newJfrogWebhooksCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "jfrog_webhook_requests_total",
		Help: "How many /v1/webhooks/jfrog requests processed, partitioned by image.",
	},
	[]string{"image"},
)

func init() {
	prometheus.MustRegister(newJfrogWebhooksCounter)
}

/** Example of jfrog trigger
{
   "domain": "docker",
   "event_type": "pushed",
   "data": {
     "repo_key":"docker-remote-cache",
     "event_type":"pushed",
     "path":"library/ubuntu/latest/list.manifest.json",
     "name":"list.manifest.json",
     "sha256":"35c4a2c15539c6c1e4e5fa4e554dac323ad0107d8eb5c582d6ff386b383b7dce",
     "size":1206,
     "image_name":"library/ubuntu",
     "tag":"latest",
     "platforms":[
        {
           "architecture":"amd64",
           "os":"linux"
        },
        {
           "architecture":"arm",
           "os":"linux"
        },
        {
           "architecture":"arm64",
           "os":"linux"
        },
        {
           "architecture":"ppc64le",
           "os":"linux"
        },
        {
           "architecture":"s390x",
           "os":"linux"
      }
    ]
  },
  "subscription_key": "test",
  "jpd_origin": "https://example.jfrog.io",
  "source": "jfrog/user@example.com"
}
**/

type jfrogWebhook struct {
	Domain    string `json:"domain"`
	EventType string `json:"event_type"`
	Data      struct {
		RepoKey   string `json:"repo_key"`
		Path      string `json:"path"`
		Name      string `json:"name"`
		Sha256    string `json:"sha256"`
		Size      int32  `json:"size"`
		ImageName string `json:"image_name"`
		Tag       string `json:"tag"`
		Platforms []struct {
			Architecture string `json:"architecture"`
			Os           string `json:"os"`
		}
	}
	SubscriptionKey string `json:"subscription_key"`
	JpdOrigin       string `json:"jpd_origin"`
	Source          string `json:"source"`
}

func (s *TriggerServer) jfrogHandler(resp http.ResponseWriter, req *http.Request) {
	jw := jfrogWebhook{}
	if err := json.NewDecoder(req.Body).Decode(&jw); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.jfrogHandler: failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if jw.Data.ImageName == "" {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "data.image_name cannot be empty")
		return
	}

	if len(jw.Data.Tag) == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "tag cannot be empty")
		return
	}

	// for every updated tag generating event
	event := types.Event{}
	event.CreatedAt = time.Now()
	event.TriggerName = "jfrog"
	event.Repository.Tag = jw.Data.Tag
	event.Repository.Name = jw.Data.ImageName
	if privReg, ok := os.LookupEnv(EnvPrivateRegistry); ok {
		if len(privReg) >= 3 {
			event.Repository.Name = fmt.Sprintf("%s/%s", privReg, jw.Data.ImageName)
		}
	}

	log.Infof("Received jfrog webhook for image: %s:%s", jw.Data.ImageName, jw.Data.Tag)
	log.Debug("jfrogWebhook data: ", jw)
	s.trigger(event)
	newJfrogWebhooksCounter.With(prometheus.Labels{"image": event.Repository.Name}).Inc()

	resp.WriteHeader(http.StatusOK)
	return
}
