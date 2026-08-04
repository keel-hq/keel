# Requirements: Prevent Unsafe Cross-Platform Image Updates

## User Stories

- As a cluster operator, I want Keel to consider the platforms on which a workload can actually be scheduled before selecting an image tag.
- As a Helm user, I want Helm-tracked images to receive the same platform-safety checks as directly tracked Kubernetes workloads.
- As an operator of a mixed-architecture cluster, I want Keel to update only when the candidate supports every eligible workload platform, rather than assuming the platform on which Keel itself runs.
- As an operator troubleshooting a skipped update, I want a diagnostic that identifies missing or incompatible workload/manifest platform evidence.

## Acceptance Criteria

- The issue #834 tag shape cannot update an amd64-eligible workload from `jellyfin/jellyfin:latest` to the ARM-only `20240303.2-unstable-armhf` candidate.
- Workload platforms come from Kubernetes workload scheduling metadata and node/pod platform data. `runtime.GOOS`, `runtime.GOARCH`, tag spelling, and Keel's own pod platform are not used as workload-platform fallbacks.
- Keel computes a conservative set of platforms on which each tracked workload may run. A candidate is eligible only when its registry manifest supports every platform in that set.
- A mixed eligible platform set remains updateable by a manifest list/index that supports the complete set. A single-platform candidate that covers only part of the set is rejected.
- If node metadata cannot be read, no eligible nodes can be established, a Helm image cannot be mapped to its Kubernetes workloads, or another ambiguity prevents a conservative platform set from being established, polling fails closed for that workload with an observable diagnostic.
- Explicit Kubernetes scheduling constraints, including stable and legacy OS/architecture node-selector labels and required node affinity, narrow the eligible node set when they can be evaluated safely. Constraints not modeled by Keel may be ignored only when doing so produces a conservative superset of eligible nodes.
- Helm-tracked images are mapped to the Kubernetes resources owned by the Helm release and are checked against the union of those resources' eligible platforms. Missing or ambiguous ownership mapping fails closed.
- Docker schema-2 and OCI single-platform manifests continue to resolve their platform from the image config. Docker manifest lists and OCI image indexes remain supported.
- A manifest/config resolution failure skips that candidate, logs the reason, and allows evaluation of the next policy-compatible candidate.
- Existing tag ordering and policy comparison semantics remain unchanged. Webhook, approval, and non-platform-specific behavior are not redesigned.
- Deterministic automated coverage includes registry manifest resolution, Kubernetes scheduling resolution, Helm workload mapping, fail-closed branches, related mixed-platform workloads, and the complete polling-to-Kubernetes selection path.
- No Jellyfin-specific tag heuristics, release, deployment, merge, UI, or unrelated dependency work is included.

## Verification Requirements

- Run focused registry, polling, Kubernetes provider, Helm provider, and platform-resolver tests.
- Run the deterministic issue #834 integration/e2e regression against the local registry fixture and Kubernetes scheduling metadata.
- Run `gofmt`, `go test ./...`, the repository build, and applicable repository lint/e2e checks.
- Report the reproduced pre-fix selection, post-fix selected candidate, mixed-platform/multi-arch evidence, diagnostics for unresolved compatibility, and any remaining registry or scheduler-model limitations.

## Open Questions

None. When exact scheduler eligibility cannot be modeled, Keel uses a conservative node-platform superset; if even that cannot be established, it fails closed.
