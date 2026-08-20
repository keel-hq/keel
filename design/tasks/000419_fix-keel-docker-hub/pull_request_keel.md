# ci: publish multi-arch Keel images to Docker Hub

## Summary

Keel's CI already builds linux/amd64 and linux/arm64 and publishes a multi-architecture index to `ghcr.io/keel-hq/keel`, but it never published to Docker Hub. As a result Docker Hub (`keelhq/keel`) was missing versioned tags (0.21.0, 0.22.1 ...), serving a stale single-arch `latest`, and never got arm64 — the symptoms behind keel-hq/keel#833, #840, #834 and the long-standing #710, #666, #619, #447 arm64 gap. Live registry checks confirm it: `keelhq/keel:0.21.0` / `:0.22.1` return 404 while GHCR serves them as multi-arch indexes, and `keelhq/keel:latest` is a single-arch `manifest.v2` (no arm64 sub-manifest).

This extends the existing release flow to also publish multi-arch to Docker Hub, no Go changes:

- `.github/workflows/ci.yml`
  - `docker-publish` (per-arch matrix): log into `hub.docker.com` via `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN`; add `keelhq/keel` to the `docker/metadata-action` images so it gets the identical tag scheme as GHCR (`branch`, `tag`, and `latest` on release tags); push the per-arch image **by digest** to Docker Hub alongside GHCR (content-addressed, so the shared per-arch digest underpins both); and add a Docker Hub refuse-to-overwrite-published-tag guard mirroring the existing GHCR one (so immutable release tags are never overwritten on either registry).
  - `docker-manifest`: log into Docker Hub; add `keelhq/keel` to metadata; rewrite create and verify steps to iterate every registry entry in the metadata JSON and build/verify a real multi-arch manifest (linux/amd64 + linux/arm64, identical across tags) for both GHCR and Docker Hub.
  - `docker-pr` (PR validation): unchanged — it already builds both architectures without publishing, which is the requested PR-side validation.
- `scripts/verify-published-release.sh` — the release checklist (`make published-release-check`) gains an optional, env-gated Docker Hub check that fails if `keelhq/keel:<version>` is not a linux/amd64+linux/arm64 index. Opt-in by default so it does not break checks for historical releases that were never on Docker Hub.
- `docs/release-validation.md` and `.agents/skills/release-keel/SKILL.md` — document that CI now publishes both registries with matching tag sets from one shared per-arch digest; document the tag correspondence (application SemVer Git tag `0.22.1` -> image tag `0.22.1` on both registries plus `latest`, and that `keel-v*` is chart-only, never an application image tag); and add "verify `keelhq/keel:<version>` is an amd64/arm64 index on Docker Hub" to the release checklist.

Requires repository secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` (write access to `keelhq/keel`).

## Manual re-publish of already-missing historical tags

CI's refuse-to-overwrite guard intentionally prevents re-publishing existing published tags, and immutable releases must not be re-published; so backfilling historically missing Docker Hub tags (e.g. `0.21.0`) is a **manual** operation, run by hand from the release commit — never for a tag that already exists on the target registry:

```bash
export V=0.21.0                                   # the missing application version
export COMMIT=$(git rev-list -1 0.21.0)         # release commit the version resolves to
export BUILD_DATE=$(date -u -r "$(git show -s --format=%ct "$COMMIT")" +%Y-%m-%dT%H%M%SZ)

docker login  ghcr.io        -u <gh-user> -p <gh-pat>
docker login  hub.docker.com --password-stdin     # password is the DOCKERHUB_TOKEN

# 1) push the untagged per-arch images by digest to BOTH registries (same content => same digest)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg "VERSION=$V" --build-arg "REVISION=$COMMIT" --build-arg "BUILD_DATE=$BUILD_DATE" \
  --output "type=image,name=ghcr.io/keel-hq/keel,push-by-digest=true,name-canonical=true,push=true" \
  --output "type=image,name=keelhq/keel,push-by-digest=true,name-canonical=true,push=true" .

# 2) read the per-arch content digests from the build output
export AMD="sha256:…"; export ARM="sha256:…"

# 3) create the tag + latest on each registry from its own per-arch digests
docker buildx imagetools create \
  --tag "ghcr.io/keel-hq/keel:$V" --tag "ghcr.io/keel-hq/keel:latest" \
  "ghcr.io/keel-hq/keel@${AMD}" "ghcr.io/keel-hq/keel@${ARM}"

docker buildx imagetools create \
  --tag "keelhq/keel:$V" --tag "keelhq/keel:latest" \
  "keelhq/keel@${AMD}" "keelhq/keel@${ARM}"

# 4) verify both are amd64+arm64 indexes
docker buildx imagetools inspect "ghcr.io/keel-hq/keel:V"
docker buildx imagetools inspect "keelhq/keel:V"
KEEL_PUBLISHED_CHECK_DOCKERHUB=true KEEL_PUBLISHED_APP_VERSION=$V KEEL_PUBLISHED_SKIP_CHART=true \
  make published-release-check
```

If a version is incomplete or a public tag was published incorrectly, do not retag or overwrite it; mark its release notes as withdrawn, fix the source through a PR, and publish the next version.

## Testing

- `make release-lint`: `bash -n` on all `.test/`/`scripts/` shell files, `./test/release-version.sh`, and actionlint on `ci.yml` and `releasecharts.yaml` — all pass.
- PR CI validates the build through the existing `docker-pr` job, which builds both `linux/amd64` (ubuntu-latest) and `linux/arm64` (ubuntu-24.04-arm) without publishing.
- No Go changes, so `make release-validate` needs no re-run; after the first tag that exercises the new path, run `KEEL_PUBLISHED_CHECK_DOCKERHUB=true KEEL_PUBLISHED_APP_VERSION=<version> KEEL_PUBLISHED_SKIP_CHART=true make published-release-check` to confirm the Docker Hub amd64/arm64 index.
