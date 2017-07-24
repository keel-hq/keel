[![CircleCI](https://circleci.com/gh/rusenask/keel/tree/master.svg?style=shield&circle-token=0239846a42cfa188de531058b9a2116a4b8600d8)](https://circleci.com/gh/rusenask/keel/tree/master)

# Keel - automated Kubernetes deployments for the rest of us

* Website [https://keel.sh](https://keel.sh)

Keel is a tool for automating [Kubernetes](https://kubernetes.io/) deployment updates. Keel is stateless, robust and lightweight.

## Install for the first time

Docker image _polling_ is set by default, we also enabling _Helm provider_ support, so Helm releases
can be upgraded when new Docker image is available:

```console
helm upgrade --install keel keel --set helmProvider.enabled="true"
```


## Run upgrades e.g. for docker image change
```console
helm upgrade keel keel --reuse-values --set image.tag="0.4.0"
```
