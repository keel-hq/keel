# Requirements: Document Keel HTTP API and Generate JavaScript Client

## Required Outcome

Generate a validated Swagger description from Keel's Go handlers and generate a reproducible browser JavaScript client from that description. Both spec generation and client generation are in scope. Replacing the frontend or migrating its existing API calls is not.

## User Stories

- As a contributor, I can use repository commands on a clean checkout to regenerate and validate the API description without global tools.
- As a future frontend developer, I can import a committed JavaScript client with stable methods, accurate models, and Keel-compatible authentication.
- As a reviewer, I can account for every Gorilla Mux route as a documented operation or a justified exclusion.

## Requirements

- Cover the 21 Keel API paths and 25 non-preflight operations registered in `pkg/http/http.go`: health, version, six auth operations, nine admin operations, and eight webhooks.
- Give every operation an explicit stable `operationId`, correct method/path, tag, enabling condition, security, parameters and headers, source-verified request/body schema, success schema/status, and meaningful error responses.
- Use existing JSON domain types when accurate. Add named documentation-only schemas for private or anonymous handler payloads, including all provider webhook formats and both GitHub package event variants, without changing runtime decoding.
- Define HTTP Basic and Bearer-token authorization as alternatives. Protected admin routes require authorization. Seven webhooks are protected only when authenticated webhooks are configured; registry notifications stay public in both router branches. Cookies are not part of Keel authentication.
- Deliberately exclude preflight/CORS operations, Prometheus metrics, static frontend routes, and DEBUG-only expvar/pprof routes, with machine-checked reasons.
- Pin Swag at `v1.16.6`, the OpenAPI Generator npm wrapper at `2.20.2`, the generator engine/validator at `7.22.0`, and JavaScript runtime dependencies through exact package entries and `ui/yarn.lock`.
- Produce only committed `docs/api/swagger.json` and `docs/api/swagger.yaml` for the API description. Go metadata generation is disabled because no runtime documentation package or endpoint is needed.
- Generate and commit Promise-based browser ES modules under `ui/src/api/generated/`. Output must work with the current Vue 2/Babel/Node 16 environment while remaining independently importable by a future frontend.
- The client must have a configurable base URL, stable operation-derived method names, source-derived models, Basic and Bearer configuration, arbitrary documented headers, access to response headers, and no invented cookie behavior.
- Provide Make targets for spec generation, client generation, combined generation, validation, route coverage, client verification, and aggregate deterministic checking.
- Mark generated ownership, never hand-edit generated outputs, suppress timestamps/unstable metadata, and require regeneration followed by `git diff --exit-code` to be clean.
- Add contributor instructions for Go 1.23, Node 16/Yarn 1, and Java 11+, including exact clean-checkout commands.
- Do not expose secrets, serve public Swagger UI/docs, hard-code a host, or change API behavior, auth, payloads, status codes, CORS, deployment, or current frontend behavior.

## Acceptance Criteria

- Pinned generation from a clean checkout reproduces both spec files and the committed client without manual edits.
- Pinned validation accepts JSON and YAML, and route coverage proves exactly 25 documented operations plus every recorded exclusion.
- Generated methods exist for all 25 operations and retain the declared operation IDs, parameters, models, security, and header behavior.
- A Jest smoke test imports generated API/model exports and verifies base URL, Basic/Bearer setup, custom headers, and full-response access; existing UI lint and build checks pass.
- Focused `pkg/http` tests and `go test ./...` pass, with any genuine environment limitation stated explicitly.
- Combined regeneration followed by `git diff --exit-code` returns success and does not alter specs, client files, configuration, dependencies, or lockfiles.
- No runtime or frontend behavior changes are introduced.

## Open Questions

None.
