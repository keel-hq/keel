---
name: keel-dev-stack
description: Start, verify, and tear down the Keel local dev environment — k3s cluster via k3d, the Keel backend (API/webhook server), and the React UI (built static assets served by Keel and/or the Vite dev server). Use when asked to launch the dev server, start the UI/dashboard, start k3s/k3d with Keel, or fix the dashboard not opening.
license: MIT
compatibility: agents
metadata:
  audience: developers
  project: keel
  ports: "6443,8000,9300"
---

## What this skill does

Operates the local Keel development stack from this repository:

1. **k3s cluster** — a k3s-in-Docker cluster created with **k3d** (named `k3d-dev-cluster`, API server on `https://0.0.0.0:6443`).
2. **Keel backend** — the Go binary `keel --no-incluster --ui-dir ui/dist` started **with basic auth enabled** (`BASIC_AUTH_USER`/`BASIC_AUTH_PASSWORD`). It connects to the cluster via the local kubeconfig and serves the API/webhooks/dashboard on **port 9300**. **Without auth enabled, Keel does NOT register the dashboard routes at all** (every URL, including `/`, returns Go's `404 page not found`) — this is the #1 cause of "dashboard doesn't open".
3. **Keel UI** — React/Vite app in `ui/`. Built assets in `ui/dist/` are served by Keel at `http://localhost:9300/` (dashboard). The Vite dev server runs on **port 8000** and proxies `/v1` to `localhost:9300`.

All commands below assume you are at the repository root (`git rev-parse --show-toplevel`).

## Ports / URLs

| Thing | URL / command |
|---|---|
| Kubernetes API (k3d/k3s) | `https://0.0.0.0:6443` |
| Keel backend + dashboard | `http://localhost:9300/` |
| Vite UI dev server | `http://localhost:8000/` |

## Start the stack

### 1. Start k3s cluster (k3d)

```bash
k3d cluster create k3d-dev-cluster --api-port 0.0.0.0:6443 --agents 1 --wait
mkdir -p ~/.kube
k3d kubeconfig get k3d-dev-cluster > ~/.kube/config && chmod 600 ~/.kube/config
kubectl cluster-info && kubectl get nodes
```

If the cluster already exists (`k3d cluster list` shows it), skip creation; re-point the kubeconfig with `k3d kubeconfig get`. On a machine without k3d, install it (`curl -fsSL https://github.com/k3d-io/k3d/releases/download/v5.9.0/k3d-linux-amd64 -o k3d && chmod +x k3d && sudo install k3d /usr/local/bin/k3d`).

### 2. Build/install the Keel backend

Go toolchain auto-downloads (`go.mod` requires `go 1.26.5`; `GOTOOLCHAIN=auto` handles it):

```bash
go build ./cmd/keel   # or: go install github.com/keel-hq/keel/cmd/keel
```

**Gotcha 1 (PATH):** `go install` puts the binary in `$(go env GOPATH)/bin` (= `~/go/bin` on Linux), which may **not be on PATH** on some machines. `make run` then fails with `keel: No such file or directory`. Launch the binary by full path (`$(go env GOPATH)/bin/keel`) instead.

**Gotcha 2 (auth, critical):** start Keel with `BASIC_AUTH_USER`/`BASIC_AUTH_PASSWORD` set, otherwise `pkg/http` skips registering the admin + UI routes (log line: `authentication is not enabled, admin HTTP handlers are not initialized`) and the dashboard 404s on every path. Launch command:

```bash
KEEL_BIN="$(go env GOPATH)/bin/keel"
setsid nohup env BASIC_AUTH_USER=admin BASIC_AUTH_PASSWORD=admin \
  "$KEEL_BIN" --no-incluster --ui-dir ui/dist > /tmp/keel-dev-server.log 2>&1 < /dev/null &
```

Verify: `ss -ltnH | grep 9300` and `tail /tmp/keel-dev-server.log` (expect `authentication enabled, setting up admin HTTP handlers`, `webhook trigger server starting... address=":9300"`, and resource watches started = cluster connection OK).

API creds are then `admin` / `admin` (Keel API requires HTTP basic auth: e.g. `curl -u admin:admin http://localhost:9300/v1/stats`).

### 3. Build the UI (so the dashboard works on :9300)

`ui/dist` does not exist by default, so the dashboard cannot be served until you build it.

```bash
cd ui
npm ci --no-audit --no-fund
npm run build   # emits ui/dist/
cd ..
```

Then reload `http://localhost:9300/`.

**Gotcha:** if `npm ci` fails with an EACCES/ownership error, the npm cache dir (e.g. `~/.npm`) may be root-owned. Fix with `sudo chown -R $(id -u):$(id -g) ~/.npm` and retry.

### 4. (Optional) Vite dev server on :8000

```bash
cd ui && npm run dev -- --host 0.0.0.0   # or: make run-ui in repo root
```

Serves the app at `http://localhost:8000/` and proxies `/v1` to `localhost:9300`.

**Gotcha:** Vite binds to loopback (`[::1]:8000`) by default — pass `--host 0.0.0.0` to make it reachable from outside the machine (`http://<host-ip>:8000/`). Keel (`:9300`) and the k3s API (`:6443`) already bind to `0.0.0.0` out of the box; external access to `:9300`/`:6443` may still be gated by host firewall/security groups.

## Verification checklist

- [ ] `kubectl get nodes` → all Ready (k3s, e.g. `v1.35.5+k3s1`)
- [ ] `ss -ltnH | grep 9300` → listening; Keel log shows `authentication enabled, setting up admin HTTP handlers` and no fatal errors
- [ ] `curl -s -o /dev/null -w "%{http_code}" http://localhost:9300/` → 200 (requires `ui/dist` built **and** auth enabled), not 404
- [ ] `curl -s -o /dev/null -w "%{http_code}" -u admin:admin http://localhost:9300/v1/stats` → 200 (401 without creds)
- [ ] `curl -s -o /dev/null -w "%{http_code}" http://localhost:8000/` → 200 (Vite dev server)
- [ ] Keel log shows started watches for deployments/daemonsets/cronjobs/statefulsets (proves it is talking to the k3s cluster)

## Teardown

```bash
pkill -f "keel --no-incluster"   # stop backend (or kill the recorded PID)
pkill -f "vite"                   # stop UI dev server if running
k3d cluster delete k3d-dev-cluster  # remove the k3s cluster
```

## Logs

- Backend: `/tmp/keel-dev-server.log`
- UI npm install: `/tmp/keel-ui-npm-ci.log`

## Notes / pitfalls

- `~/go/bin` may not be on `PATH`; call the keel binary by absolute path (`$(go env GOPATH)/bin/keel`).
- `make run` = `go install ...` + `keel --no-incluster --ui-dir ui/dist`; it works only if the Go bin dir is on PATH or `keel` resolves.
- **Dashboard 404 on `:9300` is almost always one of two things:** (1) `ui/dist` not built, or (2) Keel started without `BASIC_AUTH_USER`/`BASIC_AUTH_PASSWORD`, so the UI routes are never registered. Check the log for `authentication is not enabled`.
- npm cache ownership can break `npm ci`; fix with `sudo chown -R $(id -u):$(id -g) ~/.npm`.
