# Design: Align Image-Volume Opt-In Updates and Compatibility Documentation

## Architecture

PR #857 added image-volume discovery and rewriting to the existing Kubernetes provider flow. Discovery and `updateDeployments` already call `getImageVolumeTrackingFromMeta(labels, annotations)`, but `checkForUpdate` currently performs an exact lookup in annotations only. Replace that lookup with the existing helper, passing `resource.GetLabels()` before `resource.GetAnnotations()`.

Keep the image-volume rewrite loop, policy evaluation, `monitorContainers` volume filter, supported generic-resource kinds, and container/init-container paths unchanged. The helper searches labels before annotations, compares keys case-insensitively, and requires the exact value `"true"`; reusing it aligns eligibility without defining a new policy.

## Regression Coverage

Extend the Kubernetes provider update-path tests with distinct resources covering the canonical key in labels only, a mixed-case key (preferably in annotations to exercise both metadata sources), the existing canonical annotation, and no true opt-in. Assert the image-volume reference and update decision, not only helper or discovery output. Existing tests continue to protect `monitorContainers` filtering and other update paths.

## Compatibility Documentation

Update `readme.md` with the Kubernetes version matrix and runtime prerequisite, linking the official [feature-gate table](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) and [image-volume guide](https://kubernetes.io/docs/tasks/configure-pod-container/image-volumes/). Correct the unconditional feature-gate statement in `types/types.go`. The PR #857 `ARCHITECTURE.md` entry only states the valid Kubernetes 1.31+ minimum and does not require broader architecture changes.

## Verification

Run `gofmt` on changed Go files, focused `provider/kubernetes` tests including the new cases, `go test ./internal/k8s/... ./provider/kubernetes/...`, and `go test ./...` when practical. Report pre-existing or environment-related failures separately and confirm the final diff contains only this follow-up.

## Implementation Notes

- `provider/kubernetes/updates.go` now delegates image-volume eligibility to the existing metadata helper with labels and annotations in its declared order; the rewrite loop is unchanged.
- `provider/kubernetes/updates_test.go` directly exercises rewriting for label-only, mixed-case annotation, canonical annotation, no opt-in, and label-precedence cases. This avoids relying on helper/discovery-only assertions.
- Existing pure update and internal Kubernetes resource tests pass, covering standard/init-container updates, filtering, and supported generic-resource paths. Integration tests that construct the SQLite approval store cannot start here because the environment sets `CGO_ENABLED=0` while `go-sqlite3` requires cgo; this is environmental and occurs before provider behavior runs.
- `readme.md` now gives the complete Kubernetes gate/version matrix and states both the 1.31 minimum and container-runtime requirement, with links to both authoritative Kubernetes pages.
- The constant comment in `types/types.go` now states the minimum version, runtime support, and version-dependent gate defaults. `ARCHITECTURE.md` remains unchanged because its concise Kubernetes 1.31+ minimum is accurate and makes no unconditional gate claim.
- `gofmt` and the focused pure provider tests (`checkForUpdate`, SemVer, and the new image-volume opt-in table) pass.
- `go test ./internal/k8s/... ./provider/kubernetes/...` passes `internal/k8s` but the provider package times out before its logic runs in the pre-existing SQLite approval fixture (`CGO_ENABLED=0`; no C compiler is installed). A repository-wide compile sweep, `go test ./... -run '^$'`, passes all packages; executing the full suite is not practical under the same SQLite constraint.
- Final scope review found changes only in `updates.go`, `updates_test.go`, `readme.md`, and `types/types.go`; no unrelated architecture or update behavior was modified.
