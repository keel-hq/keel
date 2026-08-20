---
name: release-keel
description: Prepare, validate, publish, verify, and recover Keel application and Helm chart releases. Use when an agent is asked to release Keel, bump application or chart versions, create or inspect release tags, check release readiness, publish GHCR images or Helm charts, or recover a failed release.
---

# Release Keel

Release application images before publishing the GitHub Release, then release any chart that references the verified application. Read `docs/release-validation.md` for the full validation contract and diagnostics.

## Guardrails

- Require explicit user authorization before pushing tags, creating releases, or changing public registry state.
- Never use the GitHub Release form or `gh release create` to create a new application tag. Push the tag first; CI creates the GitHub Release only after the image passes verification.
- Never move, overwrite, or reuse a public release tag. Correct the source and choose a new version after a failed publication.
- Use a clean release commit on `master`. Do not tag an unmerged feature branch or a commit whose required CI failed.
- Treat the application Git tag as the application build version. Treat `Chart.yaml.appVersion` as the version installed by that chart release.
- Run `make release-validate` for any chart metadata or template change and confirm the Deployment, probes, and k3s suite pass.

## Inspect readiness

1. Run `git status -sb`, `gh auth status`, and inspect the latest application and chart releases.
2. Confirm the proposed application tag, GHCR tag, GitHub Release, chart tag, chart GitHub Release, and Helm index version do not already exist.
3. Inspect `chart/keel/Chart.yaml`. To publish a chart for the new application, set `appVersion` to the application version and choose an unused chart `version`.
4. Run a tag rehearsal before creating remote state:

   ```bash
   GITHUB_REF=refs/tags/<app-version> make release-package
   make release-validate
   ```

## Publish an application

1. Commit the release metadata and workflow changes through a PR.
2. Wait for required CI on the merged `master` commit.
3. Create and push only an annotated application tag:

   ```bash
   git tag -a <app-version> -m "Keel <app-version>" <release-commit>
   git push origin <app-version>
   ```

4. Monitor the tag's `CI` workflow. Require unit, UI, package, k3s release, behavioral, amd64, arm64, manifest, and GitHub Release jobs to succeed.
5. Verify `ghcr.io/keel-hq/keel:<app-version>` is an amd64/arm64 index and the GitHub Release exists only after it.
6. Verify `keelhq/keel:<app-version>` is an amd64/arm64 index on Docker Hub (the CI `docker-manifest` job publishes both registries; historically Docker Hub publishes are stale or missing — do not consider the release complete until Docker Hub has the versioned multi-arch manifest):

   ```bash
   KEEL_PUBLISHED_CHECK_DOCKERHUB=true \
   KEEL_PUBLISHED_APP_VERSION=<app-version> \
   KEEL_PUBLISHED_SKIP_CHART=true \
   make published-release-check
   ```

Do not manually create the GitHub Release before the image. The `github-release` CI job owns that step.

### Tag correspondence

- Application Git tag `<app-version>` is plain SemVer, e.g. `0.22.1` (not `keel-v0.22.1`). It drives both Docker tags `ghcr.io/keel-hq/keel:<app-version>` / `keelhq/keel:<app-version>` and `latest`.
- `keel-v<version>` is a chart release tag, created from the `chart-v<version>` Git tag by `releasecharts.yaml`; it is a GitHub release containing the Helm chart archive, not an application image tag.
- Never publish an application image under a `keel-v*` tag, and never create `keel-v*` from an application Git tag.

## Publish a chart

Publish a chart only after its `appVersion` has both a non-draft GitHub application release and verified amd64/arm64 GHCR image.

```bash
git tag -a chart-v<chart-version> -m "Keel chart v<chart-version>" <release-commit>
git push origin chart-v<chart-version>
```

Monitor `Release Charts`, then verify the chart release and public index:

```bash
KEEL_PUBLISHED_APP_VERSION=<app-version> \
KEEL_PUBLISHED_CHART_VERSION=v<chart-version> \
make published-release-check
```

## Recover a failure

1. Inspect the exact failed job and public GitHub, GHCR, and Helm state.
2. Rerun only when the failed workflow has not created an immutable versioned tag or artifact that the workflow refuses to overwrite.
3. If a public version is incomplete, mark its release notes as withdrawn, fix the source through a PR, and use the next version.
4. Leave orphaned content-addressed image blobs alone; they are not addressable by a release tag.
