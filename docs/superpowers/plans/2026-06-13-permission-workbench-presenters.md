# Permission Workbench Presenters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Shrink `AiAdminPermissionWorkbench.tsx` by moving pure presenter logic and one stateless display component into owned modules.

**Architecture:** Add `permissionWorkbenchPresenters.ts` for pure display/status functions and `PermissionWorkbenchParts.tsx` for small stateless JSX. Keep `AiAdminPermissionWorkbench.tsx` responsible for local approval-decision state and callback wiring. Use source-structure tests to prevent helper logic from growing back into the workbench.

**Tech Stack:** React 19, TypeScript, Node built-in test runner, existing static architecture tests.

---

### Task 1: Add Red Structure Tests

**Files:**
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/tests/permissionFlowLayout.test.mjs`

- [x] **Step 1: Add source reads for the new modules**

In `frontend/tests/styleTheme.test.mjs`, read:

```js
const permissionWorkbenchPresenters = readFileSync(new URL("../src/permissionWorkbenchPresenters.ts", import.meta.url), "utf8");
const permissionWorkbenchParts = readFileSync(new URL("../src/components/PermissionWorkbenchParts.tsx", import.meta.url), "utf8");
```

In `frontend/tests/permissionFlowLayout.test.mjs`, read:

```js
const permissionWorkbenchPresenters = readFileSync(new URL("../src/permissionWorkbenchPresenters.ts", import.meta.url), "utf8");
```

- [x] **Step 2: Update the workbench guard**

In `frontend/tests/styleTheme.test.mjs`, update `ai admin workbench has a growth guard while controller hooks are split` so it asserts:

```js
assert.ok(
  workbench.split("\n").length <= 1450,
  "AiAdminPermissionWorkbench should delegate pure presenters and small display parts before adding more UI"
);
assert.match(app, /from "\.\/components\/AiAdminPermissionWorkbench"/);
assert.match(workbench, /from "\.\.\/permissionWorkbenchPresenters"/);
assert.match(workbench, /from "\.\/PermissionWorkbenchParts"/);
assert.match(permissionWorkbenchPresenters, /export function resolvePermissionJourneyStatus/);
assert.match(permissionWorkbenchPresenters, /export function permissionWorkbenchStepDisplayDetailCode/);
assert.match(permissionWorkbenchPresenters, /export function permissionWorkbenchStepDisplayStatus/);
assert.match(permissionWorkbenchParts, /export function CapabilityChipList/);
assert.doesNotMatch(workbench, /function resolvePermissionJourneyStatus/);
assert.doesNotMatch(workbench, /function permissionWorkbenchStepDisplayDetailCode/);
assert.doesNotMatch(workbench, /function permissionWorkbenchStepDisplayStatus/);
assert.doesNotMatch(workbench, /function CapabilityChipList/);
```

- [x] **Step 3: Update flow-layout function-location assertions**

In `frontend/tests/permissionFlowLayout.test.mjs`, update the process-step test to assert the pure functions are imported and live in the presenter module:

```js
assert.match(workbench, /permissionWorkbenchStepDisplayDetailCode\(step,\s*\{[\s\S]*goLiveReady[\s\S]*runtimeValidationReady[\s\S]*\}\)/);
assert.match(workbench, /permissionWorkbenchStepDisplayStatus\(step,\s*\{[\s\S]*approvalComplete[\s\S]*applicationReady[\s\S]*goLiveReady[\s\S]*runtimeValidationReady[\s\S]*\}\)/);
assert.match(permissionWorkbenchPresenters, /export function permissionWorkbenchStepDisplayDetailCode/);
assert.match(permissionWorkbenchPresenters, /export function permissionWorkbenchStepDisplayStatus/);
assert.doesNotMatch(workbench, /function permissionWorkbenchStepDisplayDetailCode/);
assert.doesNotMatch(workbench, /function permissionWorkbenchStepDisplayStatus/);
```

- [x] **Step 4: Run focused tests and verify red**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs
```

Expected: tests fail because `permissionWorkbenchPresenters.ts` and `PermissionWorkbenchParts.tsx` do not exist yet, and the workbench still owns helper functions.

### Task 2: Extract Pure Presenter Functions

**Files:**
- Create: `frontend/src/permissionWorkbenchPresenters.ts`
- Modify: `frontend/src/components/AiAdminPermissionWorkbench.tsx`

- [x] **Step 1: Create presenter module**

Move these functions and related type aliases/constants from `AiAdminPermissionWorkbench.tsx` to `frontend/src/permissionWorkbenchPresenters.ts` and export them:

```ts
tx
accessSubjectDropdownOption
accessSubjectKindLabelKey
accessDecisionOutcomeLabel
formatDate
permissionTenantPathLabel
permissionWorkspaceDisplayName
uniquePermissionEntityOptions
readableIdentifierLabel
permissionEntityDisplayName
permissionPackageTemplateName
permissionPackageTemplateNameById
permissionApprovalRequestBusinessLabel
permissionPackageTemplateSummary
permissionRequestStepSectionId
permissionRequestStepTarget
resolvePermissionJourneyStatus
permissionDraftStatus
permissionPolicyGateDetailKey
permissionApprovalStatusLabel
permissionApprovalStatusTone
permissionPolicyReasonMessage
translatedValue
permissionInlineMessageTone
shouldShowAdvancedStatusMessage
permissionApplicationHealthLabel
permissionApplicationHealthRowSummary
productionReadinessStatusLabel
permissionProductionReadinessCheckLabel
permissionProductionReadinessCheckMessage
permissionPackageApprovalRouteLabel
productionConsoleStatusTone
permissionWorkbenchActionKey
permissionWorkbenchStatusLabelKey
permissionWorkbenchStatusDetailKey
permissionWorkbenchStatusTone
permissionWorkbenchStepLabelKey
permissionWorkbenchStepDetailKey
permissionWorkbenchStepDisplayDetailCode
permissionWorkbenchStepDisplayStatus
```

The module must import only types and pure helpers. It must not import React.

- [x] **Step 2: Import presenter functions in the workbench**

In `AiAdminPermissionWorkbench.tsx`, import the moved functions from `../permissionWorkbenchPresenters` and remove their local definitions.

- [x] **Step 3: Run TypeScript and focused tests**

Run:

```bash
pnpm --dir frontend exec tsc -p tsconfig.json --noEmit
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs
```

Expected: TypeScript compiles and focused structure tests pass for presenter extraction, unless the component extraction is still pending.

### Task 3: Extract Capability Chip Display Component

**Files:**
- Create: `frontend/src/components/PermissionWorkbenchParts.tsx`
- Modify: `frontend/src/components/AiAdminPermissionWorkbench.tsx`

- [x] **Step 1: Create `CapabilityChipList` module**

Move `CapabilityChipList` to `frontend/src/components/PermissionWorkbenchParts.tsx` and export it. The component should accept:

```ts
{
  capabilities: Capability[];
  emptyLabel: string;
  label: string;
  tone: Tone;
  t: Translator;
}
```

- [x] **Step 2: Import `CapabilityChipList` in the workbench**

Remove the local `CapabilityChipList` definition from `AiAdminPermissionWorkbench.tsx` and import it from `./PermissionWorkbenchParts`.

- [x] **Step 3: Run focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs tests/permissionJourneySafety.test.mjs
```

Expected: focused tests pass and the workbench is below the new 1450-line cap.

### Task 4: Update Documentation And Run Gates

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `docs/superpowers/plans/2026-06-13-permission-workbench-presenters.md`

- [x] **Step 1: Update changelog**

Add a bilingual Unreleased bullet:

```markdown
- Split Permission Changes presenter helpers and capability chip rendering out of `AiAdminPermissionWorkbench`, reducing the largest console component while preserving the permission-change journey.
- 权限变更页的展示辅助函数和能力标签渲染已从 `AiAdminPermissionWorkbench` 拆出，降低最大控制台组件复杂度，同时保持权限变更旅程不变。
```

- [x] **Step 2: Run full frontend verification**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
```

Expected: tests and build pass, and no whitespace errors are reported.

- [x] **Step 3: Run repository gate if frontend verification passes**

Run:

```bash
make check
```

Expected: repository check gate passes.
