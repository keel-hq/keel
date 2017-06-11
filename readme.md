# Keel

Lightweight Kubernetes controller for automating image updates for deployments. Keel uses [semantic versioning](http://semver.org/) to determine whether deployment needs an update or not.

## Getting started

Once Keel is deployed in your Kubernetes cluster - it waits for events regarding updated images. 
Images tagged with label _keel.io/policy_ will be processed.

Available policy options:

* all - update whenever there is a version bump
* major - update major versions
* minor - update only minor versions
* patch - update only patch versions

## Deployment

### GCE Kubernetes + GCR 

Google Container Registry uses pubsub events to inform about pushed/deleted images. Keel subscribes to this queue, parses events and submits them to Keel Kubernetes provider.

Since access to pubsub is required in GCE Kubernetes - your cluster node pools need to have permissions. If you are creating new cluster - just enable pubsub from the start. If you have existing cluster - currently the only way is create new node-pool through the gcloud CLI (more info in the [docs](https://cloud.google.com/sdk/gcloud/reference/container/node-pools/create?hl=en_US&_ga=1.2114551.650086469.1487625651):

    gcloud container node-pools create new-pool --cluster CLUSTER_NAME --scopes https://www.googleapis.com/auth/pubsub

Then, you need to create a subscription for registry events ([docs](https://cloud.google.com/container-registry/docs/configuring-notifications)). Just replace "PROJECT-ID" with your own project ID and "QUALIFIED-GCR-URI" with your registry/image:

```
gcloud alpha pubsub topics create projects/PROJECT-ID/topics/QUALIFIED-GCR-URI
gcloud alpha pubsub subscriptions create gcr-sub --topic=QUALIFIED-GCR-URI
```

Example:
* QUALIFIED-GCR-URI - `gcr.io%2Frepo-name`, where %2F is encoded forward slash. So if your repo is gcr.io/awesome-app, then QUALIFIED-GCR-URI would be  `gcr.io%2Fawesome-app`. 


#### Actual deployment

* Create service account:

    kubectl create serviceaccount keel --namespace=kube-system


While running Kubernetes on GCE it's convenient to use [Google Container Registry](https://cloud.google.com/container-registry/). 
