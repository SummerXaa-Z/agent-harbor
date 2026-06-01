# Tenant Access Profile Design

Date: 2026-06-02
Status: Approved by user direction to follow the recommended read-only approach

## Goal

Add a read-only access profile API that explains a tenant's current effective permission picture across MCP targets, discovered capabilities, tenant entitlements, workspace assignments, instance assignments, data scopes, and recent runtime traces.

The product question this API answers is:

```text
For this tenant or tenant subtree, what MCP/agent/tool access exists, where did it come from, what data scope applies, and which runtime calls recently proved or denied it?
```

This is the next layer after tenant hierarchy and data permission enforcement. It gives operators and future UI screens one stable explanation surface instead of forcing them to manually join many list endpoints.

## Non-Goals

- No new write APIs.
- No automatic permission repair.
- No tenant entitlement inheritance.
- No user/admin role model.
- No frontend UI in this step.
- No speculative simulation matrix for arbitrary callers, subjects, and tools. Runtime evaluation already exists; this API explains configured and observed access.

## API

```http
GET /api/v1/tenants/{id}/access-profile
```

Admin authentication follows existing management API rules.

Query parameters:

- `workspaceId`: optional exact workspace filter.
- `targetId`: optional target agent filter.
- `capabilityId`: optional capability filter.
- `callerInstanceId`: optional caller instance filter.
- `traceLimit`: optional recent trace count, default `20`, max `100`; `0` disables traces.

The path tenant ID defines the management scope. If the tenant exists in the registry, the profile covers that tenant and its descendants. If the tenant is not registered, the profile keeps the existing flat exact-match behavior.

## Response Shape

```json
{
  "tenant": {},
  "scopeTenants": [],
  "summary": {},
  "grants": [],
  "recentTraces": [],
  "generatedAt": "2026-06-02T00:00:00Z"
}
```

### Tenant and Scope

- `tenant` is the registered tenant object when it exists. For an unregistered flat tenant, the API returns a synthetic object with the requested ID, level `0`, name equal to the ID, and status `active`.
- `scopeTenants` contains the registered subtree. For a flat tenant it contains the synthetic tenant only.

### Summary

The summary is intentionally small and stable:

```json
{
  "tenantCount": 3,
  "grantCount": 5,
  "targetCount": 2,
  "capabilityCount": 5,
  "workspaceAssignmentCount": 4,
  "instanceAssignmentCount": 6,
  "recentAllowedTraceCount": 8,
  "recentDeniedTraceCount": 2
}
```

Counts are computed after the query filters are applied.

### Grants

Each grant is centered on one tenant entitlement and expands downward:

```json
{
  "tenantEntitlement": {},
  "target": {},
  "capability": {},
  "effectiveTenantDataScopes": [],
  "scopeStatus": "valid",
  "scopeReason": "",
  "workspaceAssignments": [
    {
      "workspaceAssignment": {},
      "effectiveWorkspaceDataScopes": [],
      "scopeStatus": "valid",
      "scopeReason": "",
      "instanceAssignments": [
        {
          "instanceAssignment": {},
          "callerInstance": {},
          "effectiveInstanceDataScopes": [],
          "scopeStatus": "valid",
          "scopeReason": ""
        }
      ]
    }
  ]
}
```

`scopeStatus` is `valid` or `invalid`. If historical or manually inserted data violates narrowing rules, the profile returns the row with `scopeStatus: "invalid"` instead of hiding it. This keeps the endpoint useful for audit and repair.

Denied rows are included. The profile explains configured policy, not only allowed runtime paths.

## Data Flow

The handler builds the profile using existing repository methods:

1. Load the requested tenant with `GetTenant`.
2. Load subtree tenants with `ListTenants(TenantFilter{TenantID: id})` when registered.
3. Load entitlements with `ListTenantEntitlements` scoped by the path tenant and optional `targetId` / `capabilityId`.
4. Load workspace assignments with `ListWorkspaceAssignments`, optionally scoped by `workspaceId`.
5. Load instance assignments with `ListInstanceAssignments`, optionally scoped by `callerInstanceId`.
6. Load targets and capabilities by ID for referenced entitlements.
7. Compute effective data scopes in memory using `domain.EffectiveDataScopes`.
8. Load traces with `ListTraces` for the same tenant/workspace scope, then filter/slice by optional target/caller/capability and `traceLimit`.

Existing trace list methods return rows in ascending time order. The profile should take the newest `traceLimit` rows after filtering and return them newest-first in `recentTraces`.

No new database table is needed. The first implementation can do in-process joins because the endpoint is a management explanation view, not a high-QPS runtime path.

## Code Organization

Keep `server.go` from growing further by adding:

- `internal/httpapi/access_profile.go`: handler, response structs, profile builder helpers.
- `internal/httpapi/access_profile_test.go`: HTTP-level profile tests.

The route should be registered in the admin group as `GET /tenants/{id}/access-profile` next to the existing tenant routes.

Domain permission primitives stay where they are. The response structs can remain in `httpapi` unless another package needs them later.

## Error Handling

- Missing registered tenant is not an error. It returns a flat exact-match profile for backward compatibility.
- Invalid `traceLimit` returns `400 VALIDATION_FAILED`.
- Invalid referenced target/capability records are represented in the profile with a missing object and `scopeStatus: "invalid"` rather than crashing the whole response.
- Repository errors return through existing `writeError` behavior.

## Testing

Tests should cover:

- Registered root profile includes child and grandchild tenant entitlements.
- Leaf tenant profile includes only that leaf tenant unless it has descendants.
- Parent-owned target granted to child tenant appears under the child profile.
- Effective data scopes are shown at tenant, workspace, and instance levels.
- Invalid historical data scopes are reported as `scopeStatus: "invalid"`.
- Filters for `workspaceId`, `targetId`, `capabilityId`, and `callerInstanceId` narrow the response.
- `traceLimit=0` disables traces; default trace limit includes recent traces and summary counts.
- Unregistered flat tenant keeps exact-match profile behavior.

## Demo

Add a Sprint 15 demo that extends the tenant hierarchy path:

1. Create root, child, and grandchild tenants.
2. Register an MCP target under the root tenant.
3. Discover and approve a capability.
4. Grant the capability to the child tenant.
5. Add workspace and instance assignments with narrower data scopes.
6. Call the MCP tool with an agent key.
7. Fetch `/api/v1/tenants/{child}/access-profile` and verify the grant chain, effective data scopes, and trace evidence.

## Grill-Me Pressure Test Consensus

- Should this be a write-capable permission management endpoint? No. It is an explanation surface only.
- Should it return only allowed runtime paths? No. Deny and invalid rows are important audit evidence.
- Should it duplicate the runtime evaluator? No. It uses the same data-scope narrowing primitive but does not replace `EvaluateCapabilityAccess`.
- Should missing tenant IDs become errors? No. Existing flat tenant compatibility remains part of the product contract.
- Should this be a UI first? No. The profile API is the backend contract the UI should consume later.
