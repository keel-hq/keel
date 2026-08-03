# Design: Document Keel HTTP API and Generate JavaScript Client

## Source and Artifacts

Swag annotations on `pkg/http` handlers plus API metadata near `cmd/keel/main.go` are the source of truth. A separate tooling module pins `github.com/swaggo/swag v1.16.6`; `make api-spec` invokes it with `go run` and `--outputTypes json,yaml` to produce only `docs/api/swagger.json` and `docs/api/swagger.yaml`. No `docs.go`, runtime import, hard-coded host, or documentation HTTP route is added. An adjacent contributor README and `.gitattributes` identify ownership and generated files because JSON cannot carry comments.

Named documentation DTOs mirror private and anonymous handler structs without being substituted into runtime decoding. Plain-text error bodies remain strings. GitHub uses a documentation wrapper containing the two source-verified shapes (`githubPackageV2Webhook` and `githubRegistryPackageWebhook`) selected by the required `X-GitHub-Event` header; this avoids pretending all webhooks share one generic payload.

## Route and Contract Inventory

Conditions: **Always** means registered in every configuration; **Auth** means only when `Authenticator.Enabled()`; **Webhook auth** means always registered and optionally Basic/Bearer protected when `AuthenticatedWebhooks` is true. Protected operations return 401 before the handler when credentials fail.

| Method and path | Handler | Condition / security | Request | Success response | Handler errors |
|---|---|---|---|---|---|
| `GET /healthz` | `healthHandler` | Always / public | None | 200 empty | None |
| `GET /version` | `versionHandler` | Always / public | None | 200 `types.VersionInfo` | 500 string |
| `POST /v1/auth/login` | `loginHandler` | Auth / public | `loginRequest` | 200 `auth.AuthResponse`; `Authorization` header | 400, 401 string |
| `GET /v1/auth/info` | `userInfoHandler` | Auth / Basic or Bearer | None | 200 `UserInfo` | 401 string |
| `GET /v1/auth/user` | `userInfoHandler` | Auth / Basic or Bearer | None | 200 `UserInfo` | 401 string |
| `GET /v1/auth/logout` | `logoutHandler` | Auth / Basic or Bearer | None | 200 empty JSON object | 401 string |
| `POST /v1/auth/logout` | `logoutHandler` | Auth / Basic or Bearer | None | 200 empty JSON object | 401 string |
| `GET /v1/auth/refresh` | `refreshHandler` | Auth / Basic or Bearer | None | 200 `auth.AuthResponse`; `Authorization` header | 401 or 500 string |
| `GET /v1/approvals` | `approvalsHandler` | Auth / Basic or Bearer | None | 200 array of `types.Approval` | 401 or 500 string |
| `POST /v1/approvals` | `approvalApproveHandler` | Auth / Basic or Bearer | `approveRequest` | 200 `types.Approval` or JSON null for delete | 400, 401, 404, 500 string |
| `PUT /v1/approvals` | `approvalSetHandler` | Auth / Basic or Bearer | `resourceApprovalsUpdateRequest` | 200 `APIResponse` | 400, 401, 403, 404, 500 string |
| `GET /v1/resources` | `resourcesHandler` | Auth / Basic or Bearer | None | 200 array of `resource` | 401 string |
| `PUT /v1/policies` | `policyUpdateHandler` | Auth / Basic or Bearer | `resourcePolicyUpdateRequest` | 200 `APIResponse` | 400, 401, 403, 404, 500 string |
| `GET /v1/tracked` | `trackedHandler` | Auth / Basic or Bearer | None | 200 array of `trackedImage` | 401, 403, 500 string |
| `PUT /v1/tracked` | `trackSetHandler` | Auth / Basic or Bearer | `trackRequest` | 200 `APIResponse` | 400, 401, 403, 404, 500 string |
| `GET /v1/audit` | `adminAuditLogHandler` | Auth / Basic or Bearer | Query: `filter`, `email`, `limit`, `offset` | 200 `auditLogsResponse` | 401 or 500 string |
| `GET /v1/stats` | `statsHandler` | Auth / Basic or Bearer | None | 200 array of `types.AuditLogStats` | 401, 403, 500 string |
| `POST /v1/webhooks/native` | `nativeHandler` | Webhook auth | `types.Repository` | 200 empty | 400 string; conditional 401 |
| `POST /v1/webhooks/dockerhub` | `dockerHubHandler` | Webhook auth | `dockerHubWebhook` | 200 empty | 400 string; conditional 401 |
| `POST /v1/webhooks/jfrog` | `jfrogHandler` | Webhook auth | `jfrogWebhook` | 200 empty | 400 string; conditional 401 |
| `POST /v1/webhooks/quay` | `quayHandler` | Webhook auth | `quayWebhook` | 200 empty | 400 string; conditional 401 |
| `POST /v1/webhooks/azure` | `azureHandler` | Webhook auth | `azureWebhook` | 200 empty | 400 string; conditional 401 |
| `POST /v1/webhooks/github` | `githubHandler` | Webhook auth | `X-GitHub-Event`; matching GitHub package payload | 200 empty | 400 string; conditional 401 |
| `POST /v1/webhooks/harbor` | `harborHandler` | Webhook auth | `harborWebhook` | 200 empty | 400 string; conditional 401 |
| `POST /v1/webhooks/registry` | `registryNotificationHandler` | Always / public, including when webhook auth is enabled | `registryNotification` | 200 empty | 400 string |

## Exclusions

| Route surface | Decision and rationale |
|---|---|
| Every registered `OPTIONS` method | Exclude: global CORS middleware answers preflight before handlers; these are not client operations. |
| `/metrics` | Exclude: Prometheus-owned text exposition, not Keel's JSON API. |
| `/css/`, `/assets/`, `/js/`, `/img/`, `/loading/`, and `/` catch-all | Exclude: conditional static frontend delivery, not API contracts. |
| `/debug/vars` and `/debug/pprof/*` | Exclude: DEBUG-only expvar/Go pprof diagnostics, not supported client API. The coverage test still observes and allowlists them. |

## Security and Operation Naming

Swagger 2 definitions are `basicAuth` (`type: basic`) and `bearerAuth` (`type: apiKey`, header `Authorization`, with the `Bearer <token>` convention). Admin operation security is an OR between them. The seven configurable webhooks use optional alternatives—anonymous, Basic, or Bearer—to reflect deployment configuration; the operation description and a Keel extension record when auth is enforced. Registry notifications and public operations have no security requirement. No cookie scheme or default credentials mode is added.

Every annotation has a verb-first, resource-specific `@ID` such as `getVersion`, `getAuthInfo`, `getAuthUser`, `logoutAuthGet`, `logoutAuthPost`, `listApprovals`, `updateApprovals`, and `receiveDockerHubWebhook`. IDs are reviewed in the route-coverage fixture and are never inferred from Go handler names or reordered paths.

## JavaScript Client

Use the maintained OpenAPI Generator `javascript` client: npm wrapper `@openapitools/openapi-generator-cli` exactly `2.20.2` and engine exactly `7.22.0` in `ui/openapitools.json`. The wrapper supports the repository's Node 16 CI, the engine requires Java 11+, and its stable JavaScript generator emits Babel-compatible ES modules and Promise APIs. Configure `usePromises=true`, `useURLSearchParams=true`, stable model/property naming, and a fixed project version; disable generated timestamps, docs, and sample tests.

Generate owned sources directly into `ui/src/api/generated/` and commit them. The existing handwritten `ui/src/api/index.js` stays in place and current screens are not migrated. Generated `ApiClient`, API classes, model classes, and barrel exports use the UI's root dependency `superagent` pinned to `5.3.1` and locked by Yarn; unnecessary nested package/build metadata is not emitted. The base URL remains caller-configurable. Generated auth hooks set Basic credentials or the full Bearer `Authorization` header, permit documented custom headers such as `X-GitHub-Event`, and expose login/refresh response headers. Cookies are neither required nor enabled as a Keel auth mechanism.

## Commands and Verification

Contributor prerequisites are Go 1.23, Node 16 with Yarn 1, and Java 11+. The documented clean-checkout sequence is:

```bash
go mod download
(cd tools && go mod download)
yarn --cwd ui install --frozen-lockfile
make api-generate api-validate api-coverage api-client-test
git diff --exit-code
```

Repository targets are:

- `make api-spec`: run the tooling-module `swag` command and emit deterministic JSON/YAML only.
- `make api-client`: invoke the local Yarn OpenAPI wrapper with engine `7.22.0` and regenerate the owned client directory from `swagger.json`.
- `make api-generate`: run spec then client generation.
- `make api-validate`: run pinned OpenAPI Generator `validate` against JSON and YAML.
- `make api-coverage`: run the focused Go test that walks Gorilla Mux and compares method/path/operation ID to the spec and exclusion manifest.
- `make api-client-test`: run a Jest smoke import plus existing relevant `yarn lint` and `yarn build` checks.
- `make api-check`: validate, cover, test, regenerate, then run `git diff --exit-code` over specs, client output, generator configuration, dependencies, and lockfiles.

The route test creates server configurations with auth enabled and DEBUG enabled, walks registered Gorilla Mux routes, removes OPTIONS, and requires exactly the 25 table entries. Every remaining route must match the recorded exclusions. CI runs `make api-check`, focused `pkg/http` tests, and `go test ./...`; external-service limitations are surfaced in the job output and contributor documentation rather than hidden.
