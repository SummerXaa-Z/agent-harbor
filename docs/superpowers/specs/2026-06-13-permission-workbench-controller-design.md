# Permission Workbench Controller Split Design

## Goal

Reduce `AiAdminPermissionWorkbench.tsx` state ownership by moving the approval-decision confirmation state machine into a focused hook, without changing the permission-change journey, UI copy, button behavior, or API calls.

## Design Consensus

The next maintainability bottleneck is no longer pure presenter logic. After the presenter split, the workbench still directly owns approval confirmation state: opening approve/reject/withdraw, default approval comments, reject-reason validation, comment edits, and routing the confirmed action to the parent callbacks.

This slice should extract only that state machine. It should not move the main request form, production primary action routing, DOM scroll helpers, or API mutation functions in `ConsoleController`.

## Architecture

Create `frontend/src/hooks/usePermissionApprovalDecision.ts`.

The hook owns:

- `pendingApprovalDecision`
- `beginApprovalDecision(action, requestId?)`
- `cancelApprovalDecision()`
- `updatePendingApprovalComment(comment)`
- `confirmPendingApprovalDecision()`
- `ApprovalDecisionAction` and `PendingApprovalDecision` types

The hook receives the existing callback props:

- `onSelectApprovalRequest`
- `onApproveApprovalRequest`
- `onRejectApprovalRequest`
- `onWithdrawApprovalRequest`
- `t`

`AiAdminPermissionWorkbench.tsx` keeps rendering the confirmation UI and reviewer context, but consumes the hook instead of owning `useState` directly.

## Guardrails

- `AiAdminPermissionWorkbench.tsx` must not import `useState`.
- `pendingApprovalDecision` and confirmation helpers must come from `usePermissionApprovalDecision`.
- Rejection must still require a trimmed reviewer reason before calling `onRejectApprovalRequest`.
- Approval must still default to `text.approvalApproveDefaultComment`.
- Withdraw must still route to `onWithdrawApprovalRequest(comment)`.
- The workbench line guard should tighten from 1450 to 1300 lines.

## Verification

Focused:

- `pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs tests/permissionJourneySafety.test.mjs`
- `pnpm --dir frontend exec tsc -p tsconfig.json --noEmit`

Full:

- `pnpm --dir frontend test`
- `pnpm --dir frontend build`
- `git diff --check`
- `make check`
- `make release-check`
