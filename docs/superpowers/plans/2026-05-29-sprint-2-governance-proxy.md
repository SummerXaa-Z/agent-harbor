# Sprint 2 Governance Proxy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add scoped management reads, cleanup/revoke APIs, and first real upstream proxying for allowed MCP/OpenAPI traffic.

**Architecture:** Keep the Sprint 1 repository boundary. Replace simple list arguments with filter structs, add soft-disable/revoke repository methods, and keep HTTP handlers thin. Proxying lives inside `internal/httpapi` behind small helpers so later policy/protocol work can replace the forwarding internals without changing routes.

**Tech Stack:** Go 1.25, `chi`, `pgx/v5/pgxpool`, React 19, Vite, TypeScript, browser `fetch`, Go `net/http` for upstream forwarding.

---

## File Structure

- `internal/store/memory.go`: add filter structs, stable scoped list behavior, disable/revoke methods.
- `internal/store/postgres.go`: match repository semantics for scope filters and soft cleanup.
- `internal/httpapi/server.go`: parse management scope, add cleanup routes, enforce disabled caller auth, add upstream proxy helpers.
- `internal/httpapi/server_test.go`: TDD coverage for scope, cleanup, disabled auth, revoke, MCP proxy, OpenAPI proxy.
- `internal/store/postgres_test.go`: PostgreSQL round-trip coverage for new repository methods.
- `frontend/src/types.ts`: add management scope, revoke/disable response types if needed.
- `frontend/src/api.ts`: pass scope query params, add disable/revoke API calls.
- `frontend/src/App.tsx`: add scope strip and cleanup actions.
- `frontend/src/styles.css`: style compact scope/actions.
- `scripts/demo-governance-loop.sh` or `scripts/demo-sprint2-cleanup-proxy.sh`: prove revoke/proxy behavior.
- `README.md`, `CHANGELOG.md`, `docs/sprints/sprint-2-brief.md`: document usage and decisions.

## Task 1: Scoped Repository and Cleanup APIs

- [x] Write failing HTTP tests in `internal/httpapi/server_test.go`:
  - `TestManagementScopeFiltersLists`
  - `TestDisableAgentBlocksExistingKey`
  - `TestRevokeAccessGrantDeniesLaterCalls`
- [x] Run `go test ./internal/httpapi -run 'TestManagementScope|TestDisableAgent|TestRevokeAccessGrant' -count=1` and confirm failures.
- [x] Extend `store.Repository` with:
  - `ListAgents(ctx, store.AgentFilter)`
  - `ListAgentKeys(ctx, store.ManagementScope)`
  - `ListAccessGrants(ctx, store.ManagementScope)`
  - `ListTraces(ctx, store.TraceFilter)` including scope fields
  - `DisableAgent(ctx, id, now)`
  - `RevokeAccessGrant(ctx, id, now)`
- [x] Implement memory store changes and stable sorting.
- [x] Implement PostgreSQL store changes with scoped joins.
- [x] Add `DELETE /api/v1/agents/{id}` and `DELETE /api/v1/access-grants/{id}`.
- [x] Update bearer auth so disabled callers are rejected.
- [x] Run targeted tests and then `go test ./...`.

## Task 2: Upstream Proxying

- [x] Write failing tests in `internal/httpapi/server_test.go`:
  - `TestMCPProxyRelaysAllowedUpstreamResponse`
  - `TestOpenAPIProxyRelaysRelativePath`
  - `TestProxyUpstreamFailureRecordsTraceAndReturnsBadGateway`
- [x] Run targeted tests and confirm failures.
- [x] Add proxy helpers in `server.go`:
  - extract target Agent after authorization
  - read endpoint from `channelConfig.endpoint`
  - keep stub response when endpoint is missing
  - forward request body/method to MCP endpoint or OpenAPI endpoint+relative path
  - relay upstream status/content type/body
  - return `502 UPSTREAM_ERROR` on network failure
- [x] Preserve traversal rejection for OpenAPI relative paths.
- [x] Run targeted proxy tests and then `go test ./...`.

## Task 3: Frontend Scope and Cleanup Controls

- [x] Update `frontend/src/api.ts` to send `tenantId` and `workspaceId` query params to list APIs.
- [x] Add `disableAgent(id, adminKey)` and `revokeAccessGrant(id, adminKey)`.
- [x] Add scope state to `App.tsx`; default `tenantId=default`, `workspaceId=workspace-demo`.
- [x] Use scope defaults in create-agent.
- [x] Add Disable buttons to Agent Registry and Revoke buttons to Route Governance.
- [x] Show revoked grants as disabled/deny.
- [x] Run `pnpm -C frontend build`.
- [x] Run Playwright smoke against live API with admin key.

## Task 4: Demo, Docs, Verification

- [x] Add or extend demo script to cover allowed proxy response and revoked grant denial.
- [x] Update README API list and Sprint 2 docs.
- [x] Update CHANGELOG with decisions and lessons.
- [x] Run:
  - `go test -count=1 ./...`
  - `go vet ./...`
  - `go build ./...`
  - `pnpm -C frontend build`
  - memory E2E demo
  - PostgreSQL integration test with temporary container
  - Playwright live/fallback smoke
- [ ] Request final code review and fix blockers before commit.
