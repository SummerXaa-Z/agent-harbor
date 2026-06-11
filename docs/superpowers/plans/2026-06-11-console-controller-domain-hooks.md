# Console Controller Domain Hooks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce the `ConsoleController.tsx` state monolith by moving non-AI-admin domains into owned hooks, while adding guardrails that stop `AiAdminPermissionWorkbench.tsx` from silently growing.

**Architecture:** Keep the page shell and panel composition in `ConsoleController`, but move state, effects, and handlers for management forms, capability governance, access profile, and core journey into hooks under `frontend/src/hooks/`. The main permission-package journey remains in the controller for this batch because it is the highest-risk state machine and should be split in a dedicated follow-up.

**Tech Stack:** React hooks, TypeScript, Vite, Node test runner, existing static architecture tests, Makefile release gates.

---

## Design Consensus

- A hook must own a business domain, not just wrap a collection of `useState` calls.
- This batch targets lower-risk domains first: management operations, capability governance, access profile, and core journey.
- `ConsoleController.tsx` may still contain the AI Admin permission-package journey after this batch, but it should have materially fewer local state cells and handlers.
- `AiAdminPermissionWorkbench.tsx` gets a size/structure guard so future work must split subcomponents instead of adding more sections to the same large file.

## Task 1: Add Architecture Regression Tests

- [x] Add tests that require domain hooks to exist for management operations, capability governance, access profile, and core journey.
- [x] Add tests that prevent `ConsoleController.tsx` from owning management/capability/access/core journey submit handlers after extraction.
- [x] Add a size guard for `AiAdminPermissionWorkbench.tsx` and document that new major sections must be separate components.
- [x] Run focused architecture tests and confirm they fail before implementation.

## Task 2: Extract Management Operations Hook

- [x] Create `frontend/src/hooks/useManagementOperations.ts`.
- [x] Move agent/key/credential/policy form state, submit handlers, status mutation handlers, and cleanup action state into the hook.
- [x] Keep retry validation, localized errors, and refresh callbacks behavior-equivalent.
- [x] Wire `ConsoleController.tsx` to the hook result.
- [x] Run focused architecture tests and TypeScript.

## Task 3: Extract Capability Governance Hook

- [x] Create `frontend/src/hooks/useCapabilityGovernanceController.ts`.
- [x] Move capability grant form state, default target/capability/caller selection effect, refresh/approve/grant handlers, and action message state into the hook.
- [x] Keep live API/local fallback behavior unchanged.
- [x] Wire `ConsoleController.tsx` to the hook result.
- [x] Run focused architecture tests and TypeScript.

## Task 4: Extract Access Profile Hook

- [x] Create `frontend/src/hooks/useAccessProfileController.ts`.
- [x] Move access filters, loading/message/profile/handoff/explanation state, auto-load effect, refresh/explain handlers, and reset helpers into the hook.
- [x] Expose a small `clearForPermissionChangeHandoff(...)` entry used by AI Admin completion exits.
- [x] Wire `ConsoleController.tsx` to the hook result.
- [x] Run focused architecture tests and TypeScript.

## Task 5: Extract Core Journey Hook

- [x] Create `frontend/src/hooks/useCoreJourneyController.ts`.
- [x] Move core journey form/config/message/running/result/preflight state plus refresh/reset/run handlers into the hook.
- [x] Keep data refresh, trace filters, access profile updates, and management scope updates identical.
- [x] Wire `ConsoleController.tsx` to the hook result.
- [x] Run focused architecture tests and TypeScript.

## Task 6: Document, Verify, and Ship

- [x] Update `CHANGELOG.md` with the controller-hook split and workbench growth guard.
- [x] Update this plan with completed checkboxes and measured line/state reduction.
- [x] Run `pnpm --dir frontend test`, `pnpm --dir frontend build`, `git diff --check`, `make check`, and `make release-check`.
- [x] Browser-smoke `#ai-admin`, `#registry`, `#capabilities`, `#access`, and `#cockpit`.
- [ ] Commit, push, and confirm PR checks.

## Implementation Notes

- `ConsoleController.tsx` moved from 3754 lines / 81 `useState` calls before the domain-hook split to 2888 lines / 51 total `useState` calls after the split.
- `AiAdminPermissionWorkbench.tsx` is held at 1798 lines with an architecture guard capped at 1850 lines, so future large sections must move into separate components.
- Focused verification passed with `pnpm --dir frontend exec tsc -p tsconfig.json --noEmit` and `pnpm --dir frontend exec node --test tests/styleTheme.test.mjs tests/permissionFlowLayout.test.mjs tests/permissionJourneySafety.test.mjs`.
- Full verification passed with `pnpm --dir frontend test`, `pnpm --dir frontend build`, `git diff --check`, `make check`, and `make release-check`.
- Browser smoke passed on `#ai-admin`, `#registry`, `#capabilities`, `#access`, and `#cockpit` with no current console errors from a clean browser tab.
