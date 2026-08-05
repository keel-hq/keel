#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
ARTIFACT_DIR="${KEEL_RELEASE_ARTIFACT_DIR:-${REPO_ROOT}/.test/release-artifacts}"

if [[ "${KEEL_RELEASE_REUSE_ARTIFACTS:-false}" != "true" ]]; then
  KEEL_RELEASE_ARTIFACT_DIR="${ARTIFACT_DIR}" "${REPO_ROOT}/scripts/package-release.sh"
fi
if [[ "${KEEL_RELEASE_PACKAGE_ONLY:-false}" == "true" ]]; then
  exit 0
fi

[[ -s "${ARTIFACT_DIR}/release.env" ]] || {
  printf '[release-validation] ERROR: release metadata not found: %s\n' "${ARTIFACT_DIR}/release.env" >&2
  exit 1
}
set -a
# shellcheck disable=SC1090
source "${ARTIFACT_DIR}/release.env"
set +a
KEEL_RELEASE_CHART="${ARTIFACT_DIR}/chart/$(basename "${KEEL_RELEASE_CHART}")"
HELM_BIN="${ARTIFACT_DIR}/bin/helm"
if ! docker image inspect "${KEEL_RELEASE_IMAGE}" >/dev/null 2>&1; then
  image_archive="${ARTIFACT_DIR}/image/image.tar"
  [[ -s "${image_archive}" && -s "${image_archive}.sha256" ]] || {
    printf '[release-validation] ERROR: prebuilt image archive not found: %s\n' "${image_archive}" >&2
    exit 1
  }
  (cd "$(dirname "${image_archive}")" && sha256sum --check --strict "$(basename "${image_archive}").sha256")
  docker load --input "${image_archive}"
fi
export KEEL_E2E_PREBUILT_IMAGE="${KEEL_RELEASE_IMAGE}"
export KEEL_E2E_RELEASE_CHART="${KEEL_RELEASE_CHART}"
export KEEL_E2E_HELM_BIN="${HELM_BIN}"
export KEEL_E2E_APP_VERSION="${KEEL_RELEASE_APP_VERSION}"
"${REPO_ROOT}/.test/e2e-k3s.sh"
