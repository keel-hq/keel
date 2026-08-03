# Design: Summarize the Keel Repository's Top-Level Directories

## Approach

Enumerate the immediate tracked directories in the primary `keel` repository, then inspect representative files up to two levels deep. Prefer directory READMEs where present and use filenames, package names, and manifests elsewhere. Return one concise, alphabetical list; no code, generated artifact, or repository documentation is needed.

## Repository Findings

| Directory | Contents |
| --- | --- |
| `.dependabot` | Dependabot dependency-update configuration. |
| `.github` | GitHub Actions workflows for CI and chart releases. |
| `.pipeline` | Cloud Build pipeline configuration. |
| `.scripts` | Repository indexing, synchronization, and package-generation helpers. |
| `.test` | Chart-testing configuration and a kind-based end-to-end test script. |
| `approvals` | Approval logic and its Go tests. |
| `bot` | Notification bot integrations, formatting, and Slack/HipChat support. |
| `chart` | The Keel Helm chart, values, metadata, and chart documentation. |
| `cmd` | The `keel` executable entry point. |
| `constants` | Shared Go constants. |
| `deployment` | Example Kubernetes deployment manifests for installations without Helm. |
| `extension` | Extension interfaces and helpers for approvals, credentials, and notifications. |
| `internal` | Internal Kubernetes translation/cache, policy, and workgroup packages. |
| `pkg` | Reusable application packages, including authentication and HTTP endpoints. |
| `provider` | Helm and Kubernetes deployment-provider implementations. |
| `registry` | Container-registry abstractions and Docker registry client logic. |
| `scripts` | Local development scripts, currently for starting a local cluster. |
| `secrets` | Secret matching and management logic with tests. |
| `static` | Keel logos and UI screenshots. |
| `tests` | Repository-level acceptance tests and helpers. |
| `trigger` | Polling and Pub/Sub update-trigger implementations. |
| `types` | Shared domain types, enums, and version information. |
| `ui` | The Vue 2/Ant Design web interface, assets, configuration, and frontend tests. |
| `util` | Shared utility packages for images, policies, templates, timing, testing, and versions. |
| `version` | Keel version constants. |

## Key Decisions

- Treat `keel` as “this repository” because it is identified as the primary project.
- Include tracked dot-directories because they contain project configuration.
- Exclude `.git` because it is local version-control metadata and is not tracked project content.
- Keep implementation read-only and deliver the inventory in the response rather than adding a file to a code repository.

## Verification

Compare the final names with the sorted first path component of `git ls-files`, confirm there are 25 entries, and check both code repositories remain clean.
