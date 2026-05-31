# Sprint 8 Brief: Management Audit and Credential Versions

Date: 2026-05-30
Status: Implementing on `codex/sprint-8-management-audit`

## Goal

Make control-plane changes inspectable by operators and attach a non-secret credential version to every Agent.

## User Stories

- As an operator, I can see recent management mutations such as Agent create, update, disable, and credential rotation.
- As a platform owner, I can filter management audit events by tenant, workspace, action, and resource.
- As an integrator, I can see that credential rotation advanced an Agent's credential version without exposing plaintext secrets.
- As a developer, audit events are stored in memory and PostgreSQL through the same repository contract.

## Acceptance

- Agent responses include `credentialVersion`, with `0` for Agents without credentials and `1` for Agents created with credentials.
- `POST /api/v1/agents/{id}/credentials:rotate` increments `credentialVersion` and keeps credentials redacted.
- Management mutations append audit events for Agent create, update, disable, credential rotate, Agent Key create/revoke, and Access Grant create/revoke.
- `GET /api/v1/audit/events?tenantId=&workspaceId=&action=&resourceType=&resourceId=&limit=` returns filtered management audit events.
- Audit metadata never stores plaintext credential values or Agent Key plaintext values.
- PostgreSQL migration adds persistent `agents.credential_version`, backfills existing credentialed rows, and creates `audit_events`.
- Console shows a compact management audit table alongside existing runtime traces.

## Non-goals

- No audit event actor identity beyond local/admin-key system attribution.
- No transactional outbox, rollback, or signed audit ledger.
- No credential history, rollback, or per-key versioning.
- No route-policy audit semantics beyond grant create/revoke.
