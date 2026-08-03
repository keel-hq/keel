# Implementation Tasks: Document Keel HTTP API and Generate JavaScript Client

- [ ] Add the checked route/handler/model inventory and explicit exclusions fixture for all 25 in-scope operations.
- [ ] Add documentation-only DTOs for private/anonymous request and response shapes, including every provider webhook and both GitHub payload variants.
- [ ] Add API metadata, Basic/Bearer definitions, stable operation IDs, and complete swag annotations to all in-scope handlers without changing runtime behavior.
- [ ] Add a tooling Go module pinning `github.com/swaggo/swag v1.16.6` and Make targets that invoke it without global installation.
- [ ] Generate and commit deterministic `docs/api/swagger.json` and `docs/api/swagger.yaml`; omit `docs.go` and mark artifact ownership.
- [ ] Add the Gorilla Mux route-to-operation coverage test for 25 operations, conditional routes, and every recorded exclusion.
- [ ] Pin OpenAPI Generator npm wrapper `2.20.2`, engine `7.22.0`, JavaScript options, and exact Yarn runtime dependencies.
- [ ] Generate and commit Promise-based ES-module client sources under `ui/src/api/generated/` with stable names and configurable Basic/Bearer/header behavior.
- [ ] Add Jest smoke-import coverage for generated APIs/models, base URL, auth, and headers; keep existing UI lint/build checks green.
- [ ] Document clean-checkout prerequisites and exact spec/client generation, validation, coverage, and drift-check commands.
- [ ] Add pinned spec validation and deterministic spec/client regeneration checks to CI.
- [ ] Run focused `pkg/http` tests, `go test ./...`, both-format validation, route coverage, client smoke tests, UI lint/build, and regeneration followed by `git diff --exit-code`; record any honest environment limitation.
