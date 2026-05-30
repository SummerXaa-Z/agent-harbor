# Sprint 7 Brief: Agent Update and Credential Rotation

Date: 2026-05-30
Status: Implementing on `codex/sprint-7-credential-rotation`

## Goal

Let operators safely update an existing Agent and rotate its upstream credentials without deleting and recreating the Agent.

## User Stories

- As an operator, I can update an Agent's name, description, owner, status, and non-secret channel config.
- As an operator, I can rotate upstream credentials for an Agent without exposing plaintext in any response.
- As an integrator, rotated credentials take effect on the next proxied MCP/OpenAPI call.
- As a developer, update and rotation reuse the same validation rules as create, including SSRF and secret-like `channelConfig` checks.

## Acceptance

- `PATCH /api/v1/agents/{id}` updates partial Agent metadata, status, and full `channelConfig` replacement.
- `POST /api/v1/agents/{id}/credentials:rotate` replaces the Agent credential bag.
- `tenantId`, `workspaceId`, and `channelType` are immutable in Sprint 7.
- Updating an active endpoint-required Agent validates the effective `channelConfig.endpoint`.
- `channelConfig` still rejects secret-like keys and unsafe outbound URLs.
- Credential rotation validates credential key/value shape and `credentialHeaders` references.
- Agent responses, list responses, and rotate responses never include plaintext credentials.
- PostgreSQL stores rotated credentials through the existing AES-GCM ciphertext path.

## Non-goals

- No per-credential patch or merge semantics; rotation replaces the full credential bag.
- No credential version history, rollback, or key version metadata.
- No cross-channel migration; changing `channelType` still requires a new Agent.
