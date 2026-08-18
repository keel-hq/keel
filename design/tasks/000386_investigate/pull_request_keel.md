# fix(poll): force policy must not downgrade to older tags (fixes #823)

## Summary
With `keel.sh/policy: force` (without `keel.sh/matchTag: "true"`), the poll trigger's tag watcher picked the first tag in the registry tag list that was not the current tag. Registries return tag lists in creation order (oldest first), `ForcePolicy.Filter` did not sort them, and `ForcePolicy.ShouldUpdate` accepted every tag without any version comparison, so the oldest tag in the repository won: `cp-kafka:7.9.1-1-ubi8` was "updated" to `cp-kafka:3.0.0` (issue #823) and, more recently, `8.1.0` to `8.0.0`.

Changes:
- `ForcePolicy.ShouldUpdate`: when both tags parse as semantic versions, only update to a strictly newer tag (a re-push of the same tag is still an update; non-semver tags keep the previous behavior, since `force` exists for tags like `latest`).
- `ForcePolicy.Filter`: order candidates newest first, like the other policies do — semver tags descending, non-semver tags keeping their relative order and trailing the versioned tags — so the watcher picks the latest available tag in one hop.
- Regression tests for the policy (`internal/policy/force_test.go`) and the tag watcher (`TestWatchAllTagsJobWithForcePolicy`, covering the cp-kafka and 8.1.0→8.0.0 scenarios plus the no-newer-tag case); fix a latent panic in the `testRunHelper` fixture when zero events are expected.
- Behavioral e2e coverage for the k3s suite (`tests/e2e_scenarios_test.go`, seeded by `.test/e2e-k3s.sh`): `force-latest` (workload on `1.0.0` converges on the newest tag `1.0.2`), `force-nodowngrade` (workload on `1.0.1` with an older `1.0.0` tag stays put, the 8.1.0→8.0.0 report), and `force-prerelease` (workload on `7.9.1-1-ubi8` with only older tags stays put, the original cp-kafka report).

## Testing
- `go build ./...` and full `go test ./...`: all 32 packages pass; `go vet` and `gofmt` clean.
- Verified the real `confluentinc/cp-kafka` registry tag ordering via `/v2/confluentinc/cp-kafka/tags/list` (`3.0.0` is tag #0 of 1519; `7.9.1-1-ubi8` is #1336) and confirmed the fixed selection: current `7.9.1-1-ubi8` → `8.3.1` (never `3.0.0`); current `8.1.0` → `8.2.0` (never `8.0.0`); only older tags available → no event emitted.
- `make e2e` (task-owned k3s v1.35.6 + in-cluster registry, the same harness CI runs): full suite passes, including the three new force-policy scenarios `TestPollingForcePolicyUpdatesToNewestTag`, `TestPollingForcePolicyDoesNotDowngrade`, and `TestPollingForcePolicyIgnoresOlderTagsForPrereleaseCurrent`, alongside all pre-existing webhook/polling/OAuth scenarios (438s total, under the 15m CI timeout).
- `make release-validate` packaged-release scenario is not applicable: the change is version-comparison and test-harness logic, not application startup or the Helm chart, which is the condition AGENTS.md uses to require that validation.
