# Requirements: Summarize the Keel Repository's Top-Level Directories

## User Stories

- As a contributor, I want an alphabetical list of Keel's top-level directories so I can quickly understand the repository layout.
- As a maintainer, I want each directory described briefly and accurately without changing the repository.

## Acceptance Criteria

- The result lists all 25 tracked top-level directories in `/home/retro/work/keel`, including tracked hidden configuration directories.
- Each directory has a short summary based on its README, manifests, or representative files.
- `.git` is excluded because it is version-control metadata rather than project content.
- The list is alphabetical and clearly identifies directory names.
- No files are created, modified, or deleted in `keel` or `b-alex`; the result is returned as text only.

## Open Questions

None.
