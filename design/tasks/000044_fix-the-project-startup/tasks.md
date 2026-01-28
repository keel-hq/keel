# Implementation Tasks

## Prerequisites Setup
- [x] Add function to check if Go is already installed (`which go`)
- [x] Add function to install Go if not present
- [~] Add function to check and install k3d (`which k3d`)
- [~] Add function to check and install kubectl (`which kubectl`)

## Cluster Management
- [~] Add function to check if k3d cluster "keel-dev" exists
- [~] Add function to create k3d cluster if it doesn't exist
- [ ] Add function to wait for k3d cluster to be ready (API server responding)
- [ ] Verify kubeconfig is properly configured for k3d-keel-dev context

## Keel Build and Run
- [ ] Clone master branch to /tmp/keel-source to access Keel source code
- [ ] Build Keel binary using `go build` in `cmd/keel` directory
- [ ] Create function to check if Keel is already running (PID file check)
- [ ] Start Keel with `--no-incluster` flag as background process
- [ ] Set environment variables: `KUBECONFIG`, `BASIC_AUTH_USER`, `BASIC_AUTH_PASSWORD`
- [ ] Redirect Keel output to `/tmp/keel.log`
- [ ] Store PID in `/tmp/keel.pid`

## Verification
- [ ] Wait for k3d API to be responsive (`kubectl get nodes`)
- [ ] Verify kubectl can list namespaces
- [ ] Wait for Keel to start (check port 9300)
- [ ] Print success message with UI URL

## Idempotency
- [~] Make Go install idempotent (skip if exists)
- [~] Make k3d install idempotent (skip if exists)
- [~] Make kubectl install idempotent (skip if exists)
- [ ] Make cluster creation idempotent (skip if exists)
- [ ] Make Keel startup idempotent (restart if not running, skip if healthy)

## Testing
- [ ] Test script on fresh environment
- [ ] Test script when all components already exist
- [ ] Verify deployments can be created in cluster
- [ ] Verify Keel UI is accessible at localhost:9300

## Notes
- **Discovery**: k3s direct install fails with `failed to find memory cgroup (v2)` in container environments
- **Solution**: Using k3d (k3s in Docker) instead, which bypasses host cgroup requirements