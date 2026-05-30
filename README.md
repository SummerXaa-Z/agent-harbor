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

`PATCH /api/v1/agents/{id}` updates mutable Agent metadata, status, and full `channelConfig` replacement while keeping `tenantId`, `workspaceId`, and `channelType` immutable. `POST /api/v1/agents/{id}/credentials:rotate` replaces the Agent credential bag; responses continue to omit plaintext credentials. Rotation reuses the existing credential validation and encrypted PostgreSQL persistence path.

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

The frontend reads `VITE_API_BASE` for the Go API base URL. If it is not set, it defaults to `http://127.0.0.1:9090`. When the backend is unavailable during local development, the UI uses mock fallback data so the console remains navigable. When the backend is reachable, catalog, agent, access grant, trace, and runtime signal data come from the Go runtime; evidence runs remain local sample panels until those APIs exist.

## Current API

- `GET /healthz`
- `GET /api/v1/contracts/providers`
- `GET /api/v1/contracts/channels`
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
- `POST /api/v1/mcp/agents/{targetId}`
- `POST /api/v1/mcp/agents/{targetId}/rpc`
- `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`
- `ANY /api/v1/openapi/agents/{targetId}/{relativePath...}`
- `GET /api/v1/audit/traces?tenantId=&workspaceId=&runId=&decision=&callerAgentId=&targetAgentId=`
- `GET /api/v1/metrics/runtime?tenantId=&workspaceId=`

## Next Milestones

- Export runtime trace dimensions to OpenTelemetry spans and metrics.
- Add credential version metadata and audit events for rotation/update.
- Add route-level retry overrides after route policy objects exist.
