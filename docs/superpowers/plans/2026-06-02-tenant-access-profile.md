# Tenant Access Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Add a read-only tenant access profile API that explains configured capability grants, effective data scopes, and recent runtime trace evidence for a tenant scope.

**Architecture:** Add a focused `internal/httpapi/access_profile.go` handler and builder that joins existing repository list/get methods in memory. The endpoint is management-only, does not add new storage, and preserves registered tenant subtree semantics plus flat tenant exact-match fallback.

**Tech Stack:** Go HTTP API, existing `store.Repository`, existing `domain.EffectiveDataScopes`, Go HTTP tests, shell demo scripts.

---

### Task 1: HTTP Tests for Profile Shape and Grant Chain

**Files:**
- Create: `internal/httpapi/access_profile_test.go`

- [x] **Step 1: Write failing test coverage**

Add `access_profile_test.go` in package `httpapi_test` with local response structs:

```go
type accessProfileResponse struct {
	Tenant       tenantResponse               `json:"tenant"`
	ScopeTenants []tenantResponse             `json:"scopeTenants"`
	Summary      accessProfileSummaryResponse `json:"summary"`
	Grants       []accessProfileGrantResponse `json:"grants"`
	RecentTraces []traceResponse              `json:"recentTraces"`
}

type accessProfileSummaryResponse struct {
	TenantCount                 int `json:"tenantCount"`
	GrantCount                  int `json:"grantCount"`
	TargetCount                 int `json:"targetCount"`
	CapabilityCount             int `json:"capabilityCount"`
	WorkspaceAssignmentCount    int `json:"workspaceAssignmentCount"`
	InstanceAssignmentCount     int `json:"instanceAssignmentCount"`
	RecentAllowedTraceCount     int `json:"recentAllowedTraceCount"`
	RecentDeniedTraceCount      int `json:"recentDeniedTraceCount"`
}

type accessProfileGrantResponse struct {
	TenantEntitlement         tenantEntitlementResponse              `json:"tenantEntitlement"`
	Target                    *agentResponse                         `json:"target"`
	Capability                *capabilityResponse                    `json:"capability"`
	EffectiveTenantDataScopes []domain.DataScope                     `json:"effectiveTenantDataScopes"`
	ScopeStatus               string                                 `json:"scopeStatus"`
	ScopeReason               string                                 `json:"scopeReason"`
	WorkspaceAssignments      []accessProfileWorkspaceResponse       `json:"workspaceAssignments"`
}

type accessProfileWorkspaceResponse struct {
	WorkspaceAssignment          workspaceAssignmentResponse          `json:"workspaceAssignment"`
	EffectiveWorkspaceDataScopes []domain.DataScope                  `json:"effectiveWorkspaceDataScopes"`
	ScopeStatus                  string                              `json:"scopeStatus"`
	ScopeReason                  string                              `json:"scopeReason"`
	InstanceAssignments          []accessProfileInstanceResponse     `json:"instanceAssignments"`
}

type accessProfileInstanceResponse struct {
	InstanceAssignment          instanceAssignmentResponse `json:"instanceAssignment"`
	CallerInstance              *agentResponse             `json:"callerInstance"`
	EffectiveInstanceDataScopes []domain.DataScope         `json:"effectiveInstanceDataScopes"`
	ScopeStatus                 string                     `json:"scopeStatus"`
	ScopeReason                 string                     `json:"scopeReason"`
}
```

Add `TestTenantAccessProfileExplainsGrantChainAndRecentTraces`:

- Create root, child, and grandchild tenants via `POST /api/v1/tenants`.
- Seed a root MCP target and approved capability directly through the repo.
- Create a tenant entitlement for the child tenant.
- Create a workspace assignment and instance assignment with progressively narrower data scopes.
- Append one allowed and one denied trace.
- Request `GET /api/v1/tenants/{child}/access-profile?traceLimit=1`.
- Assert:
  - status `200`
  - `tenant.id == child`
  - `scopeTenants == [child, grandchild]`
  - one grant for the child entitlement
  - target and capability are populated
  - effective tenant/workspace/instance scopes are narrowed as expected
  - returned trace is the newest trace and summary counts reflect the returned trace

Add `TestTenantAccessProfileFlatTenantIsExactMatch`:

- Seed two flat tenant strings, `tenant-flat` and `tenant-flat-child`.
- Give both tenants entitlements.
- Request `GET /api/v1/tenants/tenant-flat/access-profile?traceLimit=0`.
- Assert only `tenant-flat` appears and only the exact tenant's grant is returned.

Add `TestTenantAccessProfileReportsInvalidScopeAndValidatesTraceLimit`:

- Seed an entitlement with one data scope.
- Seed a workspace assignment directly through the repo with a broader/conflicting data scope.
- Request the profile and assert the grant exists with a workspace row whose `scopeStatus == "invalid"`.
- Request `traceLimit=101` and assert `400 VALIDATION_FAILED`.

- [x] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/httpapi -run 'TestTenantAccessProfile' -count=1
```

Expected result: FAIL because `/api/v1/tenants/{id}/access-profile` is not registered yet.

### Task 2: Access Profile Handler and Builder

**Files:**
- Create: `internal/httpapi/access_profile.go`
- Modify: `internal/httpapi/server.go`

- [x] **Step 1: Add the admin route**

In `Server.Router`, register this route in the admin group next to the tenant routes:

```go
r.Get("/tenants/{id}/access-profile", s.getTenantAccessProfile)
```

- [x] **Step 2: Implement query parsing**

In `access_profile.go`, add `accessProfileQueryFromRequest`:

```go
const (
	defaultAccessProfileTraceLimit = 20
	maxAccessProfileTraceLimit     = 100
)

type accessProfileQuery struct {
	WorkspaceID      string
	TargetID         string
	CapabilityID     string
	CallerInstanceID string
	TraceLimit       int
}
```

Rules:

- Trim all string filters.
- Default `traceLimit` to `20`.
- Allow `0` through `100`.
- Return `domain.BadRequest("VALIDATION_FAILED", "traceLimit must be between 0 and 100")` for invalid values.

- [x] **Step 3: Implement response structs**

In `access_profile.go`, define response structs mirroring the spec:

```go
type tenantAccessProfileResponse struct {
	Tenant       domain.Tenant                 `json:"tenant"`
	ScopeTenants []domain.Tenant               `json:"scopeTenants"`
	Summary      tenantAccessProfileSummary    `json:"summary"`
	Grants       []tenantAccessProfileGrant    `json:"grants"`
	RecentTraces []domain.TraceEvent           `json:"recentTraces"`
	GeneratedAt  time.Time                     `json:"generatedAt"`
}
```

Use nested structs for grant, workspace assignment, and instance assignment with `scopeStatus` and `scopeReason`.

- [x] **Step 4: Implement profile building**

Implement:

```go
func (s *Server) getTenantAccessProfile(w http.ResponseWriter, r *http.Request)
func (s *Server) buildTenantAccessProfile(ctx context.Context, tenantID string, query accessProfileQuery) (tenantAccessProfileResponse, error)
```

Builder requirements:

- Registered tenant: use `GetTenant` and `ListTenants(TenantFilter{TenantID: tenantID})`.
- Unregistered tenant: synthesize `domain.Tenant{ID: tenantID, Name: tenantID, Level: 0, Status: domain.TenantStatusActive}` and keep exact-match list semantics by passing the raw tenant ID to existing store filters.
- Load entitlements, workspace assignments, instance assignments, traces, targets, and capabilities through existing repository methods.
- Compute effective scopes with `domain.EffectiveDataScopes`.
- Include denied/disabled rows, but mark only broken references or invalid scope narrowing as `scopeStatus: "invalid"`.
- Return recent traces newest-first after all filters are applied.
- Compute summary counts from the filtered response.

- [x] **Step 5: Run focused tests and verify GREEN**

Run:

```bash
go test ./internal/httpapi -run 'TestTenantAccessProfile' -count=1
```

Expected result: PASS.

### Task 3: Docs, Demo, and Full Verification

**Files:**
- Modify: `README.md`
- Modify: `Makefile`
- Modify: `scripts/demo-all.sh`
- Create: `scripts/demo-sprint15-tenant-access-profile.sh`
- Modify: `docs/superpowers/plans/2026-06-02-tenant-access-profile.md`

- [x] **Step 1: Add README API docs**

Document:

```http
GET /api/v1/tenants/{id}/access-profile?workspaceId=&targetId=&capabilityId=&callerInstanceId=&traceLimit=
```

Explain that it is a read-only explanation endpoint and preserves registered-subtree plus flat exact-match tenant scope semantics.

- [x] **Step 2: Add Sprint 15 demo**

Create `scripts/demo-sprint15-tenant-access-profile.sh`:

- Create a tenant tree.
- Create local target/caller agents when `MCP_ENDPOINT` is absent.
- If `MCP_ENDPOINT` is present, use MCP discovery and approval.
- Create tenant entitlement, workspace assignment, and instance assignment with data scopes.
- Fetch the profile and assert grant chain and effective data scopes using Python JSON checks.

- [x] **Step 3: Register demo script**

Add the script to `DEMO_SCRIPTS` in `Makefile` and to `scripts/demo-all.sh`.

- [x] **Step 4: Run verification**

Run:

```bash
go test ./internal/httpapi -run 'TestTenantAccessProfile' -count=1
go test ./...
make demo-scripts-lint
git diff --check
```

Expected result: all commands pass.

- [x] **Step 5: Mark plan complete and commit**

Mark this plan's checkboxes complete as tasks finish, then commit implementation changes with:

```bash
git add .
git commit -m "Add tenant access profile API"
```
