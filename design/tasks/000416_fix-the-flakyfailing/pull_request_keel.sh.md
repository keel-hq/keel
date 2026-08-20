# No changes in keel.sh

## Summary
No changes were required in keel.sh for this task. The flaky unit test that
panicked on CI (`index out of range [2] with length 2` at
`bot/hipchat/hipchat_test.go:211`, `TestBotAproval`) lives entirely in the
`keel` repository and is fixed there. The keel.sh Helm chart is unaffected.

## Testing
No changes to verify in keel.sh.
