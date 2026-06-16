# Management Mutation Busy Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent duplicate resource-management mutations by guarding async form submissions and disabling submit buttons while requests are active.

**Architecture:** Add a ref-backed mutation guard in `useManagementOperations`, expose the current action to `ConsoleController`, and pass `submitting` into existing management forms. Use existing i18n and CSS patterns; no backend or dependency changes.

**Tech Stack:** React, TypeScript, Vite, node test, existing CSS/i18n.

---

## Files

- Modify: `frontend/src/hooks/useManagementOperations.ts`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/components/ManagementForms.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/permissionJourneySafety.test.mjs`
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/tests/i18n.test.mjs`
- Modify: `CHANGELOG.md`

## Task 1: Safety Guards

- [x] **Step 1: Add failing safety assertions**

In `frontend/tests/permissionJourneySafety.test.mjs`, assert that `useManagementOperations.ts` contains:

```js
assert.match(managementHook, /type ManagementMutationAction = "" \| "create_agent" \| "create_key" \| "rotate_credential" \| "create_policy"/);
assert.match(managementHook, /const managementMutationInFlightRef = useRef<ManagementMutationAction>\(""\)/);
assert.match(managementHook, /function beginManagementMutation\(action: ManagementMutationAction\)/);
assert.match(managementHook, /function endManagementMutation\(action: ManagementMutationAction\)/);
assert.match(managementHook, /managementMutationAction/);
```

For `submitAgent`, `submitKey`, `submitCredentialRotation`, and `submitRoutePolicy`, assert each block calls `beginManagementMutation(...)` and has a `finally` block calling `endManagementMutation(...)`.

- [x] **Step 2: Run focused safety test and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs
```

Expected: fail until the hook owns the guard.

- [x] **Step 3: Implement hook guard**

In `frontend/src/hooks/useManagementOperations.ts`:

```ts
type ManagementMutationAction = "" | "create_agent" | "create_key" | "rotate_credential" | "create_policy";
const managementMutationInFlightRef = useRef<ManagementMutationAction>("");
const [managementMutationAction, setManagementMutationAction] = useState<ManagementMutationAction>("");

function beginManagementMutation(action: ManagementMutationAction) {
  if (managementMutationInFlightRef.current) return false;
  managementMutationInFlightRef.current = action;
  setManagementMutationAction(action);
  return true;
}

function endManagementMutation(action: ManagementMutationAction) {
  if (managementMutationInFlightRef.current !== action) return;
  managementMutationInFlightRef.current = "";
  setManagementMutationAction("");
}
```

Wrap the four submit handlers with this guard and return `managementMutationAction`.

- [x] **Step 4: Run focused safety test and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs
```

Expected: pass.

## Task 2: Form Busy State

- [x] **Step 1: Add structure assertions**

In `frontend/tests/styleTheme.test.mjs`, assert that all four management forms accept `submitting`, that `FormFooter` receives it, and that `ConsoleController.tsx` passes action-specific comparisons such as:

```js
assert.match(app, /submitting=\{management\.managementMutationAction === "create_agent"\}/);
assert.match(app, /submitting=\{management\.managementMutationAction === "create_key"\}/);
assert.match(app, /submitting=\{management\.managementMutationAction === "rotate_credential"\}/);
assert.match(app, /submitting=\{management\.managementMutationAction === "create_policy"\}/);
```

- [x] **Step 2: Implement form props**

Add `submitting?: boolean` to `AgentCreateForm`, `KeyCreateForm`, `CredentialRotateForm`, and `PolicyCreateForm`. Pass it to `FormFooter`.

- [x] **Step 3: Disable submit button**

Change `FormFooter` to:

```tsx
function FormFooter({ message, submitLabel, submitting, submittingLabel }: { ... }) {
  return (
    <div aria-busy={submitting || undefined} className="form-footer">
      <button className="primary-button" disabled={submitting} type="submit">
        {submitting ? submittingLabel : submitLabel}
      </button>
      {message ? <span>{message}</span> : null}
    </div>
  );
}
```

- [x] **Step 4: Wire controller props**

Pass the four action-specific `submitting` values from `ConsoleController.tsx`.

## Task 3: i18n, Changelog, and Verification

- [x] **Step 1: Add i18n copy**

Add `action.processing` to EN and zh-CN. Extend `frontend/tests/i18n.test.mjs`.

- [x] **Step 2: Update changelog**

Add one EN and one zh-CN Unreleased bullet for duplicate-submit protection on resource management operations.

- [x] **Step 3: Run focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs tests/styleTheme.test.mjs tests/i18n.test.mjs
```

Expected: pass.

- [x] **Step 4: Run full gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
make check
make release-check
```

Expected: all pass.

- [x] **Step 5: Mark plan complete and ship**

Mark all plan checkboxes, commit, push, create PR, wait for CI, and merge when green.
