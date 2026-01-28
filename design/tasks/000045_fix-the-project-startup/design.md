# Design: Fix k3d Cluster Creation in Startup Script

## Overview

Add the `--agents 3` parameter to the k3d cluster creation command in `.helix/startup.sh` to create a multi-node cluster that better simulates a production Kubernetes environment.

## Current Implementation

The script currently creates a single-node k3d cluster:

```bash
k3d cluster create "$K3D_CLUSTER_NAME" --wait
```

This creates:
- 1 server node (control plane)
- 0 agent nodes (workers)

## Proposed Solution

Update the cluster creation command to include agent nodes:

```bash
k3d cluster create --agents 3 "$K3D_CLUSTER_NAME" --wait
```

This will create:
- 1 server node (control plane)
- 3 agent nodes (workers)

## Why 3 Agents?

- **Realistic**: Most production clusters have multiple worker nodes
- **Testing**: Enables testing of pod scheduling, node affinity, and multi-node scenarios
- **Performance**: Distributes workload across nodes
- **Standard**: 3 is a common default for development clusters (similar to minikube's default)

## Technical Details

### File Location
- Path: `.helix/startup.sh`
- Branch: `helix-specs`
- Line: ~156 (in the "Cluster Management" section)

### Change Required
Single line modification in the cluster creation block:

```bash
# Before
k3d cluster create "$K3D_CLUSTER_NAME" --wait

# After
k3d cluster create --agents 3 "$K3D_CLUSTER_NAME" --wait
```

### Idempotency Preserved

The existing idempotency logic remains unchanged:
1. Script checks if cluster exists: `k3d cluster list | grep "$K3D_CLUSTER_NAME"`
2. If exists, verifies it's running and starts it if needed
3. Only creates new cluster if none exists

Since the check happens **before** the create command, existing clusters are unaffected.

## Architecture Decisions

### Decision 1: Number of Agents
**Chosen**: 3 agents  
**Rationale**: 
- Balances resource usage with realistic multi-node behavior
- Standard in k8s community (matches typical dev cluster sizes)
- Enough to test scheduling policies without excessive overhead

**Alternatives Considered**:
- 1 agent: Too minimal, doesn't demonstrate multi-node scenarios
- 5+ agents: Excessive for development, wastes resources

### Decision 2: Modification Approach
**Chosen**: In-place edit of existing line  
**Rationale**:
- Minimal change reduces risk
- No logic changes needed
- Maintains existing error handling and flow

**Alternatives Considered**:
- Adding configurable agent count via environment variable: Over-engineering for this requirement

## Risk Assessment

### Low Risk
- **Change Scope**: Single line modification
- **Idempotency**: Existing clusters unaffected due to pre-creation check
- **Backwards Compat**: Script still works with old clusters

### Potential Issues
- **Resource Usage**: 3 agents use more CPU/memory than 0
  - **Mitigation**: k3d is lightweight; 3 agents acceptable for dev machines
- **Startup Time**: Cluster creation takes slightly longer
  - **Mitigation**: Only affects first run; `--wait` flag already handles this

## Testing Strategy

Manual testing steps:
1. Delete existing cluster: `k3d cluster delete keel-dev`
2. Run startup script: `.helix/startup.sh`
3. Verify cluster has 4 nodes: `kubectl get nodes` (1 server + 3 agents)
4. Run script again to verify idempotency
5. Check that Keel starts successfully and can deploy workloads

## Notes for Future Implementation

- This is a simple fix but important for realistic testing
- The startup script follows a common pattern: check existence → create if missing
- k3d stores cluster state, so the script can safely check if clusters exist
- The helix-specs branch is specifically for .helix configuration files