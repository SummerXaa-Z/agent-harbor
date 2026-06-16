# Admin Tenant Boundary Design

## Goal

Make the control plane enforce who an authenticated administrator can manage by tenant subtree and workspace, instead of relying on UI filters or caller-supplied query parameters.

This is the next production-readiness slice because AgentHarbor governs tenant-scoped access for agents and tools; the management console itself must follow the same boundary.

## Current State

- `AGENT_HARBOR_ADMIN_IDENTITIES` maps `actor=key`.
- Console sessions store only `actor` and expiration.
- Store list APIs already support `store.ManagementScope{TenantID, WorkspaceID}` and registered tenant subtree expansion.
- Approval reviewer routing supports `reviewer=tenantId/workspaceId`, but it applies only to approval queues and approval decisions.
- Most list routes use `managementScopeFromRequest(r)`, which reads `tenantId` and `workspaceId` from query parameters.
- Some internal reads still use empty scope or direct ids when validating permission packages, capabilities, applications, or access profiles.

The missing link is a server-side scope derived from the authenticated administrator identity.

## Design Decisions

### 1. Identity Scope Model

Add a first-class admin scope alongside the existing actor and key:

```text
actor
key
role
tenantId
workspaceId
```

Recommended roles for this slice:

- `platform_admin`: full management scope.
- `tenant_admin`: restricted to one tenant subtree and optional workspace.
- `security_reviewer`: restricted like `tenant_admin`, but approval reviewer validation still controls review actions.

`AGENT_HARBOR_ADMIN_IDENTITIES` remains backward compatible:

```text
actor=key
```

New scoped entries use a pipe-delimited value:

```text
actor=key|role=tenant_admin|tenant=tenant-east|workspace=ws-support
actor=key|role=platform_admin
```

Rationale: keep local setup simple, avoid a new config file or database table in this slice, and preserve current deployments.

### 2. Session Payload

Console session payload should include actor, role, tenantId, and workspaceId. `/api/v1/auth/session` returns the same fields so the UI can show a persistent "you are managing X" context.

Sessions signed before this change should become invalid naturally if they lack required fields for scoped identities. Development bypass remains `local-dev` with `platform_admin` scope.

### 3. Effective Management Scope

Add one backend helper that all management handlers use:

```text
effectiveManagementScope(requestedScope, authenticatedIdentityScope) -> allowed scope or 403
```

Rules:

- `platform_admin` may request any scope or no scope.
- `tenant_admin` and `security_reviewer` must stay inside their configured tenant subtree.
- If the identity has a workspace, requested workspace must match it.
- If the request omits tenant/workspace, the server defaults to the identity scope rather than widening access.
- If requested and identity scopes do not intersect, return `403 permission denied`.

This helper must be used for list filters and for mutation inputs before writes.

### 4. Mutation Enforcement

Every write path that creates, updates, approves, applies, refreshes, or exports production state must verify the target object is inside the authenticated admin scope before performing side effects.

Minimum covered routes for the implementation slice:

- tenants
- agents and keys
- access grants
- route policies
- capabilities refresh/update
- permission package drafts, preflight, approvals, apply, production readiness, production report
- tenant entitlements, workspace assignments, instance assignments
- audit and trace reads
- management MCP tools that call the same operations

For object-id routes, the server must load the object first, derive its tenant/workspace, then authorize. A missing object remains `404`; an existing but out-of-scope object returns `403`.

### 5. Frontend Behavior

The console should treat the authenticated admin scope as the default working context:

- Show a compact persistent identity/scope indicator near connection settings.
- Hide unavailable tenants and workspaces by relying on scoped API responses.
- Do not let the user clear a scoped identity into a wider query.
- Keep technical overrides available only when the current identity scope allows them.

No large UI redesign is required in this slice.

### 6. Audit Trail

Management audit events should continue to record the actor. For scoped identities, audit metadata should include:

- `adminRole`
- `adminTenantId`
- `adminWorkspaceId`

This makes later reviews answer whether an operator acted inside their assigned boundary.

## Alternatives Considered

### A. Full IAM With Database-Managed Users And Roles

Pros: best long-term model.

Cons: too large for the current production-hardening slice; would require user lifecycle, password or SSO integration, migrations, UI management, and new recovery paths.

Decision: defer.

### B. Frontend-Only Tenant Context

Pros: quick and visually useful.

Cons: not a security boundary. API callers and management MCP could bypass it.

Decision: reject.

### C. Reuse Approval Reviewer Rules For All Admin Scope

Pros: fewer env variables.

Cons: approval reviewer rules express review routing, not general management authority. Reusing them would blur requester, reviewer, and admin roles.

Decision: keep approval routing separate, but allow the same actor to be configured in both systems.

## Acceptance Criteria

1. A scoped admin login response includes actor, role, tenantId, and workspaceId.
2. A scoped admin listing agents, tenants, capabilities, assignments, applications, traces, or audit records only sees its configured tenant subtree and workspace.
3. A scoped admin cannot create or mutate an out-of-scope resource.
4. A scoped admin cannot approve, reject, apply, preflight, check, or export a permission package outside its scope.
5. Management MCP requests inherit the authenticated admin scope and cannot widen it through tool arguments.
6. Audit events written by scoped admins include role and scope metadata.
7. Existing `actor=key` identity config and `AGENT_HARBOR_ADMIN_KEY` behavior remain compatible.
8. `make check` and `make release-check` include a scenario proving cross-tenant access is blocked.

## Non-Goals

- No password management.
- No user invitation flow.
- No SSO.
- No persistent admin database table.
- No new frontend tenant-admin management page.
- No change to data-plane Agent Key authorization.

## Risks

- Some handlers currently load related objects with empty scope. The implementation must audit these paths carefully so permission package operations cannot bypass admin scope through indirect ids.
- `tenantId` subtree behavior must stay compatible with existing registered and unregistered tenant matching.
- Development bypass must remain easy locally, but production preflight must continue rejecting it.

## Recommended Implementation Order

1. Extend admin identity parsing and session response with role/scope.
2. Add pure authorization helpers and tests for scope intersection.
3. Replace query-only `managementScopeFromRequest` usage with authenticated effective scope.
4. Add mutation guards for primary resource and permission package operations.
5. Extend management MCP to consume the same authenticated scope.
6. Add frontend identity/scope indicator and scoped default context.
7. Add release scenario and docs.
