# Implementation Tasks: Document Keel HTTP API and Generate JavaScript Client

- [ ] Add a checked router inventory for all 25 in-scope method/path operations, their enabling configurations, stable operation IDs, and every excluded route surface.
- [ ] Add named documentation-only DTOs for handler-local admin payloads/responses and all provider-specific webhook payloads, including both GitHub event variants, without changing runtime models.
- [ ] Add host-neutral global Swagger metadata and Basic/Bearer security definitions without a runtime docs import, generated `docs.go`, public spec route, or Swagger UI.
- [ ] Annotate health, version, auth, admin, and webhook handlers with the exact operation IDs, parameters, request bodies, response headers, success/error contracts, tags, availability, and conditional security in the design inventory.
- [ ] Add an isolated Go tooling module pinning Swag `v1.16.6` and invoke it through repository Make commands without a global binary or PATH assumption.
- [ ] Configure deterministic spec generation and commit only owned `docs/api/swagger.json` and `docs/api/swagger.yaml` with generated-file guidance.
- [ ] Add pinned two-format spec validation and normalized JSON/YAML equivalence checks using OpenAPI Generator engine `7.7.0`.
- [ ] Implement Gorilla Mux route-to-operation coverage across authenticator, authenticated-webhook, and DEBUG configurations; require exactly 25 operations/21 paths and justified exclusions.
- [ ] Pin `@openapitools/openapi-generator-cli` `2.13.4`, engine `7.7.0`, `superagent` `8.1.2`, generator configuration, and corresponding Yarn lockfile entries.
- [ ] Generate and commit Promise-based ES-module JavaScript APIs/models/runtime under `ui/src/api/generated/`, with stable operation names, configurable base URL, Basic/Bearer/custom-header support, and no cookie defaults.
- [ ] Add a Jest smoke test that imports generated barrels/API groups/models and verifies serialization, base URL, Basic/Bearer and `X-GitHub-Event` headers, and full-response/header access.
- [ ] Preserve handwritten UI API calls, exclude generated code only from handwritten lint formatting, and run the relevant existing UI lint and production build checks.
- [ ] Add contributor documentation for exact prerequisites and clean-checkout spec generation, client generation, validation, route coverage, client checks, and deterministic drift checking.
- [ ] Add CI checks for pinned generation/validation, route coverage, client smoke verification, UI lint/build, and generation followed by `git diff --exit-code`.
- [ ] Run focused `pkg/http` tests and `go test ./...`, documenting any genuine environment-specific limitation without suppressing it.
- [ ] Run the aggregate `make api-check` workflow and confirm regeneration leaves the complete working tree clean before handoff.
