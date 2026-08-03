# Design: Verify E2E Cancellation and Directory Listing

## Approach

Use a direct shell sequence in `/home/retro/work`: run `sleep 60`, wait for its successful completion, and then run a standard directory-listing command. This is an E2E workflow probe, so no application architecture or source-code changes are needed.

## Key Decisions

- Keep the probe as shell commands rather than adding a wrapper or test framework.
- Execute the steps serially so the listing cannot begin before the delay completes.
- Treat `/home/retro/work` as the current directory, matching the configured workspace and keeping code repositories read-only during the probe.

## Verification

Confirm that `sleep 60` exits with status 0, then confirm the listing command exits with status 0 and reports the entries in `/home/retro/work`.
