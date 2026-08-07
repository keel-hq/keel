# Design: Create a Disposable README Diff for UI Verification

## Approach

Append one plain-text verification line to the existing README in the primary `keel` repository. Keep the working-tree modification uncommitted so the UI Diff view can detect and display it.

## Key Decisions

- Use the existing lowercase `readme.md`; the repository has no uppercase `README.md`.
- Make no source-code, configuration, or generated-file changes.
- Do not create a commit in the primary repository. Only these planning documents are committed to `helix-specs`.

## Verification

Run `git status --short` and `git diff -- readme.md` in `keel` to confirm exactly one README file is modified and the added line is visible.
