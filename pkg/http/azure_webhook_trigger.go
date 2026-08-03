package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/keel-hq/keel/types"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

var newAzureWebhooksCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "azure_webhook_requests_total",
		Help: "How many /v1/webhooks/azure requests processed, partitioned by image.",
	},
	[]string{"image"},
)

func init() {
	prometheus.MustRegister(newAzureWebhooksCounter)
}

// Example of azure trigger
// {
//  "id": "cb8c3971-9adc-488b-bdd8-43cbb4974ff5",
//  "timestamp": "2017-11-17T16:52:01.343145347Z",
//  "action": "push",
//  "target": {
//    "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
//    "size": 524,
//    "digest": "sha256:80f0d5c8786bb9e621a45ece0db56d11cdc624ad20da9fe62e9d25490f331d7d",
//    "length": 524,
//    "repository": "hello-world",
//    "tag": "v1"
//  },
//  "request": {
//    "id": "3cbb6949-7549-4fa1-86cd-a6d5451dffc7",
//    "host": "myregistry.azurecr.io",
//    "method": "PUT",
//    "useragent": "docker/17.09.0-ce go/go1.8.3 git-commit/afdb6d4 kernel/4.10.0-27-generic os/linux arch/amd64 UpstreamClient(Docker-Client/17.09.0-ce \\(linux\\))"
//  }
//}

// AzureWebhook is the Azure Container Registry push webhook payload.
type AzureWebhook struct {
	Target  AzureWebhookTarget  `json:"target"`
	Request AzureWebhookRequest `json:"request"`
}

// AzureWebhookTarget identifies the pushed manifest.
type AzureWebhookTarget struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Digest     string `json:"digest"`
}

// AzureWebhookRequest identifies the registry host.
type AzureWebhookRequest struct {
	Host string `json:"host"`
}

// azureHandler accepts Azure Container Registry pushes.
// @Summary Receive an Azure webhook
// @Description Requires Basic or Bearer authorization only when authenticatedWebhooks is enabled.
// @Tags Webhooks
// @ID receiveAzureWebhook
// @Accept json
// @Security BasicAuth
// @Security BearerAuth
// @Param body body AzureWebhook true "Azure Container Registry push"
// @Success 200 "Accepted"
// @Failure 400 {string} string "Malformed payload or missing tag"
// @Failure 401 {string} string "Unauthorized when authenticated webhooks are enabled"
// @Router /v1/webhooks/azure [post]
func (s *TriggerServer) azureHandler(resp http.ResponseWriter, req *http.Request) {
	aw := AzureWebhook{}
	if err := json.NewDecoder(req.Body).Decode(&aw); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.azureHandler: failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if aw.Target.Tag == "" {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "tag cannot be empty")
		return
	}

	// for every updated tag generating event
	var DockerURL = aw.Request.Host + "/" + aw.Target.Repository
	event := types.Event{}
	event.CreatedAt = time.Now()
	event.TriggerName = "azure"
	event.Repository.Name = DockerURL // need to build this url..
	event.Repository.Tag = aw.Target.Tag
	event.Repository.Digest = aw.Target.Digest
	s.trigger(event)
	newAzureWebhooksCounter.With(prometheus.Labels{"image": event.Repository.Name}).Inc()

	resp.WriteHeader(http.StatusOK)
	return
}
