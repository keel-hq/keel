# Add nightly pre-release workflow with unit tests

## Summary
Implements keel-hq/keel#852: users currently have to build from source to try changes that have landed on `master` but are not yet part of a tagged release. This adds a `Nightly` GitHub Actions workflow that publishes multi-architecture pre-release images to GHCR.

New `.github/workflows/nightly.yml`, structured as four jobs that mirror the existing release path in `ci.yml`:

1. **check** — resolves `master`'s HEAD, computes the `nightly-YYYYMMDD` tag from the run date and a `<appVersion>-nightly.YYYYMMDD` build version validated through `scripts/resolve-application-version.sh`, then compares HEAD against the existing `nightly` git tag. Emits `should_build=false` when no new commits have landed, which skips the whole build. A `workflow_dispatch` `force` input overrides the check.
2. **test** — runs `make test` (which already excludes the cluster-dependent `/tests` suite) plus the OpenAPI contract test. Gates every downstream job.
3. **build** — builds `linux/amd64` on `ubuntu-latest` and `linux/arm64` on `ubuntu-24.04-arm`, pushing by digest. Uses the GitHub Actions cache under a `keel-nightly-<arch>` scope, seeded from CI's `keel-<arch>` scope.
4. **manifest** — joins both digests into `ghcr.io/keel-hq/keel:nightly` and `:nightly-YYYYMMDD`, verifies both tags resolve to identical two-architecture manifests, then idempotently creates-or-moves both git tags via the GitHub API (no-op when already at HEAD, POST when the ref is absent, PATCH otherwise) so re-runs and same-day re-dispatches are safe.

Every job checks out the exact `head_sha` resolved by the `check` job, so a commit landing mid-run cannot make the tested tree and the published image diverge. The workflow runs at 03:00 UTC daily under a non-cancelling `nightly` concurrency group, with `contents: read` by default and write permissions narrowed to the jobs that need them.

`readme.md` gains a "Nightly pre-release images" section documenting the two tags, the Helm values to consume them, and that these are pre-releases not supported for production use.

## Testing
- `actionlint` (v1.7.7, the version pinned by `make release-lint`) runs clean across all workflows including the new one.
- `bash -n` on all eight `run:` scripts embedded in the workflow: no syntax errors.
- Exercised the `check` job's decision logic against the local repository through all four paths: no `nightly` tag → build, tag at HEAD → skip, tag at HEAD with `force=true` → build, tag behind HEAD → build. Each produced the expected `should_build` output.
- Confirmed the generated version string `0.22.1-nightly.20260814` is accepted by `scripts/resolve-application-version.sh`, so the nightly build args satisfy the repository's existing version validation.

Not run: `make release-validate`. `make` is not installed in this environment, and this change touches only CI configuration and the readme — neither application startup nor the Helm chart is affected.
