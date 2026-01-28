# Implementation Tasks

## Prerequisites Setup
- [~] Add function to check if k3s is already installed (`which k3s`)
- [~] Add function to install k3s via official install script (`curl -sfL https://get.k3s.io | sh -`)
- [~] Add function to verify Go is installed (required for building Keel)

## Cluster Management
- [ ] Add function to check if k3s service is running (`systemctl is-active k3s`)
- [ ] Add function to start k3s service if not running (`sudo systemctl start k3s`)
- [ ] Add function to wait for k3s cluster to be ready (API server responding)
- [ ] Copy kubeconfig from `/etc/rancher/k3s/k3s.yaml` to user-accessible location with proper permissions

## Keel Build and Run
- [ ] Checkout master branch to access Keel source code
- [ ] Build Keel binary using `go build` in `cmd/keel` directory
- [ ] Create function to check if Keel is already running (PID file check)
- [ ] Start Keel with `--no-incluster` flag as background process
- [ ] Set environment variables: `KUBERNETES_CONFIG`, `BASIC_AUTH_USER`, `BASIC_AUTH_PASSWORD`
- [ ] Redirect Keel output to `/tmp/keel.log`
- [ ] Store PID in `/tmp/keel.pid`

## Verification
- [ ] Wait for k3s API to be responsive (`kubectl get nodes`)
- [ ] Verify kubectl can list namespaces
- [ ] Wait for Keel to start (check port 9300)
- [ ] Print success message with UI URL

## Idempotency
- [ ] Make k3s install idempotent (skip if already installed)
- [ ] Make k3s service start idempotent (skip if already running)
- [ ] Make Keel startup idempotent (restart if not running, skip if healthy)

## Testing
- [ ] Test script on fresh environment
- [ ] Test script when all components already exist
- [ ] Verify deployments can be created in cluster
- [ ] Verify Keel UI is accessible at localhost:9300