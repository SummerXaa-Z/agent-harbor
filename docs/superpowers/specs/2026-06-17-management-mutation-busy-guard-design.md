# Management Mutation Busy Guard Design

## Goal

Prevent duplicate resource-management mutations caused by repeated submit clicks while an operation is in flight.

## Problem

The resource lifecycle workbench now routes Agent, key, credential, and policy operations through modal forms. These forms submit network mutations, but the management hook does not expose a per-form busy state and the submit buttons remain clickable during the async request. In production this can create duplicate Agents, duplicate route policies, repeated key issuance, or repeated credential rotations.

## Scope

- Frontend only.
- No backend API changes.
- No dependency changes.
- Cover create Agent, create key, rotate credential, and create route policy.
- Keep Agent status changes and policy disablement on the existing row-level `cleanupActionId` path.
- Keep validation messages and success messages unchanged.

## Design

Add a guarded mutation state to `useManagementOperations`:

- `managementMutationAction`: `"" | "create_agent" | "create_key" | "rotate_credential" | "create_policy"`.
- A `useRef` stores the immediate in-flight action so a second submit in the same render window is ignored.
- `beginManagementMutation(action)` returns `false` if another management mutation is already running.
- `endManagementMutation(action)` clears the ref/state only for the action that started it.

Each submit handler will:

1. `event.preventDefault()`.
2. Call `beginManagementMutation(...)`.
3. Return early if another operation is active.
4. Run existing validation and network logic inside `try`.
5. Clear the busy action in `finally`, including validation returns.

Forms receive a `submitting` prop and pass it into a shared footer. The footer disables the primary submit button, sets `aria-busy`, and switches the label to a localized "Processing..." / "处理中..." string while the request is active.

## Testing

- Structural test verifies `useManagementOperations` owns the ref-based guard, returns `managementMutationAction`, and wraps all four submit handlers with begin/finally/end.
- Structural test verifies the four management forms expose `submitting` and disabled submit buttons.
- i18n test verifies `action.processing` exists in English and Simplified Chinese.
- Existing full frontend tests, build, `make check`, and `make release-check` remain the release gate.

## Non-Goals

- No server-side idempotency keys in this increment.
- No changes to row-level cleanup/disable actions.
- No modal redesign.
