# Repository Instructions

## Validation

- For changes that affect application startup or the Helm chart, always run `make release-validate` before considering the work complete. This must exercise the repository's local k3s harness with Keel deployed; unit tests or rendered-manifest checks alone are not sufficient.
- Confirm that the Keel Deployment rolls out, its health probes succeed, and the k3s end-to-end suite passes. If the environment prevents this validation, report the exact blocker and do not claim that validation passed.

## Skills

Agent skills live in `.agents/skills/<name>/SKILL.md` and are readable by opencode, codex, and other AGENTS-compatible tools.

- `keel-dev-stack` (`.agents/skills/keel-dev-stack/SKILL.md`): Start, verify, and tear down the local Keel dev stack — k3s (k3d) cluster on :6443, the Keel backend/dashboard on :9300, and the React UI (built `ui/dist` served by Keel, plus the Vite dev server on :8000). Use when asked to start the dev server, bring up k3s/k3d with Keel, or fix the dashboard not opening.
- `release-keel` (`.agents/skills/release-keel/SKILL.md`): Prepare, validate, publish, verify, and recover Keel application and Helm chart releases. Use when asked to release Keel, bump application or chart versions, create release tags, inspect release readiness, or recover a failed release.
