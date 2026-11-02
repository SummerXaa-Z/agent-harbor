package httpapi_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestScopedAdminCannotManageAdminIdentities(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{name: "list", method: http.MethodGet, path: "/api/v1/admin-identities"},
		{name: "create", method: http.MethodPost, path: "/api/v1/admin-identities", body: map[string]any{"actor": "bad", "role": "tenant_admin", "tenantId": "tenant-east"}},
		{name: "rotate", method: http.MethodPost, path: "/api/v1/admin-identities/adm_missing/key:rotate"},
		{name: "disable", method: http.MethodPost, path: "/api/v1/admin-identities/adm_missing:disable"},
	} {
		resp := requestWithAdmin(t, router, tc.method, tc.path, tc.body, "", "east-key")
		if resp.Code != http.StatusForbidden {
			t.Fatalf("%s should be forbidden for scoped admin, got %d body=%s", tc.name, resp.Code, resp.Body.String())
		}
	}
}

func TestScopedAdminIdentityCannotWidenManagementLists(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastTarget := createDirectAgent(t, repo, "East MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, eastTarget.ID, "east_tool", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	createDirectCapabilityWithAction(t, repo, westTarget.ID, "west_tool", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	tenants := decodeData[[]tenantResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants", nil, "", "east-key"))
	if got := tenantResponseIDs(tenants); !reflect.DeepEqual(got, []string{"tenant-east"}) {
		t.Fatalf("scoped tenant admin should only see own tenant tree, got %#v", got)
	}

	agents := decodeData[[]agentResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents", nil, "", "east-key"))
	if len(agents) != 1 || agents[0].ID != eastTarget.ID {
		t.Fatalf("scoped tenant admin should only see own agents, got %#v", agents)
	}

	capabilities := decodeData[[]capabilityResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/capabilities", nil, "", "east-key"))
	if len(capabilities) != 1 || capabilities[0].Key != "east_tool" {
		t.Fatalf("scoped tenant admin should only see own capabilities, got %#v", capabilities)
	}

	widen := requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents?tenantId=tenant-west&workspaceId=ws-finance", nil, "", "east-key")
	if widen.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not widen to another tenant, got %d body=%s", widen.Code, widen.Body.String())
	}
}

func TestScopedAdminIdentityCannotReadOutsideScopeByID(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastAgent := createDirectAgent(t, repo, "East MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	westAgent := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)

	ownAgent := decodeData[agentResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents/"+eastAgent.ID, nil, "", "east-key"))
	if ownAgent.ID != eastAgent.ID {
		t.Fatalf("scoped admin should read own agent by id, got %#v", ownAgent)
	}

	westTenant := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/tenant-west", nil, "", "east-key")
	if westTenant.Code != http.StatusForbidden || strings.Contains(westTenant.Body.String(), "West tenant") {
		t.Fatalf("scoped admin should not read outside tenant by id, got %d body=%s", westTenant.Code, westTenant.Body.String())
	}

	westAgentByID := requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents/"+westAgent.ID, nil, "", "east-key")
	if westAgentByID.Code != http.StatusForbidden || strings.Contains(westAgentByID.Body.String(), "West MCP") {
		t.Fatalf("scoped admin should not read outside agent by id, got %d body=%s", westAgentByID.Code, westAgentByID.Body.String())
	}
}

func TestScopedAdminIdentityAccessGrantListHidesCrossScopeRelations(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	westCaller := createDirectAgent(t, repo, "West caller", "tenant-west", "ws-finance", "local", domain.AgentStatusActive, nil)

	supportGrant, err := repo.CreateAccessGrant(t.Context(), domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  supportCaller.ID,
		TargetID:  supportTarget.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("create support grant: %v", err)
	}
	crossGrant, err := repo.CreateAccessGrant(t.Context(), domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  westCaller.ID,
		TargetID:  supportTarget.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("create cross-scope grant: %v", err)
	}

	resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/access-grants", nil, "", "support-key")
	if resp.Code != http.StatusOK {
		t.Fatalf("list access grants failed: %d body=%s", resp.Code, resp.Body.String())
	}
	grants := decodeData[[]domain.AccessGrant](t, resp)
	if len(grants) != 1 || grants[0].ID != supportGrant.ID {
		t.Fatalf("scoped admin should only see same-scope grant %s, got %#v", supportGrant.ID, grants)
	}
	if body := resp.Body.String(); strings.Contains(body, crossGrant.ID) || strings.Contains(body, westCaller.ID) {
		t.Fatalf("scoped access grant list leaked cross-scope relation: %s", body)
	}
}

func TestScopedAdminIdentityCannotRevokeCrossScopeAccessGrant(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	crossGrant, err := repo.CreateAccessGrant(t.Context(), domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  supportCaller.ID,
		TargetID:  westTarget.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("create cross-scope grant: %v", err)
	}

	resp := requestWithAdmin(t, router, http.MethodDelete, "/api/v1/access-grants/"+crossGrant.ID, nil, "", "support-key")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("cross-scope revoke should look not found, got %d body=%s", resp.Code, resp.Body.String())
	}
	if body := resp.Body.String(); strings.Contains(body, crossGrant.ID) || strings.Contains(body, westTarget.ID) {
		t.Fatalf("cross-scope revoke leaked inaccessible grant details: %s", body)
	}
	grants, err := repo.ListAccessGrants(t.Context(), store.ManagementScope{})
	if err != nil {
		t.Fatalf("list grants: %v", err)
	}
	if len(grants) != 1 || grants[0].ID != crossGrant.ID || !grants[0].RevokedAt.IsZero() {
		t.Fatalf("cross-scope grant should remain unchanged, got %#v", grants)
	}
}

func TestScopedAdminIdentityCannotOperateDirtyRoutePolicy(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	dirtyPolicy := domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    supportCaller.TenantID,
		WorkspaceID: supportCaller.WorkspaceID,
		Name:        "Dirty cross scope allow",
		CallerID:    supportCaller.ID,
		TargetID:    westTarget.ID,
		RouteType:   "mcp",
		RouteKey:    "tools/call",
		Effect:      domain.RoutePolicyEffectAllow,
		Status:      domain.RoutePolicyStatusEnabled,
		Priority:    100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := repo.CreateRoutePolicy(t.Context(), dirtyPolicy); err != nil {
		t.Fatalf("create dirty route policy: %v", err)
	}

	listResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/route-policies?tenantId=tenant-east&workspaceId=ws-support", nil, "", "support-key")
	policies := decodeData[[]routePolicyResponse](t, listResp)
	if len(policies) != 0 {
		t.Fatalf("dirty route policy should not be visible to scoped admin, got %#v", policies)
	}
	if body := listResp.Body.String(); strings.Contains(body, dirtyPolicy.ID) || strings.Contains(body, westTarget.ID) {
		t.Fatalf("dirty route policy list leaked inaccessible details: %s", body)
	}

	patchResp := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/route-policies/"+dirtyPolicy.ID, map[string]any{
		"name": "Patched dirty route policy",
	}, "", "support-key")
	if patchResp.Code != http.StatusNotFound {
		t.Fatalf("dirty route policy patch should look not found, got %d body=%s", patchResp.Code, patchResp.Body.String())
	}
	if body := patchResp.Body.String(); strings.Contains(body, dirtyPolicy.ID) || strings.Contains(body, westTarget.ID) {
		t.Fatalf("dirty route policy patch leaked inaccessible details: %s", body)
	}

	deleteResp := requestWithAdmin(t, router, http.MethodDelete, "/api/v1/route-policies/"+dirtyPolicy.ID, nil, "", "support-key")
	if deleteResp.Code != http.StatusNotFound {
		t.Fatalf("dirty route policy delete should look not found, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	if body := deleteResp.Body.String(); strings.Contains(body, dirtyPolicy.ID) || strings.Contains(body, westTarget.ID) {
		t.Fatalf("dirty route policy delete leaked inaccessible details: %s", body)
	}
	loaded, ok, err := repo.GetRoutePolicy(t.Context(), dirtyPolicy.ID)
	if err != nil || !ok {
		t.Fatalf("dirty route policy should remain stored: ok=%v err=%v", ok, err)
	}
	if loaded.Name != dirtyPolicy.Name || loaded.Status != domain.RoutePolicyStatusEnabled {
		t.Fatalf("dirty route policy should remain unchanged, got %#v", loaded)
	}
}

func TestScopedAdminIdentityCannotMutateOutsideScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	westCaller := createDirectAgent(t, repo, "West caller", "tenant-west", "ws-finance", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	westCapability := createDirectCapabilityWithAction(t, repo, westTarget.ID, "west_tool", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	allowedAgent := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "East caller",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
		"channelType": "local",
		"status":      "active",
	}, "", "east-key")
	if allowedAgent.Code != http.StatusCreated {
		t.Fatalf("scoped tenant admin should create inside own scope, got %d body=%s", allowedAgent.Code, allowedAgent.Body.String())
	}

	deniedAgent := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "West caller",
		"tenantId":    "tenant-west",
		"workspaceId": "ws-finance",
		"channelType": "local",
	}, "", "east-key")
	if deniedAgent.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create outside tenant, got %d body=%s", deniedAgent.Code, deniedAgent.Body.String())
	}

	deniedTenant := requestWithAdmin(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-west-child",
		"parentTenantId": "tenant-west",
		"name":           "West child",
	}, "", "east-key")
	if deniedTenant.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create tenant outside scope, got %d body=%s", deniedTenant.Code, deniedTenant.Body.String())
	}

	allowedTenant := requestWithAdmin(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-east-child",
		"parentTenantId": "tenant-east",
		"name":           "East child",
	}, "", "east-key")
	if allowedTenant.Code != http.StatusCreated {
		t.Fatalf("scoped tenant admin should create child tenant inside scope, got %d body=%s", allowedTenant.Code, allowedTenant.Body.String())
	}

	updateAgent := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/agents/"+westTarget.ID, map[string]any{
		"status": "disabled",
	}, "", "east-key")
	if updateAgent.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not update outside agent, got %d body=%s", updateAgent.Code, updateAgent.Body.String())
	}

	createKey := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": westCaller.ID,
	}, "", "east-key")
	if createKey.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create outside agent key, got %d body=%s", createKey.Code, createKey.Body.String())
	}

	createGrant := requestWithAdmin(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": westCaller.ID,
		"targetAgentId": westTarget.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, "", "east-key")
	if createGrant.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create outside access grant, got %d body=%s", createGrant.Code, createGrant.Body.String())
	}

	createRoute := requestWithAdmin(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"callerAgentId": westCaller.ID,
		"targetAgentId": westTarget.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "allow",
	}, "", "east-key")
	if createRoute.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create outside route policy, got %d body=%s", createRoute.Code, createRoute.Body.String())
	}

	updateCapability := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/capabilities/"+westCapability.ID, map[string]any{
		"discoveryStatus": "approved",
	}, "", "east-key")
	if updateCapability.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not update outside capability, got %d body=%s", updateCapability.Code, updateCapability.Body.String())
	}

	refreshCapabilities := requestWithAdmin(t, router, http.MethodPost, "/api/v1/targets/"+westTarget.ID+"/capabilities:refresh", nil, "", "east-key")
	if refreshCapabilities.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not refresh outside target, got %d body=%s", refreshCapabilities.Code, refreshCapabilities.Body.String())
	}
}

func TestScopedAdminIdentityCannotOperatePermissionPackageOutsideScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
		{Actor: "platform-admin", Key: "platform-key", Role: "platform_admin"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastCaller := createDirectAgent(t, repo, "East caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westCaller := createDirectAgent(t, repo, "West caller", "tenant-west", "ws-finance", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, westTarget.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	input := map[string]any{
		"callerInstanceId": westCaller.ID,
		"region":           "us-east",
		"requestText":      "Allow sales read access.",
		"subjectSelector":  "role:sales",
		"targetId":         westTarget.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-west",
		"workspaceId":      "ws-finance",
	}

	for _, tc := range []struct {
		name   string
		path   string
		method string
	}{
		{name: "draft", method: http.MethodPost, path: "/api/v1/permission-packages/drafts"},
		{name: "preflight", method: http.MethodPost, path: "/api/v1/permission-packages:preflight"},
		{name: "apply", method: http.MethodPost, path: "/api/v1/permission-packages:apply"},
		{name: "approval", method: http.MethodPost, path: "/api/v1/permission-packages/approval-requests"},
		{name: "workbench", method: http.MethodPost, path: "/api/v1/permission-packages/workbench:preview"},
	} {
		resp := requestWithAdmin(t, router, tc.method, tc.path, input, "", "east-key")
		if resp.Code != http.StatusForbidden {
			t.Fatalf("%s should reject permission package outside admin scope, got %d body=%s", tc.name, resp.Code, resp.Body.String())
		}
	}

	ancestorTarget := createDirectAgent(t, repo, "Platform Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	ancestorCapability := createDirectCapabilityWithAction(t, repo, ancestorTarget.ID, "search_ticket", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	ancestorInput := map[string]any{
		"callerInstanceId": eastCaller.ID,
		"region":           "us-east",
		"requestText":      "Allow a child tenant to use a platform-managed target.",
		"subjectSelector":  "role:support",
		"targetId":         ancestorTarget.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	ancestorScoped := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", ancestorInput, "", "east-key")
	if ancestorScoped.Code != http.StatusForbidden || strings.Contains(ancestorScoped.Body.String(), ancestorCapability.ID) {
		t.Fatalf("tenant admin must not mutate a platform-managed ancestor target, got %d body=%s", ancestorScoped.Code, ancestorScoped.Body.String())
	}
	ancestorPlatform := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", ancestorInput, "", "platform-key")
	if ancestorPlatform.Code != http.StatusOK {
		t.Fatalf("platform admin should be able to prepare an ancestor-target permission request, got %d body=%s", ancestorPlatform.Code, ancestorPlatform.Body.String())
	}
}

func TestScopedAdminIdentityCannotGrantPermissionPackageToOutsideTarget(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	eastCaller := createDirectAgent(t, repo, "East caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	input := map[string]any{
		"callerInstanceId": eastCaller.ID,
		"region":           "us-east",
		"requestText":      "Allow support read access.",
		"subjectSelector":  "role:support",
		"targetId":         financeTarget.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}

	for _, tc := range []struct {
		name   string
		path   string
		method string
		body   any
	}{
		{name: "draft", method: http.MethodPost, path: "/api/v1/permission-packages/drafts", body: input},
		{name: "preflight", method: http.MethodPost, path: "/api/v1/permission-packages:preflight", body: input},
		{name: "apply", method: http.MethodPost, path: "/api/v1/permission-packages:apply", body: input},
		{name: "approval", method: http.MethodPost, path: "/api/v1/permission-packages/approval-requests", body: input},
		{name: "workbench", method: http.MethodPost, path: "/api/v1/permission-packages/workbench:preview", body: input},
		{name: "production readiness", method: http.MethodGet, path: permissionPackageProductionReadinessPath(input, "", "role:support"), body: nil},
		{name: "production report", method: http.MethodGet, path: permissionPackageProductionEvidenceReportPath(input, "", "role:support"), body: nil},
		{name: "tenant entitlement", method: http.MethodPost, path: "/api/v1/tenant-entitlements", body: map[string]any{
			"tenantId":     "tenant-east",
			"targetId":     financeTarget.ID,
			"capabilityId": financeCapability.ID,
			"effect":       "allow",
			"status":       "enabled",
			"priority":     40,
		}},
	} {
		resp := requestWithAdmin(t, router, tc.method, tc.path, tc.body, "", "east-key")
		if resp.Code != http.StatusForbidden || strings.Contains(resp.Body.String(), financeTarget.ID) ||
			strings.Contains(resp.Body.String(), financeCapability.ID) || strings.Contains(resp.Body.String(), "Finance MCP") ||
			strings.Contains(resp.Body.String(), "search_customer") {
			t.Fatalf("%s should reject outside target capability, got %d body=%s", tc.name, resp.Code, resp.Body.String())
		}
	}
}

func TestScopedAdminIdentityCannotUseOutsideAssignments(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeCaller := createDirectAgent(t, repo, "Finance caller", "tenant-east", "ws-finance", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	financeScopes := []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}
	financeEntitlement := createDirectTenantEntitlement(t, repo, "tenant-east", financeTarget.ID, financeCapability.ID, financeScopes, now)
	financeWorkspace := createDirectWorkspaceAssignment(t, repo, financeEntitlement.ID, "tenant-east", "ws-finance", financeScopes, now)
	createDirectInstanceAssignment(t, repo, financeWorkspace.ID, "tenant-east", "ws-finance", financeCaller.ID, financeScopes, now)

	workspaceResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": financeEntitlement.ID,
		"workspaceId":         "ws-support",
		"effect":              "allow",
		"status":              "enabled",
		"dataScopes":          financeScopes,
	}, "", "support-key")
	if workspaceResp.Code != http.StatusForbidden || strings.Contains(workspaceResp.Body.String(), "Finance") || strings.Contains(workspaceResp.Body.String(), "export_invoices") {
		t.Fatalf("scoped admin should not attach outside target entitlement, got %d body=%s", workspaceResp.Code, workspaceResp.Body.String())
	}

	instanceResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": financeWorkspace.ID,
		"callerInstanceId":      supportCaller.ID,
		"subjectSelector":       "role:support",
		"effect":                "allow",
		"status":                "enabled",
		"dataScopes":            financeScopes,
	}, "", "support-key")
	if instanceResp.Code != http.StatusForbidden || strings.Contains(instanceResp.Body.String(), "caller instance must match") ||
		strings.Contains(instanceResp.Body.String(), "Finance") || strings.Contains(instanceResp.Body.String(), "export_invoices") {
		t.Fatalf("scoped admin should not inspect outside workspace assignment, got %d body=%s", instanceResp.Code, instanceResp.Body.String())
	}
}

func TestManagementScopeFiltersLists(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)

	inScope := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-a",
		"name":        "In Scope Caller",
		"workspaceId": "ws-a",
		"channelType": "local",
		"status":      "active",
	})
	sameTenantOtherWorkspace := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-a",
		"name":        "Other Workspace",
		"workspaceId": "ws-b",
		"channelType": "local",
		"status":      "active",
	})
	otherTenantSameWorkspace := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-b",
		"name":        "Other Tenant",
		"workspaceId": "ws-a",
		"channelType": "local",
		"status":      "active",
	})

	for _, agent := range []agentResponse{inScope, sameTenantOtherWorkspace, otherTenantSameWorkspace} {
		request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
			"agentId": agent.ID,
			"name":    "key-" + agent.ID,
		}, "")
	}
	includedGrant := decodeData[grantResponse](t, request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": inScope.ID,
		"targetAgentId": inScope.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, ""))
	request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": inScope.ID,
		"targetAgentId": sameTenantOtherWorkspace.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, "")
	request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": sameTenantOtherWorkspace.ID,
		"targetAgentId": otherTenantSameWorkspace.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, "")

	now := time.Now().UTC()
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:        security.NewID("trc"),
		RunID:     "scope-run",
		CallerID:  inScope.ID,
		TargetID:  inScope.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		Decision:  domain.TraceDecisionAllowed,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("append included trace: %v", err)
	}
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:        security.NewID("trc"),
		RunID:     "scope-run",
		CallerID:  inScope.ID,
		TargetID:  sameTenantOtherWorkspace.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		Decision:  domain.TraceDecisionAllowed,
		CreatedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("append caller-only trace: %v", err)
	}
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:        security.NewID("trc"),
		RunID:     "scope-run",
		CallerID:  sameTenantOtherWorkspace.ID,
		TargetID:  otherTenantSameWorkspace.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		Decision:  domain.TraceDecisionAllowed,
		CreatedAt: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("append excluded trace: %v", err)
	}
	crossWorkspaceCapability := createDirectCapabilityWithAction(t, repo, sameTenantOtherWorkspace.ID, "export_finance", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:                security.NewID("trc"),
		RunID:             "scope-run",
		CallerID:          inScope.ID,
		TargetID:          sameTenantOtherWorkspace.ID,
		RouteType:         "mcp",
		RouteKey:          "tools/call",
		TenantID:          inScope.TenantID,
		WorkspaceID:       inScope.WorkspaceID,
		CallerInstanceID:  inScope.ID,
		CapabilityID:      crossWorkspaceCapability.ID,
		CapabilityVersion: crossWorkspaceCapability.Version,
		Decision:          domain.TraceDecisionAllowed,
		CreatedAt:         now.Add(3 * time.Second),
	}); err != nil {
		t.Fatalf("append cross-workspace capability trace: %v", err)
	}

	scopeQuery := "?tenantId=tenant-a&workspaceId=ws-a"
	agents := decodeData[[]agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents"+scopeQuery, nil, ""))
	if len(agents) != 1 || agents[0].ID != inScope.ID {
		t.Fatalf("expected scoped agents to contain only in-scope agent, got %#v", agents)
	}
	allAgents := decodeData[[]agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents", nil, ""))
	if len(allAgents) != 3 {
		t.Fatalf("unscoped agents should preserve old list behavior, got %#v", allAgents)
	}

	keys := decodeData[[]keyResponse](t, request(t, router, http.MethodGet, "/api/v1/api-keys"+scopeQuery, nil, ""))
	if len(keys) != 1 || keys[0].AgentID != inScope.ID {
		t.Fatalf("expected scoped keys to contain only in-scope caller key, got %#v", keys)
	}
	grants := decodeData[[]grantResponse](t, request(t, router, http.MethodGet, "/api/v1/access-grants"+scopeQuery, nil, ""))
	if len(grants) != 1 || grants[0].ID != includedGrant.ID {
		t.Fatalf("expected scoped grants to require caller and target in scope, got %#v", grants)
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces"+scopeQuery+"&runId=scope-run", nil, ""))
	if len(traces) != 1 || traces[0].CallerID != inScope.ID {
		t.Fatalf("expected scoped traces to require caller and target in scope, got %#v", traces)
	}
}
