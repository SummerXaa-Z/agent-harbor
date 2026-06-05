package store

import (
	"reflect"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

func TestMemoryCapabilityAssignmentEvaluation(t *testing.T) {
	repo := NewMemory()
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	ctx := t.Context()

	caller := domain.Agent{ID: "agt_caller", TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	target := domain.Agent{ID: "agt_mcp", TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "MCP", ChannelType: "mcp", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(ctx, caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	if _, err := repo.CreateAgent(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	capability := domain.Capability{
		ID:          "cap_search",
		TargetID:    target.ID,
		Type:        domain.CapabilityTypeMCPTool,
		Key:         "search_customer",
		DisplayName: "search_customer",
		Action:      domain.CapabilityActionRead,
		DataScopes: []domain.DataScope{{
			DataDomain:   "crm",
			Dataset:      "customers",
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
	if _, err := repo.UpsertCapability(ctx, capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}

	denied, err := repo.EvaluateCapabilityAccess(ctx, CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate denied: %v", err)
	}
	if denied.Allowed {
		t.Fatalf("capability should be denied before entitlement and assignments: %#v", denied)
	}

	entitlement, err := repo.CreateTenantEntitlement(ctx, domain.TenantEntitlement{
		ID:           "ent_search",
		TenantID:     caller.TenantID,
		TargetID:     target.ID,
		CapabilityID: capability.ID,
		Effect:       domain.PolicyEffectAllow,
		Status:       domain.PolicyStatusEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	workspaceAssignment, err := repo.CreateWorkspaceAssignment(ctx, domain.WorkspaceAssignment{
		ID:                  "wsa_search",
		TenantEntitlementID: entitlement.ID,
		TenantID:            caller.TenantID,
		WorkspaceID:         caller.WorkspaceID,
		Effect:              domain.PolicyEffectAllow,
		DataScopes: []domain.DataScope{{
			Region: "us-east",
		}},
		Status:    domain.PolicyStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create workspace assignment: %v", err)
	}
	instanceAssignment, err := repo.CreateInstanceAssignment(ctx, domain.InstanceAssignment{
		ID:                    "ina_search",
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:*",
		Effect:                domain.PolicyEffectAllow,
		DataScopes: []domain.DataScope{{
			Table: "accounts",
		}},
		Status:    domain.PolicyStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create instance assignment: %v", err)
	}

	allowed, err := repo.EvaluateCapabilityAccess(ctx, CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		SubjectID:        "user:ops",
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate allowed: %v", err)
	}
	if !allowed.Allowed || allowed.EntitlementID != entitlement.ID || allowed.WorkspaceAssignmentID != workspaceAssignment.ID || allowed.InstanceAssignmentID != instanceAssignment.ID {
		t.Fatalf("unexpected allowed decision: %#v", allowed)
	}
	wantScopes := []domain.DataScope{{
		DataDomain:   "crm",
		Dataset:      "customers",
		Table:        "accounts",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-a'",
	}}
	if !reflect.DeepEqual(allowed.DataScopes, wantScopes) {
		t.Fatalf("data scopes = %#v, want %#v", allowed.DataScopes, wantScopes)
	}

	withoutSubject, err := repo.EvaluateCapabilityAccess(ctx, CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate without subject: %v", err)
	}
	if withoutSubject.Allowed {
		t.Fatalf("subject-scoped assignment should deny missing subject: %#v", withoutSubject)
	}

	denyAssignment, err := repo.CreateInstanceAssignment(ctx, domain.InstanceAssignment{
		ID:                    "ina_search_deny",
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:ops",
		Effect:                domain.PolicyEffectDeny,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now.Add(time.Minute),
		UpdatedAt:             now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("create deny instance assignment: %v", err)
	}
	deniedBySubject, err := repo.EvaluateCapabilityAccess(ctx, CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		SubjectID:        "user:ops",
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate subject deny: %v", err)
	}
	if deniedBySubject.Allowed || deniedBySubject.InstanceAssignmentID != denyAssignment.ID {
		t.Fatalf("exact deny assignment should take precedence over wildcard allow: %#v", deniedBySubject)
	}
}

func TestMemoryPermissionPackageApplicationRoundTrip(t *testing.T) {
	repo := NewMemory()
	ctx := t.Context()
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	if _, err := repo.CreateTenant(ctx, domain.Tenant{ID: "tenant-root", Name: "Root", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create root tenant: %v", err)
	}
	if _, err := repo.CreateTenant(ctx, domain.Tenant{ID: "tenant-east", ParentTenantID: "tenant-root", Level: 1, Name: "East", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create child tenant: %v", err)
	}

	older := domain.PermissionPackageApplication{
		ID:                     "ppa_old",
		DraftID:                "ppd_old",
		TemplateID:             "sales-readonly",
		TemplateVersion:        1,
		TenantID:               "tenant-east",
		WorkspaceID:            "ws-sales",
		TargetID:               "agt_mcp",
		CallerInstanceID:       "agt_caller",
		SubjectSelector:        "user:sales-*",
		RequestText:            "old request",
		Region:                 "us-east",
		DataScopes:             []domain.DataScope{{DataDomain: "crm", Region: "us-east"}},
		AllowedCapabilityIDs:   []string{"cap_search"},
		AllowedCapabilityKeys:  []string{"search_customer"},
		TenantEntitlementIDs:   []string{"ent_search"},
		WorkspaceAssignmentIDs: []string{"wsa_search"},
		InstanceAssignmentIDs:  []string{"ina_search"},
		AppliedAt:              now,
	}
	newer := older
	newer.ID = "ppa_new"
	newer.DraftID = "ppd_new"
	newer.WorkspaceID = "ws-support"
	newer.CallerInstanceID = "agt_support"
	newer.AppliedAt = now.Add(time.Minute)

	if _, err := repo.CreatePermissionPackageApplication(ctx, older); err != nil {
		t.Fatalf("create older application: %v", err)
	}
	if _, err := repo.CreatePermissionPackageApplication(ctx, newer); err != nil {
		t.Fatalf("create newer application: %v", err)
	}

	rows, err := repo.ListPermissionPackageApplications(ctx, PermissionPackageApplicationFilter{
		ManagementScope: ManagementScope{TenantID: "tenant-root"},
		TemplateID:      "sales-readonly",
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != newer.ID || rows[0].TemplateVersion != 1 {
		t.Fatalf("expected newest scoped application, got %#v", rows)
	}
	rows[0].AllowedCapabilityIDs[0] = "mutated"
	again, err := repo.ListPermissionPackageApplications(ctx, PermissionPackageApplicationFilter{
		ManagementScope:  ManagementScope{TenantID: "tenant-root", WorkspaceID: "ws-support"},
		CallerInstanceID: "agt_support",
	})
	if err != nil {
		t.Fatalf("list applications again: %v", err)
	}
	if len(again) != 1 || again[0].ID != newer.ID || again[0].AllowedCapabilityIDs[0] != "cap_search" {
		t.Fatalf("expected cloned workspace/caller application, got %#v", again)
	}
}

func TestMemoryCapabilityAssignmentRejectsStoredDataScopeExpansion(t *testing.T) {
	repo := NewMemory()
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	ctx := t.Context()

	caller := domain.Agent{ID: "agt_caller", TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	target := domain.Agent{ID: "agt_mcp", TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "MCP", ChannelType: "mcp", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(ctx, caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	if _, err := repo.CreateAgent(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	capability := domain.Capability{
		ID:              "cap_search",
		TargetID:        target.ID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             "search_customer",
		DisplayName:     "search_customer",
		Action:          domain.CapabilityActionRead,
		DataScopes:      []domain.DataScope{{DataDomain: "crm", Region: "us-east"}},
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       domain.CapabilityRiskLow,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(ctx, capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}
	entitlement, err := repo.CreateTenantEntitlement(ctx, domain.TenantEntitlement{
		ID:           "ent_search",
		TenantID:     caller.TenantID,
		TargetID:     target.ID,
		CapabilityID: capability.ID,
		Effect:       domain.PolicyEffectAllow,
		Status:       domain.PolicyStatusEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	workspaceAssignment, err := repo.CreateWorkspaceAssignment(ctx, domain.WorkspaceAssignment{
		ID:                  "wsa_search",
		TenantEntitlementID: entitlement.ID,
		TenantID:            caller.TenantID,
		WorkspaceID:         caller.WorkspaceID,
		Effect:              domain.PolicyEffectAllow,
		DataScopes:          []domain.DataScope{{DataDomain: "crm", Region: "eu-west"}},
		Status:              domain.PolicyStatusEnabled,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		t.Fatalf("create workspace assignment: %v", err)
	}
	if _, err := repo.CreateInstanceAssignment(ctx, domain.InstanceAssignment{
		ID:                    "ina_search",
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		CallerInstanceID:      caller.ID,
		Effect:                domain.PolicyEffectAllow,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatalf("create instance assignment: %v", err)
	}

	decision, err := repo.EvaluateCapabilityAccess(ctx, CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if decision.Allowed || decision.Source != "workspace_assignment" {
		t.Fatalf("expected workspace scope expansion denial, got %#v", decision)
	}
}

func TestMemoryTenantHierarchyScopesManagementReads(t *testing.T) {
	repo := NewMemory()
	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	ctx := t.Context()

	for _, tenant := range []domain.Tenant{
		{ID: "tenant-root", Name: "Root", Level: 1, Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "tenant-child", ParentTenantID: "tenant-root", Name: "Child", Level: 2, Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "tenant-grandchild", ParentTenantID: "tenant-child", Name: "Grandchild", Level: 3, Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "tenant-unrelated", Name: "Unrelated", Level: 1, Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
	} {
		if _, err := repo.CreateTenant(ctx, tenant); err != nil {
			t.Fatalf("create tenant %s: %v", tenant.ID, err)
		}
	}
	for _, agent := range []domain.Agent{
		{ID: "agt_root", TenantID: "tenant-root", WorkspaceID: "ws-a", Name: "Root Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "agt_child", TenantID: "tenant-child", WorkspaceID: "ws-a", Name: "Child Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "agt_grandchild", TenantID: "tenant-grandchild", WorkspaceID: "ws-a", Name: "Grandchild Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "agt_unrelated", TenantID: "tenant-unrelated", WorkspaceID: "ws-a", Name: "Unrelated Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "agt_flat", TenantID: "tenant-flat", WorkspaceID: "ws-a", Name: "Flat Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "agt_flat_child_string", TenantID: "tenant-flat-child", WorkspaceID: "ws-a", Name: "Flat Child String", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
	} {
		if _, err := repo.CreateAgent(ctx, agent); err != nil {
			t.Fatalf("create agent %s: %v", agent.ID, err)
		}
	}

	tenants, err := repo.ListTenants(ctx, TenantFilter{TenantID: "tenant-root"})
	if err != nil {
		t.Fatalf("list tenants: %v", err)
	}
	if got := tenantIDs(tenants); !reflect.DeepEqual(got, []string{"tenant-root", "tenant-child", "tenant-grandchild"}) {
		t.Fatalf("tenant subtree = %#v", got)
	}
	scopedAgents, err := repo.ListAgents(ctx, AgentFilter{ManagementScope: ManagementScope{TenantID: "tenant-root", WorkspaceID: "ws-a"}})
	if err != nil {
		t.Fatalf("list scoped agents: %v", err)
	}
	if got := agentIDs(scopedAgents); !reflect.DeepEqual(got, []string{"agt_child", "agt_grandchild", "agt_root"}) {
		t.Fatalf("registered tenant scope agents = %#v", got)
	}
	flatAgents, err := repo.ListAgents(ctx, AgentFilter{ManagementScope: ManagementScope{TenantID: "tenant-flat", WorkspaceID: "ws-a"}})
	if err != nil {
		t.Fatalf("list flat agents: %v", err)
	}
	if got := agentIDs(flatAgents); !reflect.DeepEqual(got, []string{"agt_flat"}) {
		t.Fatalf("unregistered tenant scope should remain exact, got %#v", got)
	}
}

func tenantIDs(rows []domain.Tenant) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func agentIDs(rows []domain.Agent) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}
