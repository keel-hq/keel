# Design: List Files in the Current Working Directory

## Approach

Run `ls -la` with `/home/retro/work` as the working directory and return its
output. Including `-a` makes hidden entries visible, while `-l` provides enough
context to distinguish files from directories. This is a read-only shell task;
no changes to `keel` or `b-alex` are needed.

## Key Decisions and Learnings

- Use the session working directory from the environment rather than assuming
  either repository is the target.
- Keep the solution as a direct shell command; a wrapper or code change would
  add no value.
- The observed top-level project directories are `b-alex`, `helix-specs`, and
  `keel`, alongside hidden tool-state directories.
