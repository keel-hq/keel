# Implementation Tasks: Establish Reliable k3s End-to-End Testing for Keel

- [ ] Record the final supported k3s version, official checksum source, digest-pinned registry/fixture images, expected runner privileges, and measured baseline runtime.
- [ ] Replace `.test/e2e-kind.sh` with one idempotent `.test/e2e-k3s.sh` path that provisions verified native k3s, configures kubeconfig and the loopback registry, builds Keel, runs tests, captures diagnostics, and always tears down.
- [ ] Refactor `tests/helpers.go` into context-aware testify helpers for Kubernetes clients, Keel process/readiness management, generated namespaces, bounded state polling, and deletion verification with actionable failures.
- [ ] Restructure the acceptance tests under a shared `testify/suite` lifecycle and remove duplicated Keel and namespace setup.
- [ ] Add deterministic local-registry coverage for an eligible webhook update, an eligible polling update, and a policy-ineligible no-update case; retain the integer-tag regression if it remains focused and deterministic.
- [ ] Remove or relocate skipped, credential-dependent, mutable-public-tag scenarios so the required smoke suite needs no external registry state or secrets.
- [ ] Keep `make test` unchanged as the fast unit path and make `make e2e` invoke the authoritative clean-cluster workflow with documented configuration overrides.
- [ ] Add the least-privilege `End-to-End Tests (k3s)` GitHub Actions job for pull requests, `master` pushes, and manual dispatch, with a timeout, always-run diagnostics upload, and always-run cleanup.
- [ ] Require the e2e job before Docker image publication while preserving pull-request build-only behavior and existing unit/UI/API checks.
- [ ] Update `readme.md` and `ARCHITECTURE.md` with local prerequisites, the exact e2e command, version pins, expected duration, failure artifacts, and the unit/e2e separation.
- [ ] Run `gofmt`, focused helper tests, `make test`, shell/workflow validation, and the full e2e path on a fresh cluster; record actual duration and flakiness observations.
- [ ] Deliberately induce one test failure when practical, verify diagnostics are produced before cleanup, and confirm namespaces, Keel, registry, and k3s are removed after both successful and failed runs.
- [ ] Review the final diff for secret exposure, floating downloads/tags, overly broad permissions, unrelated production/deployment/UI changes, and any unreported external dependency.
