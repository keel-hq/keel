# Keel - automated Kubernetes deployments for the rest of us

Lightweight (uses ~10MB RAM when running) [Kubernetes](https://kubernetes.io/) controller for automating deployment updates when new image is available. Keel uses [semantic versioning](http://semver.org/) to determine whether deployment needs an update or not. Currently keel has several types of triggers:

* Google's pubsub integration with [Google Container Registry](https://cloud.google.com/container-registry/)
* [DockerHub Webhooks](https://docs.docker.com/docker-hub/webhooks/)
* Webhooks

Upcomming integrations:

* DockerHub webhooks


## Keel overview

* Stateless, runs as a single container in kube-system namespace
* Automatically detects images that you have in your Kubernetes environment and configures relevant [Google Cloud pubsub](https://cloud.google.com/pubsub/) topic, subscriptions.
* Updates deployment if you have set Keel policy and newer image is available.

## Why?

I have built Keel since I have a relatively small Golang project which doesn't use a lot of memory and introducing an antique, heavy weight CI solution with lots dependencies seemed like a terrible idea. 

You should consider using Keel if:
* You don't want your "Continous Delivery" tool to consume more resources than your actual deployment does.
* You are __not__ Netflix, Google, Amazon, {insert big company here} that already has something like Spinnaker that has too many dependencies such as "JDK8, Redis, Cassandra, Packer".
* You want simple, automated Kubernetes deployment updates.


## Getting started

Keel operates as a background service, you don't need to interact with it directly, just add labels to your deployments. 

### Example deployment

Here is an example deployment which specifies that keel should always update image if a new version is available:

```
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata: 
  name: wd
  namespace: default
  labels: 
      name: "wd"
      keel.observer/policy: all
spec:
  replicas: 1
  template:
    metadata:
      name: wd
      labels:
        app: wd        

    spec:
      containers:                    
        - image: karolisr/webhook-demo:0.0.2
          imagePullPolicy: Always            
          name: wd
          command: ["/bin/webhook-demo"]
          ports:
            - containerPort: 8090       
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8090
            initialDelaySeconds: 30
            timeoutSeconds: 10
          securityContext:
            privileged: true      
```

Available policy options:

* __all__ - update whenever there is a version bump
* __major__ - update major versions
* __minor__ - update only minor versions (ignores major)
* __patch__ - update only patch versions (ignores minor and major versions)

## Deployment

### Step 1: GCE Kubernetes + GCR pubsub configuration

Since access to pubsub is required in GCE Kubernetes - your cluster node pools need to have permissions. If you are creating new cluster - just enable pubsub from the start. If you have existing cluster - currently the only way is create new node-pool through the gcloud CLI (more info in the [docs](https://cloud.google.com/sdk/gcloud/reference/container/node-pools/create?hl=en_US&_ga=1.2114551.650086469.1487625651):

```
gcloud container node-pools create new-pool --cluster CLUSTER_NAME --scopes https://www.googleapis.com/auth/pubsub
```    

### Step 2: Kubernetes

Keel will be updating deployments, let's create a new [service account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/) in `kube-system` namespace:

```
kubectl create serviceaccount keel --namespace=kube-system
```
Now, edit [deployment file](https://github.com/rusenask/keel/blob/master/hack/deployment.sample.yml) that is supplied with the repository (basically point to the [newest Keel release](https://hub.docker.com/r/karolisr/keel/tags/) and set your PROJECT_ID to the actual project ID that you have):

```
kubectl create -f hack/deployment.yml
```

Once Keel is deployed in your Kubernetes cluster - it occasionally scans your current deployments and looks for ones that have label _keel.observer/policy_. It then checks whether appropriate subscriptions and topics are set for GCR registries, if not - auto-creates them.