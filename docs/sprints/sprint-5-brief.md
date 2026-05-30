# Sprint 5 Brief: Proxy Retry and Error Classification

Date: 2026-05-30
Status: Implementing on `codex/sprint-5-proxy-retry-classification`

## Goal

Make upstream proxy failures easier to operate by adding explicit per-Agent retry policy and more precise gateway error codes.

## User Stories

- As an operator, I can opt a target Agent into a small bounded retry policy for transient upstream failures.
- As an integrator, I can keep default calls single-attempt to avoid surprising non-idempotent tool execution.
- As an operator, I can distinguish timeout, DNS, TLS, connection, and generic upstream failures from API responses.
- As a developer, I can verify how many upstream attempts were made without reading server internals.

## Acceptance

- `channelConfig.retry.maxAttempts` accepts integers from 1 to 4. Default is 1.
- `channelConfig.retry.backoffMs` accepts integers from 0 to 1000. Default is 0.
- `channelConfig.retry.statusCodes` accepts 500-599 status codes. Default retry status codes are 502, 503, and 504.
- Network errors and configured retryable 5xx responses are retried until `maxAttempts` is exhausted.
- Successful retry returns the upstream response and header `X-AgentHarbor-Upstream-Attempts`.
- Exhausted retryable status responses return the last upstream response and the same attempt header.
- Proxied request bodies over 4MiB return `413 PAYLOAD_TOO_LARGE`.
- Gateway-generated errors classify failures as `UPSTREAM_TIMEOUT`, `UPSTREAM_DNS_ERROR`, `UPSTREAM_TLS_ERROR`, `UPSTREAM_CONNECT_ERROR`, or fallback `UPSTREAM_ERROR`.

## Non-goals

- No circuit breaker, queueing, jitter, or global retry budget.
- No route-level retry override yet.
- No metrics API yet; attempt count is exposed as a response header for now.
