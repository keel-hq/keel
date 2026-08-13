# Design: Sort Force Policy Candidate Tags

## Confirmed Failure Path

`internal/policy/force.go` `ForcePolicy.Filter()` contains a literal `// todo: why is this not sorting?` and returns the input tag slice unchanged (in registry order). The multi-tag polling watcher (`trigger/poll/multi_tags_watcher.go`) calls `trackedImage.Policy.Filter(tags)` and then iterates the filtered tags in order, selecting the first tag for which `ShouldUpdate` returns true.

For the `force` policy, `ShouldUpdate` returns true for any tag (unless `matchTag` is set, in which case it requires the tag to equal the current tag). Because `Filter` does not sort, the first tag in registry order is selected. Registries commonly return tags in lexicographic or push order, so Keel can select an older tag over a newer one. The reported case is a `force`-tracked Confluent image being "updated" from `7.9.1` to `3.0.0` because `3.0.0` appeared earlier in the registry's tag list.

## Fix

Make `ForcePolicy.Filter()` sort candidate tags semver-aware in descending order (newest first), matching the behavior of `SemverPolicy.Filter()`:

- Parse each tag with `github.com/Masterminds/semver`.
- Keep tags that parse as versions; sort them descending by version.
- Preserve non-semver tags (e.g. `latest`, `master`, `latest-staging`) so `force` continues to work for non-versioned tags. Place them after the sorted version tags, or preserve their relative order in a deterministic way.
- Remove the `// todo` comment.

The `force` policy semantics are unchanged: `ShouldUpdate` still accepts any tag (or only the matching tag when `matchTag` is set). Sorting only changes which candidate is considered first, so the newest compatible tag wins instead of an arbitrary registry-order tag.

## Verification

- Add focused unit tests in `internal/policy/force_test.go`:
  - Version tags are returned newest-first (e.g. `["7.9.1", "7.8.0", "3.0.0"]` → `["7.9.1", "7.8.0", "3.0.0"]`).
  - Non-semver tags are preserved and do not break sorting.
  - Mixed version/non-version tags produce a deterministic order.
  - The input slice is not mutated.
- Run `gofmt`, `go test ./internal/policy/...`, and `go test ./...`.
- This change does not affect application startup or the Helm chart, so `make release-validate` is not required.

## Implementation Notes

- Reuse the same `semver` dependency already used by `SemverPolicy.Filter()`.
- Decide and document the placement of non-semver tags. Recommended: keep non-semver tags after sorted version tags, preserving their original relative order, so `latest`/`master` style tags remain available to `force` users.
- No public API, event, or policy comparison changes.
