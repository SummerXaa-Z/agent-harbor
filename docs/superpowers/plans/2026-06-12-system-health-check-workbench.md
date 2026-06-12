# System Health Check Workbench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the System Self-Check page from a demo-like configuration form into a production health-check workbench.

**Architecture:** Keep existing self-check data flow and backend APIs unchanged. Recompose `CoreJourneyWorkbench` into a clear health status header, focused check task area, default-collapsed advanced configuration, and compact runtime details.

**Tech Stack:** React, TypeScript, existing CSS tokens, Node test runner, Vite.

---

### Task 1: Lock The New Information Architecture

**Files:**
- Modify: `frontend/tests/permissionFlowLayout.test.mjs`
- Modify: `frontend/src/components/CoreJourneyWorkbench.tsx`

- [x] **Step 1: Update the layout guard test**

Add assertions that `CoreJourneyWorkbench` renders:
- `core-journey-health`
- `core-journey-health-summary`
- `core-journey-task`
- `core-journey-advanced`
- `core-journey-runtime-summary`

Run: `pnpm --dir frontend test -- permissionFlowLayout.test.mjs`
Expected: FAIL until the component is updated.

- [x] **Step 2: Recompose the component**

Move the current intro, score, run button, preflight rows, runtime summary, config form, and step rows into the new IA without changing controller props or handlers.

Run: `pnpm --dir frontend test -- permissionFlowLayout.test.mjs`
Expected: PASS.

### Task 2: Apply Production UI Styling

**Files:**
- Modify: `frontend/src/styles.css`
- Modify: `frontend/tests/permissionFlowLayout.test.mjs`

- [x] **Step 1: Add CSS guards**

Assert that:
- `.core-journey-health` uses a two-column grid.
- `.core-journey-advanced summary` exists.
- `.core-journey-steps` supports compact status cards.

Run: `pnpm --dir frontend test -- permissionFlowLayout.test.mjs`
Expected: FAIL until styles are updated.

- [x] **Step 2: Update styles**

Make the first viewport read as one health-check workbench:
- Health summary and action area at the top.
- Preflight and permission-loop checks in the middle.
- Advanced configuration collapsed by default.
- Runtime IDs and technical details in compact lower sections.

Run: `pnpm --dir frontend test -- permissionFlowLayout.test.mjs`
Expected: PASS.

### Task 3: Add Bilingual Copy And Verify

**Files:**
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/i18n.test.mjs`

- [x] **Step 1: Add EN and zh-CN strings**

Add labels for health status, check task, advanced configuration, and runtime details.

Run: `pnpm --dir frontend test -- i18n.test.mjs`
Expected: PASS with translation parity.

- [x] **Step 2: Full verification**

Run:
- `pnpm --dir frontend test`
- `pnpm --dir frontend build`
- `git diff --check`
- `make check`
- `make release-check`

Expected: all commands exit 0. Vite chunk-size warning is acceptable if build exits 0.
