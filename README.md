# AI Nexus Go Rebirth

Clean-room Go implementation track for the AI Nexus Agent Gateway product model.

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

Management APIs in this skeleton are intentionally unauthenticated so local tests and clean-room iteration stay frictionless. Do not expose this process outside a developer machine until admin auth, tenant scoping, and persistence are added.

## Run

```bash
cd go-nexus
go test ./...
go build ./...
go run ./cmd/nexus-go
```

The service listens on `:9090` by default. Override with:

```bash
NEXUS_ADDR=:9091 go run ./cmd/nexus-go
```

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
- `POST /api/v1/mcp/agents/{targetId}`
- `POST /api/v1/mcp/agents/{targetId}/rpc`
- `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`
- `ANY /api/v1/openapi/agents/{targetId}/{relativePath...}`
- `GET /api/v1/audit/traces`

## Next Milestones

- Add PostgreSQL persistence via `pgx/v5`, `sqlc`, and `goose`.
- Add OpenAPI relative-path proxying with traversal protection.
- Add MCP `initialize`, `tools/list`, `tools/call` method-level policy.
- Add API-key revoke/list for cleanup residue checks.
- Add OTel spans and metrics for route/caller/target dimensions.
