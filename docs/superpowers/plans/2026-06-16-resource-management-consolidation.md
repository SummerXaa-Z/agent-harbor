# Resource Management Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Resource Management the single visible mutation entry for Agent, MCP, credential, and policy lifecycle actions, while improving modal-form affordance and removing detective-style "evidence/证据" wording from visible product copy.

**Architecture:** Frontend-only consolidation. Reuse existing `useManagementOperations` handlers and existing forms, upgrade action modal composition/styling, route policy creation back to `#registry`, and add source tests to prevent creation actions from drifting back into specialist pages.

**Tech Stack:** React, TypeScript, Vite, Node test runner, existing CSS token system, existing i18n dictionary.

---

## File Map

- Modify: `frontend/src/components/ConsolePrimitives.tsx` — command modal trigger tone and form-modal ergonomics.
- Modify: `frontend/src/ConsoleController.tsx` — centralize resource creation actions and remove create-policy triggers from routes/policies.
- Modify: `frontend/src/components/OperationalViews.tsx` — route policy empty CTA to Resource Management.
- Modify: `frontend/src/styles.css` — command buttons, modal shell, and modal form rhythm.
- Modify: `frontend/src/i18n.ts` — bilingual copy for resource/policy guidance and evidence-wording cleanup.
- Modify: `frontend/tests/styleTheme.test.mjs` — structural and visual guard tests.
- Modify: `frontend/tests/consoleNavigation.test.mjs` — IA guard tests.
- Modify: `frontend/tests/i18n.test.mjs` — bilingual copy/evidence wording guard if needed.
- Modify: `CHANGELOG.md` — user-facing release note.

---

### Task 1: Add Red Structure Tests

**Files:**
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/tests/consoleNavigation.test.mjs`

- [x] **Step 1: Assert resource actions have primary/secondary command tones**

Add assertions to `styleTheme.test.mjs`:

```js
assert.match(consolePrimitives, /tone = "secondary"/);
assert.match(consolePrimitives, /variant\?: "compact" \| "command"/);
assert.match(consolePrimitives, /tone\?: "primary" \| "secondary"/);
assert.match(app, /createAgentAction\("command", "primary"\)/);
assert.match(app, /createKeyAction\("command", "secondary"\)/);
assert.match(app, /rotateCredentialAction\("command", "secondary"\)/);
assert.match(app, /createPolicyAction\("command", "secondary"\)/);
```

- [x] **Step 2: Assert routes and policies no longer own create-policy triggers**

Add assertions to `styleTheme.test.mjs` that the route and policy cases do not pass `createPolicyAction()`:

```js
assert.doesNotMatch(app, /RoutesView[\s\S]*routeGovernancePanel=\{routeGovernancePanel\("span-12", createPolicyAction\(\)\)\}/);
assert.doesNotMatch(app, /PoliciesView[\s\S]*routeGovernancePanel=\{routeGovernancePanel\("span-12", createPolicyAction\(\)\)\}/);
```

- [x] **Step 3: Assert policy empty state routes to Resource Management**

Add assertion:

```js
assert.match(operationalViews, /href="#registry"/);
assert.doesNotMatch(operationalViews, /querySelector<HTMLButtonElement>\("#policy-create-panel/);
```

- [x] **Step 4: Run focused tests and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs
```

Expected: fail because command tones and policy empty-state routing are not implemented yet.

---

### Task 2: Centralize Mutation Entry Points

**Files:**
- Modify: `frontend/src/components/ConsolePrimitives.tsx`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/components/OperationalViews.tsx`

- [x] **Step 1: Add tone prop to ActionModalButton**

Update `ActionModalButton` props:

```ts
tone = "secondary",
variant = "compact",
```

and type:

```ts
tone?: "primary" | "secondary";
variant?: "compact" | "command";
```

For command triggers, build:

```ts
const triggerClassName = variant === "command"
  ? `action-modal-trigger action-modal-trigger-command is-${tone}`
  : "action-modal-trigger action-modal-trigger-compact";
```

- [x] **Step 2: Let action helper functions pass tone**

Update signatures in `ConsoleController.tsx`:

```ts
function createAgentAction(variant: "compact" | "command" = "compact", tone: "primary" | "secondary" = "secondary")
function createKeyAction(variant: "compact" | "command" = "compact", tone: "primary" | "secondary" = "secondary")
function rotateCredentialAction(variant: "compact" | "command" = "compact", tone: "primary" | "secondary" = "secondary")
function createPolicyAction(variant: "compact" | "command" = "compact", tone: "primary" | "secondary" = "secondary")
```

Pass `tone={tone}` to `ActionModalButton`.

- [x] **Step 3: Mark Resource Management command hierarchy**

Update `resourceLifecyclePrimaryActions`:

```tsx
{createAgentAction("command", "primary")}
{agents.length > 0 ? (
  <>
    {createKeyAction("command", "secondary")}
    {rotateCredentialAction("command", "secondary")}
    {createPolicyAction("command", "secondary")}
  </>
) : null}
```

- [x] **Step 4: Remove create-policy action from routes/policies pages**

Update route rendering:

```tsx
routeGovernancePanel={routeGovernancePanel("span-12")}
```

for both `RoutesView` and `PoliciesView`.

- [x] **Step 5: Replace policy empty-state DOM click with link**

In `AccessPolicyWorkspace`, remove `focusPolicyForm()` and render:

```tsx
<a className="policy-empty-action" href="#registry">
  <Route size={14} />
  {t("action.openResourceManagement")}
</a>
```

- [x] **Step 6: Run focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs
```

Expected: tests related to structure now pass except styling/copy guards that are added in Task 3.

---

### Task 3: Upgrade Command And Modal Visuals

**Files:**
- Modify: `frontend/src/styles.css`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/styleTheme.test.mjs`

- [x] **Step 1: Make command center a compact task bar**

Update `.resource-lifecycle-command-center` so it uses one copy column and an action area without oversized card affordance:

```css
.resource-lifecycle-command-center {
  display: grid;
  grid-template-columns: minmax(220px, 0.85fr) minmax(0, 1.15fr);
  gap: var(--space-3);
  align-items: center;
  padding: var(--space-3);
  border: 1px solid var(--line-subtle);
  border-radius: var(--radius-control);
  background: var(--surface);
}
```

- [x] **Step 2: Style command triggers as buttons**

Ensure:

```css
.action-modal-trigger-command.is-primary { background: var(--brand); color: var(--surface); }
.action-modal-trigger-command.is-secondary { background: var(--surface); color: var(--ink); }
.action-modal-trigger-command .action-modal-trigger-affordance { border: 0; background: transparent; }
```

The primary command must not look like a passive card.

- [x] **Step 3: Improve modal shell**

Update modal panel/body:

```css
.action-modal-panel {
  width: min(680px, calc(100vw - 48px));
  border-radius: 10px;
}

.action-modal-body {
  padding: var(--space-4);
  background: var(--surface-raised);
}

.action-modal-body .control-form {
  padding: var(--space-4);
  border: 1px solid var(--line-subtle);
  border-radius: var(--radius-control);
  background: var(--surface);
}
```

- [x] **Step 4: Add style guards**

Add assertions:

```js
assert.match(styles, /\.action-modal-trigger-command\.is-primary\s*\{/);
assert.match(styles, /\.action-modal-trigger-command\.is-secondary\s*\{/);
assert.match(styles, /\.action-modal-body \.control-form\s*\{/);
assert.doesNotMatch(styles, /\.action-modal-trigger-command\s*\{[^}]*box-shadow:\s*var\(--shadow-card\);/s);
```

- [x] **Step 5: Run focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs
```

Expected: pass.

---

### Task 4: Clean Visible Product Copy

**Files:**
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/i18n.test.mjs`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Add policy/resource CTA copy**

Add EN + zh-CN:

```ts
"action.openResourceManagement": "Open Resource Management",
"action.openResourceManagement": "打开资源管理",
```

and update policy empty-state detail to tell users lifecycle changes start from Resource Management.

- [x] **Step 2: Replace visible “证据” Chinese labels**

For zh-CN values, replace user-facing `证据` with `记录`, `材料`, `报告`, or `检查` where appropriate. Do not rename internal keys.

- [x] **Step 3: Keep English primary labels away from evidence wording**

For EN values in primary UI labels, prefer `Acceptance report`, `Runtime records`, `Trace records`, `Go-Live Check`, `Handoff materials`.

- [x] **Step 4: Add/extend i18n guard**

Add an i18n test that scans zh-CN visible values and fails if any value contains `证据`.

- [x] **Step 5: Update CHANGELOG**

Add an Unreleased bullet in EN and zh-CN for resource-management consolidation and copy cleanup.

- [x] **Step 6: Run i18n tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/i18n.test.mjs
```

Expected: pass.

---

### Task 5: Full Verification And PR

**Files:**
- All touched files.

- [x] **Step 1: Run frontend focused gates**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs tests/i18n.test.mjs
```

Expected: pass.

- [x] **Step 2: Run frontend full gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
```

Expected: pass.

- [x] **Step 3: Run repository gates**

Run:

```bash
git diff --check
make check
make release-check
```

Expected: pass.

Browser validation on the isolated demo stack (`AGENT_HARBOR_DEMO_API_PORT=19090`, `MOCK_MCP_PORT=18788`, `AGENT_HARBOR_DEMO_FRONTEND_PORT=5178`) confirmed:

- `#registry` opens in local development admin mode without the login shell, shows Resource Management, and exposes one primary `创建 Agent` command when no Agent exists.
- The primary command opens an independent `.action-modal-backdrop` / `.action-modal-panel` form shell with `.action-modal-body .control-form`.
- `#policies` no longer renders an inline create-policy panel; its empty state shows `打开资源管理` with `href="#registry"`.
- Clicking `打开资源管理` returns to `#registry` and the visible Chinese page text does not include `证据`.

- [x] **Step 4: Commit and push**

Commit:

```bash
git add docs/superpowers/specs/2026-06-16-resource-management-consolidation-design.md docs/superpowers/plans/2026-06-16-resource-management-consolidation.md frontend/src frontend/tests CHANGELOG.md
git commit -m "Improve resource management workflow"
git push -u origin codex/resource-management-consolidation
```

Completed with commit `375634d` on `codex/resource-management-consolidation`.

- [x] **Step 5: Open PR**

Create a ready PR against `main` with test evidence and note that no backend behavior changed.

Ready PR opened: https://github.com/SummerXaa-Z/agent-harbor/pull/94
