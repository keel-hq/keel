<p align="center">
  <a href="https://keel.sh" target="_blank"><img width="100"src="https://keel.sh/images/logo.png"></a>
</p>

<p align="center">
   
  <a href="https://hub.docker.com/r/keelhq/keel/">
    <img src="https://circleci.com/gh/keel-hq/keel/tree/master.svg?style=shield&circle-token=0239846a42cfa188de531058b9a2116a4b8600d8" alt="CircleCI">
  </a>
  
  <a href="https://goreportcard.com/report/github.com/keel-hq/keel">
    <img src="https://goreportcard.com/badge/github.com/keel-hq/keel" alt="Go Report">
  </a>
  
  <a href="https://img.shields.io/docker/pulls/karolisr/keel.svg">
    <img src="https://img.shields.io/docker/pulls/karolisr/keel.svg" alt="Docker Pulls">
  </a>   
</p>

# Keel - automated Kubernetes deployments for the rest of us

* Website [https://keel.sh](https://keel.sh)
* Slack - [kubernetes.slack.com](https://kubernetes.slack.com) look for channel #keel

Keel is a tool for automating [Kubernetes](https://kubernetes.io/) deployment updates. Keel is stateless, robust and lightweight.

Keel provides several key features:

* __[Kubernetes](https://kubernetes.io/) and [Helm](https://helm.sh) providers__ - Keel has direct integrations with Kubernetes and Helm.

* __No CLI/API__ - tired of `f***ctl` for everything? Keel doesn't have one. Gets job done through labels, annotations, charts.

* __Semver policies__ - specify update policy for each deployment/Helm release individually.

* __Automatic [Google Container Registry](https://cloud.google.com/container-registry/) configuration__ - Keel automatically sets up topic and subscriptions for your deployment images by periodically scanning your environment.

* __[Native, DockerHub and Quay webhooks](https://keel.sh/v1/guide/documentation.html#Triggers) support__ -  once webhook is received impacted deployments will be identified and updated.

*  __[Polling](https://keel.sh/v1/guide/documentation.html#Polling)__ - when webhooks and pubsub aren't available - Keel can still be useful by checking Docker Registry for new tags (if current tag is semver) or same tag SHA digest change (ie: `latest`).

* __Notifications__ - out of the box Keel has Slack, Hipchat, Mattermost and standard webhook notifications, more info [here](https://keel.sh/v1/guide/documentation.html#Notifications)

<p align="center">
  <a href="https://keel.sh" target="_blank"><img width="700"src="https://keel.sh/images/keel-overview.png"></a>
</p>

### Support

Support Keel's development by:
* [Patreon](https://patreon.com/keel)
* [Paypal](https://www.paypal.me/keelhq)
* Star this repository
* [Follow on Twitter](https://twitter.com/keel_hq)

### Quick Start

<p align="center">
  <a href="https://keel.sh" target="_blank"><img width="700"src="https://keel.sh/images/keel-workflow.png"></a>
</p>

A step-by-step guide to install Keel on your Kubernetes cluster is viewable on the Keel website:

[https://keel.sh/v1/guide/quick-start.html](https://keel.sh/v1/guide/quick-start.html)

### Configuration

Once Keel is deployed, you only need to specify update policy on your deployment file or Helm chart:

<p align="center">
  <a href="https://keel.sh/v1/guide/" target="_blank"><img width="700"src="https://keel.sh/images/keel-minimal-configuration.png"></a>
</p>

No additional configuration is required. Enabling continuous delivery for your workloads has never been this easy!

### Documentation

Documentation is viewable on the Keel Website:

[https://keel.sh/v1/guide/documentation](https://keel.sh/v1/guide/documentation)


### Contributing

Before starting to work on some big or medium features - raise an issue [here](https://github.com/keel-hq/keel/issues) so we can coordinate our efforts.

### Developing Keel

If you wish to work on Keel itself, you will need Go 1.8+ installed. Make sure you put Keel into correct Gopath and `go build` (dependency management is done through Glide). 

### Roadmap

Project [roadmap available here](https://github.com/keel-hq/keel/wiki/Roadmap).
