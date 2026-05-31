# Sprint 10 Brief: Route Policy Retry Overrides

Status: Implementing on `codex/sprint-10-route-policy-retry-overrides`

## Goal

Let RoutePolicy objects carry optional retry overrides so operators can tune upstream resiliency per governed route instead of only per target Agent.

## User Stories

- As a platform operator, I can set retry attempts/backoff/status codes on a specific allow policy.
- As a runtime operator, the matched allow policy retry override takes precedence over the target Agent `channelConfig.retry`.
- As a developer, legacy access grants and policies without retry overrides keep using existing target Agent retry behavior.

## Acceptance Criteria

- RoutePolicy responses include optional `retry`.
- `POST /api/v1/route-policies` accepts optional `retry`.
- `PATCH /api/v1/route-policies/{id}` can set, replace, or clear `retry`.
- Route policy retry validation reuses the same bounds as target Agent retry: `maxAttempts` 1-4, `backoffMs` 0-1000, `statusCodes` 5xx only.
- Data-plane proxy uses matched allow policy retry before target Agent retry.
- Deny policies do not affect upstream retry because they never proxy.
- Memory and PostgreSQL repositories persist and evaluate retry overrides consistently.
- Console Create Policy can set retry attempts/backoff.

## Non-goals

- No retry inheritance editor or per-method templates.
- No non-5xx retry status expansion.
- No retry override for legacy access grants.
- No retry-related metrics beyond existing `upstreamAttempts`.
