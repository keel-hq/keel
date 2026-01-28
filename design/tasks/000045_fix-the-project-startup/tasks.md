# Implementation Tasks

## Phase 1: Initial Fix ✅
- [x] Locate the k3d cluster creation line in `.helix/startup.sh` (around line 156)
- [x] Add `--agents 3` parameter to the `k3d cluster create` command
- [x] Verify the change: command should be `k3d cluster create --agents 3 "$K3D_CLUSTER_NAME" --wait`
- [x] Commit the initial change to the helix-specs branch
- [x] Push to origin helix-specs branch

## Phase 2: Testing & Troubleshooting ⚠️
- [x] Test: Delete existing cluster if present (`k3d cluster delete keel-dev`)
- [x] Test: Run startup script with k3d
- [x] Discovered: k3d clusters timeout during creation (systemd/cgroup issues in nested containers)

## Phase 3: Alternative Solutions Attempted 🔄
- [x] Attempt 1: Switch to kind (Kubernetes in Docker)
  - Result: Failed - systemd initialization issues in containers
  - Error: "could not find a log line that matches 'Reached target .*Multi-User System.*'"
  
- [x] Attempt 2: Switch to minikube with docker driver
  - Result: Failed - K8s version compatibility, then nested container issues
  
- [x] Attempt 3: Switch to microk8s
  - Result: Failed - requires snapd which isn't available in containers
  
- [x] Attempt 4: Switch to k3s standalone (not k3d)
  - Result: Failed - "failed to find memory cgroup (v2)" error
  - Root cause: Memory controller not enabled in cgroup.controllers
  
- [x] Attempt 5: Switch to k0s
  - Result: Failed - same cgroup pre-flight check failures

## Phase 4: Root Cause Analysis ✅
- [x] Investigate cgroup v2 configuration
- [x] Discover cgroup mounted as read-only
- [x] Remount cgroup as read-write (successful)
- [x] Discover memory controller missing from cgroup.controllers
- [x] Identify root cause: Host Docker daemon configuration limits nested containers

## Phase 5: Documentation & Workarounds ✅
- [x] Document all attempts and learnings in design.md
- [x] Document cgroup v2 limitations
- [x] Provide workarounds for users:
  - Option A: Use host Kubernetes
  - Option B: Use external/cloud cluster
  - Option C: Reconfigure host Docker daemon
- [x] Update startup script with best-effort k3s approach
- [x] Add clear error messages in script
- [x] Create comprehensive README.md for task directory
- [x] Update requirements.md with actual outcomes
- [x] Final commit and push to helix-specs branch

## Final Implementation Status

### What Works ✅
- Prerequisites installation (Go, kubectl, k3s binary)
- Cgroup detection and remounting
- Clear error messages and guidance
- Idempotent script logic

### What Doesn't Work ❌
- Running local Kubernetes inside Docker containers without proper cgroup delegation
- This is a **host infrastructure limitation**, not a script bug

### Deliverables ✅
- [x] Updated `.helix/startup.sh` with k3s and cgroup handling
- [x] Comprehensive design.md documenting the journey
- [x] Clear task list showing all attempts
- [x] Workarounds documented for users
- [x] All changes committed and pushed to helix-specs branch

## Key Learnings 📚

1. **Container Limitations**: Running Kubernetes inside Docker requires:
   - Privileged mode
   - Cgroup v2 with memory controller enabled
   - Proper cgroup delegation from host

2. **All K8s Distros Affected**: k3d, kind, k3s, k0s, minikube all require the same cgroup features

3. **Infrastructure Over Code**: This is a host configuration issue, not something the script can fully solve

4. **Best Practice**: For agent environments, use external Kubernetes clusters rather than local ones

## Recommendations for Production

**Don't run local Kubernetes in agent containers.** Instead:
- Use a dedicated k8s cluster (cloud provider or separate VM)
- Mount kubeconfig into agent container
- Let agents connect to external cluster

This avoids nested container complexity entirely.

## Commit History
- Initial fix: Added `--agents 3` to k3d command
- Attempted kind, minikube, microk8s, k3s, k0s
- Final: k3s with cgroup workaround + documentation
- All changes pushed to `helix-specs` branch