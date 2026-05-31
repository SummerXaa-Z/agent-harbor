# Sprint 11 Brief: Transactional Management Audit

Status: Planned after Sprint 10 closeout

## Goal

Make management audit events part of the write contract so successful control-plane mutations cannot commit without matching audit evidence.

## User Stories

- As a platform operator, every successful Agent, key, grant, or route policy mutation has durable audit evidence.
- As an engineer, audit persistence failures roll back covered PostgreSQL management mutations.
- As a developer, memory and PostgreSQL repositories expose the same audited mutation semantics.

## Acceptance

- Covered HTTP management handlers call audited repository methods.
- Memory store appends the audit event under the same write lock as the mutation.
- PostgreSQL store writes mutation and audit event in one transaction.
- Audit metadata remains secret-free.
- PostgreSQL rollback test proves duplicate audit insert failure prevents the Agent row from persisting.
- Existing audit listing filters continue to work.

## Non-goals

- No external outbox worker.
- No OpenTelemetry export.
- No data-plane trace transaction change.
- No route policy import/export.
