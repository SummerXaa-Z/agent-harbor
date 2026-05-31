# Sprint 7 Design: Agent Update and Credential Rotation

## Scope

Sprint 7 adds lifecycle operations after Agent creation. The design keeps Agent identity stable and avoids route/grant churn by updating only mutable fields and credential material on the existing Agent row.

## Agent Partial Update

`PATCH /api/v1/agents/{id}` accepts optional fields:

- `name`
- `description`
- `ownerId`
- `status`
- `channelConfig`

`channelConfig` is a full replacement when present. It is not deep-merged because merge semantics for nested `headers`, `credentialHeaders`, and future metadata can become ambiguous. The effective Agent is validated after applying the patch:

- name is still required when patched.
- status must be `draft`, `active`, or `disabled`.
- existing `channelType` must still exist in the channel catalog.
- active endpoint-required channels must have a non-empty `channelConfig.endpoint`.
- `endpoint` and `specUrl` must pass outbound URL validation.
- configured headers and retry/timeout controls reuse create-time validation.
- `credentialHeaders` must reference credentials already stored on the Agent.

Immutable in Sprint 7: `tenantId`, `workspaceId`, and `channelType`.

## Credential Rotation

`POST /api/v1/agents/{id}/credentials:rotate` accepts:

```json
{
  "credentials": {
    "apiToken": "Bearer new-token"
  }
}
```

Rotation replaces the complete credential map. The request is rejected if the effective `channelConfig.credentialHeaders` references a key that is absent from the new map. Responses use the existing Agent JSON shape, where credentials are intentionally omitted.

## Persistence

The repository adds `UpdateAgent(ctx, agent)` and both memory/PostgreSQL implementations replace the mutable Agent fields and encrypted credential ciphertext. PostgreSQL uses the existing `credentials_ciphertext` column and AES-GCM helpers, so no new migration is required.

## Frontend

The console gets small operator controls rather than a full wizard:

- Registry row action to mark an Agent active/draft/disabled through `PATCH`.
- Credential rotation form with Agent selector, credential key, and new secret value.

This keeps the UI To B and dense: no modal-heavy flow, no secret echo, no hidden magic.

## Follow-up

- Add credential version metadata and audit events once an audit event table exists.
- Add scoped role checks before exposing this outside local/demo admin mode.
- Add fine-grained config patch semantics if real operators need deep merge.
