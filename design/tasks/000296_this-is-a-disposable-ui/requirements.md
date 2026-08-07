# Requirements: Create a Disposable README Diff for UI Verification

## User Story

As a UI verifier, I want the primary repository README to contain one short uncommitted change so that the Diff view displays a changed file.

## Acceptance Criteria

- The change is made only in the primary `keel` repository.
- One short line is added to the repository's existing `readme.md` file.
- The README change remains uncommitted.
- No application behavior or other code is changed.
- The full agent-visible README path is reported after the change.

## Open Questions

- The repository uses lowercase `readme.md`, while the request says `README.md`. Should implementation modify the existing lowercase file, as assumed here?
- What exact short disposable line should be added, or may the implementer choose neutral verification text?
