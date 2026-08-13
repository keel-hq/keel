# Design: Retry Kubernetes Update Conflicts

## Confirmed Failure Path

`provider/kubernetes/implementer.go` contains a commented-out `retry.RetryOnConflict` block. The active `Update()` method performs a single, non-retrying API call for each supported resource kind (Deployment, StatefulSet, DaemonSet, CronJob). When the object's `resourceVersion` is stale — which happens routinely when Keel's informer/cache lags behind the API server or another controller mutates the object between read and write — the API server rejects the update with `409 Conflict` ("object has been modified"). Keel surfaces this as a failed update and the workload is not rolled out until the next poll/webhook cycle happens to succeed.

The original code (visible in the commented block) used `k8s.io/client-go/util/retry.RetryOnConflict` with `retry.DefaultRetry`, but it referenced the removed `Extensions()` API and was disabled wholesale rather than migrated to the current `AppsV1()` API.

## Fix

Re-enable conflict retry in `KubernetesImplementer.Update()`:

- Import `k8s.io/client-go/util/retry`.
- Wrap each resource-kind update in `retry.RetryOnConflict(retry.DefaultRetry, func() error { ... })`.
- Inside the retry function, re-fetch the latest object by namespace/name before updating, so the retried write carries a fresh `resourceVersion`. This mirrors the standard Kubernetes controller pattern and avoids retrying with the same stale version.
- Preserve the existing `switch` over resource kinds and the `unsupported object type` error path.
- Keep the `context.TODO()` usage consistent with the rest of the file.

The retry helper only retries on `apierrors.IsConflict` (HTTP 409); other API errors are returned immediately, so the change does not mask real failures.

## Verification

- Add a focused unit test using a fake Kubernetes clientset that returns `409 Conflict` on the first update attempt and succeeds on the second, asserting `Update()` returns nil and the object was updated.
- Add a test asserting non-conflict errors (e.g. `403 Forbidden`) are returned without retry.
- Run `gofmt`, `go test ./provider/kubernetes/...`, and `go test ./...`.
- This change does not affect application startup or the Helm chart, so `make release-validate` is not required.

## Implementation Notes

- The retry function must re-fetch the object inside the closure; retrying with the original stale object would loop until `DefaultRetry` is exhausted.
- `retry.DefaultRetry` uses exponential backoff (5 attempts) which is appropriate for transient conflicts.
- No public API, event, or policy behavior changes.
