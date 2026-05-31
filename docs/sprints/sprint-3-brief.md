# AgentHarbor Sprint 3 Brief

Date: 2026-05-30
Status: Implemented

## Goal

Turn Sprint 2's coarse route proxy into a more realistic gateway control slice: MCP method-level authorization, safe upstream header injection, and bounded proxy execution.

## User Stories

- As a platform admin, I can grant `initialize`, `tools/list`, and `tools/call` independently for MCP traffic.
- As an integrator, I can configure non-secret upstream headers for a target Agent without leaking credentials into `channelConfig`.
- As an operator, I get a fast `504 UPSTREAM_TIMEOUT` instead of a hanging request when an upstream target is too slow.
- As an auditor, traces show the actual MCP method that was allowed or denied.
- As a console user, I can choose common route keys without remembering exact strings.

## Acceptance Criteria

- MCP data-plane routeKey is derived from JSON-RPC `method` and rejects malformed or missing methods with `400 VALIDATION_FAILED`.
- Existing grants for `tools/call` do not authorize `tools/list` or `initialize`.
- `channelConfig.headers` accepts only string values, rejects secret-like header names, and forwards those headers upstream.
- `channelConfig.timeoutMs` bounds upstream proxy calls with a safe default and max.
- Upstream timeout returns `504 UPSTREAM_TIMEOUT`.
- Frontend create-grant form offers MCP route key presets.
- `go test ./...`, `go vet ./...`, `go build ./...`, `pnpm -C frontend build`, and demo scripts pass.

## Non-goals

- No encrypted credential store yet.
- No streaming MCP transport.
- No retry policy yet.
- No new database migration.
