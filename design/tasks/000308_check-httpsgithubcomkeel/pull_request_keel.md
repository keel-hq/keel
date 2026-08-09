# Centralize Keel configuration and stabilize behavioral CI

## Summary
Move Keel's environment configuration into typed structures loaded through envconfig and inject those settings into authentication, bots, notifications, providers, triggers, and Kubernetes watchers. Preserve existing integration behavior with focused configuration tests, and stabilize the k3s behavioral job by giving the Go suite a 15-minute deadline while ensuring best-effort cleanup cannot overwrite the test result.

## Testing
Ran the complete k3s behavioral suite successfully, including OAuth, polling, webhook, and StatefulSet isolation scenarios. Also ran shell syntax validation, `go test ./tests`, and verified the final harness removes its disposable run directory without changing the successful exit status.
