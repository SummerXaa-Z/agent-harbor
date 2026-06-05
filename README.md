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
- **AI-friendly permission operations**: draft tenant-scoped permission package changes from administrator intent, preview allow/deny outcomes, apply them through the existing grant chain, and review structured application records.
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

Use the repository toolchain pins before running local commands:

- Go version comes from `go.mod`.
- Node major version comes from `.node-version`.
- Frontend package manager comes from `frontend/package.json`.

```bash
make demo
```

Then open `http://127.0.0.1:5174/`. The demo command starts the API, the dependency-free mock MCP server, and the web console together for the first browser evaluation.

In the Cockpit's **Core Journey Workbench**, confirm the preflight rows show the API service and Mock MCP service as ready, then run the core journey. A successful run reaches `6/6` and leaves allowed/denied runtime evidence plus the tenant access profile visible in the console.

Use the local release gate when you want to verify the repository rather than run the browser demo:

```bash
make check
```

`make check` installs the pinned frontend dependencies from `frontend/pnpm-lock.yaml` before running frontend tests and builds.

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

For the first browser evaluation, run:

```bash
make demo
```

Then open `http://127.0.0.1:5174/` and use the Cockpit's **Core Journey Workbench**. The workbench checks API and Mock MCP readiness before enabling the run button. It also includes a **Reset demo session** action that clears the current browser session state and filters without deleting backend data; each run uses fresh `ui-core-*` identifiers so historical evidence remains inspectable.

`make demo` starts:

- AgentHarbor API at `http://127.0.0.1:9090`
- Mock MCP server at `http://127.0.0.1:8787/mcp`
- Web console at `http://127.0.0.1:5174`

Use `Ctrl+C` in the demo terminal to stop all demo services.

If you need to troubleshoot a single service, use the manual three-terminal path:

Terminal 1:

```bash
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true make run
```

Terminal 2:

```bash
make mock-mcp
```

Terminal 3:

```bash
cd frontend
pnpm install
pnpm dev
```

The console reads `VITE_API_BASE`; if unset, it uses `http://127.0.0.1:9090`. When the backend is unavailable during local development, the UI falls back to sample data so the console remains navigable.

The Core Journey Workbench creates a fresh tenant tree, caller, MCP target, scoped capability grant chain, allowed call, denied call, and tenant access profile evidence through the real API. The core console journey supports English and Simplified Chinese. The browser language is used on first load, and the visible `中文` / `EN` toggle persists the operator's choice locally.

The console also includes an **AI Admin** workspace for the v0.2.0 permission-package journey. It lets an administrator describe an access request, select a deterministic permission package template, preview allow/deny simulation rows, apply the package through the backend permission-package API, and then inspect the refreshed tenant access profile. Each successful package application records the template version, draft id, created entitlement and assignment ids, capability ids, and data scopes for later review. See [the v0.2.0 journey note](docs/product/0.2.0-ai-admin-permission-journey.md).

AgentHarbor also exposes the same workflow as a management MCP endpoint at `POST /api/v1/management/mcp`. Admin agents can call `tools/list` and then use tools such as `draft_permission_package`, `apply_permission_package`, `list_permission_package_applications`, `explain_permission_package_draft`, `explain_access_decision`, `get_tenant_access_profile`, `list_agents`, and `list_capabilities`. When `AGENT_HARBOR_ADMIN_KEY` is configured, this endpoint requires `X-Admin-Key` like the rest of the management API.

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
make frontend-deps
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
- `GET /api/v1/permission-packages/templates`
- `POST /api/v1/permission-packages/drafts`
- `POST /api/v1/permission-packages:apply`
- `GET /api/v1/permission-packages/applications?tenantId=&workspaceId=&templateId=&targetId=&callerInstanceId=&limit=`
- `POST /api/v1/management/mcp`
- `POST /api/v1/management/mcp/rpc`
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
