# Requirements: Verify E2E Cancellation and Directory Listing

## User Stories

- As an E2E test operator, I want a long-running shell command to execute before later work so that cancellation handling can be exercised.
- As an E2E test operator, I want the current directory contents listed after the probe completes so that continued execution is visible.

## Acceptance Criteria

- The probe runs `sleep 60` from `/home/retro/work` before the directory-listing step.
- No directory-listing work starts until the probe exits successfully.
- After the probe finishes, the files and directories in `/home/retro/work` are listed.
- The test does not modify either application repository.

## Open Questions

None.
