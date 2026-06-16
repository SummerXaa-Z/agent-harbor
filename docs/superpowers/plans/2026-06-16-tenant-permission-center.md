# Tenant Permission Center Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the existing tenant organization workspace into a read-only tenant permission center that shows tenant governance, administrator boundary, permission package state, capabilities, data scopes, and safe next actions.

**Architecture:** Add one backend read-only projection endpoint for `GET /api/v1/tenants/{id}/permission-center`, built from existing tenant, admin identity, entitlement, assignment, capability, permission-package application, and access-profile data. Add a focused frontend presenter and client types, then update `TenantOrganizationView` to render the projection while keeping all writes inside Permission Changes, Access Profile, or Admin Boundaries.

**Tech Stack:** Go HTTP API, in-memory and PostgreSQL-backed repository interfaces already available through `store.Repository`, React/TypeScript console, Node test runner source tests, shell release scenarios, `make check`, `make release-check`.

---

## File Map

- Create: `internal/httpapi/tenant_permission_center.go` — response types and read-only projection builder for tenant permission center.
- Modify: `internal/httpapi/server.go` — register `/api/v1/tenants/{id}/permission-center`.
- Modify: `internal/httpapi/server_test.go` — platform/scoped admin tests, redaction test, and projection summary test.
- Modify: `frontend/src/types.ts` — `TenantPermissionCenter*` response types.
- Modify: `frontend/src/api.ts` — `fetchTenantPermissionCenter`.
- Create: `frontend/src/tenantPermissionCenter.ts` — pure presenter for UI summary, next actions, and business labels.
- Create: `frontend/tests/tenantPermissionCenter.test.mjs` — presenter tests.
- Modify: `frontend/src/components/TenantOrganizationView.tsx` — render permission center state and safe actions.
- Modify: `frontend/src/ConsoleController.tsx` — load projection when tenant workspace is active and pass it to the view.
- Modify: `frontend/src/i18n.ts` — English and Simplified Chinese copy.
- Modify: `frontend/tests/i18n.test.mjs` — bilingual copy coverage.
- Modify: `frontend/tests/styleTheme.test.mjs` or `frontend/tests/permissionFlowLayout.test.mjs` — source guards for tenant center layout, raw ID containment, and handoff links.
- Create: `scripts/scenario-tenant-permission-center.sh` — release scenario for scoped tenant admin center access and handoff readiness.
- Modify: `Makefile`, `tests/makefile_targets_test.sh`, `README.md`, `CHANGELOG.md` — scenario wiring and docs.

---

### Task 1: Backend Red Tests For Tenant Permission Center

**Files:**
- Modify: `internal/httpapi/server_test.go`

- [ ] **Step 1: Add response test structs**

Add these structs near the existing test response structs:

```go
type tenantPermissionCenterResponse struct {
	Tenant           tenantResponse                            `json:"tenant"`
	ScopeTenants     []tenantResponse                          `json:"scopeTenants"`
	OperatorBoundary tenantPermissionCenterOperatorBoundary     `json:"operatorBoundary"`
	Administrators   []tenantPermissionCenterAdministrator      `json:"administrators"`
	Workspaces       []tenantPermissionCenterWorkspace          `json:"workspaces"`
	PermissionPacks  []tenantPermissionCenterPermissionPackage  `json:"permissionPackages"`
	Capabilities     []tenantPermissionCenterCapability         `json:"capabilities"`
	NextActions      []tenantPermissionCenterNextAction         `json:"nextActions"`
	GeneratedAt      string                                    `json:"generatedAt"`
}

type tenantPermissionCenterOperatorBoundary struct {
	Actor                   string `json:"actor"`
	Role                    string `json:"role"`
	TenantID                string `json:"tenantId,omitempty"`
	WorkspaceID             string `json:"workspaceId,omitempty"`
	CanManageAdministrators bool   `json:"canManageAdministrators"`
}

type tenantPermissionCenterAdministrator struct {
	ID          string `json:"id"`
	Actor       string `json:"actor"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	TenantID    string `json:"tenantId,omitempty"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	Status      string `json:"status"`
	Source      string `json:"source"`
}

type tenantPermissionCenterWorkspace struct {
	WorkspaceID     string `json:"workspaceId"`
	CallerCount     int    `json:"callerCount"`
	TargetCount     int    `json:"targetCount"`
	AssignmentCount int    `json:"assignmentCount"`
}

type tenantPermissionCenterPermissionPackage struct {
	TemplateID               string             `json:"templateId"`
	TemplateName             string             `json:"templateName"`
	Status                   string             `json:"status"`
	AllowedCapabilityCount   int                `json:"allowedCapabilityCount"`
	BlockedCapabilityCount   int                `json:"blockedCapabilityCount"`
	DataScopes               []domain.DataScope `json:"dataScopes,omitempty"`
	LatestApplicationID      string             `json:"latestApplicationId,omitempty"`
}

type tenantPermissionCenterCapability struct {
	TargetID       string             `json:"targetId"`
	TargetName     string             `json:"targetName"`
	CapabilityID   string             `json:"capabilityId"`
	CapabilityName string             `json:"capabilityName"`
	Effect         string             `json:"effect"`
	DataScopes     []domain.DataScope `json:"dataScopes,omitempty"`
	WorkspaceIDs   []string           `json:"workspaceIds"`
}

type tenantPermissionCenterNextAction struct {
	Code       string `json:"code"`
	TargetView string `json:"targetView"`
}
```

- [ ] **Step 2: Add fixture helper**

Add this helper near the scoped-admin fixture helpers:

```go
func seedTenantPermissionCenterFixture(t *testing.T, repo store.Repository) (tenantID string, workspaceID string, caller domain.Agent, target domain.Agent, capability domain.Capability) {
	t.Helper()
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root-center", "", "Platform Operations", now)
	createDirectTenant(t, repo, "tenant-child-center", "tenant-root-center", "Customer Service", now)
	caller = createDirectAgent(t, repo, "Support Assistant", "tenant-child-center", "ws-support-center", "local", domain.AgentStatusActive, nil)
	target = createDirectAgent(t, repo, "Ticket Tool Service", "tenant-child-center", "ws-support-center", "mcp", domain.AgentStatusActive, nil)
	capability = createDirectCapabilityWithAction(t, repo, target.ID, "search_ticket", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	entitlement := createDirectTenantEntitlement(t, repo, "tenant-child-center", target.ID, capability.ID, []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)
	workspaceAssignment := createDirectWorkspaceAssignment(t, repo, entitlement.ID, "tenant-child-center", "ws-support-center", []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)
	createDirectInstanceAssignment(t, repo, workspaceAssignment.ID, "tenant-child-center", "ws-support-center", caller.ID, []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)
	return "tenant-child-center", "ws-support-center", caller, target, capability
}
```

The helper definitions for tenant entitlements, workspace assignments, and instance assignments already live in `internal/httpapi/access_profile_test.go` and are available to this package test binary.

- [ ] **Step 3: Add platform projection test**

Add:

```go
func TestTenantPermissionCenterSummarizesTenantForPlatformAdmin(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	tenantID, workspaceID, _, target, capability := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	_, err := repo.CreateAdminIdentityWithAudit(context.Background(), domain.AdminIdentity{
		ID:          "adm_center_tenant",
		Actor:       "tenant-admin-center",
		DisplayName: "Tenant Center Admin",
		Role:        domain.AdminIdentityRoleTenantAdmin,
		KeyHash:     security.HashSecret("tenant-admin-key"),
		KeyPrefix:   "ahadm_center",
		Status:      domain.AdminIdentityStatusActive,
		Source:      domain.AdminIdentitySourceManaged,
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, func(identity domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: "audit_center_admin", Action: "admin_identity.created", ResourceType: "admin_identity", ResourceID: identity.ID, Actor: "platform", CreatedAt: now}
	})
	if err != nil {
		t.Fatalf("seed admin identity: %v", err)
	}

	center := decodeData[tenantPermissionCenterResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "platform-key"))
	if center.Tenant.ID != tenantID {
		t.Fatalf("unexpected tenant center tenant: %#v", center.Tenant)
	}
	if center.OperatorBoundary.Actor != "platform" || !center.OperatorBoundary.CanManageAdministrators {
		t.Fatalf("platform boundary should manage administrators, got %#v", center.OperatorBoundary)
	}
	if len(center.Administrators) != 1 || center.Administrators[0].Actor != "tenant-admin-center" {
		t.Fatalf("expected tenant administrator summary, got %#v", center.Administrators)
	}
	if len(center.Workspaces) != 1 || center.Workspaces[0].WorkspaceID != workspaceID || center.Workspaces[0].CallerCount != 1 || center.Workspaces[0].TargetCount != 1 {
		t.Fatalf("unexpected workspace summary: %#v", center.Workspaces)
	}
	if len(center.Capabilities) != 1 || center.Capabilities[0].CapabilityID != capability.ID || center.Capabilities[0].TargetID != target.ID || center.Capabilities[0].Effect != "allow" {
		t.Fatalf("unexpected capability summary: %#v", center.Capabilities)
	}
	if len(center.PermissionPacks) == 0 || center.PermissionPacks[0].Status == "" {
		t.Fatalf("expected permission package projection, got %#v", center.PermissionPacks)
	}
	if got := tenantCenterActionCodes(center.NextActions); !reflect.DeepEqual(got, []string{"manage_administrators", "open_access_profile", "start_permission_change"}) {
		t.Fatalf("unexpected next actions: %#v", got)
	}
	raw := mustJSON(t, center)
	if bytes.Contains(raw, []byte("tenant-admin-key")) || bytes.Contains(raw, []byte("keyHash")) {
		t.Fatalf("tenant center must not expose admin key material: %s", raw)
	}
}

func tenantCenterActionCodes(actions []tenantPermissionCenterNextAction) []string {
	codes := make([]string, 0, len(actions))
	for _, action := range actions {
		codes = append(codes, action.Code)
	}
	sort.Strings(codes)
	return codes
}
```

- [ ] **Step 4: Add scoped-admin boundary tests**

Add:

```go
func TestTenantPermissionCenterHonorsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, workspaceID, _, _, _ := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-west-center", "tenant-root-center", "Finance", now)

	center := decodeData[tenantPermissionCenterResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "east-key"))
	if center.OperatorBoundary.Actor != "east-admin" || center.OperatorBoundary.CanManageAdministrators {
		t.Fatalf("tenant admin boundary should be read-only for admin management, got %#v", center.OperatorBoundary)
	}
	if center.OperatorBoundary.TenantID != tenantID || center.OperatorBoundary.WorkspaceID != workspaceID {
		t.Fatalf("unexpected scoped boundary: %#v", center.OperatorBoundary)
	}
	if got := tenantCenterActionCodes(center.NextActions); slices.Contains(got, "manage_administrators") {
		t.Fatalf("tenant admin should not get manage_administrators action: %#v", got)
	}

	widen := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/tenant-west-center/permission-center", nil, "", "east-key")
	if widen.Code != http.StatusForbidden {
		t.Fatalf("tenant admin should not fetch outside tenant center, got %d body=%s", widen.Code, widen.Body.String())
	}
}
```

Add `slices` and `sort` to the test imports when these tests are added.

- [ ] **Step 5: Add unregistered tenant test**

Add:

```go
func TestTenantPermissionCenterRequiresRegisteredTenant(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/unregistered/permission-center", nil, "", "platform-key")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("permission center should require registered tenant, got %d body=%s", resp.Code, resp.Body.String())
	}
}
```

- [ ] **Step 6: Run red backend tests**

```bash
go test ./internal/httpapi -run 'TenantPermissionCenter' -count=1
```

Expected: FAIL because `/api/v1/tenants/{id}/permission-center` is not registered.

- [ ] **Step 7: Commit red tests**

```bash
git add internal/httpapi/server_test.go
git commit -m "test: cover tenant permission center projection"
```

---

### Task 2: Backend Tenant Permission Center Projection

**Files:**
- Create: `internal/httpapi/tenant_permission_center.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [ ] **Step 1: Create response types**

Create `internal/httpapi/tenant_permission_center.go` with:

```go
package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	tenantCenterStatusReady       = "ready"
	tenantCenterStatusNeedsReview = "needs_review"
	tenantCenterStatusBlocked     = "blocked"

	tenantCenterActionStartPermissionChange = "start_permission_change"
	tenantCenterActionOpenAccessProfile     = "open_access_profile"
	tenantCenterActionManageAdministrators  = "manage_administrators"
	tenantCenterActionCompleteSetup         = "complete_setup"
)

type tenantPermissionCenterResponse struct {
	Tenant           domain.Tenant                         `json:"tenant"`
	ScopeTenants     []domain.Tenant                       `json:"scopeTenants"`
	OperatorBoundary tenantPermissionCenterOperatorBoundary `json:"operatorBoundary"`
	Administrators   []tenantPermissionCenterAdministrator  `json:"administrators"`
	Workspaces       []tenantPermissionCenterWorkspace      `json:"workspaces"`
	PermissionPacks  []tenantPermissionCenterPackage        `json:"permissionPackages"`
	Capabilities     []tenantPermissionCenterCapability     `json:"capabilities"`
	NextActions      []tenantPermissionCenterNextAction      `json:"nextActions"`
	GeneratedAt      time.Time                              `json:"generatedAt"`
}

type tenantPermissionCenterOperatorBoundary struct {
	Actor                   string `json:"actor"`
	Role                    string `json:"role"`
	TenantID                string `json:"tenantId,omitempty"`
	WorkspaceID             string `json:"workspaceId,omitempty"`
	CanManageAdministrators bool   `json:"canManageAdministrators"`
}

type tenantPermissionCenterAdministrator struct {
	ID          string `json:"id"`
	Actor       string `json:"actor"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	TenantID    string `json:"tenantId,omitempty"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	Status      string `json:"status"`
	Source      string `json:"source"`
}

type tenantPermissionCenterWorkspace struct {
	WorkspaceID     string `json:"workspaceId"`
	CallerCount     int    `json:"callerCount"`
	TargetCount     int    `json:"targetCount"`
	AssignmentCount int    `json:"assignmentCount"`
}

type tenantPermissionCenterPackage struct {
	TemplateID              string             `json:"templateId"`
	TemplateName            string             `json:"templateName"`
	Status                  string             `json:"status"`
	AllowedCapabilityCount  int                `json:"allowedCapabilityCount"`
	BlockedCapabilityCount  int                `json:"blockedCapabilityCount"`
	DataScopes              []domain.DataScope `json:"dataScopes,omitempty"`
	LatestApplicationID     string             `json:"latestApplicationId,omitempty"`
}

type tenantPermissionCenterCapability struct {
	TargetID       string             `json:"targetId"`
	TargetName     string             `json:"targetName"`
	CapabilityID   string             `json:"capabilityId"`
	CapabilityName string             `json:"capabilityName"`
	Effect         string             `json:"effect"`
	DataScopes     []domain.DataScope `json:"dataScopes,omitempty"`
	WorkspaceIDs   []string           `json:"workspaceIds"`
}

type tenantPermissionCenterNextAction struct {
	Code       string `json:"code"`
	TargetView string `json:"targetView"`
}
```

- [ ] **Step 2: Add handler and scope guard**

Append to `tenant_permission_center.go`:

```go
func (s *Server) getTenantPermissionCenter(w http.ResponseWriter, r *http.Request) {
	response, err := s.buildTenantPermissionCenterForRequest(r, chi.URLParam(r, "id"), r.URL.Query().Get("workspaceId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) buildTenantPermissionCenterForRequest(r *http.Request, tenantID string, workspaceID string) (tenantPermissionCenterResponse, error) {
	tenantID = strings.TrimSpace(tenantID)
	workspaceID = strings.TrimSpace(workspaceID)
	tenant, ok, err := s.repo.GetTenant(r.Context(), tenantID)
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	if !ok {
		return tenantPermissionCenterResponse{}, domain.NotFound("tenant not found")
	}
	requestedScope := store.ManagementScope{TenantID: tenant.ID, WorkspaceID: workspaceID}
	if err := s.requireRequestedScopeAllowed(r, requestedScope); err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	principal, _ := requestAdminPrincipal(r)
	effectiveScope, err := s.effectiveManagementScope(r.Context(), requestedScope, principal)
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	return s.buildTenantPermissionCenter(r.Context(), tenant, effectiveScope.WorkspaceID, principal)
}
```

- [ ] **Step 3: Add projection builder**

Append:

```go
func (s *Server) buildTenantPermissionCenter(ctx context.Context, tenant domain.Tenant, workspaceID string, principal adminPrincipal) (tenantPermissionCenterResponse, error) {
	scopeTenants, err := s.repo.ListTenants(ctx, store.TenantFilter{TenantID: tenant.ID})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	if len(scopeTenants) == 0 {
		scopeTenants = []domain.Tenant{tenant}
	}
	scope := store.ManagementScope{TenantID: tenant.ID, WorkspaceID: workspaceID}
	agents, err := s.repo.ListAgents(ctx, store.AgentFilter{ManagementScope: scope})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	entitlements, err := s.repo.ListTenantEntitlements(ctx, store.EntitlementFilter{ManagementScope: store.ManagementScope{TenantID: tenant.ID}})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	assignments, err := s.repo.ListWorkspaceAssignments(ctx, store.AssignmentFilter{ManagementScope: scope})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	applications, err := s.repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{ManagementScope: store.ManagementScope{TenantID: tenant.ID, WorkspaceID: workspaceID}})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	admins, err := s.tenantPermissionCenterAdministrators(ctx, tenant.ID, workspaceID, principal)
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	capabilities, err := s.tenantPermissionCenterCapabilities(ctx, entitlements, assignments)
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	response := tenantPermissionCenterResponse{
		Tenant:           tenant,
		ScopeTenants:     scopeTenants,
		OperatorBoundary: tenantPermissionCenterOperatorBoundaryFromPrincipal(principal),
		Administrators:   admins,
		Workspaces:       tenantPermissionCenterWorkspaces(agents, assignments),
		PermissionPacks:  s.tenantPermissionCenterPackages(applications, entitlements),
		Capabilities:     capabilities,
		GeneratedAt:      s.now(),
	}
	response.NextActions = tenantPermissionCenterNextActions(response)
	return response, nil
}
```

- [ ] **Step 4: Implement admin redaction and scope matching**

Add:

```go
func tenantPermissionCenterOperatorBoundaryFromPrincipal(principal adminPrincipal) tenantPermissionCenterOperatorBoundary {
	principal = normalizeAdminPrincipal(principal)
	return tenantPermissionCenterOperatorBoundary{
		Actor:                   principal.Actor,
		Role:                    principal.Role,
		TenantID:                principal.TenantID,
		WorkspaceID:             principal.WorkspaceID,
		CanManageAdministrators: principal.Role == adminRolePlatformAdmin || principal.Role == "",
	}
}

func (s *Server) tenantPermissionCenterAdministrators(ctx context.Context, tenantID string, workspaceID string, principal adminPrincipal) ([]tenantPermissionCenterAdministrator, error) {
	if normalizeAdminRole(principal.Role) != adminRolePlatformAdmin {
		return nil, nil
	}
	rows := append([]domain.AdminIdentity{}, s.bootstrapAdminIdentities()...)
	managed, err := s.repo.ListAdminIdentities(ctx)
	if err != nil {
		return nil, err
	}
	rows = append(rows, managed...)
	result := []tenantPermissionCenterAdministrator{}
	for _, row := range rows {
		if !tenantPermissionCenterAdminMatches(row, tenantID, workspaceID) {
			continue
		}
		result = append(result, tenantPermissionCenterAdministrator{
			ID:          row.ID,
			Actor:       row.Actor,
			DisplayName: row.DisplayName,
			Role:        string(row.Role),
			TenantID:    row.TenantID,
			WorkspaceID: row.WorkspaceID,
			Status:      string(row.Status),
			Source:      string(row.Source),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Actor < result[j].Actor })
	return result, nil
}

func tenantPermissionCenterAdminMatches(identity domain.AdminIdentity, tenantID string, workspaceID string) bool {
	if identity.Role == domain.AdminIdentityRolePlatformAdmin {
		return false
	}
	if identity.TenantID != "" && identity.TenantID != tenantID {
		return false
	}
	if workspaceID != "" && identity.WorkspaceID != "" && identity.WorkspaceID != workspaceID {
		return false
	}
	return identity.TenantID == tenantID || identity.WorkspaceID == workspaceID
}
```

- [ ] **Step 5: Implement summaries**

Add deterministic summary helpers:

```go
func tenantPermissionCenterWorkspaces(agents []domain.Agent, assignments []domain.WorkspaceAssignment) []tenantPermissionCenterWorkspace {
	byID := map[string]*tenantPermissionCenterWorkspace{}
	ensure := func(workspaceID string) *tenantPermissionCenterWorkspace {
		if byID[workspaceID] == nil {
			byID[workspaceID] = &tenantPermissionCenterWorkspace{WorkspaceID: workspaceID}
		}
		return byID[workspaceID]
	}
	for _, agent := range agents {
		row := ensure(agent.WorkspaceID)
		if agent.ChannelType == "local" {
			row.CallerCount++
		} else {
			row.TargetCount++
		}
	}
	for _, assignment := range assignments {
		ensure(assignment.WorkspaceID).AssignmentCount++
	}
	rows := make([]tenantPermissionCenterWorkspace, 0, len(byID))
	for _, row := range byID {
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].WorkspaceID < rows[j].WorkspaceID })
	return rows
}

func (s *Server) tenantPermissionCenterPackages(applications []domain.PermissionPackageApplication, entitlements []domain.TenantEntitlement) []tenantPermissionCenterPackage {
	latestByTemplate := map[string]domain.PermissionPackageApplication{}
	for _, application := range applications {
		current, ok := latestByTemplate[application.TemplateID]
		if !ok || application.AppliedAt.After(current.AppliedAt) {
			latestByTemplate[application.TemplateID] = application
		}
	}
	rows := []tenantPermissionCenterPackage{}
	for _, template := range permissionpack.Templates() {
		application, hasApplication := latestByTemplate[template.ID]
		allowed := 0
		denied := 0
		scopes := []domain.DataScope{}
		for _, entitlement := range entitlements {
			if entitlement.Effect == domain.PolicyEffectAllow {
				allowed++
				scopes = append(scopes, entitlement.DataScopes...)
			}
			if entitlement.Effect == domain.PolicyEffectDeny {
				denied++
			}
		}
		status := tenantCenterStatusNeedsReview
		latestID := ""
		if hasApplication {
			status = tenantCenterStatusReady
			latestID = application.ID
		}
		if allowed == 0 && denied == 0 {
			status = tenantCenterStatusBlocked
		}
		rows = append(rows, tenantPermissionCenterPackage{
			TemplateID:             template.ID,
			TemplateName:           template.Name,
			Status:                 status,
			AllowedCapabilityCount: allowed,
			BlockedCapabilityCount: denied,
			DataScopes:             scopes,
			LatestApplicationID:    latestID,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].TemplateID < rows[j].TemplateID })
	return rows
}
```

Add `github.com/SummerXaa-Z/agent-harbor/internal/permissionpack` to imports.

- [ ] **Step 6: Implement capability summary**

Add:

```go
func (s *Server) tenantPermissionCenterCapabilities(ctx context.Context, entitlements []domain.TenantEntitlement, assignments []domain.WorkspaceAssignment) ([]tenantPermissionCenterCapability, error) {
	workspaceIDsByEntitlement := map[string][]string{}
	for _, assignment := range assignments {
		workspaceIDsByEntitlement[assignment.TenantEntitlementID] = appendUniqueString(workspaceIDsByEntitlement[assignment.TenantEntitlementID], assignment.WorkspaceID)
	}
	rows := []tenantPermissionCenterCapability{}
	for _, entitlement := range entitlements {
		targetName := entitlement.TargetID
		target, ok, err := s.repo.GetAgent(ctx, entitlement.TargetID)
		if err != nil {
			return nil, err
		}
		if ok && strings.TrimSpace(target.Name) != "" {
			targetName = target.Name
		}
		capabilityName := entitlement.CapabilityID
		capability, ok, err := s.repo.GetCapability(ctx, entitlement.CapabilityID)
		if err != nil {
			return nil, err
		}
		if ok && strings.TrimSpace(capability.DisplayName) != "" {
			capabilityName = capability.DisplayName
		}
		rows = append(rows, tenantPermissionCenterCapability{
			TargetID:       entitlement.TargetID,
			TargetName:     targetName,
			CapabilityID:   entitlement.CapabilityID,
			CapabilityName: capabilityName,
			Effect:         string(entitlement.Effect),
			DataScopes:     entitlement.DataScopes,
			WorkspaceIDs:   workspaceIDsByEntitlement[entitlement.ID],
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TargetName == rows[j].TargetName {
			return rows[i].CapabilityName < rows[j].CapabilityName
		}
		return rows[i].TargetName < rows[j].TargetName
	})
	return rows, nil
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
```

- [ ] **Step 7: Implement next actions**

Add:

```go
func tenantPermissionCenterNextActions(center tenantPermissionCenterResponse) []tenantPermissionCenterNextAction {
	actions := []tenantPermissionCenterNextAction{
		{Code: tenantCenterActionStartPermissionChange, TargetView: "ai-admin"},
		{Code: tenantCenterActionOpenAccessProfile, TargetView: "access"},
	}
	if center.OperatorBoundary.CanManageAdministrators {
		actions = append(actions, tenantPermissionCenterNextAction{Code: tenantCenterActionManageAdministrators, TargetView: "admin-access"})
	}
	if len(center.Workspaces) == 0 || len(center.Capabilities) == 0 {
		actions = append(actions, tenantPermissionCenterNextAction{Code: tenantCenterActionCompleteSetup, TargetView: "getting-started"})
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Code < actions[j].Code })
	return actions
}
```

- [ ] **Step 8: Register route**

In `internal/httpapi/server.go`, add the route immediately before `/tenants/{id}/access-profile` so it is not shadowed by `/tenants/{id}`:

```go
r.Get("/tenants/{id}/permission-center", s.getTenantPermissionCenter)
r.Get("/tenants/{id}/access-profile", s.getTenantAccessProfile)
```

- [ ] **Step 9: Run backend tests**

```bash
gofmt -w internal/httpapi/tenant_permission_center.go internal/httpapi/server.go internal/httpapi/server_test.go
go test ./internal/httpapi -run 'TenantPermissionCenter' -count=1
```

Expected: PASS.

- [ ] **Step 10: Commit backend projection**

```bash
git add internal/httpapi/tenant_permission_center.go internal/httpapi/server.go internal/httpapi/server_test.go
git commit -m "feat: add tenant permission center projection"
```

---

### Task 3: Frontend Types, API Client, And Presenter

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Create: `frontend/src/tenantPermissionCenter.ts`
- Create: `frontend/tests/tenantPermissionCenter.test.mjs`

- [ ] **Step 1: Add frontend response types**

Add to `frontend/src/types.ts` after admin identity types:

```ts
export type TenantPermissionCenterStatus = 'ready' | 'needs_review' | 'blocked'
export type TenantPermissionCenterActionTarget = 'ai-admin' | 'access' | 'admin-access' | 'getting-started'

export interface TenantPermissionCenterOperatorBoundary {
  actor: string
  role: AdminIdentityRole
  tenantId?: string
  workspaceId?: string
  canManageAdministrators: boolean
}

export interface TenantPermissionCenterAdministrator {
  id: string
  actor: string
  displayName: string
  role: AdminIdentityRole
  tenantId?: string
  workspaceId?: string
  status: AdminIdentityStatus
  source: AdminIdentitySource
}

export interface TenantPermissionCenterWorkspace {
  workspaceId: string
  callerCount: number
  targetCount: number
  assignmentCount: number
}

export interface TenantPermissionCenterPackage {
  templateId: string
  templateName: string
  status: TenantPermissionCenterStatus
  allowedCapabilityCount: number
  blockedCapabilityCount: number
  dataScopes?: DataScope[]
  latestApplicationId?: string
}

export interface TenantPermissionCenterCapability {
  targetId: string
  targetName: string
  capabilityId: string
  capabilityName: string
  effect: RoutePolicyEffect
  dataScopes?: DataScope[]
  workspaceIds: string[]
}

export interface TenantPermissionCenterNextAction {
  code: string
  targetView: TenantPermissionCenterActionTarget
}

export interface TenantPermissionCenterResponse {
  tenant: Tenant
  scopeTenants: Tenant[]
  operatorBoundary: TenantPermissionCenterOperatorBoundary
  administrators: TenantPermissionCenterAdministrator[]
  workspaces: TenantPermissionCenterWorkspace[]
  permissionPackages: TenantPermissionCenterPackage[]
  capabilities: TenantPermissionCenterCapability[]
  nextActions: TenantPermissionCenterNextAction[]
  generatedAt: string
}
```

- [ ] **Step 2: Add API client**

Import the new response type in `frontend/src/api.ts`, then add:

```ts
export async function fetchTenantPermissionCenter(
  tenantId: string,
  workspaceId?: string,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<TenantPermissionCenterResponse> {
  const query = queryString({ workspaceId })
  return request<TenantPermissionCenterResponse>(
    `/api/v1/tenants/${encodeURIComponent(tenantId.trim())}/permission-center${query}`,
    { adminKey, signal },
  )
}
```

- [ ] **Step 3: Write presenter red tests**

Create `frontend/tests/tenantPermissionCenter.test.mjs`:

```js
import assert from "node:assert/strict";
import test from "node:test";

import {
  buildTenantPermissionCenterViewModel,
  tenantPermissionCenterActionTarget,
} from "../src/tenantPermissionCenter.ts";

const now = "2026-06-16T12:00:00Z";
const tenant = { createdAt: now, id: "tenant-child", level: 2, name: "Customer Service", parentTenantId: "tenant-root", status: "active", updatedAt: now };
const root = { createdAt: now, id: "tenant-root", level: 1, name: "Group HQ", status: "active", updatedAt: now };

const center = {
  administrators: [
    { actor: "tenant-admin", displayName: "Tenant Admin", id: "adm-1", role: "tenant_admin", source: "managed", status: "active", tenantId: tenant.id, workspaceId: "ws-support" },
  ],
  capabilities: [
    { capabilityId: "cap-search", capabilityName: "Search tickets", dataScopes: [{ dataDomain: "support", region: "us-east" }], effect: "allow", targetId: "target-ticket", targetName: "Ticket Tool Service", workspaceIds: ["ws-support"] },
    { capabilityId: "cap-export", capabilityName: "Export tickets", dataScopes: [], effect: "deny", targetId: "target-ticket", targetName: "Ticket Tool Service", workspaceIds: ["ws-support"] },
  ],
  generatedAt: now,
  nextActions: [
    { code: "start_permission_change", targetView: "ai-admin" },
    { code: "open_access_profile", targetView: "access" },
    { code: "manage_administrators", targetView: "admin-access" },
  ],
  operatorBoundary: { actor: "platform", canManageAdministrators: true, role: "platform_admin" },
  permissionPackages: [
    { allowedCapabilityCount: 1, blockedCapabilityCount: 1, dataScopes: [{ dataDomain: "support", region: "us-east" }], latestApplicationId: "ppa-1", status: "ready", templateId: "support-ticket-triage", templateName: "Support ticket triage" },
  ],
  scopeTenants: [root, tenant],
  tenant,
  workspaces: [{ assignmentCount: 1, callerCount: 1, targetCount: 1, workspaceId: "ws-support" }],
};

test("tenant permission center presenter summarizes ready tenant governance", () => {
  const vm = buildTenantPermissionCenterViewModel(center, { selectedWorkspaceId: "ws-support" });

  assert.equal(vm.tenantName, "Customer Service");
  assert.equal(vm.tenantPath, "Group HQ / Customer Service");
  assert.equal(vm.status, "ready");
  assert.equal(vm.metric.allowedCapabilities, 1);
  assert.equal(vm.metric.blockedCapabilities, 1);
  assert.equal(vm.metric.administrators, 1);
  assert.deepEqual(vm.dataScopeLabels, ["support / us-east"]);
  assert.deepEqual(vm.primaryActions.map((action) => action.code), ["start_permission_change", "open_access_profile", "manage_administrators"]);
});

test("tenant permission center presenter hides admin management for scoped admins", () => {
  const vm = buildTenantPermissionCenterViewModel({
    ...center,
    nextActions: center.nextActions.filter((action) => action.code !== "manage_administrators"),
    operatorBoundary: { actor: "tenant-admin", canManageAdministrators: false, role: "tenant_admin", tenantId: tenant.id, workspaceId: "ws-support" },
  });

  assert.equal(vm.operatorBoundaryLabel, "tenant_admin / Customer Service / ws-support");
  assert.equal(vm.canManageAdministrators, false);
  assert.equal(tenantPermissionCenterActionTarget(vm.primaryActions, "manage_administrators"), "");
});

test("tenant permission center presenter marks empty tenant as setup needed", () => {
  const vm = buildTenantPermissionCenterViewModel({
    ...center,
    administrators: [],
    capabilities: [],
    nextActions: [{ code: "complete_setup", targetView: "getting-started" }],
    permissionPackages: [],
    workspaces: [],
  });

  assert.equal(vm.status, "blocked");
  assert.deepEqual(vm.emptyReasons, ["tenantCenter.empty.noWorkspaces", "tenantCenter.empty.noCapabilities"]);
  assert.equal(tenantPermissionCenterActionTarget(vm.primaryActions, "complete_setup"), "getting-started");
});
```

- [ ] **Step 4: Run presenter red test**

```bash
pnpm --dir frontend exec node --test tests/tenantPermissionCenter.test.mjs
```

Expected: FAIL because `frontend/src/tenantPermissionCenter.ts` does not exist.

- [ ] **Step 5: Implement presenter**

Create `frontend/src/tenantPermissionCenter.ts`:

```ts
import type {
  DataScope,
  Tenant,
  TenantPermissionCenterNextAction,
  TenantPermissionCenterResponse,
  TenantPermissionCenterStatus,
} from "./types";

export interface TenantPermissionCenterViewModel {
  canManageAdministrators: boolean;
  dataScopeLabels: string[];
  emptyReasons: string[];
  metric: {
    administrators: number;
    allowedCapabilities: number;
    blockedCapabilities: number;
    packages: number;
    workspaces: number;
  };
  operatorBoundaryLabel: string;
  primaryActions: TenantPermissionCenterNextAction[];
  selectedWorkspaceId: string;
  status: TenantPermissionCenterStatus;
  tenantName: string;
  tenantPath: string;
}

export function buildTenantPermissionCenterViewModel(
  center: TenantPermissionCenterResponse,
  options: { selectedWorkspaceId?: string } = {},
): TenantPermissionCenterViewModel {
  const selectedWorkspaceId = options.selectedWorkspaceId || center.workspaces[0]?.workspaceId || "";
  const allowedCapabilities = center.capabilities.filter((capability) => capability.effect === "allow").length;
  const blockedCapabilities = center.capabilities.filter((capability) => capability.effect === "deny").length;
  const emptyReasons = tenantPermissionCenterEmptyReasons(center);
  return {
    canManageAdministrators: center.operatorBoundary.canManageAdministrators,
    dataScopeLabels: uniqueDataScopeLabels(center.permissionPackages.flatMap((item) => item.dataScopes ?? [])),
    emptyReasons,
    metric: {
      administrators: center.administrators.length,
      allowedCapabilities,
      blockedCapabilities,
      packages: center.permissionPackages.length,
      workspaces: center.workspaces.length,
    },
    operatorBoundaryLabel: operatorBoundaryLabel(center),
    primaryActions: center.nextActions,
    selectedWorkspaceId,
    status: emptyReasons.length > 0 ? "blocked" : strongestStatus(center.permissionPackages),
    tenantName: center.tenant.name || center.tenant.id,
    tenantPath: tenantPathLabel(center.scopeTenants, center.tenant),
  };
}

export function tenantPermissionCenterActionTarget(actions: TenantPermissionCenterNextAction[], code: string) {
  return actions.find((action) => action.code === code)?.targetView ?? "";
}

function tenantPathLabel(scopeTenants: Tenant[], tenant: Tenant) {
  const tenantById = new Map(scopeTenants.map((row) => [row.id, row]));
  const path: Tenant[] = [];
  let cursor: Tenant | undefined = tenant;
  while (cursor) {
    path.unshift(cursor);
    cursor = cursor.parentTenantId ? tenantById.get(cursor.parentTenantId) : undefined;
  }
  return path.map((row) => row.name || row.id).join(" / ");
}

function operatorBoundaryLabel(center: TenantPermissionCenterResponse) {
  const tenantName = center.operatorBoundary.tenantId
    ? center.scopeTenants.find((tenant) => tenant.id === center.operatorBoundary.tenantId)?.name || center.operatorBoundary.tenantId
    : "All tenants";
  const workspace = center.operatorBoundary.workspaceId || "All workspaces";
  return `${center.operatorBoundary.role} / ${tenantName} / ${workspace}`;
}

function strongestStatus(packages: TenantPermissionCenterResponse["permissionPackages"]) {
  if (packages.some((item) => item.status === "blocked")) return "blocked";
  if (packages.some((item) => item.status === "needs_review")) return "needs_review";
  return "ready";
}

function tenantPermissionCenterEmptyReasons(center: TenantPermissionCenterResponse) {
  const reasons: string[] = [];
  if (center.workspaces.length === 0) reasons.push("tenantCenter.empty.noWorkspaces");
  if (center.capabilities.length === 0) reasons.push("tenantCenter.empty.noCapabilities");
  return reasons;
}

function uniqueDataScopeLabels(scopes: DataScope[]) {
  const labels = scopes.map(dataScopeLabel).filter(Boolean);
  return Array.from(new Set(labels));
}

function dataScopeLabel(scope: DataScope) {
  return [scope.dataDomain, scope.dataset, scope.region, scope.classification].filter(Boolean).join(" / ");
}
```

- [ ] **Step 6: Run presenter test**

```bash
pnpm --dir frontend exec node --test tests/tenantPermissionCenter.test.mjs
```

Expected: PASS.

- [ ] **Step 7: Commit frontend data layer**

```bash
git add frontend/src/types.ts frontend/src/api.ts frontend/src/tenantPermissionCenter.ts frontend/tests/tenantPermissionCenter.test.mjs
git commit -m "feat: add tenant permission center presenter"
```

---

### Task 4: Tenant Organization Workspace Integration

**Files:**
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/components/TenantOrganizationView.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/i18n.test.mjs`
- Modify: `frontend/tests/styleTheme.test.mjs`

- [ ] **Step 1: Add i18n keys**

Add English keys:

```ts
"tenantCenter.adminBoundary": "Administrator boundary",
"tenantCenter.adminBoundaryDetail": "Shows who can operate this tenant. Keys and secret material are never shown.",
"tenantCenter.capabilities": "Current capabilities",
"tenantCenter.dataScopes": "Data scope",
"tenantCenter.empty.noCapabilities": "No capabilities are assigned to this tenant yet.",
"tenantCenter.empty.noWorkspaces": "No workspace is registered under this tenant yet.",
"tenantCenter.manageAdmins": "Manage administrators",
"tenantCenter.openAccessProfile": "View access profile",
"tenantCenter.operatorBoundary": "Your operating boundary",
"tenantCenter.permissionPackages": "Permission packages",
"tenantCenter.snapshot": "Permission snapshot",
"tenantCenter.startPermissionChange": "Start permission change",
"tenantCenter.status.blocked": "Needs setup",
"tenantCenter.status.needs_review": "Needs review",
"tenantCenter.status.ready": "Ready",
"tenantCenter.workspaces": "Workspaces and access objects",
```

Add Simplified Chinese keys:

```ts
"tenantCenter.adminBoundary": "管理员边界",
"tenantCenter.adminBoundaryDetail": "展示谁可以操作该租户。密钥和敏感材料不会展示。",
"tenantCenter.capabilities": "当前能力",
"tenantCenter.dataScopes": "数据范围",
"tenantCenter.empty.noCapabilities": "该租户还没有分配能力。",
"tenantCenter.empty.noWorkspaces": "该租户下还没有注册工作区。",
"tenantCenter.manageAdmins": "管理管理员",
"tenantCenter.openAccessProfile": "查看权限画像",
"tenantCenter.operatorBoundary": "你的操作边界",
"tenantCenter.permissionPackages": "权限包",
"tenantCenter.snapshot": "权限快照",
"tenantCenter.startPermissionChange": "发起权限变更",
"tenantCenter.status.blocked": "需要配置",
"tenantCenter.status.needs_review": "需要复核",
"tenantCenter.status.ready": "可用",
"tenantCenter.workspaces": "工作区与访问对象",
```

- [ ] **Step 2: Add i18n test coverage**

In `frontend/tests/i18n.test.mjs`, add these keys to an existing bilingual tenant/administrator copy test or create:

```js
test("tenant permission center copy is bilingual", () => {
  const keys = [
    "tenantCenter.adminBoundary",
    "tenantCenter.adminBoundaryDetail",
    "tenantCenter.capabilities",
    "tenantCenter.dataScopes",
    "tenantCenter.empty.noCapabilities",
    "tenantCenter.empty.noWorkspaces",
    "tenantCenter.manageAdmins",
    "tenantCenter.openAccessProfile",
    "tenantCenter.operatorBoundary",
    "tenantCenter.permissionPackages",
    "tenantCenter.snapshot",
    "tenantCenter.startPermissionChange",
    "tenantCenter.status.blocked",
    "tenantCenter.status.needs_review",
    "tenantCenter.status.ready",
    "tenantCenter.workspaces",
  ];
  for (const language of ["en", "zh-CN"]) {
    const t = createTranslator(language);
    for (const key of keys) {
      assert.notEqual(t(key), key, `${language} missing ${key}`);
    }
  }
});
```

- [ ] **Step 3: Wire API loading in ConsoleController**

In `ConsoleController.tsx`, import `fetchTenantPermissionCenter` and `TenantPermissionCenterResponse`.

Add state near other tenant workspace state:

```ts
const [tenantOrganizationSelectedTenantId, setTenantOrganizationSelectedTenantId] = useState("");
const [tenantPermissionCenter, setTenantPermissionCenter] = useState<TenantPermissionCenterResponse | null>(null);
const [tenantPermissionCenterLoading, setTenantPermissionCenterLoading] = useState(false);
const [tenantPermissionCenterError, setTenantPermissionCenterError] = useState("");

const tenantOrganizationEffectiveTenantId = tenantOrganizationSelectedTenantId || scope.tenantId || tenants[0]?.id || "";
```

Add selection normalization and API loading effects:

```ts
useEffect(() => {
  if (tenants.length === 0) {
    setTenantOrganizationSelectedTenantId("");
    return;
  }
  if (tenantOrganizationSelectedTenantId && tenants.some((tenant) => tenant.id === tenantOrganizationSelectedTenantId)) {
    return;
  }
  const scopedTenant = scope.tenantId && tenants.some((tenant) => tenant.id === scope.tenantId)
    ? scope.tenantId
    : "";
  setTenantOrganizationSelectedTenantId(scopedTenant || tenants[0].id);
}, [scope.tenantId, tenantOrganizationSelectedTenantId, tenants]);
```

```ts
useEffect(() => {
  if (!consoleAccessReady || activeNav !== "tenants" || tenants.length === 0) return;
  if (!tenantOrganizationEffectiveTenantId) return;
  const controller = new AbortController();
  setTenantPermissionCenterLoading(true);
  setTenantPermissionCenterError("");
  fetchTenantPermissionCenter(tenantOrganizationEffectiveTenantId, undefined, adminKey, controller.signal)
    .then(setTenantPermissionCenter)
    .catch((error) => {
      if (error instanceof Error && error.name === "AbortError") return;
      setTenantPermissionCenter(null);
      setTenantPermissionCenterError(error instanceof Error ? error.message : createTranslator(language)("error.loadTenantPermissionCenter"));
    })
    .finally(() => setTenantPermissionCenterLoading(false));
  return () => controller.abort();
}, [activeNav, adminKey, consoleAccessReady, language, tenantOrganizationEffectiveTenantId, tenants.length]);
```

- [ ] **Step 4: Update TenantOrganizationView props**

Extend `TenantOrganizationViewProps` with these props:

```ts
import type { TenantPermissionCenterResponse } from "../types";

interface TenantOrganizationViewProps {
  onSelectedTenantIdChange: (tenantId: string) => void;
  selectedTenantId: string;
  tenantPermissionCenter: TenantPermissionCenterResponse | null;
  tenantPermissionCenterError: string;
  tenantPermissionCenterLoading: boolean;
}
```

Import and use the presenter:

```ts
import { buildTenantPermissionCenterViewModel, tenantPermissionCenterActionTarget } from "../tenantPermissionCenter";
```

Inside the component:

```ts
// Remove the internal selectedTenantId useState; this view is controlled by ConsoleController.
// Delete: const [selectedTenantId, setSelectedTenantId] = useState("");

const centerViewModel = tenantPermissionCenter
  ? buildTenantPermissionCenterViewModel(tenantPermissionCenter, { selectedWorkspaceId: context?.workspaceId })
  : null;
```

Update the selection synchronization effect and tree row click handler:

```ts
useEffect(() => {
  if (model.selectedTenantId && model.selectedTenantId !== selectedTenantId) {
    onSelectedTenantIdChange(model.selectedTenantId);
  }
}, [model.selectedTenantId, onSelectedTenantIdChange, selectedTenantId]);
```

```tsx
onClick={() => onSelectedTenantIdChange(node.tenant.id)}
```

- [ ] **Step 5: Render permission center sections**

In `TenantOrganizationView.tsx`, after the existing `tenant-org-actions`, render:

```tsx
{tenantPermissionCenterLoading ? (
  <div className="tenant-center-status" role="status">{t("status.loadingConsole")}</div>
) : null}
{tenantPermissionCenterError ? (
  <div className="strip-error">{tenantPermissionCenterError}</div>
) : null}
{centerViewModel ? (
  <section className="tenant-center-panel" aria-label={t("tenantCenter.snapshot")}>
    <div className="tenant-center-header">
      <div>
        <span className="section-kicker">{t("tenantCenter.operatorBoundary")}</span>
        <strong>{centerViewModel.operatorBoundaryLabel}</strong>
      </div>
      <Badge tone={centerViewModel.status === "ready" ? "success" : centerViewModel.status === "needs_review" ? "warning" : "neutral"}>
        {t(`tenantCenter.status.${centerViewModel.status}`)}
      </Badge>
    </div>
    <div className="tenant-center-metrics">
      <TenantOrgMetric icon={<ShieldCheck size={16} />} label={t("tenantCenter.permissionPackages")} value={String(centerViewModel.metric.packages)} detail={t("tenantCenter.snapshot")} />
      <TenantOrgMetric icon={<Network size={16} />} label={t("tenantCenter.capabilities")} value={`${centerViewModel.metric.allowedCapabilities}/${centerViewModel.metric.blockedCapabilities}`} detail={tx(t, "tenantOrg.permissionDetail", { allowed: centerViewModel.metric.allowedCapabilities, denied: centerViewModel.metric.blockedCapabilities })} />
      <TenantOrgMetric icon={<UserRoundCheck size={16} />} label={t("tenantCenter.adminBoundary")} value={String(centerViewModel.metric.administrators)} detail={centerViewModel.canManageAdministrators ? t("tenantCenter.manageAdmins") : t("adminAccess.readOnly")} />
    </div>
    <div className="tenant-center-actions">
      <button className="primary-button" type="button" onClick={openPermissionModal}>
        <ShieldCheck size={15} />
        {t("tenantCenter.startPermissionChange")}
      </button>
      <button className="secondary-button" type="button" onClick={() => onOpenAccessProfile(context)}>
        <LockKeyhole size={15} />
        {t("tenantCenter.openAccessProfile")}
      </button>
      {tenantPermissionCenterActionTarget(centerViewModel.primaryActions, "manage_administrators") ? (
        <a className="secondary-button" href="#admin-access">
          <UserRoundCheck size={15} />
          {t("tenantCenter.manageAdmins")}
        </a>
      ) : null}
    </div>
    {centerViewModel.emptyReasons.length > 0 ? (
      <div className="tenant-center-empty-reasons">
        {centerViewModel.emptyReasons.map((key) => <span key={key}>{t(key)}</span>)}
      </div>
    ) : null}
  </section>
) : null}
```

- [ ] **Step 6: Pass props from ConsoleController**

Where `TenantOrganizationView` is constructed, pass:

```tsx
onSelectedTenantIdChange={setTenantOrganizationSelectedTenantId}
selectedTenantId={tenantOrganizationEffectiveTenantId}
tenantPermissionCenter={tenantPermissionCenter}
tenantPermissionCenterError={tenantPermissionCenterError}
tenantPermissionCenterLoading={tenantPermissionCenterLoading}
```

- [ ] **Step 7: Add source guard test**

In `frontend/tests/styleTheme.test.mjs`, extend the tenant organization test:

```js
assert.match(tenantOrganizationView, /buildTenantPermissionCenterViewModel/);
assert.match(tenantOrganizationView, /tenantCenter\.snapshot/);
assert.match(tenantOrganizationView, /tenantCenter\.adminBoundary/);
assert.match(app, /selectedTenantId=\\{tenantOrganizationEffectiveTenantId\\}/);
assert.match(tenantOrganizationView, /href="#admin-access"/);
assert.match(app, /fetchTenantPermissionCenter/);
```

- [ ] **Step 8: Run focused frontend tests**

```bash
pnpm --dir frontend exec node --test tests/tenantPermissionCenter.test.mjs tests/i18n.test.mjs tests/styleTheme.test.mjs
pnpm --dir frontend build
```

Expected: PASS.

- [ ] **Step 9: Commit tenant workspace integration**

```bash
git add frontend/src/ConsoleController.tsx frontend/src/components/TenantOrganizationView.tsx frontend/src/i18n.ts frontend/tests/i18n.test.mjs frontend/tests/styleTheme.test.mjs
git commit -m "feat: surface tenant permission center"
```

---

### Task 5: Scenario, Docs, And Release Gates

**Files:**
- Create: `scripts/scenario-tenant-permission-center.sh`
- Modify: `Makefile`
- Modify: `tests/makefile_targets_test.sh`
- Modify: `README.md`
- Modify: `CHANGELOG.md`
- Modify: `docs/superpowers/plans/2026-06-16-tenant-permission-center.md`

- [x] **Step 1: Create scenario script**

Create `scripts/scenario-tenant-permission-center.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${BASE_URL:-http://127.0.0.1:9196}"
MOCK_MCP_HOST="${MOCK_MCP_HOST:-127.0.0.1}"
MOCK_MCP_PORT="${MOCK_MCP_PORT:-8796}"
MCP_ENDPOINT="http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/mcp"
RUN_ID="tenant-permission-center-$(date +%Y%m%d%H%M%S)"
PLATFORM_KEY="platform-key-${RUN_ID}"
TENANT_ROOT="tenant-root-${RUN_ID}"
TENANT_CHILD="tenant-child-${RUN_ID}"
WORKSPACE_ID="ws-support-${RUN_ID}"
PIDS=()

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "${pid}" >/dev/null 2>&1 || true
  done
  wait >/dev/null 2>&1 || true
}
trap cleanup EXIT

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing dependency: $1" >&2
    exit 1
  fi
}

wait_url() {
  local url="$1"
  local label="$2"
  for _ in $(seq 1 80); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return
    fi
    sleep 0.1
  done
  echo "${label} did not become ready" >&2
  exit 1
}

need curl
need jq
need python3

echo "AgentHarbor tenant permission center scenario"
echo "BASE_URL=${BASE_URL}"
echo "MCP_ENDPOINT=${MCP_ENDPOINT}"
echo "RUN_ID=${RUN_ID}"

python3 "${ROOT_DIR}/scripts/mock-mcp-server.py" --host "${MOCK_MCP_HOST}" --port "${MOCK_MCP_PORT}" >/tmp/agent-harbor-tenant-center-mcp-${RUN_ID}.log 2>&1 &
PIDS+=("$!")
wait_url "http://${MOCK_MCP_HOST}:${MOCK_MCP_PORT}/healthz" "mock MCP server"

AGENT_HARBOR_ADMIN_IDENTITIES="platform=${PLATFORM_KEY}|role=platform_admin" \
AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS=true \
AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN=true \
AGENT_HARBOR_SESSION_SECRET="scenario-session-${RUN_ID}" \
"${ROOT_DIR}/bin/agent-harbor" serve --addr "${BASE_URL#http://}" >"/tmp/agent-harbor-tenant-center-${RUN_ID}.log" 2>&1 &
PIDS+=("$!")
wait_url "${BASE_URL}/healthz" "AgentHarbor API"

curl_json() {
  local method="$1"
  local path="$2"
  local key="$3"
  local body="${4:-}"
  if [[ -n "${body}" ]]; then
    curl -fsS -X "${method}" "${BASE_URL}${path}" -H "Content-Type: application/json" -H "X-Admin-Key: ${key}" --data "${body}"
  else
    curl -fsS -X "${method}" "${BASE_URL}${path}" -H "X-Admin-Key: ${key}"
  fi
}

curl_json POST /api/v1/tenants "${PLATFORM_KEY}" "{\"id\":\"${TENANT_ROOT}\",\"name\":\"Platform Operations\"}" >/dev/null
curl_json POST /api/v1/tenants "${PLATFORM_KEY}" "{\"id\":\"${TENANT_CHILD}\",\"parentTenantId\":\"${TENANT_ROOT}\",\"name\":\"Customer Service\"}" >/dev/null
CALLER=$(curl_json POST /api/v1/agents "${PLATFORM_KEY}" "{\"tenantId\":\"${TENANT_CHILD}\",\"workspaceId\":\"${WORKSPACE_ID}\",\"name\":\"Support Assistant\",\"channelType\":\"local\",\"status\":\"active\"}" | jq -r '.data.id')
TARGET=$(curl_json POST /api/v1/agents "${PLATFORM_KEY}" "{\"tenantId\":\"${TENANT_CHILD}\",\"workspaceId\":\"${WORKSPACE_ID}\",\"name\":\"Ticket Tool Service\",\"channelType\":\"mcp\",\"channelConfig\":{\"endpoint\":\"${MCP_ENDPOINT}\"},\"status\":\"active\"}" | jq -r '.data.id')
CAP=$(curl_json POST "/api/v1/targets/${TARGET}/capabilities:refresh" "${PLATFORM_KEY}" '{}' | jq -r '.data[0].id')
ENTITLEMENT=$(curl_json POST /api/v1/tenant-entitlements "${PLATFORM_KEY}" "{\"tenantId\":\"${TENANT_CHILD}\",\"targetId\":\"${TARGET}\",\"capabilityId\":\"${CAP}\",\"effect\":\"allow\",\"status\":\"enabled\",\"dataScopes\":[{\"dataDomain\":\"support\",\"dataset\":\"tickets\",\"region\":\"us-east\"}]}" | jq -r '.data.id')
WORKSPACE_ASSIGNMENT=$(curl_json POST /api/v1/workspace-assignments "${PLATFORM_KEY}" "{\"tenantEntitlementId\":\"${ENTITLEMENT}\",\"workspaceId\":\"${WORKSPACE_ID}\",\"effect\":\"allow\",\"status\":\"enabled\",\"dataScopes\":[{\"dataDomain\":\"support\",\"dataset\":\"tickets\",\"region\":\"us-east\"}]}" | jq -r '.data.id')
curl_json POST /api/v1/instance-assignments "${PLATFORM_KEY}" "{\"workspaceAssignmentId\":\"${WORKSPACE_ASSIGNMENT}\",\"callerInstanceId\":\"${CALLER}\",\"effect\":\"allow\",\"status\":\"enabled\",\"dataScopes\":[{\"dataDomain\":\"support\",\"dataset\":\"tickets\",\"region\":\"us-east\"}]}" >/dev/null
ADMIN_CREATE=$(curl_json POST /api/v1/admin-identities "${PLATFORM_KEY}" "{\"actor\":\"tenant-admin-${RUN_ID}\",\"displayName\":\"Tenant Admin\",\"role\":\"tenant_admin\",\"tenantId\":\"${TENANT_CHILD}\",\"workspaceId\":\"${WORKSPACE_ID}\"}")
TENANT_ADMIN_KEY=$(printf '%s' "${ADMIN_CREATE}" | jq -r '.data.key')

CENTER=$(curl_json GET "/api/v1/tenants/${TENANT_CHILD}/permission-center" "${PLATFORM_KEY}")
printf '%s' "${CENTER}" | jq -e '.data.operatorBoundary.canManageAdministrators == true' >/dev/null
printf '%s' "${CENTER}" | jq -e '.data.administrators | length >= 1' >/dev/null
printf '%s' "${CENTER}" | jq -e '.data.workspaces[0].workspaceId == "'"${WORKSPACE_ID}"'"' >/dev/null
printf '%s' "${CENTER}" | jq -e '.data.capabilities | length >= 1' >/dev/null
printf '%s' "${CENTER}" | jq -e '.data.capabilities[0].dataScopes[0].dataDomain == "support"' >/dev/null

SCOPED_CENTER=$(curl_json GET "/api/v1/tenants/${TENANT_CHILD}/permission-center" "${TENANT_ADMIN_KEY}")
printf '%s' "${SCOPED_CENTER}" | jq -e '.data.operatorBoundary.canManageAdministrators == false' >/dev/null
printf '%s' "${SCOPED_CENTER}" | jq -e '.data.operatorBoundary.tenantId == "'"${TENANT_CHILD}"'"' >/dev/null
printf '%s' "${SCOPED_CENTER}" | jq -e '.data.operatorBoundary.workspaceId == "'"${WORKSPACE_ID}"'"' >/dev/null

if curl -fsS -X GET "${BASE_URL}/api/v1/tenants/${TENANT_ROOT}/permission-center" -H "X-Admin-Key: ${TENANT_ADMIN_KEY}" >/tmp/tenant-center-widen.json 2>/dev/null; then
  echo "tenant admin unexpectedly fetched parent tenant center" >&2
  exit 1
fi

echo "tenant permission center scenario complete"
```

This scenario follows the existing release-script convention and requires `curl` and `jq`, matching the other JSON-validating scenario scripts in `scripts/`.

- [x] **Step 2: Wire Makefile target**

Add:

```make
.PHONY: scenario-tenant-permission-center
scenario-tenant-permission-center: build
	bash scripts/scenario-tenant-permission-center.sh
```

Add `scenario-tenant-permission-center` to `release-check` after `scenario-admin-access-management`.

- [x] **Step 3: Update makefile target test**

In `tests/makefile_targets_test.sh`, add `scripts/scenario-tenant-permission-center.sh` to the `bash -n` list and assert the Makefile target exists using the local pattern already used for other scenarios.

- [x] **Step 4: Update README and CHANGELOG**

In `README.md`, add a short bilingual paragraph near the tenant/admin sections:

```md
The **Tenant Permission Center** turns each registered tenant into a governance workspace: platform administrators can review assigned administrators, workspaces, permission packages, allowed and blocked capabilities, data scopes, and safe next actions from one tenant detail page. Tenant administrators see the same page bounded to their assigned tenant/workspace and cannot manage administrator identities.

**租户权限中心** 会把每个已注册租户变成一个治理工作区：平台管理员可以在租户详情页查看负责管理员、工作区、权限包、允许和禁止能力、数据范围以及下一步动作。租户管理员只能看到自己被分配的租户/工作区范围，不能管理管理员身份。
```

In `CHANGELOG.md`, add an Unreleased bullet:

```md
- Added a tenant permission center projection and console summary for tenant-scoped administrators, permissions, capabilities, data scopes, and next actions.
```

- [x] **Step 5: Run full gates**

```bash
go test ./internal/httpapi -run 'TenantPermissionCenter' -count=1
pnpm --dir frontend test
pnpm --dir frontend build
bash -n scripts/scenario-tenant-permission-center.sh
make scenario-tenant-permission-center
make check
make release-check
```

Expected: all PASS.

- [x] **Step 6: Browser smoke**

Start the app:

```bash
make demo
```

Open `http://127.0.0.1:5174/#tenants` and verify:

- Tenant detail shows permission snapshot, administrator boundary, workspaces, capabilities, and data scopes.
- Primary action opens the tenant-scoped permission-change modal.
- Access Profile action preserves tenant/workspace context.
- Admin Boundaries action is visible for platform admin and hidden for tenant admin.
- Raw tenant IDs and assignment IDs are only in advanced details, not primary cards.

Actual verification:

- `go test ./internal/httpapi -run 'TenantPermissionCenter' -count=1` passed after adding a regression check that scoped tenant-admin responses return `administrators: []` instead of `null`.
- `pnpm --dir frontend exec node --test tests/tenantPermissionCenter.test.mjs tests/styleTheme.test.mjs tests/i18n.test.mjs` passed with presenter coverage for assigned capabilities, data scopes, and nullable array compatibility.
- `pnpm --dir frontend build` passed.
- `make scenario-tenant-permission-center` passed.
- `make check` passed with 238 frontend tests.
- `make release-check` passed.
- Browser smoke on `http://127.0.0.1:5177/#tenants` confirmed platform-admin tenant detail shows permission snapshot, administrator boundary, workspaces, assigned capability detail, and `support / tickets / us-east` data scope; tenant-scoped permission-change modal preserved tenant, workspace, caller, target, and access-object context; access-profile handoff preserved tenant/workspace context.
- Browser smoke on a strict-auth stack at `http://127.0.0.1:5178/#tenants` confirmed `tenant-key` loads only the scoped tenant/workspace, does not blank when administrators are hidden, hides the `管理管理员` action, and still opens the permission-change modal with scoped tenant/workspace/caller/target context.

- [x] **Step 7: Commit docs and scenario**

```bash
git add scripts/scenario-tenant-permission-center.sh Makefile tests/makefile_targets_test.sh README.md CHANGELOG.md docs/superpowers/plans/2026-06-16-tenant-permission-center.md
git commit -m "test: gate tenant permission center"
```

- [ ] **Step 8: Push and open PR**

```bash
git push -u origin codex/tenant-organization-permission-center
gh pr create --base main --head codex/tenant-organization-permission-center --title "Add tenant permission center" --body-file /tmp/agent-harbor-tenant-permission-center-pr.md
```

PR body must include:

- Summary.
- Security guardrails.
- Verification results.
- Browser smoke notes.
- Non-goals: no SSO, no direct grant editing, no My Access portal.
