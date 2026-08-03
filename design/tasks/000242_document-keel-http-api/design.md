# Design: Document Keel HTTP API and Generate JavaScript Client

## Architecture

Handler comments in `pkg/http` and API metadata near `cmd/keel/main.go` are the source of truth. A Make target runs `swag v1.16.6` and writes generated artifacts to `docs/api/`; no Swagger UI route is added. Named documentation-only DTOs may be introduced where current private or anonymous structs cannot express the wire schema cleanly, but handlers continue using the same payloads.

Every documented handler receives an explicit `@ID` so client method names remain stable. Global Basic and Bearer schemes model `requireAdminAuthorization` as alternatives. The seven configurable webhook operations describe authentication as optional/configuration-dependent, while the registry webhook is explicitly public per current code. Errors remain plain text where handlers currently return plain text.

## JavaScript Client

Use `@openapitools/openapi-generator-cli` `2.20.2` with `openapitools.json` pinning engine `7.22.0`. Generate the `javascript` browser client with ES6 modules into `ui/src/api/generated`; add any required runtime dependency and lock it with Yarn. Commit the output because Keel already commits generated source and this lets a future frontend consume it without a local generator. Generated files are not hand-edited or forced through repository style rules, but a focused UI/Jest smoke check imports the client and verifies representative APIs, models, base URL, and Basic/Bearer configuration.

This choice fits the existing Vue 2, Babel, JavaScript, Yarn, and Node 16 setup. It avoids adding TypeScript or replacing the handwritten `ui/src/api/index.js`; integrating or migrating current screens is out of scope.

## Coverage and Verification

A Go route-coverage test builds the router in configurations that expose auth and debug routes, walks Gorilla Mux, and compares method/path pairs with `docs/api/swagger.json`. Its explicit allowlist excludes OPTIONS, `/metrics`, static/index prefixes, and debug/pprof paths with reasons. The expected documented baseline is 25 operations, including separate GET and POST logout operations.

Make targets provide spec generation, client generation, validation, coverage, and an aggregate verification command. CI runs validation and regeneration followed by `git diff --exit-code`, alongside Go tests and UI checks. Generator timestamps or other unstable metadata are disabled so repeated runs are deterministic.

## Repository Learnings and Constraints

- Routes are centralized in `pkg/http/http.go`; debug routes are added by `pkg/http/debug.go` only when `DEBUG=true`, and admin/UI routes exist only when authentication is enabled.
- The global CORS middleware handles OPTIONS before handlers. OPTIONS should not become client operations.
- Admin authorization accepts HTTP Basic or an `Authorization: Bearer` token. Login and refresh also return the token in both JSON and the `Authorization` response header.
- The audit endpoint has `filter`, `email`, `limit`, and `offset` query parameters; the GitHub webhook uses `X-GitHub-Event` to select one of two payload shapes.
- The existing UI uses relative `/v1` calls through `vue-resource`, Vue CLI 3, Babel, Yarn, and Node 16 CI. The generated client must preserve configurable base URLs rather than assume a new deployment path.
- Static assets, Prometheus, expvar, and pprof are registered on the same router but are not Keel JSON client APIs. No public documentation server or authentication change is needed.
