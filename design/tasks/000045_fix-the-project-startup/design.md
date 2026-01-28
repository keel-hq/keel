# Design: Fix Startup Script - Kubernetes Cluster Setup

## Overview

Originally aimed to fix k3d cluster creation by adding `--agents 3` parameter. During implementation, discovered fundamental cgroup v2 limitations when running Kubernetes inside Docker containers without proper cgroup delegation.

## Implementation Journey

### Attempt 1: k3d with --agents 3 ✅ (Initial fix applied)
- Added `--agents 3` parameter to create multi-node cluster
- **Result**: Timeout during cluster creation - k3s couldn't start in nested containers

### Attempt 2: kind (Kubernetes in Docker) ❌
- Switched to kind based on Keel's own e2e tests
- **Result**: Failed with "could not find a log line that matches 'Reached target .*Multi-User System.*|detected cgroup v1'"
- **Root Cause**: Systemd initialization issues in nested containers

### Attempt 3: minikube with docker driver ❌
- Tried minikube as it's widely used for local dev
- **Result**: K8s version compatibility issues, then same nested container problems

### Attempt 4: microk8s ❌
- Attempted microk8s installation
- **Result**: Requires snapd which isn't available in container environments

### Attempt 5: k3s standalone (not k3d) ❌
- Tried running k3s server directly without Docker nesting
- **Result**: `failed to find memory cgroup (v2)` error
- **Root Cause**: cgroup v2 mounted as read-only, and memory controller not enabled

### Attempt 6: k0s ❌
- Tried k0s as it claims better container support
- **Result**: Same pre-flight check failure for memory cgroups

## Root Cause Analysis

### The Cgroup V2 Problem

All Kubernetes distributions require:
1. **Cgroup v2 filesystem** mounted as read-write
2. **Memory controller** enabled in cgroup.controllers
3. Proper cgroup delegation for nested containers

Our environment:
```bash
$ mount | grep cgroup
cgroup on /sys/fs/cgroup type cgroup2 (ro,...)  # READ-ONLY!

$ cat /sys/fs/cgroup/cgroup.controllers
cpuset cpu pids  # MISSING: memory
```

**Why this matters**: We're running inside a Docker container (agent environment). The container's cgroup is controlled by the host Docker daemon and doesn't have:
- Write access to cgroup filesystem
- Memory controller enabled
- Ability to create nested cgroup hierarchies for Kubernetes pods

This is a **host-level Docker configuration issue** that cannot be fixed from inside the container.

## Final Solution: k3s with Cgroup Workaround

The script now uses **k3s standalone** with the following approach:

### Implementation
1. Install k3s without systemd (INSTALL_K3S_SKIP_ENABLE=true)
2. Remount cgroup as read-write if needed
3. Run k3s server directly in background
4. Export kubeconfig for kubectl access

### Current Status
- k3s installs successfully
- Cgroup remount works (ro → rw)
- **Blocker**: Memory controller still not available in cgroup.controllers
- This requires host Docker daemon reconfiguration

## Workaround for Agent Environments

### Option A: Use Host Kubernetes (Recommended)
If the agent is running on a machine with existing Kubernetes:
```bash
# Just use the host's kubeconfig
export KUBECONFIG=/path/to/host/kubeconfig
```

### Option B: External Cluster
Point to a remote cluster (cloud or separate VM):
```bash
# Set up kubeconfig for remote cluster
kubectl config use-context remote-cluster
```

### Option C: Fix Host Docker Configuration
On the **host machine** (not in container), enable cgroup v2 delegation:

1. Edit `/etc/docker/daemon.json`:
```json
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "features": {
    "cgroup-namespaces": true
  }
}
```

2. Restart Docker:
```bash
sudo systemctl restart docker
```

3. Run container with proper cgroup delegation:
```bash
docker run --cgroupns=private --privileged ...
```

## Architecture Decisions

### Decision 1: k3s vs k3d
**Chosen**: k3s standalone  
**Rationale**: 
- Avoids Docker-in-Docker complexity
- Direct control over k3s configuration
- Smaller attack surface

**Trade-off**: Still requires proper cgroup v2 support

### Decision 2: Document Limitations vs Continue Trying
**Chosen**: Document the cgroup limitation clearly  
**Rationale**:
- This is a fundamental infrastructure constraint
- Cannot be solved from inside the container
- Better to document workarounds than provide broken solution

### Decision 3: Fail Fast vs Silently Skip
**Chosen**: Script attempts to start k3s, will fail with clear error  
**Rationale**:
- Users need to know the cluster didn't start
- Error messages point to cgroup issues
- Allows manual troubleshooting

## Testing Strategy

### What Works
- ✅ Go installation
- ✅ kubectl installation  
- ✅ k3s binary installation
- ✅ Cgroup remounting (ro → rw)

### What Doesn't Work (in container)
- ❌ k3s server startup (memory cgroup missing)
- ❌ k3d nested containers (systemd/cgroup issues)
- ❌ kind clusters (systemd initialization)
- ❌ minikube (same issues)

### Testing Checklist
1. ✅ Script installs prerequisites
2. ✅ Script attempts k3s startup
3. ⚠️ k3s fails with documented cgroup error
4. 📝 Document workarounds in script output

## Implementation Notes

### What We Learned
1. **Nested Kubernetes is hard**: Running k8s inside Docker containers requires privileged mode and proper cgroup delegation
2. **Cgroup v2 is strict**: Unlike cgroup v1, v2 requires explicit controller enablement
3. **All distros have same issue**: k3d, kind, k3s, k0s, minikube - all need memory cgroups
4. **Container environments differ**: What works on bare metal doesn't work in containers
5. **Host configuration matters**: The Docker daemon on the host controls what's possible inside

### For Future Cloners
If implementing similar tasks:
- Check if you're in a container: `[ -f /.dockerenv ] && echo "in container"`
- Check cgroup v2 support: `cat /sys/fs/cgroup/cgroup.controllers`
- Consider using external clusters instead of local ones
- Document infrastructure requirements clearly

### Recommended Approach for Production
For agent environments, **don't run local Kubernetes**. Instead:
1. Provision a dedicated k8s cluster (GKE, EKS, etc.)
2. Mount kubeconfig into agent container
3. Let agents connect to external cluster

This avoids all the nested container and cgroup issues.

## Files Modified
- `.helix/startup.sh` - Updated to use k3s with cgroup handling
- `design/tasks/000045_fix-the-project-startup/tasks.md` - Tracked implementation attempts
- `design/tasks/000045_fix-the-project-startup/design.md` - This document

## Final Status
**Status**: Partially implemented with documented limitations

The script now:
- ✅ Installs all prerequisites correctly
- ✅ Attempts to start k3s with best-effort cgroup handling
- ⚠️ Will fail on memory cgroup check (expected in container environments)
- 📝 Provides clear error messages and workarounds

**Next Steps for User**:
1. Use external Kubernetes cluster (recommended)
2. Configure host Docker for cgroup delegation
3. Run agents on bare metal instead of in containers