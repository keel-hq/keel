# Implementation Tasks: Apply Digest on Same-Tag Updates

- [ ] In `provider/kubernetes/updates.go` `checkForUpdate`, treat a non-empty `repo.Digest` as an update trigger when the event repository matches and the running digest differs.
- [ ] Rewrite matching container/init-container/image-volume image references to `repo@sha256:<digest>` when the event carries a digest.
- [ ] Reuse the existing `RunningDigestResolver` (`p.runningDigests`) to compare the workload's running digest against the event digest.
- [ ] Preserve existing `repo:tag` behavior when `repo.Digest == ""`.
- [ ] Extend `getDesiredImage` (and the `UpdateContainer`/`UpdateInitContainer`/`UpdateImageVolume` call sites) to emit digest-pinned references.
- [ ] Keep setting the `keel.sh/update-time` annotation so the pod template changes.
- [ ] Add focused unit tests in `provider/kubernetes/updates_test.go` for same-tag digest update, running-digest equality no-op, tag-change without digest, init containers, image volumes, and `getDesiredImage` with a digest.
- [ ] Run `gofmt`, `go test ./provider/kubernetes/...`, and `go test ./...`.
- [ ] Review the final diff for behavior changes beyond digest pinning.
- [ ] Commit and push the implementation branch; open a PR.
