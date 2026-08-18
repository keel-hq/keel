# feat(notifications): report image digests in update notifications

## Summary
`latest -> latest` upgrades (poll trigger with `force`/`matchTag` policy) already work — `WatchTagJob` detects the digest change on the same tag and submits an event carrying the new digest — but the notifications were useless in that case: they rendered `Successfully updated deployment xxxx/deployment-1 latest->latest (gcr.io/image:latest)` with no way to see what actually changed. The new digest from the event was dropped before reaching the notification layer, and the previous digest was never tracked.

This change makes update notifications report the image hashes:

- All three notification messages (preparing, success, failure) for both the `kubernetes` and `helm3` providers now show the digest transition, e.g. `Successfully updated deployment xxxx/deployment-1 latest (sha256:old…)->latest (sha256:new…) (gcr.io/…)`.
- The **new digest** comes from the event's repository: the poll trigger always resolves it, and the GCR pubsub, ACR, Harbor, and GitHub Container Registry webhooks also provide it.
- The **previous digest** is resolved from the workloads' running pods (matched per image reference), falling back to the `keel.sh/digest` annotation, which was previously defined but unused and now records the digest keel last deployed.
- Notifications gain structured metadata (`previousDigest`, `newDigest`) for webhook/audit consumers, and the `kubernetes.io/change-cause` annotation also includes the digest transition.
- Backward compatible: when no digest is known (e.g. no running pods, tag-based updates with no event digest), the traditional message shape is preserved.
- `ARCHITECTURE.md` documents the new `keel.sh/digest` annotation behavior.

## Testing
- `gofmt` clean on all changed files; `go build ./...` passes.
- Full unit test suite: all 31 packages pass, including 7 new digest tests covering running-pod resolution, annotation fallback, resolution preference order, the no-digest legacy message shape, and the helm3 release variants.
- Full k3s e2e suite (`make e2e`): Keel deployed from the source-built image with this change; all 5 scenarios pass (polling eligible/ineligible, webhook eligible, init-container isolation, OAuth proxy). The change touches no startup or chart code.
