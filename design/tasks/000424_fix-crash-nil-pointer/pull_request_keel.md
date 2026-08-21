# fix(poll): guard against nil or unparsable image references in the poll watcher

## Summary
A tracked image carrying a nil `*image.Reference` — or a zero-valued / unparsed
reference (the shape a helm chart image can produce when it fails to resolve) —
panicked the poll watcher with a nil-pointer dereference. `addJob` calls
`Image.Scheme()`, `Image.Registry()`, and `Image.ShortName()`, and the
`Watch`/`watch` paths call `getImageIdentifier`, all of which dereference the
reference's unexported `named` field and panic instead of returning an error.

This adds `Reference.IsNil()` — safe to call on a nil pointer, and true for a
zero-valued (`&Reference{}`) reference — and guards the three call sites:

- `Watch` skips a malformed image in both the running-digests loop and the main
  loop, recording an error instead of crashing;
- `watch` returns an error up front for a nil/unparsable reference;
- `addJob` returns an error before touching the reference.

A malformed image is now skipped with a recorded error while valid images in the
same batch continue to be watched — one bad release can no longer take the whole
poll loop down.

## Testing
- New reproduction tests in `trigger/poll/watcher_nil_repro_test.go`:
  - `TestAddJobNilImage` — nil image reference yields an error and no watcher entry;
  - `TestAddJobUnparsedReference` — zero-valued reference yields an error instead of a panic in `Registry()`/`ShortName()`;
  - `TestWatchSkipsBadReleaseButSurvives` — a bad image is skipped while a valid one in the same batch is still watched.
  All three panicked against the pre-fix source and pass with the fix.
- `go test ./trigger/poll/... ./util/image/...` passes.
- `go build ./...` compiles; `go vet` is clean for the changed code (the only remaining vet output is pre-existing json-tag noise on unexported fields in `util/image/parse.go`).
