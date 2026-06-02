# AgentHarbor

AgentHarbor is a tenant-first access control plane for AI agents, MCP servers, OpenAPI services, and governed data access.

It gives platform teams one place to register agents, discover MCP capabilities, assign tenant/workspace/caller permissions, enforce data scopes, and inspect audit evidence for every allowed or denied runtime decision.

## Project Status

AgentHarbor is in developer preview. It is ready for local evaluation, design feedback, and early contribution, but it is not yet recommended for production traffic.

## What It Provides

- **Tenant-first governance**: register a three-level tenant tree and scope management views by tenant subtree.
- **Agent registry**: manage caller agents, MCP targets, OpenAPI services, webhooks, credentials, and short-lived Agent Keys.
- **Route policy controls**: allow or deny MCP/OpenAPI routes with priority, wildcard matching, and bounded retry overrides.
- **MCP capability governance**: discover target tools, approve capabilities, and grant them through tenant, workspace, and caller-instance assignments.
- **Data permission enforcement**: narrow `dataScopes` across capability, tenant entitlement, workspace assignment, and instance assignment boundaries.
- **Runtime evidence**: record traces, audit events, metrics, upstream attempts, effective data scopes, and deny reasons.
- **Tenant Permission Console**: inspect each tenant's effective access profile, grant chain, invalid scope rows, and recent trace evidence.

## Core Model

```text
Tenant
  -> Agent or target service
  -> MCP/OpenAPI capability
  -> Tenant entitlement
  -> Workspace assignment
  -> Caller instance assignment
  -> Runtime decision and trace evidence
```

The tenant is the primary control boundary. A registered tenant can manage its own subtree; unregistered tenant strings keep exact-match behavior for compatibility.

The data plane uses short-lived Agent Keys. Management APIs can be protected with `AGENT_HARBOR_ADMIN_KEY`, which requires callers to send the same value in `X-Admin-Key`.

## Quick Start

Use the repository toolchain pins before running local checks:

- Go version comes from `go.mod`.
- Node major version comes from `.node-version`.
- Frontend package manager comes from `frontend/package.json`.

```bash
make check
make run
```

The API listens on `:9090` by default. Override it with:

```bash
AGENT_HARBOR_ADDR=:9091 go run ./cmd/agent-harbor
```

## Try the Core Journey in 10 Minutes

This local scenario runs the most important AgentHarbor workflow: create a three-level tenant tree, register a mock MCP target, discover tools, approve one tool, assign it to a tenant/workspace/caller instance, run allowed and denied calls, and verify access-profile plus audit evidence.

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make core-journey
```

The scenario starts `scripts/mock-mcp-server.py` automatically and points AgentHarbor at `http://127.0.0.1:8787/mcp`. The `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` flag is required only for local development scenarios that use loopback or private-network upstreams; do not enable it for production deployments.

## Web Console

```bash
cd frontend
pnpm install
pnpm dev
```

The console reads `VITE_API_BASE`; if unset, it uses `http://127.0.0.1:9090`. When the backend is unavailable during local development, the UI falls back to sample data so the console remains navigable.

## Runtime Configuration

Use `.env.example` as the local configuration template.

| Variable | Purpose |
| --- | --- |
| `AGENT_HARBOR_ADDR` | API listen address. Defaults to `:9090`. |
| `AGENT_HARBOR_ADMIN_KEY` | Optional management API key. When set, management and audit endpoints require `X-Admin-Key`. |
| `AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS` | Development-only boolean. Allows loopback/private upstream endpoints for local scenarios when set to `true`. Defaults to `false`. |
| `AGENT_HARBOR_DATABASE_URL` | Optional PostgreSQL connection string. If unset, AgentHarbor uses the in-memory repository. |
| `AGENT_HARBOR_CREDENTIAL_KEY` | 32-byte raw or base64 key used to encrypt persisted agent credentials. Required with PostgreSQL. |
| `AGENT_HARBOR_TEST_DATABASE_URL` | PostgreSQL connection string used by integration tests. |
| `VITE_API_BASE` | Frontend API base URL. |

PostgreSQL example:

```bash
AGENT_HARBOR_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
AGENT_HARBOR_CREDENTIAL_KEY='0123456789abcdef0123456789abcdef' \
  go run ./cmd/agent-harbor
```

## Local Verification

```bash
make test
make test-fresh
make vet
make build
make frontend-test
make frontend-build
make scenario-scripts-lint
make github-config-lint
```

PostgreSQL integration remains opt-in:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

## Scenario Scripts

The repository includes executable scenario scripts under `scripts/` for local end-to-end smoke coverage. Start the API first, then run:

```bash
make scenario-all
```

The core journey has its own script because it intentionally uses a local mock MCP endpoint:

```bash
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
make core-journey
```

With admin protection enabled:

```bash
AGENT_HARBOR_ADMIN_KEY=local-admin-key go run ./cmd/agent-harbor
ADMIN_KEY=local-admin-key make scenario-all
```

For MCP capability scenarios, provide a safe public test endpoint because AgentHarbor rejects loopback and private-network target endpoints by design:

```bash
MCP_ENDPOINT=https://mcp.example.test/rpc \
ALLOWED_TOOL=search_customer \
DENIED_TOOL=export_contracts \
ADMIN_KEY=local-admin-key \
  make scenario-all
```

## API Overview

### Health and Contracts

- `GET /healthz`
- `GET /api/v1/contracts/providers`
- `GET /api/v1/contracts/channels`

### Tenants and Access Profile

- `POST /api/v1/tenants`
- `GET /api/v1/tenants?tenantId=&parentTenantId=`
- `GET /api/v1/tenants/{id}`
- `GET /api/v1/tenants/{id}/access-profile?workspaceId=&targetId=&capabilityId=&callerInstanceId=&traceLimit=`

### Agents and Keys

- `POST /api/v1/agents`
- `GET /api/v1/agents?tenantId=&workspaceId=`
- `GET /api/v1/agents/{id}`
- `PATCH /api/v1/agents/{id}`
- `DELETE /api/v1/agents/{id}`
- `POST /api/v1/agents/{id}/credentials:rotate`
- `POST /api/v1/agent-keys`
- `GET /api/v1/api-keys?tenantId=&workspaceId=`
- `POST /api/v1/api-keys`
- `DELETE /api/v1/api-keys/{id}`

### Route Policies and Legacy Grants

- `POST /api/v1/access-grants`
- `GET /api/v1/access-grants?tenantId=&workspaceId=`
- `DELETE /api/v1/access-grants/{id}`
- `POST /api/v1/route-policies`
- `GET /api/v1/route-policies?tenantId=&workspaceId=`
- `PATCH /api/v1/route-policies/{id}`
- `DELETE /api/v1/route-policies/{id}`

### Capabilities and Assignments

- `POST /api/v1/targets/{targetId}/capabilities:refresh`
- `GET /api/v1/capabilities?tenantId=&workspaceId=&targetId=&status=`
- `PATCH /api/v1/capabilities/{id}`
- `POST /api/v1/tenant-entitlements`
- `GET /api/v1/tenant-entitlements?tenantId=&workspaceId=&targetId=&capabilityId=`
- `POST /api/v1/workspace-assignments`
- `GET /api/v1/workspace-assignments?tenantId=&workspaceId=&entitlementId=`
- `POST /api/v1/instance-assignments`
- `GET /api/v1/instance-assignments?tenantId=&workspaceId=&callerInstanceId=&capabilityId=`

### Data Plane

- `POST /api/v1/mcp/agents/{targetId}`
- `POST /api/v1/mcp/agents/{targetId}/rpc`
- `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`
- `ANY /api/v1/openapi/agents/{targetId}/{relativePath...}`

### Audit, Traces, and Metrics

- `GET /api/v1/audit/events?tenantId=&workspaceId=&action=&resourceType=&resourceId=&limit=`
- `GET /api/v1/audit/traces?tenantId=&workspaceId=&runId=&decision=&callerAgentId=&targetAgentId=`
- `GET /api/v1/metrics/runtime?tenantId=&workspaceId=`

## Policy and Data Scope Semantics

Route policies match `routeType` and optional `routeKey`; for MCP, route keys include `initialize`, `tools/list`, and `tools/call`. Higher priority wins, `deny` wins ties, disabled policies are ignored, and direct access grants remain as a compatibility fallback when no route policy matches.

MCP capabilities must be approved before they can be granted. Tenant entitlements, workspace assignments, and caller instance assignments form the effective grant chain for capability-aware MCP calls.

`dataScopes` are hierarchical OR alternatives. A child assignment may fill an empty parent dimension, but it cannot change a fixed parent dimension such as `region` or `tenantFilter`. Runtime traces record the effective inherited scope list, and governed MCP `tools/call` forwards the same list in `X-AgentHarbor-Context`. Caller-supplied, static target, and credential-backed values for `X-AgentHarbor-Context` are reserved and not forwarded.

The tenant access profile endpoint is read-only. It returns configured grants, effective scope calculations, invalid historical scope evidence, and recent trace evidence for a registered tenant subtree. `traceLimit=0` disables recent traces.

## Project Docs

- [CONTRIBUTING.md](CONTRIBUTING.md): contribution workflow and verification expectations.
- [SECURITY.md](SECURITY.md): private vulnerability reporting and security handling.
- [.env.example](.env.example): local configuration template.
- [ROADMAP.md](ROADMAP.md): public product and contribution direction.
- [docs/engineering/](docs/engineering): release, review, dependency, and engineering workflow references.
- [CHANGELOG.md](CHANGELOG.md): public release notes and notable changes.

## License

AgentHarbor is released under the [Apache License 2.0](LICENSE).
