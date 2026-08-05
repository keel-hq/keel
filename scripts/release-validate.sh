#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
ARTIFACT_DIR="${KEEL_RELEASE_ARTIFACT_DIR:-${REPO_ROOT}/.test/release-artifacts}"

KEEL_RELEASE_ARTIFACT_DIR="${ARTIFACT_DIR}" "${REPO_ROOT}/scripts/package-release.sh"
if [[ "${KEEL_RELEASE_PACKAGE_ONLY:-false}" == "true" ]]; then
  exit 0
fi

set -a
# shellcheck disable=SC1090
source "${ARTIFACT_DIR}/release.env"
set +a
export KEEL_E2E_PREBUILT_IMAGE="${KEEL_RELEASE_IMAGE}"
export KEEL_E2E_RELEASE_CHART="${KEEL_RELEASE_CHART}"
export KEEL_E2E_HELM_BIN="${HELM_BIN}"
export KEEL_E2E_APP_VERSION="${KEEL_RELEASE_APP_VERSION}"
"${REPO_ROOT}/.test/e2e-k3s.sh"
