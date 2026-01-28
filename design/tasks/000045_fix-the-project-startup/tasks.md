# Implementation Tasks

- [x] Locate the k3d cluster creation line in `.helix/startup.sh` (around line 156)
- [x] Add `--agents 3` parameter to the `k3d cluster create` command
- [x] Verify the change: command should be `k3d cluster create --agents 3 "$K3D_CLUSTER_NAME" --wait`
- [~] Test: Delete existing cluster if present (`k3d cluster delete keel-dev`)
- [ ] Test: Run the startup script and verify it creates 4 nodes (1 server + 3 agents)
- [ ] Test: Run the startup script again to confirm idempotency (should detect existing cluster)
- [ ] Test: Verify Keel starts successfully and cluster accepts deployments
- [x] Commit the change to the helix-specs branch
- [x] Push to origin helix-specs branch