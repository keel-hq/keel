# Implementation Tasks: Align Image-Volume Opt-In Updates and Compatibility Documentation

- [x] Replace the annotation-only image-volume gate in `checkForUpdate` with `getImageVolumeTrackingFromMeta(resource.GetLabels(), resource.GetAnnotations())` without changing unrelated update logic.
- [x] Add update-path regression tests for canonical label-only, case-variant, canonical annotation, and no-true-opt-in metadata cases.
- [x] Confirm existing `monitorContainers`, workload-kind, standard container, and init-container coverage remains passing.
- [x] Update `readme.md` with the Kubernetes 1.31-1.34, 1.35, and 1.36+ feature-state matrix, runtime support requirement, and both official Kubernetes links.
- [~] Correct stale image-volume compatibility comments introduced by PR #857, keeping `ARCHITECTURE.md` changes limited to wording that is genuinely inaccurate.
- [ ] Run `gofmt` on changed Go files and the focused `provider/kubernetes` tests.
- [ ] Run `go test ./internal/k8s/... ./provider/kubernetes/...` and, if practical, `go test ./...`; separate pre-existing or environmental failures from regressions.
- [ ] Review the final diff to ensure the change is narrowly scoped to image-volume opt-in consistency and compatibility documentation.
