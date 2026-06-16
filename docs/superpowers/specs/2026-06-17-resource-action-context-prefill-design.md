# Resource Action Context Prefill Design

## Goal

Make Resource Management actions preserve the operator's selected resource context when opening operational forms. If an operator selects an Agent or MCP target and chooses the next recommended action, the modal should show what resource is being changed and prefill the Agent/target fields that can be inferred safely.

## Problem

The Resource Management page now has a selected-resource context panel, but action modals still feel generic. A user can click a resource row, open an action, and still face a form that looks detached from the resource they just chose. This breaks the production journey because operators must re-check which tenant, workspace, Agent, or MCP target they are acting on.

## Scope

- Keep all changes in the frontend.
- Do not add new dependencies.
- Do not change backend APIs, route policy semantics, or permission package semantics.
- Keep business inference in pure frontend helpers, not in `ConsoleController.tsx`.
- Preserve existing manual command buttons for create Agent, create key, rotate credential, and create policy.
- Add context-aware prefill only when the action is launched from a selected resource or lifecycle row.

## Design

Introduce a small resource action form context model in `resourceLifecycleActionPlanner.ts`.

The planner will return richer `open_modal` plans:

- `modal`: which operational form to open.
- `agentId`: the resource Agent when the action is scoped to one Agent.
- `callerAgentId`: the same-scope local caller when creating a policy from an MCP target.
- `targetAgentId`: the same-scope target when creating a policy from a caller.
- `context`: display-only business labels for the modal header.

`ConsoleController.tsx` will only apply the plan:

- `rotate_credential` preselects `rotateForm.agentId`.
- `create_key` preselects `keyForm.agentId`.
- `create_policy` preselects `policyForm.callerAgentId` and `policyForm.targetAgentId`.
- A lightweight `resourceActionContext` state is passed to modal forms for display.

`ManagementForms.tsx` will render a compact, read-only context strip above the form when provided. This strip will use business-readable labels and avoid raw IDs in the primary path.

## UX Behavior

- From an MCP target missing credentials: recommended action opens rotate credential with that MCP target selected.
- From an MCP target needing authorization: recommended action continues to permission change, unchanged.
- From command-center create key / create policy buttons: forms stay generic and editable.
- From a future row action or selected resource action that opens create key or create policy: inferred fields are filled, while the operator can still adjust them before submitting.
- The context strip answers: "Which resource am I changing?" and "Which tenant/workspace does this belong to?"

## Testing

- Extend `resourceLifecycleActionPlanner.test.mjs` for richer modal plans and same-scope caller/target inference.
- Extend `styleTheme.test.mjs` to guard that form context stays in `ManagementForms.tsx`, planner owns inference, and controller only applies plans.
- Extend `i18n.test.mjs` for all new English and Chinese strings.
- Run focused tests, full frontend tests, build, `make check`, and `make release-check`.

## Non-Goals

- No resource creation wizard.
- No backend persistence changes.
- No modal design system rewrite.
- No new tenant or policy permission model.
