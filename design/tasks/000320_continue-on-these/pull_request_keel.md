# fix(poll): seed the tag digest watcher from the running image digest

## Summary
Keel's same-tag digest watcher (`force` policy with `keel.sh/match-tag=true`) fetched the tag's registry digest at registration and stored it as its baseline, then immediately compared the registry against itself. A workload already running an older digest of a mutable tag therefore produced no event. Because the baseline lives only in the in-memory `watched` map and is never persisted, this happened on every Keel start or restart, and drift went unnoticed until the tag moved again. Fixes keel-hq/keel#845.

The baseline is now taken from what the workload is actually running:

- `internal/k8s/running_digests.go` adds `RunningDigestResolver`. It lists a workload's pods by its selector and maps each pod-spec container image to the digests reported by the corresponding container statuses. Statuses are matched to containers by name, so init containers are covered and multi-container pods do not cross-contaminate. Terminating pods and waiting containers are skipped, and the runtime forms of `imageID` (`docker-pullable://repo@sha256:…`, `repo@sha256:…`, bare `sha256:…`) are normalized to a bare digest.
- `types.TrackedImage.RunningDigests` carries that per-image set. The Kubernetes provider populates it in `TrackedImages`; the Helm 3 provider does the same through `releaseImageRunningDigests`, reusing the release-to-workload annotation mapping already used for platform resolution, wired up in `cmd/keel/main.go` via the new `helm3.WithRunningDigests` option.
- `registry.Client.Digests` / `docker.ManifestDigests` return the digest a tag resolves to plus, for an image index, the digests of its child manifests.
- `trigger/poll/watcher.go` seeds `watchDetails.digest` from the running digest when it no longer matches the tag, so the poll that runs immediately after registration emits an event and the workload is corrected.

The drift check is deliberately narrow. It runs only for `KeepTag()` policies, the sole path that compares digests, so semver and tag-discovery watchers make no extra registry call. Drift is reported only when a workload runs *none* of the digests the tag maps to: a workload mid-rollout with at least one updated replica is not stale, a pod running a per-platform child of a multi-arch image index is not stale (that digest never equals the index digest a registry reports), and if the manifest cannot be fetched the previous behaviour is kept rather than guessing. Running digests are grouped per workload rather than merged, so a watcher shared by several workloads still fires when one of them is stale even if the first workload seen is current. The check runs at watcher registration only; re-checking on every provider rescan would re-emit events for workloads blocked on approvals.

No chart change was needed: the existing ClusterRole already grants `list` on `pods`.

## Testing
`CGO_ENABLED=1 go test ./...` — all 32 packages pass (CGO is required for the `go-sqlite3` based store used by the polling tests).

New tests cover:
- `internal/k8s` — the resolver across replicas reporting different digests, per-container-image keying, init containers, terminating pods, waiting containers, statuses with no matching container, pods without a digest, pod-list errors and nil inputs, plus a table for every `imageID` form.
- `provider/kubernetes` — `TrackedImages` carries the running digest observed on the pod.
- `provider/helm3` — release-owned workloads contribute digests while other releases and other images do not, and the absence of a resolver yields nothing.
- `registry/docker` — `ManifestDigests` against a fixture registry for an image index, a single manifest, and a response with no `Docker-Content-Digest` header, plus the error path.
- `trigger/poll` — the six baseline-seeding cases (stale, current, mid-rollout, multi-arch child, unresolvable manifest, no runtime information), a shared watcher where only the second workload is stale, and the absence of manifest lookups without `match-tag`.
