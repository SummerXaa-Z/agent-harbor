# Tenant Hierarchy and Scoped Administration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add explicit three-level tenant hierarchy, subtree-scoped management reads, and parent-to-child capability entitlement validation.

**Architecture:** Tenant hierarchy is a small domain model plus store-backed descendant resolution. Existing flat `tenantId` behavior remains exact-match when a tenant is not registered. Capability entitlement creation allows same-tenant grants or target-tenant-to-descendant grants, while runtime still requires an explicit child entitlement chain.

**Tech Stack:** Go domain/store/httpapi layers, in-memory repository, PostgreSQL migrations and repository, existing Go tests and shell demo scripts.

---

### Task 1: Tenant Domain and Repository Contract

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/store/memory.go`

- [ ] Add tenant types in `internal/domain/types.go`:

```go
type TenantStatus string

const (
    TenantStatusActive   TenantStatus = "active"
    TenantStatusDisabled TenantStatus = "disabled"
)

type Tenant struct {
    ID             string       `json:"id"`
    ParentTenantID string       `json:"parentTenantId,omitempty"`
    Level          int          `json:"level"`
    Name           string       `json:"name"`
    Status         TenantStatus `json:"status"`
    CreatedAt      time.Time    `json:"createdAt"`
    UpdatedAt      time.Time    `json:"updatedAt"`
}

type CreateTenantRequest struct {
    ID             string       `json:"id"`
    ParentTenantID string       `json:"parentTenantId"`
    Name           string       `json:"name"`
    Status         TenantStatus `json:"status"`
}
```

- [ ] Add store filter and repository methods in `internal/store/memory.go`:

```go
type TenantFilter struct {
    TenantID       string
    ParentTenantID string
}

type TenantAuditBuilder func(domain.Tenant) domain.AuditEvent

CreateTenant(context.Context, domain.Tenant) (domain.Tenant, error)
CreateTenantWithAudit(context.Context, domain.Tenant, TenantAuditBuilder) (domain.Tenant, error)
ListTenants(context.Context, TenantFilter) ([]domain.Tenant, error)
GetTenant(context.Context, string) (domain.Tenant, bool, error)
```

### Task 2: Memory Store Tenant Hierarchy and Subtree Scope

**Files:**
- Modify: `internal/store/memory.go`
- Modify: `internal/store/memory_test.go`

- [ ] Add `tenants map[string]domain.Tenant` to `Memory` and initialize it.
- [ ] Implement create/list/get tenant methods. `ListTenants(TenantFilter{TenantID: "root"})` returns root plus descendants; `ParentTenantID` filters direct children.
- [ ] Add helper:

```go
func (m *Memory) tenantIDsForScopeLocked(tenantID string) map[string]struct{} {
    tenantID = strings.TrimSpace(tenantID)
    if tenantID == "" {
        return nil
    }
    if _, ok := m.tenants[tenantID]; !ok {
        return map[string]struct{}{tenantID: {}}
    }
    ids := map[string]struct{}{tenantID: {}}
    for changed := true; changed; {
        changed = false
        for id, tenant := range m.tenants {
            if _, exists := ids[id]; exists {
                continue
            }
            if _, parentIncluded := ids[tenant.ParentTenantID]; parentIncluded {
                ids[id] = struct{}{}
                changed = true
            }
        }
    }
    return ids
}
```

- [ ] Update memory list filters to use descendant tenant sets where applicable.
- [ ] Add tests proving registered root scope includes descendant agents and unregistered flat scope remains exact.

### Task 3: PostgreSQL Tenant Persistence

**Files:**
- Create: `internal/db/migrations/008_tenant_hierarchy.sql`
- Modify: `internal/store/postgres.go`
- Modify: `internal/store/postgres_test.go`

- [ ] Add migration:

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

create index if not exists tenants_parent_idx on tenants(parent_tenant_id);
```

- [ ] Implement Postgres tenant create/list/get and audit transaction method.
- [ ] Add Postgres descendant helper using recursive CTE and fallback exact-match when no tenant row exists.
- [ ] Update Postgres list methods to use descendant tenant arrays.
- [ ] Extend Postgres tests with tenant hierarchy round trip and descendant agent list scope.

### Task 4: Tenant HTTP API

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [ ] Register routes inside the admin group:

```go
r.Post("/tenants", s.createTenant)
r.Get("/tenants", s.listTenants)
r.Get("/tenants/{id}", s.getTenant)
```

- [ ] Add request normalization:

```go
func normalizeTenantStatus(value domain.TenantStatus, fallback domain.TenantStatus) (domain.TenantStatus, error)
```

- [ ] Implement `tenantFromRequest`:
  - Trim ID, parent, and name.
  - Generate `security.NewID("ten")` when ID is empty.
  - Require name.
  - Root level is 1.
  - Parent must exist, be active, and have level lower than 3.
  - Child level is parent level + 1.

- [ ] Implement create/list/get handlers and management audit event `tenant.created`.
- [ ] Add HTTP tests for root/child/grandchild creation, fourth-level rejection, and tenant subtree listing.

### Task 5: Parent-to-Child Entitlement Validation

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [ ] Replace same-tenant-only validation in `createTenantEntitlement` with:

```go
allowed, err := s.tenantCanReceiveTargetEntitlement(r.Context(), target.TenantID, req.TenantID)
if err != nil {
    writeError(w, err)
    return
}
if !allowed {
    writeError(w, domain.BadRequest("VALIDATION_FAILED", "tenantId must match target tenantId or be a descendant tenant"))
    return
}
```

- [ ] Implement ancestry helper:

```go
func (s *Server) tenantCanReceiveTargetEntitlement(ctx context.Context, targetTenantID string, granteeTenantID string) (bool, error) {
    if targetTenantID == granteeTenantID {
        return true, nil
    }
    grantee, ok, err := s.repo.GetTenant(ctx, granteeTenantID)
    if err != nil || !ok {
        return false, err
    }
    for current := grantee; current.ParentTenantID != ""; {
        if current.ParentTenantID == targetTenantID {
            return true, nil
        }
        parent, ok, err := s.repo.GetTenant(ctx, current.ParentTenantID)
        if err != nil || !ok {
            return false, err
        }
        current = parent
    }
    return false, nil
}
```

- [ ] Add HTTP tests:
  - Root target can grant approved capability to descendant tenant.
  - Root target cannot grant approved capability to unrelated registered tenant.
  - Same-tenant legacy grant still works without tenant records.

### Task 6: Docs and Demo

**Files:**
- Modify: `README.md`
- Modify: `Makefile`
- Modify: `scripts/demo-all.sh`
- Create: `scripts/demo-sprint14-tenant-hierarchy.sh`

- [ ] Document tenant API and grant semantics.
- [ ] Add demo script that creates root/child/grandchild tenants, rejects level 4, and proves root target can grant to child tenant.
- [ ] Add the script to demo lint and demo-all.

### Task 7: Verification and Finish

- [ ] Run focused tests:

```sh
go test ./internal/store ./internal/httpapi
```

- [ ] Run full tests:

```sh
go test ./...
```

- [ ] Run demo lint:

```sh
make demo-scripts-lint
```

- [ ] Run diff checks:

```sh
git diff --check
git status --short
```

- [ ] Commit, push, and create a stacked PR on top of `codex/data-permission-enforcement`.

