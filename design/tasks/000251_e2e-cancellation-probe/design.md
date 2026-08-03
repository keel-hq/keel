# Design: List Current Directory Files After a Cancellation Probe

## Approach

Use two direct shell operations in sequence: run `sleep 60`, wait for it to exit successfully, and then run `ls -la` with `/home/retro/work` as the working directory. Report the listing as the E2E result.

## Key Decisions

- Keep the probe as direct shell commands; no wrapper script or application changes are needed.
- Use an explicit working directory so the meaning of “current directory” is deterministic.
- Use `ls -la` so hidden workspace entries are included in the result.

## Repository Notes

The workspace currently contains the local repositories `keel`, `b-alex`, and `helix-specs`, along with hidden tool-state directories. This task does not require inspecting or modifying the application repositories.
