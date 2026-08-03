# Requirements: List Files in the Current Working Directory

## User Story

As a user, I want to see the files and directories in the current working
directory so that I can confirm the E2E command workflow completed.

## Acceptance Criteria

- The command runs from the session's current directory, `/home/retro/work`.
- The result lists every visible top-level entry in that directory.
- The operation is read-only and does not change either code repository.
- The listing is returned directly to the user; no application feature or
  reusable script is created.

## Open Questions

None.
