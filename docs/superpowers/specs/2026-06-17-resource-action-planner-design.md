# Resource Action Planner Design

## Goal

Reduce Resource Management action complexity in `ConsoleController.tsx` by moving row-action decision making into a pure, tested frontend planner.

## Design Consensus

Resource Management row clicks should still behave exactly as in PR #95:

- Credential blockers open the existing credential rotation modal in place.
- Capability blockers navigate to Tool Capabilities with the target preselected.
- Permission blockers navigate to Permission Changes with a Resource Management handoff context.
- Runtime follow-ups navigate to Runtime Audit with caller or target filters set from the clicked resource.
- Disabled resources stay in Resource Management.

The next increment changes structure, not behavior. `ConsoleController.tsx` will execute action plans but no longer decide the caller/target fallback rules inline.

## Architecture

Create `frontend/src/resourceLifecycleActionPlanner.ts` as a pure module. It accepts a `ResourceLifecycleItem`, resource collections, and formatting callbacks, then returns a discriminated `ResourceLifecycleActionPlan`.

Controller responsibilities stay limited to side effects:

- update management forms,
- update capability form,
- update trace filters,
- open an existing modal,
- set the active navigation workspace,
- hand off permission-change context.

The planner owns business choices:

- same tenant/workspace caller selection,
- same tenant/workspace MCP target selection,
- caller-row versus target-row permission defaults,
- caller-row versus target-row runtime filters,
- fallback navigation for disabled resources.

## Non-Goals

- No backend changes.
- No new dependencies.
- No visible copy changes.
- No new modal system.
- No behavior change to PR #95 row actions.

## Testing

Add `frontend/tests/resourceLifecycleActionPlanner.test.mjs` to cover:

- credential blocker returns a rotate-credential modal plan,
- capability blocker returns a capability prefill plan,
- target permission blocker selects the same-scope caller,
- caller permission blocker selects the same-scope target,
- runtime plans filter caller rows by caller and target rows by target,
- disabled rows navigate back to Resource Management.

Existing structure tests continue guarding controller state count and component size.
