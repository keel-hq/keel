# Keel API Artifacts

`swagger.yaml` is generated from the annotations in `cmd/keel` and `pkg/http` by `swag` v1.16.6. Do not edit it directly.

## Prerequisites

- Go 1.26.5
- GNU Make
- Docker (for OpenAPI Generator v7.22.0 and Spectral v6.16.1)
- Node.js 16.20 and Yarn 1.x for UI verification

No global `swag`, OpenAPI Generator, or Spectral installation is used.

## Generate and Validate

Run from the repository root:

```sh
make api-spec
make api-client
make api-generate
make api-validate
make api-check
```

`api-generate` replaces both generated artifacts. `api-validate` lints the spec and prints the 25 router-operation mappings and exclusions. `api-check` regenerates everything and fails unless `docs/swagger.yaml` and `ui/src/api/generated` have a clean Git diff.

Run the client and existing project checks with:

```sh
cd ui
yarn install --frozen-lockfile
yarn run test:generated-api
yarn run lint:nofix
yarn run build
cd ..
go test ./pkg/http
go test ./...
```

Some Go tests use CGO-backed SQLite and require `CGO_ENABLED=1` plus a C compiler. Report unavailable external services or toolchain limitations; do not skip failures in committed automation.

## Ownership and Updates

Swagger JSON and `docs.go` are intentionally not generated or linked into the Keel runtime. The repository does not serve this specification or a Swagger UI over HTTP.

`openapi-generator-config.yaml` and `openapi-generator-ignore` configure OpenAPI Generator v7.22.0. `make api-client` replaces `ui/src/api/generated` with the ES-module client generated from `swagger.yaml`; ignored package scaffolding, tests, docs, and generator metadata remain owned by the existing UI project or are redundant.

Handler annotations and documentation models are human-owned. `swagger.yaml` and every file under `ui/src/api/generated` are generator-owned and retain generated notices; never edit them manually. Regenerate and commit the spec and client together whenever a route, wire model, annotation, or tool configuration changes. Tool upgrades require an explicit pinned version change and a successful `make api-check`.

The client defaults to same-origin URLs and disabled cookies. Configure `ApiClient.authentications.BasicAuth` or set `BearerAuth.apiKey` to the complete `Bearer <token>` header value at runtime; never commit credentials or a deployment host.
