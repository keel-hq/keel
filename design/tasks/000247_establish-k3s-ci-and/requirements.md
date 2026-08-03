# Requirements: Establish Reliable k3s End-to-End Testing for Keel

## Goal

Provide a deterministic, maintainable end-to-end test path that builds Keel, runs it against a fresh single-node k3s cluster, and gates pull requests and image publication without slowing or changing the existing unit-test path.

## User Stories

- As a maintainer, I can run one documented command locally or in GitHub Actions to provision k3s, execute isolated end-to-end tests, collect useful failure evidence, and clean up.
- As a contributor, I get fast feedback that webhook and polling updates work through the real Kubernetes provider, including an ineligible update that must not be applied.
- As an investigator, I can diagnose a failed run from retained Keel logs, test output, Kubernetes state, events, and pod logs without reproducing it first.

## Acceptance Criteria

- A fresh GitHub-hosted Ubuntu runner provisions native single-node k3s with no pre-existing Kubernetes context. The k3s version is exact and visible in source, downloads are verified against the official release checksum, and the workflow prints the installed version.
- The authoritative local command provisions and tears down k3s and its registry fixture. The obsolete kind helper is removed, and the Makefile and contributor/architecture documentation point to the same path.
- The freshly built Keel binary runs outside the cluster with an explicit kubeconfig and isolated writable data directory. Tests poll the Kubernetes API and Keel `/healthz` endpoint with deadlines before scenarios begin.
- The Go tests use `github.com/stretchr/testify/suite`. Suite setup/teardown owns the Keel process and Kubernetes client; per-test setup creates a generated namespace; per-test teardown deletes it and waits for deletion. Cleanup errors fail the test and do not hide the original failure.
- Readiness and state transitions use bounded polling with the last observed state in timeout messages. No correctness assertion relies on a fixed sleep. A negative assertion observes the unchanged image for at least two configured polling cycles.
- The PR-gating suite covers: a successful registry-webhook-driven semver update, a successful polling-driven semver update, and rejection of an update outside the configured policy. Existing webhook and polling intent is retained; redundant public-registry variants and credential-dependent cases may be replaced by deterministic fixtures.
- Core assertions use a local registry populated with explicitly versioned fixture tags derived from a digest-pinned image. Tests require no Docker Hub, GitLab, GHCR, or other registry credentials and do not print credential payloads.
- `make test` remains the fast unit-test command and continues to exclude `./tests`. A separate, clearly named `make e2e` path runs the end-to-end suite.
- GitHub Actions includes a clearly named `End-to-End Tests (k3s)` job on pull requests, pushes to `master`, and `workflow_dispatch`. Image build/publication depends on unit tests, UI lint/tests, and this e2e job; pull requests still build without pushing.
- The e2e job has least-privilege `contents: read` permissions. Package-write permission remains confined to the image job, secrets are not required, and logs/artifacts do not contain tokens or kubeconfig credentials.
- On every test failure, and before cleanup, the workflow captures test output, Keel logs, k3s service logs, node and namespace resource descriptions, events, and relevant pod/container logs. Diagnostics upload uses `if: always()` and a bounded retention period.
- Teardown runs with `if: always()`/shell traps, removes test namespaces, stops the local registry and Keel, uninstalls k3s, and verifies that no task-owned processes or cluster state remain. Cleanup is also exercised after a deliberately failed local test when practical.
- Existing unit, UI, API-contract, and image-build behavior remains intact. No production Keel behavior, runtime dependency, Helm deployment behavior, or UI is changed to accommodate the tests.
- Verification includes `gofmt` on changed Go files, focused helper tests, `make test`, the full e2e command on clean k3s, workflow and shell lint where available, and a report of duration, flakiness, external downloads, runner privileges, and cleanup results.

## Open Questions

- Is an estimated 8–12 minute, one-runner e2e job acceptable as a required check on every pull request? The design recommends this small suite on every PR; path filtering or a larger scheduled suite can be added later if measured cost or queue time is excessive.
- The proposed initial pin is k3s `v1.35.5+k3s1`, a supported stable release verified from the official release assets. Should the implementation instead pin another supported minor to match the project's Kubernetes compatibility policy before CI is enabled?

