package httpapi_test

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type accessProfileResponse struct {
	Tenant       tenantResponse               `json:"tenant"`
	ScopeTenants []tenantResponse             `json:"scopeTenants"`
	Summary      accessProfileSummaryResponse `json:"summary"`
	Grants       []accessProfileGrantResponse `json:"grants"`
	RecentTraces []traceResponse              `json:"recentTraces"`
}

type accessProfileSummaryResponse struct {
	TenantCount              int `json:"tenantCount"`
	GrantCount               int `json:"grantCount"`
	TargetCount              int `json:"targetCount"`
	CapabilityCount          int `json:"capabilityCount"`
	WorkspaceAssignmentCount int `json:"workspaceAssignmentCount"`
	InstanceAssignmentCount  int `json:"instanceAssignmentCount"`
	RecentAllowedTraceCount  int `json:"recentAllowedTraceCount"`
	RecentDeniedTraceCount   int `json:"recentDeniedTraceCount"`
}

type accessProfileGrantResponse struct {
	TenantEntitlement         tenantEntitlementResponse        `json:"tenantEntitlement"`
	Target                    *agentResponse                   `json:"target"`
	Capability                *capabilityResponse              `json:"capability"`
	EffectiveTenantDataScopes []domain.DataScope               `json:"effectiveTenantDataScopes"`
	ScopeStatus               string                           `json:"scopeStatus"`
	ScopeReason               string                           `json:"scopeReason"`
	WorkspaceAssignments      []accessProfileWorkspaceResponse `json:"workspaceAssignments"`
}

type accessProfileWorkspaceResponse struct {
	WorkspaceAssignment          workspaceAssignmentResponse     `json:"workspaceAssignment"`
	EffectiveWorkspaceDataScopes []domain.DataScope              `json:"effectiveWorkspaceDataScopes"`
	ScopeStatus                  string                          `json:"scopeStatus"`
	ScopeReason                  string                          `json:"scopeReason"`
	InstanceAssignments          []accessProfileInstanceResponse `json:"instanceAssignments"`
}

type accessProfileInstanceResponse struct {
	InstanceAssignment          instanceAssignmentResponse `json:"instanceAssignment"`
	CallerInstance              *agentResponse             `json:"callerInstance"`
	EffectiveInstanceDataScopes []domain.DataScope         `json:"effectiveInstanceDataScopes"`
	ScopeStatus                 string                     `json:"scopeStatus"`
	ScopeReason                 string                     `json:"scopeReason"`
}

func TestTenantAccessProfileExplainsGrantChainAndRecentTraces(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC().Truncate(time.Second)
	rootID := "tenant-root-profile"
	childID := "tenant-child-profile"
	grandchildID := "tenant-grandchild-profile"

	for _, body := range []map[string]any{
		{"id": rootID, "name": "Root", "status": "active"},
		{"id": childID, "parentTenantId": rootID, "name": "Child", "status": "active"},
		{"id": grandchildID, "parentTenantId": childID, "name": "Grandchild", "status": "active"},
	} {
		resp := request(t, router, http.MethodPost, "/api/v1/tenants", body, "")
		if resp.Code != http.StatusCreated {
			t.Fatalf("create tenant failed: status=%d body=%s", resp.Code, resp.Body.String())
		}
	}

	capabilityScopes := []domain.DataScope{{DataDomain: "crm", Dataset: "customers", Region: "apac"}}
	entitlementScopes := []domain.DataScope{{DataDomain: "crm", Dataset: "customers", Region: "apac", TenantFilter: childID}}
	workspaceScopes := []domain.DataScope{{DataDomain: "crm", Dataset: "customers", Table: "accounts", Region: "apac", TenantFilter: childID}}
	instanceScopes := []domain.DataScope{{DataDomain: "crm", Dataset: "customers", Table: "accounts", Field: "email", Region: "apac", TenantFilter: childID}}
	target := createDirectAgent(t, repo, "Root MCP", rootID, "ws-platform", "mcp", domain.AgentStatusActive, nil)
	caller := createDirectAgent(t, repo, "Child Caller", childID, "ws-child", "local", domain.AgentStatusActive, nil)
	capability := createDirectCapability(t, repo, target.ID, "search_customer", capabilityScopes, now)
	entitlement := createDirectTenantEntitlement(t, repo, childID, target.ID, capability.ID, entitlementScopes, now)
	workspaceAssignment := createDirectWorkspaceAssignment(t, repo, entitlement.ID, childID, "ws-child", workspaceScopes, now)
	instanceAssignment := createDirectInstanceAssignment(t, repo, workspaceAssignment.ID, childID, "ws-child", caller.ID, instanceScopes, now)

	appendProfileTrace(t, repo, domain.TraceEvent{
		ID:                    "trace-older-allowed",
		RunID:                 "run-profile",
		CallerID:              caller.ID,
		TargetID:              target.ID,
		RouteType:             "mcp",
		RouteKey:              "tools/call:search_customer",
		TenantID:              childID,
		WorkspaceID:           "ws-child",
		CallerInstanceID:      caller.ID,
		CapabilityID:          capability.ID,
		CapabilityVersion:     capability.Version,
		EntitlementID:         entitlement.ID,
		WorkspaceAssignmentID: workspaceAssignment.ID,
		InstanceAssignmentID:  instanceAssignment.ID,
		DataScopes:            instanceScopes,
		Decision:              domain.TraceDecisionAllowed,
		Reason:                "allowed older",
		CreatedAt:             now.Add(-time.Minute),
	})
	appendProfileTrace(t, repo, domain.TraceEvent{
		ID:                    "trace-newer-denied",
		RunID:                 "run-profile",
		CallerID:              caller.ID,
		TargetID:              target.ID,
		RouteType:             "mcp",
		RouteKey:              "tools/call:search_customer",
		TenantID:              childID,
		WorkspaceID:           "ws-child",
		CallerInstanceID:      caller.ID,
		CapabilityID:          capability.ID,
		CapabilityVersion:     capability.Version,
		EntitlementID:         entitlement.ID,
		WorkspaceAssignmentID: workspaceAssignment.ID,
		InstanceAssignmentID:  instanceAssignment.ID,
		DataScopes:            instanceScopes,
		Decision:              domain.TraceDecisionDenied,
		Reason:                "denied newest",
		CreatedAt:             now,
	})

	profile := decodeData[accessProfileResponse](t, request(t, router, http.MethodGet, "/api/v1/tenants/"+childID+"/access-profile?traceLimit=1", nil, ""))
	if profile.Tenant.ID != childID {
		t.Fatalf("tenant = %#v", profile.Tenant)
	}
	if got := tenantResponseIDs(profile.ScopeTenants); !reflect.DeepEqual(got, []string{childID, grandchildID}) {
		t.Fatalf("scope tenants = %#v", got)
	}
	if profile.Summary.TenantCount != 2 || profile.Summary.GrantCount != 1 || profile.Summary.TargetCount != 1 ||
		profile.Summary.CapabilityCount != 1 || profile.Summary.WorkspaceAssignmentCount != 1 ||
		profile.Summary.InstanceAssignmentCount != 1 || profile.Summary.RecentDeniedTraceCount != 1 ||
		profile.Summary.RecentAllowedTraceCount != 0 {
		t.Fatalf("unexpected summary: %#v", profile.Summary)
	}
	if len(profile.Grants) != 1 {
		t.Fatalf("grant count = %d", len(profile.Grants))
	}
	grant := profile.Grants[0]
	if grant.ScopeStatus != "valid" || grant.Target == nil || grant.Target.ID != target.ID || grant.Capability == nil || grant.Capability.ID != capability.ID {
		t.Fatalf("unexpected grant: %#v", grant)
	}
	if !reflect.DeepEqual(grant.EffectiveTenantDataScopes, entitlementScopes) {
		t.Fatalf("tenant scopes = %#v", grant.EffectiveTenantDataScopes)
	}
	if len(grant.WorkspaceAssignments) != 1 {
		t.Fatalf("workspace assignment count = %d", len(grant.WorkspaceAssignments))
	}
	workspace := grant.WorkspaceAssignments[0]
	if workspace.ScopeStatus != "valid" || !reflect.DeepEqual(workspace.EffectiveWorkspaceDataScopes, workspaceScopes) {
		t.Fatalf("unexpected workspace profile: %#v", workspace)
	}
	if len(workspace.InstanceAssignments) != 1 {
		t.Fatalf("instance assignment count = %d", len(workspace.InstanceAssignments))
	}
	instance := workspace.InstanceAssignments[0]
	if instance.ScopeStatus != "valid" || instance.CallerInstance == nil || instance.CallerInstance.ID != caller.ID ||
		!reflect.DeepEqual(instance.EffectiveInstanceDataScopes, instanceScopes) {
		t.Fatalf("unexpected instance profile: %#v", instance)
	}
	if got := traceIDs(profile.RecentTraces); !reflect.DeepEqual(got, []string{"trace-newer-denied"}) {
		t.Fatalf("recent traces = %#v", got)
	}

	filteredPath := "/api/v1/tenants/" + childID + "/access-profile?workspaceId=ws-child&targetId=" + target.ID +
		"&capabilityId=" + capability.ID + "&callerInstanceId=" + caller.ID + "&traceLimit=5"
	filtered := decodeData[accessProfileResponse](t, request(t, router, http.MethodGet, filteredPath, nil, ""))
	if filtered.Summary.GrantCount != 1 || filtered.Summary.WorkspaceAssignmentCount != 1 ||
		filtered.Summary.InstanceAssignmentCount != 1 || filtered.Summary.RecentAllowedTraceCount != 1 ||
		filtered.Summary.RecentDeniedTraceCount != 1 {
		t.Fatalf("filtered profile summary = %#v", filtered.Summary)
	}
	empty := decodeData[accessProfileResponse](t, request(t, router, http.MethodGet, "/api/v1/tenants/"+childID+"/access-profile?workspaceId=ws-other&traceLimit=0", nil, ""))
	if empty.Summary.GrantCount != 0 || len(empty.Grants) != 0 {
		t.Fatalf("workspace filter should remove grants, summary=%#v grants=%#v", empty.Summary, empty.Grants)
	}
}

func TestTenantAccessProfileFlatTenantIsExactMatch(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC().Truncate(time.Second)
	scope := []domain.DataScope{{DataDomain: "crm"}}
	flatTarget := createDirectAgent(t, repo, "Flat Target", "tenant-flat", "ws-flat", "mcp", domain.AgentStatusActive, nil)
	flatCapability := createDirectCapability(t, repo, flatTarget.ID, "flat_tool", scope, now)
	flatEntitlement := createDirectTenantEntitlement(t, repo, "tenant-flat", flatTarget.ID, flatCapability.ID, scope, now)
	createDirectWorkspaceAssignment(t, repo, flatEntitlement.ID, "tenant-flat", "ws-flat", scope, now)
	otherTarget := createDirectAgent(t, repo, "Flat Child String Target", "tenant-flat-child", "ws-flat", "mcp", domain.AgentStatusActive, nil)
	otherCapability := createDirectCapability(t, repo, otherTarget.ID, "other_tool", scope, now)
	createDirectTenantEntitlement(t, repo, "tenant-flat-child", otherTarget.ID, otherCapability.ID, scope, now)

	profile := decodeData[accessProfileResponse](t, request(t, router, http.MethodGet, "/api/v1/tenants/tenant-flat/access-profile?traceLimit=0", nil, ""))
	if profile.Tenant.ID != "tenant-flat" || profile.Tenant.Level != 0 || profile.Tenant.Name != "tenant-flat" {
		t.Fatalf("unexpected synthetic tenant: %#v", profile.Tenant)
	}
	if got := tenantResponseIDs(profile.ScopeTenants); !reflect.DeepEqual(got, []string{"tenant-flat"}) {
		t.Fatalf("flat scope tenants = %#v", got)
	}
	if len(profile.Grants) != 1 || profile.Grants[0].TenantEntitlement.ID != flatEntitlement.ID {
		t.Fatalf("flat grants = %#v", profile.Grants)
	}
	if len(profile.RecentTraces) != 0 {
		t.Fatalf("traceLimit=0 should disable traces, got %#v", profile.RecentTraces)
	}
}

func TestTenantAccessProfileReportsInvalidScopeAndValidatesTraceLimit(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC().Truncate(time.Second)
	capabilityScopes := []domain.DataScope{{DataDomain: "crm", Region: "eu"}}
	validEntitlementScopes := []domain.DataScope{{DataDomain: "crm", Region: "eu"}}
	invalidWorkspaceScopes := []domain.DataScope{{DataDomain: "crm", Region: "us"}}
	target := createDirectAgent(t, repo, "Invalid Scope Target", "tenant-invalid", "ws-invalid", "mcp", domain.AgentStatusActive, nil)
	capability := createDirectCapability(t, repo, target.ID, "invalid_scope_tool", capabilityScopes, now)
	entitlement := createDirectTenantEntitlement(t, repo, "tenant-invalid", target.ID, capability.ID, validEntitlementScopes, now)
	createDirectWorkspaceAssignment(t, repo, entitlement.ID, "tenant-invalid", "ws-invalid", invalidWorkspaceScopes, now)

	profile := decodeData[accessProfileResponse](t, request(t, router, http.MethodGet, "/api/v1/tenants/tenant-invalid/access-profile?traceLimit=0", nil, ""))
	if len(profile.Grants) != 1 || len(profile.Grants[0].WorkspaceAssignments) != 1 {
		t.Fatalf("unexpected invalid-scope profile: %#v", profile)
	}
	workspace := profile.Grants[0].WorkspaceAssignments[0]
	if workspace.ScopeStatus != "invalid" || workspace.ScopeReason == "" {
		t.Fatalf("workspace should report invalid scope, got %#v", workspace)
	}

	invalid := request(t, router, http.MethodGet, "/api/v1/tenants/tenant-invalid/access-profile?traceLimit=101", nil, "")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid traceLimit should fail, got %d body=%s", invalid.Code, invalid.Body.String())
	}
}

func createDirectCapability(t *testing.T, repo store.Repository, targetID string, key string, dataScopes []domain.DataScope, now time.Time) domain.Capability {
	t.Helper()
	capability := domain.Capability{
		ID:              security.NewID("cap"),
		TargetID:        targetID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             key,
		DisplayName:     key,
		Action:          domain.CapabilityActionRead,
		DataScopes:      dataScopes,
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       domain.CapabilityRiskLow,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	created, err := repo.UpsertCapability(t.Context(), capability)
	if err != nil {
		t.Fatalf("upsert capability: %v", err)
	}
	return created
}

func createDirectTenantEntitlement(t *testing.T, repo store.Repository, tenantID string, targetID string, capabilityID string, dataScopes []domain.DataScope, now time.Time) domain.TenantEntitlement {
	t.Helper()
	entitlement := domain.TenantEntitlement{
		ID:           security.NewID("ent"),
		TenantID:     tenantID,
		TargetID:     targetID,
		CapabilityID: capabilityID,
		Effect:       domain.PolicyEffectAllow,
		DataScopes:   dataScopes,
		Status:       domain.PolicyStatusEnabled,
		Priority:     100,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	created, err := repo.CreateTenantEntitlement(t.Context(), entitlement)
	if err != nil {
		t.Fatalf("create tenant entitlement: %v", err)
	}
	return created
}

func createDirectWorkspaceAssignment(t *testing.T, repo store.Repository, entitlementID string, tenantID string, workspaceID string, dataScopes []domain.DataScope, now time.Time) domain.WorkspaceAssignment {
	t.Helper()
	assignment := domain.WorkspaceAssignment{
		ID:                  security.NewID("wsa"),
		TenantEntitlementID: entitlementID,
		TenantID:            tenantID,
		WorkspaceID:         workspaceID,
		Effect:              domain.PolicyEffectAllow,
		DataScopes:          dataScopes,
		Status:              domain.PolicyStatusEnabled,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	created, err := repo.CreateWorkspaceAssignment(t.Context(), assignment)
	if err != nil {
		t.Fatalf("create workspace assignment: %v", err)
	}
	return created
}

func createDirectInstanceAssignment(t *testing.T, repo store.Repository, workspaceAssignmentID string, tenantID string, workspaceID string, callerID string, dataScopes []domain.DataScope, now time.Time) domain.InstanceAssignment {
	t.Helper()
	assignment := domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: workspaceAssignmentID,
		TenantID:              tenantID,
		WorkspaceID:           workspaceID,
		CallerInstanceID:      callerID,
		Effect:                domain.PolicyEffectAllow,
		DataScopes:            dataScopes,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	created, err := repo.CreateInstanceAssignment(t.Context(), assignment)
	if err != nil {
		t.Fatalf("create instance assignment: %v", err)
	}
	return created
}

func appendProfileTrace(t *testing.T, repo store.Repository, event domain.TraceEvent) {
	t.Helper()
	if _, err := repo.AppendTrace(t.Context(), event); err != nil {
		t.Fatalf("append trace: %v", err)
	}
}

func traceIDs(rows []traceResponse) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}
