package http

import (
	"encoding/json"
	"fmt"
	"github.com/keel-hq/keel/types"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var newGithubWebhooksCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "Github_webhook_requests_total",
		Help: "How many /v1/webhooks/github requests processed, partitioned by image.",
	},
	[]string{"image"},
)

func init() {
	prometheus.MustRegister(newGithubWebhooksCounter)
}

// GitHubWebhook documents the two payload variants selected by X-GitHub-Event.
type GitHubWebhook struct {
	Action          string                 `json:"action"`
	RegistryPackage *GitHubRegistryPackage `json:"registry_package,omitempty"`
	Repository      *GitHubRepository      `json:"repository,omitempty"`
	Package         *GitHubPackage         `json:"package,omitempty"`
}

// GitHubRegistryPackageWebhook is the registry_package event payload.
type GitHubRegistryPackageWebhook struct {
	Action          string                `json:"action"`
	RegistryPackage GitHubRegistryPackage `json:"registry_package"`
	Repository      GitHubRepository      `json:"repository"`
}

// GitHubRegistryPackage describes a GitHub Packages container version.
type GitHubRegistryPackage struct {
	Name           string                       `json:"name"`
	PackageType    string                       `json:"package_type"`
	PackageVersion GitHubRegistryPackageVersion `json:"package_version"`
	UpdatedAt      string                       `json:"updated_at"`
}

// GitHubRegistryPackageVersion identifies the published version.
type GitHubRegistryPackageVersion struct {
	Version string `json:"version"`
}

// GitHubRepository identifies the repository for a registry_package event.
type GitHubRepository struct {
	FullName string `json:"full_name"`
}

// GitHubPackageV2Webhook is the package event payload used by GHCR.
type GitHubPackageV2Webhook struct {
	Action  string        `json:"action"`
	Package GitHubPackage `json:"package"`
}

// GitHubPackage describes a GHCR package.
type GitHubPackage struct {
	ID             int                  `json:"id"`
	Name           string               `json:"name"`
	Namespace      string               `json:"namespace"`
	Ecosystem      string               `json:"ecosystem"`
	PackageVersion GitHubPackageVersion `json:"package_version"`
}

// GitHubPackageVersion contains GHCR container metadata.
type GitHubPackageVersion struct {
	Name              string                  `json:"name"`
	ContainerMetadata GitHubContainerMetadata `json:"container_metadata"`
}

// GitHubContainerMetadata contains the published tag.
type GitHubContainerMetadata struct {
	Tag GitHubContainerTag `json:"tag"`
}

// GitHubContainerTag identifies a GHCR tag and digest.
type GitHubContainerTag struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

// githubHandler - used to react to github webhooks
// githubHandler accepts GitHub Packages and GHCR pushes.
// @Summary Receive a GitHub webhook
// @Description X-GitHub-Event selects package or registry_package decoding; absent and other values currently submit an empty event and return 200. Requires Basic or Bearer authorization only when authenticatedWebhooks is enabled.
// @Tags Webhooks
// @ID receiveGitHubWebhook
// @Accept json
// @Security BasicAuth
// @Security BearerAuth
// @Param X-GitHub-Event header string false "Payload type: package or registry_package"
// @Param body body GitHubWebhook true "GitHub Packages or GHCR push"
// @Success 200 "Accepted"
// @Failure 400 {string} string "Malformed or incomplete supported payload"
// @Failure 401 {string} string "Unauthorized when authenticated webhooks are enabled"
// @Router /v1/webhooks/github [post]
func (s *TriggerServer) githubHandler(resp http.ResponseWriter, req *http.Request) {
	// GitHub provides different webhook events for each registry.
	// Github Package uses 'registry_package'
	// Github Container Registry uses 'package_v2'
	// events can be classified as 'X-GitHub-Event' in Request Header.
	hookEvent := req.Header.Get("X-GitHub-Event")

	var imageName, imageTag string

	switch hookEvent {
	case "package":
		payload := new(GitHubPackageV2Webhook)
		if err := json.NewDecoder(req.Body).Decode(payload); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("trigger.githubHandler: failed to decode request")
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		if payload.Package.Ecosystem != "CONTAINER" {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "registry package type was not container")
		}

		if payload.Package.Name == "" { // github package name
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "repository name cannot be empty")
			return
		}

		if payload.Package.Namespace == "" { // github package org
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "repository namespace cannot be empty")
			return
		}

		if payload.Package.PackageVersion.ContainerMetadata.Tag.Name == "" { // tag
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "repository tag cannot be empty")
			return
		}

		imageName = strings.Join(
			[]string{"ghcr.io", payload.Package.Namespace, payload.Package.Name},
			"/",
		)
		imageTag = payload.Package.PackageVersion.ContainerMetadata.Tag.Name

		break

	case "registry_package":
		payload := new(GitHubRegistryPackageWebhook)
		if err := json.NewDecoder(req.Body).Decode(payload); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("trigger.githubHandler: failed to decode request")
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		if payload.RegistryPackage.PackageType != "CONTAINER" {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "registry package type was not CONTAINER")
		}

		if payload.Repository.FullName == "" { // github package name
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "repository full name cannot be empty")
			return
		}

		if payload.RegistryPackage.Name == "" { // github package name
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "repository package name cannot be empty")
			return
		}

		if payload.RegistryPackage.PackageVersion.Version == "" { // tag
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(resp, "repository tag cannot be empty")
			return
		}

		// XXX <jsonroot>.registry_package.package_version.package_url could work too but it ends with colon
		imageName = strings.Join(
			[]string{"ghcr.io", payload.Repository.FullName},
			"/",
		)
		imageTag = payload.RegistryPackage.PackageVersion.Version

		break
	}

	event := types.Event{}
	event.CreatedAt = time.Now()
	event.TriggerName = "github"
	event.Repository.Name = imageName
	event.Repository.Tag = imageTag

	s.trigger(event)

	resp.WriteHeader(http.StatusOK)

	newGithubWebhooksCounter.With(prometheus.Labels{"image": event.Repository.Name}).Inc()
}
