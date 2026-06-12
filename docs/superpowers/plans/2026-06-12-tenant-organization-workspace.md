# Tenant Organization Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a discoverable tenant workspace and clarify permission ownership while improving primary button prominence.

**Architecture:** A pure presenter builds tenant hierarchy data. A focused React view renders the tenant workspace and delegates workflow transitions to `ConsoleController`. Global button CSS changes stay token-based.

**Tech Stack:** React 19, TypeScript, Vite, Node test runner, existing CSS token system.

---

### Task 1: Tenant Presenter

**Files:**
- Create: `frontend/src/tenantOrganization.ts`
- Test: `frontend/tests/tenantOrganization.test.mjs`

- [ ] Build pure functions that derive tenant hierarchy nodes, selected tenant summary, workspace list, active agents, and permission counts from existing console arrays.
- [ ] Add tests for hierarchy ordering, fallback selection, and selected-tenant summary.
- [ ] Run `pnpm --dir frontend test -- tenantOrganization.test.mjs`.

### Task 2: Tenant Workspace UI

**Files:**
- Create: `frontend/src/components/TenantOrganizationView.tsx`
- Modify: `frontend/src/components/ConsoleViews.tsx`
- Modify: `frontend/src/styles.css`
- Test: `frontend/tests/styleTheme.test.mjs`

- [ ] Render a two-column tenant workspace with hierarchy, selected tenant summary, workspace list, agent list, and workflow actions.
- [ ] Add token-based CSS classes for the tenant workspace.
- [ ] Extend visual structure tests to guard against orphaned constant forms and ensure the tenant page has a primary workflow action.
- [ ] Run `pnpm --dir frontend test -- styleTheme.test.mjs`.

### Task 3: Navigation And Handoff

**Files:**
- Modify: `frontend/src/consoleNavigation.ts`
- Modify: `frontend/src/gettingStarted.ts`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/types.ts`
- Test: `frontend/tests/consoleNavigation.test.mjs`
- Test: `frontend/tests/gettingStarted.test.mjs`

- [ ] Add the `tenants` nav key and route.
- [ ] Point the tenant setup step to `#tenants`.
- [ ] Wire tenant page actions to existing permission-change and access-profile flows.
- [ ] Run focused navigation and getting-started tests.

### Task 4: I18n And Button Hierarchy

**Files:**
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/src/styles.css`
- Test: `frontend/tests/i18n.test.mjs`
- Test: `frontend/tests/styleTheme.test.mjs`

- [ ] Add EN and zh-CN strings for tenant workspace labels, empty states, and actions.
- [ ] Strengthen `.primary-button` using existing brand tokens only.
- [ ] Keep secondary buttons neutral and semantic colors reserved for status.
- [ ] Run focused i18n and style tests.

### Task 5: Verification

**Files:**
- Modify: `CHANGELOG.md`

- [ ] Update CHANGELOG with tenant workspace and button hierarchy changes.
- [ ] Run `pnpm --dir frontend test`.
- [ ] Run `pnpm --dir frontend build`.
- [ ] Run `make check`.
- [ ] Run `make release-check`.
- [ ] Browser-check `#tenants`, `#ai-admin`, and `#access` for visible navigation and handoff.
