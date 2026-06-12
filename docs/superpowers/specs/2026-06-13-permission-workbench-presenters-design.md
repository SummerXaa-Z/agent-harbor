# Permission Workbench Presenter Split Design

## Goal

Reduce `AiAdminPermissionWorkbench.tsx` complexity without changing the permission-change journey, approval behavior, copy, or visual layout.

## Product Decision

This is a production maintainability slice. The operator should see the same Permission Changes workspace and the same `配置 -> 审批 -> 应用 -> 状态检查 -> 验收` journey. The only user-visible effect should be lower future regression risk.

Do not split the main request form, approval process sidebar, or approval confirmation behavior in this slice. Those areas are tightly coupled to local state and mutation callbacks, so they should be separate follow-up PRs after pure presenter extraction is complete.

## Architecture

Create a pure presenter module:

- `frontend/src/permissionWorkbenchPresenters.ts`
- Contains display labels, status/tone mapping, date formatting, workbench step display mapping, and entity/template naming helpers.
- Has no React imports, no hooks, no DOM access, and no network/API calls.

Create a small display component module:

- `frontend/src/components/PermissionWorkbenchParts.tsx`
- Contains `CapabilityChipList`, a stateless component that renders allowed/blocked capability chips.

Keep `AiAdminPermissionWorkbench.tsx` as the orchestration component for now. It imports presenter functions and the display component, but still owns local approval decision state, scroll behavior, and all callbacks.

## Guardrails

- `AiAdminPermissionWorkbench.tsx` should shrink below 1450 lines in this slice.
- New helper logic should not be added back to the workbench file.
- Existing permission flow tests should continue to assert the same process-step behavior, but against the presenter module where the pure functions now live.

## Verification

Run focused tests first:

- `pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs tests/permissionJourneySafety.test.mjs`

Then run:

- `pnpm --dir frontend test`
- `pnpm --dir frontend build`

If focused tests reveal a broader regression, run `make check` before PR.
