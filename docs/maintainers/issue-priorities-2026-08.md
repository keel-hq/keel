# Open issue priorities (August 2026)

This is a point-in-time triage of the highest-impact unresolved work in the
Keel issue tracker. It prioritizes security and correctness over feature work
and reaction counts. Pull requests are listed separately where they already
provide part or all of the fix.

## Top five

| Priority | Issue | Why it matters | Recommended next step |
| --- | --- | --- | --- |
| P0 | [#859: gRPC vulnerability](https://github.com/keel-hq/keel/issues/859) | Keel pins `google.golang.org/grpc` 1.68.0. The report identifies CVE-2026-33186, a public exploit, and 1.79.3 as the minimum fixed version. This is the clearest remotely exploitable dependency risk in the backlog. | Upgrade the Go toolchain to at least 1.24 and gRPC to at least 1.79.3, then run unit, race, and k3s end-to-end tests. Confirm the final module graph with a vulnerability scan before merging. |
| P0 | [#844: webhook signing secret in logs](https://github.com/keel-hq/keel/pull/844) | A signing secret embedded in a webhook URL can be copied into ordinary logs. Anyone with log access could then forge webhook requests. A proposed fix already exists, reducing time to remediation. | Review and rebase #844, add tests proving non-debug logs redact URL paths and queries, and avoid exposing secrets even at debug level unless that behavior is explicitly documented and accepted. |
| P1 | [#834: wrong-architecture image selected](https://github.com/keel-hq/keel/issues/834) | Keel can replace an amd64 workload with an old `armhf` tag, causing a production crash loop. This is an unsafe-update failure: automation makes a healthy workload unavailable. It may share a root cause with version-selection reports such as #823 and #757. | Reproduce with the Jellyfin tag set; define strict semver/prerelease behavior for a non-semver current tag such as `latest`; reject architecture-flavored and otherwise non-semver tags under semver policies; add regression tests before changing selection logic. |
| P1 | [#845: use the running pod digest as polling baseline](https://github.com/keel-hq/keel/issues/845) | Initializing from the registry digest hides real cluster drift after Keel starts or restarts. A stale workload can remain stale indefinitely until the registry changes again, undermining Keel's core purpose. This also subsumes the long-standing #724 report. | Validate/rebase [#847](https://github.com/keel-hq/keel/pull/847). Test multi-replica, multi-container, unscheduled-pod, digest-normalization, and RBAC failure cases; fall back to registry state only when no trustworthy pod `imageID` exists. |
| P1 | [#865: migrate end-of-life AWS and Pub/Sub SDK majors](https://github.com/keel-hq/keel/issues/865) | AWS SDK v1 is archived and no longer receives security fixes. Pub/Sub v1 is deprecated and approaches end of support. Both are direct dependencies and will increasingly block safe upgrades. | Split into two PRs. Migrate AWS ECR credentials to AWS SDK v2 first; migrate Pub/Sub separately and explicitly preserve current receive concurrency. Add focused tests around auth errors, topic/subscription creation races, and cleanup. |

## Immediate dependency updates alongside the five

The following open Dependabot pull requests are low-effort security hygiene and
should not wait for the larger migrations:

1. [#885](https://github.com/keel-hq/keel/pull/885): upgrade
   `github.com/moby/spdystream` 0.5.0 to 0.5.1 for CVE-2026-35469.
2. [#883](https://github.com/keel-hq/keel/pull/883): upgrade
   `github.com/containerd/containerd` 1.7.24 to 1.7.33 for multiple published
   security fixes.
3. [#884](https://github.com/keel-hq/keel/pull/884): upgrade
   `github.com/slack-go/slack` 0.15.0 to 0.23.1, including rejection of empty
   signing secrets.

Treat these as a coordinated dependency train: merge one at a time, run the
same test matrix for each, and inspect transitive version changes rather than
blindly combining all lockfile updates.

## Triage notes

- [#822](https://github.com/keel-hq/keel/issues/822), where a notification
  failure stopped later deployments, appears addressed by the non-blocking
  notification changes already on `master`; verify and close it with a
  regression reference.
- [#827](https://github.com/keel-hq/keel/issues/827), accepting HTTP 202 from
  notification webhooks, also appears addressed on `master`; verify and close.
- [#833](https://github.com/keel-hq/keel/issues/833) and
  [#840](https://github.com/keel-hq/keel/issues/840) are release-artifact
  incidents rather than application-code fixes. Confirm current image tags and
  close or move remaining work to release operations.
- [#858](https://github.com/keel-hq/keel/issues/858) is likely expected behavior:
  polling a mutable `latest` tag through the tag-list policy does not detect a
  digest change. Improve documentation or configuration diagnostics after the
  correctness and security work above.

## Validation baseline

`go test ./...` currently includes registry integration tests that contact
Docker Hub, Quay, and other external registries. In a network-restricted
checkout those packages fail despite the remaining packages passing. Separate
network integration tests from hermetic unit tests so dependency and security
updates can be validated reliably in CI.
