# Requirements: Fix k3d Cluster Creation in Startup Script

## Problem Statement

The current startup script at `.helix/startup.sh` creates a k3d cluster without specifying agent nodes. According to k3d best practices and the user's requirement, clusters should be created with explicit agent nodes for proper workload distribution.

Current command:
```bash
k3d cluster create "$K3D_CLUSTER_NAME" --wait
```

Required command:
```bash
k3d cluster create --agents 3 "$K3D_CLUSTER_NAME" --wait
```

## User Stories

**As a** developer starting work on the Keel project  
**I want** the startup script to create a k3d cluster with multiple agent nodes  
**So that** I have a realistic multi-node environment for testing Kubernetes deployments

## Acceptance Criteria

1. ✅ The `k3d cluster create` command includes `--agents 3` parameter
2. ✅ The script remains idempotent - running it multiple times doesn't break existing clusters
3. ✅ The script still checks if a cluster already exists before attempting creation
4. ✅ The fix is applied to `.helix/startup.sh` in the helix-specs branch
5. ✅ All other functionality of the startup script remains unchanged

## Non-Functional Requirements

- **Idempotency**: Script must be safe to run multiple times
- **Backwards Compatibility**: Existing clusters named `keel-dev` should continue to work
- **Clarity**: The change should be simple and obvious in the code

## Out of Scope

- Changing the cluster name or other cluster configuration
- Modifying the Keel build or deployment process
- Adding new features to the startup script