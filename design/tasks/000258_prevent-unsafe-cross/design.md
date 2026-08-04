# Design: Prevent Unsafe Cross-Platform Image Updates

## Confirmed Failure Path

The multi-tag polling path discovers repository tags, applies the configured policy filter, and evaluates candidates in policy order. For a current tag of `latest`, the existing SemVer `major` policy deliberately accepts any parseable version. The reported date-like prerelease sorts ahead of ordinary compatible releases, so `20240303.2-unstable-armhf` becomes the first accepted candidate. The watcher currently submits that tag without resolving its manifest, and the Kubernetes provider independently repeats the policy decision and rewrites the workload image.

The policy behavior is valid and remains unchanged. The defect is the missing compatibility gate between policy acceptance and event submission/application.

## Workload Platform Resolution

Introduce a Kubernetes-backed platform resolver that returns a set of eligible `os/architecture/variant` platforms plus evidence or a typed unresolved reason for a tracked workload. Do not default to `runtime.GOOS`, `runtime.GOARCH`, the Keel pod's node, or an architecture-looking tag suffix.

The resolver reads the workload pod template and cluster Node objects. It begins with schedulable nodes and conservatively filters them using constraints that can safely narrow eligibility:

- `spec.nodeName`, when present;
- stable `kubernetes.io/os` and `kubernetes.io/arch` selectors;
- legacy `beta.kubernetes.io/os` and `beta.kubernetes.io/arch` selectors;
- all ordinary node-selector keys by matching them against node labels;
- required node affinity terms and their standard `In`, `NotIn`, `Exists`, `DoesNotExist`, `Gt`, and `Lt` operators;
- the pod OS field when present.

Required node-selector terms are ORed and expressions inside a term are ANDed, matching Kubernetes semantics. Preferred affinity never narrows eligibility. Taints, pod affinity/anti-affinity, topology spread, resource pressure, and other scheduler inputs may be ignored only as narrowing constraints: retaining extra nodes produces a conservative platform superset and cannot permit an incompatible image. If a constraint cannot be interpreted without potentially excluding an eligible node, retain the node. Invalid scheduling metadata, an empty eligible set, unavailable Node data, or missing platform labels/status produces an unresolved result rather than a runtime-platform assumption.

Actual Pods owned by the workload provide corroborating scheduling evidence and cover pinned/created workloads, but a running pod does not by itself prove the only future platform unless the template constraints establish that fact. A platform observed on a running pod is always included. If pod evidence conflicts with the computed node set, use the union and emit a diagnostic; this is conservative during rollouts and scheduling changes.

Represent the result on `TrackedImage` as an internal platform-set/evidence value that is not added to the public tracked-image API. The set is deduplicated and deterministic for tests and logs. Node access is supplied through the Kubernetes client/cache abstraction, and the chart's ClusterRole gains only the `get`, `list`, and `watch` access required for Nodes if the resolver uses an informer. Resolution should be cached/informer-backed rather than listing every node for every candidate.

## Helm-Tracked Workloads

Helm image values do not contain scheduling information. Inject the same workload platform resolver/resource index into the Helm provider. Map each Helm release to cached Kubernetes resources using Helm's release name/namespace ownership metadata, then retain resources whose pod templates reference the tracked image repository. Resolve each matched resource and take the union of eligible platforms.

Multiple resources or mixed architectures are valid: a candidate must support their entire union. If no owned workload can be mapped, ownership is contradictory, a matching resource has unresolved platform evidence, or a release image is used by an unsupported workload kind, mark the Helm tracked image unresolved and skip its polling update with the release/image/reason in the diagnostic. This makes Helm safe without pretending the Helm values describe a workload platform.

The Kubernetes and Helm providers must share one resolver contract so platform rules do not diverge. Provider construction in `cmd/keel` wires the existing generic-resource cache and Kubernetes node/pod data into both providers. Helm release policy/value semantics remain unchanged.

## Registry Compatibility Gate

Keep the existing registry platform resolver behavior:

- advertise Docker manifest-list, Docker schema-2 manifest, OCI image-index, and OCI image-manifest media types;
- obtain all descriptor platforms from a Docker manifest list or OCI index;
- fetch and decode the config blob for a single-platform Docker/OCI manifest;
- treat missing, malformed, schema-v1, or unsupported platform metadata as unresolved.

After policy approval, resolve the candidate tag once and cache the result for the polling run. The candidate is compatible only if at least one manifest platform matches each eligible workload platform. For related tracked images sharing a watcher, require support for every platform of every workload that the candidate's policy would update. A mixed-platform workload therefore accepts a complete multi-arch index and rejects a partial single-architecture manifest.

If candidate manifest resolution fails, log the repository, tag, and reason, then continue to the next policy-compatible tag. If workload platform resolution is unresolved, do not evaluate candidates for that workload; log the provider, workload/release identity, image, and reason. The submitted polling event should carry or reference the resolved compatibility evidence so the Kubernetes/Helm application path can reject a stale or mismatched event instead of relying solely on discovery-time checking. External webhook events without platform evidence preserve their existing semantics unless a later separately scoped design adds registry resolution to those triggers.

## Deterministic Regression Architecture

Extend the repository's isolated registry/Kubernetes test infrastructure rather than using Docker Hub or mutable Jellyfin tags. Seed synthetic Docker/OCI metadata backed by known runnable layers:

| Tag/candidate | Registry platform metadata | Purpose |
| --- | --- | --- |
| `latest` | linux/amd64 | Current issue-shape workload image |
| `10.10.7` | linux/amd64 | Compatible fallback under the same `major` policy |
| `20240303.2-unstable-armhf` | linux/arm/v7 only | Higher policy-ordered incompatible candidate |
| a deterministic higher multi-arch tag | linux/amd64 and linux/arm64 | Proves manifest-list selection for mixed eligibility |

The integration test uses the real registry client, polling watcher, provider aggregation, Kubernetes generic-resource cache, platform resolver, policy implementation, and Kubernetes update-plan/application selection. Node and Pod objects are deterministic fixtures (or run-scoped k3s resources in the existing e2e harness), not `runtime.GOARCH` assumptions.

Run two isolated repository scenarios so each assertion is unambiguous:

1. An amd64-eligible Deployment starts at `latest`; discovery returns the exact issue #834 ARM-only tag plus `10.10.7`. Before the fix, capture selection/application of the ARM tag. After the fix, require a warning for the ARM candidate and require the Deployment template to select `10.10.7`.
2. Related or Helm-owned workloads are eligible on amd64 and arm64; a higher single-platform candidate is rejected and the complete multi-arch candidate is selected. Require the resulting Kubernetes/Helm update selection to reference the multi-arch tag.

The same fixture exercises an unresolved manifest candidate followed by a compatible candidate, proving evaluation continues, and an unresolved workload-platform case, proving no update is applied. Focused tests separately cover node selectors, required affinity term semantics, running-pod union behavior, missing node metadata, Helm ownership mapping, single manifests/config blobs, Docker manifest lists, OCI indexes, and related-workload unions.

If implemented in the native-k3s suite, use its run-scoped registry and namespaces, assert exact tag sets before polling, wait on observed Deployment template changes rather than sleeps, capture Keel warnings in diagnostics, and retain the existing ownership-safe teardown. Synthetic manifests must not depend on the host architecture for their declared platform metadata.

## Observability and Failure Modes

Use structured warnings with stable reasons such as `node_metadata_unavailable`, `no_eligible_nodes`, `workload_platform_missing`, `helm_workload_unmapped`, `manifest_platform_unresolved`, and `candidate_platform_incompatible`. Include eligible and candidate platform sets when known, without credentials or manifest bodies. Avoid repeating the same warning for every related image/candidate within one polling run.

Known registry limitations remain explicit: schema-v1 manifests and registries that omit or misreport required content types/config platform fields cannot establish compatibility and are skipped. The conservative scheduler model may include more node platforms than Kubernetes would ultimately choose; this can skip an otherwise runnable single-platform update, but it cannot silently approve a known-incompatible one. Users can make eligibility precise with workload scheduling constraints or publish complete multi-arch images.

## Verification

Run `gofmt` on all changed Go files; focused registry, poll, platform-resolver, Kubernetes, and Helm tests; `go test ./...`; the repository binary build and OpenAPI contract check; and the native k3s e2e path when used by the regression. Run repository lint/UI checks only as verification, without unrelated UI changes. Record baseline failures separately and report pre-fix versus post-fix selected tags and platform evidence.

## Implementation Notes

- The first revision removed the `runtime.GOOS`/`runtime.GOARCH` fallback from both Kubernetes tracking and polling compatibility. `TrackedImage` now carries a platform set plus a typed unresolved reason; an empty or unresolved set is skipped with a warning.
- This sandbox currently has `CGO_ENABLED=0` and no C compiler. SQLite-backed polling tests wait in the existing SQL connector rather than reaching assertions. Compile-only polling checks and pure Kubernetes tests work on the host; executed polling/full-suite verification must use the repository's cgo-capable container path or another environment with a compiler.
