# Requirements: Security Review and Remediation of Keel API Surfaces

## Background

Keel (keel-hq/keel) exposes its entire external API over HTTP (gorilla/mux on `:9300`, `pkg/http/http.go`). A planning-time reconnaissance pass produced the following candidate findings, which this task formalizes into a written, severity-rated report checked into `docs/security/` and remediates high/critical items with tests.

Important scope correction from recon: **there are no gRPC servers in this repo** (no `grpc.NewServer` anywhere; gRPC appears only as a client dependency for GCP Pub/Sub in `trigger/pubsub/`). The "gRPC APIs" audit item is satisfied by documenting this fact in the findings report. There is also no golangci-lint configured; validation uses `go vet` + `go test` and any linter added must not be a rabbit hole.

## User Stories

- US-1: As a cluster operator, I want webhooks (GitHub, GitLab, Quay, Docker Hub, registry notifications, etc.) to verify request authenticity via HMAC signatures, so that attackers cannot forge "image updated" events that mutate my workloads.
- US-2: As a cluster operator, I want replayed webhook deliveries to be rejected within a time window, so captured payloads cannot be re-sent.
- US-3: As an admin, I want the CORS policy to not reflect `*` with an exposed `Authorization` header, so a malicious website cannot read issued JWTs from my Keel UI backend.
- US-4: As an admin, I want the JWT signing secret to never be derived from a weak PRNG and never printed in logs, so tokens cannot be forged offline or leaked via log pipelines.
- US-5: As an operator, I want HTTP server timeouts and request body size limits, so slow-loris and oversized-payload DoS are mitigated.
- US-6: As a security reviewer, I want a severity-rated findings report in the repo listing every audited surface and its status, including unfixed lower-severity items, so the review is auditable follow-up work.
- US-7: As a deploying user, I want the chart/deployment RBAC and defaults documented and tightened where feasible without breaking the provider flows, so Keel runs with least privilege.

## Acceptance Criteria

- AC-1 (`docs/security/SECURITY-REVIEW-2026-09.md`): report exists, rates each finding Critical/High/Medium/Low/Info, lists explicitly the no-gRPC-server conclusion, and lists follow-up items filed as separate spec tasks.
- AC-2 (Webhook signature verification): GitHub webhooks verify `X-Hub-Signature-256` (HMAC-SHA256, constant-time compare) when a secret is configured; GitHub App webhook secrets and registry-notification endpoints support a shared secret/HMAC or token check. When verification is not configured, behavior stays backward compatible (existing `AUTHENTICATED_WEBHOOKS` gating unchanged) but a startup WARN log is emitted. Verification happens **before** any payload triggers `s.trigger(event)` / provider submission.
- AC-3 (Replay protection): signed-webhook requests whose timestamp is older than a configurable window (default 5 minutes) are rejected with 401/408 where the payload carries a usable timestamp (GitHub `X-Hub-Signature-256` + `X-GitHub-Delivery` dedup acceptable per provider capability).
- AC-4 (CORS): `Access-Control-Expose-Headers: Authorization` is removed; `Access-Control-Allow-Origin: *` is kept only if it does not combine with credentials/exposed auth header — decided in design, tested.
- AC-5 (Token secret): the fallback `TOKEN_SECRET` derivation no longer uses `math/rand` (`pkg/auth/auth.go`); a cryptographically random secret is used. The error at `auth.go:118` must not embed `a.secret`; malformed-token warning at `auth.go:142` must not log the raw token. Tests assert redaction (pattern exists: `pkg/http/webhook_log_redaction_test.go`).
- AC-6 (Hardening): the HTTP server sets non-zero ReadHeaderTimeout, ReadTimeout, WriteTimeout, IdleTimeout, and webhook body reads are capped (`http.MaxBytesReader`), with no breaking change to legitimate payloads at documented sizes.
- AC-7 (RBAC): chart `values.yaml` / manifests keep default behavior (no breakage) but ship a documented least-privilege variant (namespace-scoped, restricted verbs) and prose in `docs/security/` explaining what `secrets` get/list/watch cluster-wide is needed for and how to narrow it.
- AC-8 (Backward compatibility): no removed/renamed endpoints, no changed request/response schemas, defaults keep existing deployments working. `go vet ./...`, `go build ./...`, and `go test ./pkg/... ./trigger/...` (and any touched packages) pass. UI untouched unless AC-4 requires it; UI code unchanged by this task, so no UI build required.
- AC-9 (Follow-ups filed): findings not fixed here are created as separate backlog spec tasks and linked from the report.

## Out of Scope

- Encrypting the sqlite store at rest (filed as follow-up; requires driver/storage decision).
- Changing the default `AUTHENTICATED_WEBHOOKS=false`, the LoadBalancer default in `deployment/`, or any admin API contract (breaking).
- Adding golangci-lint to CI (repo has none today).

## Open Questions

- GitHub webhook secrets: does Keel already receive per-endpoint secrets from anywhere (e.g., tracked-trigger metadata), or is a single global `GITHUB_WEBHOOK_SECRET` env var acceptable as the configuration surface? (Spec assumes global env var.)
- Is `Access-Control-Allow-Origin: *` relied upon by any known external integration, or is it safe to make CORS configurable? (Spec assumes keep `*` but strip `Expose-Headers: Authorization` as the minimal non-breaking fix.)
- AC-3 replay protection: GitHub's delivery-ID dedup requires a small in-memory cache; is an in-memory dedup window acceptable (lost on restart), or do you want store-backed dedup? (Spec assumes in-memory.)
