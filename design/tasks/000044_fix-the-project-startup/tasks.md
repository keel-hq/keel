# Implementation Tasks

## Prerequisites Setup
- [ ] Add function to check and install k3d if not present
- [ ] Add function to check and install kubectl if not present
- [ ] Add function to verify Go is installed (required for building Keel)

## Cluster Management
- [ ] Add function to check if k3d cluster "keel-dev" exists
- [ ] Add function to create k3d cluster if it doesn't exist
- [ ] Add function to wait for cluster to be ready
- [ ] Configure kubeconfig context to `k3d-keel-dev`

## Keel Build and Run
- [ ] Checkout master branch to access Keel source code
- [ ] Build Keel binary using `go build` in `cmd/keel` directory
- [ ] Create function to check if Keel is already running (PID file check)
- [ ] Start Keel with `--no-incluster` flag as background process
- [ ] Set environment variables: `KUBERNETES_CONTEXT`, `BASIC_AUTH_USER`, `BASIC_AUTH_PASSWORD`
- [ ] Redirect Keel output to `/tmp/keel.log`
- [ ] Store PID in `/tmp/keel.pid`

## Verification
- [ ] Wait for Kubernetes API to be responsive
- [ ] Verify kubectl can list namespaces
- [ ] Wait for Keel to start (check port 9300)
- [ ] Print success message with UI URL

## Idempotency
- [ ] Make k3d install idempotent (skip if exists)
- [ ] Make kubectl install idempotent (skip if exists)
- [ ] Make cluster creation idempotent (skip if exists)
- [ ] Make Keel startup idempotent (restart if not running, skip if healthy)

## Testing
- [ ] Test script on fresh environment
- [ ] Test script when all components already exist
- [ ] Verify deployments can be created in cluster
- [ ] Verify Keel UI is accessible at localhost:9300