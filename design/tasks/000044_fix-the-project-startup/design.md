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
│  (k3d-keel-dev context)                              │
└─────────────────────────────────────────────────────┘
```

## Key Decisions

### Decision 1: Use k3d (k3s in Docker) instead of k3s directly
**Rationale**: The development environment runs inside a container without proper cgroup support. k3s directly fails with `failed to find memory cgroup (v2)`. k3d runs k3s inside Docker containers, bypassing cgroup requirements on the host.

**Discovery**: Attempted k3s direct installation first, but it failed due to container environment limitations:
```
level=fatal msg="Error: failed to find memory cgroup (v2)"
```

### Decision 2: Run Keel natively (not in cluster)
**Rationale**: For development, running Keel outside the cluster with `--no-incluster` allows:
- Faster iteration (no image builds needed)
- Easy debugging with local tools
- Direct access to logs and UI

### Decision 3: Use background process with PID file
**Rationale**: The startup script cannot run indefinitely. Keel will be started as a background process with its PID stored in `/tmp/keel.pid` for management.

## Component Details

### 1. k3d Installation
- **Install method**: `curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash`
- **Prerequisite**: Docker must be available
- **kubectl**: Installed separately if needed

### 2. Cluster Configuration
- **Name**: `keel-dev`
- **Type**: Single-node k3s inside Docker container
- **API Server**: Exposed via k3d on random port, accessible via kubeconfig
- **Kubeconfig**: Merged into `~/.kube/config` with context `k3d-keel-dev`

### 3. Keel Configuration
- **Build**: `go build` in `cmd/keel` directory
- **Run flags**: `--no-incluster` to use external kubeconfig
- **Environment**:
  - `KUBECONFIG=~/.kube/config`
  - `BASIC_AUTH_USER=admin`
  - `BASIC_AUTH_PASSWORD=admin`

### 4. Idempotency Strategy
| Component | Check | Action if exists | Action if missing |
|-----------|-------|------------------|-------------------|
| Go | `which go` | Skip install | Install |
| k3d binary | `which k3d` | Skip install | Install |
| kubectl binary | `which kubectl` | Skip install | Install |
| k3d cluster | `k3d cluster list \| grep keel-dev` | Skip create | Create |
| Keel process | Check PID file | Verify running | Start |

## File Locations

| Item | Path |
|------|------|
| Startup script | `.helix/startup.sh` |
| Kubeconfig | `~/.kube/config` |
| Keel binary | `/tmp/keel-source/cmd/keel/keel` |
| Keel PID file | `/tmp/keel.pid` |
| Keel log file | `/tmp/keel.log` |

## Codebase Patterns Discovered

- Keel uses `kingpin` for CLI flags
- `--no-incluster` flag enables external kubeconfig mode
- Default UI port is 9300 (from `types.KeelDefaultPort`)
- Environment variables: `KUBECONFIG` is used by kubectl and Keel
- k3d automatically manages kubeconfig merging

## Implementation Notes

- **Blocker discovered**: k3s direct install fails in container environments without cgroup v2 support
- **Solution**: Switched to k3d which runs k3s inside Docker, avoiding host cgroup requirements
- Go 1.21.6 installed automatically if not present
- Script clones master branch to `/tmp/keel-source` since helix-specs branch doesn't have source code