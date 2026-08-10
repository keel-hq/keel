# Prevent Keel startup probe crash loops

## Summary
Add an enabled-by-default, configurable Helm startup probe so Kubernetes defers liveness and readiness checks until Keel is accepting health requests. Route the probe through oauth2-proxy in external-proxy mode, document the new values, add Helm rendering coverage, and extend the native-k3s release validator to assert the live startup probe. Add repository guidance requiring deployed local-k3s validation for future startup and Helm chart changes.

## Testing
- `go test ./chart/keel`
- `bash -n .test/e2e-k3s.sh`
- `make release-validate` using k3s v1.35.6 with Keel deployed: fresh default and external-proxy installs, `/healthz`, `/version`, probe wiring, upgrade from 0.20.0, rollback, persistence, cleanup, polling, webhook, and OAuth scenarios all passed.
