# Requirements: Add Workspace Inspector E2E Fixture

## User Stories

- As an end-to-end UI test author, I want a small TypeScript workspace fixture so the workspace inspector can display source and documentation files.
- As a reader, I want the fixture documented so its purpose and exported function are clear.

## Acceptance Criteria

- A `workspace-inspector-e2e/alpha.ts` file exists and exports an `add` function.
- `add` accepts two typed numeric arguments and returns their numeric sum.
- A `workspace-inspector-e2e/README.md` file exists and briefly describes the fixture and `add` export.
- Both files remain uncommitted, and no implementation commit is pushed.
- Verification confirms both files exist before finishing.

## Open Questions

- Which local repository should contain `workspace-inspector-e2e/`? The request gives a relative path but does not identify `keel`, `keel.sh`, or `b-alex`.
