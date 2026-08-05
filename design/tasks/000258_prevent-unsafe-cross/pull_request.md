# Prevent unsafe cross-platform image updates

## Summary
Prevent polling updates from selecting registry tags that do not support every Kubernetes platform eligible for the tracked workload. Workload compatibility now comes from Kubernetes Node, Pod, and scheduling metadata, including Helm-owned workload mappings, instead of Keel's own runtime platform.

## Changes
- Resolve conservative workload platform sets from cached Node metadata, pod templates, required affinity, and observed Pods; fail closed with typed diagnostics when unresolved.
- Resolve Docker/OCI manifest and config platform metadata, preserve multi-architecture indexes, and carry verified evidence through Kubernetes and Helm update-plan selection.
- Add minimal read-only Node RBAC to the chart and native e2e harness.
- Add focused registry, resolver, polling, Kubernetes, and Helm tests plus a deterministic registry-to-poll-to-Kubernetes integration regression.

## Evidence
- Issue #834 fixture before the compatibility gate: policy ordering selected `20240303.2-unstable-armhf` from `latest`.
- After: linux/amd64 eligibility rejects the ARM/v7 candidate with a warning and applies `10.10.7`.
- Mixed linux/amd64 and linux/arm64 eligibility rejects amd64-only `3.0.0` and applies multi-arch `2.0.0`.

## Testing
- `gofmt` and `git diff --check`
- Focused resolver, registry, poll, Kubernetes, Helm, and integration tests
- `go test ./...` in a cgo-enabled Alpine container
- Compile-only `go test ./... -run '^$'` on the host
- `go build ./cmd/keel`
- OpenAPI contract test
- Docker static/cgo build and UI checks
- Native k3s e2e suite (polling positive/negative and webhook): pass

Host-wide SQLite-backed execution cannot run directly because the sandbox forces `CGO_ENABLED=0`; the full suite passed in the cgo-enabled container. `go vet ./...` continues to report pre-existing unkeyed Kubernetes literals and unexported-field JSON tags. Registries that omit/misreport manifest content types or config platform fields, and schema-v1 manifests, remain fail-closed with warnings.
