# Generate Keel API docs and JavaScript client

## Summary

Document Keel's 25 HTTP API operations from source and add a reproducibly generated JavaScript client for future frontend use.

## Changes

- Add complete swag annotations, exact wire models, and canonical Swagger YAML.
- Add pinned spec/client generation, Spectral validation, and clean-diff checks.
- Commit the generated ES-module client with focused route, security, and client contract tests.
- Document artifact ownership and contributor regeneration commands.

## Testing

- Focused route, conditional-security, and OpenAPI contract tests pass.
- Spectral 6.16.1 reports no errors; clean generation produces no diff.
- Generated-client Jest smoke tests, existing UI lint, and production build pass under Node 16.20.
- `go test ./... -timeout=2m` is environment-limited because this workspace has `CGO_ENABLED=0` and no C compiler for existing SQLite-backed tests; those tests report that `go-sqlite3 requires cgo to work` and time out.
