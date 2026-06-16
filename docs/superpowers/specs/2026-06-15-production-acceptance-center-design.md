# Production Acceptance Center Design

## Goal

Make the final production journey answer one operator question: can this permission change go live now, and if not, what is the next blocking item?

## Design Consensus

- Use the existing backend `permission-packages/production-readiness` response as the source of truth for permission-change readiness. Do not add a second backend readiness endpoint in this slice.
- Add a frontend-only acceptance model that combines live data availability, connection diagnostics, the permission-change status summary, and production readiness into a single operator-facing state.
- Rename the visible go-live workspace toward "Go-Live Check" / "上线检查" so the page reads like an operational gate, not an archive.
- Keep historical acceptance records, runtime logs, and management audit available below the main check, but they must not be the primary first-viewport task.

## Scope

### In Scope

- A pure `productionAcceptance.ts` model with stable statuses, check rows, primary action intent, and blocking counts.
- `GoLiveAcceptanceOverview` consumes the model instead of rebuilding status, next action, and counts inline.
- Bilingual copy for the new operator language.
- Tests that verify the ready, blocked, and connection-attention states.
- Release smoke source guards so the production journey gate covers the new model and view wiring.

### Out of Scope

- New backend persistence or a new readiness endpoint.
- Visual redesign outside the go-live check workspace.
- Changing the permission-package approval, apply, or runtime validation workflow.

## Operator Experience

The go-live page starts with a single decision panel:

1. Status badge: ready, blocked, attention needed, or pending.
2. Primary action: run status check, open permission change, export acceptance report, or run diagnostics.
3. Blocking list: only the items that need operator action.
4. Current permission-change context: tenant, workspace, caller, target, and permission package.

The page then shows supporting history and audit panels. This preserves auditability without forcing users to understand tables before they know the current state.

## Risk Controls

- The model is pure and unit-tested, so the UI does not become another state machine.
- The existing backend production-readiness contract remains authoritative.
- If connection diagnostics are unknown, the center shows an attention state rather than silently claiming readiness.
- If sample/fallback data is visible, the center blocks production actions.

## Validation

- `frontend/tests/productionAcceptance.test.mjs` covers ready, blocked, connection warning/error, and fallback data states.
- Existing i18n parity tests keep EN and zh-CN keys aligned.
- `scripts/scenario-web-console-production-journey.sh` checks the served source contains the acceptance model and go-live view wiring.
