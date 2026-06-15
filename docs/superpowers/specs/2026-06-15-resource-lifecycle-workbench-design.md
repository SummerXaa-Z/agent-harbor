# Resource Lifecycle Workbench Design

## Context

The console already exposes a Resource Management workspace and a `ResourceLifecycleView`, but the page still splits lifecycle actions across separate areas:

- Agent creation, key creation, and credential rotation sit in the Agent Registry panel header.
- Route policy creation sits in Route Governance and Policy Review.
- Capability discovery and permission changes require users to know which secondary page to open.

This is usable for engineers who already know the product, but it does not meet the current product bar: an administrator should be able to start from one resource lifecycle entry and understand how to move from registration to credentials, capabilities, permissions, and runtime validation.

## Design Consensus

Use the existing Resource Management page as the single lifecycle workbench. Do not add a new route, backend API, or separate resource CRUD surface. The resource page should make the common lifecycle operations visible at the top, then keep the registry and contract matrix as supporting inventory.

Recommended approach:

1. Extend `ResourceLifecycleView` with two action slots:
   - `primaryActions`: mutation actions such as create Agent, create key, rotate credential, and create policy.
   - `secondaryActions`: navigation actions such as reviewing capabilities and starting permission changes.
2. Render those actions inside the lifecycle workbench body, not only in the panel header.
3. Keep all existing mutation forms and handlers. They continue to use `ActionModalButton`, `useManagementOperations`, and existing API calls.
4. Move Agent Registry away from being the main mutation launcher. It remains the detailed list and status/change surface for existing Agents.
5. Keep Capability Governance, Route Governance, and Permission Changes pages as full detail pages. The resource workbench links into them instead of duplicating their full UI.

## Boundaries

In scope:

- Frontend-only lifecycle entry consolidation.
- i18n updates in English and zh-CN.
- Structure tests that prevent the create actions from drifting back to the Agent Registry as the primary entry.
- Visual styling for the lifecycle command area and non-compact command modal triggers.
- CHANGELOG entry.

Out of scope:

- Backend API or persistence changes.
- New resource types.
- Removing specialist pages from navigation.
- Rewriting Agent, policy, or capability forms.
- Changing permission semantics.

## User Experience

On `#registry`, the first panel should answer:

- What is the resource lifecycle?
- What has been registered and what still needs action?
- Which action should I take next without searching other pages?

The workbench action area should be visually obvious as clickable controls. It should not look like passive cards. The Agent Registry remains below as a searchable inventory and detail view.

## Technical Plan

- `frontend/src/components/ResourceLifecycleView.tsx`
  - Add `primaryActions?: ReactNode` and `secondaryActions?: ReactNode`.
  - Render a command section after the stage strip and before the resource list.
- `frontend/src/components/ConsolePrimitives.tsx`
  - Let `ActionModalButton` accept a `variant` prop, defaulting to compact for existing panel headers.
  - Add a command variant for resource workbench buttons.
- `frontend/src/ConsoleController.tsx`
  - Build `resourceLifecyclePrimaryActions` from existing modal action elements.
  - Build `resourceLifecycleSecondaryActions` from existing links to capabilities and permission changes.
  - Pass those slots to `ResourceLifecycleView`.
  - Stop passing mutation action buttons into the Agent Registry header on the Resource Management page.
- `frontend/src/styles.css`
  - Add command section and command trigger styles using existing tokens.
- `frontend/src/i18n.ts`
  - Add EN + zh-CN copy for the command section.
- Tests
  - Extend existing structure tests in `frontend/tests/styleTheme.test.mjs` and `frontend/tests/consoleNavigation.test.mjs`.

## Risks

- Duplicated policy create button can appear in different routes. This is acceptable because only one route renders at a time.
- `ConsoleController.tsx` is still large. Keep this change as composition-only and avoid adding new state.
- Command buttons must be visibly clickable without adding a decorative card grid. Use real button affordances and existing modal dialog behavior.

