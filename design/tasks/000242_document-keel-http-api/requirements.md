# Requirements: Document Keel HTTP API and Generate JavaScript Client

## Outcome

Use `github.com/swaggo/swag` annotations as the source of a validated Swagger 2.0 description for Keel's HTTP API, then reproducibly generate a JavaScript client for the existing Vue/Babel environment and a future frontend. API-spec generation and JavaScript SDK generation are both in scope. Migrating existing UI calls or redesigning the frontend is not.

## User Stories

- As a contributor, I can regenerate and validate the API description and JavaScript client from a clean checkout with pinned repository commands.
- As a frontend developer, I can import stable, meaningful client operations whose payloads and authentication match Keel's server contract.
- As a reviewer, I can trace every registered Keel route to a documented operation or an explicit, justified exclusion.

## Functional Requirements

- Document exactly 25 in-scope non-preflight operations across 21 paths: `GET /healthz`; `GET /version`; auth-enabled login, info/user aliases, GET and POST logout, and refresh; all methods on approvals/resources/policies/tracked/audit/stats; and the eight native, Docker Hub, JFrog, Quay, Azure, GitHub, Harbor, and registry webhook POST routes.
- Give each operation a unique stable `operationId`, summary, description, tag, method/path, availability condition, security, parameters and headers, source-verified request schema, success status/schema, and meaningful error responses.
- Reuse actual domain types where they match the JSON wire format. Add documentation-only named DTOs for private or anonymous handler models, including provider-specific webhook bodies and both GitHub event shapes, without changing handler decoding or public semantics.
- Define HTTP Basic and `Authorization: Bearer <token>` schemes. Auth-enabled admin routes require either scheme. Seven provider webhooks are protected only when authenticated webhooks are configured; the registry-notification webhook remains public exactly as the router currently registers it. Keel does not use cookie authentication.
- Record route availability: admin/auth routes and static UI routes exist only when the authenticator is enabled, the seven configurable webhook wrappers depend on `AuthenticatedWebhooks`, and diagnostics depend on `DEBUG=true`.
- Exclude `OPTIONS`/CORS preflight operations, `/metrics`, static UI assets and SPA catch-all routes, and DEBUG-only expvar/pprof routes from the client API. Keep these exclusions in checked coverage evidence with a rationale.
- Pin Swag and every spec-validation/client-generation tool to explicit versions compatible with Go 1.23 and the repository's Node 16/Yarn 1 UI toolchain. Committed workflows must contain no floating version selector and must not depend on a global Go binary or an assumed `GOPATH/bin` entry.
- Commit deterministic Swagger JSON and YAML artifacts in a clearly owned location. Do not generate or import `docs.go`: Keel will not serve a Swagger UI or runtime documentation package.
- Generate and commit a Promise-based ES-module JavaScript client under the existing UI source tree. It must provide a configurable base URL, all 25 operation-derived methods, Basic/Bearer and arbitrary-header support, response-header access, accurate request/response models, and no invented cookie behavior.
- Provide repository targets for spec generation, client generation, combined generation, pinned validation, route-to-operation coverage, client smoke verification, and an aggregate regeneration/drift check.
- Add contributor documentation for prerequisites and exact clean-checkout commands. Generated files must be marked as generated/owned, must not be hand-edited, and regeneration must end with no Git diff.

## Compatibility and Safety

- Do not change route behavior, availability, authentication, CORS, payloads, response status codes, deployment behavior, or existing frontend behavior.
- Do not expose secrets, embed credentials, hard-code a deployment host or scheme, or add a public Swagger/OpenAPI endpoint or UI.
- Preserve unrelated changes and keep the work limited to documentation annotations, documentation-only schemas, generation tooling/artifacts, validation/tests, and contributor instructions.

## Acceptance Criteria

- A clean checkout with documented Go 1.23, Node 16, Yarn 1, and Java prerequisites can regenerate Swagger JSON/YAML and the JavaScript client without global tool installation or manual edits.
- Pinned validation succeeds for both spec formats, and a checked route report proves all 25 in-scope method/path operations and all deliberate exclusions are accounted for.
- Every generated client operation has the specified stable name and matching body, query/header parameters, response type, and security behavior.
- A UI-context smoke test imports the generated client and checks representative model serialization, configurable base URL, Basic/Bearer/custom headers, and full-response/header access. Relevant existing UI lint and production-build commands pass.
- Focused `pkg/http` tests and `go test ./...` pass; any real environment-specific limitation is reported rather than bypassed.
- Running combined spec/client generation and then `git diff --exit-code` succeeds, including generated artifacts, manifests, and lockfiles.
- No runtime API, deployment, authentication, or existing frontend behavior changes, and no public documentation endpoint is added.

## Open Questions

None.
