#!/usr/bin/env bash

set -Eeuo pipefail

readonly REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly K3S_VERSION="v1.35.6+k3s1"
readonly K3S_RELEASE_URL="https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION}"
readonly REGISTRY_IMAGE="registry@sha256:46faa9a1ae6813194b53921a370f2f4f8c5e1aae228a89bceafef5847a6a3278"
readonly FIXTURE_IMAGE="busybox@sha256:7a3ebe5bfd1a4a19797d20b0c0bb39d44393e9a03fd852c0865b0f540d868df0"

RUN_ID="${KEEL_E2E_RUN_ID:-$(date -u +%Y%m%d%H%M%S)-$$}"
RUN_DIR="${KEEL_E2E_RUN_DIR:-${REPO_ROOT}/.test/.runs/${RUN_ID}}"
BIN_DIR="${RUN_DIR}/bin"
ARTIFACT_DIR="${KEEL_E2E_ARTIFACT_DIR:-${REPO_ROOT}/.test/artifacts/${RUN_ID}}"
K3S_BIN="${BIN_DIR}/k3s"
K3S_DATA_DIR="${RUN_DIR}/k3s-data"
KUBECONFIG="${RUN_DIR}/kubeconfig"
K3S_CONFIG="${RUN_DIR}/k3s.yaml"
K3S_PID_FILE="${RUN_DIR}/k3s.pid"
K3S_LOG="${ARTIFACT_DIR}/k3s.log"

log() {
  printf '[keel-e2e] %s\n' "$*"
}

download_k3s() {
  mkdir -p "${BIN_DIR}" "${ARTIFACT_DIR}"
  curl --fail --location --show-error --silent \
    --output "${K3S_BIN}" "${K3S_RELEASE_URL}/k3s"
  curl --fail --location --show-error --silent \
    --output "${RUN_DIR}/sha256sum-amd64.txt" \
    "${K3S_RELEASE_URL}/sha256sum-amd64.txt"
  (
    cd "${BIN_DIR}"
    grep -E '  k3s$' "${RUN_DIR}/sha256sum-amd64.txt" | sha256sum --check --strict -
  )
  chmod 0755 "${K3S_BIN}"
}

print_versions() {
  "${K3S_BIN}" --version
  log "registry image: ${REGISTRY_IMAGE}"
  log "fixture image: ${FIXTURE_IMAGE}"
}

write_k3s_config() {
  mkdir -p "${K3S_DATA_DIR}" "${ARTIFACT_DIR}"
  umask 077
  {
    printf 'data-dir: %s\n' "${K3S_DATA_DIR}"
    printf 'write-kubeconfig: %s\n' "${KUBECONFIG}"
    printf 'write-kubeconfig-mode: "0600"\n'
    printf 'node-name: keel-e2e-%s\n' "${RUN_ID}"
    printf 'cluster-cidr: 10.52.0.0/16\n'
    printf 'service-cidr: 10.53.0.0/16\n'
    printf 'cluster-dns: 10.53.0.10\n'
    printf 'disable:\n  - traefik\n  - servicelb\n  - metrics-server\n  - local-storage\n'
  } >"${K3S_CONFIG}"
}

start_k3s() {
  write_k3s_config
  log "starting task-owned k3s process"
  sudo sh -c 'printf "%s\n" "$$" >"$1"; exec setsid "$2" server --config "$3"' \
    sh "${K3S_PID_FILE}" "${K3S_BIN}" "${K3S_CONFIG}" \
    >"${K3S_LOG}" 2>&1 &

  for _ in $(seq 1 120); do
    if [[ -s "${KUBECONFIG}" ]] && sudo "${K3S_BIN}" kubectl \
      --kubeconfig "${KUBECONFIG}" get nodes >/dev/null 2>&1; then
      sudo chown "$(id -u):$(id -g)" "${KUBECONFIG}"
      export KUBECONFIG
      sudo "${K3S_BIN}" kubectl --kubeconfig "${KUBECONFIG}" \
        wait --for=condition=Ready node --all --timeout=120s
      return
    fi
    sleep 1
  done
  log "k3s did not become ready; see ${K3S_LOG}"
  return 1
}

stop_k3s() {
  [[ -s "${K3S_PID_FILE}" ]] || return 0
  local pid
  pid="$(<"${K3S_PID_FILE}")"
  if sudo kill -0 "${pid}" 2>/dev/null; then
    sudo kill -TERM "-${pid}" 2>/dev/null || sudo kill -TERM "${pid}" 2>/dev/null || true
    for _ in $(seq 1 30); do
      sudo kill -0 "${pid}" 2>/dev/null || break
      sleep 1
    done
    sudo kill -KILL "-${pid}" 2>/dev/null || true
  fi
}

cleanup() {
  stop_k3s
  if [[ -d "${RUN_DIR}" ]]; then
    sudo find "${RUN_DIR}" -depth -delete
  fi
}

main() {
  trap cleanup EXIT INT TERM
  download_k3s
  print_versions
  start_k3s
  log "k3s is ready"
}

main "$@"
