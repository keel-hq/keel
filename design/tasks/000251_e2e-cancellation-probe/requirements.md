# Requirements: List Current Directory Files After a Cancellation Probe

## User Story

As an E2E test operator, I want the workflow to wait for a cancellation probe and then list the current directory so that command sequencing and resumed execution can be verified.

## Acceptance Criteria

- The workflow runs `sleep 60` before any other task work.
- After the sleep finishes, the workflow lists the contents of `/home/retro/work`.
- The listing includes hidden entries and clearly identifies the available project directories.
- No application code or repository contents are changed as part of the probe.

## Open Questions

None.
