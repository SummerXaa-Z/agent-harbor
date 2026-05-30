# AgentHarbor Sprint 2 Brief

Date: 2026-05-29
Status: Implemented

## Goal

Make AgentHarbor feel like the first usable gateway slice: scoped management views, cleanup/revoke controls, and real upstream forwarding for allowed routes.

## User Stories

- As a platform admin, I can filter management data by tenant and workspace while keeping local defaults easy.
- As a platform admin, I can disable an Agent so existing keys no longer authenticate.
- As a platform admin, I can revoke an Access Grant so later calls are denied and traced.
- As an integrator, I can register an MCP or OpenAPI target endpoint and see allowed calls forwarded upstream.
- As an auditor, I can filter traces by tenant/workspace as well as run/caller/target/decision.

## Acceptance Criteria

- `go test ./...`, `go vet ./...`, and `go build ./...` pass.
- `pnpm -C frontend build` passes.
- Demo script proves grant revoke changes allowed traffic back to denied.
- Backend tests prove disabled agents cannot authenticate via old keys.
- Backend tests prove MCP and OpenAPI proxying relay an upstream test server response.
- Console can set tenant/workspace scope, disable agents, revoke grants, and refresh live data.

## Demo Notes

Sprint 2 keeps `scripts/demo-governance-loop.sh` working by using the local stub response for targets without an endpoint. Cleanup behavior is covered by `scripts/demo-sprint2-cleanup.sh`. Upstream proxying is covered by deterministic `httptest` backend tests because public API endpoint registration intentionally rejects unsafe local loopback targets.
