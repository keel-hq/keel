# Design: Fix Project Startup Script

## Architecture Overview

The startup script uses k3d (k3s in Docker) to create a lightweight Kubernetes cluster, then builds and runs Keel outside the cluster using an external kubeconfig.

```
┌─────────────────────────────────────────────────────┐
│                   Host Machine                       │
│                                                      │
│  ┌──────────────┐      ┌─────────────────────────┐  │
│  │   Keel       │      │   k3d Container         │  │
│  │ (native Go)  │─────▶│   ┌─────────────────┐   │  │
│  │              │      │   │  k3s cluster    │   │  │
│  │ Port 9300    │      │   │  (single node)  │   │  │
│  └──────────────┘      │   └─────────────────┘   │  │
│         │              │          │              │  │
│         │              │   Port 6443 (API)       │  │
│         ▼              └─────────────────────────┘  │
│  ~/.kube/config                                      │
│  (k3d-keel-dev)                                      │
└─────────────────────────────────────────────────────┘
```

## Key Decisions

### Decision 1: Use k3d over k3s native
**Rationale**: Docker is already available on the system. k3d runs k3s inside Docker containers, making it easier to manage, clean up, and doesn't require systemd or elevated privileges for installation.

### Decision 2: Run Keel natively (not in cluster)
**Rationale**: For development, running Keel outside the cluster with `--no-incluster` allows:
- Faster iteration (no image builds needed)
- Easy debugging with local tools
- Direct access to logs and UI

### Decision 3: Use background process with PID file
**Rationale**: The startup script cannot run indefinitely. Keel will be started as a background process with its PID stored in `/tmp/keel.pid` for management.

## Component Details

### 1. Prerequisites Installation
- **k3d**: Downloaded from GitHub releases if missing
- **kubectl**: Downloaded from Kubernetes releases if missing
- **Go**: Required for building Keel (assumed present)

### 2. Cluster Configuration
- **Name**: `keel-dev`
- **Ports**: API server on random port, mapped automatically by k3d
- **Kubeconfig**: Merged into `~/.kube/config` with context `k3d-keel-dev`

### 3. Keel Configuration
- **Build**: `go build` in `cmd/keel` directory
- **Run flags**: `--no-incluster` to use external kubeconfig
- **Environment**:
  - `KUBERNETES_CONFIG=~/.kube/config`
  - `KUBERNETES_CONTEXT=k3d-keel-dev`
  - `BASIC_AUTH_USER=admin`
  - `BASIC_AUTH_PASSWORD=admin`

### 4. Idempotency Strategy
| Component | Check | Action if exists | Action if missing |
|-----------|-------|------------------|-------------------|
| k3d binary | `which k3d` | Skip install | Install |
| kubectl binary | `which kubectl` | Skip install | Install |
| k3d cluster | `k3d cluster list` | Skip create | Create |
| Keel process | Check PID file | Verify running | Start |

## File Locations

| Item | Path |
|------|------|
| Startup script | `.helix/startup.sh` |
| Keel binary | `cmd/keel/keel` |
| Keel PID file | `/tmp/keel.pid` |
| Keel log file | `/tmp/keel.log` |
| Kubeconfig | `~/.kube/config` |

## Codebase Patterns Discovered

- Keel uses `kingpin` for CLI flags
- `--no-incluster` flag enables external kubeconfig mode
- Default UI port is 9300 (from `types.KeelDefaultPort`)
- Environment variables: `KUBERNETES_CONFIG`, `KUBERNETES_CONTEXT` override defaults