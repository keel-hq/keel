# Establish reliable k3s end-to-end tests

## Summary
Add a deterministic native-k3s smoke workflow that builds and runs the checked-out Keel image in-cluster and verifies its highest-value update paths.

## Changes
- Add a safely guarded, checksum-verified native k3s lifecycle with isolated registry fixtures and always-captured diagnostics.
- Replace duplicated acceptance tests with a testify/suite harness covering webhook update, polling update, and policy rejection.
- Add the opt-in e2e CI job and block image publication when an enabled e2e run fails.
- Fix non-root access to Kubernetes ServiceAccount projected volumes in the built image.

## Testing
- `make test` with Go 1.26.5 and cgo build tools: passed.
- Focused e2e helper tests, ShellCheck 0.10.0, and actionlint 1.7.7: passed.
- Repeated clean native-k3s runs: passed in 5m03s cold, 3m59s cached, 5m01s after lifecycle isolation, and 5m00s on the final merged Go 1.26.5 tree.
- Success and failure cleanup audits: all run-owned cluster, runtime, kubelet, and temporary paths removed; diagnostics captured before cleanup.
