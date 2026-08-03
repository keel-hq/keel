# Upgrade Keel to Go 1.26.5

## Summary

Upgrade Keel's active Go toolchain declarations to the latest stable Go 1.26 patch, verified as 1.26.5 in the official Go releases feed.

## Changes

- Pin Go 1.26.5 in `go.mod`, GitHub Actions, and all active Go-based Docker stages.
- Update contributor prerequisites and replace obsolete GOPATH/dep guidance with the existing module workflow.
- Leave dependencies, application source, generated artifacts, and historical webhook payload fixtures unchanged.

## Testing

- Passed Go 1.26.5 package compilation, binary build, focused API contract testing, pinned API regeneration/lint, and clean generated-artifact checks.
- Passed the default `linux/amd64` and `linux/arm64` image build plus AMD64 debug and test image builds.
- Full tests require CGO and an unavailable C compiler; e2e requires an unavailable Kubernetes cluster. Existing formatting/vet findings remain outside this change.
- The Debian image reaches its Go 1.26.5 builder but its unchanged runtime stage fails because `debian:latest` lacks `addgroup`.
