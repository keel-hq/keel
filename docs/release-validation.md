# Release process and validation

This document describes the release path as audited on 2026-08-05. Release validation is read-only with respect to GitHub and public registries. It never creates tags, releases, images, or charts.

## Release paths and version contract

Keel has two related release paths.

An application release starts with a SemVer Git tag such as `0.21.1`. `.github/workflows/ci.yml` runs Go tests, the UI install/typecheck/lint/test/build, API generation checks, and the packaged-chart k3s suite before any image job can publish. Native amd64 and arm64 jobs build `Dockerfile`, push content-addressed images without tags, and the manifest job creates the GHCR tag and `latest` only after both digests exist. It then verifies that every produced tag resolves to the same amd64/arm64 index. The workflow needs `contents: read`; only image publication jobs receive `packages: write`. Pull requests do not log in or push. GitHub application releases are currently created outside this repository's workflows after the tag/image succeeds; that manual boundary is not changed here.

A chart release starts with `chart-v<Chart.yaml version>`. `.github/workflows/releasecharts.yaml` validates the tag, packages the source chart without editing it, and confirms that `Chart.yaml.appVersion` already has a public GitHub application release and amd64/arm64 GHCR image. Only then does chart-releaser package the chart, create the `keel-v<chart version>` GitHub release asset, and update the GitHub Pages Helm index. The validation job has `contents: read`; only chart-releaser receives `contents: write`. The workflow cannot run from `pull_request`.

`chart/keel/Chart.yaml` is the release contract:

- `appVersion` is the canonical application version and must exactly equal an application Git tag.
- `version` is the canonical source chart version; the trigger must be `chart-v${version}`. The shared packager and the chart release's temporary checkout add the historical leading `v` to packaged chart metadata and filenames.
- `Dockerfile` receives the validated application version and Git revision as build arguments. Its fallback tag discovery remains available for ordinary developer builds.
- The chart defaults its image tag to `appVersion`. Validation renders and inspects that exact wiring and packages with explicit `--version` and `--app-version` arguments.

This makes tag, binary, container, and chart drift fail before publication. A chart release also refuses to run when its GitHub release or Helm index version already exists. An application image release refuses to overwrite an existing GHCR release tag. Branch tags such as `master` retain their existing mutable semantics.

## Local commands

Run the complete non-publishing candidate validation from a clean Linux/amd64 host:

```bash
make release-validate
```

Prerequisites are Go 1.26.5, Docker with BuildKit, `curl`, `jq`, `sha256sum`, `tar`, iproute2, `setsid`, and passwordless `sudo`. The command downloads checksum-verified Helm v3.18.4 and k3s v1.35.6+k3s1, builds the release Dockerfile, lints/templates/packages the chart, starts the repository's isolated native-k3s harness, and cleans it automatically. It refuses to share an existing k3s installation or the test ports.

Faster, non-cluster checks are:

```bash
make release-lint
make release-package
make published-release-check
```

`release-package` creates the candidate image and chart under ignored `.test/release-artifacts/`. Set `KEEL_RELEASE_PACKAGE_ONLY=true` only when invoking `scripts/release-validate.sh` directly; the Make target already does this. `published-release-check` reads GitHub, the Helm index/archive, GHCR, and the chart-referenced registry without authentication or mutation.

For a tag rehearsal without publishing, set `GITHUB_REF` locally. These examples should pass only when metadata agrees:

```bash
GITHUB_REF=refs/tags/0.21.1 make release-package
GITHUB_REF=refs/tags/chart-v1.2.0 KEEL_RELEASE_SKIP_IMAGE=true make release-package
```

## Kubernetes coverage

The release validator uses the packaged `.tgz`, not source templates. It performs these isolated checks before running the existing webhook and polling behavior suite:

1. Install candidate defaults with only the local immutable image and ClusterIP Service overrides. Verify Deployment image wiring, both HTTP probes, Service endpoints/routing, ServiceAccount, Secret, ClusterRole, and ClusterRoleBinding. Call `/version` through the Service and require the embedded candidate version. Verify persistence remains disabled, then uninstall and check cluster-scoped RBAC did not leak.
2. Install the candidate with `auth.mode=external-proxy`, the chart's digest-pinned oauth2-proxy sidecar, a disposable Secret, and a ClusterIP Service. Verify sidecar/image, environment, probes, and Service target wiring; require `/ping` through the Service and verify that an unauthenticated client cannot gain access by supplying a forged identity header. Uninstall and check cleanup.
3. Download published chart `v1.0.5`, verify SHA-256 `8826a68c962d2641232897997f4fb2b547b6f1d252bffa8e9860113361cfb938`, and install its amd64 image by digest (`keelhq/keel:0.20.0`, digest `sha256:8c7492034b10f3cf718e8ab9f860501aeb5bf13812fa675695ea59c191295fa4`). Its historical `/version` response is required to match the observed empty embedded version field.
4. Confirm the historical image wrote `/data/keel.db`, then normalize ownership inside the disposable static-hostPath fixture (hostPath does not implement `fsGroup` ownership changes). Upgrade to the candidate using `fsGroup: 666`, Basic Auth configuration, debug mode, Helm-v2-provider disabled, ClusterIP Service, and a non-privilege-escalating container security context. Require readiness, Service version, rendered environment/security settings, Helm history, and preservation of the PVC UID. CSI-backed production volumes should apply the declared filesystem group directly; the explicit `chown` is limited to this test fixture.
5. Roll back to revision 1, require the prior version through the Service, verify the PVC is still preserved, uninstall, delete the namespace, and check for leaked global RBAC.
6. Run meaningful Keel behavior checks against the same candidate image: registry webhook update, polling update, negative policy cases, the external OAuth-proxy Admin flow, and the existing regression scenarios.

## Diagnostics and failure recovery

Failures retain `.test/artifacts/<run>/` locally and upload it in CI only on failure. The bundle includes Helm status/history/values/manifest, Kubernetes objects and events, Pod descriptions and current/previous logs, registry logs, k3s/containerd logs, rendered chart output, chart archive metadata/checksum, image inspection, and embedded version output. It deliberately excludes Secrets, kubeconfig contents, tokens, and environment dumps.

The public release paths are not transactionally atomic. Application architecture digests become public before their manifest tag; chart-releaser creates a GitHub release asset and updates the Pages index in separate remote operations. On failure, do not retag or overwrite. Inspect the workflow logs and public state, leave orphaned content-addressed blobs alone, correct the source, and use a new version. A successful Helm rollback is:

```bash
helm history keel --namespace <namespace>
helm rollback keel <known-good-revision> --namespace <namespace> --wait --timeout 3m
kubectl --namespace <namespace> rollout status deployment/keel --timeout=3m
```

## Published evidence and known gaps

The read-only check currently records:

- GitHub application release `0.21.1`, GHCR `ghcr.io/keel-hq/keel:0.21.1`, and a Linux amd64/arm64 OCI index.
- Helm index/chart `v1.0.5`, archive digest `8826a68c962d2641232897997f4fb2b547b6f1d252bffa8e9860113361cfb938`, `appVersion: 0.20.0`, and its available Docker Hub image.
- The chart release asset and Helm index URL/checksum agree exactly.

There are no release checksums attached separately to application GitHub releases, chart provenance (`.prov`), signatures, attestations, or SBOM artifacts in the audited process. GHCR's index currently includes attestations emitted by the image build, but their predicate and signer are not part of a documented verification contract. Adding keyless signing, verified provenance, and SBOM policy should be a separately scoped release-security change.

The published `keelhq/keel:0.20.0` image used by chart `v1.0.5` reports an empty application version and revision from `/version` (build date `2024-12-22T191328Z`). The candidate path fixes this class of drift by passing and asserting explicit Docker build metadata, but the immutable historical artifact is intentionally not altered.

The chart supports repository and tag fields but has no native digest value; consequently, published chart defaults remain vulnerable if an external registry tag is moved. Candidate tests use a run-unique local tag and separately record the image inspection result. Adding digest-native chart wiring would be a compatibility-affecting chart feature and is left for a separately scoped change.

The current application GitHub release creation step is manual and therefore cannot be proven or recovered by repository CI alone. The release jobs also cannot make GitHub Release, GHCR, and GitHub Pages publication atomic; the new preflight checks, dependency gates, immutable-version refusal, and diagnostics limit rather than eliminate that residual risk.
