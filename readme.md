[![CircleCI](https://circleci.com/gh/rusenask/keel/tree/master.svg?style=shield&circle-token=0239846a42cfa188de531058b9a2116a4b8600d8)](https://hub.docker.com/r/karolisr/keel/) [![Go Report Card](https://goreportcard.com/badge/github.com/rusenask/keel)](https://goreportcard.com/report/github.com/rusenask/keel) [![Pulls](https://img.shields.io/docker/pulls/karolisr/keel.svg)](https://img.shields.io/docker/pulls/karolisr/keel.svg)

# Keel - automated Kubernetes deployments for the rest of us

* Website [https://keel.sh](https://keel.sh)
* Slack - [kubernetes.slack.com](https://kubernetes.slack.com) look for channel #keel

Keel is a tool for automating [Kubernetes](https://kubernetes.io/) deployment updates. Keel is stateless, robust and lightweight.

Keel provides several key features:

* __[Kubernetes](https://kubernetes.io/) and [Helm](https://helm.sh) providers__ - Keel has direct integrations with Kubernetes and Helm.

* __No CLI/API__ - tired of `f***ctl` for everything? Keel doesn't have one. Gets job done through labels, annotations, charts.

* __Semver policies__ - specify update policy for each deployment/Helm release individually.

* __Automatic [Google Container Registry](https://cloud.google.com/container-registry/) configuration__ - Keel automatically sets up topic and subscriptions for your deployment images by periodically scanning your environment.

* __[Native, DockerHub and Quay webhooks](https://keel.sh/user-guide/triggers/#webhooks) support__ -  once webhook is received impacted deployments will be identified and updated.

*  __[Polling](https://keel.sh/user-guide/#polling-deployment-example)__ - when webhooks and pubsub aren't available - Keel can still be useful by checking Docker Registry for new tags (if current tag is semver) or same tag SHA digest change (ie: `latest`).

* __Notifications__ - out of the box Keel has Slack and standard webhook notifications, more info [here](https://keel.sh/user-guide/#notifications)

<img src="https://keel.sh/images/keel-overview.png">

### Support

Support Keel's development with:
* [Patreon](https://patreon.com/keel)
* [Paypal](https://www.paypal.me/keelhq)

### Quick Start

A step-by-step guide to install Keel on your Kubernetes cluster is viewable on the Keel website:

[https://keel.sh/install](https://keel.sh/install)

### Documentation

Documentation is viewable on the Keel Website:

[https://keel.sh/user-guide/](https://keel.sh/user-guide/)


### Contributing

Before starting to work on some big or medium features - raise an issue [here](https://github.com/rusenask/keel/issues) so we can coordinate our efforts.

### Developing Keel

If you wish to work on Keel itself, you will need Go 1.8+ installed. Make sure you put Keel into correct Gopath and `go build` (dependency management is done through Glide). 

### Roadmap

Project [roadmap available here](https://github.com/rusenask/keel/wiki/Roadmap).
