# Implementation Tasks: Retry Kubernetes Update Conflicts

- [ ] Re-enable `retry.RetryOnConflict` in `provider/kubernetes/implementer.go` for Deployment, StatefulSet, DaemonSet, and CronJob updates.
- [ ] Re-fetch the latest object by namespace/name inside the retry closure before each update attempt.
- [ ] Preserve the `unsupported object type` error path and existing context usage.
- [ ] Add a fake-client unit test that returns `409 Conflict` on the first update and succeeds on the second, asserting `Update()` returns nil and the object is updated.
- [ ] Add a unit test asserting a non-conflict error (e.g. `403 Forbidden`) is returned without retry.
- [ ] Run `gofmt`, `go test ./provider/kubernetes/...`, and `go test ./...`.
- [ ] Review the final diff for behavior changes beyond conflict retry.
- [ ] Commit and push the implementation branch; open a PR.
