# AgentHarbor Sprint 1 Brief

Date: 2026-05-29
Status: Implemented and locally verified

## Goal

Turn AgentHarbor from a Sprint 0 skeleton into a small but credible governance-loop product demo: an admin can register agents, issue a caller key, create a caller-to-target grant, exercise an allowed/denied data-plane call, and inspect trace evidence after process restart.

## Non-goals

- No full multi-tenant RBAC or user management.
- No real upstream MCP/OpenAPI proxying yet.
- No provider credential vaulting yet.
- No production deployment packaging.
- No broad visual redesign beyond forms, empty states, and workflow panels needed for Sprint 1.

## User Stories

- As a platform admin, I can protect management APIs with a local admin key so the demo is not openly writable.
- As a platform admin, I can create caller and target agents from the console or API.
- As a platform admin, I can create a short-lived Agent Key for a local caller and copy it once.
- As a platform admin, I can create an Access Grant between a caller and target, optionally scoped by route type/key.
- As an auditor, I can see allowed and denied trace evidence from real data-plane calls.
- As a developer, I can restart the service and keep agents, keys, grants, and traces through PostgreSQL.

## API/Data Model Impact

- Add PostgreSQL-backed repository for existing domain entities.
- Add migrations for `agents`, `agent_keys`, `access_grants`, and `trace_events`.
- Keep existing public endpoint paths stable.
- Add admin-key middleware for management endpoints using `AGENT_HARBOR_ADMIN_KEY`.
- Keep data-plane Bearer Agent Key auth unchanged.
- Add optional trace filters: `decision`, `callerAgentId`, `targetAgentId`, and `runId`.
- Add a read endpoint for access grants so the frontend can render real route policies.

## Frontend Impact

- Replace core mock panels with API-backed agents, grants, keys, and traces.
- Add create-agent, create-key, and create-grant workflows in the existing enterprise console shell.
- Add a visible admin key connection state and a local-only input for `X-Admin-Key`.
- Preserve mock fallback only for design/demo mode when the API is unreachable.

## Acceptance Criteria

- `go test ./...`, `go vet ./...`, and `go build ./...` pass.
- `pnpm -C frontend build` passes.
- With PostgreSQL configured, service restart keeps agents, grants, keys, and traces.
- Management endpoints reject writes without `X-Admin-Key` when `AGENT_HARBOR_ADMIN_KEY` is set.
- Console can create an agent, create a key, create a grant, and refresh real audit traces.
- A scripted E2E flow shows denied trace before grant and allowed trace after grant.

## Demo Script

Run the Sprint 1 governance loop against a running local API:

```bash
bash scripts/demo-governance-loop.sh
```

Defaults:

- `BASE_URL=http://127.0.0.1:9090`
- `RUN_ID=demo-<timestamp>`
- `WORKSPACE_ID=demo-workspace`
- `ADMIN_KEY` unset

If the server is started with `AGENT_HARBOR_ADMIN_KEY`, pass the same value as `ADMIN_KEY` so the script sends `X-Admin-Key` on management and audit requests:

```bash
AGENT_HARBOR_ADMIN_KEY=local-admin-key go run ./cmd/agent-harbor
ADMIN_KEY=local-admin-key bash scripts/demo-governance-loop.sh
```

Expected result: the first MCP data-plane call returns `403` and records a `denied` trace, the grant creation succeeds, the second MCP call returns `200` and records an `allowed` trace, and `GET /api/v1/audit/traces?runId=<RUN_ID>` contains both decisions.
