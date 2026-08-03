# Implementation Tasks: Document Keel HTTP API and Generate JavaScript Client

- [ ] Add a checked inventory fixture for the 25 method/path/operation-ID contracts and all excluded router surfaces.
- [ ] Add named documentation-only schemas for private/anonymous admin and provider webhook wire formats, including both GitHub event variants.
- [ ] Add global API metadata and Basic/Bearer security definitions without a host, runtime docs import, or public Swagger route.
- [ ] Annotate health, version, auth, admin, and webhook handlers with the exact operation IDs, parameters, bodies, responses, errors, tags, and conditional security in the design inventory.
- [ ] Add an isolated Go tooling module pinning Swag `v1.16.6` and Make invocation that does not depend on a globally installed binary.
- [ ] Add deterministic spec configuration and generate only committed `docs/api/swagger.json` and `docs/api/swagger.yaml` with ownership markers.
- [ ] Implement the Gorilla Mux route-to-operation test covering 25 operations, auth/DEBUG configurations, preflight filtering, and explicit exclusions.
- [ ] Pin OpenAPI Generator wrapper `2.20.2`, engine/validator `7.22.0`, JavaScript generator options, `superagent` `5.3.1`, and Yarn lock changes.
- [ ] Generate and commit Promise-based ES-module API/model sources in `ui/src/api/generated/` with stable operation names and configurable URL/auth/header behavior.
- [ ] Add Jest smoke tests for generated exports, representative models, all API groups, base URL, Basic/Bearer auth, custom request headers, and full response headers.
- [ ] Keep generated files out of handwritten style rewrites while running the relevant existing UI lint and build checks.
- [ ] Document prerequisites and exact clean-checkout generation, validation, coverage, client-test, and drift-check commands.
- [ ] Add pinned two-format spec validation, route coverage, client verification, and deterministic regeneration checks to CI.
- [ ] Run focused `pkg/http` tests and `go test ./...`; report any genuine environment-specific limitation.
- [ ] Run the aggregate generation/check target and confirm `git diff --exit-code` is clean before handoff.
