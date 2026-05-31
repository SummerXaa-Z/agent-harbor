# Sprint 8 Management Audit Design

Sprint 8 adds management-plane evidence to complement the existing data-plane trace evidence. It focuses on changes operators make through the API: Agent lifecycle updates, credential rotation, Agent Key lifecycle, and Access Grant lifecycle.

## Model

`Agent` gains `credentialVersion`. Version `0` means no credential material has been stored. Version `1` is assigned when an Agent is created with credentials. Each full credential rotation increments the version by one. Agent metadata or `channelConfig` updates do not change the version.

`AuditEvent` records one immutable management event:

- `id`
- `tenantId`
- `workspaceId`
- `actor`
- `action`
- `resourceType`
- `resourceId`
- `summary`
- `metadata`
- `createdAt`

Actions use stable dotted names such as `agent.created`, `agent.updated`, and `agent.credentials_rotated`. Metadata is intentionally small and secret-free. Credential events may include credential key names and the new credential version, but never credential values.

## API

`GET /api/v1/audit/events` is an admin endpoint. It accepts the same tenant/workspace scope style as existing management endpoints and adds `action`, `resourceType`, `resourceId`, and `limit` filters. Results are ordered by `createdAt` ascending, then `id` ascending, matching the rest of the in-memory and PostgreSQL stores. `limit` defaults to 100 and is capped at 500.

Mutation handlers append audit events after successful persistence. The current repository contract has no transaction boundary across mutation plus audit append, so Sprint 8 keeps the append path best-effort: audit persistence failure must not turn an already committed mutation into a failed API response. A later outbox sprint can make this fully transactional.

## Storage

The memory store keeps audit events in an append-only slice and filters in memory. Credential rotation uses a dedicated repository operation so version increments happen in the store instead of the HTTP handler.

PostgreSQL migration `004_sprint8_management_audit.sql` adds:

- `agents.credential_version integer not null default 0`, with a backfill to version `1` for rows that already have encrypted credentials
- `audit_events` with JSONB metadata and indexes for tenant/workspace, action, resource, and time scans

The existing encrypted credential persistence path remains unchanged.

## Console

The console loads management audit events with the rest of the operational data and renders a compact table. It is read-only in Sprint 8. The table emphasizes action, resource, summary, credential version, and timestamp so operators can scan recent control-plane changes without reading raw JSON.

## Testing

Backend tests cover:

- Agent create/update/credential-rotate audit events and redaction.
- Scope/action/resource filters.
- Credential version increment behavior.
- PostgreSQL round trip for `credentialVersion` and `AuditEvent`.

Frontend checks cover TypeScript build and local UI smoke through the existing Vite preview path.
