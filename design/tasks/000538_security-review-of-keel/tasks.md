# Implementation Tasks: Security Review and Remediation of Keel API Surfaces

- [ ] Create `docs/security/SECURITY-REVIEW-2026-09.md` skeleton: scope, method, findings table (ID, severity, location file:line, status)
- [ ] Audit pass and fill report: confirm no gRPC servers exist (search `grpc.NewServer`), document provider/trigger/bot as internal packages, closing gRPC TLS/message-size items as N/A
- [ ] Audit injection surface: trace webhook image strings through `provider/kubernetes` update plans and `provider/helm3` release/upgrade calls; document findings (no code change expected)
- [ ] Fill report findings for all recon items: unauthenticated webhooks, missing HMAC, replay, CORS `Expose-Headers: Authorization`, JWT math/rand secret, secret/token log leakage, missing timeouts/body limits, RBAC breadth, plaintext sqlite store, LoadBalancer default, public `/metrics`, `/v1/webhooks/registry` never gated
- [ ] F-4: replace `math/rand` TOKEN_SECRET fallback with `crypto/rand` in `pkg/auth/auth.go` (~lines 43-46, 171-181)
- [ ] F-4: remove secret from `GenerateToken` error (`auth.go:117`) and raw token from `parseToken` warn log (`auth.go:142`); add redaction unit tests (pattern: `pkg/http/webhook_log_redaction_test.go`)
- [ ] F-3: remove `Access-Control-Expose-Headers: Authorization` and `Access-Control-Request-Headers` from `corsHeadersMiddleware` (`pkg/http/http.go:320-335`); add CORS header test; verify UI login flow unaffected (same-origin)
- [ ] F-1: add `pkg/http/webhook_signature.go` with constant-time HMAC-SHA256 verifier for `X-Hub-Signature-256` plus unit tests (valid/invalid/missing/malformed)
- [ ] F-1: add config `GITHUB_WEBHOOK_SECRET` and `REGISTRY_NOTIFICATIONS_TOKEN` in `pkg/config/config.go` (defaults empty = off) with startup WARN when all webhook auth is off
- [ ] F-1: wire GitHub signature verification into `pkg/http/github_webhook_trigger.go` before dispatch; registry-notifications bearer check in `registry_notifications.go`; handler tests for 401 paths
- [ ] F-2: implement bounded in-memory delivery-ID replay cache (~5 min TTL, stdlib only); wire into GitHub handler when secret configured; unit tests (fresh id accepted, repeat rejected, expiry)
- [ ] F-5: set ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout on `http.Server` in `pkg/http/http.go` ListenAndServe; add `http.MaxBytesReader` (4 MB) middleware on `/v1/webhooks/` routes; test oversized body → error
- [ ] F-6: write `docs/security/rbac.md` (per-rule justification for clusterrole.yaml, namespace-scoped least-privilege values snippets); add comments in `chart/keel/values.yaml` referencing it; keep default rules unchanged
- [ ] File follow-up backlog spec tasks (store at-rest encryption, configurable CORS origins, multi-user auth, deployment LB default) and link them in the report
- [ ] Final validation: `go vet ./...`, `go build ./...`, `go test ./pkg/http/... ./pkg/auth/... ./pkg/config/...`; confirm `api_inventory_test.go` passes (no route changes); update report status column
- [ ] Commit on focused branch `security-review` per repo conventions; PR with report as the headline artifact
