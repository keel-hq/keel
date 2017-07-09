# Keel - automated Kubernetes deployments for the rest of us

* Website [https://keel.sh](https://keel.sh)
* Slack - [kubernetes.slack.com](kubernetes.slack.com) look for @karolis

Keel is a tool for automating [Kubernetes](https://kubernetes.io/) deployment updates. Keel is stateless, robust and lightweight.

Keel provides several key features:

* __Semver policies__ - specify update policy for each deployment individually.

* __Automatic [Google Container Registry](https://cloud.google.com/container-registry/) configuration__ - Keel automatically sets up topic and subscriptions for your deployment images by periodically scanning your environment.

* __[DockerHub Webhooks](https://docs.docker.com/docker-hub/webhooks/) support__ - Keel accepts dockerhub style webhooks on `/v1/webhooks/dockerhub` endpoint. Impacted deployments will be identified and updated.

*  __[Polling](https://keel.sh/user-guide/#polling-deployment-example)__ - when webhooks and pubsub aren't available - Keel can still be useful by checking Docker Registry for changed SHA digest.

* __Notifications__ - out of the box Keel has Slack and standard webhook notifications, more info [here](https://keel.sh/user-guide/#notifications)

<img src="https://keel.sh/images/keel-overview.png">

### Quick Start

A step-by-step guide to install Keel on your Kubernetes cluster is viewable on the Keel website:

[https://keel.sh/install](https://keel.sh/install)

### Documentation

Documentation is viewable on the Keel Website:

[https://keel.sh/user-guide/](https://keel.sh/user-guide/)


### Contributing

Before starting to work on some big or medium features - raise an issue [here](https://github.com/rusenask/keel/issues) so we can coordinate our efforts.

### Developing Keel

If you wish to work on Keel itself, you will need Go 1.8+ installed. Make sure you put Keel into correct Gopath and get remaining dependencies (some dependencies are already locked through glide). 

### Roadmap

Project [roadmap available here](https://github.com/rusenask/keel/wiki/Roadmap).
