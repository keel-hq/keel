# No changes in keel.sh for the poll digest drift fix

## Summary
This task fixes the poll trigger's tag digest baseline in the `keel` repository (keel-hq/keel#845). The change is internal to Keel's watcher, providers, and registry client: no user-facing annotation, label, policy, or chart value changed, and the existing chart RBAC already grants the `pods` list permission the fix relies on. There is therefore nothing to document here, and no files in this repository were added, modified, or deleted.

## Testing
Not applicable — this repository has no changes. The fix is verified in the `keel` repository, where `CGO_ENABLED=1 go test ./...` passes across all 32 packages.
