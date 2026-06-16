# Production Acceptance Center Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a focused go-live check center that tells operators whether the current permission change can go live and what must be fixed next.

**Architecture:** Reuse the backend `production-readiness` response. Add a frontend `productionAcceptance.ts` pure model, wire it into `GoLiveAcceptanceOverview`, update bilingual copy, and extend release smoke source guards.

**Tech Stack:** React, TypeScript, Node test runner, existing shell release smoke scripts.

---

### Task 1: Red Tests For Acceptance Model

**Files:**
- Create: `frontend/tests/productionAcceptance.test.mjs`
- Create: `frontend/src/productionAcceptance.ts`

- [x] **Step 1: Add failing tests**

Create `frontend/tests/productionAcceptance.test.mjs` with tests for:

- ready status chooses `export_acceptance_report`
- blocked permission status chooses `open_permission_change`
- connection warning chooses `run_diagnostics`
- fallback data blocks production actions

- [x] **Step 2: Run red test**

Run: `pnpm --dir frontend exec node --test tests/productionAcceptance.test.mjs`

Expected: FAIL because `productionAcceptance.ts` does not export the model yet.

### Task 2: Implement Pure Model

**Files:**
- Modify: `frontend/src/productionAcceptance.ts`

- [x] **Step 1: Implement model**

Export `buildProductionAcceptanceCenter(input)` returning:

- `status`
- `primaryAction`
- `readyCount`
- `totalCount`
- `blockingCount`
- `checkRows`

- [x] **Step 2: Run focused tests**

Run: `pnpm --dir frontend exec node --test tests/productionAcceptance.test.mjs`

Expected: PASS.

### Task 3: Wire Go-Live Overview

**Files:**
- Modify: `frontend/src/components/GoLiveAcceptanceOverview.tsx`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/src/styles.css`

- [x] **Step 1: Pass connection diagnostics state into the overview**

`ConsoleController.tsx` passes `connectionDiagnosticsStatus`.

- [x] **Step 2: Render the acceptance model**

`GoLiveAcceptanceOverview` uses `buildProductionAcceptanceCenter` for status, counts, primary action, and blockers.

- [x] **Step 3: Update bilingual copy**

Add EN and zh-CN keys for the go-live check center.

- [x] **Step 4: Add minimal CSS**

Keep the first viewport simple: decision, blockers, compact checklist, context.

### Task 4: Source Guards And Docs

**Files:**
- Modify: `frontend/tests/permissionFlowLayout.test.mjs`
- Modify: `frontend/tests/styleTheme.test.mjs`
- Modify: `frontend/tests/i18n.test.mjs`
- Modify: `scripts/scenario-web-console-production-journey.sh`
- Modify: `tests/makefile_targets_test.sh`
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Add structural assertions**

Guard that `GoLiveAcceptanceOverview` imports and uses `productionAcceptance.ts`.

- [x] **Step 2: Extend release smoke source checks**

The web console production journey script must fetch `src/productionAcceptance.ts` and assert `buildProductionAcceptanceCenter`.

- [x] **Step 3: Update docs**

Document that release smoke now covers the go-live check center.

### Task 5: Verification

- [x] **Step 1: Run focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/productionAcceptance.test.mjs tests/permissionFlowLayout.test.mjs tests/styleTheme.test.mjs tests/i18n.test.mjs
```

- [x] **Step 2: Run full gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
git diff --check
```
