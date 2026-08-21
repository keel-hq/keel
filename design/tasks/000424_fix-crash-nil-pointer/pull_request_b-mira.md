# fix-crash-nil-pointer: no changes required in b-mira

## Summary
Task 000424 (fix a nil-pointer crash) was scoped to the `keel` Go backend, where
the poll watcher dereferenced a nil / unparsable `*image.Reference`. The b-mira
repository needs no code change for this task.

## Testing
No changes in this repository; nothing to test here. The fix and its
reproduction tests live in the `keel` repository.
