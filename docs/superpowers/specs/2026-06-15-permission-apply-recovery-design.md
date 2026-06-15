# Permission Apply Recovery Design

## Context

AgentHarbor's main production journey is now centered on permission change configuration, approval, application, status checking, and acceptance. The backend already applies approval-required permission packages through a single store mutation, consumes the approval request once, and rejects retries that attempt to reuse a consumed approval. That protects data integrity, but the operator experience still has a recovery gap: when the apply request succeeds and a later UI refresh fails, or when an operator retries with an already consumed approval, the console can look like a generic failure instead of a recoverable state.

For a production control plane, this is a trust issue. Operators need to know whether the permission change may already be applied and which status screen to verify next.

## Approaches Considered

1. **Introduce full idempotency keys for permission package application.** This is the strongest retry model, but it requires API contract changes, persistence-level uniqueness, migration decisions, and careful replay semantics. It is too large for the current hardening slice.
2. **Return a stable consumed-approval error and teach the console to recover around it.** This preserves the current security semantics, keeps the backend transaction model unchanged, and lets the UI give a precise next step.
3. **Only change the copy in the frontend.** This is low effort but fragile because it depends on English backend message text and does not give tests a stable product contract.

The selected design is approach 2.

## Design

Add a small, explicit recovery contract for approval-consumption retries:

- The permission package apply API returns a stable error code when the referenced approval request has already been consumed.
- The scenario script asserts that consumed approval retry returns that stable code and still creates only one application.
- The frontend API client preserves backend error codes on `ApiRequestError`.
- The permission change console maps the consumed-approval error code to a localized recovery message.
- The permission change console attempts a best-effort state refresh after this error so the operator can see the current application health/readiness instead of being stranded on a generic failure.

The product message should be direct: the approval was already used; refresh or check current permission change status before retrying; do not submit a duplicate change unless the current status proves it was not applied.

## Files

- `internal/httpapi/server.go`: return a stable consumed-approval application error code.
- `internal/httpapi/server_test.go`: assert the stable code on consumed approval retry.
- `frontend/src/api.ts`: carry the backend error code through `ApiRequestError`.
- `frontend/src/ConsoleController.tsx`: detect the consumed-approval error code, show a recovery message, and best-effort refresh application status.
- `frontend/src/i18n.ts`: add English and Simplified Chinese recovery copy.
- `frontend/tests/api.test.mjs`, `frontend/tests/permissionJourneySafety.test.mjs`, `frontend/tests/i18n.test.mjs`: prevent regression.
- `scripts/scenario-permission-package-approval.sh`: assert the stable API error code in the end-to-end approval scenario.
- `CHANGELOG.md`: document the recovery improvement.

## Non-Goals

- No database schema change.
- No idempotency-key API in this slice.
- No change to the one-time approval consumption security model.
- No new frontend dependencies.
- No visual redesign.
- No backend writes after a consumed-approval retry.

## Error Handling

The backend still rejects a consumed approval retry with `400`. The stable error code makes the failure machine-readable. The frontend treats this as a recoverable operator state, not a successful replay. If the best-effort refresh also fails, the user still gets the recovery message and can use the status-check screens manually.

## Testing

- Backend focused test proves the consumed approval retry returns the stable error code and does not duplicate application records.
- Frontend source-level tests prove `ApiRequestError` stores the code, localized recovery copy exists in both languages, and `applyAiAdminPermissionPackage` branches on the code.
- Scenario script proves the CLI production journey sees the stable code.
- Final verification runs frontend tests/build plus `make check` and `make release-check`.

## Review Notes

This is intentionally a narrow reliability slice. It improves production operator confidence without weakening the approval model or committing to a larger idempotency contract before the product has real deployment feedback.
