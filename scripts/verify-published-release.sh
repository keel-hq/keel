#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
ARTIFACT_DIR="${KEEL_PUBLISHED_ARTIFACT_DIR:-${REPO_ROOT}/.test/published-artifacts}"
APP_VERSION="${KEEL_PUBLISHED_APP_VERSION:-0.21.1}"
CHART_VERSION="${KEEL_PUBLISHED_CHART_VERSION:-v1.0.5}"
INDEX_URL="https://keel-hq.github.io/keel/index.yaml"
GHCR_IMAGE="ghcr.io/keel-hq/keel:${APP_VERSION}"

fail() { printf '[published-release] ERROR: %s\n' "$*" >&2; exit 1; }
log() { printf '[published-release] %s\n' "$*"; }
require() { command -v "$1" >/dev/null || fail "required command not found: $1"; }
for command in awk curl docker jq sha256sum tar; do require "${command}"; done
mkdir -p "${ARTIFACT_DIR}"

log "checking GitHub application release ${APP_VERSION}"
curl --fail --location --show-error --silent \
  "https://api.github.com/repos/keel-hq/keel/releases/tags/${APP_VERSION}" \
  >"${ARTIFACT_DIR}/github-app-release.json"
jq -e --arg version "${APP_VERSION}" '.tag_name == $version and .draft == false' \
  "${ARTIFACT_DIR}/github-app-release.json" >/dev/null || fail "GitHub application release is missing, draft, or mismatched"

if [[ "${KEEL_PUBLISHED_SKIP_CHART:-false}" == "true" ]]; then
  log "checking GHCR application manifest ${GHCR_IMAGE}"
  docker buildx imagetools inspect "${GHCR_IMAGE}" --raw >"${ARTIFACT_DIR}/app-manifest.json"
  jq -e '
    any(.manifests[]; .platform.os == "linux" and .platform.architecture == "amd64") and
    any(.manifests[]; .platform.os == "linux" and .platform.architecture == "arm64")
  ' "${ARTIFACT_DIR}/app-manifest.json" >/dev/null || fail "${GHCR_IMAGE} is not a linux/amd64+linux/arm64 image index"
  log "published application release and multi-architecture image are aligned"
  exit 0
fi

log "checking Helm repository chart ${CHART_VERSION}"
curl --fail --location --show-error --silent "${INDEX_URL}" >"${ARTIFACT_DIR}/index.yaml"
chart_fields="$(awk -v wanted="${CHART_VERSION}" '
  /^  - apiVersion:/ {app=""; digest=""; url=""; version=""}
  /^    appVersion:/ {app=$2; gsub(/\"/, "", app)}
  /^    digest:/ {digest=$2}
  /^    - https?:/ {url=$2}
  /^    version:/ {version=$2; gsub(/\"/, "", version); if (version == wanted) {print app "\t" digest "\t" url; exit}}
' "${ARTIFACT_DIR}/index.yaml")"
[[ -n "${chart_fields}" ]] || fail "chart ${CHART_VERSION} is absent from ${INDEX_URL}"
IFS=$'\t' read -r chart_app_version chart_digest chart_url <<<"${chart_fields}"
[[ "${chart_digest}" =~ ^[0-9a-f]{64}$ ]] || fail "invalid chart digest in Helm index: ${chart_digest}"
[[ "${chart_url}" == https://github.com/keel-hq/keel/releases/download/* ]] || fail "unexpected chart URL: ${chart_url}"

chart_archive="${ARTIFACT_DIR}/keel-${CHART_VERSION}.tgz"
curl --fail --location --show-error --silent --output "${chart_archive}" "${chart_url}"
actual_digest="$(sha256sum "${chart_archive}" | awk '{print $1}')"
[[ "${actual_digest}" == "${chart_digest}" ]] || fail "chart checksum mismatch: index=${chart_digest}, archive=${actual_digest}"
tar -xOf "${chart_archive}" keel/Chart.yaml >"${ARTIFACT_DIR}/Chart.yaml"
tar -xOf "${chart_archive}" keel/values.yaml >"${ARTIFACT_DIR}/values.yaml"
grep -Eq "^version: ['\"]?${CHART_VERSION//./\\.}['\"]?$" "${ARTIFACT_DIR}/Chart.yaml" || fail "chart archive version does not match the index"
grep -Eq "^appVersion: ['\"]?${chart_app_version//./\\.}['\"]?$" "${ARTIFACT_DIR}/Chart.yaml" || fail "chart archive appVersion does not match the index"

chart_release_tag="keel-v${CHART_VERSION#v}"
curl --fail --location --show-error --silent \
  "https://api.github.com/repos/keel-hq/keel/releases/tags/${chart_release_tag}" \
  >"${ARTIFACT_DIR}/github-chart-release.json"
jq -e --arg asset "keel-${CHART_VERSION}.tgz" 'any(.assets[]; .name == $asset)' \
  "${ARTIFACT_DIR}/github-chart-release.json" >/dev/null || fail "GitHub chart release does not contain the indexed archive"

curl --fail --location --show-error --silent \
  "https://api.github.com/repos/keel-hq/keel/releases/tags/${chart_app_version}" \
  >"${ARTIFACT_DIR}/github-chart-app-release.json"
jq -e --arg version "${chart_app_version}" '.tag_name == $version and .draft == false' \
  "${ARTIFACT_DIR}/github-chart-app-release.json" >/dev/null || \
  fail "chart appVersion ${chart_app_version} has no matching non-draft GitHub application release"

log "checking GHCR application manifest ${GHCR_IMAGE}"
docker buildx imagetools inspect "${GHCR_IMAGE}" --raw >"${ARTIFACT_DIR}/app-manifest.json"
jq -e '
  any(.manifests[]; .platform.os == "linux" and .platform.architecture == "amd64") and
  any(.manifests[]; .platform.os == "linux" and .platform.architecture == "arm64")
' "${ARTIFACT_DIR}/app-manifest.json" >/dev/null || fail "${GHCR_IMAGE} is not a linux/amd64+linux/arm64 image index"
docker buildx imagetools inspect "${GHCR_IMAGE}" >"${ARTIFACT_DIR}/app-image.txt"

chart_repository="$(awk '$1 == "repository:" {print $2; exit}' "${ARTIFACT_DIR}/values.yaml")"
[[ -n "${chart_repository}" ]] || fail "published chart has no default image repository"
chart_image="${chart_repository}:${chart_app_version}"
log "checking chart-referenced image ${chart_image}"
docker buildx imagetools inspect "${chart_image}" --raw >"${ARTIFACT_DIR}/chart-image-manifest.json"
docker buildx imagetools inspect "${chart_image}" >"${ARTIFACT_DIR}/chart-image.txt"

cat >"${ARTIFACT_DIR}/summary.txt" <<EOF
application_release=${APP_VERSION}
application_image=${GHCR_IMAGE}
chart_version=${CHART_VERSION}
chart_app_version=${chart_app_version}
chart_app_release=${chart_app_version}
chart_image=${chart_image}
chart_sha256=${chart_digest}
chart_url=${chart_url}
EOF
log "published metadata is aligned; details are in ${ARTIFACT_DIR}/summary.txt"
