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
