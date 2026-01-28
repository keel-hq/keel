# Design: Fix Project Startup Script

## Architecture Overview

The startup script uses k3s (lightweight Kubernetes) to create a single-node cluster, then builds and runs Keel outside the cluster using an external kubeconfig.

```
┌─────────────────────────────────────────────────────┐
│                   Host Machine                       │
│                                                      │
│  ┌──────────────┐      ┌─────────────────────────┐  │
│  │   Keel       │      │   k3s Service           │  │
│  │ (native Go)  │─────▶│   ┌─────────────────┐   │  │
│  │              │      │   │  k3s cluster    │   │  │
│  │ Port 9300    │      │   │  (single node)  │   │  │
│  └──────────────┘      │   └─────────────────┘   │  │
│         │              │          │              │  │
│         │              │   Port 6443 (API)       │  │
│         ▼              └─────────────────────────┘  │
│  /etc/rancher/k3s/k3s.yaml                          │
│  (kubeconfig)                                        │
└─────────────────────────────────────────────────────┘
```

## Key Decisions

### Decision 1: Use k3s directly
**Rationale**: k3s is a lightweight, certified Kubernetes distribution. It installs via a simple script, runs as a systemd service, and includes kubectl. Perfect for development environments.

### Decision 2: Run Keel natively (not in cluster)
**Rationale**: For development, running Keel outside the cluster with `--no-incluster` allows:
- Faster iteration (no image builds needed)
- Easy debugging with local tools
- Direct access to logs and UI

### Decision 3: Use background process with PID file
**Rationale**: The startup script cannot run indefinitely. Keel will be started as a background process with its PID stored in `/tmp/keel.pid` for management.

## Component Details

### 1. k3s Installation
- **Install method**: `curl -sfL https://get.k3s.io | sh -`
- **Service**: Runs as systemd service, auto-restarts
- **Kubeconfig**: Written to `/etc/rancher/k3s/k3s.yaml`
- **kubectl**: Included with k3s installation

### 2. Cluster Configuration
- **Type**: Single-node server (fully functional cluster)
- **API Server**: Port 6443 on localhost
- **Kubeconfig**: `/etc/rancher/k3s/k3s.yaml` (requires sudo to read, or copy with proper permissions)

### 3. Keel Configuration
- **Build**: `go build` in `cmd/keel` directory
- **Run flags**: `--no-incluster` to use external kubeconfig
- **Environment**:
  - `KUBERNETES_CONFIG=/etc/rancher/k3s/k3s.yaml` (or copied to user-accessible location)
  - `BASIC_AUTH_USER=admin`
  - `BASIC_AUTH_PASSWORD=admin`

### 4. Idempotency Strategy
| Component | Check | Action if exists | Action if missing |
|-----------|-------|------------------|-------------------|
| k3s | `which k3s` or `systemctl is-active k3s` | Ensure running | Install |
| k3s service | `systemctl is-active k3s` | Skip start | Start service |
| Keel process | Check PID file | Verify running | Start |

## File Locations

| Item | Path |
|------|------|
| Startup script | `.helix/startup.sh` |
| k3s kubeconfig | `/etc/rancher/k3s/k3s.yaml` |
| User kubeconfig copy | `~/.kube/config` (optional) |
| Keel binary | `cmd/keel/keel` |
| Keel PID file | `/tmp/keel.pid` |
| Keel log file | `/tmp/keel.log` |

## Codebase Patterns Discovered

- Keel uses `kingpin` for CLI flags
- `--no-incluster` flag enables external kubeconfig mode
- Default UI port is 9300 (from `types.KeelDefaultPort`)
- Environment variables: `KUBERNETES_CONFIG`, `KUBERNETES_CONTEXT` override defaults
- k3s writes kubeconfig to `/etc/rancher/k3s/k3s.yaml` with root ownership