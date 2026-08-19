# fix(slack): include concrete missing scopes in bot startup error (fixes #842)

## Summary
When Keel's Slack bot is started with a token that lacks the required scopes, the fatal startup log only showed `main: failed to setup slack bot, error=missing_scope` — with no indication of WHICH scopes are missing, and no pointer to documentation of the required scopes (issue #842).

Changes:
- `bot/slack/slack.go`: the Slack API calls made during bot startup (`users.list` via `GetUsers()` and `conversations.list` via `findChannelId`) now pass their errors through `augmentSlackAuthError`. When the Slack error payload's `error` field is `missing_scope`, the returned error names the failing API call, lists the concrete scopes that call requires (e.g. `users:read`), and the full set of scopes Keel's Slack integration needs — bot token scopes (`app_mentions:read`, `channels:history`, `channels:read`, `chat:write`, `files:write`, `users:read`) and app-level token scopes (`connections:write`) — with a note about the `groups:` variants for private approval channels and a link to the setup guide. `invalid_auth` errors get a hint to check the `xoxb-`/`xapp-` token configuration. All other errors are returned unchanged.
- The required scopes are defined once in code (`requiredBotScopes`/`requiredAppScopes`) and rendered into the error message, so the log can never drift from the documented list.
- `readme.md`: new "Slack bot" section under Configuration documenting the exact scopes Keel requires for its Slack integration (bot token scope table with per-scope purpose, private-channel `groups:` note, app-level `connections:write` scope), linking to the full [Configuring Slack](https://keel.sh/docs/#configuring-slack) guide on keel.sh, which also ships a ready-to-import app manifest.
- `bot/slack/slack_test.go`: unit tests that parse representative Slack error payloads (`{"ok":false,"error":"missing_scope","needed":"users:read","provided":""}` and `{"ok":false,"error":"invalid_auth"}`) through the slack-go client's own `SlackResponse` decoding path and assert the enriched error message contains the concrete missing scopes, the full required-scope lists, and the docs link; a third test verifies non-auth errors and `nil` pass through unchanged.

Resulting startup log for a bot token missing `users:read`:

```
main: failed to setup slack bot,
error=slack api error "missing_scope" from "users.list": the token is missing the required scope(s) [users:read]. Keel requires the bot token (SLACK_BOT_TOKEN) to have scopes [app_mentions:read, channels:history, channels:read, chat:write, files:write, users:read] and the app-level token (SLACK_APP_TOKEN) to have scopes [connections:write]; for a private approval channel the channels: scopes become groups: scopes. See https://keel.sh/docs/#configuring-slack
```

## Testing
- All checks re-run 2026-08-19 on the branch after rebasing onto the current master (`5e64f9f6`); the single fix commit is now `781d22fd` (previously `785d0a2a`).
- `gofmt` clean on all touched files (`bot/slack/*`); pre-existing unformatted files elsewhere were left untouched.
- `CGO_ENABLED=1 go test ./...`: full suite passes (all 32 test packages, exit 0; the `tests` e2e package self-skips outside `make e2e`), including the three new `TestAugmentSlackAuthError*` tests in `bot/slack` (missing_scope payload → concrete scopes in the message, invalid_auth payload → token hint, non-auth/nil errors pass through unchanged).
- `make release-validate` (k3s v1.35.6 harness with Keel 0.22.1 packaged from this branch): Keel deployment rolls out, packaged chart install/upgrade/rollback and cleanup succeed, and the full testify e2e suite passes (all 8 scenarios, ~381s, under the 15m timeout).
