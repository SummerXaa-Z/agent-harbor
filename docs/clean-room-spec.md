# AI Nexus Go Rebirth Clean-room Spec

Date: 2026-05-29
Branch: `codex/ai-nexus-go-rebirth`
Status: Sprint 0 implementation spec

## Clean-room Boundary

This rebirth track is a fresh Go implementation of the AI Nexus Agent Gateway problem space. It must not copy Rust, TypeScript, migration, test fixture, deployment, or adapter source from the existing implementation. Existing AI Nexus documents may be used only as product requirements. Public protocols and public product references are allowed: MCP, OpenAPI, OIDC, A2A concepts, and Higress public AI Gateway/MCP Gateway materials.

The first code milestone lives in a new isolated module and does not replace the current production runtime.

## Technical Choice

Recommended stack for Sprint 0:

- Go HTTP service with `chi`
- PostgreSQL-first domain model, initially backed by an in-memory repository so tests and local demos do not need infrastructure
- Future data layer: `pgx/v5`, `sqlc`, `goose`
- Redis and OpenTelemetry reserved for later milestones
- JSON-only REST API, no frontend in Sprint 0

Why Go: the gateway/data-plane path benefits from simple concurrency, small binaries, readable handlers, and easy operational ownership. It also makes the clean-room boundary obvious because all runtime code is new.

## Sprint 0 Scope

Build a runnable Agent Gateway skeleton that proves the product model:

- Caller Agent and Target Agent registry
- Provider/channel contract catalog
- Short-lived Agent Key issuing with one-time plaintext return and stored hash
- Caller-to-target access grant
- Protected MCP and OpenAPI data-plane simulation
- Audit trace for allowed and denied calls
- Health endpoint and tests

## Entity Model

| Entity | Purpose |
|---|---|
| Agent | A managed caller or target. Fields include id, tenantId, workspaceId, name, channelType, channelConfig, status, ownerId, createdAt, updatedAt |
| ProviderContract | Server-owned provider preset and schema metadata |
| ChannelContract | Server-owned channel schema and endpoint policy |
| AgentKey | A hashed caller credential. Plaintext is returned only at creation |
| AccessGrant | Caller-to-target permission with optional route/tool/operation scope |
| TraceEvent | Append-only evidence for allowed/denied data-plane attempts |

## API Surface

| Method | Path | Behavior |
|---|---|---|
| GET | `/healthz` | Liveness check |
| GET | `/api/v1/contracts/providers` | Provider contract catalog |
| GET | `/api/v1/contracts/channels` | Channel contract catalog |
| POST | `/api/v1/agents` | Create caller/target Agent with basic schema and endpoint validation |
| GET | `/api/v1/agents` | List Agents by optional workspaceId |
| GET | `/api/v1/agents/{id}` | Read one Agent |
| POST | `/api/v1/agent-keys` | Create Agent Key for a caller Agent |
| GET | `/api/v1/api-keys` | List Agent Keys for cleanup residue checks |
| POST | `/api/v1/api-keys` | Compatibility alias for creating Agent Keys |
| DELETE | `/api/v1/api-keys/{id}` | Revoke Agent Key |
| POST | `/api/v1/access-grants` | Grant caller -> target |
| POST | `/api/v1/mcp/agents/{targetId}` | Simulate protected MCP JSON-RPC call, record trace |
| POST | `/api/v1/mcp/agents/{targetId}/rpc` | Compatibility alias for protected MCP call |
| POST | `/api/v1/openapi/agents/{targetId}/operations/{operationId}` | Simulate protected OpenAPI call, record trace |
| ANY | `/api/v1/openapi/agents/{targetId}/{relativePath...}` | Simulate protected relative-path OpenAPI call |
| GET | `/api/v1/audit/traces` | List trace evidence |

## Security Rules

- Data-plane requires `Authorization: Bearer <agent-key>`.
- A valid key identifies caller Agent. A caller-to-target AccessGrant is required before allowed data-plane response.
- Sprint 0 only issues Agent Keys for `channelType=local` caller Agents.
- Denied calls must return `403` with stable `PERMISSION_DENIED` code and must record a trace.
- Endpoint values in `channelConfig.endpoint` reject loopback, RFC1918, link-local, and metadata hostnames.
- Active `mcp`, `openapi`, and `a2a` target Agents require `channelConfig.endpoint`.
- Secret-like keys must not appear in `channelConfig`; use credentials in later milestones.
- Agent Key plaintext is never returned after creation and is never stored un-hashed.
- Management APIs are unauthenticated in Sprint 0 and must not be exposed beyond local development.

## Non-goals

- No production database persistence in Sprint 0.
- No frontend rewrite yet.
- No migration from existing production data.
- No real upstream MCP/OpenAPI proxying yet; Sprint 0 simulates data-plane authorization and evidence shape.
- No direct reuse of existing source code, schemas, migrations, tests, or deployment files.

## Acceptance

- `go test ./...` passes in the new module.
- `go build ./...` passes in the new module.
- Tests cover health, contract catalogs, create/list/read Agent, key creation, allowed data-plane trace, denied data-plane trace, endpoint validation, and secret-like channel config rejection.
- The module can run with `go run ./cmd/nexus-go`.
