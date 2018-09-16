#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# This will set Capabilities.KubeVersion.Major/Minor when generating manifests
KUBE_VERSION=1.9

gen() {
	VALUES=$1
	OUTPUT=$2

  TMP_VALUES=$(mktemp -d)
	TMP_OUTPUT=$(mktemp)

	mkdir -p "$(dirname ${OUTPUT})"
  cp chart/keel/values.yaml "${TMP_VALUES}"/
  sed -i 's/false/true/g' "${TMP_VALUES}/values.yaml"
	helm template \
		"chart/keel" \
    --values "${TMP_VALUES}/values.yaml" \
		--values "deployment/values/${VALUES}.yaml" \
		--kube-version "${KUBE_VERSION}" \
		--namespace "keel" \
		--name "keel" \
		--set "createNamespaceResource=true" > "${TMP_OUTPUT}"
  mv "${TMP_OUTPUT}" "${OUTPUT}"
  rm -fr "${TMP_VALUES}"
}

gen rbac "deployment/deployment-rbac.yaml"
gen norbac "deployment/deployment-norbac.yaml"
gen rbac-whr-sidecar "deployment/deployment-rbac-whr-sidecar.yaml"
gen norbac-whr-sidecar "deployment/deployment-norbac-whr-sidecar.yaml"
