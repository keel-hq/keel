# Document top-level directory layout of Keel

## Summary

Read-only survey of the Keel repository root, requested as an orientation aid.
The request explicitly forbade modifying, creating or deleting any files, so
**this task produces no code changes** — the deliverable is the directory
summary itself, captured in the design docs on the `helix-specs` branch.

The `keel` working tree is untouched and remains on `master`; no feature branch
was created.

## Changes

- No changes to the `keel` codebase.
- `design/tasks/000246_list-the-top-level/design.md` — full table of every
  top-level directory, grouped into Go application code, frontend, deployment
  and packaging, tests/tooling/assets, and dot-directories.
- `design/tasks/000246_list-the-top-level/requirements.md` — user stories,
  the read-only constraint, and open questions on scope.
- `design/tasks/000246_list-the-top-level/tasks.md` — checklist.

## Notes for reviewers

Two findings worth carrying forward:

- Keel already ships `ARCHITECTURE.md`, which covers most of this plus data
  flow and the `keel.sh/*` annotations. Its directory table is however
  incomplete — it omits `deployment/`, `tests/`, `scripts/`, `static/` and all
  dot-directories. Updating it was **not** done, as that would have meant
  modifying the repo.
- Registry webhook triggers do not live in `trigger/`; they are in
  `pkg/http/*_webhook_trigger.go`.

## Testing

Not applicable — no code changed. Verified `git status --porcelain` in
`/home/retro/work/keel/` is empty.
