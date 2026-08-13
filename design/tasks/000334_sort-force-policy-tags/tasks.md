# Implementation Tasks: Sort Force Policy Candidate Tags

- [ ] Update `ForcePolicy.Filter()` in `internal/policy/force.go` to sort version tags newest-first using `github.com/Masterminds/semver`.
- [ ] Preserve non-semver tags after sorted version tags in a deterministic order.
- [ ] Ensure the input tag slice is not mutated.
- [ ] Remove the `// todo: why is this not sorting?` comment.
- [ ] Add focused unit tests in `internal/policy/force_test.go` for newest-first ordering, non-semver preservation, mixed ordering, and input immutability.
- [ ] Run `gofmt`, `go test ./internal/policy/...`, and `go test ./...`.
- [ ] Review the final diff for behavior changes beyond candidate ordering.
- [ ] Commit and push the implementation branch; open a PR.
