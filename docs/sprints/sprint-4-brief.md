# Sprint 4 Brief: Secret Header Injection

Date: 2026-05-30
Status: Implemented on `codex/sprint-4-secret-header-injection`

## Goal

Let a governed target Agent call credentialed upstream MCP/OpenAPI services without putting secrets in `channelConfig`.

## User Stories

- As an integrator, I can register an upstream credential separately from `channelConfig`.
- As an operator, I can map a credential to a secret-bearing upstream header such as `Authorization`.
- As a platform owner, I can inspect Agent records without management API responses leaking credential plaintext.
- As a developer, I can keep non-secret upstream headers in `channelConfig.headers` while secret headers remain blocked there.

## Acceptance

- `POST /api/v1/agents` accepts a `credentials` object at Agent create time.
- `channelConfig.credentialHeaders` maps upstream header names to credential keys.
- Create/get/list Agent responses never contain the `credentials` object or plaintext secret values.
- Allowed data-plane proxy requests inject credential headers before calling the upstream target.
- Missing credential references fail with `400 VALIDATION_FAILED`.
- PostgreSQL persists non-empty credentials as AES-GCM ciphertext using `AGENT_HARBOR_CREDENTIAL_KEY`.
- PostgreSQL mode fails fast when `AGENT_HARBOR_CREDENTIAL_KEY` is missing.
- Frontend Create Agent form supports one credential header mapping.

## Non-goals

- No credential update or rotation API.
- No multi-secret UI builder beyond one create-time mapping.
- No external KMS integration.
