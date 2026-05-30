# AgentHarbor Sprint 2 Governance Proxy Design

Date: 2026-05-29
Status: Approved for implementation

## Goal

Move AgentHarbor from a governed demo loop toward a usable gateway by adding scoped management reads, cleanup/revoke APIs, and a first real upstream proxy path while keeping the Sprint 1 local developer flow intact.

## Scope

Sprint 2 implements three product slices:

- Management scope: management APIs can be narrowed by `tenantId` and `workspaceId` query parameters, and create/list flows preserve those fields consistently.
- Cleanup controls: admins can disable an Agent and revoke an Access Grant so demo residue can be cleaned up without direct database edits.
- Upstream proxy: allowed MCP and OpenAPI data-plane calls forward to the target Agent `channelConfig.endpoint` when present, while the existing stub response remains available for agents without an endpoint.

## Non-Goals

- No full RBAC, user sessions, or workspace membership model.
- No credential vaulting or provider secrets in `channelConfig`.
- No streaming/SSE proxying yet.
- No complete MCP protocol implementation beyond forwarding the caller request.
- No production-grade retry/circuit breaker layer.

## API Behavior

Management endpoints stay under the existing `X-Admin-Key` protection when `AGENT_HARBOR_ADMIN_KEY` is set.

Query scoping:

- `GET /api/v1/agents?tenantId=<id>&workspaceId=<id>` filters by both fields when supplied.
- `GET /api/v1/api-keys?tenantId=<id>&workspaceId=<id>` lists keys whose Agent belongs to the scope.
- `GET /api/v1/access-grants?tenantId=<id>&workspaceId=<id>` lists grants whose caller or target belongs to the scope.
- `GET /api/v1/audit/traces?tenantId=<id>&workspaceId=<id>` lists traces whose caller or target belongs to the scope.

Cleanup:

- `DELETE /api/v1/agents/{id}` soft-disables the Agent by setting `status=disabled` and `updatedAt=now`.
- Disabled callers cannot authenticate through existing Agent Keys.
- `DELETE /api/v1/access-grants/{id}` sets `revokedAt=now`; revoked grants no longer authorize traffic.

Proxying:

- Denied data-plane calls still record a denied trace and return `403`.
- Allowed calls record an allowed trace before proxying.
- If the target Agent has no string `channelConfig.endpoint`, response remains the Sprint 1 accepted stub.
- MCP calls `POST` the original request body to the endpoint.
- OpenAPI relative-path calls join the endpoint and relative path safely, preserve method/body, and reject traversal as today.
- Proxy responses are relayed with upstream status, `Content-Type`, and JSON/body bytes.
- Upstream network failures return `502` with a clear `UPSTREAM_ERROR` response.

## Frontend Behavior

The console adds a compact scope strip for `tenantId` and `workspaceId`. Lists and creates use the selected scope; create-agent defaults to the current scope. Route Governance shows revoked grants as disabled/deny. Agent Registry exposes a disable action, and Route Governance exposes revoke for live grants.

The UI does not store admin keys or agent keys beyond current component state.

## Data Model

No new migration is required. Sprint 1 schema already contains `tenant_id`, `workspace_id`, `status`, `updated_at`, and `revoked_at`.

Repository changes are interface-level:

- Replace string-only list filters with small filter structs.
- Add `DisableAgent(ctx, id, now)` and `RevokeAccessGrant(ctx, id, now)`.
- Make `FindAgentByKeyHash` require `a.status='active'`.

## Testing

Backend tests must cover:

- Scope filters for agents, keys, grants, and traces.
- Disable Agent blocks later bearer authentication.
- Revoke Access Grant changes an allowed route to denied.
- MCP proxy relays upstream JSON after authorization.
- OpenAPI relative path proxy preserves path/method and blocks traversal.

Frontend tests are build + Playwright smoke:

- Missing/wrong admin key still shows API error, not fallback.
- Valid admin key loads live data.
- Scope values affect API calls.
- Disable/revoke actions refresh live tables.

