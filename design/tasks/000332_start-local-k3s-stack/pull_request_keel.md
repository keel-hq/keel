# Remove refresh buttons from tables and add keel-dev-stack agent skill

## Summary
Remove the Refresh buttons from all data tables in the admin UI (dashboard Kubernetes resources, approvals, tracked images, and audit logs) and clean up the now-unused `RefreshCw` and `Button` imports. Add a generic `.agents/skills/keel-dev-stack/SKILL.md` agent skill that documents how to start, verify, and tear down the local Keel dev stack (k3s cluster via k3d on :6443, Keel backend/dashboard on :9300, and the React UI with the Vite dev server on :8000), and link it from AGENTS.md with a short description.

## Testing
- `npm run typecheck` and `npm run lint` pass in `ui/`.
- `npm run build` succeeds; the rebuilt `ui/dist` is served by Keel at `http://localhost:9300/` (200) and the Vite dev server at `http://localhost:8000/` (200).
