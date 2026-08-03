# Requirements: Document Keel HTTP API and Generate JavaScript Client

## User Stories

- As a Keel contributor, I can regenerate an accurate Swagger description from handler annotations with pinned tools.
- As a future frontend developer, I can use a generated browser JavaScript client whose operations and models match Keel's current wire contract.
- As a reviewer, I can audit every registered route as documented or deliberately excluded.

## Acceptance Criteria

- `github.com/swaggo/swag` is pinned to the Go-1.23-compatible stable release `v1.16.6`, and a documented repository command generates committed Swagger JSON, YAML, and Go metadata without exposing a runtime documentation endpoint.
- All 25 Keel-owned, non-OPTIONS operations registered in `pkg/http/http.go` are documented: health and version (2); auth routes including both logout methods (6); approvals, resources, policies, tracked images, audit, and stats (9); and webhooks (8).
- Each operation has a unique, stable `operationId`, method/path, summary, description, tag, applicable Basic/Bearer security, parameters, body schema, success response, and meaningful error responses matching existing behavior.
- Reusable request, response, error, and webhook schemas describe the actual JSON and header wire formats. Documentation work does not change handlers, status codes, authentication, or public semantics.
- Webhook security reflects current configuration: seven provider webhooks may require Basic or Bearer authentication when authenticated webhooks are enabled; registry notifications remain unauthenticated as currently registered.
- OPTIONS preflight, `/metrics`, static UI/assets, and conditional debug/pprof handlers are excluded because they are middleware, third-party/standard-library, or non-client API surfaces. The coverage evidence records these exclusions and their rationale.
- OpenAPI Generator CLI `2.20.2` (compatible with the repository's Node 16 CI) and generator engine `7.22.0` are pinned. It generates committed browser JavaScript ES modules into `ui/src/api/generated`, with models and authentication support usable by the existing Vue 2/Babel toolchain.
- Contributor documentation lists Go 1.23, Node 16/Yarn, and Java 11+ prerequisites and exact commands to generate, validate, test, and check for drift.
- Automated checks validate the Swagger document, compare registered routes with documented operations plus the explicit exclusion list, exercise the generated client in the UI toolchain, and verify regeneration leaves no git diff.
- `go test ./...`, focused HTTP tests, the relevant UI lint/build or client smoke checks, Swagger validation, route coverage, and deterministic regeneration pass; environment-specific failures are reported rather than hidden.

## Open Questions

None.
