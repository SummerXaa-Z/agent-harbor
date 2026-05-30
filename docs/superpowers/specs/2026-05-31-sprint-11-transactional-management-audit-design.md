# Sprint 11 Design: Transactional Management Audit

## Problem

Sprint 8 made management audit events visible but intentionally best-effort. HTTP handlers currently persist the management mutation, then call `AppendAuditEvent` and ignore any error. That keeps already-committed API calls from failing, but it leaves a gap in the engineering foundation: a successful Agent, key, grant, or route policy change can exist without matching management evidence.

Sprint 11 closes that gap for the local runtime and PostgreSQL path. Successful management mutations must commit their audit event with the state change, and failures must not leave partial business state.

## Scope

Sprint 11 covers management-plane mutations that already produce audit events:

- Agent create, update, disable, and credential rotation.
- Agent Key create and revoke.
- Access Grant create and revoke.
- Route Policy create, update, and disable.

The data-plane trace path stays unchanged. Trace append failures remain non-blocking for successful proxy calls because traces describe runtime attempts and are already tested separately.

## Design

Move audited management writes behind repository-level atomic operations. Each operation takes the domain mutation input plus a fully formed `AuditEvent`, and returns the persisted domain object only after both writes succeed.

The HTTP layer remains responsible for request parsing, validation, actor detection, audit metadata construction, and response shaping. It no longer calls a separate best-effort audit append for these covered mutations.

The repository layer becomes responsible for atomicity:

- Memory store performs the domain mutation and audit append while holding one write lock.
- PostgreSQL store performs the domain mutation and audit insert in one database transaction.
- If the domain mutation fails, no audit event is appended.
- If the audit insert fails, the domain mutation is rolled back and the API returns an error.

This keeps the audit event as the durable management evidence. Sprint 11 does not introduce a separate asynchronous outbox table. The `audit_events` table is the local transactional event log; a later export sprint can read from it or add delivery state.

## Repository Contract

Add audited variants for management mutations instead of broad transaction callbacks. Explicit methods keep the interface discoverable and avoid leaking PostgreSQL transaction details into HTTP handlers.

Representative methods:

- `CreateAgentWithAudit(ctx, agent, audit)`
- `UpdateAgentWithAudit(ctx, agent, audit)`
- `RotateAgentCredentialsWithAudit(ctx, id, credentials, now, audit)`
- `DisableAgentWithAudit(ctx, id, now, audit)`
- `CreateAgentKeyWithAudit(ctx, key, audit)`
- `RevokeAgentKeyWithAudit(ctx, id, now, audit)`
- `CreateAccessGrantWithAudit(ctx, grant, audit)`
- `RevokeAccessGrantWithAudit(ctx, id, now, audit)`
- `CreateRoutePolicyWithAudit(ctx, policy, audit)`
- `UpdateRoutePolicyWithAudit(ctx, policy, audit)`
- `DisableRoutePolicyWithAudit(ctx, id, now, audit)`

Existing unaudited methods can remain for read paths, tests, setup helpers, and data-plane internals. New HTTP management handlers should use the audited methods for covered mutations.

## PostgreSQL Behavior

Each audited method opens a transaction with `BeginTx`, executes the existing mutation SQL through transaction-scoped helpers, inserts the audit event, and commits. Rollback is deferred until commit succeeds.

The audit insert should reuse the current metadata JSON marshaling behavior. If metadata cannot be marshaled or the insert fails, the transaction rolls back and the handler returns a server error.

Route policy retry JSON persistence from Sprint 10 must stay in the same transaction when policy create or update is audited.

## Memory Behavior

Memory store methods hold the existing write mutex for the full mutation plus audit append. They should avoid appending audit events until after the domain mutation has passed existence checks.

For failure-injection tests, a small test wrapper can force audit append failure for audited methods. The expected behavior is that the business mutation is not visible after the failed request.

For updates, this means the previous object state remains visible.

## HTTP Behavior

Covered management mutations now fail if their audit event cannot be persisted. The response should use the existing error envelope and return `500` for unexpected audit persistence failures.

This is a deliberate change from Sprint 8. The management plane treats audit evidence as part of the write contract, not optional telemetry.

Audit metadata remains small and secret-free:

- No credential values.
- No Agent Key plaintext or hash.
- Credential events may include credential key names and credential version.
- Route policy events may include effect, status, priority, route type, route key, and retry summary.

## Testing

Backend tests should cover both stores where practical:

- HTTP regression tests proving all covered management mutations still produce the expected audit actions.
- Failure-injection tests proving audit failure prevents a newly created object from persisting and prevents an updated object from replacing the previous state.
- PostgreSQL round-trip tests proving an audited mutation and audit event commit together.
- PostgreSQL failure test using invalid audit metadata passed directly to an audited repository method to prove rollback semantics.

Existing frontend build and retry form tests should continue to pass. Sprint 11 does not require frontend UI changes unless TypeScript types need adjustment.

## Demo

Add `scripts/demo-sprint11-transactional-audit.sh`.

The demo should run against a local API and prove:

- A normal audited management mutation produces a visible audit event.
- Credential and key values are still redacted from audit responses.
- Existing Sprint 8 audit listing filters continue to work.

Rollback-on-audit-failure should stay in automated tests because forcing that condition through the public API would require test-only runtime controls.

## Non-goals

- No external message broker.
- No asynchronous audit delivery worker.
- No OpenTelemetry export.
- No data-plane trace transaction changes.
- No route policy import/export or dry-run simulator.
- No large repository abstraction rewrite beyond the audited mutation boundary.
