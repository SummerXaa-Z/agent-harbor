# Resource Context Panel Design

## Goal

Make Resource Management feel like a continuous production workflow instead of a list of actions. Operators should be able to select a resource, understand why it needs attention, and take the next safe action without losing tenant, workspace, capability, permission, or runtime context.

## Design Consensus

Add a compact "current resource" context panel inside the existing Resource Lifecycle workbench.

The panel should:

- default to the first resource that needs attention, falling back to the first resource,
- let the operator select a resource from the lifecycle rows,
- show business-readable tenant and workspace context,
- show the current blocker/status and counts for capabilities, permissions, and runtime,
- keep the existing next-action button behavior routed through `planResourceLifecycleAction`,
- avoid raw tenant IDs in the primary path.

## Architecture

Keep this frontend-only:

- `ResourceLifecycleView` owns local selected-resource state.
- `ConsoleController` injects tenant/workspace display formatters.
- Existing row action planning remains in `resourceLifecycleActionPlanner.ts`.
- No backend API, schema, dependency, or modal-system changes.

The context panel is presentational. It does not mutate data and does not submit requests. It only improves the continuity between the resource list and the existing action flow.

## Non-Goals

- No new route.
- No resource detail drawer.
- No CRUD expansion.
- No backend resource projection.
- No replacement for the Agent Registry table.

## Acceptance Criteria

1. Resource Management shows a current-resource panel when lifecycle rows exist.
2. The panel defaults to the first non-ready resource and can switch when a row resource button is selected.
3. The panel shows tenant and workspace through formatter callbacks, not raw IDs.
4. The panel shows status, blocker detail, capability count, permission count, runtime count, and next action.
5. Existing next-action routing behavior remains unchanged.
6. EN and zh-CN strings are paired.
7. `pnpm --dir frontend test`, `pnpm --dir frontend build`, `git diff --check`, `make check`, and `make release-check` pass.

