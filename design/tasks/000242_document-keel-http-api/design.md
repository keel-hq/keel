# Design: Document Keel HTTP API and Generate JavaScript Client

## Pinned Generation Architecture

Handler annotations in `pkg/http` and general API annotations near `cmd/keel/main.go` are the description source. A small `tools` Go module requires `github.com/swaggo/swag v1.16.6`; Make runs that module with `go run`, so no global binary or shell PATH convention is involved. Swag is configured for deterministic JSON/YAML output in `docs/api/`, with Go metadata output disabled. The specification has relative paths and no fixed host or scheme.

The same spec drives the maintained OpenAPI Generator `javascript` client. `ui/package.json` pins `@openapitools/openapi-generator-cli` to `2.20.2`, `ui/openapitools.json` pins engine `7.22.0`, and the engine's `validate` command is the pinned spec validator. This wrapper supports the repository's Node 16 CI; the engine requires Java 11+.

## Auditable Operation Inventory

`Auth-enabled` routes exist only when `Authenticator.Enabled()` is true. `Conditional webhook auth` means the route always exists but is wrapped with Basic/Bearer authorization only when `AuthenticatedWebhooks` is true. A failed wrapper check returns 401 before the handler.

| Operation ID | Method and path | Handler | Availability / security | Request | Success | Errors |
|---|---|---|---|---|---|---|
| `getHealth` | `GET /healthz` | `healthHandler` | Always / public | None | 200 empty | None |
| `getVersion` | `GET /version` | `versionHandler` | Always / public | None | 200 `types.VersionInfo` | 500 string |
| `loginAuth` | `POST /v1/auth/login` | `loginHandler` | Auth-enabled / public | `loginRequest` | 200 `auth.AuthResponse` and `Authorization` header | 400, 401 string |
| `getAuthInfo` | `GET /v1/auth/info` | `userInfoHandler` | Auth-enabled / Basic or Bearer | None | 200 `UserInfo` | 401 string |
| `getAuthUser` | `GET /v1/auth/user` | `userInfoHandler` | Auth-enabled / Basic or Bearer | None | 200 `UserInfo` | 401 string |
| `logoutAuthGet` | `GET /v1/auth/logout` | `logoutHandler` | Auth-enabled / Basic or Bearer | None | 200 `{}` | 401 string |
| `logoutAuthPost` | `POST /v1/auth/logout` | `logoutHandler` | Auth-enabled / Basic or Bearer | None | 200 `{}` | 401 string |
| `refreshAuth` | `GET /v1/auth/refresh` | `refreshHandler` | Auth-enabled / Basic or Bearer | None | 200 `auth.AuthResponse` and `Authorization` header | 401, 500 string |
| `listApprovals` | `GET /v1/approvals` | `approvalsHandler` | Auth-enabled / Basic or Bearer | None | 200 `[]types.Approval` | 401, 500 string |
| `actOnApproval` | `POST /v1/approvals` | `approvalApproveHandler` | Auth-enabled / Basic or Bearer | `approveRequest` | 200 `types.Approval`, or JSON null for delete | 400, 401, 404, 500 string |
| `setApprovalRequirements` | `PUT /v1/approvals` | `approvalSetHandler` | Auth-enabled / Basic or Bearer | `resourceApprovalsUpdateRequest` | 200 `APIResponse` | 400, 401, 403, 404, 500 string |
| `listResources` | `GET /v1/resources` | `resourcesHandler` | Auth-enabled / Basic or Bearer | None | 200 `[]resource` | 401 string |
| `updatePolicy` | `PUT /v1/policies` | `policyUpdateHandler` | Auth-enabled / Basic or Bearer | `resourcePolicyUpdateRequest` | 200 `APIResponse` | 400, 401, 403, 404, 500 string |
| `listTrackedImages` | `GET /v1/tracked` | `trackedHandler` | Auth-enabled / Basic or Bearer | None | 200 `[]trackedImage` | 401, 403, 500 string |
| `updateTracking` | `PUT /v1/tracked` | `trackSetHandler` | Auth-enabled / Basic or Bearer | `trackRequest` | 200 `APIResponse` | 400, 401, 403, 404, 500 string |
| `listAuditLogs` | `GET /v1/audit` | `adminAuditLogHandler` | Auth-enabled / Basic or Bearer | Query `filter`, `email`, `limit`, `offset` | 200 `auditLogsResponse` | 401, 500 string |
| `getStats` | `GET /v1/stats` | `statsHandler` | Auth-enabled / Basic or Bearer | None | 200 `[]types.AuditLogStats` | 401, 403, 500 string |
| `receiveNativeWebhook` | `POST /v1/webhooks/native` | `nativeHandler` | Conditional webhook auth | `types.Repository` | 200 empty | 400 string; conditional 401 |
| `receiveDockerHubWebhook` | `POST /v1/webhooks/dockerhub` | `dockerHubHandler` | Conditional webhook auth | `dockerHubWebhook` | 200 empty | 400 string; conditional 401 |
| `receiveJFrogWebhook` | `POST /v1/webhooks/jfrog` | `jfrogHandler` | Conditional webhook auth | `jfrogWebhook` | 200 empty | 400 string; conditional 401 |
| `receiveQuayWebhook` | `POST /v1/webhooks/quay` | `quayHandler` | Conditional webhook auth | `quayWebhook` | 200 empty | 400 string; conditional 401 |
| `receiveAzureWebhook` | `POST /v1/webhooks/azure` | `azureHandler` | Conditional webhook auth | `azureWebhook` | 200 empty | 400 string; conditional 401 |
| `receiveGitHubWebhook` | `POST /v1/webhooks/github` | `githubHandler` | Conditional webhook auth | `X-GitHub-Event` plus matching GitHub payload | 200 empty | 400 string; conditional 401 |
| `receiveHarborWebhook` | `POST /v1/webhooks/harbor` | `harborHandler` | Conditional webhook auth | `harborWebhook` | 200 empty | 400 string; conditional 401 |
| `receiveRegistryWebhook` | `POST /v1/webhooks/registry` | `registryNotificationHandler` | Always / public, even when webhook auth is enabled | `registryNotification` | 200 empty | 400 string |

## Schema Evidence

Documentation DTOs retain the source JSON names and field structure. They exist only to give Swag stable named definitions; handlers keep their current types.

| Schema | Source-verified wire fields |
|---|---|
| `loginRequest` / `auth.AuthResponse` | `username`, `password` / `token`; login and refresh also expose `Authorization` response header |
| `approveRequest` | `id`, `voter`, `identifier`, `action` (`approve`, `reject`, `delete`, `archive`) |
| `resourceApprovalsUpdateRequest` | `identifier`, `provider`, `votesRequired` |
| `resourcePolicyUpdateRequest` | `policy`, `identifier`, `provider` |
| `trackRequest` | `provider`, `identifier`, `trigger`, `schedule` |
| `types.Repository` | `host`, `name`, `tag`, `digest` |
| `dockerHubWebhook` | `push_data` including tag/pusher, `callback_url`, and Docker Hub `repository` metadata including `repo_name` |
| `jfrogWebhook` | `domain`, `event_type`, `data` including image/tag/platforms, `subscription_key`, `jpd_origin`, `source` |
| `quayWebhook` | `name`, `repository`, `namespace`, `docker_url`, `homepage`, `updated_tags` |
| `azureWebhook` | `target` (`repository`, `tag`, `digest`) and `request.host` |
| GitHub package DTOs | Required `X-GitHub-Event` chooses `package` or `registry_package`; definitions preserve namespace/name/version/tag/digest fields used by the handler |
| `harborWebhook` | `type`, `occur_at`, `operator`, and `event_data.resources`/`event_data.repository` |
| `registryNotification` | `events` with action/target/request/actor/source, including repository, host, tag, and digest |
| Admin responses | `UserInfo`, `APIResponse`, `resource`, `trackedImage`, `auditLogsResponse`, `types.Approval`, `types.AuditLogStats`, and `types.VersionInfo` retain their current JSON tags |

Swagger 2 cannot express a body union directly, so the GitHub request definition is a documentation-only superset with optional named branches and an enum-constrained `X-GitHub-Event` header. Validation and focused handler tests ensure both real variants remain represented.

## Security Contract

Define `basicAuth` as Swagger 2 `type: basic` and `bearerAuth` as `type: apiKey`, `in: header`, `name: Authorization`, with the documented `Bearer <token>` convention. Protected admin operations use alternative security requirement objects, not a combined requirement. The seven configurable webhooks use anonymous, Basic, or Bearer alternatives plus a Keel extension/description explaining the runtime flag. Registry and public operations declare no security requirement.

Login and refresh describe both the JSON token and exposed `Authorization` response header. The generated client offers Basic credentials and full Bearer-header configuration; its `WithHttpInfo` methods provide response headers. No cookie auth or cross-origin credential default is introduced.

## Exclusion Inventory

| Router surface | Decision |
|---|---|
| Registered `OPTIONS` methods | Excluded: `corsHeadersMiddleware` handles preflight before endpoint logic; client methods would be misleading. |
| `/metrics` | Excluded: Prometheus text exposition is third-party and not a Keel JSON contract. |
| `/css/`, `/assets/`, `/js/`, `/img/`, `/loading/`, `/` | Excluded: auth/UI-directory-dependent static delivery and SPA fallback, not HTTP API operations. |
| `/debug/vars`, `/debug/pprof/cmdline`, `/profile`, `/symbol`, `/trace`, and `/debug/pprof/` | Excluded: `DEBUG=true` diagnostics owned by expvar/pprof rather than the supported Keel API. Coverage still inventories them. |

## Generated JavaScript Client

Generate the stable OpenAPI Generator `javascript` target with `usePromises=true`, `useURLSearchParams=true`, deterministic ordering/naming, a fixed project version, and generated docs/samples/timestamps disabled. Emit `ApiClient`, API classes, models, and barrel exports directly under `ui/src/api/generated/`; do not emit a nested package or duplicate build configuration.

The committed client uses ES2015 modules and syntax accepted by the repository's Vue CLI 3/Babel pipeline. It uses root-level `superagent` pinned to `5.3.1` and the existing Yarn lockfile. Generated sources are excluded only from handwritten style rules; Jest must parse and import them. The handwritten `ui/src/api/index.js` and all current call sites remain unchanged.

Method names are determined solely by the 25 explicit operation IDs in the inventory. The client has a caller-set base URL rather than a compiled hostname. API methods serialize documented query/body/header fields, auth schemes set Basic or `Authorization: Bearer ...`, and full-response variants expose status and response headers. Generated artifacts are committed, marked in `.gitattributes`, owned by the generator, and changed only by regeneration.

## Commands, Coverage, and Determinism

Prerequisites are Go 1.23, Node 16, Yarn 1, and Java 11+. Contributor documentation gives this clean-checkout sequence:

```bash
go mod download
(cd tools && go mod download)
yarn --cwd ui install --frozen-lockfile
make api-generate
make api-validate api-coverage api-client-test
git diff --exit-code
```

Make targets:

- `api-spec`: run the module-pinned Swag command and replace JSON/YAML output.
- `api-client`: run the local Yarn wrapper with engine `7.22.0` and replace only the owned client directory.
- `api-generate`: run `api-spec` then `api-client`.
- `api-validate`: validate both formats with the pinned generator validator.
- `api-coverage`: run a Go test that walks Gorilla Mux with auth and DEBUG enabled, drops preflight methods, and compares method/path/operation ID with the 25-row inventory plus exclusions.
- `api-client-test`: run the generated-client Jest smoke import, relevant UI lint, and UI build.
- `api-check`: validate, cover, test, regenerate, and finish with `git diff --exit-code` across specs, client, generator configuration, dependencies, and locks.

CI runs `api-check`, focused `pkg/http` tests, and `go test ./...`. Environment-dependent failures are documented with their actual cause; checks are not silently bypassed. The design adds no Swagger server, frontend migration, or runtime behavior.
