# Requirements: Document the Keel API and Generate a Reproducible JavaScript Client

## Goal

Make Keel's existing gorilla/mux HTTP API accurately discoverable as a generated Swagger 2.0 description, and generate a deterministic JavaScript client that a future frontend can use. Documentation and generated code must describe current behavior; they must not redefine it.

## User Stories

- As an API consumer, I can inspect every supported Keel operation, payload, response, status, security rule, and configuration condition.
- As a frontend contributor, I can import a generated JavaScript client that works with the existing Vue 2/Babel environment.
- As a maintainer, I can regenerate and verify the spec and client from a clean checkout with pinned tools and detect stale artifacts in CI.

## Acceptance Criteria

- The Swagger source is complete `swag` annotations grounded in the gorilla/mux router, handlers, domain models, tests, and `ui/package.json`.
- The generated specification contains exactly the 21 in-scope paths and 25 non-OPTIONS operations listed in the design inventory. Each operation has a unique stable operation ID, method/path, tag, summary/description, exact inputs, outputs/statuses, security, and enabling condition.
- Provider-specific webhook schemas match the native, Docker Hub, JFrog, Quay, Azure, GitHub Packages/GHCR, Harbor, and Docker Registry notification structs in source. Documentation-only schema wrappers may be used only when they preserve the current JSON wire shape exactly.
- The documentation records that authenticator-backed auth/admin routes exist only when the authenticator is enabled. It also records that `authenticatedWebhooks` conditionally protects native, Docker Hub, JFrog, Quay, Azure, GitHub, and Harbor webhooks with Basic or Bearer authorization, while the registry webhook is always registered and never wrapped by admin authorization.
- OPTIONS/CORS handling, `/metrics`, static UI/assets and the UI catch-all, and DEBUG-only `/debug/vars` and pprof routes are explicitly inventoried and excluded from the public application API specification.
- `github.com/swaggo/swag/cmd/swag` is pinned to v1.16.6 and invoked explicitly through a repository command; no `@latest`, PATH lookup, or install-if-missing workflow is committed.
- OpenAPI Generator's maintained `javascript` client generator is pinned to v7.22.0. The generated ES-module, Promise-based client is committed under `ui/src/api/generated`, is compatible with the current Vue 2/Babel toolchain, and uses exactly SuperAgent 10.3.0.
- The client uses stable operation IDs for method names, a same-origin default base URL with an explicit runtime override, disabled cookies/credentials by default, and configurable Basic and `Authorization: Bearer ...` authentication. Successful Promises expose both deserialized data and the raw response; failures expose status, body, response, and transport error.
- The client SDK is in scope and is generated from the committed Swagger file. It is not hand-maintained, published, deployed, or used to migrate the current frontend in this change.
- Only the canonical Swagger YAML and useful JavaScript SDK source are committed. Swagger JSON, `docs.go`, generator metadata, generated SDK docs/tests/package scaffolding, and other redundant outputs are not generated and are ignored. Generated artifacts have ownership/notices and are updated atomically with annotations and tool versions.
- Spectral CLI v6.16.1 validates/lints the generated Swagger document. Repository targets deterministically generate the spec and client, validate them, print the 25-operation router-to-spec mapping plus exclusions, and fail a generate-then-`git diff --exit-code` clean-diff check.
- Verification includes the pinned spec lint, client generation, a generated-client Jest smoke import/contract test, existing UI lint and build, focused `pkg/http` tests, and `go test ./...`. Environment-dependent failures are reported honestly rather than skipped or converted into success.
- Contributor documentation states prerequisites and exact generation, validation, smoke-test, UI, Go, and clean-diff commands.
- Backward compatibility is preserved: no route, payload, status code, authentication/security behavior, runtime/deployment behavior, or current frontend behavior changes. No secret is logged or embedded; no public Swagger UI/docs route or deployment-specific host is added; nothing is merged, published, or deployed.

## Open Questions

None.
