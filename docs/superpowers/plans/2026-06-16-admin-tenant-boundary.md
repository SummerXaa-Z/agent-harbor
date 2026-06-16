# Admin Tenant Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce authenticated administrator tenant/workspace boundaries across the management control plane.

**Architecture:** Extend configured admin identities with role and scope, persist that principal in the signed console session, and derive every management query/mutation scope from the authenticated principal plus requested parameters. Existing store `ManagementScope` filtering stays the data-access primitive; HTTP and management MCP handlers become responsible for intersecting requested scope with the admin's allowed scope before reads or writes.

**Tech Stack:** Go HTTP API, in-memory and PostgreSQL store scope filters, React/TypeScript console, Node test runner, shell release scenarios.

---

## File Map

- `internal/httpapi/admin_scope.go`: new helper types/functions for admin principals, scope intersection, and mutation authorization.
- `internal/httpapi/auth.go`: session payload/response includes admin role and scope.
- `internal/httpapi/server.go`: `AdminIdentity` gains role/scope; `requireAdmin` stores a full principal; management routes use effective scopes and mutation guards.
- `internal/httpapi/management_mcp.go`: management MCP tools apply the authenticated principal scope before repo reads/writes.
- `internal/app/app.go`: `AGENT_HARBOR_ADMIN_IDENTITIES` parses backward-compatible and scoped identity formats.
- `internal/app/app_test.go`: env parsing regression tests.
- `internal/httpapi/server_test.go`: authentication, scope intersection, route, mutation, and MCP regression tests.
- `frontend/src/types.ts`: `ConsoleSession` includes role and scope.
- `frontend/src/ConsoleController.tsx`: session/scope chip displays authenticated admin scope and prevents widening a scoped admin context.
- `frontend/src/i18n.ts`: bilingual labels for admin role and scope.
- `frontend/tests/i18n.test.mjs`, `frontend/tests/permissionFlowLayout.test.mjs`: source guards for visible scope context.
- `scripts/scenario-admin-tenant-boundary.sh`: release scenario proving cross-tenant access is blocked.
- `Makefile`, `tests/makefile_targets_test.sh`, `README.md`, `CHANGELOG.md`: wire and document the new gate.

---

### Task 1: Red Tests For Scoped Admin Identities

**Files:**
- Modify: `internal/app/app_test.go`
- Modify: `internal/httpapi/server_test.go`

- [x] **Step 1: Add env parser test**

Add this test near `TestApprovalReviewersFromEnvParsesScopedRules` in `internal/app/app_test.go`:

```go
func TestAdminIdentitiesFromEnvParsesScopedRules(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", "platform=platform-key|role=platform_admin,tenant-east=east-key|role=tenant_admin|tenant=tenant-east|workspace=ws-support,legacy=legacy-key")

	identities, err := adminIdentitiesFromEnv()
	if err != nil {
		t.Fatalf("parse admin identities: %v", err)
	}
	if len(identities) != 3 {
		t.Fatalf("expected three identities, got %#v", identities)
	}
	if identities[0].Actor != "platform" || identities[0].Key != "platform-key" || identities[0].Role != "platform_admin" || identities[0].TenantID != "" || identities[0].WorkspaceID != "" {
		t.Fatalf("unexpected platform identity: %#v", identities[0])
	}
	if identities[1].Actor != "tenant-east" || identities[1].Key != "east-key" || identities[1].Role != "tenant_admin" || identities[1].TenantID != "tenant-east" || identities[1].WorkspaceID != "ws-support" {
		t.Fatalf("unexpected scoped identity: %#v", identities[1])
	}
	if identities[2].Actor != "legacy" || identities[2].Key != "legacy-key" || identities[2].Role != "platform_admin" || identities[2].TenantID != "" || identities[2].WorkspaceID != "" {
		t.Fatalf("legacy identity should default to platform admin: %#v", identities[2])
	}
}
```

- [x] **Step 2: Add auth session test**

Add this test after `TestConsoleAuthSessionSupportsNamedAdminIdentities` in `internal/httpapi/server_test.go`:

```go
func TestConsoleAuthSessionReportsScopedAdminIdentity(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "east-key"}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login should accept scoped admin key, got %d body=%s", login.Code, login.Body.String())
	}
	session := decodeData[map[string]any](t, login)
	if session["actor"] != "east-admin" || session["role"] != "tenant_admin" || session["tenantId"] != "tenant-east" || session["workspaceId"] != "ws-support" {
		t.Fatalf("unexpected scoped session response: %#v", session)
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %#v", cookies)
	}
	me := decodeData[map[string]any](t, requestWithCookie(t, router, http.MethodGet, "/api/v1/auth/session", nil, cookies[0]))
	if me["actor"] != "east-admin" || me["role"] != "tenant_admin" || me["tenantId"] != "tenant-east" || me["workspaceId"] != "ws-support" {
		t.Fatalf("session endpoint should return scoped principal, got %#v", me)
	}
}
```

- [x] **Step 3: Run red tests**

Run:

```bash
go test ./internal/app ./internal/httpapi -run 'TestAdminIdentitiesFromEnvParsesScopedRules|TestConsoleAuthSessionReportsScopedAdminIdentity' -count=1
```

Expected: FAIL because `AdminIdentity` and `consoleSessionResponse` do not expose `Role`, `TenantID`, or `WorkspaceID`.

### Task 2: Implement Admin Principal And Session Scope

**Files:**
- Create: `internal/httpapi/admin_scope.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/auth.go`
- Modify: `internal/app/app.go`

- [x] **Step 1: Add admin principal helpers**

Create `internal/httpapi/admin_scope.go`:

```go
package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	adminRolePlatformAdmin   = "platform_admin"
	adminRoleTenantAdmin     = "tenant_admin"
	adminRoleSecurityReviewer = "security_reviewer"
)

type adminPrincipal struct {
	Actor       string
	Role        string
	TenantID    string
	WorkspaceID string
}

func normalizeAdminIdentity(identity AdminIdentity) (AdminIdentity, bool) {
	normalized := AdminIdentity{
		Actor:       strings.TrimSpace(identity.Actor),
		Key:         strings.TrimSpace(identity.Key),
		Role:        normalizeAdminRole(identity.Role),
		TenantID:    strings.TrimSpace(identity.TenantID),
		WorkspaceID: strings.TrimSpace(identity.WorkspaceID),
	}
	return normalized, normalized.Actor != "" && normalized.Key != ""
}

func normalizeAdminRole(role string) string {
	switch strings.TrimSpace(role) {
	case "", adminRolePlatformAdmin:
		return adminRolePlatformAdmin
	case adminRoleTenantAdmin:
		return adminRoleTenantAdmin
	case adminRoleSecurityReviewer:
		return adminRoleSecurityReviewer
	default:
		return strings.TrimSpace(role)
	}
}

func (identity AdminIdentity) principal() adminPrincipal {
	return adminPrincipal{
		Actor:       strings.TrimSpace(identity.Actor),
		Role:        normalizeAdminRole(identity.Role),
		TenantID:    strings.TrimSpace(identity.TenantID),
		WorkspaceID: strings.TrimSpace(identity.WorkspaceID),
	}
}

func platformAdminPrincipal(actor string) adminPrincipal {
	return adminPrincipal{Actor: strings.TrimSpace(actor), Role: adminRolePlatformAdmin}
}

func requestAdminPrincipal(r *http.Request) (adminPrincipal, bool) {
	principal, ok := r.Context().Value(adminActorContextKey{}).(adminPrincipal)
	if !ok || strings.TrimSpace(principal.Actor) == "" {
		return adminPrincipal{}, false
	}
	principal.Actor = strings.TrimSpace(principal.Actor)
	principal.Role = normalizeAdminRole(principal.Role)
	principal.TenantID = strings.TrimSpace(principal.TenantID)
	principal.WorkspaceID = strings.TrimSpace(principal.WorkspaceID)
	return principal, true
}

func (s *Server) effectiveManagementScope(ctx context.Context, requested store.ManagementScope, principal adminPrincipal) (store.ManagementScope, error) {
	requested.TenantID = strings.TrimSpace(requested.TenantID)
	requested.WorkspaceID = strings.TrimSpace(requested.WorkspaceID)
	principal.Role = normalizeAdminRole(principal.Role)
	principal.TenantID = strings.TrimSpace(principal.TenantID)
	principal.WorkspaceID = strings.TrimSpace(principal.WorkspaceID)
	if principal.Role == adminRolePlatformAdmin || principal.TenantID == "" {
		return requested, nil
	}
	tenantID, err := s.intersectAdminTenantScope(ctx, requested.TenantID, principal.TenantID)
	if err != nil {
		return store.ManagementScope{}, err
	}
	workspaceID, ok := intersectAdminWorkspaceScope(requested.WorkspaceID, principal.WorkspaceID)
	if !ok {
		return store.ManagementScope{}, domain.PermissionDenied("requested workspace is outside authenticated admin scope")
	}
	return store.ManagementScope{TenantID: tenantID, WorkspaceID: workspaceID}, nil
}

func (s *Server) intersectAdminTenantScope(ctx context.Context, requestedTenantID string, principalTenantID string) (string, error) {
	requestedTenantID = strings.TrimSpace(requestedTenantID)
	principalTenantID = strings.TrimSpace(principalTenantID)
	if principalTenantID == "" {
		return requestedTenantID, nil
	}
	if requestedTenantID == "" || requestedTenantID == principalTenantID {
		return principalTenantID, nil
	}
	principalSubtree, err := s.tenantSubtreeIDs(ctx, principalTenantID)
	if err != nil {
		return "", err
	}
	if _, ok := principalSubtree[requestedTenantID]; ok {
		return requestedTenantID, nil
	}
	requestedSubtree, err := s.tenantSubtreeIDs(ctx, requestedTenantID)
	if err != nil {
		return "", err
	}
	if _, ok := requestedSubtree[principalTenantID]; ok {
		return principalTenantID, nil
	}
	return "", domain.PermissionDenied("requested tenant is outside authenticated admin scope")
}

func (s *Server) tenantSubtreeIDs(ctx context.Context, tenantID string) (map[string]struct{}, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, nil
	}
	rows, err := s.repo.ListTenants(ctx, store.TenantFilter{TenantID: tenantID})
	if err != nil {
		return nil, err
	}
	result := map[string]struct{}{}
	if len(rows) == 0 {
		result[tenantID] = struct{}{}
		return result, nil
	}
	for _, row := range rows {
		result[row.ID] = struct{}{}
	}
	return result, nil
}

func intersectAdminWorkspaceScope(requestedWorkspaceID string, principalWorkspaceID string) (string, bool) {
	requestedWorkspaceID = strings.TrimSpace(requestedWorkspaceID)
	principalWorkspaceID = strings.TrimSpace(principalWorkspaceID)
	if principalWorkspaceID == "" {
		return requestedWorkspaceID, true
	}
	if requestedWorkspaceID == "" || requestedWorkspaceID == principalWorkspaceID {
		return principalWorkspaceID, true
	}
	return "", false
}
```

- [x] **Step 2: Extend `AdminIdentity` and option normalization**

In `internal/httpapi/server.go`, replace `AdminIdentity` and `WithAdminIdentities` normalization with:

```go
type AdminIdentity struct {
	Actor       string
	Key         string
	Role        string
	TenantID    string
	WorkspaceID string
}
```

and:

```go
func WithAdminIdentities(identities []AdminIdentity) Option {
	return func(s *Server) {
		s.adminIdentities = make([]AdminIdentity, 0, len(identities))
		for _, identity := range identities {
			normalized, ok := normalizeAdminIdentity(identity)
			if !ok {
				continue
			}
			s.adminIdentities = append(s.adminIdentities, normalized)
		}
	}
}
```

- [x] **Step 3: Store principal in request context**

Change `requireAdmin`, `adminActorForKey`, and `requestAuthenticatedAdminActor` in `internal/httpapi/server.go` to store `adminPrincipal` instead of only actor:

```go
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if principal, _, ok := s.developmentSession(); ok {
			ctx := context.WithValue(r.Context(), adminActorContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if s.adminKey == "" && len(s.adminIdentities) == 0 {
			writeError(w, domain.Unauthorized("admin authentication is required"))
			return
		}
		provided := r.Header.Get("X-Admin-Key")
		if principal, ok := s.adminPrincipalForKey(provided); ok {
			ctx := context.WithValue(r.Context(), adminActorContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if principal, _, ok := s.consoleSessionFromRequest(r); ok {
			ctx := context.WithValue(r.Context(), adminActorContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		writeError(w, domain.Unauthorized("missing or invalid admin key"))
	})
}

func (s *Server) adminPrincipalForKey(provided string) (adminPrincipal, bool) {
	for _, identity := range s.adminIdentities {
		if len(provided) == len(identity.Key) && subtle.ConstantTimeCompare([]byte(provided), []byte(identity.Key)) == 1 {
			return identity.principal(), true
		}
	}
	if s.adminKey != "" {
		if len(provided) != len(s.adminKey) || subtle.ConstantTimeCompare([]byte(provided), []byte(s.adminKey)) != 1 {
			return adminPrincipal{}, false
		}
		return platformAdminPrincipal("admin-key"), true
	}
	return adminPrincipal{}, false
}

func requestAuthenticatedAdminActor(r *http.Request) (string, bool) {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return "", false
	}
	return principal.Actor, true
}
```

- [x] **Step 4: Extend console session payload and response**

In `internal/httpapi/auth.go`, change payload/session helpers to use `adminPrincipal`:

```go
type consoleSessionPayload struct {
	Actor       string `json:"actor"`
	Role        string `json:"role"`
	TenantID    string `json:"tenantId,omitempty"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type consoleSessionResponse struct {
	Actor         string `json:"actor,omitempty"`
	Authenticated bool   `json:"authenticated"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	RequiresLogin bool   `json:"requiresLogin"`
	Role          string `json:"role,omitempty"`
	TenantID      string `json:"tenantId,omitempty"`
	WorkspaceID   string `json:"workspaceId,omitempty"`
}
```

Update `login`, `getAuthSession`, `logout`, `consoleSessionFromRequest`, `developmentSession`, `consoleSessionResponse`, `signConsoleSession`, and `verifyConsoleSession` so they pass/return `adminPrincipal`.

- [x] **Step 5: Parse scoped env identities**

In `internal/app/app.go`, update `adminIdentitiesFromEnv()` so the value after `actor=` is parsed as `key|role=...|tenant=...|workspace=...`:

```go
func parseAdminIdentityValue(actor string, rawValue string) (httpapi.AdminIdentity, error) {
	parts := strings.Split(rawValue, "|")
	identity := httpapi.AdminIdentity{
		Actor: strings.TrimSpace(actor),
		Key:   strings.TrimSpace(parts[0]),
		Role:  "platform_admin",
	}
	for _, part := range parts[1:] {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			return httpapi.AdminIdentity{}, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES scoped attributes must use key=value")
		}
		switch strings.TrimSpace(key) {
		case "role":
			identity.Role = strings.TrimSpace(value)
		case "tenant":
			identity.TenantID = strings.TrimSpace(value)
		case "workspace":
			identity.WorkspaceID = strings.TrimSpace(value)
		default:
			return httpapi.AdminIdentity{}, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES contains unsupported scoped attribute %q", key)
		}
	}
	if identity.Actor == "" || identity.Key == "" {
		return httpapi.AdminIdentity{}, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES entries must include actor and key")
	}
	return identity, nil
}
```

Call this helper from `adminIdentitiesFromEnv()`.

- [x] **Step 6: Run focused tests**

Run:

```bash
go test ./internal/app ./internal/httpapi -run 'TestAdminIdentitiesFromEnvParsesScopedRules|TestConsoleAuthSessionReportsScopedAdminIdentity|TestConsoleAuthSessionProtectsManagementEndpoints|TestConsoleAuthSessionReportsDevelopmentMode' -count=1
```

Expected: PASS.

### Task 3: Effective Scope For Management Lists

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [x] **Step 1: Add red list-scope test**

Add this test near `TestManagementScopeFiltersLists`:

```go
func TestScopedAdminIdentityCannotWidenManagementLists(t *testing.T) {
	repo := store.NewMemory()
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	east := createDirectAgent(t, repo, "East Agent", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	createDirectAgent(t, repo, "West Agent", "tenant-west", "ws-support", "local", domain.AgentStatusActive, nil)
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	unscoped := decodeData[[]agentResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents", nil, "", "east-key"))
	if len(unscoped) != 1 || unscoped[0].ID != east.ID {
		t.Fatalf("scoped admin without query should see only own scope, got %#v", unscoped)
	}
	rootQuery := decodeData[[]agentResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents?tenantId=tenant-root", nil, "", "east-key"))
	if len(rootQuery) != 1 || rootQuery[0].ID != east.ID {
		t.Fatalf("scoped admin root query should be narrowed to own tenant, got %#v", rootQuery)
	}
	westQuery := requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents?tenantId=tenant-west&workspaceId=ws-support", nil, "", "east-key")
	if westQuery.Code != http.StatusForbidden {
		t.Fatalf("out-of-scope tenant query should be forbidden, status=%d body=%s", westQuery.Code, westQuery.Body.String())
	}
	tenants := decodeData[[]tenantResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants", nil, "", "east-key"))
	if got := tenantResponseIDs(tenants); !reflect.DeepEqual(got, []string{"tenant-east"}) {
		t.Fatalf("scoped admin tenant list = %#v", got)
	}
}
```

- [x] **Step 2: Replace request-only scope helper**

Keep `managementScopeFromRequest(r)` as requested scope only, and add:

```go
func (s *Server) effectiveManagementScopeFromRequest(r *http.Request) (store.ManagementScope, error) {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return store.ManagementScope{}, domain.Unauthorized("admin authentication is required")
	}
	return s.effectiveManagementScope(r.Context(), managementScopeFromRequest(r), principal)
}
```

Replace list handlers that currently call `managementScopeFromRequest(r)` with `s.effectiveManagementScopeFromRequest(r)` and handle errors before calling the repository.

Minimum list handlers:

- `listAgents`
- `listAgentKeys`
- `listAccessGrants`
- `listRoutePolicies`
- `listCapabilities`
- `listPermissionPackageApplications`
- `listPermissionPackageApplicationHealth`
- `listPermissionPackageApprovalRequests`
- `listTenantEntitlements`
- `listWorkspaceAssignments`
- `listInstanceAssignments`
- `listAuditEvents`
- `listTraces`
- `runtimeMetrics`

- [x] **Step 3: Scope tenant listing**

Update `listTenants` to intersect requested `tenantId` with the admin principal. For a scoped admin with no `tenantId` query, set the filter to the admin tenant. If `parentTenantId` is supplied, verify that parent tenant intersects the admin scope before listing.

- [x] **Step 4: Run focused tests**

Run:

```bash
go test ./internal/httpapi -run 'TestScopedAdminIdentityCannotWidenManagementLists|TestManagementScopeFiltersLists|TestTenantHierarchyAPIsAndScopedManagementLists' -count=1
```

Expected: PASS.

### Task 4: Mutation Guards For Primary Management Resources

**Files:**
- Modify: `internal/httpapi/admin_scope.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [x] **Step 1: Add red mutation-scope test**

Add this test near the new scoped list test:

```go
func TestScopedAdminIdentityCannotMutateOutsideScope(t *testing.T) {
	repo := store.NewMemory()
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastAgent := createDirectAgent(t, repo, "East Agent", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westAgent := createDirectAgent(t, repo, "West Agent", "tenant-west", "ws-support", "local", domain.AgentStatusActive, nil)
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	allowed := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name": "East Created", "tenantId": "tenant-east", "workspaceId": "ws-support", "channelType": "local", "status": "active",
	}, "", "east-key")
	if allowed.Code != http.StatusCreated {
		t.Fatalf("in-scope create should succeed, status=%d body=%s", allowed.Code, allowed.Body.String())
	}
	blockedCreate := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name": "West Created", "tenantId": "tenant-west", "workspaceId": "ws-support", "channelType": "local", "status": "active",
	}, "", "east-key")
	if blockedCreate.Code != http.StatusForbidden {
		t.Fatalf("out-of-scope create should be forbidden, status=%d body=%s", blockedCreate.Code, blockedCreate.Body.String())
	}
	blockedUpdate := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/agents/"+westAgent.ID, map[string]any{"name": "Changed"}, "", "east-key")
	if blockedUpdate.Code != http.StatusForbidden {
		t.Fatalf("out-of-scope update should be forbidden, status=%d body=%s", blockedUpdate.Code, blockedUpdate.Body.String())
	}
	keyAllowed := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{"agentId": eastAgent.ID, "name": "east-key"}, "", "east-key")
	if keyAllowed.Code != http.StatusCreated {
		t.Fatalf("in-scope key create should succeed, status=%d body=%s", keyAllowed.Code, keyAllowed.Body.String())
	}
	keyBlocked := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{"agentId": westAgent.ID, "name": "west-key"}, "", "east-key")
	if keyBlocked.Code != http.StatusForbidden {
		t.Fatalf("out-of-scope key create should be forbidden, status=%d body=%s", keyBlocked.Code, keyBlocked.Body.String())
	}
}
```

- [x] **Step 2: Add reusable mutation guard helpers**

In `internal/httpapi/admin_scope.go`, add:

```go
func (s *Server) requireRequestedScopeAllowed(r *http.Request, requested store.ManagementScope) error {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return domain.Unauthorized("admin authentication is required")
	}
	_, err := s.effectiveManagementScope(r.Context(), requested, principal)
	return err
}

func (s *Server) requireAgentManagementScope(r *http.Request, agentID string) (domain.Agent, error) {
	agent, ok, err := s.repo.GetAgent(r.Context(), strings.TrimSpace(agentID))
	if err != nil {
		return domain.Agent{}, err
	}
	if !ok {
		return domain.Agent{}, domain.NotFound("agent not found")
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: agent.TenantID, WorkspaceID: agent.WorkspaceID}); err != nil {
		return domain.Agent{}, err
	}
	return agent, nil
}
```

Add these additional helpers in the same file:

```go
func (s *Server) requireTenantManagementScope(r *http.Request, tenantID string) error {
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: strings.TrimSpace(tenantID)})
}

func (s *Server) requireApplicationManagementScope(r *http.Request, application domain.PermissionPackageApplication) error {
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: application.TenantID, WorkspaceID: application.WorkspaceID})
}

func (s *Server) requireApprovalRequestManagementScope(r *http.Request, approval domain.PermissionPackageApprovalRequest) error {
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: approval.TenantID, WorkspaceID: approval.WorkspaceID})
}

func (s *Server) requireCapabilityManagementScope(r *http.Request, capability domain.Capability) error {
	target, ok, err := s.repo.GetAgent(r.Context(), strings.TrimSpace(capability.TargetID))
	if err != nil {
		return err
	}
	if !ok {
		return domain.NotFound("target agent not found")
	}
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: target.TenantID, WorkspaceID: target.WorkspaceID})
}
```

- [x] **Step 3: Guard primary resource mutations**

Before side effects, call the guard named in each bullet:

- `createTenant`: call `requireTenantManagementScope` for non-root parents; root tenant creation is allowed only when `requestAdminPrincipal(r).Role == "platform_admin"`.
- `createAgent`: call `requireRequestedScopeAllowed` with `agent.TenantID` and `agent.WorkspaceID`.
- `updateAgent`, `disableAgent`, `rotateAgentCredentials`: call `requireAgentManagementScope` with the existing agent id before update.
- `createAgentKey`, `revokeAgentKey`: load the key's agent id, then call `requireAgentManagementScope`.
- `createAccessGrant`, `revokeAccessGrant`: load the caller agent id and call `requireAgentManagementScope`; the target may be a shared platform target.
- `createRoutePolicy`, `updateRoutePolicy`, `disableRoutePolicy`: call `requireRequestedScopeAllowed` with the policy tenant/workspace.
- `refreshTargetCapabilities`, `updateCapability`: call `requireAgentManagementScope` for the target agent or `requireCapabilityManagementScope` for a capability row.
- `createTenantEntitlement`: call `requireRequestedScopeAllowed` with the entitlement tenant; the target may be a shared platform target.
- `createWorkspaceAssignment`: call `requireRequestedScopeAllowed` with assignment tenant/workspace.
- `createInstanceAssignment`: call `requireRequestedScopeAllowed` with assignment tenant/workspace and `requireAgentManagementScope` for the caller instance.

- [x] **Step 4: Run focused tests**

Run:

```bash
go test ./internal/httpapi -run 'TestScopedAdminIdentityCannotMutateOutsideScope|TestManagementAuditFailureBlocksAgentCreateAndUpdate|TestTenantHierarchyAPIsAndScopedManagementLists' -count=1
```

Expected: PASS.

### Task 5: Permission Package And Management MCP Scope Enforcement

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/management_mcp.go`
- Modify: `internal/httpapi/server_test.go`
- Modify: `internal/httpapi/management_mcp_test.go`

- [x] **Step 1: Add permission package red test**

Add this test near `TestPermissionPackageApprovalReviewerUsesAuthenticatedAdminIdentity`:

```go
func TestScopedAdminIdentityCannotOperatePermissionPackageOutsideScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
		{Actor: "west-admin", Key: "west-key", Role: "tenant_admin", TenantID: "tenant-west", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	caller := createDirectAgent(t, repo, "East Caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	target := createDirectAgent(t, repo, "Shared MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "role:support-agent",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}

	eastDraft := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, "", "east-key")
	if eastDraft.Code != http.StatusOK {
		t.Fatalf("in-scope draft should succeed, status=%d body=%s", eastDraft.Code, eastDraft.Body.String())
	}
	westDraft := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, "", "west-key")
	if westDraft.Code != http.StatusForbidden {
		t.Fatalf("out-of-scope draft should be forbidden, status=%d body=%s", westDraft.Code, westDraft.Body.String())
	}
	westPreflight := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, "", "west-key")
	if westPreflight.Code != http.StatusForbidden {
		t.Fatalf("out-of-scope preflight should be forbidden, status=%d body=%s", westPreflight.Code, westPreflight.Body.String())
	}
	westApply := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, "", "west-key")
	if westApply.Code != http.StatusForbidden {
		t.Fatalf("out-of-scope apply should be forbidden, status=%d body=%s", westApply.Code, westApply.Body.String())
	}
}
```

- [x] **Step 2: Guard permission package routes**

Guard the request tenant/workspace before building drafts or reading related state in:

- `createPermissionPackageDraft`
- `previewPermissionPackageWorkbench`
- `preflightPermissionPackage`
- `createPermissionPackageApprovalRequest`
- `approvePermissionPackageApprovalRequest`
- `rejectPermissionPackageApprovalRequest`
- `withdrawPermissionPackageApprovalRequest`
- `applyPermissionPackage`
- `getPermissionPackageProductionReadiness`
- `getPermissionPackageProductionEvidenceReport`
- `listPermissionPackageApplications`
- `listPermissionPackageApplicationHealth`
- `getPermissionPackageApplicationImpact`

For approval resolution, both checks apply:

```text
authenticated admin scope must include approval tenant/workspace
reviewer routing must allow the reviewer to resolve the approval
```

- [x] **Step 3: Add management MCP red test**

Add a management MCP test proving a scoped admin key cannot list or draft outside its scope:

```go
func TestManagementMCPInheritsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-east", "", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "", "West tenant", now)
	createDirectAgent(t, repo, "East Agent", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	createDirectAgent(t, repo, "West Agent", "tenant-west", "ws-support", "local", domain.AgentStatusActive, nil)

	resp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_agents",
			"arguments": map[string]any{
				"tenantId": "tenant-west",
			},
		},
	}, "", "east-key")
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "outside authenticated admin scope") {
		t.Fatalf("management MCP should reject out-of-scope tool arguments, status=%d body=%s", resp.Code, resp.Body.String())
	}
}
```

- [x] **Step 4: Scope management MCP tools**

In `internal/httpapi/management_mcp.go`, before each repo call that accepts `tenantId`/`workspaceId`, call the same effective-scope helper. For tool calls that mutate permission packages, validate the input scope before building the draft or applying changes.

- [x] **Step 5: Run focused tests**

Run:

```bash
go test ./internal/httpapi -run 'TestScopedAdminIdentityCannotOperatePermissionPackageOutsideScope|TestManagementMCPInheritsScopedAdminBoundary|TestPermissionPackageApprovalReviewerUsesAuthenticatedAdminIdentity|TestPermissionPackageApprovalReviewerRouting' -count=1
```

Expected: PASS.

### Task 6: Frontend Admin Scope Context

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/i18n.test.mjs`
- Modify: `frontend/tests/permissionFlowLayout.test.mjs`

- [x] **Step 1: Add source-guard tests**

In `frontend/tests/permissionFlowLayout.test.mjs`, add assertions that the console uses session scope and does not present scope as a free widening control for scoped sessions:

```js
test("console shows authenticated admin scope as persistent context", () => {
  assert.match(types, /role\\?: string/);
  assert.match(types, /tenantId\\?: string/);
  assert.match(types, /workspaceId\\?: string/);
  assert.match(app, /const sessionScope = consoleAuth\\.session/);
  assert.match(app, /admin-scope-chip/);
  assert.match(app, /auth\\.scopeTenant/);
  assert.match(app, /auth\\.role\\./);
});
```

If `types` is not already loaded in the test file, add:

```js
const types = readFileSync(new URL("../src/types.ts", import.meta.url), "utf8");
```

- [x] **Step 2: Extend `ConsoleSession`**

In `frontend/src/types.ts`, update:

```ts
export interface ConsoleSession {
  actor?: string
  authenticated: boolean
  expiresAt?: string
  requiresLogin: boolean
  role?: string
  tenantId?: string
  workspaceId?: string
}
```

- [x] **Step 3: Derive session scope and show chip**

In `ConsoleController.tsx`, derive:

```tsx
const sessionScope = consoleAuth.session?.tenantId || consoleAuth.session?.workspaceId
  ? {
      tenantId: consoleAuth.session.tenantId ?? "",
      workspaceId: consoleAuth.session.workspaceId ?? ""
    }
  : null;
const scopedSessionActive = Boolean(sessionScope);
```

Render a compact chip near the existing session chip:

```tsx
{sessionScope ? (
  <div className="admin-scope-chip" title={t("auth.scopeTitle")}>
    <ShieldCheck size={14} />
    <span>{t(`auth.role.${consoleAuth.session?.role ?? "platform_admin"}`, consoleAuth.session?.role ?? "")}</span>
    <span>{tx(t, "auth.scopeTenant", { tenant: sessionScope.tenantId || t("filter.anyTenant") })}</span>
    {sessionScope.workspaceId ? <span>{tx(t, "auth.scopeWorkspace", { workspace: sessionScope.workspaceId })}</span> : null}
  </div>
) : null}
```

When `scopedSessionActive` is true, initialize or clamp the local `scope` state to `sessionScope`. The user may narrow inside that scope, but clearing it must snap back to the session scope.

- [x] **Step 4: Add bilingual copy**

Add i18n keys:

```ts
"auth.role.platform_admin": "Platform admin",
"auth.role.security_reviewer": "Security reviewer",
"auth.role.tenant_admin": "Tenant admin",
"auth.scopeTenant": "Tenant {tenant}",
"auth.scopeTitle": "Authenticated management scope",
"auth.scopeWorkspace": "Workspace {workspace}",
```

and Chinese equivalents:

```ts
"auth.role.platform_admin": "平台管理员",
"auth.role.security_reviewer": "安全审批人",
"auth.role.tenant_admin": "租户管理员",
"auth.scopeTenant": "租户 {tenant}",
"auth.scopeTitle": "当前登录管理范围",
"auth.scopeWorkspace": "工作区 {workspace}",
```

- [x] **Step 5: Run frontend focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionFlowLayout.test.mjs tests/i18n.test.mjs
```

Expected: PASS.

### Task 7: Release Scenario, Docs, And Full Gates

**Files:**
- Create: `scripts/scenario-admin-tenant-boundary.sh`
- Modify: `Makefile`
- Modify: `tests/makefile_targets_test.sh`
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Add scenario script**

Create `scripts/scenario-admin-tenant-boundary.sh` that:

1. Starts the API with:

```bash
AGENT_HARBOR_ADMIN_IDENTITIES="platform=platform-key|role=platform_admin;east=east-key|role=tenant_admin|tenant=tenant-east|workspace=ws-support"
AGENT_HARBOR_SESSION_SECRET="admin-boundary-session-secret"
```

2. Uses platform key to create `tenant-root`, `tenant-east`, `tenant-west`, and agents in both tenants.
3. Uses east key to list agents with no query and verifies only east rows are returned.
4. Uses east key to query `tenant-west` and verifies `403`.
5. Uses east key to create an east agent and verifies `201`.
6. Uses east key to create a west agent and verifies `403`.
7. Uses east key through `/api/v1/management/mcp` and verifies out-of-scope tool arguments are rejected.

- [x] **Step 2: Wire Makefile**

Add:

```make
.PHONY: scenario-admin-tenant-boundary
scenario-admin-tenant-boundary:
	./scripts/scenario-admin-tenant-boundary.sh
```

Add `scenario-admin-tenant-boundary` to `release-check`.

- [x] **Step 3: Extend makefile target tests**

In `tests/makefile_targets_test.sh`, assert:

```bash
assert_target_exists "scenario-admin-tenant-boundary"
assert_target_depends_on "release-check" "scenario-admin-tenant-boundary"
assert_file_contains "scripts/scenario-admin-tenant-boundary.sh" "AGENT_HARBOR_ADMIN_IDENTITIES"
assert_file_contains "scripts/scenario-admin-tenant-boundary.sh" "tenant_admin"
assert_file_contains "scripts/scenario-admin-tenant-boundary.sh" "403"
```

- [x] **Step 4: Update docs**

Add README text explaining scoped admin identity syntax:

```text
AGENT_HARBOR_ADMIN_IDENTITIES="platform=platform-key|role=platform_admin;east=east-key|role=tenant_admin|tenant=tenant-east|workspace=ws-support"
```

Add CHANGELOG entries in English and Chinese for scoped admin tenant boundaries.

- [x] **Step 5: Run full gates**

Run:

```bash
go test ./...
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
git diff --check
```

Expected: all pass.
