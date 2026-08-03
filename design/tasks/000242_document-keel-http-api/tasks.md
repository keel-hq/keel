# Implementation Tasks: Document Keel HTTP API and Generate JavaScript Client

- [ ] Add pinned `swag v1.16.6` tooling, API metadata, Make targets, and contributor prerequisites.
- [ ] Define reusable documentation DTOs and annotate all health, version, auth, admin, and webhook handlers with stable operation IDs and accurate contracts.
- [ ] Generate and commit deterministic Swagger JSON, YAML, and Go metadata under `docs/api/` with generated-file guidance.
- [ ] Add the explicit route inventory/exclusion manifest and a Gorilla Mux route-to-operation coverage test for the 25 documented operations.
- [ ] Pin OpenAPI Generator CLI `2.20.2` and engine `7.22.0`, plus deterministic JavaScript generator configuration.
- [ ] Generate and commit browser ES-module client code under `ui/src/api/generated` and lock required UI runtime dependencies.
- [ ] Add a focused UI client smoke test covering exports, models, base URL, and Basic/Bearer configuration without migrating existing screens.
- [ ] Add Swagger validation and generated-artifact drift checks to repository verification/CI.
- [ ] Run `go test ./...`, focused HTTP tests, route coverage, Swagger validation, UI lint/build/client checks, and generation followed by `git diff --exit-code`; document any environment limitation.
