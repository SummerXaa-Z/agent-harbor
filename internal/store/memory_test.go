package store

import (
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
		ID:              "cap_search",
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
		Status:              domain.PolicyStatusEnabled,
		CreatedAt:           now,
		UpdatedAt:           now,
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
		Effect:                domain.PolicyEffectAllow,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err != nil {
		t.Fatalf("create instance assignment: %v", err)
	}

	allowed, err := repo.EvaluateCapabilityAccess(ctx, CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
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
}
