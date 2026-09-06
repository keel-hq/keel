# Add self-hosted manifests, make Helm chart the primary install (#684)

## Summary

Keel's documented install path is unreachable, so new users cannot install Keel. This adds a self-hosted replacement inside the repository and re-anchors the install docs on targets that resolve.

`sunstone.dev` times out on every request and `install.onecontainer.net` no longer resolves. No reference to either host survives in the current trees (`git grep -i sunstone origin/master` is empty, and the live `keel.sh/docs/` and `keel.sh/examples/` pages are clean), because the dead links were already deleted earlier: from this repository's `readme.md` in `a6f6a9d9` (2023-03-05, "cleanup"), and from the documentation site's `docs/README.md` — including the exact `kubectl apply -f https://sunstone.dev/keel?...` line — in `e1ce099` (2026-08-06, "ci: automate site deploys and drop sunstone.dev references"). Issue #684 itself is still open with no linked pull request.

So there was no stale link left to delete. What was still missing is a working install path a new user can follow today, which is what this change adds: a static manifest set that lives in-tree, plus an explicit statement that the Helm chart is the primary method and the manifests are the fallback.

## Changes

- Add `docs/manifests/keel/`: a ready-to-apply set (namespace, ServiceAccount, ClusterRole, ClusterRoleBinding, Secret, Deployment, Service) modelled on `chart/keel`, plus a README covering placeholders, the pinned image, and verification steps.
- Filenames carry a numeric prefix (`00-namespace` … `40-service`) so a single `kubectl apply -f docs/manifests/keel/` orders dependencies correctly, matching the chart's existing `00-namespace.yaml` convention.
- `readme.md`: state that the Helm chart is the recommended and maintained method, and add a "Static Manifest (alternative)" section for users who cannot use Helm.
- `chart/keel/README.md`: cross-reference the static-manifest alternative from the chart's Installing section.
- Image is pinned to `ghcr.io/keel-hq/keel:0.22.3` (the chart's `appVersion`, with `chart/keel/Chart.yaml` named as the source of truth) rather than `nightly`, which this repo already documents as unsupported for production.
- No Go code, chart template, or CI changes.

## Testing

- **Basic Auth pairing verified against the published image.** Keel requires `BASIC_AUTH_USER` and `BASIC_AUTH_PASSWORD` to be set together or both unset (`pkg/auth/config.go`). Running `ghcr.io/keel-hq/keel:0.22.1` with only `BASIC_AUTH_PASSWORD` logs `level=fatal msg="invalid administrator authentication configuration"` — the crash-loop the first revision of these manifests would have produced. With both variables set, `0.22.1` and the pinned `0.22.3` both clear auth validation and proceed to Kubernetes client setup, which is the expected failure outside a cluster.
- **Apply ordering confirmed from kubectl's own behaviour.** `kubectl` visits a directory lexically; the error output lists the manifests in `00`→`40` order, which is why one `apply -f` is now safe on an empty cluster and why the earlier unsorted names were not.
- All seven manifests parse as YAML (`yaml.safe_load_all`), every relative link target in the touched files resolves on disk, and the pinned `0.22.3` tag pulls successfully.
- **Not run — live cluster.** Applying the set against a real API server could not be verified here: nested privileged containers are blocked on this host (`crun: mount sysfs to sys: Operation not permitted`), so neither k3s-in-Docker nor the k3d harness in `keel-dev-stack` can start, and there is no reachable cluster for `kubectl apply --dry-run=server`. The manifests are structurally derived from `chart/keel`, whose install path is covered by the packaged-chart k3s suite in CI.
- `make release-validate` was not run: the change touches no chart template, application startup path, or Go code (`chart/keel/README.md` is documentation only).

## Note

The `helm repo add keel https://keel-hq.github.io/keel/` instruction is working and was left unchanged — its `index.yaml` returns 200. The directory root returns 404 because GitHub Pages serves no index there, which does not affect `helm repo add`.
