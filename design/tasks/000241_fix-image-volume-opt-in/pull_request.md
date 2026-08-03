# Align image-volume opt-in updates

## Summary

Make image-volume rewrites honor the same label and case-insensitive annotation opt-ins used during discovery, and correct Kubernetes compatibility guidance.

## Changes

- Reuse the existing metadata helper in `checkForUpdate` with label-first precedence.
- Add update-path regressions for label-only, case-variant, annotation, disabled, and precedence cases.
- Document the Kubernetes 1.31-1.34, 1.35, and 1.36+ feature states and runtime requirement.

## Testing

- `gofmt` on changed Go files.
- Focused `provider/kubernetes` update tests pass.
- `internal/k8s` tests pass.
- `go test ./... -run '^$'` passes all-package compilation.
- Full provider and repository test execution is blocked by the existing SQLite fixture because this environment has `CGO_ENABLED=0` and no C compiler; `go-sqlite3` cannot start.
