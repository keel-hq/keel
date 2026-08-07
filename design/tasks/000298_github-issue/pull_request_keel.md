# Update gRPC to remediate CVE-2026-33186

## Summary
Upgrade google.golang.org/grpc to v1.82.1, which fixes CVE-2026-33186 / GO-2026-4762 and the subsequently disclosed GO-2026-6061. Update the Pub/Sub keepalive dialer to the supported context-aware API while preserving TCP keepalive behavior, and add coverage proving that the configured dial option establishes a connection.

The vulnerable GO-2026-4762 server symbols were not reachable in Keel: Keel uses gRPC as a Pub/Sub client and does not construct or serve a gRPC server.

## Testing
- `go test ./trigger/pubsub` passed.
- `go test ./...` passed.
- `go build ./cmd/keel` passed.
- `go vet ./trigger/pubsub` passed.
- `gofmt`, `git diff --check`, `go mod tidy` consistency, `go mod verify`, and `go list -m all` passed.
- `govulncheck` no longer reports GO-2026-4762 or GO-2026-6061. Unrelated pre-existing findings remain.
- Full-repository `go vet ./...` continues to report pre-existing unkeyed Kubernetes literals and unexported JSON-tagged fields outside this change.