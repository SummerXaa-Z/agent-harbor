# Admin Tenant Access Management Design

## Goal

Make administrator identities and tenant/workspace boundaries manageable inside AgentHarbor instead of relying only on deployment-time environment variables.

The previous boundary slice made scoped administrators enforceable at runtime. This slice makes those boundaries operable: a platform administrator can create, review, rotate, disable, and audit scoped administrator identities without editing deployment configuration, while tenant administrators remain confined to their assigned tenant subtree and workspace.

## Current State

- `AGENT_HARBOR_ADMIN_KEY` creates a global `admin-key` platform administrator.
- `AGENT_HARBOR_ADMIN_IDENTITIES` creates named in-memory administrators with actor, key, role, tenant, and workspace scope.
- Console sessions resolve an authenticated actor back to configured identities.
- Management REST and MCP handlers already intersect tenant/workspace requests with the authenticated administrator scope.
- Store implementations persist tenants, agents, permissions, route policies, runtime traces, and management audit records.
- There is no persistent administrator identity table, no key rotation path for administrators, no disabled state, no "last platform admin" guard, and no product workspace for tenant-boundary administration.

## Design Decisions

### 1. Scope This Slice To Platform-Managed Administrator Identities

Only `platform_admin` can manage administrator identities in this slice.

`tenant_admin` and `security_reviewer` keep their existing operational authority inside their scope, but they cannot create, rotate, disable, or widen administrator identities. This avoids introducing delegated IAM before recovery, audit, and least-privilege rules are mature.

### 2. Keep Bootstrap Identities, Add Managed Identities

Environment-configured identities remain supported as bootstrap identities:

- `AGENT_HARBOR_ADMIN_KEY`
- `AGENT_HARBOR_ADMIN_IDENTITIES`

Bootstrap identities are visible in the product as read-only administrator entries with `source=bootstrap`. They cannot be rotated or disabled from the console because deployment configuration is still their source of truth.

New product-created administrator identities are persisted with `source=managed`.

### 3. Persist Only Hashed Administrator Keys

Add a persisted `AdminIdentity` domain model:

```text
id
actor
displayName
role
tenantId
workspaceId
status
source
keyHash
keyPrefix
createdAt
updatedAt
lastUsedAt
rotatedAt
disabledAt
createdBy
updatedBy
disabledBy
```

Rules:

- `role` is one of `platform_admin`, `tenant_admin`, or `security_reviewer`.
- `status` is `active` or `disabled`.
- `source` is `bootstrap` or `managed`.
- Managed keys are generated server-side, returned once on create or rotate, and never stored in plaintext.
- `tenant_admin` and `security_reviewer` require `tenantId`; `workspaceId` is optional.
- `platform_admin` must not carry a tenant/workspace restriction in this slice.
- `actor` must be unique across bootstrap and managed identities.

### 4. Authentication Resolution

Administrator authentication checks active identities in this order:

1. Managed identities by key hash.
2. Bootstrap named identities.
3. Legacy `AGENT_HARBOR_ADMIN_KEY`.
4. Explicit local development bypass, only when enabled.

Actor lookup for signed console sessions must re-resolve the actor against active identities. Disabled managed identities immediately invalidate future session checks.

Bootstrap identity actor names reserve their namespace. A managed identity cannot reuse a bootstrap actor.

### 5. Lifecycle Operations

Add REST endpoints under existing admin authentication:

```text
GET  /api/v1/admin-identities
POST /api/v1/admin-identities
POST /api/v1/admin-identities/{id}/key:rotate
POST /api/v1/admin-identities/{id}:disable
```

The first implementation intentionally does not support deletion. Disable is reversible only by creating a new identity or a future re-enable endpoint, which reduces accidental recovery ambiguity.

Required guards:

- Only platform administrators can call these endpoints.
- A platform administrator cannot disable their own active identity.
- The API must reject disabling or rotating the last active platform administrator if no other active platform administrator remains across bootstrap and managed identities.
- A managed identity cannot widen beyond valid role rules.
- Key material never appears in list responses or audit metadata.
- Every lifecycle operation writes a management audit record.

### 6. Frontend Workspace

Add a configuration workspace named **Administrators & Boundaries** / **管理员与边界**.

The page should be business-readable:

- Header summary: active administrators, tenant-scoped administrators, disabled administrators, bootstrap entries.
- Table/list: display name, actor, role, scope, source, status, key prefix, last used, updated time.
- Primary action: "Create administrator" / "创建管理员", opening a modal form.
- Row actions: rotate key and disable for managed identities only.
- One-time key reveal after create or rotate with copy affordance and clear warning that it will not be shown again.
- Current session chip should link operators to this workspace when they need to understand their boundary.

Avoid raw tenant IDs in the main path when tenant names are available. Technical IDs can stay in collapsed technical details.

### 7. AI-Friendly Management Surface

Expose the same lifecycle through management MCP tools so an admin agent can configure the platform:

- `list_admin_identities`
- `create_admin_identity`
- `rotate_admin_identity_key`
- `disable_admin_identity`

The MCP tools use the same platform-admin guard and return the same one-time key behavior for create/rotate. They must not allow tenant admins or scoped reviewers to manage administrators.

### 8. Audit And Runtime Signals

Lifecycle audit actions:

- `admin_identity.created`
- `admin_identity.key_rotated`
- `admin_identity.disabled`

Audit metadata may include role, target tenant/workspace, source, and key prefix. It must never include the plaintext key or key hash.

Successful managed-key authentication should update `lastUsedAt` best-effort. A failure to update `lastUsedAt` must not block login, but store errors during identity create, rotate, or disable must fail the operation.

## Alternatives Considered

### A. Full IAM With Passwords, Invitations, And SSO

Pros: complete long-term account model.

Cons: too broad for the next production-readiness slice; it would require password policy, invitation lifecycle, SSO/OIDC decisions, recovery flows, and more UI than the current product needs.

Decision: defer.

### B. Environment-Only Admin Identity UI

Pros: small implementation.

Cons: it does not solve key rotation, disabled administrators, auditability, or low-friction tenant-boundary operations for non-technical administrators.

Decision: reject as insufficient.

### C. Let Tenant Administrators Create Subordinate Administrators

Pros: closer to delegated enterprise administration.

Cons: high risk without a mature IAM model. It introduces role delegation, subtree creation rules, recovery edge cases, and self-lockout risks.

Decision: defer. Tenant admins remain scoped operators, not identity administrators, in this slice.

## Acceptance Criteria

1. A platform administrator can create a managed tenant administrator with tenant/workspace scope and receives the generated key only once.
2. The managed tenant administrator can log in and is restricted by the existing tenant/workspace boundary enforcement.
3. A tenant administrator or security reviewer cannot list, create, rotate, or disable administrator identities.
4. Rotating a managed administrator key invalidates the old key and returns a new one-time key.
5. Disabling a managed administrator invalidates key login and future console-session resolution.
6. The API blocks disabling the current administrator and blocks removing the last active platform administrator.
7. Bootstrap identities appear read-only and cannot be disabled or rotated from the product.
8. Audit records exist for create, rotate, and disable without plaintext key material.
9. The console includes a bilingual Administrators & Boundaries workspace using modal forms for create, rotate, and disable flows.
10. Management MCP exposes equivalent AI-friendly administrator lifecycle tools with identical authorization.
11. `make check` and `make release-check` include focused coverage for creation, scoped login, key rotation, disable, audit, and last-platform-admin protection.

## Non-Goals

- No password login.
- No SSO/OIDC.
- No email invitation flow.
- No delegated tenant-admin identity management.
- No administrator deletion endpoint.
- No changes to data-plane Agent key authorization.
- No external publishing or repository visibility change.

## Risks

- Mixing bootstrap and managed identity lookup can create actor conflicts. The implementation must reject managed actors that collide with bootstrap actors.
- Session re-resolution must stay strict; otherwise disabled identities may keep working until cookie expiry.
- Last-platform-admin protection must count both active bootstrap platform identities and active managed platform identities.
- Returning generated keys once is easy to regress through frontend state or logs; tests should assert list responses and audit metadata omit secrets.
- Management MCP adds an AI-friendly path, so it must share backend handlers or guards rather than duplicating authorization logic.

## Recommended Implementation Order

1. Add red backend tests for managed identity creation, login, key rotation, disable, last-platform guard, and forbidden scoped-admin management.
2. Add `domain.AdminIdentity`, repository methods, memory store behavior, and PostgreSQL migration.
3. Refactor admin authentication to resolve managed and bootstrap identities through one identity provider path.
4. Add REST handlers and audit events for lifecycle operations.
5. Add management MCP tools using the same lifecycle service.
6. Add frontend types/API client, controller hook, i18n strings, and Administrators & Boundaries workspace.
7. Add scenario gate and docs updates.
