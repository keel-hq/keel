package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/keel-hq/keel/types"

	log "github.com/sirupsen/logrus"
)

// Example of dockerhub trigger
// {
// 	"push_data": {
// 		"pushed_at": 1497467660,
// 		"images": [],
// 		"tag": "0.1.7",
// 		"pusher": "karolisr"
// 	},
// 	"callback_url": "https://registry.hub.docker.com/u/karolisr/keel/hook/22hagb51h1gfb4eefc5f1g4j3abi0beg4/",
// 	"repository": {
// 		"status": "Active",
// 		"description": "",
// 		"is_trusted": false,
// 		"full_description": "# Keel - automated Kubernetes deployments for the rest of us\n\nLightweight (11MB image size, uses 12MB RAM when running) [Kubernetes](https://kubernetes.io/) controller for automating image updates for deployments. Keel uses [semantic versioning](http://semver.org/) to determine whether deployment needs an update or not. Currently keel has several types of triggers:\n\n* Google's pubsub integration with [Google Container Registry](https://cloud.google.com/container-registry/)\n* Webhooks\n\nUpcomming integrations:\n\n* DockerHub webhooks\n\n## Why?\n\nI have built Keel since I have a relatively small Golang project which doesn't use a lot of memory and introducing an antique, heavy weight CI solution with lots dependencies seemed like a terrible idea. \n\nYou should consider using Keel:\n* If you are not Netflix, Google, Amazon, {insert big company here} - you might not want to run something like Spinnaker that has heavy dependencies such as \"JDK8, Redis, Cassandra, Packer\". You probably need something lightweight, stateless, that you don't have to think about.\n* If you are not a bank that uses RedHat's OpenShift which embedded Jenkins that probably already does what Keel is doing.\n* You want automated Kubernetes deployment updates.\n\nHere is a list of Keel dependencies:\n\n1.\n\nYes, none.\n\n## Getting started\n\nKeel operates as a background service, you don't need to interact with it directly, just add labels to your deployments. \n\n### Example deployment\n\nHere is an example deployment which specifies that keel should always update image if a new version is available:\n\n```\n---\napiVersion: extensions/v1beta1\nkind: Deployment\nmetadata: \n name: wd\n namespace: default\n labels: \n name: \"wd\"\n keel.observer/policy: all\nspec:\n replicas: 1\n template:\n metadata:\n name: wd\n labels:\n app: wd \n\n spec:\n containers: \n - image: karolisr/webhook-demo:0.0.2\n imagePullPolicy: Always \n name: wd\n command: [\"/bin/webhook-demo\"]\n ports:\n - containerPort: 8090 \n livenessProbe:\n httpGet:\n path: /healthz\n port: 8090\n initialDelaySeconds: 30\n timeoutSeconds: 10\n securityContext:\n privileged: true \n```\n\nAvailable policy options:\n\n* all - update whenever there is a version bump\n* major - update major versions\n* minor - update only minor versions (ignores major)\n* patch - update only patch versions (ignores minor and major versions)\n\n## Deployment\n\n### Step 1: GCE Kubernetes + GCR pubsub configuration\n\nSince access to pubsub is required in GCE Kubernetes - your cluster node pools need to have permissions. If you are creating new cluster - just enable pubsub from the start. If you have existing cluster - currently the only way is create new node-pool through the gcloud CLI (more info in the [docs](https://cloud.google.com/sdk/gcloud/reference/container/node-pools/create?hl=en_US&_ga=1.2114551.650086469.1487625651):\n\n```\ngcloud container node-pools create new-pool --cluster CLUSTER_NAME --scopes https://www.googleapis.com/auth/pubsub\n``` \n\n### Step 2: Kubernetes\n\nSince keel will be updating deployments, let's create a new [service account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/) in `kube-system` namespace:\n\n```\nkubectl create serviceaccount keel --namespace=kube-system\n```\n\nNow, edit [deployment file](https://github.com/keel-hq/keel/blob/master/hack/deployment.sample.yml) that is supplied with the repo (basically point to the newest keel release and set your PROJECT_ID to the actual project ID that you have):\n\n```\nkubectl create -f hack/deployment.yml\n```\n\nOnce Keel is deployed in your Kubernetes cluster - it occasionally scans your current deployments and looks for ones that have label _keel.observer/policy_. It then checks whether appropriate subscriptions and topics are set for GCR registries, if not - auto-creates them.\n\n",
// 		"repo_url": "https://hub.docker.com/r/karolisr/keel",
// 		"owner": "karolisr",
// 		"is_official": false,
// 		"is_private": false,
// 		"name": "keel",
// 		"namespace": "karolisr",
// 		"star_count": 0,
// 		"comment_count": 0,
// 		"date_created": 1497032538,
// 		"dockerfile": "FROM golang:1.8.1-alpine\nCOPY . /go/src/github.com/keel-hq/keel\nWORKDIR /go/src/github.com/keel-hq/keel\nRUN apk add --no-cache git && go get\nRUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags -'w' -o keel .\n\nFROM alpine:latest\nRUN apk --no-cache add ca-certificates\nCOPY --from=0 /go/src/github.com/keel-hq/keel/keel /bin/keel\nENTRYPOINT [\"/bin/keel\"]\n\nEXPOSE 9300",
// 		"repo_name": "karolisr/keel"
// 	}
// }

type dockerHubWebhook struct {
	PushData struct {
		PushedAt int           `json:"pushed_at"`
		Images   []interface{} `json:"images"`
		Tag      string        `json:"tag"`
		Pusher   string        `json:"pusher"`
	} `json:"push_data"`
	CallbackURL string `json:"callback_url"`
	Repository  struct {
		Status          string `json:"status"`
		Description     string `json:"description"`
		IsTrusted       bool   `json:"is_trusted"`
		FullDescription string `json:"full_description"`
		RepoURL         string `json:"repo_url"`
		Owner           string `json:"owner"`
		IsOfficial      bool   `json:"is_official"`
		IsPrivate       bool   `json:"is_private"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		StarCount       int    `json:"star_count"`
		CommentCount    int    `json:"comment_count"`
		DateCreated     int    `json:"date_created"`
		Dockerfile      string `json:"dockerfile"`
		RepoName        string `json:"repo_name"`
	} `json:"repository"`
}

// dockerHubHandler - used to react to dockerhub webhooks
func (s *TriggerServer) dockerHubHandler(resp http.ResponseWriter, req *http.Request) {
	dw := dockerHubWebhook{}
	if err := json.NewDecoder(req.Body).Decode(&dw); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("trigger.dockerHubHandler: failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if dw.Repository.RepoName == "" {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "repository name cannot be empty")
		return
	}

	if dw.PushData.Tag == "" {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "repository tag cannot be empty")
		return
	}

	event := types.Event{}
	event.CreatedAt = time.Now()
	event.TriggerName = "dockerhub"
	event.Repository.Name = dw.Repository.RepoName
	event.Repository.Tag = dw.PushData.Tag

	s.trigger(event)

	resp.WriteHeader(http.StatusOK)
	return
}
