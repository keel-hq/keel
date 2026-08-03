# Requirements: Establish Reliable k3s End-to-End Testing for Keel

## Goal

Provide a deterministic end-to-end path that builds the Keel container, runs it inside a fresh native k3s cluster with production-equivalent RBAC, and verifies the highest-value update flows without changing Keel production behavior or the existing fast test jobs.

## User Stories

- As a maintainer, I can run one documented command locally or in GitHub Actions to create an isolated k3s environment, execute the smoke suite, retain useful diagnostics, and safely clean up only that run.
- As a contributor, I can prove that the deployable Keel image can update Kubernetes workloads through webhook and polling triggers and reject an ineligible update.
- As an investigator, I can diagnose a failure from the uploaded test, cluster, registry, and in-cluster Keel evidence.

## Acceptance Criteria

- A fresh GitHub-hosted Ubuntu runner downloads native k3s `v1.35.6+k3s1` and its official checksum file, verifies the binary, and prints the exact version. No floating or unverified executable is downloaded.
- The checked-out Dockerfile builds the Keel image under test. That exact image digest runs as a pod inside k3s with a dedicated ServiceAccount, the minimum chart-equivalent ClusterRole/ClusterRoleBinding needed by the Kubernetes provider, readiness/liveness probes, and a ClusterIP Service.
- Tests wait with deadlines for the node, registry, Keel Deployment, and `/healthz`. A task-owned `kubectl port-forward` to the Keel Service drives webhooks without exposing Keel externally, and Keel failures are diagnosed from pod logs.
- The Go suite uses `github.com/stretchr/testify/suite`. Suite setup/teardown owns the Keel test namespace, RBAC, Deployment, Service, and port-forward; per-test setup creates a generated namespace; per-test teardown deletes it and waits for deletion.
- Every scenario uses a repository name containing the run ID and test identity. Only that scenario's immutable tags are seeded, the test verifies the registry's exact tag set before exercising Keel, and no repository/tag state from another test can affect discovery.
- Registry and fixture source images are digest-pinned. Core assertions need no public mutable tags, registry credentials, GitHub tokens, or private registry state, and no secret or encoded credential is logged.
- The smoke suite covers: a successful registry-webhook semver update, a successful polling semver update, and a patch-policy case that rejects a minor update. The negative case observes the unchanged image across at least two poll cycles.
- Fixed sleeps are not used for readiness or correctness. Bounded polling failures include the namespace/resource, expected condition or image, last observed state, and timeout.
- `make test`, the existing `Unit Tests` job, the existing `Lint UI` job and its steps, and existing Docker build/push behavior remain unchanged. `make e2e` is separate and authoritative; no new UI-test or API-contract job is introduced.
- The only CI addition is a clearly named `End-to-End Tests (k3s)` job plus adding it to the Docker job's `needs` so image publication cannot follow an e2e failure. The proposed pull-request, `master`, and `workflow_dispatch` cadence remains pending cost approval.
- The complete three-test PR smoke path targets 6–8 minutes and must finish in at most 10 minutes on a standard GitHub Ubuntu runner. Implementation records measured end-to-end runtime; if it cannot meet 10 minutes, gating is not enabled and the evidence and cost tradeoff are reported for a decision.
- The e2e job has `contents: read` only. Package-write permission remains confined to the Docker job.
- An `if: always()` diagnostics step runs before cleanup and uploads an artifact with bounded retention containing registry logs, Keel pod logs, k3s server and containerd logs, Go test output, Kubernetes `get`/`describe` output, sorted events, and node/pod state. Secrets, tokens, environment dumps, and kubeconfig contents are excluded.
- Local startup refuses to overwrite, reuse, uninstall, or stop an existing k3s installation, standard kubeconfig, active API port, or existing k3s network state. It uses a unique run directory, kubeconfig, data directory, transient service/process identity, namespaces, labels, registry resources, and port-forward.
- Teardown is idempotent, captures diagnostics first, and deletes/stops only resources and process identities recorded as created by the run. It never invokes a broad k3s uninstall or unscoped process kill. Cleanup is verified after success and, when practical, a deliberately induced failure.
- The obsolete kind helper is replaced so there is one e2e path. The Makefile, `readme.md`, and `ARCHITECTURE.md` document prerequisites, exact local command, version pin, expected duration, diagnostics, and cleanup guards.
- Verification includes `gofmt`, focused helper tests, `make test`, the full clean-cluster e2e command, shell/workflow lint where available, and separate reporting of runtime, flakiness, external downloads, permissions, and cleanup results.
- No unrelated runtime dependency, production behavior, Helm behavior, deployment behavior, UI, unit-test, lint, or build change is included.

## Open Questions

- Does Nessie approve the proposed one-runner, 6–8 minute smoke job on every pull request and `master` push, with `workflow_dispatch` retained? Until cost approval, the implementation may prepare the job but must keep the cadence decision visibly pending rather than representing it as approved.

