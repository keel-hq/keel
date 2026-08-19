# fix(security): keep webhook signing secrets and credentials out of log output

## Summary

Supersedes and extends community PR #844 (https://github.com/keel-hq/keel/pull/844), which redacted only the Microsoft Teams sender and still logged the full signed URL in debug mode. PR #844 is intentionally left open and unmerged.

Webhook-related log statements could leak secrets into log output:

- `pkg/http` auth middleware logged the Basic auth **password** at error level and the **bearer token** at warn level whenever an authenticated webhook request failed authentication.
- The `teams`, `discord`, `webhook` and `mattermost` notification senders logged the **full configured endpoint URL** at info level. These URLs embed the webhook signing secret in the path (e.g. Teams `<guid>@<tenant>`, Discord/Mattermost token segments) or query string.
- `mattermost` additionally logged the raw endpoint on the invalid-endpoint error path, and all four senders returned `url.ParseRequestURI` errors — which echo the input — back to the notification manager, which logs returned configure errors.

Changes:

- New `extension/notification/sanitize.go` with two helpers:
  - `SafeURL` — safe at any log level: keeps only `scheme://host`; path, query and userinfo (which may carry tokens) are dropped.
  - `DebugURL` — for debug level only: keeps the path structure for diagnostics while redacting userinfo, token-like path segments (≥16 chars, UUIDs/GUIDs, `@`-suffixed keys) and all query values.
- All four webhook senders now log only `SafeURL` at info level. The more detailed endpoint is logged **only when debug logging is enabled** (`log.IsLevelEnabled(log.DebugLevel)`) and is redacted via `DebugURL` even there.
- Auth middleware no longer logs passwords or tokens on failed authentication (also fixes the `failed uath` typo).
- Endpoint parse-error messages no longer echo the raw URL.

Audit scope: every log statement in the inbound webhook trigger handlers (`pkg/http/*_webhook_trigger.go`, `registry_notifications.go`, `auth.go`) and the outbound notification senders, plus any code logging request paths, query strings, headers or payloads. No other statement can carry a secret: remaining payload logs (jfrog, harbor, registry notifications) are debug-level only, and the shoutrrr sender already redacts (covered by its existing tests). `bot/` and `cmd/keel/main.go` log no request data or secrets.

## Testing

- New log-capture unit tests (logrus hook capturing all emitted entries):
  - `pkg/http/webhook_log_redaction_test.go` — `TestWebhookHandlerNeverLogsCredentials`: sends real webhook requests to the `/v1/webhooks/native` endpoint (authenticated-webhooks mode) with a Basic auth password and a bearer token, and asserts neither secret appears in captured log output at info level **and** debug level. A mutation check confirmed the test fails when the original leak is re-introduced.
  - Sender tests (`webhook`, `teams`, `discord`, `mattermost`) — configure each sender with an endpoint embedding a signing secret and assert the captured log output never contains the secret at info or debug level, while the host is still logged for operability.
  - `mattermost` — invalid-endpoint error path: the raw endpoint is never logged.
  - `discord` — `TestConfigureInvalidEndpointRejected`: a malformed webhook URL (no scheme) makes `Configure` return `(false, err)`, and the raw endpoint leaks neither into the error message nor into captured log output at debug level.
  - `extension/notification/sanitize_test.go` — table tests for `SafeURL`/`DebugURL` across Teams, Discord, Mattermost, query-token and user:info URL shapes.
- `gofmt`: clean on all changed files.
- `go test ./...`: passes (exit 0, all packages).
