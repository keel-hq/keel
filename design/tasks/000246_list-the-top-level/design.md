# Design: Summarise Top-Level Directories of the Keel Repository

## Approach

This is a pure read-only inspection task — there is nothing to build. The
"implementation" is running `ls` / `find` over the repository root and reporting
the result. No script, no generator, no tooling.

Method used: `ls -la` at the root, then `ls <dir> | head` and
`find <dir> -maxdepth 2 -type d` for each entry, cross-checked against the
repo's own `ARCHITECTURE.md` (which contains a partial directory table) and
`readme.md`.

## Result — Keel top-level directories

Repository: `/home/retro/work/keel/` — `github.com/keel-hq/keel`, Go + Vue.js.

### Application code (Go)

| Directory | Contents |
|-----------|----------|
| `cmd/` | Entry point. `cmd/keel/main.go` wires everything together — providers, triggers, notification senders (via blank imports), HTTP server. Best file to read first after `types/`. |
| `types/` | Core domain types shared across the codebase: `Repository`, `Event`, `Approval`, policy/trigger/notification enums (several `*_jsonenums.go` are generated). |
| `provider/` | Deployment handlers implementing the `Provider` interface (`Submit`, `TrackedImages`, `GetName`, `Stop`). `kubernetes/` covers Deployments, StatefulSets, DaemonSets, CronJobs; `helm3/` covers Helm v3 releases. |
| `trigger/` | Event sources that detect new image versions. `poll/` periodically scans registries; `pubsub/` consumes Google Cloud Pub/Sub for GCR. Webhook triggers live in `pkg/http/` instead. |
| `registry/` | Docker Registry v2 API client (`registry.go`, `registry/docker/`) used to list tags and resolve digests. |
| `approvals/` | Manual approval workflow — the `Manager` that creates, approves and rejects pending updates when `keel.sh/approvals` is set. |
| `bot/` | Chat bots for interacting with approvals from Slack and HipChat, plus a shared message `formatter/`. |
| `secrets/` | Extracts container-registry credentials from Kubernetes image pull secrets (`secrets.go`, `match.go`). |
| `extension/` | Pluggable extension points registered by name: `notification/` (Slack, Teams, Discord, Mattermost, HipChat, mail, generic webhook, auditor), `credentialshelper/` (AWS ECR, GCR, K8s secrets), `approval/`. |
| `pkg/` | Public-ish supporting packages: `http/` (REST API, UI serving, and all registry webhook triggers), `auth/` (basic auth + JWT), `store/` (persistence, `store/sql/` SQLite). |
| `internal/` | Internal-only packages: `policy/` (semver / force / glob / regexp version matching), `k8s/` (watchers, resource cache), `workgroup/` (goroutine lifecycle grouping). |
| `util/` | Small shared helpers: `image/` (image reference parsing), `policies/`, `codecs/`, `templates/`, `timeutil/`, `stopper/`, `version/`, `testing/`. |
| `constants/` | Environment-variable names and other config constants in one file. |
| `version/` | Build metadata (version, git revision) stamped in at link time. |

### Frontend

| Directory | Contents |
|-----------|----------|
| `ui/` | Vue.js web dashboard — `src/` (api, views, components, store, router, layouts), `public/`, `tests/unit/`, `docs/`, plus its own `package.json` / `yarn.lock` and Vue CLI config. Built assets are served by `pkg/http/`. |

### Deployment & packaging

| Directory | Contents |
|-----------|----------|
| `chart/` | Helm chart used to deploy Keel itself (`chart/keel/` + `templates/`). |
| `deployment/` | Plain Kubernetes manifests — `deployment-template.yaml` and a README, for installing without Helm. |

### Tests, tooling and assets

| Directory | Contents |
|-----------|----------|
| `tests/` | End-to-end / acceptance tests (`acceptance_test.go`, `acceptance_polling_test.go`, `helpers.go`) that exercise Keel against a real cluster. Unit tests live next to their packages as `*_test.go`. |
| `scripts/` | Developer helper scripts — currently `start-local-cluster.sh`. |
| `static/` | Project images (logo, screenshots) referenced by the readme and site. |

### Dot-directories (tooling / CI)

| Directory | Contents |
|-----------|----------|
| `.github/` | GitHub Actions workflows. |
| `.pipeline/` | Google Cloud Build config (`cloudbuild.yaml`). |
| `.scripts/` | Helm-repo publishing scripts — `gen_packages.sh`, `index_repo.sh`, `repo-sync.sh`. |
| `.test/` | Chart-testing config (`ct.yaml`) and the kind-based e2e runner (`e2e-kind.sh`). |
| `.dependabot/` | Legacy Dependabot v1 `config.yml`. |

Root files worth knowing: `ARCHITECTURE.md` (agent-oriented codebase guide),
`readme.md`, `Makefile` / `build.ps1`, the several `Dockerfile*` and `compose*.yml`
variants (debug, tests, local, debian), `azure-pipelines.yml`, `values.yaml`,
`go.mod` / `go.sum`.

## Key Decisions

- **No artefact generated.** The request is informational. Producing a script or
  a committed `DIRECTORIES.md` inside `keel` would violate the "do not create
  files" constraint and duplicate the existing `ARCHITECTURE.md`.
- **Answer captured here.** The table above is the deliverable, stored in
  `helix-specs` so it survives the session and future agents can reuse it.
- **Scope resolved to `keel`.** It is flagged as the primary project; `b-alex`
  was left untouched. Flagged in Open Questions.

## Learnings for future agents

- Keel already contains `ARCHITECTURE.md` — a well-maintained, agent-oriented
  guide with a directory table, the `Provider`/`Event` interfaces, the full list
  of `keel.sh/*` annotations, environment variables and the data-flow sequence.
  **Read it before exploring manually.** Its directory table is not exhaustive
  though: it omits `deployment/`, `tests/`, `scripts/`, `static/` and every
  dot-directory.
- Architectural shape to expect: `trigger/` → `types.Event` → `provider/`
  (policy check in `internal/policy/` → optional approval in `approvals/`) →
  resource patch → `extension/notification/` fan-out.
- Registry **webhook** triggers are not in `trigger/` — they live in
  `pkg/http/*_webhook_trigger.go`. Easy to miss.
- Extensions (notification senders, credentials helpers) use a
  register-by-name pattern and are activated by blank imports in
  `cmd/keel/main.go`; adding one means touching that file too.
- Several files under `types/` are code-generated (`*_jsonenums.go`) — edit the
  source enum, not the generated file.
