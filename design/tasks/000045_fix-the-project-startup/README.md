# Task 000045: Fix the Project Startup Script

## Quick Summary

**Original Goal**: Add `--agents 3` parameter to k3d cluster creation for multi-node testing.

**Actual Outcome**: Discovered fundamental infrastructure limitations preventing Kubernetes from running inside Docker containers without proper cgroup v2 support.

## Status: ⚠️ Partially Implemented

- ✅ Script updated with k3s and error handling
- ✅ Comprehensive documentation of limitations
- ✅ Three workarounds provided for users
- ❌ Cannot run local Kubernetes in standard container environments (by design, not a bug)

## What Happened?

We attempted to fix the startup script to create a multi-node k3d cluster. During testing, we discovered that **running Kubernetes inside Docker containers requires cgroup v2 memory controller**, which isn't available in standard container environments.

### Attempts Made
1. **k3d with --agents 3** → Timeout during cluster creation
2. **kind** → Systemd initialization failures
3. **minikube** → Nested container issues
4. **microk8s** → Requires snapd (not available in containers)
5. **k3s standalone** → Memory cgroup controller missing
6. **k0s** → Same cgroup pre-flight check failures

### Root Cause
All Kubernetes distributions require:
- Cgroup v2 filesystem (read-write)
- Memory controller enabled in `/sys/fs/cgroup/cgroup.controllers`
- Proper cgroup delegation from host Docker daemon

**These cannot be configured from inside a container** - they require host-level changes.

## Solutions for Users

### Option 1: Use External Cluster (Recommended)
```bash
# Set up a cloud K8s cluster (GKE, EKS, AKS, etc.)
# Copy kubeconfig to the container
export KUBECONFIG=/path/to/external/kubeconfig
```

### Option 2: Use Host Kubernetes
```bash
# If host has K8s, mount its kubeconfig
docker run -v ~/.kube:/root/.kube:ro ...
```

### Option 3: Fix Host Docker Configuration
On the **host machine** (not in container):

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

3. Run container with cgroup delegation:
```bash
docker run --cgroupns=private --privileged ...
```

## Files in This Directory

- **README.md** (this file) - Quick overview
- **requirements.md** - Original requirements + actual outcome
- **design.md** - Comprehensive journey through all attempts and technical details
- **tasks.md** - Detailed checklist of everything we tried

## Key Learnings

1. **Nested Kubernetes is hard** - Requires privileged mode and proper cgroup delegation
2. **All distros affected** - k3d, kind, k3s, k0s, minikube all need memory cgroups
3. **Infrastructure over code** - This is a platform limitation, not a code bug
4. **Best practice** - For agent environments, use external clusters instead of local ones

## For Future Task Cloners

If implementing similar tasks in other projects:

✅ **Do this:**
- Check if you're in a container: `[ -f /.dockerenv ] && echo "in container"`
- Check cgroup support: `cat /sys/fs/cgroup/cgroup.controllers`
- Consider external clusters from the start
- Document infrastructure requirements clearly

❌ **Avoid this:**
- Assuming local Kubernetes "just works" in containers
- Trying to fix cgroup issues from inside containers
- Over-engineering when simple solutions exist

## Testing the Current Script

The script will attempt to start k3s but will likely fail with memory cgroup errors in containerized environments. This is **expected behavior**.

```bash
# Run the startup script
bash .helix/startup.sh

# If it fails, check the logs
tail -f /tmp/k3s-server.log

# Expected error:
# "Error: failed to find memory cgroup (v2)"
```

## Recommended Production Setup

**Don't run local Kubernetes in agent containers.**

Instead:
1. Provision a dedicated K8s cluster (cloud or VM)
2. Mount kubeconfig into agent container
3. Let agents connect to external cluster

This avoids all nested container and cgroup complexity.

## Questions?

Read the comprehensive documentation in `design.md` for full technical details, or see `tasks.md` for the step-by-step implementation journey.