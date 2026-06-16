# Resource Lifecycle Inline Actions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep operators in Resource Management while they handle the next action for each resource.

**Architecture:** Add a frontend-only action kind to the resource lifecycle model, wire rows to a presentational `onResourceAction` callback, and let `ConsoleController` either open an existing modal or navigate with a focused handoff context. No backend endpoints or new dependencies are introduced.

**Tech Stack:** React, TypeScript, Vite, Node test runner, existing i18n dictionary and CSS token system.

---

## File Map

- Modify: `frontend/src/resourceLifecycle.ts` — add `ResourceLifecycleActionKind`, `nextActionKind`, and blocker detail keys.
- Modify: `frontend/tests/resourceLifecycle.test.mjs` — assert action kinds and detail copy for each status.
- Modify: `frontend/src/components/ResourceLifecycleView.tsx` — render row actions through `onResourceAction` when provided.
- Modify: `frontend/src/components/ConsolePrimitives.tsx` — allow `ActionModalButton` to open from an external token without adding a second modal system.
- Modify: `frontend/src/ConsoleController.tsx` — prefill forms and route row actions.
- Modify: `frontend/src/types.ts` — allow permission handoff source `"registry"`.
- Modify: `frontend/src/components/AiAdminPermissionWorkbench.tsx` — show registry-specific handoff wording.
- Modify: `frontend/src/i18n.ts` — EN + zh-CN action/detail/handoff copy.
- Modify: `frontend/tests/styleTheme.test.mjs` and `frontend/tests/i18n.test.mjs` — structural and copy guards.
- Modify: `CHANGELOG.md` — user-facing release note.

---

### Task 1: Add Lifecycle Action Model

**Files:**
- Modify: `frontend/src/resourceLifecycle.ts`
- Modify: `frontend/tests/resourceLifecycle.test.mjs`

- [x] **Step 1: Write failing model tests**

Add expectations to `resource lifecycle directs incomplete MCP targets to the right next action`:

```js
assert.equal(byId["agt-missing-credential"].nextActionKind, "rotate_credential");
assert.equal(byId["agt-missing-credential"].detailKey, "resource.detail.needsCredentials");
assert.equal(byId["agt-missing-capability"].nextActionKind, "review_capabilities");
assert.equal(byId["agt-needs-approval"].nextActionKind, "start_permission_change");
assert.equal(byId["agt-disabled"].nextActionKind, "review_resource");
```

Add expectations to `resource lifecycle treats approved but unverified resources as runtime follow-up`:

```js
assert.equal(summary.items[0].nextActionKind, "review_runtime");
assert.equal(summary.items[0].detailKey, "resource.detail.needsRuntime");
```

- [x] **Step 2: Run focused test and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/resourceLifecycle.test.mjs
```

Expected: FAIL because `nextActionKind` and `detailKey` are not defined.

- [x] **Step 3: Add action kind and detail key fields**

In `frontend/src/resourceLifecycle.ts`, add:

```ts
export type ResourceLifecycleActionKind =
  | "rotate_credential"
  | "review_capabilities"
  | "start_permission_change"
  | "review_runtime"
  | "review_resource";
```

Extend `ResourceLifecycleItem`:

```ts
detailKey: string;
nextActionKind: ResourceLifecycleActionKind;
```

Set both in `buildResourceLifecycleItem`:

```ts
detailKey: detailKey(status),
nextActionKind: nextActionKind(status),
```

Add mapping helpers:

```ts
function detailKey(status: ResourceLifecycleStatus) {
  const keys: Record<ResourceLifecycleStatus, string> = {
    disabled: "resource.detail.disabled",
    needs_approval: "resource.detail.needsApproval",
    needs_capabilities: "resource.detail.needsCapabilities",
    needs_credentials: "resource.detail.needsCredentials",
    needs_runtime: "resource.detail.needsRuntime",
    ready: "resource.detail.ready"
  };
  return keys[status];
}

function nextActionKind(status: ResourceLifecycleStatus): ResourceLifecycleActionKind {
  const actions: Record<ResourceLifecycleStatus, ResourceLifecycleActionKind> = {
    disabled: "review_resource",
    needs_approval: "start_permission_change",
    needs_capabilities: "review_capabilities",
    needs_credentials: "rotate_credential",
    needs_runtime: "review_runtime",
    ready: "review_runtime"
  };
  return actions[status];
}
```

- [x] **Step 4: Run focused test and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/resourceLifecycle.test.mjs
```

Expected: PASS.

---

### Task 2: Make Row Actions Presentational And Contextual

**Files:**
- Modify: `frontend/src/components/ResourceLifecycleView.tsx`
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/i18n.test.mjs`

- [x] **Step 1: Add structural test guards**

In `styleTheme.test.mjs`, assert:

```js
assert.match(resourceLifecycleView, /onResourceAction/);
assert.match(resourceLifecycleView, /type="button"[\s\S]*onClick=\{\(\) => onResourceAction\(item\)\}/);
assert.match(resourceLifecycleView, /t\(item\.detailKey\)/);
assert.doesNotMatch(resourceLifecycleView, /ManagementForms/);
assert.doesNotMatch(resourceLifecycleView, /TechnicalId/);
```

- [x] **Step 2: Add bilingual detail copy tests**

In `i18n.test.mjs`, assert Simplified Chinese keys:

```js
assert.equal(t("resource.detail.needsCredentials"), "需要先配置目标服务凭据。");
assert.equal(t("resource.detail.needsCapabilities"), "需要发现并复核可用能力。");
assert.equal(t("resource.detail.needsApproval"), "需要创建权限变更并完成授权。");
assert.equal(t("resource.detail.needsRuntime"), "需要一次真实调用来确认运行结果。");
```

- [x] **Step 3: Run focused tests and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/i18n.test.mjs
```

Expected: FAIL before view/copy changes.

- [x] **Step 4: Update ResourceLifecycleView props and row rendering**

Update props:

```ts
onResourceAction?: (item: ResourceLifecycleItem) => void;
```

Pass it to rows:

```tsx
summary.items.map((item) => (
  <ResourceLifecycleRow item={item} key={item.id} onResourceAction={onResourceAction} t={t} />
))
```

Render detail and action:

```tsx
<span>{t(item.detailKey)}</span>
```

```tsx
{onResourceAction ? (
  <button
    className={`${requiresAction ? "primary-button" : "secondary-button"} resource-lifecycle-action`}
    type="button"
    onClick={() => onResourceAction(item)}
  >
    {t(item.nextActionKey)}
    <ArrowRight size={14} />
  </button>
) : (
  <a className={`${requiresAction ? "primary-button" : "secondary-button"} resource-lifecycle-action`} href={item.nextActionHash}>
    {t(item.nextActionKey)}
    <ArrowRight size={14} />
  </a>
)}
```

- [x] **Step 5: Add i18n detail copy**

Add EN and zh-CN:

```ts
"resource.detail.disabled": "This resource is disabled; review its registry status before using it.",
"resource.detail.needsApproval": "Create a permission change and finish authorization.",
"resource.detail.needsCapabilities": "Discover and review available capabilities.",
"resource.detail.needsCredentials": "Configure target service credentials first.",
"resource.detail.needsRuntime": "Run one real call to confirm runtime behavior.",
"resource.detail.ready": "Lifecycle checks are complete; keep reviewing runtime activity.",
```

```ts
"resource.detail.disabled": "该资源已停用，使用前请先复核注册状态。",
"resource.detail.needsApproval": "需要创建权限变更并完成授权。",
"resource.detail.needsCapabilities": "需要发现并复核可用能力。",
"resource.detail.needsCredentials": "需要先配置目标服务凭据。",
"resource.detail.needsRuntime": "需要一次真实调用来确认运行结果。",
"resource.detail.ready": "生命周期检查已完成，可继续查看运行情况。",
```

- [x] **Step 6: Run focused tests and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/i18n.test.mjs
```

Expected: PASS.

---

### Task 3: Open Existing Modals From Resource Rows

**Files:**
- Modify: `frontend/src/components/ConsolePrimitives.tsx`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/tests/styleTheme.test.mjs`

- [x] **Step 1: Add external modal-open guard tests**

Add assertions:

```js
assert.match(consolePrimitives, /openToken\?: number/);
assert.match(consolePrimitives, /useEffect\(\(\) => \{[\s\S]*if \(openToken === undefined\) return;[\s\S]*setOpen\(true\)/);
assert.match(app, /resourceActionModal/);
assert.match(app, /handleResourceLifecycleAction/);
assert.match(app, /openToken=\{resourceActionModal === "rotate_credential" \? resourceActionOpenToken : undefined\}/);
```

- [x] **Step 2: Run focused test and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
```

Expected: FAIL.

- [x] **Step 3: Extend ActionModalButton with openToken**

Add prop:

```ts
openToken?: number;
```

Pass it into `ActionModalLauncher`, then in `ActionModalLauncher`:

```ts
useEffect(() => {
  if (openToken === undefined) return;
  setOpen(true);
}, [openToken]);
```

- [x] **Step 4: Add modal state in ConsoleController**

Add state:

```ts
const [resourceActionModal, setResourceActionModal] = useState<"" | "rotate_credential" | "create_policy" | "create_key">("");
const [resourceActionOpenToken, setResourceActionOpenToken] = useState(0);
```

Add helper:

```ts
function openResourceActionModal(modal: "rotate_credential" | "create_policy" | "create_key") {
  setResourceActionModal(modal);
  setResourceActionOpenToken((token) => token + 1);
}
```

Pass tokens:

```tsx
openToken={resourceActionModal === "rotate_credential" ? resourceActionOpenToken : undefined}
```

Do the same for create policy and create key.

- [x] **Step 5: Wire row action for credential state**

Add:

```ts
function handleResourceLifecycleAction(item: ResourceLifecycleItem) {
  if (item.nextActionKind === "rotate_credential") {
    management.setRotateForm({
      ...management.rotateForm,
      agentId: item.id
    });
    openResourceActionModal("rotate_credential");
    return;
  }
  setActiveNav(navKeyFromHash(item.nextActionHash));
}
```

Pass:

```tsx
onResourceAction={handleResourceLifecycleAction}
```

- [x] **Step 6: Run focused test and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
```

Expected: PASS.

---

### Task 4: Preserve Context For Capability, Permission, And Runtime Handoffs

**Files:**
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/components/AiAdminPermissionWorkbench.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/askAccessView.test.mjs`
- Modify: `frontend/tests/i18n.test.mjs`

- [x] **Step 1: Add handoff tests**

Add assertions:

```js
assert.match(types, /sourceView: 'ask' \| 'tenants' \| 'registry'/);
assert.match(controller, /sourceView: "registry"/);
assert.match(workbench, /permissionHandoffContext\?\.sourceView === "registry"/);
```

- [x] **Step 2: Run focused tests and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/askAccessView.test.mjs tests/i18n.test.mjs
```

Expected: FAIL.

- [x] **Step 3: Extend permission handoff source**

In `frontend/src/types.ts`:

```ts
sourceView: 'ask' | 'tenants' | 'registry'
```

- [x] **Step 4: Add resource action navigation helper**

In `handleResourceLifecycleAction`, add:

```ts
if (item.nextActionKind === "review_capabilities") {
  capabilityGovernance.setForm({
    ...capabilityGovernance.form,
    targetId: item.id
  });
  userSelectedNavRef.current = true;
  setActiveNav("capabilities");
  return;
}
```

For permission change:

```ts
if (item.nextActionKind === "start_permission_change") {
  openTenantPermissionChange({
    sourceView: "registry",
    targetId: item.id,
    targetName: item.name,
    tenantId: item.tenantId,
    workspaceId: item.workspaceId,
    intentText: tx(t, "resource.permissionIntent", { target: item.name })
  });
  return;
}
```

For runtime:

```ts
if (item.nextActionKind === "review_runtime") {
  setTraceFilters((current) => ({ ...current, targetAgentId: item.id }));
  userSelectedNavRef.current = true;
  setActiveNav("traces");
  return;
}
```

- [x] **Step 5: Add registry handoff copy**

In `AiAdminPermissionWorkbench`, choose title:

```ts
const permissionHandoffTitle = permissionHandoffContext?.sourceView === "tenants"
  ? t("text.permissionHandoffTenantTitle")
  : permissionHandoffContext?.sourceView === "registry"
    ? t("text.permissionHandoffRegistryTitle")
    : t("text.permissionHandoffTitle");
```

Add EN + zh-CN:

```ts
"resource.permissionIntent": "Create authorization for {target}.",
"text.permissionHandoffRegistryTitle": "Brought in from Resource Management",
```

```ts
"resource.permissionIntent": "为 {target} 创建授权。",
"text.permissionHandoffRegistryTitle": "已从资源管理带入",
```

- [x] **Step 6: Run focused tests and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/askAccessView.test.mjs tests/i18n.test.mjs
```

Expected: PASS.

---

### Task 5: Full Verification, Browser Check, And PR

**Files:**
- Modify: `CHANGELOG.md`
- Modify: this plan file.

- [x] **Step 1: Update changelog**

Add under Unreleased:

```md
- Resource Management row actions now keep resource context while opening credential actions in place and routing capability, permission, and runtime follow-ups with focused handoff state.
- 资源管理的行级操作现在会保留资源上下文：凭据操作就地打开，能力、权限和运行后续动作带着上下文进入对应工作区。
```

- [x] **Step 2: Run focused gates**

Run:

```bash
pnpm --dir frontend exec node --test tests/resourceLifecycle.test.mjs tests/styleTheme.test.mjs tests/askAccessView.test.mjs tests/i18n.test.mjs
```

Expected: PASS.

- [x] **Step 3: Run full gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
make check
make release-check
```

Expected: PASS.

- [ ] **Step 4: Browser validation**

  Status: blocked by Browser tool URL policy on `http://127.0.0.1:5178/#registry`.
  The plugin refused reload/access for the local URL and explicitly disallowed browser-automation workarounds. Do not mark this complete until the policy allows localhost automation or a human manually verifies the five paths below.

Run the local demo stack on isolated ports if needed. Verify:

1. `#registry` resource rows show blocker detail copy.
2. A credential-blocked row opens the credential modal in place.
3. A capability-blocked row moves to `#capabilities` with the target selected.
4. An authorization-blocked row moves to `#ai-admin` and shows the Resource Management handoff notice.
5. A runtime-blocked or ready row moves to `#traces` with the target filter set.

- [ ] **Step 5: Commit, push, and open PR**

Commit:

```bash
git add CHANGELOG.md docs/superpowers/plans/2026-06-17-resource-lifecycle-inline-actions.md frontend/src frontend/tests
git commit -m "Improve resource lifecycle row actions"
git push -u origin codex/resource-lifecycle-inline-actions
gh pr create --base main --head codex/resource-lifecycle-inline-actions --title "Improve resource lifecycle row actions" --body-file /tmp/resource-lifecycle-inline-actions-pr.md
```
