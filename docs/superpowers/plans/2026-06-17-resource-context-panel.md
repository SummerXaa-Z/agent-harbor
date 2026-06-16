# Resource Context Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a selected-resource context panel to Resource Management so operators keep resource scope while taking lifecycle actions.

**Architecture:** `ResourceLifecycleView` owns local selection and renders a presentational context panel. `ConsoleController` passes tenant/workspace formatter callbacks. Row next actions continue through `planResourceLifecycleAction`.

**Tech Stack:** React, TypeScript, existing CSS tokens, existing Node test runner.

---

## File Map

- Modify: `frontend/src/components/ResourceLifecycleView.tsx` — selected-resource state, row selection, context panel.
- Modify: `frontend/src/ConsoleController.tsx` — inject tenant/workspace display formatters.
- Modify: `frontend/src/i18n.ts` — paired EN and zh-CN copy.
- Modify: `frontend/src/styles.css` — compact B-end context panel styling.
- Modify: `frontend/tests/styleTheme.test.mjs` — structure and visual regression guards.
- Modify: `frontend/tests/i18n.test.mjs` — copy regression guard.
- Modify: `CHANGELOG.md` — bilingual user-facing note.

---

### Task 1: Add Failing Guards

- [x] **Step 1: Add ResourceLifecycleView structure guards**

Update `frontend/tests/styleTheme.test.mjs` to require:

- `useState` in `ResourceLifecycleView`,
- `selectedResourceId`,
- `defaultSelectedItem`,
- `resource-lifecycle-context-panel`,
- `formatTenantName` and `formatWorkspaceName` props,
- row selection button separate from `onResourceAction`.

- [x] **Step 2: Add i18n guards**

Update `frontend/tests/i18n.test.mjs` for EN/zh-CN resource context labels:

- `resource.contextTitle`
- `resource.contextDetail`
- `resource.contextScope`
- `resource.contextHealth`
- `resource.contextNext`

- [x] **Step 3: Confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/i18n.test.mjs
```

Expected: FAIL because the context panel and strings do not exist yet.

---

### Task 2: Implement Resource Context Panel

- [x] **Step 1: Add view props and selected-resource state**

In `ResourceLifecycleView`, add optional formatter props:

```ts
formatTenantName?: (tenantId: string) => string;
formatWorkspaceName?: (workspaceId: string) => string;
```

Use local `selectedResourceId` and derive:

```ts
const defaultSelectedItem = summary.items.find((item) => item.status !== "ready") ?? summary.items[0];
const selectedItem = summary.items.find((item) => item.id === selectedResourceId) ?? defaultSelectedItem;
```

- [x] **Step 2: Render context panel**

Render a `resource-lifecycle-context-panel` before the list when `selectedItem` exists. Include readable resource name, kind, status badge, blocker detail, tenant/workspace labels, capability count, permission count, runtime count, and next-action button.

- [x] **Step 3: Add row selection**

Make the resource name a button that calls `onSelectResource(item.id)`, marks the selected row, and keeps the existing next-action button unchanged.

- [x] **Step 4: Inject display formatters from controller**

Pass:

```tsx
formatTenantName={(tenantId) => permissionTenantPathLabel(tenantId, tenants, t).primary}
formatWorkspaceName={(workspaceId) => permissionWorkspaceDisplayName(workspaceId, agents, t)}
```

---

### Task 3: Style, Copy, And Verify

- [x] **Step 1: Add compact styles**

Add styles for:

- `.resource-lifecycle-context-panel`
- `.resource-lifecycle-context-main`
- `.resource-lifecycle-context-grid`
- `.resource-lifecycle-resource-button`
- `.resource-lifecycle-row.is-selected`

Use existing neutral surfaces, enterprise blue focus/selection, 12-14px text, and no decorative shadows.

- [x] **Step 2: Add EN and zh-CN copy plus changelog**

Update `frontend/src/i18n.ts` and `CHANGELOG.md`.

- [x] **Step 3: Run focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/i18n.test.mjs
```

Expected: PASS.

- [x] **Step 4: Run full gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
make check
make release-check
```

Expected: PASS.

- [ ] **Step 5: Commit, push, open PR, and merge when CI is green**

Run:

```bash
git add CHANGELOG.md docs/superpowers/plans/2026-06-17-resource-context-panel.md docs/superpowers/specs/2026-06-17-resource-context-panel-design.md frontend/src frontend/tests
git commit -m "Add resource lifecycle context panel"
git push -u origin codex/resource-lifecycle-context-panel
gh pr create --base main --head codex/resource-lifecycle-context-panel --title "Add resource lifecycle context panel" --body "..."
```
