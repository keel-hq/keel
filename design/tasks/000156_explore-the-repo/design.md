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