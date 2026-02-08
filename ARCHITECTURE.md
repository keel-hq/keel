# Keel Architecture Guide

> A comprehensive guide for AI agents and developers to understand and navigate the Keel codebase.

## Quick Start - Read These First

If you're new to this codebase, read these files in order:

1. `types/types.go` - Core domain types (Repository, Event, Policy, TriggerType)
2. `cmd/keel/main.go` - Application entry point, shows how everything connects
3. `provider/provider.go` - Provider interface definition
4. `trigger/poll/watcher.go` - Example trigger implementation

## What is Keel?

Keel is a **Kubernetes deployment automation tool** written in Go. It watches container registries for new image versions and automatically updates Kubernetes deployments based on configured policies.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              TRIGGERS                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────────┐  │
│  │ Poll Trigger │  │ PubSub (GCR) │  │ Webhooks (DockerHub, etc.)   │  │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┬───────────────┘  │
└─────────┼─────────────────┼──────────────────────────┼──────────────────┘
          │                 │                          │
          └─────────────────┴────────────┬─────────────┘
                                         │
                                         ▼
                               ┌─────────────────┐
                               │     Event       │
                               │  (Repository +  │
                               │   new version)  │
                               └────────┬────────┘
                                        │
                    ┌───────────────────┴───────────────────┐
                    │                                       │
                    ▼                                       ▼
          ┌─────────────────┐                    ┌─────────────────┐
          │   Kubernetes    │                    │     Helm3       │
          │    Provider     │                    │    Provider     │
          └────────┬────────┘                    └────────┬────────┘
                   │                                      │
                   ▼                                      ▼
          ┌─────────────────┐                    ┌─────────────────┐
          │ Update Deployment│                   │ Update Release  │
          │ (if policy match)│                   │ (if policy match)│
          └─────────────────┘                    └─────────────────┘
```

## Directory Structure

| Directory | Purpose | Key Files |
|-----------|---------|-----------|
| `cmd/keel/` | **Entry point** - Application startup, wiring | `main.go` |
| `provider/` | **Deployment handlers** - Update K8s/Helm resources | `provider.go`, `kubernetes/`, `helm3/` |
| `trigger/` | **Event sources** - Detect new image versions | `poll/`, `pubsub/` |
| `pkg/http/` | **HTTP server + webhooks** - REST API, registry webhooks | `http.go`, `*_webhook_trigger.go` |
| `types/` | **Domain types** - Core data structures | `types.go` |
| `internal/policy/` | **Version matching** - Semver, glob, force, regexp | `policy.go`, `semver.go` |
| `extension/` | **Plugins** - Notifications, credentials helpers | `notification/`, `credentialshelper/` |
| `approvals/` | **Approval workflow** - Manual approval before updates | `approvals.go` |
| `bot/` | **Chat bots** - Slack/HipChat for approvals | `bot.go`, `slack/`, `hipchat/` |
| `registry/` | **Registry client** - Docker registry API | `registry.go` |
| `secrets/` | **K8s secrets** - Extract registry credentials | `secrets.go` |
| `ui/` | **Web dashboard** - Vue.js frontend | `src/` |
| `pkg/store/` | **Persistence** - SQLite database | `sql/` |
| `pkg/auth/` | **Authentication** - Basic auth, JWT | |
| `internal/k8s/` | **K8s utilities** - Watchers, resource cache | |
| `chart/` | **Helm chart** - Deploy Keel itself | |
| `constants/` | **Environment variables** - Config constants | |
| `version/` | **Build info** - Version, revision | |
| `util/` | **Utilities** - Image parsing, etc. | |

## Core Concepts

### 1. Providers

Providers handle deployment updates for different platforms. They implement the `Provider` interface:

```go
// provider/provider.go
type Provider interface {
    Submit(event types.Event) error      // Process an update event
    TrackedImages() ([]*types.TrackedImage, error)  // List monitored images
    GetName() string
    Stop()
}
```

**Available providers:**
- `provider/kubernetes/` - Native Kubernetes Deployments, StatefulSets, DaemonSets, CronJobs
- `provider/helm3/` - Helm v3 releases (enabled via `HELM3_PROVIDER=true`)

### 2. Triggers

Triggers detect new image versions and emit `Event` objects:

```go
// types/types.go
type Event struct {
    Repository  Repository  // Image info (host, name, tag, digest)
    CreatedAt   time.Time
    TriggerName string      // "poll", "pubsub", "webhook", etc.
}
```

**Available triggers:**
- `trigger/poll/` - Periodically polls registries for new tags
- `trigger/pubsub/` - Google Cloud Pub/Sub for GCR events
- `pkg/http/*_webhook_trigger.go` - Webhooks from DockerHub, Azure, GitHub, Harbor, Quay, JFrog

### 3. Policies

Policies determine which version updates are allowed. Set via `keel.sh/policy` annotation:

```go
// internal/policy/policy.go - Policy types
type PolicyType int
const (
    PolicyTypeNone PolicyType = iota
    PolicyTypeSemver  // major, minor, patch, all
    PolicyTypeForce   // always update (for :latest)
    PolicyTypeGlob    // glob pattern matching
    PolicyTypeRegexp  // regex pattern matching
)
```

**Policy examples:**
- `keel.sh/policy: major` - Allow major version bumps (1.x.x → 2.x.x)
- `keel.sh/policy: minor` - Allow minor bumps (1.1.x → 1.2.x)
- `keel.sh/policy: patch` - Allow patch bumps only (1.1.1 → 1.1.2)
- `keel.sh/policy: force` - Always update (for mutable tags like `latest`)
- `keel.sh/policy: glob:release-*` - Match glob patterns

### 4. Notifications

Extensible notification system using sender registration pattern:

```go
// extension/notification/notification.go
func RegisterSender(name string, s Sender) { ... }
```

**Available senders:** Slack, Teams, Discord, Mattermost, HipChat, Mail, Webhook, Auditor

Notifications are registered via blank imports in `cmd/keel/main.go`:
```go
_ "github.com/keel-hq/keel/extension/notification/slack"
```

### 5. Approvals

Manual approval workflow before updates proceed:

```go
// approvals/approvals.go
type Manager interface {
    Create(r *types.Approval) error
    Approve(id, voter string) (*types.Approval, error)
    Reject(id string) (*types.Approval, error)
    // ...
}
```

Set via `keel.sh/approvals: "2"` annotation to require N approvals.

## Data Flow

1. **Trigger detects new version** → Creates `types.Event`
2. **Event submitted to Providers** → `provider.Submit(event)`
3. **Provider checks policies** → `internal/policy/` evaluates if update allowed
4. **Approval check** → If approvals required, waits for manual approval
5. **Deployment updated** → Provider patches K8s resource or Helm release
6. **Notifications sent** → Slack/webhook/etc. notified of update

## Key Annotations

| Annotation | Purpose | Example |
|------------|---------|---------|
| `keel.sh/policy` | Update policy | `minor`, `patch`, `force`, `glob:v*` |
| `keel.sh/trigger` | Trigger type | `poll` (default: webhooks) |
| `keel.sh/pollSchedule` | Poll frequency | `@every 5m` |
| `keel.sh/approvals` | Required approvals | `2` |
| `keel.sh/approvalDeadline` | Approval timeout (hours) | `24` |
| `keel.sh/notify` | Override notification channel | `#deployments` |
| `keel.sh/matchTag` | Force tag matching | `true` |
| `keel.sh/matchPreRelease` | Match pre-release versions | `true` |
| `keel.sh/digest` | Track by digest (internal) | SHA256 digest |
| `keel.sh/imagePullSecret` | Registry credentials secret | `my-registry-secret` |
| `keel.sh/releaseNotes` | Release notes URL | `https://...` |
| `keel.sh/initContainers` | Track init containers | `true` |

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `PUBSUB` | Enable GCR Pub/Sub trigger | (disabled) |
| `POLL` | Enable/disable poll trigger | `1` (enabled) |
| `PROJECT_ID` | GCP project for Pub/Sub | |
| `HELM3_PROVIDER` | Enable Helm3 provider | `false` |
| `DEBUG` | Enable debug logging | `false` |
| `NOTIFICATION_LEVEL` | Min notification level | `info` |
| `BASIC_AUTH_USER` | HTTP basic auth username | |
| `BASIC_AUTH_PASSWORD` | HTTP basic auth password | |
| `AUTHENTICATED_WEBHOOKS` | Require auth for webhooks | `false` |
| `DOCKER_REGISTRY_CFG` | Default registry credentials | |
| `XDG_DATA_HOME` | Data directory (SQLite) | `/data` |
| `UI_DIR` | Web UI static files | `www` |
| `KUBERNETES_CONFIG` | Kubeconfig path | `~/.kube/config` |
| `POLL_DEFAULTSCHEDULE` | Default poll interval | `@every 1m` |

## Extension Points

### Adding a New Notification Sender

1. Create `extension/notification/mynotifier/mynotifier.go`
2. Implement `notification.Sender` interface
3. Register via `init()`:
   ```go
   func init() {
       notification.RegisterSender("mynotifier", &sender{})
   }
   ```
4. Add blank import in `cmd/keel/main.go`:
   ```go
   _ "github.com/keel-hq/keel/extension/notification/mynotifier"
   ```

### Adding a New Webhook Trigger

1. Create `pkg/http/myregistry_webhook_trigger.go`
2. Parse the webhook payload, extract repository/tag info
3. Create `types.Event` and call `providers.Submit(event)`
4. Register route in `pkg/http/http.go`

### Adding a New Provider

1. Create `provider/myprovider/`
2. Implement `provider.Provider` interface
3. Initialize in `cmd/keel/main.go` `setupProviders()`

### Adding a New Credentials Helper

1. Create `extension/credentialshelper/myhelper/`
2. Implement `credentialshelper.CredentialsHelper` interface
3. Register via `init()` and blank import in `main.go`

## Building & Running

```bash
# Build
make build

# Run locally (outside cluster)
make run

# Run tests
make test

# Build Docker image
make image
```

## Common Tasks

| Task | Where to Look |
|------|---------------|
| Add new webhook support | `pkg/http/*_webhook_trigger.go` |
| Change version matching logic | `internal/policy/` |
| Modify K8s update behavior | `provider/kubernetes/` |
| Add notification channel | `extension/notification/` |
| Change polling behavior | `trigger/poll/` |
| Modify approval workflow | `approvals/`, `bot/` |
| Add registry authentication | `extension/credentialshelper/` |
| Parse image references | `util/` |
| HTTP API endpoints | `pkg/http/` |

## Testing

```bash
# Unit tests
make test

# E2E tests (requires running cluster)
make e2e
```

Test files follow Go convention: `*_test.go` alongside source files.

## Frontend (UI)

The web dashboard is a Vue.js application in `ui/`:

```bash
cd ui
yarn install
yarn run serve  # Development
yarn run build  # Production build
```

Built assets go to `ui/dist/`, served by Keel's HTTP server.