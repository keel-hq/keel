# No changes in b-mira for the poll digest drift fix

## Summary
This task fixes the poll trigger's tag digest baseline in the `keel` repository (keel-hq/keel#845). Nothing in `b-mira` is involved: the affected code paths are Keel's poll watcher, its Kubernetes and Helm 3 providers, and its registry client. No files in this repository were added, modified, or deleted.

## Testing
Not applicable — this repository has no changes. The fix is verified in the `keel` repository, where `CGO_ENABLED=1 go test ./...` passes across all 32 packages.
