# fix(security): keep webhook signing secrets out of log output — no changes to b-mira

## Summary

This task (000395, security fix for webhook secret logging, related to keel-hq/keel PR #844) is scoped entirely to the `keel` repository. No code in this repository (b-mira) reads, writes or logs webhook endpoints, signing secrets, request paths, query strings, headers or payloads, so no changes were made here. The feature branch points at the current default-branch commit; the actual implementation and tests live in the keel PR.

## Testing

No code changes in this repository. No build or test impact.
