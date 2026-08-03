# Design: Upgrade Keel to the Latest Go 1.26.x Release

## Current State

Keel currently declares Go 1.23 in `go.mod`, both `actions/setup-go` steps in `.github/workflows/ci.yml`, and `docs/README.md`. `Dockerfile`, `Dockerfile.debian`, the build stage of `Dockerfile.debug`, and `Dockerfile.tests` use Go 1.23.4; the final debug stage still uses Go 1.22.8. The top-level `readme.md` has active but stale Go 1.12/GOPATH/dep contributor guidance. There is no `go.work`, `.go-version`, `.tool-versions`, or `mise.toml`.

The official Go downloads feed reports Go 1.26.5 as stable during planning on 2026-08-03. This is evidence only, not a frozen choice: implementation must query `https://go.dev/dl/?mode=json` again and select the highest stable `go1.26.x` entry.

## Version Synchronization

Use the selected exact `1.26.x` patch in the existing `go.mod` directive, both GitHub Actions setup steps, every Go-based stage in the four active Dockerfiles, and the two contributor prerequisite documents. Updating the debug runtime stage is required because it contains and runs the Go-built Delve tool. Correct the top-level contributor sentence only as needed to replace its obsolete Go/GOPATH/dep instructions with the current module-based prerequisite.

Do not add a `toolchain` directive or a new version-management file: exact existing pins are sufficient and avoid a second selection mechanism. Do not change embedded `golang:1.8.1-alpine` webhook payload examples in `pkg/http`; they are historical external data used by fixtures/comments, not an authoritative project toolchain declaration. Non-Go base images and unrelated pipeline versions are also out of scope.

## Compatibility and Metadata

After changing only the version declarations, use the selected toolchain to run `go mod tidy` and inspect its diff. Commit resulting `go.mod`/`go.sum` changes only if required for Go 1.26; revert incidental churn and do not upgrade dependencies speculatively. If compilation, vetting, generation, or tests expose a Go 1.26 incompatibility, make the smallest source or tooling adjustment that restores existing behavior and explain it independently in the final inventory.

## Verification

Record `go version` and the official release-feed entry, then verify formatting (`gofmt` clean check), `go vet ./...`, `go test ./...`, `make test`, `make api-check`, and the normal binary build. Exercise the default Buildx path for `linux/amd64,linux/arm64` and build the Debian, debug, and test Dockerfiles when supported. Run integration/e2e validation where its Kubernetes or external-service prerequisites are available; otherwise capture the exact environmental limitation rather than treating it as success.

Finish with a repository-wide Go-version search and `git diff` review. The implementation evidence must list each changed declaration, deliberately excluded fixture references, metadata/compatibility adjustments if any, supported container platforms, and every validation result. No image push, release, deployment, merge, or generated artifact update is performed unless the Go change actually alters that artifact and the change is reviewed.

## Key Decisions

- Pin the exact stable patch everywhere so local, CI, container, and documentation paths cannot silently select different 1.26 releases.
- Recheck the official feed at implementation time because a newer 1.26 patch may supersede the planning-time 1.26.5 release.
- Preserve Keel's existing direct-version pattern; previous upgrades changed the current module and Docker declarations without adding a version abstraction.
- Treat dependency and source edits as compatibility exceptions requiring evidence, not as part of routine modernization.

## Implementation Notes

- On 2026-08-03, the implementation-time query of `https://go.dev/dl/?mode=json` returned `go1.26.5` as the first entry with `stable: true`; implementation therefore selects Go 1.26.5.
- The Keel feature branch was absent in the initial clone and was created from the clean local `master` as `feature/000248-upgrade-keel-to-the`.
- The final active inventory is `go.mod`; two setup steps in `.github/workflows/ci.yml`; `Dockerfile`; `Dockerfile.debian`; both Go stages in `Dockerfile.debug`; `Dockerfile.tests`; `docs/README.md`; and the active development prerequisite in `readme.md`. Compose files consume the same Dockerfiles and contain no separate Go pin.
- `Dockerfile.local` contains only an Alpine runtime and has no Go declaration. The two `golang:1.8.1-alpine` strings under `pkg/http` are an external Docker Hub webhook payload example and its test fixture, so they remain unchanged.
- Version synchronization changed eight files: `go.mod`, `.github/workflows/ci.yml`, four Go-based Dockerfiles, `docs/README.md`, and `readme.md`. The top-level README now describes the repository's existing module-based workflow instead of the obsolete GOPATH/dep workflow.
- The installed Go launcher downloaded and ran `go1.26.5 linux/amd64` through `GOTOOLCHAIN=auto`. `go mod tidy` completed successfully and produced no dependency or `go.sum` changes, so no module metadata churn is retained.
- A Go 1.26.5 compile-only pass (`go test ./... -run '^$'`) succeeded for every package, so no source or tooling compatibility adjustment is required. The first full test attempt timed out in existing SQLite-backed tests because the workspace sets `CGO_ENABLED=0`; the emitted `go-sqlite3 requires cgo to work` error is environmental rather than a Go 1.26 compile failure.
