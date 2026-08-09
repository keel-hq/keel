# Centralize Keel configuration and stabilize behavioral CI

## Summary
Move Keel's environment configuration into typed structures loaded through envconfig and inject those settings into authentication, bots, notifications, providers, triggers, and Kubernetes watchers. Preserve upgrade compatibility for explicitly empty values and legacy Pub/Sub flags, reject undocumented nested aliases, keep notifier credentials scoped away from application secrets, and make typed DEBUG and polling behavior consistent. Stabilize the k3s behavioral job by giving the Go suite a 15-minute deadline while ensuring best-effort cleanup cannot overwrite the test result.

## Testing
Ran `go test ./...`, focused configuration/authentication/notification tests, and `go mod tidy`; all passed. Built the branch image and installed it into a fresh local k3s cluster, where the packaged chart install/upgrade/rollback validation passed. Verified a deployment with explicitly empty boolean and integer environment variables became Ready with zero restarts and default polling enabled, then rolled it to `POLL=false` and confirmed it remained Ready without starting the polling trigger. `go vet ./...` reports only pre-existing unkeyed Kubernetes literals and JSON tags on unexported fields.
