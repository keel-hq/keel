# Keel API Artifacts

`swagger.yaml` is generated from the annotations in `cmd/keel` and `pkg/http` by `swag` v1.16.6. Do not edit it directly.

Regenerate it from the repository root with:

```sh
make api-spec
```

Swagger JSON and `docs.go` are intentionally not generated or linked into the Keel runtime. The repository does not serve this specification or a Swagger UI over HTTP.

`openapi-generator-config.yaml` and `openapi-generator-ignore` configure OpenAPI Generator v7.22.0. `make api-client` replaces `ui/src/api/generated` with the ES-module client generated from `swagger.yaml`; ignored package scaffolding, tests, docs, and generator metadata remain owned by the existing UI project or are redundant.
