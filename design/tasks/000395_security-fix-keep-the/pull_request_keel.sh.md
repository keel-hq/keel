# fix(security): keep webhook signing secrets out of log output — no changes to keel.sh

## Summary

This task (000395, security fix for webhook secret logging, related to keel-hq/keel PR #844) is scoped entirely to the `keel` repository. This repository (keel.sh) is the keel.sh website/docs repository and contains no Go webhook handlers or notification sender code, so no code changes were made here. The feature branch points at the current `master` commit; the actual implementation and tests live in the keel PR.

## Testing

No code changes in this repository. No build or test impact.
