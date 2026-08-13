# Design: Apply Digest on Same-Tag Updates

## Confirmed Failure Path

The single-tag poll watcher (`trigger/poll/single_tag_watcher.go`) detects a digest change for a mutable tag (e.g. `latest`) and submits an event with `Repository.Digest` set to the new digest. The Kubernetes provider's `checkForUpdate` (`provider/kubernetes/updates.go`) only compares tags via `plc.ShouldUpdate(currentTag, newTag)`. For a same-tag digest change, `currentTag == newTag`, so:

- For semver policies, `ShouldUpdate` returns false (new version is not higher than current), so no update plan is created.
- For the `force` policy with `matchTag` set, `ShouldUpdate` returns false when tags are equal.
- Even when an update is triggered (e.g. via the `keel.sh/update-time` annotation or `imagePullPolicy: Always`), the Deployment image reference is rewritten as `repo:tag` — the `Digest` field from the event is never used. The image reference is never pinned to `repo@sha256:...`, so the workload continues to pull whatever the mutable tag currently points to.

The reported issues (#846, #803) describe workloads that never roll out when a mutable tag's digest changes, or that roll out but keep running the old digest because the image reference is not pinned.

## Fix

When a poll event carries a non-empty `Repository.Digest`, the Kubernetes provider should pin the workload image reference to the digest so the rollout actually runs the new digest:

- In `checkForUpdate` (`provider/kubernetes/updates.go`), when `repo.Digest != ""` and the event repository matches the container's repository, treat the digest change as an update trigger even when the tag is unchanged.
- When rewriting the image reference, use `repo@sha256:<digest>` instead of `repo:tag` for the matching container/init-container/image-volume.
- Preserve the existing tag-based behavior when `repo.Digest == ""` (webhook events, registry notifications without digest, etc.).
- Keep the `keel.sh/update-time` annotation update so the pod template changes and the rollout is forced even if the image reference string is unchanged.

### Policy interaction

A same-tag digest change must bypass the tag-comparison gate in `ShouldUpdate` only when the digest actually differs from what the workload is running. The provider already has access to running digests via `p.runningDigests` (the `RunningDigestResolver`). The update plan should be created when:

- `repo.Digest != ""`, AND
- the workload's running digest for the matching image differs from `repo.Digest` (or the running digest is unknown and the tag is unchanged).

This avoids needless rollouts when the workload is already running the digest the event describes.

### Image reference rewriting

When the digest is applied, the container image reference becomes `registry/repo@sha256:<digest>`. The `image.Parse`/`Reference` helpers already support canonical (`@digest`) references, so `getDesiredImage` and the `UpdateContainer`/`UpdateInitContainer`/`UpdateImageVolume` paths must be extended to accept a digest and emit `repo@sha256:...` rather than `repo:tag`.

## Verification

- Add focused unit tests in `provider/kubernetes/updates_test.go`:
  - A same-tag event with a new digest creates an update plan and rewrites the image reference to `repo@sha256:<digest>`.
  - A same-tag event whose digest equals the running digest does not create an update plan.
  - A tag-change event without a digest preserves the existing `repo:tag` behavior.
  - Init containers and image volumes are handled consistently.
- Add a test for `getDesiredImage` with a digest.
- Run `gofmt`, `go test ./provider/kubernetes/...`, and `go test ./...`.
- This change does not affect application startup or the Helm chart, so `make release-validate` is not required.

## Implementation Notes

- The `RunningDigestResolver` already exists and is wired into the Kubernetes provider (`p.runningDigests`); reuse it to compare the running digest against the event digest.
- Be careful with the `force` policy: a same-tag digest change should still be applied when the digest differs, even though `ShouldUpdate` returns false for equal tags. The digest check must be an additional trigger condition, not a replacement for policy evaluation.
- No public API, event, or policy comparison changes.
