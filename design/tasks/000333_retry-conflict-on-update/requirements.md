# Requirements: Retry Kubernetes Update Conflicts

## User Stories

- As a cluster operator, I want Keel to survive transient `409 Conflict` responses when updating workloads, so a stale `resourceVersion` does not cause a missed rollout.
- As a maintainer, I want the update path to use the standard Kubernetes retry-on-conflict pattern rather than a single non-retrying API call.

## Acceptance Criteria

- `KubernetesImplementer.Update()` retries with exponential backoff when the API server returns `409 Conflict` for Deployment, StatefulSet, DaemonSet, and CronJob updates.
- Each retry re-fetches the latest object before writing, so the retried update carries a fresh `resourceVersion`.
- Non-conflict errors (for example `403 Forbidden`, `404 Not Found`, network errors) are returned immediately without retry.
- The `unsupported object type` error path is preserved.
- Existing update semantics, event handling, and policy behavior are unchanged.

## Verification Requirements

- Run `gofmt` on changed files.
- Run focused `go test ./provider/kubernetes/...` including a fake-client test that returns `409` once then succeeds, and a test that a non-conflict error is not retried.
- Run `go test ./...`.
- No startup or Helm chart changes, so `make release-validate` is not required.

## Open Questions

None.
