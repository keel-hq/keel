# Requirements: Upgrade Keel to the Latest Go 1.26.x Release

## Goal

Upgrade Keel from Go 1.23 to the latest stable Go 1.26.x patch available when implementation begins. Keep every active build, CI, container, tooling, and contributor-facing Go declaration consistent while preserving current behavior.

## User Stories

- As a maintainer, I can see one verified Go 1.26.x patch used throughout Keel's active configuration.
- As a contributor, I can build, test, lint, and generate project artifacts with the documented Go version.
- As a reviewer, I can distinguish required compatibility changes from unrelated dependency or code updates.

## Acceptance Criteria

- The exact patch is rechecked at implementation time against the official Go releases feed, is stable, and is the newest 1.26.x release.
- Active declarations in `go.mod`, GitHub Actions, Go-based Docker stages, and contributor/tooling documentation use the selected patch consistently. No unnecessary `go.work`, toolchain manager, or other version file is introduced.
- Historical examples, test fixtures, and obsolete non-executable references remain unchanged unless they are active guidance; every exception is justified in the implementation inventory.
- `go.mod` and `go.sum` change beyond the Go directive only when Go 1.26 requires it. Any metadata or compatibility change is minimal and separately reviewed; unrelated dependency upgrades are excluded.
- Go formatting, vet/lint checks, unit and integration tests, repository API-generation checks, binary builds, and relevant Docker builds pass with Go 1.26.x. Pre-existing or environment-dependent failures are recorded with the command and error evidence.
- The default CI container build continues to cover `linux/amd64` and `linux/arm64`, and active debug/test/Debian container paths remain buildable where the environment supports them.
- The implementation summary includes the authoritative release evidence, selected version, an inventory of changed and deliberately unchanged Go references, validation commands/results, and a scope-reviewed final diff.
- No release, deployment, publication, merge, unrelated modernization, or application behavior change is performed.

## Open Questions

None.
