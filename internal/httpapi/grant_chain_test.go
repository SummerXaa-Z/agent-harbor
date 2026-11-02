package httpapi_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestTenantHierarchyAllowsParentTargetEntitlementToDescendant(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()

	for _, body := range []map[string]any{
		{"id": "tenant-root", "name": "Root", "status": "active"},
		{"id": "tenant-child", "parentTenantId": "tenant-root", "name": "Child", "status": "active"},
		{"id": "tenant-unrelated", "name": "Unrelated", "status": "active"},
	} {
		resp := request(t, router, http.MethodPost, "/api/v1/tenants", body, "")
		if resp.Code != http.StatusCreated {
			t.Fatalf("create tenant failed: status=%d body=%s", resp.Code, resp.Body.String())
		}
	}
	target := createDirectAgent(t, repo, "Root MCP", "tenant-root", "ws-root", "mcp", domain.AgentStatusActive, nil)
	capability := domain.Capability{
		ID:              security.NewID("cap"),
		TargetID:        target.ID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             "search_customer",
		DisplayName:     "search_customer",
		Action:          domain.CapabilityActionRead,
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       domain.CapabilityRiskLow,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(t.Context(), capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}

	entitlement := request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-child",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, "")
	if entitlement.Code != http.StatusCreated {
		t.Fatalf("descendant entitlement should be allowed, got %d body=%s", entitlement.Code, entitlement.Body.String())
	}
	unrelated := request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-unrelated",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, "")
	if unrelated.Code != http.StatusBadRequest {
		t.Fatalf("unrelated entitlement should be rejected, got %d body=%s", unrelated.Code, unrelated.Body.String())
	}
}

func TestGrantChainDeleteDisablesLeafFirst(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "Grant Chain Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Grant Chain MCP", "tenant-a", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	capability := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	approved := request(t, router, http.MethodPatch, "/api/v1/capabilities/"+capability.ID, map[string]any{"discoveryStatus": "approved"}, "")
	if approved.Code != http.StatusOK {
		t.Fatalf("approve capability: status=%d body=%s", approved.Code, approved.Body.String())
	}

	entitlement := decodeData[tenantEntitlementResponse](t, request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-a",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, ""))
	workspaceAssignment := decodeData[workspaceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": entitlement.ID,
		"workspaceId":         "ws-sales",
		"effect":              "allow",
		"status":              "enabled",
	}, ""))
	instanceAssignment := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": workspaceAssignment.ID,
		"callerInstanceId":      caller.ID,
		"subjectSelector":       "user:sales-*",
		"effect":                "allow",
		"status":                "enabled",
	}, ""))

	evaluate := func() domain.CapabilityAccessDecision {
		t.Helper()
		decision, err := repo.EvaluateCapabilityAccess(t.Context(), store.CapabilityAccessRequest{
			TenantID:         "tenant-a",
			WorkspaceID:      "ws-sales",
			CallerInstanceID: caller.ID,
			SubjectID:        "user:sales-42",
			TargetID:         target.ID,
			CapabilityID:     capability.ID,
			Now:              now,
		})
		if err != nil {
			t.Fatalf("evaluate capability access: %v", err)
		}
		return decision
	}
	if decision := evaluate(); !decision.Allowed {
		t.Fatalf("expected chain to allow before disable, got %#v", decision)
	}

	entitlementBlocked := request(t, router, http.MethodDelete, "/api/v1/tenant-entitlements/"+entitlement.ID, nil, "")
	if entitlementBlocked.Code != http.StatusConflict || !strings.Contains(entitlementBlocked.Body.String(), "GRANT_CHAIN_CHILDREN_ACTIVE") {
		t.Fatalf("entitlement delete should conflict while workspace assignment is enabled, status=%d body=%s", entitlementBlocked.Code, entitlementBlocked.Body.String())
	}
	workspaceBlocked := request(t, router, http.MethodDelete, "/api/v1/workspace-assignments/"+workspaceAssignment.ID, nil, "")
	if workspaceBlocked.Code != http.StatusConflict || !strings.Contains(workspaceBlocked.Body.String(), "GRANT_CHAIN_CHILDREN_ACTIVE") {
		t.Fatalf("workspace delete should conflict while instance assignment is enabled, status=%d body=%s", workspaceBlocked.Code, workspaceBlocked.Body.String())
	}
	for _, path := range []string{
		"/api/v1/tenant-entitlements/ent_missing",
		"/api/v1/workspace-assignments/wsa_missing",
		"/api/v1/instance-assignments/ina_missing",
	} {
		missing := request(t, router, http.MethodDelete, path, nil, "")
		if missing.Code != http.StatusNotFound {
			t.Fatalf("delete %s should 404, got %d body=%s", path, missing.Code, missing.Body.String())
		}
	}

	leafDisabled := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodDelete, "/api/v1/instance-assignments/"+instanceAssignment.ID, nil, ""))
	if leafDisabled.Status != "disabled" || leafDisabled.CallerInstanceID != caller.ID {
		t.Fatalf("unexpected disabled instance assignment: %#v", leafDisabled)
	}
	workspaceDisabled := decodeData[workspaceAssignmentResponse](t, request(t, router, http.MethodDelete, "/api/v1/workspace-assignments/"+workspaceAssignment.ID, nil, ""))
	if workspaceDisabled.Status != "disabled" {
		t.Fatalf("unexpected disabled workspace assignment: %#v", workspaceDisabled)
	}
	entitlementDisabled := decodeData[tenantEntitlementResponse](t, request(t, router, http.MethodDelete, "/api/v1/tenant-entitlements/"+entitlement.ID, nil, ""))
	if entitlementDisabled.Status != "disabled" {
		t.Fatalf("unexpected disabled entitlement: %#v", entitlementDisabled)
	}

	if decision := evaluate(); decision.Allowed || !strings.Contains(decision.Reason, "no entitlement") {
		t.Fatalf("expected denied decision after disable, got %#v", decision)
	}

	idempotent := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodDelete, "/api/v1/instance-assignments/"+instanceAssignment.ID, nil, ""))
	if idempotent.Status != "disabled" {
		t.Fatalf("repeat delete should stay idempotent, got %#v", idempotent)
	}

	entitlements := decodeData[[]tenantEntitlementResponse](t, request(t, router, http.MethodGet, "/api/v1/tenant-entitlements", nil, ""))
	if len(entitlements) != 1 || entitlements[0].Status != "disabled" {
		t.Fatalf("list should show disabled entitlement, got %#v", entitlements)
	}
	assignments := decodeData[[]workspaceAssignmentResponse](t, request(t, router, http.MethodGet, "/api/v1/workspace-assignments?entitlementId="+entitlement.ID, nil, ""))
	if len(assignments) != 1 || assignments[0].Status != "disabled" {
		t.Fatalf("list should show disabled workspace assignment, got %#v", assignments)
	}
	instances := decodeData[[]instanceAssignmentResponse](t, request(t, router, http.MethodGet, "/api/v1/instance-assignments?callerInstanceId="+caller.ID, nil, ""))
	if len(instances) != 1 || instances[0].Status != "disabled" {
		t.Fatalf("list should show disabled instance assignment, got %#v", instances)
	}

	instanceAudits := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?resourceType=instance_assignment", nil, ""))
	if len(instanceAudits) < 2 || instanceAudits[len(instanceAudits)-1].Action != "instance_assignment.disabled" {
		t.Fatalf("expected instance_assignment.disabled audit, got %#v", instanceAudits)
	}
	if instanceAudits[len(instanceAudits)-1].Metadata["callerInstanceId"] != caller.ID {
		t.Fatalf("disabled audit should carry callerInstanceId, got %#v", instanceAudits[len(instanceAudits)-1])
	}
	workspaceAudits := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?resourceType=workspace_assignment", nil, ""))
	if len(workspaceAudits) < 2 || workspaceAudits[len(workspaceAudits)-1].Action != "workspace_assignment.disabled" {
		t.Fatalf("expected workspace_assignment.disabled audit, got %#v", workspaceAudits)
	}
	entitlementAudits := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?resourceType=tenant_entitlement", nil, ""))
	if len(entitlementAudits) < 2 || entitlementAudits[len(entitlementAudits)-1].Action != "tenant_entitlement.disabled" {
		t.Fatalf("expected tenant_entitlement.disabled audit, got %#v", entitlementAudits)
	}
	if entitlementAudits[len(entitlementAudits)-1].Metadata["capabilityKey"] != "search_customer" {
		t.Fatalf("disabled audit should carry capabilityKey, got %#v", entitlementAudits[len(entitlementAudits)-1])
	}
}

func TestCapabilityAssignmentDataScopesMustNarrowHierarchy(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "Scoped Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Scoped MCP", "tenant-a", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	capability := domain.Capability{
		ID:          security.NewID("cap"),
		TargetID:    target.ID,
		Type:        domain.CapabilityTypeMCPTool,
		Key:         "search_customer",
		DisplayName: "search_customer",
		Action:      domain.CapabilityActionRead,
		DataScopes: []domain.DataScope{{
			DataDomain:   "crm",
			Region:       "us-east",
			TenantFilter: "tenant_id = 'tenant-a'",
		}},
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       domain.CapabilityRiskLow,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(t.Context(), capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}

	badEntitlement := request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-a",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
		"dataScopes": []map[string]any{{
			"dataDomain": "crm",
			"region":     "eu-west",
		}},
	}, "")
	if badEntitlement.Code != http.StatusBadRequest {
		t.Fatalf("tenant entitlement scope expansion should fail, got %d body=%s", badEntitlement.Code, badEntitlement.Body.String())
	}

	entitlement := decodeData[tenantEntitlementResponse](t, request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-a",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, ""))
	badWorkspace := request(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": entitlement.ID,
		"workspaceId":         "ws-sales",
		"effect":              "allow",
		"status":              "enabled",
		"dataScopes": []map[string]any{{
			"dataDomain": "crm",
			"region":     "eu-west",
		}},
	}, "")
	if badWorkspace.Code != http.StatusBadRequest {
		t.Fatalf("workspace assignment scope expansion should fail, got %d body=%s", badWorkspace.Code, badWorkspace.Body.String())
	}

	workspaceAssignment := decodeData[workspaceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": entitlement.ID,
		"workspaceId":         "ws-sales",
		"effect":              "allow",
		"status":              "enabled",
		"dataScopes": []map[string]any{{
			"table": "accounts",
		}},
	}, ""))
	instanceAssignment := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": workspaceAssignment.ID,
		"callerInstanceId":      caller.ID,
		"subjectSelector":       "user:sales-*",
		"effect":                "allow",
		"status":                "enabled",
	}, ""))
	decision, err := repo.EvaluateCapabilityAccess(t.Context(), store.CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		SubjectID:        "user:sales-001",
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate capability access: %v", err)
	}
	wantScopes := []domain.DataScope{{
		DataDomain:   "crm",
		Table:        "accounts",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-a'",
	}}
	if !decision.Allowed || decision.InstanceAssignmentID != instanceAssignment.ID || !reflect.DeepEqual(decision.DataScopes, wantScopes) {
		t.Fatalf("unexpected decision: %#v want scopes %#v", decision, wantScopes)
	}
}
