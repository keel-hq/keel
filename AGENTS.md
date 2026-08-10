# Repository Instructions

## Validation

- For changes that affect application startup or the Helm chart, always run `make release-validate` before considering the work complete. This must exercise the repository's local k3s harness with Keel deployed; unit tests or rendered-manifest checks alone are not sufficient.
- Confirm that the Keel Deployment rolls out, its health probes succeed, and the k3s end-to-end suite passes. If the environment prevents this validation, report the exact blocker and do not claim that validation passed.
