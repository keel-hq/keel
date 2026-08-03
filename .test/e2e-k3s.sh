#!/usr/bin/env bash

set -Eeuo pipefail

readonly REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly K3S_VERSION="v1.35.6+k3s1"
readonly K3S_RELEASE_URL="https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION}"
readonly REGISTRY_IMAGE="registry@sha256:46faa9a1ae6813194b53921a370f2f4f8c5e1aae228a89bceafef5847a6a3278"
readonly FIXTURE_IMAGE="busybox@sha256:7a3ebe5bfd1a4a19797d20b0c0bb39d44393e9a03fd852c0865b0f540d868df0"
readonly REGISTRY_ADDRESS="10.53.0.50:5000"

RUN_ID="${KEEL_E2E_RUN_ID:-$(date -u +%Y%m%d%H%M%S)-$$}"
RUN_DIR="${KEEL_E2E_RUN_DIR:-${REPO_ROOT}/.test/.runs/${RUN_ID}}"
BIN_DIR="${RUN_DIR}/bin"
ARTIFACT_DIR="${KEEL_E2E_ARTIFACT_DIR:-${REPO_ROOT}/.test/artifacts/${RUN_ID}}"
K3S_BIN="${BIN_DIR}/k3s"
K3S_DATA_DIR="${RUN_DIR}/k3s-data"
KUBECONFIG="${RUN_DIR}/kubeconfig"
K3S_CONFIG="${RUN_DIR}/k3s.yaml"
REGISTRIES_CONFIG="${RUN_DIR}/registries.yaml"
K3S_PID_FILE="${RUN_DIR}/k3s.pid"
K3S_LOG="${ARTIFACT_DIR}/k3s.log"
PORT_FORWARD_PID_FILE="${RUN_DIR}/port-forward.pid"
E2E_ENV_FILE="${RUN_DIR}/e2e.env"
CLEANUP_STARTED=0

log() {
  printf '[keel-e2e] %s\n' "$*"
}

fail() {
  log "ERROR: $*"
  exit 1
}

port_is_listening() {
  local port="$1"
  ss -ltnH | awk '{print $4}' | grep -Eq "(^|:)${port}$"
}

preflight() {
  local command
  for command in awk curl docker grep ip setsid sha256sum ss sudo; do
    command -v "${command}" >/dev/null || fail "required command not found: ${command}"
  done
  sudo -n true 2>/dev/null || fail "passwordless sudo is required"
  docker info >/dev/null 2>&1 || fail "Docker daemon is not available"

  [[ "${RUN_ID}" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ ]] || \
    fail "KEEL_E2E_RUN_ID must contain only lowercase letters, digits, and hyphens"
  ((${#RUN_ID} <= 40)) || fail "KEEL_E2E_RUN_ID must be at most 40 characters"
  [[ "${RUN_DIR}" == "${REPO_ROOT}/.test/.runs/"* ]] || \
    fail "run directory must be under ${REPO_ROOT}/.test/.runs"
  [[ "${ARTIFACT_DIR}" == "${REPO_ROOT}/.test/artifacts/"* ]] || \
    fail "artifact directory must be under ${REPO_ROOT}/.test/artifacts"
  [[ ! -e "${RUN_DIR}" ]] || fail "run directory already exists: ${RUN_DIR}"

  if command -v systemctl >/dev/null && systemctl is-active --quiet k3s 2>/dev/null; then
    fail "an existing k3s service is active"
  fi
  pgrep -x k3s >/dev/null 2>&1 && fail "an existing k3s process is active"
  [[ ! -e /etc/rancher/k3s/k3s.yaml ]] || fail "existing k3s kubeconfig found"
  [[ ! -e /var/lib/rancher/k3s ]] || fail "existing k3s data directory found"
  [[ ! -e "${HOME}/.kube/config" ]] || fail "existing default kubeconfig found"
  ip link show cni0 >/dev/null 2>&1 && fail "existing cni0 interface found"
  ip link show flannel.1 >/dev/null 2>&1 && fail "existing flannel.1 interface found"
  port_is_listening 6443 && fail "Kubernetes API port 6443 is already in use"
  port_is_listening 5000 && fail "registry port 5000 is already in use"
  port_is_listening 19300 && fail "Keel port-forward port 19300 is already in use"
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
    printf 'private-registry: %s\n' "${REGISTRIES_CONFIG}"
    printf 'disable:\n  - traefik\n  - servicelb\n  - metrics-server\n  - local-storage\n'
  } >"${K3S_CONFIG}"
  {
    printf 'mirrors:\n'
    printf '  "%s":\n' "${REGISTRY_ADDRESS}"
    printf '    endpoint:\n'
    printf '      - "http://%s"\n' "${REGISTRY_ADDRESS}"
  } >"${REGISTRIES_CONFIG}"
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

kubectl() {
  "${K3S_BIN}" kubectl --kubeconfig "${KUBECONFIG}" "$@"
}

ctr() {
  sudo "${K3S_BIN}" ctr \
    --address "${K3S_DATA_DIR}/agent/containerd/containerd.sock" \
    --namespace k8s.io "$@"
}

deploy_registry() {
  log "deploying isolated registry at ${REGISTRY_ADDRESS}"
  kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: keel-e2e-infra-${RUN_ID}
  labels:
    keel.sh/e2e-run: ${RUN_ID}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
  namespace: keel-e2e-infra-${RUN_ID}
  labels:
    app: keel-e2e-registry
    keel.sh/e2e-run: ${RUN_ID}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keel-e2e-registry
  template:
    metadata:
      labels:
        app: keel-e2e-registry
        keel.sh/e2e-run: ${RUN_ID}
    spec:
      containers:
        - name: registry
          image: ${REGISTRY_IMAGE}
          ports:
            - name: registry
              containerPort: 5000
          readinessProbe:
            httpGet:
              path: /v2/
              port: registry
          resources:
            requests:
              cpu: 20m
              memory: 32Mi
            limits:
              cpu: 200m
              memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: registry
  namespace: keel-e2e-infra-${RUN_ID}
  labels:
    keel.sh/e2e-run: ${RUN_ID}
spec:
  clusterIP: 10.53.0.50
  selector:
    app: keel-e2e-registry
  ports:
    - name: registry
      port: 5000
      targetPort: registry
EOF
  kubectl -n "keel-e2e-infra-${RUN_ID}" rollout status deployment/registry --timeout=120s
  curl --fail --show-error --silent "http://${REGISTRY_ADDRESS}/v2/" >/dev/null
}

build_keel_image() {
  local local_image="keel-e2e:${RUN_ID}"
  local registry_image="${REGISTRY_ADDRESS}/keel-under-test:${RUN_ID}"
  local image_archive="${RUN_DIR}/keel-image.tar"
  local digest

  log "building Keel container from ${REPO_ROOT}/Dockerfile"
  docker build --tag "${local_image}" --file "${REPO_ROOT}/Dockerfile" "${REPO_ROOT}"
  docker save --output "${image_archive}" "${local_image}"
  ctr images import "${image_archive}"
  ctr images tag "docker.io/library/${local_image}" "${registry_image}"
  ctr images push --plain-http "${registry_image}"

  digest="$(curl --fail --show-error --silent --head \
    --header 'Accept: application/vnd.docker.distribution.manifest.v2+json' \
    "http://${REGISTRY_ADDRESS}/v2/keel-under-test/manifests/${RUN_ID}" | \
    awk 'tolower($1) == "docker-content-digest:" {gsub("\\r", "", $2); print $2}')"
  [[ "${digest}" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "could not resolve Keel image digest"

  {
    printf 'KEEL_E2E_RUN_ID=%q\n' "${RUN_ID}"
    printf 'KEEL_E2E_REGISTRY=%q\n' "${REGISTRY_ADDRESS}"
    printf 'KEEL_E2E_REPOSITORY_PREFIX=%q\n' "${REGISTRY_ADDRESS}/${RUN_ID}"
    printf 'KEEL_E2E_IMAGE=%q\n' "${REGISTRY_ADDRESS}/keel-under-test@${digest}"
    printf 'KEEL_E2E_KUBECTL=%q\n' "${K3S_BIN}"
    printf 'KEEL_E2E_KUBECONFIG=%q\n' "${KUBECONFIG}"
    printf 'KEEL_E2E_ARTIFACT_DIR=%q\n' "${ARTIFACT_DIR}"
  } >"${E2E_ENV_FILE}"
  log "Keel image: ${REGISTRY_ADDRESS}/keel-under-test@${digest}"
}

seed_fixture_repositories() {
  local destination
  log "seeding isolated registry repositories"
  ctr images pull --platform linux/amd64 "${FIXTURE_IMAGE}"
  for destination in \
    "${RUN_ID}/webhook:1.0.0" "${RUN_ID}/webhook:1.0.1" \
    "${RUN_ID}/polling:1.0.0" "${RUN_ID}/polling:1.0.1" \
    "${RUN_ID}/negative:1.0.0" "${RUN_ID}/negative:1.1.0"; do
    ctr images tag "${FIXTURE_IMAGE}" "${REGISTRY_ADDRESS}/${destination}"
    ctr images push --plain-http "${REGISTRY_ADDRESS}/${destination}"
  done
}

run_tests() {
  log "running testify end-to-end suite"
  set -a
  # shellcheck disable=SC1090
  source "${E2E_ENV_FILE}"
  set +a
  go test -count=1 -v ./tests 2>&1 | tee "${ARTIFACT_DIR}/go-test.log"
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

collect_diagnostics() {
  mkdir -p "${ARTIFACT_DIR}"
  if [[ -s "${K3S_LOG}" ]]; then
    cp "${K3S_LOG}" "${ARTIFACT_DIR}/k3s-server.log" 2>/dev/null || true
  fi
}

delete_suite_resources() {
  [[ -x "${K3S_BIN}" && -s "${KUBECONFIG}" ]] || return 0
  "${K3S_BIN}" kubectl --kubeconfig "${KUBECONFIG}" delete \
    namespaces,clusterroles,clusterrolebindings \
    --selector "keel.sh/e2e-run=${RUN_ID}" --ignore-not-found --wait=true \
    >/dev/null 2>&1 || true
}

stop_port_forward() {
  [[ -s "${PORT_FORWARD_PID_FILE}" ]] || return 0
  local pid
  pid="$(<"${PORT_FORWARD_PID_FILE}")"
  if kill -0 "${pid}" 2>/dev/null; then
    kill -TERM "${pid}" 2>/dev/null || true
    wait "${pid}" 2>/dev/null || true
  fi
}

cleanup() {
  ((CLEANUP_STARTED == 0)) || return 0
  CLEANUP_STARTED=1
  collect_diagnostics
  delete_suite_resources
  stop_port_forward
  stop_k3s
  if ip link show cni0 >/dev/null 2>&1; then
    sudo ip link delete cni0 2>/dev/null || true
  fi
  if ip link show flannel.1 >/dev/null 2>&1; then
    sudo ip link delete flannel.1 2>/dev/null || true
  fi
  if [[ -d "${RUN_DIR}" ]]; then
    sudo find "${RUN_DIR}" -depth -delete
  fi
}

main() {
  preflight
  trap cleanup EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  download_k3s
  print_versions
  start_k3s
  log "k3s is ready"
  deploy_registry
  seed_fixture_repositories
  build_keel_image
  run_tests
}

main "$@"
