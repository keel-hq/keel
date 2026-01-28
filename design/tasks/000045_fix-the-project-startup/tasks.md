# Implementation Tasks

## Completed
- [x] Locate the k3d cluster creation line in `.helix/startup.sh` (around line 156)
- [x] Add `--agents 3` parameter to the `k3d cluster create` command
- [x] Verify the change: command should be `k3d cluster create --agents 3 "$K3D_CLUSTER_NAME" --wait`
- [x] Test: Delete existing cluster if present (`k3d cluster delete keel-dev`)
- [x] Commit the initial change to the helix-specs branch
- [x] Push to origin helix-specs branch
- [x] Attempted kind - failed due to systemd/cgroup issues in container
- [x] Attempted minikube - K8s version compatibility issues
- [x] Switch to microk8s - better for containerized environments

## In Progress - MicroK8s Implementation
- [x] Replace minikube with microk8s
- [x] Add microk8s installation via snap
- [x] Configure microk8s to start and enable addons (dns, storage)
- [x] Update kubeconfig setup to use microk8s config
- [~] Test: Run the startup script and verify cluster is created
- [ ] Test: Verify kubectl can connect to the cluster
- [ ] Test: Run the startup script again to confirm idempotency
- [ ] Test: Verify Keel starts successfully and cluster accepts deployments
- [ ] Update design.md with implementation notes about k3d→kind→minikube→microk8s journey
- [ ] Commit and push final changes to helix-specs branch