# Implementation Tasks: Retry Kubernetes Update Conflicts

- [x] Re-enable `retry.RetryOnConflict` in `provider/kubernetes/implementer.go` for Deployment, StatefulSet, DaemonSet, and CronJob updates.
- [x] Re-fetch the latest object by namespace/name inside the retry closure before each update attempt.
- [x] Preserve the `unsupported object type` error path and existing context usage.
- [x] Add a fake-client unit test that returns `409 Conflict` on the first update and succeeds on the second, asserting `Update()` returns nil and the object is updated.
- [x] Add a unit test asserting a non-conflict error (e.g. `403 Forbidden`) is returned without retry.
- [x] Run `gofmt`, `go test ./provider/kubernetes/...`, and `go test ./...`.
- [x] Review the final diff for behavior changes beyond conflict retry.
- [x] Commit and push the implementation branch; open a PR.
