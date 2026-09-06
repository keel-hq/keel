# Design: Security Review and Remediation of Keel API Surfaces

## Architecture Context (learned during planning)

- All external API surface is HTTP via negroni + gorilla/mux (`pkg/http/http.go:65,104-121`), port 9300. **No gRPC servers exist** — `provider/`, `trigger/`, `bot/` are internal Go packages, not RPC servers; gRPC is only a client dep for GCP Pub/Sub (`trigger/pubsub/pubsub.go`). The report documents this to close audit item 1.
- Route registration: `pkg/http/http.go:144-237`. Always-public: `/healthz`, `/version`, `/metrics`, `/v1/webhooks/registry` (never gated, line ~224). Other webhooks gated only when `AUTHENTICATED_WEBHOOKS=true` (default false, `pkg/config/config.go:164`).
- Trigger flow to avoid re-auditing: handler → validation → `s.trigger(event)` → `provider.Providers.Submit` (`provider/provider.go:81`) → helm3 (`provider/helm3/implementer.go:134`, uses `BearerToken: &i.KubeToken`) or kubernetes patch (`provider/kubernetes/kubernetes.go:184`). Injection risk is low: image strings go through typed k8s update plans and Helm release names, not shell/template string concat — verify and note in report.
- Auth: single-admin JWT HS512, 12h expiry (`pkg/auth/auth.go`). Known weak spots confirmed at `auth.go:117` (secret in error), `auth.go:142` (raw token in log), `math/rand` fallback secret (`auth.go:43-46,171-181`).
- Existing test anchors to imitate: `pkg/http/webhook_log_redaction_test.go` (log-redaction pattern via hook), `pkg/http/api_inventory_test.go` (route inventory — update if routes change), `pkg/http/external_auth_proxy_test.go`.
- No golangci-lint config exists; validation = `go vet` + targeted `go test`.

## Findings → Fixes (key decisions)

### F-1 Critical: unauthenticated webhooks forge deployments; no HMAC anywhere
Fix additively, preserving defaults (no breaking change):
- New helper `pkg/http/webhook_signature.go`: `verifyGitHubSignature(payload []byte, signatureHeader, secret string) error` using `crypto/hmac` + `sha256` + `hmac.Equal`, and a generic bearer-token-or-HMAC check for registry-notification endpoints.
- New config (env, defaults empty = feature off, backward compatible): `GITHUB_WEBHOOK_SECRET`, `REGISTRY_NOTIFICATIONS_TOKEN` (checked against `Authorization` header on `/v1/webhooks/registry` only when set). When `AUTHENTICATED_WEBHOOKS=false` AND no webhook secrets configured, log one startup WARN ("webhooks unauthenticated") — loud, not breaking.
- Wire into `github_webhook_trigger.go` before event dispatch (`:111-160` region) and `registry_notifications.go` at handler entry. Verification precedes any `s.trigger` call.

### F-2 High: replay of signed webhooks
- GitHub: dedup `X-GitHub-Delivery` IDs in a bounded in-memory LRU/TTL cache (~5 min, `container/list` map, no new dep) when `GITHUB_WEBHOOK_SECRET` is set. Registry notifications already carry per-event IDs if the token check is on; keep dedup scoped to GitHub only — other providers lack usable nonces, documented as limitation.

### F-3 High: CORS exposes Authorization cross-origin
- Remove `Access-Control-Expose-Headers: Authorization` and the stray `Access-Control-Request-Headers` response header (`corsHeadersMiddleware`, `pkg/http/http.go:320-335`). Keel's own UI runs same-origin so it never needed exposure. Keep `Allow-Origin: *` (non-breaking; no credentials mode in use, cookie-less bearer auth → CSRF not applicable; document reasoning).

### F-4 High: JWT secret weak-PRNG fallback + secret/token in logs
- Replace `math/rand` fallback secret with `crypto/rand` (32 bytes, base64). (Cross-restart token invalidation behavior is identical to today.)
- `GenerateToken` error: drop `string(a.secret)` from the message. `parseToken`: log token prefix (first 8 chars) or JWT `kid`-less ID-free message instead of raw token. Tests: unit test asserting generated secret length/entropy source is trivial; redaction tests follow `webhook_log_redaction_test.go` logrus-hook pattern.

### F-5 Medium: no server timeouts / body limits
- Set on `http.Server` in `ListenAndServe` (`pkg/http/http.go:112`): ReadHeaderTimeout 10s, ReadTimeout 30s, WriteTimeout 30s, IdleTimeout 60s. Wrap webhook handler bodies with `http.MaxBytesReader` (4 MB) via a small negroni middleware applied to `/v1/webhooks/` routes only — generous vs real payloads (Docker notifications are KBs).

### F-6 Medium: RBAC breadth (cluster-wide `secrets` read, workload `delete`)
- No default changes (would break pull-secret resolution; `secrets/secrets.go:127-171` reads dockerconfigjson across namespaces by design). Ship a documented least-privilege path: new `docs/security/rbac.md` explaining each rule and providing values snippets for namespace-scoped `roles:` and restricting delete verbs when only helm provider is used. Chart `values.yaml` comments updated to reference it.

### F-7 Low/Medium: plaintext sqlite store, LB default, `/metrics` public, basic-auth-only single admin, hardcoded admin claim
- All go into the findings report as accepted-risk/follow-up entries → filed as backlog spec tasks (store encryption, configurable CORS origins, multi-user auth, `deployment/` LB default).

## Report Layout

`docs/security/SECURITY-REVIEW-2026-09.md`: scope & method, per-surface findings table (ID, severity, location, status fixed/follow-up/accepted), gRPC no-server conclusion, follow-up task links. `docs/security/rbac.md` for F-6.

## Testing Strategy

- Unit: signature verify helper (valid/invalid/missing/oversized), replay cache, secret fallback (crypto/rand), CORS header assertions via existing server test harness, redaction tests, timeout fields present.
- Regression: `go vet ./...`; `go test ./pkg/http/... ./pkg/auth/... ./pkg/config/...`; `go build ./...`. `api_inventory_test.go` must still pass (no route changes). No UI changes → no UI lint/build.
