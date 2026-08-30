#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
ARTIFACT_DIR="${KEEL_RELEASE_ARTIFACT_DIR:-${REPO_ROOT}/.test/release-artifacts}"
HELM_BIN="${HELM_BIN:-${ARTIFACT_DIR}/bin/helm}"
CHART_DIR="${REPO_ROOT}/chart/keel"
APP_VERSION="$("${REPO_ROOT}/scripts/resolve-application-version.sh")"
SOURCE_APP_VERSION="$(awk '$1 == "appVersion:" {print $2; exit}' "${CHART_DIR}/Chart.yaml")"
CHART_VERSION="${KEEL_CHART_VERSION:-$(awk '$1 == "version:" {print $2; exit}' "${CHART_DIR}/Chart.yaml")}"
PACKAGED_CHART_VERSION="v${CHART_VERSION#v}"
REVISION="${KEEL_BUILD_REVISION:-$(git -C "${REPO_ROOT}" rev-parse --short=12 HEAD)}"
BUILD_EPOCH="${SOURCE_DATE_EPOCH:-$(git -C "${REPO_ROOT}" show -s --format=%ct HEAD)}"
if BUILD_DATE="$(date -u -d "@${BUILD_EPOCH}" +%Y-%m-%dT%H%M%SZ 2>/dev/null)"; then
  :
elif BUILD_DATE="$(date -u -r "${BUILD_EPOCH}" +%Y-%m-%dT%H%M%SZ 2>/dev/null)"; then
  :
else
  printf '[release-package] ERROR: could not convert build epoch: %s\n' "${BUILD_EPOCH}" >&2
  exit 1
fi
IMAGE="${KEEL_RELEASE_IMAGE:-keel-release-validation:${APP_VERSION}-${REVISION}}"

fail() { printf '[release-package] ERROR: %s\n' "$*" >&2; exit 1; }
log() { printf '[release-package] %s\n' "$*"; }

[[ "${APP_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.+][0-9A-Za-z.-]+)?$ ]] || fail "invalid appVersion: ${APP_VERSION}"
[[ "${CHART_VERSION}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+([-.+][0-9A-Za-z.-]+)?$ ]] || fail "invalid chart version: ${CHART_VERSION}"
mkdir -p "${ARTIFACT_DIR}/bin" "${ARTIFACT_DIR}/chart" "${ARTIFACT_DIR}/image"

if [[ "${GITHUB_REF:-}" == refs/tags/* ]]; then
  tag="${GITHUB_REF#refs/tags/}"
  if [[ "${tag}" == chart-* ]]; then
    [[ "${tag}" == "chart-v${CHART_VERSION#v}" ]] || \
      fail "chart tag ${tag} must equal chart-v${CHART_VERSION#v} from Chart.yaml"
  else
    [[ "${tag}" == "${APP_VERSION}" ]] || \
      fail "application tag ${tag} must equal appVersion ${APP_VERSION} from Chart.yaml"
    [[ "${SOURCE_APP_VERSION}" == "${APP_VERSION}" ]] || \
      fail "Chart.yaml appVersion ${SOURCE_APP_VERSION} must equal application tag ${tag} so the matching chart can be published"
  fi
fi

if [[ "${KEEL_ENFORCE_UNPUBLISHED:-false}" == "true" ]]; then
  release_tag="keel-v${CHART_VERSION#v}"
  status="$(curl --silent --output /dev/null --write-out '%{http_code}' \
    "https://api.github.com/repos/keel-hq/keel/releases/tags/${release_tag}")"
  case "${status}" in
    404) ;;
    200) fail "GitHub chart release ${release_tag} already exists; refusing to overwrite it" ;;
    *) fail "could not establish whether GitHub chart release ${release_tag} exists (HTTP ${status})" ;;
  esac
  curl --fail --location --show-error --silent --output "${ARTIFACT_DIR}/public-index.yaml" \
    https://keel-hq.github.io/keel/index.yaml || fail "could not read the public Helm index"
  if awk -v wanted="${CHART_VERSION}" '$1 == "version:" {gsub(/\"/, "", $2); if ($2 == wanted || $2 == "v" wanted) found=1} END {exit !found}' \
    "${ARTIFACT_DIR}/public-index.yaml"; then
    fail "chart ${CHART_VERSION} already exists in the public Helm index; refusing to overwrite it"
  fi
fi

if [[ ! -x "${HELM_BIN}" ]] || ! "${HELM_BIN}" version --short >/dev/null 2>&1; then
  "${REPO_ROOT}/.test/install-helm.sh" "${HELM_BIN}"
fi

log "linting and rendering chart ${CHART_VERSION} (appVersion ${APP_VERSION})"
"${HELM_BIN}" lint "${CHART_DIR}" --strict
"${HELM_BIN}" template keel-validation "${CHART_DIR}" --namespace keel-validation \
  --set image.repository=example.invalid/keel \
  --set image.tag="${APP_VERSION}" \
  --set service.enabled=true >"${ARTIFACT_DIR}/rendered.yaml"
grep -Fq "image: \"example.invalid/keel:${APP_VERSION}\"" "${ARTIFACT_DIR}/rendered.yaml" || \
  fail "rendered candidate does not use the expected application version"

archive="$(${HELM_BIN} package "${CHART_DIR}" --destination "${ARTIFACT_DIR}/chart" \
  --version "${PACKAGED_CHART_VERSION}" --app-version "${APP_VERSION}" | awk -F': ' '/saved it to:/ {print $2}')"
[[ -s "${archive}" ]] || fail "Helm did not create the chart archive"
tar -tzf "${archive}" >"${ARTIFACT_DIR}/chart-contents.txt"
for required in keel/Chart.yaml keel/values.yaml keel/templates/deployment.yaml keel/templates/service.yaml; do
  grep -Fxq "${required}" "${ARTIFACT_DIR}/chart-contents.txt" || fail "chart archive is missing ${required}"
done
tar -xOf "${archive}" keel/Chart.yaml >"${ARTIFACT_DIR}/packaged-Chart.yaml"
grep -Eq "^version: ['\"]?${PACKAGED_CHART_VERSION//./\\.}['\"]?$" "${ARTIFACT_DIR}/packaged-Chart.yaml" || fail "packaged chart version mismatch"
grep -Eq "^appVersion: ['\"]?${APP_VERSION//./\\.}['\"]?$" "${ARTIFACT_DIR}/packaged-Chart.yaml" || fail "packaged appVersion mismatch"
sha256sum "${archive}" >"${archive}.sha256"

if [[ "${KEEL_RELEASE_SKIP_IMAGE:-false}" != "true" ]]; then
  log "building release-equivalent image ${IMAGE}"
  docker build --platform "${KEEL_RELEASE_PLATFORM:-linux/amd64}" \
    --build-arg "VERSION=${APP_VERSION}" --build-arg "REVISION=${REVISION}" \
    --build-arg "BUILD_DATE=${BUILD_DATE}" --tag "${IMAGE}" \
    --file "${REPO_ROOT}/Dockerfile" "${REPO_ROOT}"
  docker image inspect "${IMAGE}" >"${ARTIFACT_DIR}/image/inspect.json"
  docker run --rm --entrypoint /bin/keel "${IMAGE}" --version \
    </dev/null >"${ARTIFACT_DIR}/image/version.txt" 2>&1
  grep -Fxq "${APP_VERSION}" "${ARTIFACT_DIR}/image/version.txt" || \
    fail "container output does not contain the exact version ${APP_VERSION}; see ${ARTIFACT_DIR}/image/version.txt"
  if [[ "${KEEL_RELEASE_EXPORT_IMAGE:-false}" == "true" ]]; then
    log "exporting release-equivalent image for isolated validation jobs"
    docker save --output "${ARTIFACT_DIR}/image/image.tar" "${IMAGE}"
    sha256sum "${ARTIFACT_DIR}/image/image.tar" >"${ARTIFACT_DIR}/image/image.tar.sha256"
  fi
fi

{
  printf 'KEEL_RELEASE_APP_VERSION=%q\n' "${APP_VERSION}"
  printf 'KEEL_RELEASE_CHART_VERSION=%q\n' "${PACKAGED_CHART_VERSION}"
  printf 'KEEL_RELEASE_CHART=%q\n' "${archive}"
  printf 'KEEL_RELEASE_IMAGE=%q\n' "${IMAGE}"
  printf 'KEEL_RELEASE_REVISION=%q\n' "${REVISION}"
  printf 'HELM_BIN=%q\n' "${HELM_BIN}"
} >"${ARTIFACT_DIR}/release.env"
log "validated artifacts in ${ARTIFACT_DIR}"
