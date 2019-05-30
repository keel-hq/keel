package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/keel-hq/keel/types"

	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

var newRegistryNotificationWebhooksCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "registry_notification_requests_total",
		Help: "How many /v1/webhooks/registry requests processed, partitioned by image.",
	},
	[]string{"image"},
)

func init() {
	prometheus.MustRegister(newRegistryNotificationWebhooksCounter)
}

// {
// 	"events": [
// 	   {
// 		  "id": "d83e8796-7ba5-46ad-b239-d88473e21b2b",
// 		  "timestamp": "2018-10-11T13:53:21.859222576Z",
// 		  "action": "push",
// 		  "target": {
// 			 "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
// 			 "size": 2206,
// 			 "digest": "sha256:4afff550708506c5b8b7384ad10d401a02b29ed587cb2730cb02753095b5178d",
// 			 "length": 2206,
// 			 "repository": "shinebayar-g/a",
// 			 "url": "https://registry.git.erxes.io/v2/shinebayar-g/a/manifests/sha256:4afff550708506c5b8b7384ad10d401a02b29ed587cb2730cb02753095b5178d",
// 			 "tag": "0.01"
// 		  },
// 		  "request": {
// 			 "id": "18690582-6d1a-4e08-8825-251a0adc58ce",
// 			 "addr": "46.101.177.27",
// 			 "host": "registry.git.erxes.io",
// 			 "method": "PUT",
// 			 "useragent": "docker/18.06.1-ce go/go1.10.3 git-commit/e68fc7a kernel/4.4.0-135-generic os/linux arch/amd64 UpstreamClient(Docker-Client/18.06.1-ce \\(linux\\))"
// 		  },
// 		  "actor": {
// 			 "name": "shinebayar-g"
// 		  },
// 		  "source": {
// 			 "addr": "git.erxes.io:5000",
// 			 "instanceID": "bde27723-d67e-4775-a9bd-55f771a2f895"
// 		  }
// 	   }
// 	]
//  }

type registryNotification struct {
	Events []struct {
		ID        string    `json:"id"`
		Timestamp time.Time `json:"timestamp"`
		Action    string    `json:"action"`
		Target    struct {
			MediaType  string `json:"mediaType"`
			Size       int    `json:"size"`
			Digest     string `json:"digest"`
			Length     int    `json:"length"`
			Repository string `json:"repository"`
			URL        string `json:"url"`
			Tag        string `json:"tag"`
		} `json:"target"`
		Request struct {
			ID        string `json:"id"`
			Addr      string `json:"addr"`
			Host      string `json:"host"`
			Method    string `json:"method"`
			Useragent string `json:"useragent"`
		} `json:"request"`
		Actor struct {
			Name string `json:"name"`
		} `json:"actor"`
		Source struct {
			Addr       string `json:"addr"`
			InstanceID string `json:"instanceID"`
		} `json:"source"`
	} `json:"events"`
}

func (s *TriggerServer) registryNotificationHandler(resp http.ResponseWriter, req *http.Request) {
	rn := registryNotification{}
	if err := json.NewDecoder(req.Body).Decode(&rn); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.dockerHubHandler: failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"event": rn,
	}).Debug("registryNotificationHandler: received event, looking for a push tag")

	for _, e := range rn.Events {

		if e.Action != "push" {
			continue
		}

		if e.Target.Tag == "" {
			continue
		}

		dockerURL := e.Request.Host + "/" + e.Target.Repository

		event := types.Event{}
		event.Repository.Name = dockerURL
		event.CreatedAt = time.Now()
		event.TriggerName = "registry-notification"
		event.Repository.Tag = e.Target.Tag
		event.Repository.Digest = e.Target.Digest

		log.WithFields(log.Fields{
			"action":     e.Action,
			"tag":        e.Target.Tag,
			"repository": dockerURL,
			"digest":     e.Target.Digest,
		}).Debug("registryNotificationHandler: got registry notification, processing")

		s.trigger(event)

		newRegistryNotificationWebhooksCounter.With(prometheus.Labels{"image": event.Repository.Name}).Inc()
	}

}
