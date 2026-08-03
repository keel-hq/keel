# Design: Document the Keel API and Generate a Reproducible JavaScript Client

## Source Findings and Scope

`pkg/http/http.go` registers gorilla/mux routes. Auth/admin routes are inside `authenticator.Enabled()`, the UI catch-all is additionally gated by a non-empty `uiDir`, and debug routes are gated by `DEBUG=true`. The helper `requireAdminAuthorization` accepts either HTTP Basic credentials or an `Authorization` bearer token and returns 401 before a protected handler. Current handlers use private request/response structs plus models in `types`, `pkg/auth`, and `internal/k8s`; annotations must reference those exact fields and JSON names.

The table is the auditable contract. “Text” means the handler's existing plain/empty error body; response documentation will not invent a JSON error envelope. Protected operations also have the middleware's 401 response. Handler calls through `response` may return 403 for an error containing `Permission denied`, otherwise 500.

| Method | Path | operationId / handler | Enabled when | Request model or parameters | Responses | Security |
|---|---|---|---|---|---|---|
| GET | `/healthz` | `healthCheck` / `healthHandler` | Always | None | 200 empty | None |
| GET | `/version` | `getVersion` / `versionHandler` | Always | None | 200 `types.VersionInfo`; 500 empty | None |
| POST | `/v1/auth/login` | `login` / `loginHandler` | Authenticator enabled | `loginRequest` JSON | 200 `auth.AuthResponse` body (`token`) plus `Authorization` header; 400/401 text | None |
| GET | `/v1/auth/info` | `getAuthInfo` / route-specific wrapper to `userInfoHandler` | Authenticator enabled | None | 200 `UserInfo`; 401 text | Basic or Bearer |
| GET | `/v1/auth/user` | `getAuthUser` / route-specific wrapper to `userInfoHandler` | Authenticator enabled | None | 200 `UserInfo`; 401 text | Basic or Bearer |
| GET | `/v1/auth/logout` | `logoutViaGet` / route-specific wrapper to `logoutHandler` | Authenticator enabled | None | 200 empty object; 401 text | Basic or Bearer |
| POST | `/v1/auth/logout` | `logout` / route-specific wrapper to `logoutHandler` | Authenticator enabled | None | 200 empty object; 401 text | Basic or Bearer |
| GET | `/v1/auth/refresh` | `refreshAuth` / `refreshHandler` | Authenticator enabled | None | 200 `auth.AuthResponse` plus `Authorization` header; 401/500 text | Basic or Bearer |
| GET | `/v1/approvals` | `listApprovals` / `approvalsHandler` | Authenticator enabled | None | 200 array of `types.Approval`; current write-order error path remains documented/tested as observed | Basic or Bearer |
| POST | `/v1/approvals` | `updateApproval` / `approvalApproveHandler` | Authenticator enabled | `approveRequest` JSON (`approve`, `reject`, `delete`, `archive`) | 200 `types.Approval` or JSON `null` for delete; 400/404/500 text, including observed write-order behavior | Basic or Bearer |
| PUT | `/v1/approvals` | `setResourceApprovals` / `approvalSetHandler` | Authenticator enabled | `resourceApprovalsUpdateRequest` JSON | 200 `APIResponse`; 400/403/404/500 text | Basic or Bearer |
| GET | `/v1/resources` | `listResources` / `resourcesHandler` | Authenticator enabled | None | 200 array of `resource` (JSON `null` when the source slice is nil); 401 | Basic or Bearer |
| PUT | `/v1/policies` | `updateResourcePolicy` / `policyUpdateHandler` | Authenticator enabled | `resourcePolicyUpdateRequest` JSON | 200 `APIResponse`; 400/403/404/500 text | Basic or Bearer |
| GET | `/v1/tracked` | `listTrackedImages` / `trackedHandler` | Authenticator enabled | None | 200 array of `trackedImage` (possibly JSON `null`); 403/500 text | Basic or Bearer |
| PUT | `/v1/tracked` | `updateTrackedImage` / `trackSetHandler` | Authenticator enabled | `trackRequest` JSON | 200 `APIResponse`; 400/403/404/500 text | Basic or Bearer |
| GET | `/v1/audit` | `listAuditLogs` / `adminAuditLogHandler` | Authenticator enabled | Optional query: integer `limit`, integer `offset`, comma-separated `filter`, string `email`; invalid integers currently become zero | 200 `auditLogsResponse`; 403/500 text | Basic or Bearer |
| GET | `/v1/stats` | `getStats` / `statsHandler` | Authenticator enabled | None | 200 array of `types.AuditLogStats`; 403/500 text | Basic or Bearer |
| POST | `/v1/webhooks/native` | `receiveNativeWebhook` / `nativeHandler` | Always | `types.Repository` JSON; `name` and `tag` required by handler | 200 empty; 400 text | Basic or Bearer only when `authenticatedWebhooks=true`; otherwise none |
| POST | `/v1/webhooks/dockerhub` | `receiveDockerHubWebhook` / `dockerHubHandler` | Always | Provider-specific `dockerHubWebhook` JSON | 200 empty; 400 text | Conditional webhook auth |
| POST | `/v1/webhooks/jfrog` | `receiveJFrogWebhook` / `jfrogHandler` | Always | Provider-specific `jfrogWebhook` JSON, including `data.platforms` | 200 empty; 400 text | Conditional webhook auth |
| POST | `/v1/webhooks/quay` | `receiveQuayWebhook` / `quayHandler` | Always | Provider-specific `quayWebhook` JSON | 200 empty; 400 text | Conditional webhook auth |
| POST | `/v1/webhooks/azure` | `receiveAzureWebhook` / `azureHandler` | Always | Provider-specific `azureWebhook` JSON | 200 empty; 400 text | Conditional webhook auth |
| POST | `/v1/webhooks/github` | `receiveGitHubWebhook` / `githubHandler` | Always | Required `X-GitHub-Event` header selects exact `githubPackageV2Webhook` (`package`) or `githubRegistryPackageWebhook` (`registry_package`) JSON; other values currently return 200 | 200 empty; 400 text | Conditional webhook auth |
| POST | `/v1/webhooks/harbor` | `receiveHarborWebhook` / `harborHandler` | Always | Provider-specific `harborWebhook` JSON for `pushImage` or `PUSH_ARTIFACT` | 200 empty; 400 text | Conditional webhook auth |
| POST | `/v1/webhooks/registry` | `receiveRegistryWebhook` / `registryNotificationHandler` | Always | Docker Registry `registryNotification` JSON with `events` | Implicit 200 empty; 400 empty | None, even when `authenticatedWebhooks=true` |

This is 21 distinct paths and 25 operations. During implementation, focused tests must pin any subtle status produced by `Write`/`WriteHeader` order before annotations are finalized; the implementation must document the observed status rather than “fix” it.

## Explicit Exclusions

| Route/category | Availability | Reason excluded |
|---|---|---|
| OPTIONS and CORS preflight | Global middleware; OPTIONS is also named on application routes | Transport/browser plumbing, not a client operation; current wildcard CORS behavior remains unchanged. |
| `/metrics` | Always | Prometheus exposition endpoint, not the JSON application API. |
| `/css/`, `/assets/`, `/js/`, `/img/`, `/loading/`, and `/` catch-all | Authenticator enabled and `uiDir` non-empty | Static UI delivery, not an API contract. |
| `/debug/vars`, `/debug/pprof/` and its cmdline/profile/symbol/trace routes | `DEBUG=true` | Operational DEBUG-only endpoints with standard-library wire formats. |

No Swagger UI, spec-serving handler, or other public docs route will be registered.

## Swagger Annotations and Models

Use `github.com/swaggo/swag/cmd/swag@v1.16.6`, whose Go 1.18 minimum is compatible with this repository's Go 1.23 directive. General API annotations define Swagger 2.0 metadata, base path `/`, no fixed host, and two security definitions: HTTP Basic and an API-key-in-header definition named Bearer whose value is the complete `Bearer <token>` Authorization header. Protected operations express Basic-or-Bearer alternatives. Conditional webhook security is also stated in each description and in a vendor extension because a static Swagger 2.0 document cannot select security from runtime configuration; registry explicitly has no security annotation.

Each router operation gets one annotation block and a unique ID from the table. Small route-specific forwarding handlers are used only for the `/auth/info` and `/auth/user` aliases and the two logout methods so IDs remain unique; they call the existing implementation unchanged. Anonymous nested webhook fields may be promoted to named private structs, and exact documentation-only union/empty wrappers may be added where Swagger 2.0 needs a named schema. JSON names, optionality, numeric widths, accepted GitHub variants, and response bodies must match source and focused tests exactly.

## Deterministic Tooling and Artifacts

Add repository Make targets with explicit commands:

- `api-spec`: run `go run github.com/swaggo/swag/cmd/swag@v1.16.6 init` from the repository root with the general-info entry point, internal/dependency parsing needed for referenced Keel models, output directory `docs`, and `--outputTypes yaml` only.
- `api-client`: run the official `openapitools/openapi-generator-cli:v7.22.0` container with the repository bind-mounted, generator `javascript`, the committed config/ignore files, and input `docs/swagger.yaml`.
- `api-validate`: run Spectral 6.16.1 from the exact Yarn lock and the route/spec coverage checker.
- `api-generate`: run spec then client generation from a clean generated-client directory so deleted models cannot linger.
- `api-check`: run generation and validation, then `git diff --exit-code -- docs/swagger.yaml ui/src/api/generated ui/package.json ui/yarn.lock` (and other committed generator config/notice files) to reject stale or nondeterministic output.

Use `hideGenerationTimestamp=true`, stable sorting, Promise mode, URLSearchParams, and a checked-in generator config. Never invoke a global binary, `@latest`, or an implicit install fallback.

Artifact policy:

- Commit `docs/swagger.yaml` as the sole canonical spec. Add ignore entries for `docs/swagger.json` and `docs/docs.go` and do not generate them; `docs.go` is neither imported nor linked into the runtime.
- Commit the generator configuration/ignore file and all useful generated ES-module SDK files under `ui/src/api/generated` (`ApiClient`, tag API classes, models, and index). Each generated JavaScript file retains its generator notice.
- Ignore generator metadata, SDK README/docs/tests, Git/npm scaffolding, and generated package manifests; the existing `ui/package.json` owns runtime dependencies and scripts.
- Treat annotations/models as human-owned and generated YAML/client files as generator-owned. Any annotation, route, schema, tool-version, or generator-config change regenerates and commits both YAML and client in the same change. Tool upgrades are explicit reviewed version bumps followed by the full clean-diff verification.
- Add a short generated notice in `docs/README.md`/contributor documentation because YAML comments are not a reliable generated-file marker.

## JavaScript Client Contract

OpenAPI Generator v7.22.0 is actively maintained, accepts Swagger 2.0, and marks its JavaScript client stable. Generate into `ui/src/api/generated` with `sourceFolder=.` and ES module `import`/`export` syntax; Vue CLI 3 and Babel 7 already process this format. Use `usePromises=true`, `useURLSearchParams=true`, original JSON wire property names, and tags to produce System, Auth, Admin, and Webhooks API classes. Method names come only from the stable operation IDs above.

The spec omits a deployment host and the generated `ApiClient` defaults to same-origin `/`; consumers can construct/configure an instance with another base URL. Cookies are disabled (`enableCookies=false`) because current Keel auth uses the Authorization header and wildcard CORS cannot support credentialed cross-origin cookies. Callers may deliberately enable credentials only for a compatible deployment. Basic username/password and the complete Bearer header value are runtime configuration, never generated secrets.

Pin `superagent` 10.3.0 exactly in `ui/package.json` and both committed lockfiles, rather than accepting the generator template's range; it supports the repository's Node 16.20 runtime. Promise calls resolve `{data, response}`, preserving response headers needed after login/refresh, and reject with the generated structured error containing status, body, raw response, and transport error. Empty/text responses and nullable list behavior follow the spec rather than being normalized by handwritten code. The generated client is built and smoke-imported, but the current `ui/src/api/index.js` and Vuex calls are not migrated in this task.

## Validation

Add a focused route/spec contract test that registers representative enabled/disabled server configurations, parses `docs/swagger.yaml`, and prints an auditable line for each expected `METHOD path -> operationId`. It must fail on missing, extra, duplicate, or mismatched operations and print/verify the exclusions above. Security tests verify admin 401 behavior, open versus protected provider webhooks, and that registry remains unprotected in both webhook modes.

The documented verification sequence is: generate; Spectral 6.16.1 lint/Swagger validation; 25-operation coverage report; generated-client Jest smoke import and representative method/auth/base-path assertions under Node 16.20; `yarn lint:nofix`; `yarn build`; `go test ./pkg/http`; `go test ./...`; then the clean-diff check. Full Go tests may require network, container, database, or Kubernetes-related environment support; any unavailable dependency or failure is recorded with the command and error, not silently skipped. CI runs the deterministic API checks in addition to existing checks.

## Compatibility and Security

Only comments, exact schema types, route-specific forwarding wrappers, generated artifacts, tests, docs, and build/CI commands change. Route registration conditions, middleware order, request parsing, responses/statuses, CORS, UI serving, and deployment binaries remain behaviorally identical. Generated outputs contain no credentials or fixed environment URL, and implementation does not expose docs, migrate the frontend, merge, publish, or deploy.

## Implementation Notes

- `api_inventory_test.go` is the checked-in source of the expected 25-operation mapping. Router walking filters OPTIONS and non-application endpoints, while request-level cases pin authenticator and webhook-security conditions, including registry's deliberate exception.
