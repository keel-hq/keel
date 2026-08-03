# Implementation Tasks: Document the Keel API and Generate a Reproducible JavaScript Client

- [~] Add the checked-in 25-operation/exclusion inventory and focused tests that pin current routes, conditions, payloads, statuses, and registry webhook authentication behavior.
- [ ] Name or add exact documentation models for handler-local requests, responses, nested provider webhook payloads, GitHub variants, and empty/nullable wire formats without changing behavior.
- [ ] Add general Swagger metadata/security definitions and complete, uniquely identified `swag` annotations for all 25 operations, using forwarding handlers only for shared auth implementations.
- [ ] Add pinned `swag` v1.16.6 Make generation for canonical `docs/swagger.yaml` only, plus generated-artifact notices and ignore policy for JSON and `docs.go`.
- [ ] Add pinned OpenAPI Generator v7.22.0 JavaScript configuration/ignore files and deterministic generation into `ui/src/api/generated`.
- [ ] Pin SuperAgent 10.3.0 and Spectral CLI v6.16.1 in the existing UI package and committed lockfiles.
- [ ] Generate and commit the canonical Swagger YAML and ES-module Promise client with stable operation names, same-origin base URL, explicit auth configuration, and no redundant SDK scaffolding.
- [ ] Add the generated-client Jest smoke/contract test and run it in the existing Node 16/Vue 2/Babel environment.
- [ ] Add the pinned Swagger lint/validation, 25-operation mapping/exclusion report, deterministic clean-generation check, and corresponding CI coverage.
- [ ] Document prerequisites, artifact ownership/versioning, and exact generation, validation, client smoke-test, UI lint/build, focused Go test, full Go test, and clean-diff commands.
- [ ] Run all verification commands, record any genuine environment limitations, confirm only generated/docs/tooling changes are present, and verify no API, security, UI, runtime, or deployment behavior changed.
