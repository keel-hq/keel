# Refresh admin dashboard screenshots for the new Keel UI

## Summary

`img/keel_ui.png` still showed the old Vue admin UI — blue buttons, a top tab bar and a paginated table. Keel now ships a React/shadcn console with a sidebar layout, so the image on the home page and in the guide no longer resembled the product a reader would see after enabling the dashboard.

Recaptured the dashboard and added screenshots for the three views that had none:

- `img/keel_ui.png` — replaced (referenced from `README.md` and `docs/README.md`)
- `img/docs/ui-tracked-images.png` — new
- `img/docs/ui-approvals.png` — new
- `img/docs/ui-audit-logs.png` — new

`docs/README.md` gains three subsections under "Enabling admin dashboard". That bullet list already advertised "View all tracked images" and "See audit logs (updates, approvals)" without ever showing them, so each new screenshot sits under a short heading explaining what the view is for.

Screenshots were taken against a local k3s cluster (k3d) running 13 Keel-managed workloads — Deployments, StatefulSets, a DaemonSet and a CronJob across `production`, `staging` and `monitoring` — with Keel running out-of-cluster against the cluster's kubeconfig. The data shown is genuinely produced rather than mocked: Keel polled the real registries and applied four updates on its own, and the approval rows come from driving the real webhook and approval endpoints, which in turn populated the audit log and the "updates per week" statistic.

One detail shaped the choice of demo images. When Keel applies an update it rewrites the image reference in the spec (`provider/kubernetes/updates.go`), and Docker Hub official images go through `ShortName()`, which expands them to their `library/` form. A first pass using `nginx`, `redis`, `postgres` and `traefik` therefore produced a table containing both `nginx:1.27.3-alpine` and `library/nginx:1.31.3-alpine`. The demo cluster now uses images that keep a stable name across an update — `ghcr.io/…`, `quay.io/…` and namespaced Docker Hub repositories — so the image column reads consistently. Workload names were also aligned with what they actually run (`edge-proxy`, `valkey`).

All screenshots are light theme at 1760px wide, matching the light VuePress theme the site uses.

## Testing

- Verified all 22 image references across the repo's markdown resolve to files under `.vuepress/public`, including the three new ones.
- Confirmed every pod in the demo cluster reached `Running`/`Completed` before capturing, so no view shows an unavailable workload or a broken pull.
- Checked the rendered captures individually: the dashboard shows 18 resources with no truncated columns, approvals shows 2 pending / 1 approved / 1 rejected, tracked images shows 13 images across 3 registries, and audit logs shows the approve → archive → deployment-update chain.
- Not run: a full `vuepress build`. The repo has no `node_modules` and VuePress 1.x is unlikely to build under Node 24; the change is images plus markdown, and image paths were validated programmatically instead. Worth a local `npm i && npm run build` before merging if you want certainty.
