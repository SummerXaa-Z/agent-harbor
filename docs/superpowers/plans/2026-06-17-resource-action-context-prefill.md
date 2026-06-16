# Resource Action Context Prefill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make resource lifecycle actions open operational forms with the selected resource context and safe prefilled Agent fields.

**Architecture:** Keep inference in `frontend/src/resourceLifecycleActionPlanner.ts`, apply the resulting plan in `ConsoleController.tsx`, and render display-only context in `ManagementForms.tsx`. The feature is frontend-only and leaves backend APIs unchanged.

**Tech Stack:** React, TypeScript, Vite, Vitest/node test, existing CSS/i18n system.

---

## Files

- Modify: `frontend/src/resourceLifecycleActionPlanner.ts`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/components/ManagementForms.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/src/styles.css`
- Modify: `frontend/tests/resourceLifecycleActionPlanner.test.mjs`
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/tests/i18n.test.mjs`
- Modify: `CHANGELOG.md`

## Task 1: Planner Contract

- [ ] **Step 1: Write failing planner assertions**

Add tests in `frontend/tests/resourceLifecycleActionPlanner.test.mjs` that assert:

```js
assert.equal(plan.kind, "open_modal");
assert.equal(plan.modal, "rotate_credential");
assert.equal(plan.agentId, "target-a");
assert.deepEqual(plan.context, {
  resourceName: "工单工具服务",
  resourceKindKey: "resource.kind.mcpTarget",
  tenantName: "客户服务中心",
  workspaceName: "ws-a"
});
```

Also add synthetic `create_key` and `create_policy` cases using the existing `item(...)` helper:

```js
assert.equal(keyPlan.modal, "create_key");
assert.equal(keyPlan.agentId, "caller-a");
assert.equal(policyPlan.modal, "create_policy");
assert.equal(policyPlan.callerAgentId, "caller-a");
assert.equal(policyPlan.targetAgentId, "target-a");
```

- [ ] **Step 2: Run focused test and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/resourceLifecycleActionPlanner.test.mjs
```

Expected: fail because modal plans do not include context or create-key/create-policy handling yet.

- [ ] **Step 3: Implement richer planner plan types**

Change `ResourceLifecycleActionPlan` so `open_modal` can include:

```ts
context: ResourceLifecycleActionContext;
agentId?: string;
callerAgentId?: string;
targetAgentId?: string;
```

Add `ResourceLifecycleActionContext` with `resourceName`, `resourceKindKey`, `tenantName`, and `workspaceName`.

- [ ] **Step 4: Run focused planner test and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/resourceLifecycleActionPlanner.test.mjs
```

Expected: pass.

## Task 2: Controller Prefill

- [ ] **Step 1: Add controller structure guards**

In `frontend/tests/styleTheme.test.mjs`, assert that:

```js
assert.match(app, /const \[resourceActionContext, setResourceActionContext\] = useState<ResourceLifecycleActionContext \| null>\(null\)/);
assert.match(app, /management\.setKeyForm\(\{[\s\S]*agentId: plan\.agentId/);
assert.match(app, /management\.setPolicyForm\(\{[\s\S]*callerAgentId: plan\.callerAgentId/);
assert.match(app, /management\.setPolicyForm\(\{[\s\S]*targetAgentId: plan\.targetAgentId/);
assert.doesNotMatch(app, /sameScopeCaller|sameScopeTarget/);
```

- [ ] **Step 2: Run style guard and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
```

Expected: fail until controller applies the richer plan.

- [ ] **Step 3: Apply modal plans in `ConsoleController.tsx`**

Import `ResourceLifecycleActionContext`, add `resourceActionContext` state, set it from `plan.context`, and update the matching form state:

```ts
setResourceActionContext(plan.context);
if (plan.modal === "create_key" && plan.agentId) {
  management.setKeyForm({ ...management.keyForm, agentId: plan.agentId });
}
if (plan.modal === "rotate_credential" && plan.agentId) {
  management.setRotateForm({ ...management.rotateForm, agentId: plan.agentId });
}
if (plan.modal === "create_policy") {
  management.setPolicyForm({
    ...management.policyForm,
    callerAgentId: plan.callerAgentId ?? management.policyForm.callerAgentId,
    targetAgentId: plan.targetAgentId ?? management.policyForm.targetAgentId
  });
}
```

- [ ] **Step 4: Pass context into action forms**

Pass `context={resourceActionContext}` to `KeyCreateForm`, `CredentialRotateForm`, and `PolicyCreateForm`. Pass no context to `AgentCreateForm`.

## Task 3: Form Context UI and i18n

- [ ] **Step 1: Add i18n assertions**

In `frontend/tests/i18n.test.mjs`, assert EN and zh-CN include:

- `resource.actionContext.title`
- `resource.actionContext.resource`
- `resource.actionContext.scope`

- [ ] **Step 2: Add form context component**

In `frontend/src/components/ManagementForms.tsx`, import `ResourceLifecycleActionContext` and add:

```tsx
function ResourceActionContextStrip({ context, t }: { context?: ResourceLifecycleActionContext | null; t: Translator }) {
  if (!context) return null;
  return (
    <div className="resource-action-context">
      <span className="section-kicker">{t("resource.actionContext.title")}</span>
      <div>
        <strong>{context.resourceName}</strong>
        <small>{t(context.resourceKindKey)}</small>
      </div>
      <p>{t("resource.actionContext.scope")}: {context.tenantName} / {context.workspaceName}</p>
    </div>
  );
}
```

Render it at the top of `KeyCreateForm`, `CredentialRotateForm`, and `PolicyCreateForm`.

- [ ] **Step 3: Style the context strip**

Add `.resource-action-context` styles in `frontend/src/styles.css` using existing tokens only.

- [ ] **Step 4: Run focused UI/i18n tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/i18n.test.mjs
```

Expected: pass.

## Task 4: Docs, Full Verification, and PR

- [ ] **Step 1: Update `CHANGELOG.md`**

Add one EN and one zh-CN bullet under Unreleased describing context-aware resource action forms.

- [ ] **Step 2: Run full frontend gate**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
```

Expected: all pass.

- [ ] **Step 3: Run repository gates**

Run:

```bash
make check
make release-check
```

Expected: all pass.

- [ ] **Step 4: Mark plan complete**

Update every checkbox in this plan to `[x]`.

- [ ] **Step 5: Commit, push, and open PR**

Run:

```bash
git add CHANGELOG.md docs/superpowers/plans/2026-06-17-resource-action-context-prefill.md frontend/src/ConsoleController.tsx frontend/src/components/ManagementForms.tsx frontend/src/i18n.ts frontend/src/resourceLifecycleActionPlanner.ts frontend/src/styles.css frontend/tests/i18n.test.mjs frontend/tests/resourceLifecycleActionPlanner.test.mjs frontend/tests/styleTheme.test.mjs
git commit -m "Prefill resource action forms from selected context"
git push -u origin codex/resource-action-context-prefill
gh pr create --base main --head codex/resource-action-context-prefill --title "Prefill resource action forms from selected context" --body "..."
```
