# No changes required in this repository

## Summary

This task updated the dashboard screenshots on the keel.sh website. All of that work landed in the `keel.sh` repository; the `keel` repository needs no changes and this branch has no commits.

The `keel` repo was used only as the source of the application under test. Its UI was built (`ui/npm run build`) and the binary compiled (`go build ./cmd/keel`) so Keel could be run out-of-cluster against a local k3s cluster and photographed, but nothing in the repo was modified. Its `readme.md` references screenshots by absolute URL (`https://keel.sh/img/…`), so the refreshed images are picked up automatically once the keel.sh change is deployed — no edit needed here either.

Two behaviours of the existing code shaped how the demo environment was built, both worth knowing but neither a defect:

- When Keel applies an update it rewrites the image reference in the spec (`provider/kubernetes/updates.go`). Docker Hub images go through `ShortName()`, which expands official images to their `library/` form, while other registries go through `Repository()` and keep the full path. The demo cluster was built on registry-qualified images so the dashboard's image column reads consistently.
- `GET /v1/audit` returns no rows unless a `filter` parameter is supplied, because the query builds `resource_kind IN ()` from an empty slice (`pkg/store/sql/audit.go`). The UI is unaffected — it always sends `filter=*` — so this only surfaces when calling the API directly.

## Testing

- `go build ./cmd/keel` — succeeded.
- `cd ui && npm ci && npm run build` — succeeded; the static bundle was served by Keel via `--ui-dir` and exercised end to end through the browser.
- Ran the resulting binary with `--no-incluster` against a k3d-hosted k3s cluster with 13 annotated workloads. Keel watched all four resource kinds, polled the real registries, applied four updates, and served the dashboard, approvals, tracked-images and audit-log views correctly.
- `git status` on this branch is clean and it is 0 commits ahead of `master`.
