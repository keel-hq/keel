# Design: Establish Reliable k3s End-to-End Testing for Keel

## Current-State Findings

`ci.yml` runs `make test`, which deliberately excludes `./tests`. `make e2e` runs the tests against an assumed kubeconfig. Each current acceptance test starts a host Keel process and repeats namespace lifecycle; readiness uses fixed sleeps, namespace deletion is not awaited, and failures lack cluster diagnostics.

| Current scenario | Confidence provided | Main failure modes |
| --- | --- | --- |
| Docker Hub webhook `0.0.14` to `0.0.15` | Webhook-to-Kubernetes update path | Mutable public image, fixed sleep, host process rather than deployable image/RBAC |
| Integer tag rejection | Semver regression coverage | Weak time-based no-update proof |
| Approval flows | Approval/authentication behavior | One is skipped; the other uses sleeps and public state; not part of this smoke scope |
| Four polling variants | Semver/prerelease selection | Mutable tags, rate limits, tight deadlines, duplicated manifests |
| Private registry polling | Secret discovery/authenticated polling | Usually skipped, long-lived credentials, private state, and encoded credential output |

`tests/helpers.go` contains useful kubeconfig, namespace, and image polling primitives, but panics instead of test-aware failures and does not manage readiness or deterministic cleanup. `.test/e2e-kind.sh` is an obsolete chart-install experiment with floating downloads, outdated kind commands, undeclared inputs, and no safe cluster teardown.

## Execution Architecture

One `.test/e2e-k3s.sh` script is the entry point for `make e2e` and CI. It creates a unique run directory under the repository, verifies the official k3s `v1.35.6+k3s1` release checksum, and starts that binary as a uniquely named transient service/cgroup with explicit task-owned data, kubeconfig, config, log, PID/unit, and artifact paths. It does not install into `/usr/local`, create a normal `k3s.service`, write the default kubeconfig, or call the k3s uninstall script.

Before startup, guards fail closed if a k3s service/process, standard k3s data or kubeconfig, API listener, default k3s network interface, or requested registry/forward port already exists. The run records every created identity. Cleanup first deletes labeled suite resources and waits, then stops only the recorded port-forward and transient service/cgroup and removes only recorded run paths/resources. Repeated cleanup is harmless. Pre-existing installations or clusters are never stopped or removed.

The script deploys a registry pod from a digest-pinned registry image in a run-labeled infrastructure namespace and exposes it through a reserved, task-owned ClusterIP usable by the runner, Keel pods, and k3s containerd without a public listener. k3s receives a run-specific registry configuration for this HTTP test endpoint. A digest-pinned runnable source image is copied into three unique repositories:

- `<run-id>/webhook`: `1.0.0`, `1.0.1`
- `<run-id>/polling`: `1.0.0`, `1.0.1`
- `<run-id>/negative`: `1.0.0`, `1.1.0`

The repositories are never reused or mutated after seeding. Before each test, the suite queries the registry catalog/tags and requires exactly its expected repository and tag set. The negative test therefore cannot discover the polling test's `1.0.1` tag.

The checked-out Dockerfile builds Keel once. The image is pushed to the test registry and referenced by its resulting digest. `SetupSuite` creates a dedicated namespace, ServiceAccount, ClusterRole, ClusterRoleBinding, Keel Deployment, and Service. RBAC follows the chart's production permissions but is reduced to resources exercised by the Kubernetes provider; it grants no secret mutation, impersonation, or cluster administration. The pod uses in-cluster configuration, `INSECURE_REGISTRY=true` only for the isolated fixture, resource bounds, and `/healthz` readiness/liveness probes.

`SetupSuite` waits for the Deployment and starts a uniquely tracked `kubectl port-forward` from loopback to the Keel Service, then polls `/healthz`. `TearDownSuite` removes and awaits all suite-owned Kubernetes resources. `SetupTest` creates a generated workload namespace tied to the test identity; `TearDownTest` deletes and awaits it. Helpers accept contexts, use `require` for prerequisites and `assert` for outcomes, and report last-observed resource/image/condition details.

## Scenarios

1. Deploy `<run-id>/webhook:1.0.0`, send a Docker Registry notification for `1.0.1` through the port-forward, and wait for the Deployment template to reference `1.0.1`.
2. Deploy poll-enabled `<run-id>/polling:1.0.0`, allow the configured poller to discover `1.0.1`, and wait for the Deployment template update.
3. Deploy patch-policy `<run-id>/negative:1.0.0` where the repository contains only `1.1.0` as a newer version, then require the image to remain unchanged across at least two poll intervals.

The integer-tag regression is retained only if it uses its own repository and keeps the total path within the limit. Approval and credential-dependent private-registry cases remain outside this three-test smoke suite; this task does not expand their scope.

## Compatibility Decision

Pin k3s to `v1.35.6+k3s1`, a maintained Kubernetes 1.35 release, and verify its official checksum. Keel currently resolves `k8s.io/client-go` and related libraries to `v0.31.3`. Kubernetes' [component version-skew policy](https://kubernetes.io/releases/version-skew-policy/) does not define client-go skew. The official [client-go compatibility matrix](https://github.com/kubernetes/client-go#compatibility-client-go--kubernetes-clusters) gives exact feature parity only when client and server minors match, but states that older clients work with many newer clusters and that shared APIs continue to work; it does not promise all newer APIs to an older client.

This is a deliberate deployability check, not an assertion that client-go 0.31 has Kubernetes 1.35 feature parity. Keel's tested path uses stable core/apps/RBAC APIs shared by both versions. The suite must fail if those operations are incompatible; dependency upgrades are not hidden inside this CI task. Kubernetes image-volume feature maturity is unrelated because these scenarios do not create image volumes. Using an end-of-life 1.31 server merely to match the library would provide weaker shipping confidence.

## CI, Runtime, and Diagnostics

The only workflow addition is `End-to-End Tests (k3s)` with `contents: read`, a job timeout, and the checked-in local command. The existing `test`, `lint-ui`, and Docker job steps remain unchanged; only the Docker job's `needs` gains e2e. Proposed PR/`master`/manual cadence remains pending Nessie's cost approval.

The three-test path is expected to take 6–8 minutes and has a hard 10-minute acceptance limit measured from provisioning through cleanup on a standard Ubuntu runner. If repeated measurement exceeds 10 minutes, implementation stops before enabling the gate and reports timings and cost/options for review.

Before any cleanup, an `if: always()` step collects and uploads with bounded retention: Go test output; registry pod logs; current and previous Keel pod logs; k3s server and containerd logs; node/pod state; scoped and cluster `kubectl get`/`describe` output; and timestamp-sorted events. Collection excludes Kubernetes Secrets, token-bearing objects, kubeconfig contents, and full environment dumps. Artifact upload and teardown remain useful even when tests or earlier diagnostic commands fail.

## Key Decisions

- Native k3s, not k3d, directly exercises the requested distribution without another wrapper CLI.
- Keel runs in-cluster from the built image with production-equivalent identity, RBAC, probes, and Service access; host `--no-incluster` execution is removed.
- Each test gets a unique immutable registry repository, not merely a namespace.
- `.test/e2e-k3s.sh` replaces the kind helper as the sole local/CI path.
- Unit/UI/build jobs and production behavior stay unchanged; only e2e infrastructure and focused acceptance coverage are in scope.

## Implementation Notes

- The verified amd64 image pins are `registry@sha256:46faa9a1ae6813194b53921a370f2f4f8c5e1aae228a89bceafef5847a6a3278` and `busybox@sha256:7a3ebe5bfd1a4a19797d20b0c0bb39d44393e9a03fd852c0865b0f540d868df0`.
- The official k3s `v1.35.6+k3s1` binary checksum was successfully validated with `sha256sum-amd64.txt`; the resolved build reports commit `87243446` and Go `1.25.11`.
- This sandbox has systemd tools but no running systemd manager. The authoritative script therefore uses a unique `setsid` process group with a recorded root PID instead of installing a service; scoped termination preserves the same ownership guarantee and also works on GitHub runners.
- The registry uses the reserved service address `10.53.0.50:5000` inside the run-specific `10.53.0.0/16` service CIDR. This gives the runner, pods, and k3s containerd one non-public endpoint and avoids changing the host Docker daemon's insecure-registry configuration.
