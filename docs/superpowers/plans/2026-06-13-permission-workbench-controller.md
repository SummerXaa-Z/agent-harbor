# Permission Workbench Controller Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move approval-decision confirmation state out of `AiAdminPermissionWorkbench.tsx` into a focused hook.

**Architecture:** Add `usePermissionApprovalDecision` under `frontend/src/hooks`. The workbench remains the renderer and callback orchestrator for the broader permission journey, but the approval confirmation state machine lives in the hook. Static architecture tests prevent `useState` and helper definitions from returning to the workbench.

**Tech Stack:** React 19, TypeScript, Node built-in test runner, existing static architecture tests.

---

### Task 1: Add Red Structure Tests

**Files:**
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/tests/permissionFlowLayout.test.mjs`

- [x] **Step 1: Read the new hook in tests**

In `frontend/tests/styleTheme.test.mjs`, add:

```js
const permissionApprovalDecisionHook = readExistingFile(new URL("../src/hooks/usePermissionApprovalDecision.ts", import.meta.url));
```

In `frontend/tests/permissionFlowLayout.test.mjs`, add:

```js
const permissionApprovalDecisionHook = readFileSync(new URL("../src/hooks/usePermissionApprovalDecision.ts", import.meta.url), "utf8");
```

- [x] **Step 2: Tighten the workbench growth guard**

Update the `ai admin workbench has a growth guard while controller hooks are split` test in `frontend/tests/styleTheme.test.mjs`:

```js
assert.ok(
  workbench.split("\n").length <= 1300,
  "AiAdminPermissionWorkbench should delegate approval decision state before adding more UI"
);
assert.match(workbench, /from "\.\.\/hooks\/usePermissionApprovalDecision"/);
assert.match(permissionApprovalDecisionHook, /export function usePermissionApprovalDecision/);
assert.match(permissionApprovalDecisionHook, /export type ApprovalDecisionAction = "approve" \| "reject" \| "withdraw"/);
assert.doesNotMatch(workbench, /import \{ useState \} from "react"/);
assert.doesNotMatch(workbench, /setPendingApprovalDecision/);
```

Keep the existing presenter and `CapabilityChipList` assertions.

- [x] **Step 3: Move approval decision assertions to the hook**

In `frontend/tests/permissionFlowLayout.test.mjs`, update:

```js
test("permission request approval decisions show reviewer context before resolving", () => {
  assert.match(permissionApprovalDecisionHook, /export function usePermissionApprovalDecision/);
  assert.match(workbench, /usePermissionApprovalDecision\(\{/);
  assert.match(workbench, /pendingApprovalDecision/);
  assert.match(workbench, /permissionEntityDisplayName\(approvalReviewer\.trim\(\), t\)/);
  assert.match(workbench, /beginApprovalDecision\("approve"/);
  assert.match(workbench, /beginApprovalDecision\("reject"/);
  assert.match(workbench, /className="approval-reviewer-context"/);
  assert.match(workbench, /text\.approvalReviewerIdentity/);
  assert.match(workbench, /text\.approvalReviewerSeparationDetail/);
  assert.match(workbench, /className="approval-decision-confirmation"/);
  assert.match(workbench, /action\.confirmApprovePermissionRequest/);
  assert.match(workbench, /action\.cancelApprovalDecision/);
  assert.doesNotMatch(workbench, /onClick=\{\(\) => onApproveApprovalRequest\(\)\}/);
  assert.doesNotMatch(workbench, /onClick=\{\(\) => onRejectApprovalRequest\(\)\}/);
  assert.match(styles, /\.approval-reviewer-context\s*\{/);
  assert.match(styles, /\.approval-decision-confirmation\s*\{/);
});
```

Update rejection and withdraw tests:

```js
assert.match(permissionApprovalDecisionHook, /pendingApprovalDecision\.comment\.trim\(\)/);
assert.match(permissionApprovalDecisionHook, /message\.permissionApprovalRejectReasonRequired/);
assert.match(permissionApprovalDecisionHook, /onRejectApprovalRequest\(pendingApprovalDecision\.requestId, comment\)/);
assert.match(permissionApprovalDecisionHook, /export type ApprovalDecisionAction = "approve" \| "reject" \| "withdraw"/);
assert.match(permissionApprovalDecisionHook, /onWithdrawApprovalRequest\(comment\)/);
```

- [x] **Step 4: Run focused tests and verify red**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs
```

Expected: tests fail because `frontend/src/hooks/usePermissionApprovalDecision.ts` does not exist and the workbench still imports `useState`.

### Task 2: Extract Approval Decision Hook

**Files:**
- Create: `frontend/src/hooks/usePermissionApprovalDecision.ts`
- Modify: `frontend/src/components/AiAdminPermissionWorkbench.tsx`

- [x] **Step 1: Create the hook**

Create `frontend/src/hooks/usePermissionApprovalDecision.ts`:

```ts
import { useState } from "react";

import type { Translator } from "../permissionWorkbenchPresenters";

export type ApprovalDecisionAction = "approve" | "reject" | "withdraw";

export interface PendingApprovalDecision {
  action: ApprovalDecisionAction;
  requestId?: string;
  comment: string;
  error: string;
}

interface UsePermissionApprovalDecisionArgs {
  onApproveApprovalRequest: (requestId?: string, comment?: string) => void;
  onRejectApprovalRequest: (requestId?: string, comment?: string) => void;
  onSelectApprovalRequest: (requestId: string) => void;
  onWithdrawApprovalRequest: (comment?: string) => void;
  t: Translator;
}

export function usePermissionApprovalDecision({
  onApproveApprovalRequest,
  onRejectApprovalRequest,
  onSelectApprovalRequest,
  onWithdrawApprovalRequest,
  t
}: UsePermissionApprovalDecisionArgs) {
  const [pendingApprovalDecision, setPendingApprovalDecision] = useState<PendingApprovalDecision | null>(null);

  function beginApprovalDecision(action: ApprovalDecisionAction, requestId?: string) {
    if (requestId) onSelectApprovalRequest(requestId);
    setPendingApprovalDecision({
      action,
      requestId,
      comment: action === "approve" ? t("text.approvalApproveDefaultComment") : "",
      error: ""
    });
  }

  function cancelApprovalDecision() {
    setPendingApprovalDecision(null);
  }

  function updatePendingApprovalComment(comment: string) {
    setPendingApprovalDecision((current) => current ? { ...current, comment, error: "" } : current);
  }

  function confirmPendingApprovalDecision() {
    if (!pendingApprovalDecision) return;
    const comment = pendingApprovalDecision.comment.trim();
    if (pendingApprovalDecision.action === "reject" && !comment) {
      setPendingApprovalDecision({
        ...pendingApprovalDecision,
        error: t("message.permissionApprovalRejectReasonRequired")
      });
      return;
    }
    if (pendingApprovalDecision.action === "approve") {
      onApproveApprovalRequest(pendingApprovalDecision.requestId, comment || t("text.approvalApproveDefaultComment"));
    } else if (pendingApprovalDecision.action === "withdraw") {
      onWithdrawApprovalRequest(comment);
    } else {
      onRejectApprovalRequest(pendingApprovalDecision.requestId, comment);
    }
    setPendingApprovalDecision(null);
  }

  return {
    beginApprovalDecision,
    cancelApprovalDecision,
    confirmPendingApprovalDecision,
    pendingApprovalDecision,
    updatePendingApprovalComment
  };
}
```

- [x] **Step 2: Wire the hook into the workbench**

In `frontend/src/components/AiAdminPermissionWorkbench.tsx`:

```ts
import {
  usePermissionApprovalDecision,
  type ApprovalDecisionAction
} from "../hooks/usePermissionApprovalDecision";
```

Remove the local `useState` import, `ApprovalDecisionAction`, `PendingApprovalDecision`, and local approval decision helper functions. Add:

```ts
  const {
    beginApprovalDecision,
    cancelApprovalDecision,
    confirmPendingApprovalDecision,
    pendingApprovalDecision,
    updatePendingApprovalComment
  } = usePermissionApprovalDecision({
    onApproveApprovalRequest,
    onRejectApprovalRequest,
    onSelectApprovalRequest,
    onWithdrawApprovalRequest,
    t
  });
```

- [x] **Step 3: Run TypeScript and focused tests**

Run:

```bash
pnpm --dir frontend exec tsc -p tsconfig.json --noEmit
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs tests/permissionJourneySafety.test.mjs
```

Expected: TypeScript compiles and focused tests pass.

### Task 3: Document And Run Gates

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `docs/superpowers/plans/2026-06-13-permission-workbench-controller.md`

- [x] **Step 1: Update changelog**

Add under `## [Unreleased] > ### Added`:

```markdown
- Moved Permission Changes approval-decision confirmation state into a focused hook, reducing the workbench component while keeping approval, rejection, and withdraw behavior unchanged.
- 权限变更页的审批确认状态已下沉到独立 hook，减少工作台组件复杂度，同时保持审批、拒绝和撤回行为不变。
```

- [x] **Step 2: Run full frontend verification**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
```

Expected: tests and build pass, and no whitespace errors are reported.

- [x] **Step 3: Run repository gates**

Run:

```bash
make check
make release-check
```

Expected: both gates pass.
