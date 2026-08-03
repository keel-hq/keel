# Requirements: Document Keel HTTP API and Generate JavaScript Client

## User Stories

- As a Keel contributor, I can regenerate and validate the Swagger description from handler annotations on a clean checkout with pinned tools.
- As a future frontend developer, I can import a generated browser JavaScript client with stable operations, accurate models, and Keel-compatible authentication.
- As a reviewer, I can audit every registered route as either one of the 25 documented operations or a justified exclusion.

## Functional Requirements

- Annotate all 21 in-scope paths and 25 non-OPTIONS operations registered by `pkg/http/http.go`, including both auth aliases, both logout methods, all admin methods, and all eight provider-specific webhooks.
- Each operation must define a unique explicit `operationId`, method/path, tag, summary/description, enabling condition, security, parameters/headers, real request schema, success schema/status, and meaningful error responses verified from its handler.
- Reuse existing domain models where their JSON is the wire contract. Add documentation-only named wrappers for handler-local/anonymous payloads, including each provider webhook and the two GitHub event shapes, without changing decoding or runtime behavior.
- Model HTTP Basic and `Authorization: Bearer` as alternative admin authentication schemes. Login is public when auth routes are enabled; other admin routes require authorization. Seven webhooks are optionally protected by `AuthenticatedWebhooks`; `/v1/webhooks/registry` remains public exactly as currently registered. Keel has no cookie-auth requirement.
- Exclude OPTIONS/CORS, `/metrics`, static UI/assets/index catch-alls, and DEBUG-only expvar/pprof routes. Record each exclusion and rationale in machine-checked coverage evidence.
- Client generation is in scope; migrating existing UI calls or redesigning/replacing the frontend is not.
- Do not add a Swagger UI or public docs endpoint, hard-code a deployment host, expose secrets, or change routes, payloads, status codes, authentication, CORS, deployment, or current frontend behavior.

## Generation Requirements

- Pin `github.com/swaggo/swag` at Go-1.23-compatible `v1.16.6` in a tooling module and invoke it through `go run` from Make; never use `@latest`, a global install, or an assumed `GOPATH/bin`.
- Commit deterministic `docs/api/swagger.json` and `docs/api/swagger.yaml`. Do not generate or commit `docs.go`: Keel does not serve Swagger at runtime, so an importable generated Go package has no justified use.
- Pin `@openapitools/openapi-generator-cli` at Node-16-compatible `2.20.2`, its Java generator engine at `7.22.0`, and generated runtime dependencies in `ui/yarn.lock`.
- Generate and commit Promise-based browser JavaScript ES modules under `ui/src/api/generated/`. The client must remain configurable for the deployment base URL and support Basic auth, Bearer headers, ordinary request headers, response headers, and the actual request/response models; it must not invent cookie behavior.
- Generated operation names come only from explicit handler `operationId` values and remain stable across regeneration. Generated files are owned by the generation command, marked as generated where the format permits, and never hand-edited.
- Document clean-checkout prerequisites and Make targets for spec generation, client generation, validation, route coverage, client smoke testing, and aggregate drift checking.

## Acceptance Criteria

- `make api-spec`, `make api-client`, `make api-generate`, `make api-validate`, `make api-coverage`, and `make api-check` are reproducible from a clean checkout after documented dependency installation.
- Pinned validation accepts both committed spec formats and a route-to-operation test accounts for exactly 25 operations plus the explicit exclusions.
- The generated client exposes callable, meaningful methods for every documented operation and accurately represents Basic/Bearer configuration, `Authorization` response headers, the GitHub event header, audit query parameters, and typed provider payloads.
- A focused Jest smoke test imports generated APIs/models and exercises base URL plus authentication/header configuration; existing UI lint and build checks remain green. Generated code may be style-lint-excluded, but it must be parsed/imported by the smoke test.
- Focused `pkg/http` tests and `go test ./...` pass. Any environment-dependent integration limitation is reported explicitly rather than skipped silently.
- Running spec and client regeneration followed by `git diff --exit-code` produces no diff, including specs, client output, configuration, and lockfiles.
- No runtime API, auth, deployment, or existing frontend behavior changes.

## Open Questions

None.
