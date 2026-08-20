# test(bot/hipchat): make TestBotAproval deterministic

## Summary
The `Unit Tests` CI job (and master) failed intermittently with
`panic: runtime error: index out of range [2] with length 2` at
`bot/hipchat/hipchat_test.go:211` in `TestBotAproval`. This happened even on
PRs unrelated to hipchat (e.g. the event-buffer change in keel-hq/keel#905)
because the test's behaviour was timing-dependent, not because of the change.

Root cause: the bot tests posted responses from goroutines
(`ProcessBotMessages` / `ProcessApprovalResponses`), but the test used fixed
`time.Sleep(1 * time.Second)` waits and then indexed `postedMessages[i]`
directly. When a slow runner delivered a message in more than 1s, the length
check failed via `t.Errorf` (non-fatal) but execution continued, and the
subsequent `postedMessages[2]` index panicked. `TestBotReject` shared the
identical flaw. A secondary data race existed because `Say` appended to
`postedMessages` (from a goroutine) while the test read it directly.

Fix (assertions unchanged in strength, only the wait mechanism is stronger):
- Replaced every fixed `time.Sleep(1s)` in `TestHelpCommand`, `TestBotAproval`,
  and `TestBotReject` with a `waitFor(t, 5*time.Second, cond)` helper that polls
  until the expected number of messages is present, failing (not panicking) on
  timeout.
- Added a `sync.Mutex` to `fakeXmppImplementer`; `Say` now locks on append and
  a new `snapshot()` method returns a locked, defensive copy so the test never
  reads `postedMessages` concurrently with a goroutine appending to it.
- Kept all message-content assertions exactly as before (`Approval required!`,
  `Update approved!`, `Change rejected`, the `1/1 false` / `0/1 true` approval
  text, the greeting and help message prefixes).

This unblocks CI for keel-hq/keel#443, whose Unit Tests job was failing purely
due to this flaky timing.

## Testing
- `go vet ./bot/hipchat/` and `gofmt` — clean.
- `go test ./bot/hipchat/ -race -count=5` — PASS (TestHelpCommand,
  TestBotAproval, TestBotReject, TestApprovalsManager, TestConfigure* all
  pass repeatedly, including under the race detector).
- `go test ./...` — all packages PASS, no FAIL or panic.
