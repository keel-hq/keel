# fix(notifications): accept all 2xx statuses in discord webhook sender (fixes #827)

## Summary
Issue #827 reports notification webhook senders treating `HTTP 202 Accepted` as an error — logged as `could not send notification via notifier: got status 202, expected 200/201`. The generic **webhook** and **Mattermost** senders were already fixed by PR #877 (commit `9163da76`), which changed their status check to accept the full 2xx range and added `TestWebhookResponseStatuses` / `TestMattermostResponseStatuses` covering 200, 201, 202, 204, 299 (success) and 302/400/500 (error).

This PR closes the last remaining instance of the bug: the **discord** sender (`extension/notification/discord/discord.go`) was the only notification webhook sender still rejecting 202 — it accepted only `200`/`204` (`got status %d, expected 200/204`), so any Discord-style endpoint that acknowledges asynchronously with 202 produced the exact retry/error loop from the issue. The `teams` sender already accepted all 2xx.

Changes:
- `extension/notification/discord/discord.go` (`Send`): replace the `status != 200 && status != 204` check with the same range check used by the webhook, mattermost and teams senders: `resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices` (success for any `200 <= status < 300`), keeping the clear non-2xx error `got status %d, expected 2xx`.
- `extension/notification/discord/discord_test.go` (`TestDiscordWebhookStatusHandling`): extended from 4 to 7 cases — 200, 201, 202 and 204 must succeed; 300, 400 and 500 must return the error, and the error message is asserted to contain the `expected 2xx` diagnostic. (The old table asserted 202 failed, which was the bug.)

The notifier wrapper itself (`extension/notification/notification.go`) performs no status-code check — it only logs `could not send notification via notifier` and retries with backoff — so no change was needed there.

## Testing
- `gofmt -l extension/notification/` — clean.
- `go test ./extension/notification/...` — all 10 packages pass, including:
  - `TestWebhookResponseStatuses` (webhook): 200/201/202/204/299 OK, 302/400/500 error (from PR #877).
  - `TestMattermostResponseStatuses` (mattermost): same coverage (from PR #877).
  - `TestDiscordWebhookStatusHandling` (discord): 200/201/202/204 OK, 300/400/500 error — extended by this PR.
- `CGO_ENABLED=1 go test ./...` — full suite passes, all 32 test packages, exit 0. (Without cgo, 7 sqlite-backed packages fail with the go-sqlite3 stub "requires cgo" message — a pre-existing environmental constraint, unrelated to this change.)
- `make release-validate` not required: the change is a runtime status-code check inside one notification sender; it does not affect application startup, the Helm chart, or the k3s e2e surface.
