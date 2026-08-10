# chore(helm): remove dead Helm 2 config knobs

## Summary

Investigated whether upstream PR keel-hq/keel#547 ("Feature/helm v3 overall upgrade") is still needed. It is not: its substance already landed in master. `provider/helm3/` exists and has been actively maintained since (io/ioutil cleanup #778, SemVer TagsWatcher fix #812), `provider/helm/` (Helm 2) is gone, and `go.mod` is on `helm.sh/helm/v3 v3.16.3`. The PR branch is not an ancestor of HEAD and diffing it against HEAD is 476 files / +53k / -36k — it would revert the entire modern Vite/TS UI back to the old vue-cli/yarn one. It should simply be closed, not rebased.

What the sweep did miss were the Tiller-era configuration knobs, which this change removes. The only Helm enablement switch the binary reads is `HELM3_PROVIDER` (`pkg/config/config.go`); `HELM_PROVIDER` and `TILLER_NAMESPACE` are read nowhere in Go.

- `chart/keel/values.yaml` — dropped `version`, `tillerNamespace`, `tillerAddress`; only `enabled` and the two `helmDriver*` knobs remain.
- `chart/keel/templates/deployment.yaml` — dropped the `eq .Values.helmProvider.version "v3"` gate, so `HELM3_PROVIDER` keys off `helmProvider.enabled` alone; driver blocks re-indented one level.
- `deployment/deployment-template.yaml` — replaced the dead `HELM_PROVIDER` / `TILLER_NAMESPACE` block with `HELM3_PROVIDER`. This also fixes `{{ if.tiller_namespace }}`, which was malformed and would not have parsed as a Go template action. No consumer of this file exists in `keel` or `keel.sh`.
- `values.yaml` (root) and `readme.md` — dropped the `version: "v3"` / `--set helmProvider.version="v3"` instructions.

`chart/keel/README.md` already documented only `enabled` / `helmDriver` / `helmDriverSqlConnectionString`, so it needed no edit.

**Behavior change to be aware of on upgrade:** anyone still passing `helmProvider.version="v2"`, or sitting on the old chart default of `v2`, previously got no Helm provider at all — the binary has shipped no Helm 2 code for years, so they were silently broken. After this change they get the Helm 3 provider. That is the intended fix, but it is a live change rather than a pure no-op.

`chart/keel/Chart.yaml` is deliberately untouched: it notes the version is a template replaced at chart-build time and released via a `chart-{VERSION}` tag, so the bump is a release-time decision.

## Testing

No Go code changed, so this is a chart/templating change only. `helm` is not installed in this environment, so the chart was rendered through the Helm v3 SDK already present in `go.mod` (`loader.Load` + `chartutil.ToRenderValues` + `engine.Render`) and the resulting `deployment.yaml` inspected:

| Case | Result |
| --- | --- |
| defaults | `HELM3_PROVIDER` present |
| `helmProvider.enabled=false` | `HELM3_PROVIDER` absent |
| `helmProvider.version=v2` (stale user value) | `HELM3_PROVIDER` present — confirms the documented behavior change |
| `helmDriver` + `helmDriverSqlConnectionString` set | `HELM_DRIVER` and `HELM_DRIVER_SQL_CONNECTION_STRING` both present |

All four rendered without a template error, confirming the re-indented `{{- if }}` nesting is balanced. Passing a now-removed value such as `helmProvider.version` is ignored by Helm rather than being an error, so existing user values files will not break on upgrade.

Verified by grep that no Go source reads `HELM_PROVIDER`, `TILLER_NAMESPACE`, or `TILLER_ADDRESS`, and that no file in `keel` or `keel.sh` consumes `deployment/deployment-template.yaml`.

CI is green on this PR. Note that CI cannot exercise the upgrade-path behavior change described above — no automated check covers a user whose existing values file pins `helmProvider.version="v2"`, so that case is called out for reviewer judgement rather than being test-covered.
