# Design: Establish Reliable k3s End-to-End Testing for Keel

## Current-State Findings

`ci.yml` runs `make test`, and that target deliberately excludes `./tests`. `make e2e` installs Keel and runs `go test` from `tests/`, assuming a usable default kubeconfig, a writable runtime data location, and external registry access. Nothing currently provisions a cluster, waits for Keel, captures diagnostics, or guarantees teardown.

| Current scenario | Confidence provided | Main failure modes |
| --- | --- | --- |
| Docker Hub webhook `0.0.14` to `0.0.15` | Webhook parsing reaches the Kubernetes deployment update path | Mutable public image, fixed startup sleep, repeated process/namespace lifecycle |
| Integer tag `45000` rejection | Regression coverage that a plain integer is not treated as semver | Fixed observation sleep; the existing final check can succeed immediately and weakly proves non-update |
| Approval flows | Approval creation and authenticated API behavior | One test is unconditionally skipped; the other uses fixed sleeps/public images and is outside the minimum smoke path |
| Four public polling variants | Semver and prerelease selection through registry polling | Mutable/removed tags, network/rate limits, tight ten-second deadlines, duplicated manifests |
| Private Docker Hub/GitLab polling | Registry-secret discovery and authenticated polling | Usually skipped, requires long-lived credentials, depends on private external state, and one test prints encoded credentials |

`tests/helpers.go` already centralizes kubeconfig loading, namespace creation/deletion, Keel process control, and image polling, but errors are often converted to panics, namespace deletion is not awaited, process startup has no readiness handshake, child environment handling is incomplete, and timeout messages omit useful resource context. `.test/e2e-kind.sh` is a chart-install experiment rather than the current acceptance-test path; it downloads floating binaries from obsolete locations, uses outdated kind commands, depends on undeclared variables, and does not remove the cluster.

## Architecture

The repository will have one orchestration script, `.test/e2e-k3s.sh`, used by both `make e2e` and GitHub Actions. It will download the exact k3s release binary and its official `sha256sum-amd64.txt`, verify the checksum, start a minimal native k3s service with a task-owned kubeconfig, wait for the node and core services, start a digest-pinned Docker Registry container, seed deterministic semver tags, build Keel into a temporary directory, run `go test -v ./tests`, gather diagnostics, and tear everything down through an idempotent trap. The initial pin is `v1.35.5+k3s1`; upgrades are explicit reviewable changes.

The local registry will bind only to loopback. k3s receives an explicit `registries.yaml` mirror configuration for that HTTP endpoint, and Keel runs with `INSECURE_REGISTRY=true` only in this isolated test environment. A small, runnable image pinned by digest is copied into the registry under fixed tags such as `1.0.0`, `1.0.1`, and `1.1.0`. This permits eligible and ineligible policy assertions without mutable third-party tags. The upstream image digest and registry container digest are pinned; no credentials are needed.

The Go entry point becomes one `suite.Run` over an `E2ESuite`. `SetupSuite` validates configuration, creates the Kubernetes client, launches the freshly built Keel process with inherited safe environment plus explicit kubeconfig/data settings, and polls `/healthz`. `SetupTest` creates a generated namespace and common deployment fixture. `TearDownTest` deletes and waits for that namespace, while `TearDownSuite` stops Keel and reports its exit/log state. Helpers accept `context.Context` and `testing.T`, return errors rather than panic, use `require` for prerequisites and `assert` for scenario results, and include namespace, resource, expected image, last image, and relevant conditions in failures.

The initial smoke suite contains three focused tests:

1. Create a deployment at `1.0.0`, submit a Docker Registry notification payload for eligible `1.0.1`, and wait for the deployment template image to change.
2. Create a poll-enabled deployment at `1.0.0`, expose `1.0.1` in the local registry, and wait for Keel polling to update it.
3. Create a patch-policy deployment at `1.0.0` while only `1.1.0` is newer, then prove the image remains unchanged across at least two poll intervals.

The existing integer-tag regression remains useful but is not required in the first PR gate; retain it only if it can share the same fixture and adds negligible runtime. Approval/authentication and private-registry coverage should remain unit/integration coverage or a future extended suite rather than bringing secrets and public mutable state into the smoke gate.

## CI and Diagnostics

`ci.yml` gains `End-to-End Tests (k3s)` with `contents: read`, a job timeout, and the same checked-in command used locally. It runs for every current CI trigger, including pull requests and `workflow_dispatch`. The Docker image job adds the e2e job to `needs`, so publication cannot proceed after an e2e failure. This is estimated at 8–12 minutes and one standard Ubuntu runner, with no external cluster cost.

The script writes test output and Keel logs to a task-owned diagnostics directory. On failure, the workflow adds `kubectl get/describe` output, sorted events, logs from failing workload pods, k3s version/status/journal, and registry logs before teardown, then uploads the directory with short retention. Kubeconfig content, service-account tokens, Secrets, and environment dumps are excluded. Expected external dependencies are GitHub release assets, the digest-pinned registry/fixture images, and Go modules; all versionable tools/images are pinned and checksummed or digest-addressed.

## Key Decisions and Constraints

- Use native k3s, not k3d: the requested platform is exercised directly and GitHub Ubuntu runners support the required privileged/system service operations. k3d would add another pinned CLI and Docker wrapper layer.
- Run the built Keel binary on the host with `--no-incluster`: this preserves the existing acceptance-test model, directly tests the checked-out code, makes logs/process cleanup simple, and still exercises the real k3s API and Kubernetes provider. Deploying Keel via its chart is a separate chart test.
- Gate every PR with only the three deterministic scenarios. Add scheduled/full coverage only after runtime and flakiness data justify it.
- Replace `.test/e2e-kind.sh` instead of maintaining two cluster paths. Update `readme.md`, `ARCHITECTURE.md`, and the Makefile narrowly with prerequisites, exact commands, ownership boundaries, and troubleshooting.
- Keep unit tests separate and do not alter production behavior. Any discovered product defect must be reported and separately justified before changing runtime code.

