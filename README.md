# AgentHarbor

Clean-room Go implementation track for the AgentHarbor Agent Gateway product model.

This module is intentionally isolated from the existing Rust/React runtime. Do not copy source code, migrations, tests, deployment scripts, adapter code, or generated assets from the existing implementation into this directory. Use only product requirements, public protocols, and public product references.

## Scope

Sprint 0 proves the smallest governed data-plane loop:

```text
Caller Agent
  -> short-lived Agent Key
  -> access grant
  -> MCP/OpenAPI target route
  -> allowed or denied decision
  -> trace evidence
```

Management APIs stay open by default for local tests and clean-room iteration. Set `AGENT_HARBOR_ADMIN_KEY` before exposing the process outside a developer machine so management and audit endpoints require `X-Admin-Key`; the data plane continues to use short-lived Agent Keys.

## Run

```bash
go test ./...
go build ./...
go run ./cmd/agent-harbor
```

The service listens on `:9090` by default. Override with:

```bash
AGENT_HARBOR_ADDR=:9091 go run ./cmd/agent-harbor
```

## Governance Loop Demo

With the API running, use the governance-loop demo script to exercise the end-to-end control plane and data plane:

```bash
bash scripts/demo-governance-loop.sh
```

The script defaults to `BASE_URL=http://127.0.0.1:9090`. It creates a local caller Agent, a governed local target, a short-lived Agent Key, proves the MCP data-plane call is denied before a grant, creates an Access Grant, proves the call is allowed, then checks run traces for both `denied` and `allowed` decisions. Targets without `channelConfig.endpoint` keep the local accepted stub response; MCP/OpenAPI targets with an endpoint are forwarded upstream after authorization.

Override the target service or run id when needed:

```bash
BASE_URL=http://127.0.0.1:9091 RUN_ID=demo-manual-1 bash scripts/demo-governance-loop.sh
```

## Sprint 2 Cleanup Demo

Sprint 2 adds scoped management reads plus cleanup controls. Against a running API, this script proves grant revoke and agent disable semantics:

```bash
bash scripts/demo-sprint2-cleanup.sh
```

It creates a caller/target pair in one workspace, confirms an allowed call, revokes the grant and sees the next call denied, recreates a grant, disables the caller Agent, sees its old Agent Key rejected, then verifies `tenantId`/`workspaceId` scoped agent listing.

## Sprint 3 MCP Policy Demo

Sprint 3 makes MCP authorization method-aware. Against a running API, this script grants only `tools/list`, proves a `tools/list` JSON-RPC call is allowed, proves `tools/call` is denied, and checks traces for the actual MCP methods:

```bash
bash scripts/demo-sprint3-mcp-policy.sh
```

## Admin Key

Management APIs are open by default for local clean-room iteration. If `AGENT_HARBOR_ADMIN_KEY` is set on the server, management endpoints require the same value in `X-Admin-Key`.

```bash
AGENT_HARBOR_ADMIN_KEY=local-admin-key go run ./cmd/agent-harbor
ADMIN_KEY=local-admin-key bash scripts/demo-governance-loop.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint2-cleanup.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint3-mcp-policy.sh
```

The Agent Key data plane still uses `Authorization: Bearer <agent-key>`; the admin key only protects management and audit APIs. Agent Key TTLs must be between 1 and 3600 seconds, with a 1800 second server default when omitted.

## Proxy Controls

MCP calls derive authorization from the JSON-RPC `method`. Use `AccessGrant.routeType=mcp` with route keys such as `initialize`, `tools/list`, or `tools/call`; an empty route key remains a wildcard.

Target `channelConfig` also supports optional proxy controls:

```json
{
  "endpoint": "https://api.example.com/mcp",
  "headers": {
    "X-AgentHarbor-Tenant": "default"
  },
  "timeoutMs": 10000
}
```

`headers` must be string-to-string and cannot contain secret-like names such as authorization, token, cookie, or api key. `timeoutMs` must be an integer from 1 to 30000. Upstream timeout returns `504 UPSTREAM_TIMEOUT`; other network failures return `502 UPSTREAM_ERROR`.

## PostgreSQL Env

Sprint 1 persistence is configured with `AGENT_HARBOR_DATABASE_URL`. When the variable is unset, the service uses the in-memory repository for local development. When it is set, the service should migrate and use PostgreSQL:

```bash
AGENT_HARBOR_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  go run ./cmd/agent-harbor
```

PostgreSQL integration tests use `AGENT_HARBOR_TEST_DATABASE_URL` when present.

## Frontend

`frontend/` is a clean-room Vite + React + TypeScript enterprise console. It must not copy source code, styles, component structure, generated assets, or mock data from the legacy `web/` implementation.

```bash
cd frontend
pnpm install
pnpm build
pnpm dev
```

The frontend reads `VITE_API_BASE` for the Go API base URL. If it is not set, it defaults to `http://127.0.0.1:9090`. When the backend is unavailable during local development, the UI uses mock fallback data so the console remains navigable. When the backend is reachable, catalog, agent, access grant, and trace data come from the Go runtime; evidence runs and runtime signals remain local sample panels until those APIs exist.

## Current API

- `GET /healthz`
- `GET /api/v1/contracts/providers`
- `GET /api/v1/contracts/channels`
- `POST /api/v1/agents`
- `GET /api/v1/agents?tenantId=&workspaceId=`
- `GET /api/v1/agents/{id}`
- `DELETE /api/v1/agents/{id}`
- `POST /api/v1/agent-keys`
- `GET /api/v1/api-keys?tenantId=&workspaceId=`
- `POST /api/v1/api-keys`
- `DELETE /api/v1/api-keys/{id}`
- `POST /api/v1/access-grants`
- `GET /api/v1/access-grants?tenantId=&workspaceId=`
- `DELETE /api/v1/access-grants/{id}`
- `POST /api/v1/mcp/agents/{targetId}`
- `POST /api/v1/mcp/agents/{targetId}/rpc`
- `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`
- `ANY /api/v1/openapi/agents/{targetId}/{relativePath...}`
- `GET /api/v1/audit/traces?tenantId=&workspaceId=&runId=&decision=&callerAgentId=&targetAgentId=`

## Next Milestones

- Add encrypted per-agent upstream credentials and secret header injection.
- Add proxy retry policy and richer upstream error classification.
- Add OTel spans and metrics for route/caller/target dimensions.
