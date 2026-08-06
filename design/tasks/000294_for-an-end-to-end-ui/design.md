# Design: Add Workspace Inspector E2E Fixture

## Architecture

Add one self-contained fixture directory, `workspace-inspector-e2e/`, at the selected repository root. It contains only:

- `alpha.ts`: a directly exported, explicitly typed `add(a: number, b: number): number` function.
- `README.md`: a short explanation of the fixture and the function's inputs and result.

No runtime integration, package changes, build configuration, or generated files are needed.

## Key Decisions

- Use explicit parameter and return types so the inspector has clear TypeScript information to display.
- Keep the implementation to a direct arithmetic return because the fixture tests inspection, not application behavior.
- Do not commit or push the fixture files, as their uncommitted state is part of the requested E2E scenario.

## Repository Learnings and Constraints

- The available repositories are `keel`, `keel.sh`, and `b-alex`; no existing `workspace-inspector-e2e` directory was found.
- The target repository remains uncertain and must be confirmed before implementation.
- Planning documents are committed to `helix-specs`; implementation files must not be included in that commit.
