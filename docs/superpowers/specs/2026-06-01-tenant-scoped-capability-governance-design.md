# Tenant-Scoped Capability Governance Design

Date: 2026-06-01
Status: Product and architecture consensus

## Product Thesis

AgentHarbor should evolve from an Agent-to-Agent gateway into a tenant-scoped AI capability, permission, and data governance gateway.

The primary user journey is:

```text
Register a target
  -> discover its capabilities and native permission model
  -> classify the data and risk boundaries
  -> grant selected capabilities and data scopes to a tenant
  -> assign those grants to tenant workspaces and caller instances
  -> enforce the decision at runtime
  -> prove the result with trace and audit evidence
```

The core product moment is not creating an Agent. It is when a platform administrator can register a black-box MCP server, Agent, OpenAPI service, workflow, database, warehouse, or lake resource, see exactly what it can do, grant a safe subset to a tenant, and verify that an actual call is allowed or denied for the expected reason.

## Three-Level Scope Model

AgentHarbor needs a stable scope hierarchy so tenant isolation, permission inheritance, and audit evidence do not become ambiguous.

```text
L1 Platform / Provider
  Owns global targets, connector definitions, discovery jobs, risk defaults, and platform policies.

L2 Tenant / Customer
  Owns customer-level entitlements and the primary data boundary.

L3 Workspace / Business Unit / Project
  Owns tenant-internal segmentation for departments, environments, projects, or use cases.
```

Concrete runtime identity then sits below the three scope levels:

```text
platformId
tenantId
workspaceId
callerInstanceId
subjectId
```

Where:

- `platformId` identifies the AgentHarbor control-plane provider or deployment boundary.
- `tenantId` identifies the customer-level entitlement and data boundary.
- `workspaceId` identifies the tenant-internal operational boundary.
- `callerInstanceId` identifies the concrete execution identity, such as an Agent, app, service account, or connector instance.
- `subjectId` is optional and identifies the end user, service principal, or workload subject behind the caller instance.

Permissions must only narrow as they move down the hierarchy:

```text
Instance permissions <= Workspace assignments <= Tenant entitlements <= Platform-open capabilities
```

A workspace cannot enable a capability the tenant has not received. A caller instance cannot use a capability the workspace has not assigned. A newly discovered capability is denied everywhere until it is explicitly reviewed and granted.

## Core Objects

### Target

A target is a registered external or internal system that exposes callable capabilities. Existing `Agent` records cover some targets today, but the product model should not be limited to Agent-shaped systems.

Initial target types:

- `mcp_server`
- `agent`
- `openapi_service`
- `database`
- `warehouse`
- `data_lake`
- `workflow`
- `saas_api`

For incremental implementation, existing `Agent` can continue to represent the target record while the capability layer is introduced around it.

### Capability

A capability is the smallest governable unit AgentHarbor can discover, grant, assign, enforce, and audit.

Examples:

- MCP tool: `search_customer`
- MCP method: `tools/list`
- Agent action: `summarize_account`
- OpenAPI operation: `GET /orders/{id}`
- Database table: `crm.customers`
- Warehouse dataset: `sales.forecast`
- Data lake object prefix: `s3://lake/contracts/tenant-a/`
- Workflow action: `approve_invoice`
- SaaS API scope: `crm.contacts.read`

Canonical shape:

```text
Capability {
  id
  targetId
  type
  key
  displayName
  description
  action
  inputSchema
  outputSchema
  nativeScopes
  dataDomains
  dataScopes
  sensitivity
  riskLevel
  enforcementMode
  discoveryStatus
  version
  discoveredAt
  updatedAt
}
```

Important fields:

- `type`: `mcp_tool`, `mcp_method`, `agent_action`, `openapi_operation`, `database_table`, `warehouse_dataset`, `lake_object`, `workflow_action`, `saas_scope`.
- `action`: `read`, `write`, `delete`, `export`, `admin`, or `execute`.
- `nativeScopes`: permissions required by the target's own permission system.
- `dataScopes`: structured data boundaries AgentHarbor can reason about.
- `sensitivity`: `public`, `internal`, `confidential`, `restricted`.
- `riskLevel`: `low`, `medium`, `high`, `critical`.
- `enforcementMode`: `gateway`, `context_forwarded`, `downstream_native`, `advisory`.
- `discoveryStatus`: `pending_review`, `approved`, `deprecated`, `removed`.

### Data Scope

Data permissions are part of capability governance, not a separate afterthought. A grant should answer both "what can be called" and "what data can that call touch."

Initial data scope dimensions:

```text
dataDomain
dataset
schema
table
field
classification
region
tenantFilter
maskingPolicy
rowFilter
```

P0 should store, evaluate, forward, and audit these scopes. It should not attempt to replace every downstream data authorization engine at once.

### Tenant Entitlement

A tenant entitlement grants a tenant access to a selected capability and selected data scopes.

```text
TenantEntitlement {
  id
  platformId
  tenantId
  targetId
  capabilityId
  effect
  dataScopes
  status
  priority
  createdAt
  updatedAt
}
```

Default behavior is deny. Explicit deny wins over allow at the same or lower level.

### Workspace Assignment

A workspace assignment narrows a tenant entitlement to a workspace.

```text
WorkspaceAssignment {
  id
  tenantEntitlementId
  workspaceId
  effect
  dataScopes
  status
  createdAt
  updatedAt
}
```

Workspace data scopes must be equal to or narrower than the tenant entitlement's data scopes.

### Instance Assignment

An instance assignment grants a workspace-available capability to a concrete caller instance.

```text
InstanceAssignment {
  id
  workspaceAssignmentId
  callerInstanceId
  subjectSelector
  effect
  dataScopes
  status
  createdAt
  updatedAt
}
```

Initial caller instance types:

- `agent`
- `app`
- `service_account`
- `user_session`

Current Agent Keys can continue to authenticate Agent caller instances, but runtime context should grow beyond `callerAgentId`.

## Discovery Model

AgentHarbor should discover capabilities from each target type through a target-specific adapter, then normalize the result into the Capability Catalog.

Initial discovery priorities:

1. MCP tools and MCP methods
2. Manually registered Agent actions
3. OpenAPI operations from an OpenAPI document
4. Manually registered data capabilities for database, warehouse, and lake targets

MCP discovery should call the target's `tools/list` method and store:

- tool name
- tool title or display name when available
- description
- input schema
- output schema when available
- target id
- discovery version
- discovered timestamp

Discovery diff rules:

- New capability: create as `pending_review`, default denied.
- Changed schema or description: increment version and mark for review.
- Removed capability: mark `removed`, keep historical trace references resolvable.
- Reappeared capability: restore to `pending_review` unless an administrator explicitly approved the new version.

## Runtime Enforcement

Runtime enforcement must evaluate the full identity and scope chain.

```text
Authenticate caller
  -> resolve platformId, tenantId, workspaceId, callerInstanceId, subjectId
  -> identify target and capability
  -> check target status
  -> check capability status and version
  -> evaluate tenant entitlement
  -> evaluate workspace assignment
  -> evaluate instance assignment
  -> evaluate data scope policy
  -> apply explicit deny overrides
  -> forward scoped context or deny the call
  -> record trace evidence
```

For MCP:

- `tools/list` should return only tools available to the resolved tenant, workspace, and caller instance.
- `tools/call` should parse the tool name from `params.name` and evaluate the corresponding `mcp_tool` capability.
- Other MCP methods can initially be represented as `mcp_method` capabilities.
- If a requested tool is not discovered, removed, pending review, or unassigned, return `PERMISSION_DENIED`.

For downstream forwarding:

- AgentHarbor should inject a signed or otherwise trusted context containing tenant, workspace, caller instance, subject, capability, and data scope claims.
- Downstream systems may enforce detailed row, column, object, or business permissions from that context.
- AgentHarbor trace evidence must record what context it forwarded, without leaking secrets or sensitive payloads.

## Trace And Audit Evidence

Trace evidence should explain runtime decisions.

Minimum fields to add over the current trace shape:

```text
platformId
tenantId
workspaceId
callerInstanceId
subjectId
targetId
capabilityId
capabilityVersion
entitlementId
workspaceAssignmentId
instanceAssignmentId
dataScopes
decision
reason
upstreamStatus
upstreamError
createdAt
```

Management audit should record:

- target registration
- capability discovery refresh
- capability review and approval
- tenant entitlement creation, update, disable
- workspace assignment creation, update, disable
- instance assignment creation, update, disable
- data scope policy changes

Audit metadata must contain capability ids, versions, data scope keys, and policy ids, but never plaintext credentials or raw sensitive payloads.

## Console Implications

The console's primary journey should become:

```text
Register Target
  -> Discover Capabilities
  -> Review Capability Diff
  -> Grant to Tenant
  -> Assign to Workspace
  -> Assign to Instance
  -> Test Call
  -> Inspect Trace Evidence
```

Primary views:

- Capability Catalog
- Discovery Diff
- Tenant Entitlements
- Workspace Assignments
- Instance Assignments
- Data Scope Policies
- Call Simulator
- Trace Evidence

The existing cockpit can remain useful, but the first-screen product story should be capability governance rather than Agent inventory.

## First Build Slice

The first engineering slice should be MCP Capability Governance MVP.

Acceptance:

1. Register or reuse an MCP target with an upstream endpoint.
2. Refresh discovery and store MCP tools as capabilities.
3. Newly discovered tools are default denied.
4. Grant one discovered tool to a tenant.
5. Assign that entitlement to one workspace and one caller Agent instance.
6. `tools/list` returns only assigned tools for that instance.
7. `tools/call` for an unassigned tool is denied.
8. `tools/call` for an assigned tool is allowed and proxied or locally accepted.
9. Trace evidence includes tenant, workspace, caller instance, capability id, decision, and reason.

## Non-Goals For The First Build Slice

- No full replacement of existing Agent, AccessGrant, or RoutePolicy APIs.
- No general SQL parser or database proxy.
- No full OpenAPI operation discovery.
- No self-service tenant administration UI beyond the minimal console path.
- No external audit outbox.
- No automatic risk classification beyond deterministic defaults.

## Migration Strategy

Do not big-bang rewrite the gateway.

The existing model can bridge as follows:

- Existing `Agent` remains the initial target and caller instance record.
- Existing `tenantId` and `workspaceId` continue to scope management records.
- Existing `RoutePolicy` and `AccessGrant` remain as compatibility paths.
- New capability-based evaluation should run before legacy route-policy and access-grant fallback for MCP tool calls.
- Trace records should be backward compatible: existing fields remain, new fields are optional during migration.

Once MCP capability governance is stable, later sprints can fold route policies into capability policies and expand discovery to OpenAPI, Agent actions, and data systems.
