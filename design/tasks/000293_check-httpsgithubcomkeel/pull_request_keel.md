# Migrate AWS ECR and Google Pub/Sub integrations to supported SDKs

## Summary
Migrate the ECR credentials helper from AWS SDK for Go v1 to the modular v2 SDK, and migrate the Google Cloud Pub/Sub trigger from v1 to v2. Pub/Sub topic and subscription management now uses the v2 admin clients with explicit NotFound and AlreadyExists handling, while preserving the previous ten-stream receive concurrency. Retired SDK dependencies are removed and module dependencies are updated.

## Testing
- `go test ./extension/credentialshelper/aws ./trigger/pubsub -run 'TestCallback|TestDecode|TestParse' -count=1 -timeout=90s`
- `go vet ./extension/credentialshelper/aws ./trigger/pubsub`
- `go test ./... -run '^$'`
- `git diff --check`

All checks passed.
