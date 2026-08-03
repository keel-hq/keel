# Design: Document Keel HTTP API and Generate JavaScript Client

## Source and Artifact Architecture

Swag annotations on the handlers in `pkg/http` define each operation; general API metadata and security definitions live with the Keel command entry point. A small repository tooling module pins `github.com/swaggo/swag v1.16.6`, which is compatible with the repository's Go 1.23 module. Make invokes it through that module with `go run`, not a global executable or `GOPATH/bin` lookup.

Generation commits only `docs/api/swagger.json` and `docs/api/swagger.yaml`. JSON is the canonical client-generator input and YAML is the review-friendly published form; validation also checks that they describe the same document. Swag's Go output is disabled, so there is no `docs.go`, importable documentation package, runtime registration, Swagger UI, or public spec route. The metadata omits `host` and `schemes` so deployments choose their own origin.

Provider and handler-local anonymous/private structs are represented by named documentation DTOs alongside the HTTP package. They mirror existing JSON names and types but are never used to decode requests or encode responses. This keeps generated schemas stable without changing wire behavior.

## Auditable Route Contract

`Auth enabled` means `Authenticator.Enabled()` is true. `Conditional auth` means the route always exists, but Basic or Bearer authorization wraps the handler only when `AuthenticatedWebhooks` is true. Authentication failure returns 401 before handler execution.

| `operationId` | Method and path | Handler | Availability / security | Request contract | Success contract | Meaningful errors |
|---|---|---|---|---|---|---|
| `getHealth` | `GET /healthz` | `healthHandler` | Always / public | No parameters or body | 200, empty body | None emitted by handler |
| `getVersion` | `GET /version` | `versionHandler` | Always / public | No parameters or body | 200, `types.VersionInfo` | 500 text |
| `login` | `POST /v1/auth/login` | `loginHandler` | Auth enabled / public | JSON `LoginRequest` | 200, `auth.AuthResponse`; `Authorization` response header | 400 or 401 text |
| `getAuthInfo` | `GET /v1/auth/info` | `userInfoHandler` | Auth enabled / Basic or Bearer | No body | 200, `UserInfo` | 401 text |
| `getAuthUser` | `GET /v1/auth/user` | `userInfoHandler` | Auth enabled / Basic or Bearer | No body | 200, `UserInfo` | 401 text |
| `logoutGet` | `GET /v1/auth/logout` | `logoutHandler` | Auth enabled / Basic or Bearer | No body | 200, empty JSON object | 401 text |
| `logoutPost` | `POST /v1/auth/logout` | `logoutHandler` | Auth enabled / Basic or Bearer | No body | 200, empty JSON object | 401 text |
| `refreshAuth` | `GET /v1/auth/refresh` | `refreshHandler` | Auth enabled / Basic or Bearer | No body | 200, `auth.AuthResponse`; `Authorization` response header | 401, 403, or 500 text |
| `listApprovals` | `GET /v1/approvals` | `approvalsHandler` | Auth enabled / Basic or Bearer | No body | 200, array of `types.Approval` | Store/encoding failure documented from handler behavior |
| `actOnApproval` | `POST /v1/approvals` | `approvalApproveHandler` | Auth enabled / Basic or Bearer | JSON `ApprovalActionRequest` | 200, `types.Approval`; delete may return JSON `null` | 400, 401, 404, or 500 text |
| `setApprovalRequirements` | `PUT /v1/approvals` | `approvalSetHandler` | Auth enabled / Basic or Bearer | JSON `ApprovalRequirementsRequest` | 200, `APIResponse` | 400, 401, 403, 404, or 500 text |
| `listResources` | `GET /v1/resources` | `resourcesHandler` | Auth enabled / Basic or Bearer | No body | 200, array of `Resource` (currently nullable when empty) | 401 text |
| `updatePolicy` | `PUT /v1/policies` | `policyUpdateHandler` | Auth enabled / Basic or Bearer | JSON `PolicyUpdateRequest` | 200, `APIResponse` | 400, 401, 403, 404, or 500 text |
| `listTrackedImages` | `GET /v1/tracked` | `trackedHandler` | Auth enabled / Basic or Bearer | No body | 200, array of `TrackedImage` (currently nullable when empty) | 401, 403, or 500 text |
| `updateTracking` | `PUT /v1/tracked` | `trackSetHandler` | Auth enabled / Basic or Bearer | JSON `TrackingUpdateRequest` | 200, `APIResponse` | 400, 401, 403, 404, or 500 text |
| `listAuditLogs` | `GET /v1/audit` | `adminAuditLogHandler` | Auth enabled / Basic or Bearer | Optional query: integer `limit`, integer `offset`, comma-separated `filter`, string `email` | 200, `AuditLogsResponse` | 401 or 500 text |
| `getStats` | `GET /v1/stats` | `statsHandler` | Auth enabled / Basic or Bearer | No body | 200, array of `types.AuditLogStats` | 401, 403, or 500 text |
| `receiveNativeWebhook` | `POST /v1/webhooks/native` | `nativeHandler` | Always / conditional auth | JSON `types.Repository` | 200, empty body | 400 text; conditional 401 |
| `receiveDockerHubWebhook` | `POST /v1/webhooks/dockerhub` | `dockerHubHandler` | Always / conditional auth | JSON `DockerHubWebhook` | 200, empty body | 400 text; conditional 401 |
| `receiveJFrogWebhook` | `POST /v1/webhooks/jfrog` | `jfrogHandler` | Always / conditional auth | JSON `JFrogWebhook` | 200, empty body | 400 text; conditional 401 |
| `receiveQuayWebhook` | `POST /v1/webhooks/quay` | `quayHandler` | Always / conditional auth | JSON `QuayWebhook` | 200, empty body | 400 text; conditional 401 |
| `receiveAzureWebhook` | `POST /v1/webhooks/azure` | `azureHandler` | Always / conditional auth | JSON `AzureWebhook` | 200, empty body | 400 text; conditional 401 |
| `receiveGitHubWebhook` | `POST /v1/webhooks/github` | `githubHandler` | Always / conditional auth | Optional `X-GitHub-Event` header (`package` or `registry_package`) and matching `GitHubWebhook` body | 200, empty body | 400 text; conditional 401 |
| `receiveHarborWebhook` | `POST /v1/webhooks/harbor` | `harborHandler` | Always / conditional auth | JSON `HarborWebhook` | 200, empty body | 400 text; conditional 401 |
| `receiveRegistryWebhook` | `POST /v1/webhooks/registry` | `registryNotificationHandler` | Always / public in both router branches | JSON `RegistryNotification` | 200, empty body | 400 text |

The approvals list handler currently writes some error text before calling `WriteHeader(500)`, which can commit HTTP 200. The annotations and focused tests must capture the observed contract without silently fixing this ordering issue; correcting runtime status behavior is outside this task.

## Schema Evidence

The schema names below are stable documentation names. Fields come from the actual local/domain models inspected in `pkg/http`, `pkg/auth`, `types`, and `internal/k8s`.

| Schema | Wire shape to preserve |
|---|---|
| `LoginRequest`, `auth.AuthResponse`, `UserInfo` | `username`/`password`; token-only JSON response; user-info identity/status fields. Login and refresh also expose `Authorization`. |
| `ApprovalActionRequest` | `id`, `voter`, `identifier`, `action`; action values are `approve`, `reject`, `delete`, `archive`, with empty meaning approve. |
| `ApprovalRequirementsRequest`, `PolicyUpdateRequest`, `TrackingUpdateRequest` | Existing identifier/provider/vote, policy, and trigger/schedule JSON fields and validation ranges/enums. |
| `Resource`, `TrackedImage`, `AuditLogsResponse` | Exact handler-local response fields; resource status uses `internal/k8s.Status`; audit data uses `types.AuditLog`. |
| `types.Repository` | `host`, `name`, `tag`, `digest`. |
| `DockerHubWebhook` | `push_data` (`pushed_at`, `images`, `tag`, `pusher`), `callback_url`, and full repository metadata including `repo_name`. |
| `JFrogWebhook` | `domain`, `event_type`, nested `data` including image/tag/platforms, `subscription_key`, `jpd_origin`, `source`. |
| `QuayWebhook` | `name`, `repository`, `namespace`, `docker_url`, `homepage`, `updated_tags`. |
| `AzureWebhook` | Nested `target.repository/tag/digest` and `request.host`. |
| `GitHubWebhook` | Both source variants: `package` with namespace/name/container tag/digest and `registry_package` with package type/version plus `repository.full_name`. Swagger 2 cannot express a body union, so one documentation-only superset has optional branches and the header description explains their pairing. Unknown/missing event headers remain documented as current handler behavior, not rejected by the spec. |
| `HarborWebhook` | `type`, `occur_at`, `operator`, `event_data.resources` and `event_data.repository`. |
| `RegistryNotification` | `events` containing id/time/action plus complete target, request, actor, and source structures used by Docker Distribution notifications. |

Response schemas reuse `types.VersionInfo`, `types.Approval`, `types.AuditLog`, `types.AuditLogStats`, `auth.AuthResponse`, and named documentation response DTOs. Text error bodies are documented as strings. Documentation wrappers do not alter nullability, field names, payloads, or handler code paths.

## Security and Availability

Swagger 2 definitions are `basicAuth` (`type: basic`) and `bearerAuth` (`type: apiKey`, `in: header`, `name: Authorization`, documented value `Bearer <token>`). Protected operations use alternative security requirements, so Basic and Bearer are alternatives rather than cumulative requirements.

The seven configurable webhooks cannot be fully described by static Swagger 2 security. Their operation descriptions and a checked `x-keel-conditional-security` extension record that anonymous access is allowed when `AuthenticatedWebhooks=false` and Basic/Bearer is required when true. `/v1/webhooks/registry` has no security requirement because `registerWebhookRoutes` never wraps it. The generated client allows callers to set either authorization mode or an arbitrary header but does not send credentials by default and does not enable cookies.

## Explicit Exclusions

| Registered surface | Decision and rationale |
|---|---|
| All registered `OPTIONS` methods | Exclude. `corsHeadersMiddleware` answers preflight before handler logic; generated business operations would be misleading. |
| `/metrics` | Exclude. It is third-party Prometheus text exposition, not a Keel-owned JSON API. |
| `/css/`, `/assets/`, `/js/`, `/img/`, `/loading/`, and `/` catch-all | Exclude. These auth/UI-directory-dependent routes serve static frontend files or the SPA shell, not API contracts. |
| `/debug/vars` and `/debug/pprof/cmdline`, `/profile`, `/symbol`, `/trace`, `/debug/pprof/` | Exclude. They exist only with `DEBUG=true` and are expvar/net/http/pprof diagnostics, not supported Keel API operations. |

A route-coverage fixture/test inventories these surfaces despite excluding them, walks Gorilla Mux registrations under authenticator enabled/disabled, authenticated webhooks enabled/disabled, and DEBUG enabled/disabled, then compares in-scope method/path pairs with the generated spec. It must report exactly 25 operations across 21 paths and fail on an unclassified route.

## Pinned Deterministic Tooling

- Swag is pinned to `v1.16.6` in an isolated Go tooling module and invoked with `go run` from that module. Its output type is explicitly `json,yaml`; generated Go output is disabled.
- `ui/package.json` pins `@openapitools/openapi-generator-cli` to `2.13.4`, and `ui/openapitools.json` pins the OpenAPI Generator engine and its `validate` command to `7.7.0`. Both versions are represented in `ui/yarn.lock`; no global npm package or floating download is used.
- The client runtime pins `superagent` to `8.1.2` in `ui/package.json` and `ui/yarn.lock` rather than accepting a generator-produced range.
- Generator configuration pins the `javascript` generator, output selection, package metadata, sort behavior, Promise mode, and `hideGenerationTimestamp=true`. API/model docs, samples, tests, timestamps, and redundant nested package files are disabled.
- Generation deletes only the owned `docs/api` outputs and `ui/src/api/generated` directory before recreating them, preventing stale generated files. Generated headers plus `.gitattributes` identify ownership and discourage manual edits.

The pinned OpenAPI Generator `validate` command validates both JSON and YAML; a deterministic comparison normalizes both and proves semantic equivalence. No command conditionally installs tools, uses a floating version selector, or assumes a Go bin directory is on `PATH`.

## JavaScript Client Contract

OpenAPI Generator `7.7.0` with its maintained `javascript` generator produces Promise-based ES2015 modules under `ui/src/api/generated/`. This format is plain JavaScript, is consumable by Vue CLI 3/Babel in the current Node 16 frontend, and remains framework-neutral for a future frontend. Existing `ui/src/api/index.js` and call sites remain untouched.

The committed output contains only the API client runtime, tag-grouped API classes, models, and barrel exports. Method names derive exclusively from the 25 explicit `operationId` values in the inventory; generation fails on duplicate or missing IDs. The client exposes a caller-configurable base path with no compiled host, Basic username/password configuration, Bearer `Authorization` configuration, custom documented headers such as `X-GitHub-Event`, and full-response methods that expose status and response headers. Cookies and browser credential mode are not enabled.

Generated code is committed because it lives with and is verified by the existing frontend, gives future frontend work an immediately consumable SDK, and lets drift checks review spec/client changes together. It is ignored by handwritten ESLint formatting rules, but a Jest smoke test imports its barrel/API/model exports through Babel and verifies base URL, serialization, authentication/header configuration, and full-response access. Existing `yarn lint` still covers handwritten UI code, and `yarn build` verifies the current frontend remains healthy.

## Repository Commands and Verification

Make exposes these stable entry points:

- `api-spec-generate`: regenerate only JSON/YAML from annotations using pinned Swag.
- `api-client-generate`: clean and regenerate the committed JavaScript client from `swagger.json` using the pinned engine/configuration.
- `api-generate`: run both generators in dependency order.
- `api-spec-validate`: validate both formats and compare their normalized content.
- `api-route-coverage`: run the router/spec operation and exclusion check.
- `api-client-check`: run the generated-client Jest smoke import plus relevant UI lint/build checks.
- `api-check`: generate, validate, check coverage, run client checks, and finish with `git diff --exit-code`.

Contributor documentation records Go 1.23, Node 16, Yarn 1, and the Java version required by OpenAPI Generator 7.7.0, then gives the clean-checkout workflow:

```bash
go mod download
(cd tools/api && go mod download)
yarn --cwd ui install --frozen-lockfile
make api-check
```

CI runs pinned spec validation, exact route coverage, focused `go test ./pkg/http/...`, `go test ./...`, client smoke verification, existing UI lint/build, and regeneration followed by `git diff --exit-code`. Environment-dependent failures are reported with their real prerequisite or external-service cause; checks are not silently skipped.

## Key Constraints and Learnings

- Gorilla Mux registration in `pkg/http/http.go`, rather than existing UI usage, is the API coverage authority.
- Current UI code uses Vue 2.5, Vue CLI/Babel 7, Node 16, Yarn, and a handwritten `vue-resource` wrapper; generated-client adoption is deliberately deferred.
- Response helpers sometimes emit plain text and derive 403 from an error string. Documentation must describe this behavior, not normalize it into a new error envelope.
- Webhook trigger submission errors are currently ignored by handlers. The documentation task must not invent new failure statuses or alter this behavior.
