# Tenant Hierarchy and Scoped Administration Design

Date: 2026-06-02
Status: Approved by user direction to proceed autonomously

## Goal

Add an explicit three-level tenant hierarchy so AgentHarbor can distinguish tenants, delegate MCP capability access from a parent tenant to child tenants, and scope management reads across a tenant subtree.

## Product Journey

The next most important journey is:

1. A platform/provider tenant registers a target MCP server or agent.
2. The platform creates customer or sub-customer tenants under it.
3. The platform grants selected target capabilities to a child tenant.
4. The child tenant narrows access through workspace and instance assignments.
5. Runtime evaluates the concrete caller tenant, workspace, instance, subject, capability, and data scopes.
6. Management views can query a parent tenant and see records in its tenant subtree.

This extends the capability/data-permission path without building a full organization/RBAC product yet.

## Scope

This MVP adds tenant hierarchy and parent-to-child capability entitlement validation.

In scope:

- Tenant objects with `id`, `parentTenantId`, `level`, `name`, `status`, `createdAt`, and `updatedAt`.
- Maximum depth of 3 tenant levels.
- `POST /api/v1/tenants`
- `GET /api/v1/tenants?tenantId=&parentTenantId=`
- `GET /api/v1/tenants/{id}`
- Management list filters that include descendants when `tenantId` identifies a registered tenant.
- Tenant entitlement creation that allows a target owner tenant to grant capabilities to itself or a descendant tenant.
- Backward compatibility for existing flat tenant IDs when no tenant record exists.

Out of scope:

- Tenant update/delete.
- Workspace registry.
- User/admin role membership.
- Full admin actor authorization by tenant.
- Automatic entitlement inheritance from parent tenant to descendants.
- Frontend tenant tree UI.

## Tenant Model

```text
Tenant {
  id
  parentTenantId
  level
  name
  status
  createdAt
  updatedAt
}
```

`parentTenantId` is empty for level-1 tenants. A child tenant's level is computed from the parent:

```text
root level = 1
child level = parent.level + 1
max level = 3
```

Tenant IDs remain external-facing strings. `POST /api/v1/tenants` accepts an optional `id`; when omitted AgentHarbor generates a `ten_` ID. This keeps compatibility with existing demo values such as `default` and `tenant-a` while still supporting generated IDs.

Tenant status values:

- `active`
- `disabled`

Only active parent tenants can receive new child tenants.

## Management Scope Semantics

Existing management APIs already accept `tenantId` and `workspaceId`. This feature changes tenant filtering only when `tenantId` maps to a registered tenant:

- If `tenantId` exists in the tenant registry, list APIs match that tenant and all descendants.
- If `tenantId` is not registered, list APIs keep current exact-match behavior.
- `workspaceId` remains an exact filter.

This preserves existing tests and local flows that use tenant strings before creating tenant records.

Affected management reads:

- agents
- api keys
- access grants
- route policies
- capabilities
- tenant entitlements
- workspace assignments
- instance assignments
- audit events
- audit traces
- runtime metrics through trace filtering

## Capability Entitlement Semantics

Current behavior requires the entitlement tenant to equal the target agent tenant. That blocks the actual delegation journey where a platform tenant owns the target and grants it to a child tenant.

New behavior:

```text
tenant entitlement tenant must be:
  target tenant itself
  OR a descendant of target tenant
```

If either tenant is unregistered, cross-tenant grants are rejected, except the existing same-tenant case. This gives flat legacy tenants the old behavior while allowing registered hierarchy tenants to delegate safely.

Runtime access remains explicit. A parent entitlement does not automatically apply to child tenants. The child tenant must have its own tenant entitlement, then workspace and instance assignments.

## Data Permission Interaction

This feature does not change `dataScopes`. It relies on the previous enforcement layer:

```text
capability scopes >= tenant entitlement scopes >= workspace assignment scopes >= instance assignment scopes
```

Tenant hierarchy validates who may receive an entitlement. Data scopes validate how far that entitlement may reach.

## Error Handling

- Creating a level-4 tenant returns `400 VALIDATION_FAILED`.
- Creating a child under a missing parent returns `404 NOT_FOUND`.
- Creating a child under a disabled parent returns `400 VALIDATION_FAILED`.
- Creating a cross-tenant entitlement outside the target tenant subtree returns `400 VALIDATION_FAILED`.
- Duplicate tenant IDs surface as repository errors; API tests avoid relying on database-specific duplicate messages.

## Storage

Memory store adds a `tenants` map.

PostgreSQL adds migration `008_tenant_hierarchy.sql`:

```sql
create table if not exists tenants (
  id text primary key,
  parent_tenant_id text references tenants(id) on delete restrict,
  level integer not null,
  name text not null,
  status text not null,
  created_at timestamptz not null,
  updated_at timestamptz not null
);
```

PostgreSQL descendant lookups use a recursive CTE. Because max depth is 3, the query is bounded in practice and simple to reason about.

## API

Create tenant:

```http
POST /api/v1/tenants
```

```json
{
  "id": "tenant-enterprise-a",
  "parentTenantId": "platform",
  "name": "Enterprise A",
  "status": "active"
}
```

List tenants:

```http
GET /api/v1/tenants?tenantId=platform
GET /api/v1/tenants?parentTenantId=platform
```

`tenantId` returns the subtree. `parentTenantId` returns direct children. If both are supplied, both predicates apply.

Get tenant:

```http
GET /api/v1/tenants/{id}
```

## Testing

Unit and integration tests should prove:

- Creating root, child, and grandchild tenants works.
- Creating a fourth-level tenant fails.
- Listing by root tenant returns descendants.
- Existing flat management scope behavior remains exact when no tenant record exists.
- Management scope for a registered root tenant includes descendant agent records.
- A root target can grant a capability to a descendant tenant.
- A root target cannot grant a capability to an unrelated registered tenant.
- PostgreSQL round trip covers tenant create/list/get and descendant list scope.

## Grill-Me Pressure Test Consensus

- Should parent entitlements automatically flow to children? No. That would be convenient but too broad. This MVP requires explicit child entitlements.
- Should workspace become tenant level 3? Not in this implementation. Existing `workspaceId` stays tenant-internal, because replacing it now would explode migration and UI scope.
- Should unregistered tenant IDs be rejected everywhere? No. Existing clean-room demos and local workflows use tenant strings freely. Backward compatibility is more valuable until the UI owns tenant creation.
- Should target tenant be allowed to grant to a child tenant? Yes. This is the core missing step in the platform/provider-to-tenant journey.
- Should list scopes include descendants by default? Yes, but only when the filter tenant is registered. That gives scoped administration without breaking flat exact-match semantics.

