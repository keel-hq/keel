# Implementation Tasks: Prevent Unsafe Cross-Platform Image Updates

- [x] Replace every `runtime.GOOS`/`runtime.GOARCH` workload fallback with a Kubernetes-backed platform resolver and typed unresolved results.
- [x] Add Node access/cache support and the minimum chart RBAC needed to read node OS/architecture metadata.
- [x] Extract pod-template scheduling metadata for Deployment, StatefulSet, DaemonSet, and CronJob generic resources.
- [x] Resolve a conservative eligible-node platform set from node name, node selectors, required node affinity, pod OS, and actual owned-pod/node evidence.
- [x] Treat mixed eligible platforms as a set that candidates must fully support; fail closed with structured diagnostics when the set cannot be established.
- [ ] Share the resolver/resource index with the Helm provider and map Helm releases/images to their owned Kubernetes workloads without inferring platform from values or Keel's runtime.
- [ ] Preserve Docker/OCI single-manifest config resolution, Docker manifest-list and OCI index resolution, and candidate-resolution fail-closed behavior.
- [ ] Carry compatibility evidence far enough into Kubernetes and Helm update selection to prevent an incompatible or stale polling event from being applied.
- [ ] Keep policy filtering, ordering, comparison, tag spelling, webhook semantics, approval behavior, and unrelated registry/provider behavior unchanged.
- [ ] Add focused resolver tests for stable/legacy selectors, required-affinity operators/term semantics, conservative supersets, pod evidence, mixed nodes, empty/unavailable node metadata, and diagnostic reasons.
- [ ] Add Helm tests for one owned workload, multiple/mixed owned workloads, multi-arch compatibility, missing ownership mapping, and unresolved child workload platforms.
- [ ] Retain focused registry tests for amd64, arm/v7, arm64, Docker manifest lists, OCI indexes, single-manifest configs, malformed/missing metadata, and media-type negotiation.
- [ ] Add polling tests proving issue #834's ARM-only candidate is skipped for amd64, the compatible next tag is selected, related mixed-platform workloads require a complete multi-arch candidate, and a per-tag metadata failure does not stop fallback evaluation.
- [ ] Add a deterministic integration/e2e regression from isolated registry metadata through the real poll watcher and Kubernetes workload update selection, with separate issue-shape and mixed/multi-arch scenarios.
- [ ] Capture before/after selected tags and require observable warnings for incompatible candidate, unresolved manifest, and unresolved workload-platform branches.
- [ ] Run `gofmt`, focused registry/poll/Kubernetes/Helm/resolver tests, `go test ./...`, repository build/OpenAPI checks, and the applicable native-k3s e2e and lint checks.
- [ ] Review the final diff for runtime-platform assumptions, tag-name heuristics, overbroad RBAC, unsafe ambiguity handling, policy regressions, mutable external fixtures, and unrelated dependency/UI work.
- [ ] Commit and push the revised implementation branch; do not create a PR until all revision requirements and verification evidence are complete.
