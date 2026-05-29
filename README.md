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

## Sprint 1 Demo

With the API running, use the governance-loop demo script to exercise the end-to-end control plane and data plane:

```bash
bash scripts/demo-governance-loop.sh
```

The script defaults to `BASE_URL=http://127.0.0.1:9090`. It creates a local caller Agent, an MCP target Agent, a short-lived Agent Key, proves the MCP data-plane call is denied before a grant, creates an Access Grant, proves the call is allowed, then checks run traces for both `denied` and `allowed` decisions.

Override the target service or run id when needed:

```bash
BASE_URL=http://127.0.0.1:9091 RUN_ID=demo-manual-1 bash scripts/demo-governance-loop.sh
```

## Admin Key

Management APIs are open by default for local clean-room iteration. If `AGENT_HARBOR_ADMIN_KEY` is set on the server, management endpoints require the same value in `X-Admin-Key`.

```bash
AGENT_HARBOR_ADMIN_KEY=local-admin-key go run ./cmd/agent-harbor
ADMIN_KEY=local-admin-key bash scripts/demo-governance-loop.sh
```

The Agent Key data plane still uses `Authorization: Bearer <agent-key>`; the admin key only protects management and audit APIs. Agent Key TTLs must be between 1 and 3600 seconds, with a 1800 second server default when omitted.

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
- `GET /api/v1/agents`
- `GET /api/v1/agents/{id}`
- `POST /api/v1/agent-keys`
- `GET /api/v1/api-keys`
- `POST /api/v1/api-keys`
- `DELETE /api/v1/api-keys/{id}`
- `POST /api/v1/access-grants`
- `GET /api/v1/access-grants`
- `POST /api/v1/mcp/agents/{targetId}`
- `POST /api/v1/mcp/agents/{targetId}/rpc`
- `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`
- `ANY /api/v1/openapi/agents/{targetId}/{relativePath...}`
- `GET /api/v1/audit/traces`

## Next Milestones

- Add tenant/workspace scoping across management reads and writes.
- Add OpenAPI relative-path proxying with traversal protection.
- Add MCP `initialize`, `tools/list`, `tools/call` method-level policy.
- Add agent/access-grant revoke APIs for cleanup residue checks.
- Add OTel spans and metrics for route/caller/target dimensions.
