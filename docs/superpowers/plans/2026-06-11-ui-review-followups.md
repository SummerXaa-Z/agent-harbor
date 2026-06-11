# Console UI Review Follow-Ups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close every finding in `docs/engineering/ui-review-2026-06-11.md` without weakening safety or production readiness. The primary journey remains configure -> approval -> apply -> status check -> acceptance.

**Architecture:** Keep behavior stable while reducing UI complexity in bounded increments. Each increment must include frontend behavior, i18n where copy changes, docs or changelog notes, and focused tests before wider gates.

**Tech Stack:** React, TypeScript, Vite, Node test runner, Makefile release gates, branch `codex/production-readiness-gate`.

---

## Product Consensus

- Platform admins and security reviewers should see business terms first; protocol IDs and selectors belong in details.
- Language switching must re-render user-facing state from translation keys, not from stored strings.
- Lists used for operating production systems need search, filtering, and a details path before real data volume arrives.
- "Advanced settings" cannot mean technical override, trace details, and filters at the same time.
- Documentation and in-product guidance should lower the concept load for first-time operators.

## Task 1: Fix Permission Journey Result Message Language Snapshots (`ui-2`)

- [x] **Step 1: Add a regression test** in `frontend/tests/permissionJourneySafety.test.mjs` that requires `aiAdminMessage` to store translation descriptors instead of rendered strings.
- [x] **Step 2: Replace string state** in `frontend/src/App.tsx` with `LocalizedMessage | null` and render through `localizedMessageText(...)`.
- [x] **Step 3: Convert approval/apply/evidence result messages** to `{ key, params }` descriptors or localized render closures for dynamic readiness details.
- [x] **Step 4: Run focused gate** for permission journey safety, layout, i18n, and frontend build.
- [x] **Step 5: Browser-check language switching** on the permission journey result message.
- [x] **Step 6: Record changelog/docs evidence** for the increment.

## Task 2: Split Overloaded Advanced Controls (`ui-4`)

- [ ] Rename technical override, trace details, and filter controls with separate i18n keys.
- [ ] Add expand/collapse-all controls where repeated audit trace rows are shown.
- [ ] Test copy semantics and visible controls.

## Task 3: Tighten Self-Check and Onboarding Surfaces (`ui-6`, `ui-7`)

- [ ] Align self-check rows into label/value grids and reuse copyable technical detail affordances.
- [ ] Split README onboarding sections into short numbered steps.
- [ ] Add lightweight in-product concept guidance for the permission journey.

## Task 4: Add Search, Filters, and Details to Operational Lists (`ui-3`)

- [ ] Add keyword and status filters to Agent registry.
- [ ] Add search/grouping affordances to capability lists.
- [ ] Add a details entry that does not overload row actions.

## Task 5: Rework Access Policy Page Information Architecture (`ui-5`)

- [ ] Make the policy page action-oriented when empty.
- [ ] Move or fold management audit content so it does not dominate the empty state.
- [ ] Add focused tests for page structure.

## Task 6: Extract Views From `App.tsx` (`ui-1`)

- [ ] Extract view components by `NavKey` without changing behavior.
- [ ] Keep `App.tsx` as shell/orchestration and move presentational blocks into owned files.
- [ ] Run full frontend tests and release gates after each safe extraction batch.

## Verification Gates

- Focused frontend tests after each task.
- `pnpm --dir frontend build` for UI changes.
- `git diff --check` before commit.
- `make check` at major milestones.
- `make release-check` before considering the review fully closed.
