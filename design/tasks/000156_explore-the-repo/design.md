# Design: ARCHITECTURE.md Documentation

## Overview

Create an `ARCHITECTURE.md` file in the root of the keel repository that serves as a comprehensive guide for AI agents and developers to understand the codebase structure and locate relevant code quickly.

## Architecture Observations

### Project Type
Keel is a Go application (Go 1.23) that automates Kubernetes deployment updates. It's a stateless service that watches for container image updates and automatically updates deployments.

### Core Abstractions Discovered

1. **Providers** (`provider/`) - Handle deployment updates for different platforms
   - `kubernetes/` - Native Kubernetes deployments
   - `helm3/` - Helm v3 releases
   - Interface defined in `provider/provider.go`

2. **Triggers** (`trigger/`) - Detect new image versions
   - `poll/` - Periodically polls registries for new tags
   - `pubsub/` - Google Cloud Pub/Sub for GCR events

3. **Webhooks** (`pkg/http/`) - Receive push notifications from registries
   - DockerHub, Azure, GitHub, Harbor, Quay, JFrog, native webhooks

4. **Policies** (`internal/policy/`) - Version matching strategies
   - semver, force, glob, regexp

5. **Extensions** (`extension/`) - Pluggable components
   - `notification/` - Slack, Teams, Discord, etc.
   - `credentialshelper/` - AWS ECR, GCR auth
   - `approval/` - Manual approval workflows

### Entry Point
- `cmd/keel/main.go` - Single entry point, wires all components together

### Key Types
- `types/types.go` - Core domain types (Repository, Event, Policy, etc.)

## Design Decisions

### Approach: Single Comprehensive Document
Create one `ARCHITECTURE.md` file rather than scattered documentation because:
- AI agents benefit from a single file they can read in full
- Reduces context switching during exploration
- Easier to keep in sync with codebase

### Content Structure
1. Quick Start section for immediate orientation
2. Directory reference with purposes
3. Architectural concepts explained
4. Data flow diagrams (text-based)
5. Extension guide for common modifications
6. Key files to read first

### Format Choices
- Use plain Markdown (no special tooling required)
- ASCII diagrams for flows (portable, git-friendly)
- Link to actual source files for deeper exploration
- Include grep-able keywords for common tasks

## Dependencies

None - this is a documentation-only change.

## Risks

- Documentation may drift from code over time
- Mitigate: Include instructions for keeping it updated

## Implementation Notes

### Approach Taken
- Created a single comprehensive `ARCHITECTURE.md` file in the repository root
- Structured for AI agents: starts with "Quick Start - Read These First" section
- Used ASCII diagrams for data flow (git-friendly, portable)
- Included tables for quick reference (directories, annotations, env vars)

### Key Discoveries During Implementation
- The codebase uses a clean plugin pattern via blank imports and `init()` registration
- All notifications registered in `cmd/keel/main.go` via `_ "github.com/keel-hq/keel/extension/notification/..."`
- Same pattern for credentials helpers and bots
- Providers are explicitly wired in `setupProviders()` function
- The `types/types.go` file is the best starting point - contains all core domain concepts

### File Created
- `keel/ARCHITECTURE.md` - 301 lines covering:
  - Quick start reading list
  - High-level ASCII architecture diagram
  - Directory structure table
  - Core concepts (Providers, Triggers, Policies, Notifications, Approvals)
  - Data flow explanation
  - Key annotations reference
  - Environment variables reference
  - Extension points guide (how to add notifiers, webhooks, providers, credentials helpers)
  - Build/run/test commands
  - Common tasks lookup table