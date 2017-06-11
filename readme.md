# Keel

Lightweight Kubernetes controller for automating image updates for deployments. Keel uses [semantic versioning](http://semver.org/) to determine whether deployment needs an update or not. Currently keel has several types of triggers:

* Google's pubsub integration with [Google Container Registry](https://cloud.google.com/container-registry/)
* Webhooks

Upcomming integrations:

* DockerHub webhooks

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

* all - update whenever there is a version bump
* major - update major versions
* minor - update only minor versions (ignores major)
* patch - update only patch versions (ignores minor and major versions)

## Deployment

### Step 1: GCE Kubernetes + GCR pubsub configuration

Since access to pubsub is required in GCE Kubernetes - your cluster node pools need to have permissions. If you are creating new cluster - just enable pubsub from the start. If you have existing cluster - currently the only way is create new node-pool through the gcloud CLI (more info in the [docs](https://cloud.google.com/sdk/gcloud/reference/container/node-pools/create?hl=en_US&_ga=1.2114551.650086469.1487625651):

```
gcloud container node-pools create new-pool --cluster CLUSTER_NAME --scopes https://www.googleapis.com/auth/pubsub
```    

### Step 2: Kubernetes

Since keel will be updating deployments, let's create a new [service account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/) in `kube-system` namespace:

```
kubectl create serviceaccount keel --namespace=kube-system
```

Now, edit [deployment file](https://github.com/rusenask/keel/blob/master/hack/deployment.sample.yml) that is supplied with the repo (basically point to the newest keel release and set your PROJECT_ID to the actual project ID that you have):

```
kubectl create -f hack/deployment.yml
```

Once Keel is deployed in your Kubernetes cluster - it occasionally scans your current deployments and looks for ones that have label _keel.observer/policy_. It then checks whether appropriate subscriptions and topics are set for GCR registries, if not - auto-creates them.

