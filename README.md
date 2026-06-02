# AgentHarbor

Clean-room Go implementation track for the AgentHarbor Agent Gateway product model.

This module is intentionally isolated from the existing Rust/React runtime. Do not copy source code, migrations, tests, deployment scripts, adapter code, or generated assets from the existing implementation into this directory. Use only product requirements, public protocols, and public product references.

For contribution workflow, verification expectations, and clean-room guardrails, see [CONTRIBUTING.md](CONTRIBUTING.md). For private vulnerability reporting and security handling, see [SECURITY.md](SECURITY.md).

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

Use the repository toolchain pins before running local checks:

- Go version comes from `go.mod`.
- Node major version comes from `.node-version`.
- Frontend package manager comes from `frontend/package.json` (`packageManager`).

```bash
make check
make run
```

The service listens on `:9090` by default. Override with:

```bash
AGENT_HARBOR_ADDR=:9091 go run ./cmd/agent-harbor
```

For individual local checks:

```bash
make test
make test-fresh
make vet
make build
make frontend-test
make frontend-build
make demo-scripts-lint
```

PostgreSQL integration remains opt-in and uses the same environment variable as CI:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
  make test-postgres
```

## Governance Loop Demo

With the API running, use the governance-loop demo script to exercise the end-to-end control plane and data plane:

```bash
bash scripts/demo-governance-loop.sh
```

To run the full Sprint 1-11 demo suite against a running API:

```bash
make demo-all
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

## Sprint 4 Credential Demo

Sprint 4 separates upstream secrets from `channelConfig`. Against a running API, this script creates a credentialed MCP Agent and verifies create/get/list responses do not leak plaintext credentials:

```bash
bash scripts/demo-sprint4-credentials.sh
```

## Sprint 5 Retry Config Demo

Sprint 5 adds bounded proxy retry controls and richer upstream error classification. Against a running API, this script proves valid retry config is accepted and invalid retry config is rejected:

```bash
bash scripts/demo-sprint5-retry-config.sh
```

## Sprint 6 Runtime Metrics Demo

Sprint 6 backs Runtime Signals with trace-derived metrics. Against a running API, this script creates one denied and one allowed data-plane call, then verifies `GET /api/v1/metrics/runtime` reports gateway calls and allowed rate from real traces:

```bash
bash scripts/demo-sprint6-runtime-metrics.sh
```

## Sprint 7 Credential Rotation Demo

Sprint 7 adds Agent partial update and credential rotation. Against a running API, this script creates a credentialed Agent, patches mutable metadata/status, rotates credentials, and verifies responses remain redacted:

```bash
bash scripts/demo-sprint7-credential-rotation.sh
```

## Sprint 8 Management Audit Demo

Sprint 8 adds non-secret credential versions and management audit events. Against a running API, this script creates, patches, and rotates a credentialed Agent, then verifies audit events contain lifecycle actions without credential values:

```bash
bash scripts/demo-sprint8-management-audit.sh
```

## Sprint 9 Route Policy Demo

Sprint 9 promotes Route Governance into first-class route policy objects. Against a running API, this script creates allow and deny policies, verifies priority-based data-plane decisions, disables a policy, and checks trace reasons:

```bash
bash scripts/demo-sprint9-route-policies.sh
```

## Sprint 10 Route Policy Retry Demo

Sprint 10 adds per-policy retry overrides. Against a running API, this script creates an allow policy with retry, verifies normalized retry shape, confirms invalid retry values are rejected, and clears the override with PATCH:

```bash
bash scripts/demo-sprint10-route-policy-retry.sh
```

## Sprint 11 Transactional Audit Demo

Sprint 11 makes covered management audit writes transactional with the management mutation. Against a running API, this script creates and rotates a credentialed Agent, verifies audit events are visible, and checks credential values remain redacted:

```bash
bash scripts/demo-sprint11-transactional-audit.sh
```

## Sprint 12 MCP Capability Governance Demo

Sprint 12 introduces tenant-scoped MCP capability governance. It discovers MCP tools, keeps new tools denied until approval, grants an approved tool to a tenant/workspace/caller instance chain, filters `tools/list`, denies unassigned `tools/call`, and records capability evidence in traces.

Because AgentHarbor rejects loopback and private-network target endpoints by design, the demo requires a safe test MCP HTTP endpoint:

```bash
MCP_ENDPOINT=https://mcp.example.test/rpc \
ALLOWED_TOOL=search_customer \
DENIED_TOOL=export_contracts \
  bash scripts/demo-sprint12-mcp-capability-governance.sh
```

## Sprint 13 Data Permission Enforcement Demo

Sprint 13 adds hierarchical data permission enforcement for MCP capabilities. Capability, tenant entitlement, workspace assignment, and instance assignment `dataScopes` form OR alternatives. Each child scope must be equal to or narrower than a parent scope; omitted child scopes inherit the parent boundary. Governed MCP `tools/call` requests also send a reserved `X-AgentHarbor-Context` header containing tenant/workspace/caller/capability identity and the effective `dataScopes` as base64url JSON.

AgentHarbor still does not rewrite arbitrary tool arguments. Downstream MCP servers, agents, data lakes, warehouses, and databases should apply concrete predicates from the trusted context envelope.

```bash
MCP_ENDPOINT=https://mcp.example.test/rpc \
ALLOWED_TOOL=search_customer \
  bash scripts/demo-sprint13-data-permission-enforcement.sh
```

## Sprint 14 Tenant Hierarchy Demo

Sprint 14 adds explicit tenant hierarchy for the primary governance journey. Tenants can now be registered as a three-level tree, parent tenant management filters include descendants, and a target owned by a parent tenant can grant approved capabilities to a descendant tenant. Existing flat tenant IDs keep exact-match behavior until they are registered as tenants.

```bash
bash scripts/demo-sprint14-tenant-hierarchy.sh

MCP_ENDPOINT=https://mcp.example.test/rpc \
ALLOWED_TOOL=search_customer \
  bash scripts/demo-sprint14-tenant-hierarchy.sh
```

## Sprint 15 Tenant Access Profile Demo

Sprint 15 adds a read-only tenant access profile that explains the current grant chain for a tenant scope: tenant subtree, target, capability, tenant entitlement, workspace assignment, instance assignment, effective `dataScopes`, and recent trace evidence. It is an explanation API; writes still use the existing entitlement and assignment endpoints.

```bash
bash scripts/demo-sprint15-tenant-access-profile.sh

MCP_ENDPOINT=https://mcp.example.test/rpc \
ALLOWED_TOOL=search_customer \
  bash scripts/demo-sprint15-tenant-access-profile.sh
```

## Admin Key

Management APIs are open by default for local clean-room iteration. If `AGENT_HARBOR_ADMIN_KEY` is set on the server, management endpoints require the same value in `X-Admin-Key`.

```bash
AGENT_HARBOR_ADMIN_KEY=local-admin-key go run ./cmd/agent-harbor
ADMIN_KEY=local-admin-key bash scripts/demo-governance-loop.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint2-cleanup.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint3-mcp-policy.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint4-credentials.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint5-retry-config.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint6-runtime-metrics.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint7-credential-rotation.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint8-management-audit.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint9-route-policies.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint10-route-policy-retry.sh
ADMIN_KEY=local-admin-key bash scripts/demo-sprint11-transactional-audit.sh
ADMIN_KEY=local-admin-key MCP_ENDPOINT=https://mcp.example.test/rpc bash scripts/demo-sprint12-mcp-capability-governance.sh
ADMIN_KEY=local-admin-key MCP_ENDPOINT=https://mcp.example.test/rpc bash scripts/demo-sprint13-data-permission-enforcement.sh
ADMIN_KEY=local-admin-key MCP_ENDPOINT=https://mcp.example.test/rpc bash scripts/demo-sprint14-tenant-hierarchy.sh
ADMIN_KEY=local-admin-key MCP_ENDPOINT=https://mcp.example.test/rpc bash scripts/demo-sprint15-tenant-access-profile.sh
```

The Agent Key data plane still uses `Authorization: Bearer <agent-key>`; the admin key only protects management and audit APIs. Agent Key TTLs must be between 1 and 3600 seconds, with a 1800 second server default when omitted.

## Proxy Controls

MCP calls derive authorization from the JSON-RPC `method`. New control-plane flows should use `RoutePolicy.routeType=mcp` with route keys such as `initialize`, `tools/list`, or `tools/call`; an empty route key remains a wildcard. Route policies require caller and target Agents to share the same tenant and workspace. Enabled route policies are evaluated before legacy access grants: higher `priority` wins, `deny` wins ties, and disabled policies are ignored. If no matching route policy exists, the runtime falls back to existing `AccessGrant` rows for backward compatibility.

Allow route policies may include a `retry` override:

```json
{
  "retry": {
    "maxAttempts": 2,
    "backoffMs": 0,
    "statusCodes": [503]
  }
}
```

When a matching allow policy has `retry`, the proxy uses that policy-level retry instead of the target Agent `channelConfig.retry`. Policies without retry, deny policies, and legacy access grants keep the existing target-level retry behavior.

Target `channelConfig` also supports optional proxy controls:

```json
{
  "endpoint": "https://api.example.com/mcp",
  "headers": {
    "X-AgentHarbor-Tenant": "default"
  },
  "credentialHeaders": {
    "Authorization": "apiToken"
  },
  "timeoutMs": 10000,
  "retry": {
    "maxAttempts": 3,
    "backoffMs": 50,
    "statusCodes": [502, 503, 504]
  }
}
```

`headers` must be string-to-string and cannot contain secret-like names such as authorization, token, cookie, or api key. Put secret header bindings in `credentialHeaders` instead, where the object maps an upstream header name to a key in the Agent-level `credentials` object submitted at create time. Credentials are never returned by management responses; PostgreSQL stores non-empty credentials as AES-GCM ciphertext. `timeoutMs` must be an integer from 1 to 30000.

`retry` is opt-in. `maxAttempts` defaults to `1` and accepts `1` through `4`; `backoffMs` defaults to `0` and accepts `0` through `1000`; `statusCodes` defaults to `[502,503,504]` and only accepts 5xx codes. Proxied upstream responses include `X-AgentHarbor-Upstream-Attempts`. The proxy buffers request bodies up to 4MiB so retry attempts can replay the same payload; larger bodies return `413 PAYLOAD_TOO_LARGE`. Gateway-generated upstream failures use `UPSTREAM_TIMEOUT`, `UPSTREAM_DNS_ERROR`, `UPSTREAM_TLS_ERROR`, `UPSTREAM_CONNECT_ERROR`, or fallback `UPSTREAM_ERROR`.

`PATCH /api/v1/agents/{id}` updates mutable Agent metadata, status, and full `channelConfig` replacement while keeping `tenantId`, `workspaceId`, and `channelType` immutable. `POST /api/v1/agents/{id}/credentials:rotate` replaces the Agent credential bag; responses continue to omit plaintext credentials. Rotation reuses the existing credential validation and encrypted PostgreSQL persistence path. Agent responses include non-secret `credentialVersion`: `0` means no credential material has been stored, create-time credentials start at `1`, and each rotation increments the version.

Management mutations append audit events to `GET /api/v1/audit/events`. Events record action, actor, resource, scope, summary, and small metadata such as `credentialVersion`, credential key names, route policy effect, and priority. Audit metadata must not contain credential values or Agent Key plaintext. Audit listing accepts `limit`, defaults to 100 events, and caps at 500 events.

Management mutations that produce audit events commit the audit event with the business state change. If audit persistence fails, covered Agent, Agent Key, Access Grant, and Route Policy mutations fail instead of leaving unaudited state behind.

## PostgreSQL Env

Sprint 1 persistence is configured with `AGENT_HARBOR_DATABASE_URL`. When the variable is unset, the service uses the in-memory repository for local development. When it is set, the service should migrate and use PostgreSQL:

```bash
AGENT_HARBOR_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' \
AGENT_HARBOR_CREDENTIAL_KEY='0123456789abcdef0123456789abcdef' \
  go run ./cmd/agent-harbor
```

PostgreSQL integration tests use `AGENT_HARBOR_TEST_DATABASE_URL` when present. `AGENT_HARBOR_CREDENTIAL_KEY` accepts either a raw 32-byte value or a base64-encoded 32-byte value, and is required whenever `AGENT_HARBOR_DATABASE_URL` is set.

## Frontend

`frontend/` is a clean-room Vite + React + TypeScript enterprise console. It must not copy source code, styles, component structure, generated assets, or mock data from the legacy `web/` implementation.

```bash
cd frontend
pnpm install
pnpm build
pnpm dev
```

The frontend reads `VITE_API_BASE` for the Go API base URL. If it is not set, it defaults to `http://127.0.0.1:9090`. When the backend is unavailable during local development, the UI uses mock fallback data so the console remains navigable. When the backend is reachable, catalog, agent, route policy, access grant, management audit, trace, and runtime signal data come from the Go runtime; evidence runs remain local sample panels until those APIs exist.

## Current API

- `GET /healthz`
- `GET /api/v1/contracts/providers`
- `GET /api/v1/contracts/channels`
- `POST /api/v1/tenants`
- `GET /api/v1/tenants?tenantId=&parentTenantId=`
- `GET /api/v1/tenants/{id}`
- `GET /api/v1/tenants/{id}/access-profile?workspaceId=&targetId=&capabilityId=&callerInstanceId=&traceLimit=`
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
- `POST /api/v1/access-grants`
- `GET /api/v1/access-grants?tenantId=&workspaceId=`
- `DELETE /api/v1/access-grants/{id}`
- `POST /api/v1/route-policies`
- `GET /api/v1/route-policies?tenantId=&workspaceId=`
- `PATCH /api/v1/route-policies/{id}`
- `DELETE /api/v1/route-policies/{id}`
- `POST /api/v1/targets/{targetId}/capabilities:refresh`
- `GET /api/v1/capabilities?tenantId=&workspaceId=&targetId=&status=`
- `PATCH /api/v1/capabilities/{id}`
- `POST /api/v1/tenant-entitlements`
- `GET /api/v1/tenant-entitlements?tenantId=&workspaceId=&targetId=&capabilityId=`
- `POST /api/v1/workspace-assignments`
- `GET /api/v1/workspace-assignments?tenantId=&workspaceId=&entitlementId=`
- `POST /api/v1/instance-assignments`
- `GET /api/v1/instance-assignments?tenantId=&workspaceId=&callerInstanceId=&capabilityId=`
- `POST /api/v1/mcp/agents/{targetId}`
- `POST /api/v1/mcp/agents/{targetId}/rpc`
- `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`
- `ANY /api/v1/openapi/agents/{targetId}/{relativePath...}`

`instance-assignments.subjectSelector` is optional. Empty selectors match any subject for the caller instance; suffix `*` selectors such as `user:*` match subjects with that prefix from `X-AgentHarbor-Subject-Id`.

`dataScopes` are hierarchical. A child assignment may fill an empty parent dimension such as `table`, but it cannot change a fixed parent dimension such as `region` or `tenantFilter`. The runtime trace records the effective inherited scope list, and governed MCP `tools/call` forwards that same list in `X-AgentHarbor-Context`. Caller-supplied, static target, and credential-backed values for `X-AgentHarbor-Context` are reserved and not forwarded.

Registered tenants are hierarchical with a maximum depth of three. `GET` list APIs that accept `tenantId` include descendants when the tenant exists in the registry; unregistered tenant strings keep the previous exact-match behavior. Tenant entitlements may grant a target capability to the target tenant itself or to a registered descendant tenant.

The tenant access profile is a read-only explanation view over the same model. It returns configured grants and effective scope calculations for a registered tenant subtree, while keeping exact-match behavior for unregistered flat tenant strings. `traceLimit=0` disables recent trace evidence in the response.
- `GET /api/v1/audit/events?tenantId=&workspaceId=&action=&resourceType=&resourceId=&limit=`
- `GET /api/v1/audit/traces?tenantId=&workspaceId=&runId=&decision=&callerAgentId=&targetAgentId=`
- `GET /api/v1/metrics/runtime?tenantId=&workspaceId=`

## Next Milestones

- Export runtime trace dimensions to OpenTelemetry spans and metrics.
- Add external audit outbox/export semantics for management mutations.
- Add route policy import/export and dry-run simulation.
