# Implementation Tasks

## Completed
- [x] Locate the k3d cluster creation line in `.helix/startup.sh` (around line 156)
- [x] Add `--agents 3` parameter to the `k3d cluster create` command
- [x] Verify the change: command should be `k3d cluster create --agents 3 "$K3D_CLUSTER_NAME" --wait`
- [x] Test: Delete existing cluster if present (`k3d cluster delete keel-dev`)
- [x] Commit the initial change to the helix-specs branch
- [x] Push to origin helix-specs branch

## In Progress - Switch to Kind
- [~] Replace k3d with kind based on `.test/e2e-kind.sh` pattern
- [ ] Remove k3d installation and cluster creation code
- [ ] Add kind installation (download from storage.googleapis.com)
- [ ] Add kind cluster creation with appropriate node image
- [ ] Update kubeconfig setup to use kind's kubeconfig
- [ ] Test: Run the startup script and verify cluster is created
- [ ] Test: Verify kubectl can connect to the cluster
- [ ] Test: Run the startup script again to confirm idempotency
- [ ] Test: Verify Keel starts successfully and cluster accepts deployments
- [ ] Update design.md with implementation notes about the switch
- [ ] Commit and push final changes to helix-specs branch