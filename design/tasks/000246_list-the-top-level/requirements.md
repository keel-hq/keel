# Requirements: Summarise Top-Level Directories of the Keel Repository

## Background

The user asked for a read-only orientation of the primary repository (`keel` at
`/home/retro/work/keel/`): list its top-level directories and briefly say what
each one contains. The request explicitly forbids modifying, creating or
deleting any files **in the repository under inspection**.

Keel is a Kubernetes deployment-automation tool written in Go (with a Vue.js
web dashboard). It watches container registries for new image versions and
updates Kubernetes Deployments / Helm releases according to per-workload
policies.

## User Stories

**US-1 — Orientation**
As a developer new to the Keel codebase, I want a list of every top-level
directory with a one-line description, so that I can find the right place to
start reading without opening each folder myself.

Acceptance criteria:
- Every directory at the repository root is listed, including dot-directories
  (`.github`, `.pipeline`, `.scripts`, `.test`, `.dependabot`), excluding `.git`.
- Each entry has a short (1–2 sentence) description of its contents/purpose.
- Descriptions are derived from actually reading the tree, not guessed.

**US-2 — No side effects**
As the requester, I want the inspection to be strictly read-only, so that the
`keel` working tree is left byte-for-byte unchanged.

Acceptance criteria:
- No file inside `/home/retro/work/keel/` is created, edited or removed.
- `git status` in `keel` reports a clean tree after the work.
- Spec documents are written only to `/home/retro/work/helix-specs/`.

**US-3 — Reusable notes**
As a future agent, I want the directory summary captured in the design doc, so
that I can skip the discovery step entirely.

Acceptance criteria:
- The full directory table lives in `design.md`, not only in chat output.

## Non-Goals

- No file-by-file documentation, no call graphs, no API reference.
- No changes to `ARCHITECTURE.md` or `readme.md` in the Keel repo.
- The second local repo (`b-alex`) is out of scope; the request says "this
  repository", which resolves to the primary project `keel`.

## Open Questions

- **Which repository is "this repository"?** Assumed `keel`, since it is marked
  the primary project. `b-alex` is also cloned locally. If the intended target
  was `b-alex` (or both), say so and the summary can be redone.
- **Where should the answer land?** Assumed: delivered in the chat response and
  captured in `design.md` here. The "do not modify/create files" constraint was
  read as applying to the inspected repo, not to the `helix-specs` spec
  directory, which this phase is required to write. If no files at all should be
  written, this task is already fully answered by the chat response and the
  implementation phase can be skipped.
- **Does `ARCHITECTURE.md` already suffice?** Keel ships an
  `ARCHITECTURE.md` with a directory-structure table. It is accurate but
  incomplete (it omits `deployment/`, `static/`, `tests/`, `scripts/`,
  `approvals/` vs `bot/` nuances and all dot-directories). This spec covers the
  full set; no update to that file is proposed unless requested.
