# Task 000392: fix-notification-webhook (keel-hq/keel #827) — OUTCOME: ALREADY FIXED

## Finding
Issue #827 ("notification webhook senders treat HTTP 202 Accepted as an error",
log: `got status 202, expected 200/201`) is **already fixed and merged into
master** before this task's branch was cut. The fix landed via
**PR #877** (branch `feature/000265-accept-valid-http-202`, merge commit
`1423d525`, fix commit `9163da76` "fix(notifications): accept all successful
webhook statuses").

## Verification performed (2026-08-18)
- `origin/master` (`7a529a38`, re-fetched from the remote) contains:
  - `extension/notification/webhook/webhook.go` — accepts all 2xx
    (`resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices`),
    clear non-2xx error: `got HTTP status %s, expected 2xx`.
  - `extension/notification/mattermost/mattermost.go` — same 2xx check and error.
  - Unit tests `TestWebhookResponseStatuses` and `TestMattermostResponseStatuses`
    asserting 200, 201, 202, 204, 299 succeed and 302/400/500 return the
    status error.
- `gofmt -l extension/notification/` → clean.
- `go test ./extension/notification/...` → all 10 packages pass.
- `CGO_ENABLED=1 go test -count=1 ./...` → exit 0, all 32 test packages pass.
  (A CGO_ENABLED=0 run fails 7 sqlite-backed packages with
  "go-sqlite3 requires cgo" — environmental, unrelated to this change; the
  repo's Makefile builds with CGO enabled.)

## PR
Branch `feature/000392-fix-notification-webhook` was pushed to origin, but it
contains **no commits beyond master** (the working tree was clean and identical
to master at task start). A PR with no diff cannot be opened; the existing
PR #877 already covers issue #827, so no new PR was created and no
placeholder commit was fabricated.
