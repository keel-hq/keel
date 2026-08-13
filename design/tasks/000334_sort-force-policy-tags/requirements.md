# Requirements: Sort Force Policy Candidate Tags

## User Stories

- As a cluster operator using the `force` policy, I want Keel to select the newest available tag rather than an arbitrary registry-order tag, so a `force` update never downgrades my workload to an older tag.
- As a maintainer, I want the `force` policy's candidate ordering to be deterministic and semver-aware, matching the behavior of the semver policies.

## Acceptance Criteria

- `ForcePolicy.Filter()` returns version tags sorted newest-first (descending semver order).
- Non-semver tags (e.g. `latest`, `master`, `latest-staging`) are preserved and do not break sorting; they are placed after sorted version tags in a deterministic order.
- The input tag slice is not mutated.
- The `// todo: why is this not sorting?` comment is removed.
- `ForcePolicy.ShouldUpdate` semantics are unchanged: any tag is accepted (or only the matching tag when `matchTag` is set).
- A `force`-tracked workload whose registry returns `["3.0.0", "7.9.1", "7.8.0"]` selects `7.9.1`, not `3.0.0`.

## Verification Requirements

- Run `gofmt` on changed files.
- Run focused `go test ./internal/policy/...` including newest-first ordering, non-semver preservation, mixed ordering, and input immutability tests.
- Run `go test ./...`.
- No startup or Helm chart changes, so `make release-validate` is not required.

## Open Questions

None.
