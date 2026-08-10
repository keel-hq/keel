#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
readonly K3S_VERSION="v1.35.6+k3s1"
readonly K3S_RELEASE_URL="https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION}"
readonly REGISTRY_IMAGE="registry@sha256:46faa9a1ae6813194b53921a370f2f4f8c5e1aae228a89bceafef5847a6a3278"
readonly FIXTURE_IMAGE="docker.io/library/busybox@sha256:7a3ebe5bfd1a4a19797d20b0c0bb39d44393e9a03fd852c0865b0f540d868df0"
readonly PRIOR_CHART_URL="https://github.com/keel-hq/keel/releases/download/keel-v1.0.5/keel-v1.0.5.tgz"
readonly PRIOR_CHART_SHA256="8826a68c962d2641232897997f4fb2b547b6f1d252bffa8e9860113361cfb938"
readonly PRIOR_IMAGE="docker.io/keelhq/keel@sha256:8c7492034b10f3cf718e8ab9f860501aeb5bf13812fa675695ea59c191295fa4"
readonly REGISTRY_ADDRESS="10.53.0.50:5000"
readonly K3S_RUNTIME_DIR="/run/k3s"
readonly DEFAULT_K3S_DATA_DIR="/var/lib/rancher/k3s"
readonly DEFAULT_KUBELET_DIR="/var/lib/kubelet"
# Keep this below the workflow's 20-minute deadline so cleanup and diagnostics can run.
readonly E2E_GO_TEST_TIMEOUT="15m"

RUN_ID="${KEEL_E2E_RUN_ID:-$(date -u +%Y%m%d%H%M%S)-$$}"
RUN_DIR="${KEEL_E2E_RUN_DIR:-${REPO_ROOT}/.test/.runs/${RUN_ID}}"
BIN_DIR="${RUN_DIR}/bin"
ARTIFACT_DIR="${KEEL_E2E_ARTIFACT_DIR:-${REPO_ROOT}/.test/artifacts/${RUN_ID}}"
K3S_BIN="${BIN_DIR}/k3s"
K3S_DATA_DIR="/var/lib/keel-e2e-k3s-${RUN_ID}"
K3S_CLIENT_DATA_DIR="${RUN_DIR}/client-data"
KUBECONFIG="${RUN_DIR}/kubeconfig"
K3S_CONFIG="${RUN_DIR}/k3s.yaml"
REGISTRIES_CONFIG="${RUN_DIR}/registries.yaml"
K3S_PID_FILE="${RUN_DIR}/k3s.pid"
K3S_LOG="${ARTIFACT_DIR}/k3s.log"
PORT_FORWARD_PID_FILE="${RUN_DIR}/port-forward.pid"
E2E_ENV_FILE="${RUN_DIR}/e2e.env"
RELEASE_NAMESPACE="keel-release-${RUN_ID}"
RELEASE_PORT_FORWARD_PID_FILE="${RUN_DIR}/release-port-forward.pid"
PERSISTENCE_CLASS="keel-e2e-${RUN_ID}"
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
  for command in awk curl docker findmnt grep ip jq ps setsid sha256sum sort ss sudo tar; do
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
  [[ "${K3S_DATA_DIR}" == "/var/lib/keel-e2e-k3s-${RUN_ID}" ]] || \
    fail "k3s data directory does not match this run"
  [[ ! -e "${RUN_DIR}" ]] || fail "run directory already exists: ${RUN_DIR}"
  [[ ! -e "${K3S_DATA_DIR}" ]] || fail "run data directory already exists: ${K3S_DATA_DIR}"

  if command -v systemctl >/dev/null && systemctl is-active --quiet k3s 2>/dev/null; then
    fail "an existing k3s service is active"
  fi
  pgrep -x k3s >/dev/null 2>&1 && fail "an existing k3s process is active"
  [[ ! -e /etc/rancher/k3s/k3s.yaml ]] || fail "existing k3s kubeconfig found"
  [[ ! -e "${DEFAULT_K3S_DATA_DIR}" ]] || fail "existing k3s data directory found"
  [[ ! -e "${DEFAULT_KUBELET_DIR}" ]] || fail "existing kubelet data directory found"
  [[ ! -e "${K3S_RUNTIME_DIR}" ]] || fail "existing k3s runtime directory found"
  [[ ! -e "${HOME}/.kube/config" ]] || fail "existing default kubeconfig found"
  ip link show cni0 >/dev/null 2>&1 && fail "existing cni0 interface found"
  ip link show flannel.1 >/dev/null 2>&1 && fail "existing flannel.1 interface found"
  port_is_listening 6443 && fail "Kubernetes API port 6443 is already in use"
  port_is_listening 5000 && fail "registry port 5000 is already in use"
  port_is_listening 19300 && fail "Keel port-forward port 19300 is already in use"
  port_is_listening 19301 && fail "release-validation port-forward port 19301 is already in use"
  port_is_listening 19418 && fail "oauth2-proxy port-forward port 19418 is already in use"
  return 0
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
  env K3S_DATA_DIR="${K3S_CLIENT_DATA_DIR}" "${K3S_BIN}" --version
  log "registry image: ${REGISTRY_IMAGE}"
  log "fixture image: ${FIXTURE_IMAGE}"
}

write_k3s_config() {
  mkdir -p "${ARTIFACT_DIR}"
  sudo install -d -m 0755 -o root -g root "${K3S_DATA_DIR}"
  umask 077
  {
    printf 'data-dir: %s\n' "${K3S_DATA_DIR}"
    printf 'write-kubeconfig: %s\n' "${KUBECONFIG}"
    printf 'write-kubeconfig-mode: "0600"\n'
    printf 'node-name: keel-e2e-%s\n' "${RUN_ID}"
    printf 'cluster-cidr: 10.52.0.0/16\n'
    printf 'service-cidr: 10.53.0.0/16\n'
    printf 'cluster-dns: 10.53.0.10\n'
    printf 'snapshotter: native\n'
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
  local node_name="keel-e2e-${RUN_ID}"
  local pid
  write_k3s_config
  log "starting task-owned k3s process"
  # The calling user owns the artifact log; only the k3s process needs sudo.
  # shellcheck disable=SC2024
  sudo env K3S_DATA_DIR="${K3S_DATA_DIR}" \
    setsid "${K3S_BIN}" server --config "${K3S_CONFIG}" >"${K3S_LOG}" 2>&1 &

  for _ in $(seq 1 30); do
    pid="$(sudo pgrep -f -x "${K3S_BIN} server" || true)"
    if [[ "${pid}" =~ ^[0-9]+$ ]]; then
      printf '%s\n' "${pid}" >"${K3S_PID_FILE}"
      break
    fi
    sleep 1
  done
  [[ -s "${K3S_PID_FILE}" ]] || fail "could not record task-owned k3s process"

  for _ in $(seq 1 120); do
    if [[ -s "${KUBECONFIG}" ]] && sudo env K3S_DATA_DIR="${K3S_DATA_DIR}" "${K3S_BIN}" kubectl \
      --kubeconfig "${KUBECONFIG}" get "node/${node_name}" >/dev/null 2>&1; then
      sudo chown "$(id -u):$(id -g)" "${KUBECONFIG}"
      export KUBECONFIG
      sudo env K3S_DATA_DIR="${K3S_DATA_DIR}" "${K3S_BIN}" kubectl --kubeconfig "${KUBECONFIG}" \
        wait --for=condition=Ready "node/${node_name}" --timeout=120s
      return
    fi
    sleep 1
  done
  log "k3s did not become ready; see ${K3S_LOG}"
  return 1
}

kubectl() {
  env K3S_DATA_DIR="${K3S_CLIENT_DATA_DIR}" \
    "${K3S_BIN}" kubectl --kubeconfig "${KUBECONFIG}" "$@"
}

ctr() {
  sudo env K3S_DATA_DIR="${K3S_DATA_DIR}" "${K3S_BIN}" ctr \
    --address "${K3S_RUNTIME_DIR}/containerd/containerd.sock" \
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
  for _ in $(seq 1 60); do
    if curl --fail --show-error --silent "http://${REGISTRY_ADDRESS}/v2/" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  fail "registry Service did not become reachable at ${REGISTRY_ADDRESS}"
}

build_keel_image() {
  local local_image="${KEEL_E2E_PREBUILT_IMAGE:-keel-e2e:${RUN_ID}}"
  local registry_image="${REGISTRY_ADDRESS}/keel-under-test:${RUN_ID}"
  local image_archive="${RUN_DIR}/keel-image.tar"
  local digest

  if [[ -z "${KEEL_E2E_PREBUILT_IMAGE:-}" ]]; then
    log "building Keel container from ${REPO_ROOT}/Dockerfile"
    docker build --tag "${local_image}" --file "${REPO_ROOT}/Dockerfile" "${REPO_ROOT}"
  else
    docker image inspect "${local_image}" >/dev/null || fail "prebuilt image not found: ${local_image}"
    log "using release-validated image ${local_image}"
  fi
  docker save --output "${image_archive}" "${local_image}"
  ctr images import "${image_archive}"
  ctr images tag "docker.io/library/${local_image}" "${registry_image}"
  ctr images push --plain-http "${registry_image}"

  digest="$(curl --fail --show-error --silent --head \
    --header 'Accept: application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json' \
    "http://${REGISTRY_ADDRESS}/v2/keel-under-test/manifests/${RUN_ID}" | \
    awk 'tolower($1) == "docker-content-digest:" {gsub("\\r", "", $2); print $2}')"
  [[ "${digest}" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "could not resolve Keel image digest"

  {
    printf 'KEEL_E2E_RUN_ID=%q\n' "${RUN_ID}"
    printf 'KEEL_E2E_REGISTRY=%q\n' "${REGISTRY_ADDRESS}"
    printf 'KEEL_E2E_REPOSITORY_PREFIX=%q\n' "${REGISTRY_ADDRESS}/${RUN_ID}"
    printf 'KEEL_E2E_IMAGE=%q\n' "${REGISTRY_ADDRESS}/keel-under-test@${digest}"
    printf 'KEEL_E2E_KUBECTL=%q\n' "${K3S_BIN}"
    printf 'KEEL_E2E_K3S_DATA_DIR=%q\n' "${K3S_CLIENT_DATA_DIR}"
    printf 'KEEL_E2E_KUBECONFIG=%q\n' "${KUBECONFIG}"
    printf 'KEEL_E2E_ARTIFACT_DIR=%q\n' "${ARTIFACT_DIR}"
  } >"${E2E_ENV_FILE}"
  log "Keel image: ${REGISTRY_ADDRESS}/keel-under-test@${digest}"
}

release_helm() {
  env KUBECONFIG="${KUBECONFIG}" K3S_DATA_DIR="${K3S_CLIENT_DATA_DIR}" \
    "${KEEL_E2E_HELM_BIN}" "$@"
}

wait_for_release() {
  kubectl -n "${RELEASE_NAMESPACE}" rollout status deployment/keel --timeout=180s
  # rollout status already waits for the current revision to become available.
  # Do not wait on every pod with app=keel: during an upgrade that selector can
  # include an old, terminating revision which will never become Ready again.
  kubectl -n "${RELEASE_NAMESPACE}" get deployment keel -o json | jq -e '
    (.spec.replicas // 1) > 0 and
    (.status.observedGeneration // 0) >= .metadata.generation and
    (.status.updatedReplicas // 0) == (.spec.replicas // 1) and
    (.status.availableReplicas // 0) == (.spec.replicas // 1)
  ' >/dev/null || fail "Keel Deployment did not converge on the current revision"
  kubectl -n "${RELEASE_NAMESPACE}" get endpoints keel -o json | \
    jq -e '.subsets | any(.addresses | length > 0)' >/dev/null || fail "Keel Service has no ready endpoints"
}

smoke_release_service() {
  local expected_version="$1"
  local diagnostic_name="${2:-${expected_version}}"
  local log_file="${ARTIFACT_DIR}/release-port-forward.log"
  stop_release_port_forward
  kubectl -n "${RELEASE_NAMESPACE}" port-forward service/keel 19301:9300 >"${log_file}" 2>&1 &
  printf '%s\n' "$!" >"${RELEASE_PORT_FORWARD_PID_FILE}"
  for _ in $(seq 1 60); do
    if curl --fail --show-error --silent http://127.0.0.1:19301/healthz >/dev/null 2>&1; then
      curl --fail --show-error --silent http://127.0.0.1:19301/version \
        >"${ARTIFACT_DIR}/release-version-${diagnostic_name//\//-}.json"
      jq -e --arg expected "${expected_version}" '.version == $expected' \
        "${ARTIFACT_DIR}/release-version-${diagnostic_name//\//-}.json" >/dev/null || \
        fail "Service /version did not report the expected value for ${diagnostic_name}"
      stop_release_port_forward
      return 0
    fi
    sleep 1
  done
  fail "Keel Service did not become reachable; see ${log_file}"
}

smoke_oauth_release_service() {
  local log_file="${ARTIFACT_DIR}/release-oauth-port-forward.log"
  local status
  stop_release_port_forward
  kubectl -n "${RELEASE_NAMESPACE}" port-forward service/keel 19301:9300 >"${log_file}" 2>&1 &
  printf '%s\n' "$!" >"${RELEASE_PORT_FORWARD_PID_FILE}"
  for _ in $(seq 1 60); do
    if curl --fail --show-error --silent http://127.0.0.1:19301/ping >/dev/null 2>&1; then
      status="$(curl --silent --output /dev/null --write-out '%{http_code}' \
        --header 'X-Forwarded-User: spoofed@example.test' http://127.0.0.1:19301/version)"
      case "${status}" in
        302|401|403) ;;
        *) fail "OAuth-proxy Service accepted or unexpectedly handled an unauthenticated spoofed identity (HTTP ${status})" ;;
      esac
      stop_release_port_forward
      return 0
    fi
    sleep 1
  done
  fail "OAuth-proxy Service did not become reachable; see ${log_file}"
}

stop_release_port_forward() {
  [[ -s "${RELEASE_PORT_FORWARD_PID_FILE}" ]] || return 0
  local pid
  pid="$(<"${RELEASE_PORT_FORWARD_PID_FILE}")"
  if kill -0 "${pid}" 2>/dev/null; then
    kill -TERM "${pid}" 2>/dev/null || true
    wait "${pid}" 2>/dev/null || true
  fi
  : >"${RELEASE_PORT_FORWARD_PID_FILE}"
}

assert_candidate_objects() {
  local expected_image="${REGISTRY_ADDRESS}/keel-under-test:${RUN_ID}"
  kubectl -n "${RELEASE_NAMESPACE}" get deployment keel -o json >"${ARTIFACT_DIR}/candidate-deployment.json"
  jq -e --arg image "${expected_image}" '
    .spec.template.spec.serviceAccountName == "keel" and
    .spec.template.spec.containers[0].image == $image and
    .spec.template.spec.containers[0].startupProbe.httpGet.path == "/healthz" and
    .spec.template.spec.containers[0].startupProbe.httpGet.port == 9300 and
    .spec.template.spec.containers[0].startupProbe.periodSeconds == 5 and
    .spec.template.spec.containers[0].startupProbe.failureThreshold == 30 and
    .spec.template.spec.containers[0].livenessProbe.httpGet.path == "/healthz" and
    .spec.template.spec.containers[0].readinessProbe.httpGet.path == "/healthz"
  ' "${ARTIFACT_DIR}/candidate-deployment.json" >/dev/null || fail "candidate Deployment wiring is invalid"
  kubectl -n "${RELEASE_NAMESPACE}" get service/keel serviceaccount/keel secret/keel >/dev/null
  kubectl get clusterrole/keel clusterrolebinding/keel >/dev/null
}

assert_oauth_candidate_objects() {
  kubectl -n "${RELEASE_NAMESPACE}" get deployment keel -o json >"${ARTIFACT_DIR}/candidate-oauth-deployment.json"
  kubectl -n "${RELEASE_NAMESPACE}" get service keel -o json >"${ARTIFACT_DIR}/candidate-oauth-service.json"
  jq -e '
    any(.spec.template.spec.containers[]; .name == "oauth2-proxy" and
      .image == "quay.io/oauth2-proxy/oauth2-proxy@sha256:d62e2d81c6f5048f652f67c302083be1272c181b971fad80e5a30ebe2b8b75d8") and
    any(.spec.template.spec.containers[]; .name == "keel" and
      any(.env[]; .name == "AUTH_MODE" and .value == "external-proxy") and
      .startupProbe.httpGet.path == "/ping" and .startupProbe.httpGet.port == 4180 and
      .livenessProbe.httpGet.path == "/ping" and .livenessProbe.httpGet.port == 4180 and
      .readinessProbe.httpGet.path == "/ready" and .readinessProbe.httpGet.port == 4180)
  ' "${ARTIFACT_DIR}/candidate-oauth-deployment.json" >/dev/null || fail "OAuth-proxy candidate Deployment wiring is invalid"
  jq -e '.spec.ports[0].targetPort == 4180' "${ARTIFACT_DIR}/candidate-oauth-service.json" >/dev/null || \
    fail "OAuth-proxy Service does not target the sidecar"
}

uninstall_release() {
  release_helm uninstall keel --namespace "${RELEASE_NAMESPACE}" --wait --timeout 120s
  kubectl delete namespace "${RELEASE_NAMESPACE}" --wait=true --timeout=120s
  kubectl delete persistentvolume --selector "keel.sh/e2e-run=${RUN_ID}" \
    --ignore-not-found --wait=true --timeout=120s
  [[ -z "$(kubectl get clusterrole,clusterrolebinding -o name | grep -E '(^|/)keel$' || true)" ]] || \
    fail "Helm uninstall leaked Keel cluster-scoped RBAC resources"
  [[ -z "$(kubectl get persistentvolume --selector "keel.sh/e2e-run=${RUN_ID}" -o name)" ]] || \
    fail "release validation leaked its persistence fixture"
}

create_persistence_fixture() {
  kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolume
metadata:
  name: keel-release-${RUN_ID}
  labels:
    keel.sh/e2e-run: ${RUN_ID}
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: ${PERSISTENCE_CLASS}
  hostPath:
    path: ${K3S_DATA_DIR}/release-persistence
    type: DirectoryOrCreate
EOF
}

validate_packaged_release() {
  [[ -n "${KEEL_E2E_RELEASE_CHART:-}" ]] || return 0
  [[ -x "${KEEL_E2E_HELM_BIN:-}" ]] || fail "release validation requires a pinned Helm binary"
  [[ -s "${KEEL_E2E_RELEASE_CHART}" ]] || fail "packaged chart not found: ${KEEL_E2E_RELEASE_CHART}"
  local release_scenario="${KEEL_E2E_RELEASE_SCENARIO:-all}"
  local candidate_repository="${REGISTRY_ADDRESS}/keel-under-test"
  local candidate_version="${KEEL_E2E_APP_VERSION}"
  local prior_archive="${RUN_DIR}/keel-v1.0.5.tgz"
  local pvc_uid

  case "${release_scenario}" in
    all|install|upgrade) ;;
    *) fail "KEEL_E2E_RELEASE_SCENARIO must be all, install, or upgrade" ;;
  esac

  if [[ "${release_scenario}" != "upgrade" ]]; then
  log "installing packaged chart with backward-compatible defaults"
  release_helm install keel "${KEEL_E2E_RELEASE_CHART}" --namespace "${RELEASE_NAMESPACE}" --create-namespace \
    --set-string image.repository="${candidate_repository}" --set-string image.tag="${RUN_ID}" \
    --set image.pullPolicy=IfNotPresent --set service.enabled=true --set service.type=ClusterIP \
    --set helmProvider.enabled=false --wait --timeout 180s
  wait_for_release
  assert_candidate_objects
  smoke_release_service "${candidate_version}"
  [[ "$(kubectl -n "${RELEASE_NAMESPACE}" get pvc -o name)" == "" ]] || fail "default values unexpectedly created persistence"
  uninstall_release

  log "installing packaged chart with OAuth-proxy topology"
  kubectl create namespace "${RELEASE_NAMESPACE}"
  kubectl -n "${RELEASE_NAMESPACE}" create secret generic keel-oauth2-proxy \
    --from-literal=OAUTH2_PROXY_CLIENT_ID=release-validator \
    --from-literal=OAUTH2_PROXY_CLIENT_SECRET=not-a-production-secret \
    --from-literal=OAUTH2_PROXY_COOKIE_SECRET=0123456789abcdef0123456789abcdef
  release_helm install keel "${KEEL_E2E_RELEASE_CHART}" --namespace "${RELEASE_NAMESPACE}" \
    --set-string image.repository="${candidate_repository}" --set-string image.tag="${RUN_ID}" \
    --set image.pullPolicy=IfNotPresent --set service.enabled=true --set service.type=ClusterIP \
    --set helmProvider.enabled=false --set auth.mode=external-proxy \
    --set oauth2Proxy.enabled=true --set oauth2Proxy.existingSecret=keel-oauth2-proxy \
    --set-string 'oauth2Proxy.extraArgs[0]=--provider=github' \
    --set-string 'oauth2Proxy.extraArgs[1]=--email-domain=*' \
    --set-string 'oauth2Proxy.extraArgs[2]=--cookie-secure=false' \
    --set-string 'oauth2Proxy.extraArgs[3]=--redirect-url=http://127.0.0.1:19301/oauth2/callback' \
    --wait --timeout 180s
  wait_for_release
  assert_oauth_candidate_objects
  smoke_oauth_release_service
  uninstall_release

  if [[ "${release_scenario}" == "install" ]]; then
    log "packaged default and OAuth-proxy chart installs succeeded"
    return 0
  fi
  fi

  log "installing immutable published chart/image upgrade fixture"
  curl --fail --location --show-error --silent --output "${prior_archive}" "${PRIOR_CHART_URL}"
  printf '%s  %s\n' "${PRIOR_CHART_SHA256}" "${prior_archive}" | sha256sum --check --strict -
  create_persistence_fixture
  release_helm install keel "${prior_archive}" --namespace "${RELEASE_NAMESPACE}" --create-namespace \
    --set-string image.repository="${PRIOR_IMAGE%:*}" --set-string image.tag="${PRIOR_IMAGE##*:}" \
    --set image.pullPolicy=IfNotPresent --set service.enabled=true --set service.type=ClusterIP \
    --set helmProvider.enabled=false --set persistence.enabled=true --set-string persistence.storageClass="${PERSISTENCE_CLASS}" \
    --wait --timeout 180s
  wait_for_release
  # The immutable 0.20.0 image is known to have empty embedded Version/Revision fields.
  smoke_release_service "" "published-0.20.0"
  # Static hostPath volumes do not implement fsGroup ownership changes. Normalize this
  # disposable fixture to the current image UID after proving the prior image wrote state.
  kubectl -n "${RELEASE_NAMESPACE}" exec deployment/keel -- test -s /data/keel.db
  kubectl -n "${RELEASE_NAMESPACE}" exec deployment/keel -- chown -R 666:666 /data
  pvc_uid="$(kubectl -n "${RELEASE_NAMESPACE}" get pvc keel -o jsonpath='{.metadata.uid}')"
  [[ -n "${pvc_uid}" ]] || fail "published release did not create its persistence claim"

  log "upgrading published release to the packaged candidate with production-style values"
  release_helm upgrade keel "${KEEL_E2E_RELEASE_CHART}" --namespace "${RELEASE_NAMESPACE}" \
    --set-string image.repository="${candidate_repository}" --set-string image.tag="${RUN_ID}" \
    --set image.pullPolicy=IfNotPresent --set service.enabled=true --set service.type=ClusterIP \
    --set helmProvider.enabled=false --set persistence.enabled=true --set-string persistence.storageClass="${PERSISTENCE_CLASS}" \
    --set debug=true --set basicauth.enabled=true --set basicauth.user=release-validator \
    --set basicauth.password=not-a-production-secret \
    --set podSecurityContext.fsGroup=666 \
    --set containerSecurityContext.allowPrivilegeEscalation=false \
    --wait --timeout 180s
  wait_for_release
  assert_candidate_objects
  smoke_release_service "${candidate_version}"
  [[ "$(kubectl -n "${RELEASE_NAMESPACE}" get pvc keel -o jsonpath='{.metadata.uid}')" == "${pvc_uid}" ]] || \
    fail "upgrade replaced the persistence claim"
  kubectl -n "${RELEASE_NAMESPACE}" get deployment keel -o json | jq -e '
    any(.spec.template.spec.containers[0].env[]; .name == "DEBUG" and .value == "true") and
    any(.spec.template.spec.containers[0].env[]; .name == "BASIC_AUTH_USER" and .value == "release-validator") and
    .spec.template.spec.securityContext.fsGroup == 666 and
    .spec.template.spec.containers[0].securityContext.allowPrivilegeEscalation == false
  ' >/dev/null || fail "production-style configuration was not preserved in the upgraded Deployment"
  release_helm history keel --namespace "${RELEASE_NAMESPACE}" -o json | jq -e 'length == 2 and .[1].status == "deployed"' >/dev/null || \
    fail "Helm did not record the expected upgrade revision"

  log "rolling back to the supported published revision"
  release_helm rollback keel 1 --namespace "${RELEASE_NAMESPACE}" --wait --timeout 180s
  wait_for_release
  smoke_release_service "" "rollback-0.20.0"
  [[ "$(kubectl -n "${RELEASE_NAMESPACE}" get pvc keel -o jsonpath='{.metadata.uid}')" == "${pvc_uid}" ]] || \
    fail "rollback replaced the persistence claim"
  uninstall_release
  log "packaged chart install, upgrade, rollback, and cleanup succeeded"
}

seed_fixture_repositories() {
  local destination
  log "seeding isolated registry repositories"
  ctr images pull --platform linux/amd64 "${FIXTURE_IMAGE}"
  for destination in \
    "${RUN_ID}/webhook:1.0.0" "${RUN_ID}/webhook:1.0.1" \
    "${RUN_ID}/polling:1.0.0" "${RUN_ID}/polling:1.0.1" \
    "${RUN_ID}/negative:1.0.0" "${RUN_ID}/negative:1.1.0" \
    "${RUN_ID}/webhook-regression:main_0000000000000000000000000000000000000000" \
    "${RUN_ID}/webhook-regression:main_1111111111111111111111111111111111111111" \
    "${RUN_ID}/webhook-regression:main_ffffffffffffffffffffffffffffffffffffffff"; do
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
  if [[ "${KEEL_E2E_MANUAL:-false}" == "true" ]]; then
    go test -count=1 -timeout="${E2E_GO_TEST_TIMEOUT}" -v ./tests \
      -run '^TestE2ESuite/TestExternalOAuthProxyAdminFlow$' 2>&1 | \
      tee "${ARTIFACT_DIR}/go-test.log"
    return
  fi
  go test -count=1 -timeout="${E2E_GO_TEST_TIMEOUT}" -v ./tests 2>&1 | tee "${ARTIFACT_DIR}/go-test.log"
}

hold_for_manual_inspection() {
  [[ "${KEEL_E2E_KEEP_CLUSTER:-false}" == "true" ]] || return 0
  printf '%s\n' "$$" >"${RUN_DIR}/harness.pid"
  log "manual inspection stack is ready"
  log "Admin UI: http://127.0.0.1:19418/"
  log "Dex credentials: alice@example.test / password"
  log "KUBECONFIG: ${KUBECONFIG}"
  log "Artifacts: ${ARTIFACT_DIR}"
  log "Cleanup: kill -INT \"\$(cat '${RUN_DIR}/harness.pid')\""
  while true; do
    sleep 30
  done
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

stop_task_runtime() {
  local endpoint="${K3S_RUNTIME_DIR}/containerd/containerd.sock"
  local pid
  local pids
  pids="$(ps -eo pid=,cmd= | awk -v endpoint="${endpoint}" \
    '$0 ~ /containerd-shim-runc-v2/ && index($0, "-address " endpoint) {print $1}')"
  for pid in ${pids}; do
    sudo kill -TERM "${pid}" 2>/dev/null || true
  done
  for _ in $(seq 1 15); do
    pids="$(ps -eo pid=,cmd= | awk -v endpoint="${endpoint}" \
      '$0 ~ /containerd-shim-runc-v2/ && index($0, "-address " endpoint) {print $1}')"
    [[ -n "${pids}" ]] || return 0
    sleep 1
  done
  for pid in ${pids}; do
    sudo kill -KILL "${pid}" 2>/dev/null || true
  done
}

unmount_task_runtime() {
  local target
  findmnt -rn -o TARGET | awk -v runtime="${K3S_RUNTIME_DIR}" -v data="${K3S_DATA_DIR}" \
    -v kubelet="${DEFAULT_KUBELET_DIR}" \
    '$0 == runtime || index($0, runtime "/") == 1 || $0 == data || index($0, data "/") == 1 || \
      $0 == kubelet || index($0, kubelet "/") == 1' | \
    sort -r | while IFS= read -r target; do
      sudo umount -l "${target}" 2>/dev/null || true
    done
}

collect_diagnostics() {
  mkdir -p "${ARTIFACT_DIR}"
  if [[ -s "${K3S_LOG}" ]]; then
    cp "${K3S_LOG}" "${ARTIFACT_DIR}/k3s-server.log" 2>/dev/null || true
  fi
  if [[ -s "${K3S_DATA_DIR}/agent/containerd/containerd.log" ]]; then
    sudo cp "${K3S_DATA_DIR}/agent/containerd/containerd.log" \
      "${ARTIFACT_DIR}/containerd.log" 2>/dev/null || true
    sudo chown "$(id -u):$(id -g)" "${ARTIFACT_DIR}/containerd.log" 2>/dev/null || true
  elif [[ -s "${K3S_LOG}" ]]; then
    {
      printf 'k3s embeds containerd output; extracted containerd-related server lines follow.\n'
      grep -Ei 'containerd|snapshotter|shim' "${K3S_LOG}" || true
    } >"${ARTIFACT_DIR}/containerd.log"
  fi
  [[ -x "${K3S_BIN}" && -s "${KUBECONFIG}" ]] || return 0

  kubectl get nodes -o wide >"${ARTIFACT_DIR}/nodes.txt" 2>&1 || true
  kubectl get pods -A -o wide >"${ARTIFACT_DIR}/pods.txt" 2>&1 || true
  kubectl get deployments,statefulsets,daemonsets,jobs,cronjobs,services -A -o wide \
    >"${ARTIFACT_DIR}/resources.txt" 2>&1 || true
  kubectl describe nodes >"${ARTIFACT_DIR}/describe-nodes.txt" 2>&1 || true
  kubectl describe pods -A >"${ARTIFACT_DIR}/describe-pods.txt" 2>&1 || true
  kubectl get events -A --sort-by=.metadata.creationTimestamp \
    >"${ARTIFACT_DIR}/events.txt" 2>&1 || true
  kubectl -n "keel-e2e-infra-${RUN_ID}" logs deployment/registry --all-containers=true \
    >"${ARTIFACT_DIR}/registry.log" 2>&1 || true
  kubectl -n "keel-e2e-system-${RUN_ID}" logs deployment/keel --all-containers=true \
    >"${ARTIFACT_DIR}/keel.log" 2>&1 || true
  kubectl -n "keel-e2e-system-${RUN_ID}" logs deployment/keel --all-containers=true --previous \
    >"${ARTIFACT_DIR}/keel-previous.log" 2>&1 || true
  if [[ -n "${KEEL_E2E_HELM_BIN:-}" && -x "${KEEL_E2E_HELM_BIN}" ]]; then
    release_helm status keel --namespace "${RELEASE_NAMESPACE}" --show-resources \
      >"${ARTIFACT_DIR}/release-helm-status.txt" 2>&1 || true
    release_helm history keel --namespace "${RELEASE_NAMESPACE}" \
      >"${ARTIFACT_DIR}/release-helm-history.txt" 2>&1 || true
    release_helm get values keel --namespace "${RELEASE_NAMESPACE}" --all \
      >"${ARTIFACT_DIR}/release-helm-values.yaml" 2>&1 || true
    release_helm get manifest keel --namespace "${RELEASE_NAMESPACE}" \
      >"${ARTIFACT_DIR}/release-helm-manifest.yaml" 2>&1 || true
    kubectl -n "${RELEASE_NAMESPACE}" get all,pvc,serviceaccount,role,rolebinding -o wide \
      >"${ARTIFACT_DIR}/release-resources.txt" 2>&1 || true
    kubectl -n "${RELEASE_NAMESPACE}" describe pods \
      >"${ARTIFACT_DIR}/release-describe-pods.txt" 2>&1 || true
    kubectl -n "${RELEASE_NAMESPACE}" logs deployment/keel --all-containers=true \
      >"${ARTIFACT_DIR}/release-keel.log" 2>&1 || true
  fi
  kubectl -n "keel-e2e-system-${RUN_ID}" logs deployment/keel-oauth -c keel \
    >"${ARTIFACT_DIR}/keel-oauth.log" 2>&1 || true
  kubectl -n "keel-e2e-system-${RUN_ID}" logs deployment/keel-oauth -c oauth2-proxy \
    >"${ARTIFACT_DIR}/oauth2-proxy.log" 2>&1 || true
  kubectl -n "keel-e2e-system-${RUN_ID}" logs deployment/dex \
    >"${ARTIFACT_DIR}/dex.log" 2>&1 || true
}

delete_suite_resources() {
  [[ -x "${K3S_BIN}" && -s "${KUBECONFIG}" ]] || return 0
  env K3S_DATA_DIR="${K3S_CLIENT_DATA_DIR}" \
    "${K3S_BIN}" kubectl --kubeconfig "${KUBECONFIG}" delete \
    namespaces,clusterroles,clusterrolebindings \
    --selector "keel.sh/e2e-run=${RUN_ID}" --ignore-not-found --wait=true --timeout=60s \
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

stop_oauth_port_forward() {
  local pid_file="${RUN_DIR}/oauth-port-forward.pid"
  [[ -s "${pid_file}" ]] || return 0
  local pid
  pid="$(<"${pid_file}")"
  if kill -0 "${pid}" 2>/dev/null; then
    kill -TERM "${pid}" 2>/dev/null || true
    wait "${pid}" 2>/dev/null || true
  fi
}

cleanup() {
  local exit_status=$?
  ((CLEANUP_STARTED == 0)) || return "${exit_status}"
  CLEANUP_STARTED=1
  # Teardown is best-effort and must not turn a successful test run into a
  # failure. In particular, k3s may remove runtime paths concurrently while
  # the EXIT trap is deleting them.
  set +e
  if ((exit_status != 0)); then
    log "collecting diagnostics after exit status ${exit_status}"
    collect_diagnostics
  fi
  stop_release_port_forward
  delete_suite_resources
  stop_port_forward
  stop_oauth_port_forward
  stop_k3s
  stop_task_runtime
  unmount_task_runtime
  if ip link show cni0 >/dev/null 2>&1; then
    sudo ip link delete cni0 2>/dev/null || true
  fi
  if ip link show flannel.1 >/dev/null 2>&1; then
    sudo ip link delete flannel.1 2>/dev/null || true
  fi
  if [[ -d "${K3S_RUNTIME_DIR}" ]] && ! pgrep -x k3s >/dev/null 2>&1; then
    sudo find "${K3S_RUNTIME_DIR}" -depth -delete
  fi
  if [[ -d "${DEFAULT_K3S_DATA_DIR}" ]] && ! pgrep -x k3s >/dev/null 2>&1; then
    sudo find "${DEFAULT_K3S_DATA_DIR}" -depth -delete
  fi
  if [[ -d "${DEFAULT_KUBELET_DIR}" ]] && ! pgrep -x k3s >/dev/null 2>&1; then
    sudo find "${DEFAULT_KUBELET_DIR}" -depth -delete
  fi
  if [[ -d "${K3S_DATA_DIR}" ]] && ! pgrep -x k3s >/dev/null 2>&1; then
    sudo find "${K3S_DATA_DIR}" -depth -delete
  fi
  if [[ -d "${RUN_DIR}" ]]; then
    sudo find "${RUN_DIR}" -depth -delete
  fi
  return "${exit_status}"
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
  validate_packaged_release
  if [[ "${KEEL_E2E_RELEASE_ONLY:-false}" != "true" ]]; then
    run_tests
  fi
  hold_for_manual_inspection
}

main "$@"
