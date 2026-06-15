# Resource Lifecycle Workbench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Resource Management the visible single entry for Agent, MCP, credential, policy, capability, permission, and runtime lifecycle actions.

**Architecture:** This is a frontend composition change. Reuse existing API handlers and forms, add action slots to `ResourceLifecycleView`, and keep specialist pages as details reached from the resource workbench.

**Tech Stack:** React, TypeScript, Vite, Node test runner, existing CSS token system.

---

### Task 1: Add Failing Structure Tests

**Files:**
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/tests/consoleNavigation.test.mjs`

- [x] **Step 1: Assert resource lifecycle owns lifecycle actions**

Add expectations that `ResourceLifecycleView` accepts `primaryActions` and `secondaryActions`, and that `ConsoleController` passes existing create actions to the lifecycle panel rather than Agent Registry.

- [x] **Step 2: Assert command modal variant exists**

Add expectations that `ActionModalButton` supports a `variant` prop and that resource lifecycle actions use `variant="command"`.

- [x] **Step 3: Run the focused tests and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs
```

Expected: fail because `primaryActions`, `secondaryActions`, and command modal variant do not exist yet.

### Task 2: Implement Resource Workbench Action Slots

**Files:**
- Modify: `frontend/src/components/ResourceLifecycleView.tsx`
- Modify: `frontend/src/components/ConsolePrimitives.tsx`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/styles.css`
- Modify: `frontend/src/i18n.ts`

- [x] **Step 1: Add action slots to ResourceLifecycleView**

Add optional `primaryActions` and `secondaryActions` props and render them in a `resource-lifecycle-command-center` section after the lifecycle stages.

- [x] **Step 2: Add command modal trigger variant**

Add `variant?: "compact" | "command"` to `ActionModalButton`. Keep compact as the default. Use `action-modal-trigger-command` for the command variant.

- [x] **Step 3: Compose lifecycle actions in ConsoleController**

Create `resourceLifecyclePrimaryActions` with create Agent, create key, rotate credential, and create policy. Create `resourceLifecycleSecondaryActions` with links to capabilities and permission changes. Pass them to `ResourceLifecycleView`.

- [x] **Step 4: Remove Resource Management page dependency on Agent Registry header actions**

Render the Agent Registry panel on `#registry` without mutation action buttons. The lifecycle workbench is now the primary mutation launcher.

- [x] **Step 5: Add styling and i18n**

Add EN + zh-CN command section labels and style command buttons with clear clickable affordance, restrained B2B visual language, and existing tokens.

- [x] **Step 6: Run focused tests and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/consoleNavigation.test.mjs
```

Expected: pass.

### Task 3: Verification and Release Gate

**Files:**
- Modify: `CHANGELOG.md`

- [x] **Step 1: Update CHANGELOG**

Add an Unreleased entry in EN and zh-CN describing the resource lifecycle workbench consolidation.

- [x] **Step 2: Run frontend gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
```

Expected: both pass.

- [x] **Step 3: Run repository gates**

Run:

```bash
git diff --check
make check
make release-check
```

Expected: all pass.

- [x] **Step 4: Commit and create PR**

Commit the implementation, push `codex/resource-lifecycle-workbench`, create a ready PR, and wait for GitHub checks.
