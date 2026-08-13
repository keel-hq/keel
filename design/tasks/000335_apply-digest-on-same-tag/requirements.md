# Requirements: Apply Digest on Same-Tag Updates

## User Stories

- As a cluster operator tracking a mutable tag (e.g. `latest`), I want Keel to roll out my workload when the tag's digest changes, so my workload actually runs the new image.
- As a cluster operator, I want the workload image reference to be pinned to the new digest (`repo@sha256:...`) so the rollout is not dependent on `imagePullPolicy: Always` or a forced annotation.
- As a maintainer, I want tag-change behavior to remain unchanged when no digest is present in the event.

## Acceptance Criteria

- A poll event with a non-empty `Repository.Digest` and an unchanged tag creates an update plan when the workload's running digest differs from the event digest.
- The updated image reference is pinned to `repo@sha256:<digest>` for containers, init containers, and image volumes.
- A same-tag event whose digest equals the workload's running digest does not create a needless update plan.
- A tag-change event without a digest preserves the existing `repo:tag` behavior.
- The `keel.sh/update-time` annotation is still set so the pod template changes and the rollout is forced.
- Webhook and registry-notification events without a digest retain their existing semantics.

## Verification Requirements

- Run `gofmt` on changed files.
- Run focused `go test ./provider/kubernetes/...` including same-tag digest update, running-digest equality no-op, tag-change without digest, init-container, image-volume, and `getDesiredImage` digest tests.
- Run `go test ./...`.
- No startup or Helm chart changes, so `make release-validate` is not required.

## Open Questions

- Should the digest pinning also apply to Helm-tracked images? The initial scope is the Kubernetes provider; Helm can be a follow-up.
