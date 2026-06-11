# Capability Governance Visual Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate the Capability Governance page into a quieter production-console layout.

**Architecture:** Keep `CapabilityGovernanceView` as the owning view, add small presentational sections inside it, and use CSS grid areas to make the capability catalog primary. Grant-chain creation opens in an on-demand side panel so the default workspace stays readable. No backend or controller changes beyond panel copy.

**Tech Stack:** React, TypeScript, Vite, Node test runner, CSS tokens, i18n.

---

## Task 1: Lock Visual Structure With Focused Tests

- [x] Add assertions that capability governance renders a scope bar, primary catalog, grant launcher, side-panel form, and compact assignment list.
- [x] Add assertions that desktop nav descriptions are hidden by default and active-only.
- [x] Run `pnpm --dir frontend test -- capabilityVisualStructure`.

## Task 2: Reorder Capability Page IA

- [x] Update `CapabilityGovernanceView.tsx` so the top bar explains the current scope, the catalog comes before existing grants, and grant creation opens from a side panel.
- [x] Keep access-query handoff, approve, refresh, and grant creation handlers unchanged.
- [x] Run focused frontend tests.

## Task 3: Reduce Visual Noise

- [x] Update `frontend/src/styles.css` for compact nav descriptions, catalog-first grid areas, on-demand grant side panel, quieter borders, and business-first table rows.
- [x] Ensure mobile navigation still hides descriptions.
- [x] Run focused frontend tests and `pnpm --dir frontend build`.

## Task 4: Ship Copy And Evidence

- [x] Add EN and zh-CN keys for capability scope, catalog, grant panel, and existing grant sections.
- [x] Update `CHANGELOG.md`.
- [x] Browser-check `#capabilities` visually.
- [x] Run `make check` if focused gates pass.

## Verification Notes

- `pnpm --dir frontend test -- capabilityVisualStructure styleTheme permissionFlowLayout askAccessView emptyStateGuidance`: passed; the project test script runs the full frontend test suite.
- `pnpm --dir frontend build`: passed.
- `git diff --check`: passed.
- `make check`: passed.
- Browser visual check: passed after the user opened `http://127.0.0.1:5176/#capabilities` in the active browser tab. Captured the capability catalog and the on-demand grant side panel; DOM confirmed `hasMainInlineForm=false` and `hasSheet=true`.
