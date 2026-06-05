package store

import (
	"errors"
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

func TestMemoryPermissionPackageApprovalRequestRoundTrip(t *testing.T) {
	repo := NewMemory()
	ctx := t.Context()
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	if _, err := repo.CreateTenant(ctx, domain.Tenant{ID: "tenant-root", Name: "Root", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create root tenant: %v", err)
	}
	if _, err := repo.CreateTenant(ctx, domain.Tenant{ID: "tenant-east", ParentTenantID: "tenant-root", Level: 1, Name: "East", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create child tenant: %v", err)
	}

	older := domain.PermissionPackageApprovalRequest{
		ID:                   "ppar_old",
		DraftID:              "ppd_old",
		TemplateID:           "support-ticket-triage",
		TemplateVersion:      1,
		PolicyVersion:        1,
		TenantID:             "tenant-east",
		WorkspaceID:          "ws-support",
		TargetID:             "agt_mcp",
		CallerInstanceID:     "agt_caller",
		SubjectSelector:      "user:support-*",
		RequestText:          "old request",
		Region:               "us-east",
		DataScopes:           []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs: []string{"cap_update"},
		AllowedCapabilityKeys: []string{
			"update_ticket",
		},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
			Reasons: []domain.PermissionPackagePolicyReason{{
				ID:            "policy:cap_update:action",
				CapabilityID:  "cap_update",
				CapabilityKey: "update_ticket",
				Severity:      "high",
				Message:       "Requires approval.",
				ReasonKey:     "permissionPolicy.actionApprovalRequired",
				ReasonValues:  map[string]string{"capability": "update_ticket", "action": "write"},
			}},
			NextActions: []string{"Request approval before applying this permission package."},
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "admin-key",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	}
	newer := older
	newer.ID = "ppar_new"
	newer.DraftID = "ppd_new"
	newer.WorkspaceID = "ws-escalations"
	newer.Status = domain.PermissionPackageApprovalStatusApproved
	newer.ReviewedBy = "security-admin"
	newer.ReviewComment = "approved for break-glass support"
	newer.ResolvedAt = now.Add(time.Minute)
	newer.ConsumedAt = now.Add(2 * time.Minute)
	newer.ConsumedByApplicationID = "ppa_new"
	newer.CreatedAt = now.Add(time.Minute)
	newer.UpdatedAt = now.Add(time.Minute)
	newer.ExpiresAt = now.Add(25 * time.Hour)

	if _, err := repo.CreatePermissionPackageApprovalRequest(ctx, older); err != nil {
		t.Fatalf("create older approval request: %v", err)
	}
	if _, err := repo.CreatePermissionPackageApprovalRequest(ctx, newer); err != nil {
		t.Fatalf("create newer approval request: %v", err)
	}

	rows, err := repo.ListPermissionPackageApprovalRequests(ctx, PermissionPackageApprovalRequestFilter{
		ManagementScope: ManagementScope{TenantID: "tenant-root"},
		TemplateID:      "support-ticket-triage",
		Status:          domain.PermissionPackageApprovalStatusApproved,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("list approval requests: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != newer.ID || rows[0].ReviewedBy != "security-admin" ||
		rows[0].ConsumedByApplicationID != "ppa_new" || rows[0].ConsumedAt.IsZero() ||
		!rows[0].ExpiresAt.Equal(newer.ExpiresAt) {
		t.Fatalf("expected newest approved request, got %#v", rows)
	}
	rows[0].AllowedCapabilityIDs[0] = "mutated"
	rows[0].PolicyGate.Reasons[0].ReasonValues["capability"] = "mutated"

	again, ok, err := repo.GetPermissionPackageApprovalRequest(ctx, older.ID)
	if err != nil || !ok {
		t.Fatalf("get older approval request: ok=%v err=%v", ok, err)
	}
	again.Status = domain.PermissionPackageApprovalStatusRejected
	again.ReviewedBy = "security-admin"
	again.ReviewComment = "too broad"
	again.ResolvedAt = now.Add(2 * time.Minute)
	again.UpdatedAt = now.Add(2 * time.Minute)
	again.ConsumedAt = now.Add(3 * time.Minute)
	again.ConsumedByApplicationID = "ppa_old"
	updated, ok, err := repo.UpdatePermissionPackageApprovalRequest(ctx, again)
	if err != nil || !ok {
		t.Fatalf("update approval request: ok=%v err=%v", ok, err)
	}
	if updated.Status != domain.PermissionPackageApprovalStatusRejected || updated.ReviewComment != "too broad" ||
		updated.ConsumedByApplicationID != "ppa_old" || updated.ConsumedAt.IsZero() {
		t.Fatalf("unexpected updated approval request: %#v", updated)
	}
	finalRows, err := repo.ListPermissionPackageApprovalRequests(ctx, PermissionPackageApprovalRequestFilter{
		ManagementScope: ManagementScope{TenantID: "tenant-root", WorkspaceID: "ws-support"},
		Status:          domain.PermissionPackageApprovalStatusRejected,
	})
	if err != nil {
		t.Fatalf("list rejected approval requests: %v", err)
	}
	if len(finalRows) != 1 || finalRows[0].ID != older.ID || finalRows[0].AllowedCapabilityIDs[0] != "cap_update" ||
		finalRows[0].PolicyGate.Reasons[0].ReasonValues["capability"] != "update_ticket" {
		t.Fatalf("expected cloned rejected request, got %#v", finalRows)
	}
}

func TestMemoryPermissionPackageApplyConsumesApprovalOnce(t *testing.T) {
	repo := NewMemory()
	ctx := t.Context()
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	capability := domain.Capability{
		ID:              "cap_update",
		TargetID:        "agt_mcp",
		Type:            domain.CapabilityTypeMCPTool,
		Key:             "update_ticket",
		DisplayName:     "Update ticket",
		Action:          domain.CapabilityActionWrite,
		DataScopes:      []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		Sensitivity:     domain.CapabilitySensitivityConfidential,
		RiskLevel:       domain.CapabilityRiskHigh,
		DiscoveryStatus: domain.CapabilityDiscoveryPendingReview,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(ctx, capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}

	approval := domain.PermissionPackageApprovalRequest{
		ID:                    "ppar_apply",
		DraftID:               "ppd_apply",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              "agt_mcp",
		CallerInstanceID:      "agt_caller",
		SubjectSelector:       "user:support-*",
		RequestText:           "need write access",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{capability.ID},
		AllowedCapabilityKeys: []string{capability.Key},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
		},
		Status:      domain.PermissionPackageApprovalStatusApproved,
		RequestedBy: "admin-key",
		ReviewedBy:  "security-admin",
		CreatedAt:   now,
		UpdatedAt:   now,
		ResolvedAt:  now,
		ExpiresAt:   now.Add(time.Hour),
	}
	if _, err := repo.CreatePermissionPackageApprovalRequest(ctx, approval); err != nil {
		t.Fatalf("create approval request: %v", err)
	}

	updatedCapability := capability
	updatedCapability.DiscoveryStatus = domain.CapabilityDiscoveryApproved
	updatedCapability.UpdatedAt = now.Add(time.Minute)
	entitlement := domain.TenantEntitlement{
		ID:           "ent_update",
		TenantID:     approval.TenantID,
		TargetID:     approval.TargetID,
		CapabilityID: capability.ID,
		Effect:       domain.PolicyEffectAllow,
		DataScopes:   approval.DataScopes,
		Status:       domain.PolicyStatusEnabled,
		Priority:     40,
		CreatedAt:    now.Add(time.Minute),
		UpdatedAt:    now.Add(time.Minute),
	}
	workspaceAssignment := domain.WorkspaceAssignment{
		ID:                  "wsa_update",
		TenantEntitlementID: entitlement.ID,
		TenantID:            approval.TenantID,
		WorkspaceID:         approval.WorkspaceID,
		Effect:              domain.PolicyEffectAllow,
		DataScopes:          approval.DataScopes,
		Status:              domain.PolicyStatusEnabled,
		CreatedAt:           now.Add(time.Minute),
		UpdatedAt:           now.Add(time.Minute),
	}
	instanceAssignment := domain.InstanceAssignment{
		ID:                    "ina_update",
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              approval.TenantID,
		WorkspaceID:           approval.WorkspaceID,
		CallerInstanceID:      approval.CallerInstanceID,
		SubjectSelector:       approval.SubjectSelector,
		Effect:                domain.PolicyEffectAllow,
		DataScopes:            approval.DataScopes,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now.Add(time.Minute),
		UpdatedAt:             now.Add(time.Minute),
	}
	application := domain.PermissionPackageApplication{
		ID:                     "ppa_apply",
		DraftID:                approval.DraftID,
		TemplateID:             approval.TemplateID,
		TemplateVersion:        approval.TemplateVersion,
		TenantID:               approval.TenantID,
		WorkspaceID:            approval.WorkspaceID,
		TargetID:               approval.TargetID,
		CallerInstanceID:       approval.CallerInstanceID,
		SubjectSelector:        approval.SubjectSelector,
		RequestText:            approval.RequestText,
		Region:                 approval.Region,
		DataScopes:             approval.DataScopes,
		AllowedCapabilityIDs:   []string{capability.ID},
		AllowedCapabilityKeys:  []string{capability.Key},
		TenantEntitlementIDs:   []string{entitlement.ID},
		WorkspaceAssignmentIDs: []string{workspaceAssignment.ID},
		InstanceAssignmentIDs:  []string{instanceAssignment.ID},
		AppliedAt:              now.Add(time.Minute),
	}
	consumedApproval := approval
	consumedApproval.ConsumedAt = now.Add(time.Minute)
	consumedApproval.ConsumedByApplicationID = application.ID
	consumedApproval.UpdatedAt = now.Add(time.Minute)

	mutation := PermissionPackageApplyMutation{
		Capabilities:         []domain.Capability{updatedCapability},
		TenantEntitlements:   []domain.TenantEntitlement{entitlement},
		WorkspaceAssignments: []domain.WorkspaceAssignment{workspaceAssignment},
		InstanceAssignments:  []domain.InstanceAssignment{instanceAssignment},
		Application:          application,
		ApprovalRequest:      &consumedApproval,
		AuditEvent: domain.AuditEvent{
			ID:           "aud_apply",
			TenantID:     approval.TenantID,
			WorkspaceID:  approval.WorkspaceID,
			Actor:        "admin-key",
			Action:       "permission_package.applied",
			ResourceType: "permission_package",
			ResourceID:   application.ID,
			CreatedAt:    now.Add(time.Minute),
		},
	}
	applied, err := repo.ApplyPermissionPackage(ctx, mutation)
	if err != nil {
		t.Fatalf("apply permission package: %v", err)
	}
	if applied.Application.ID != application.ID || applied.ApprovalRequest == nil ||
		applied.ApprovalRequest.ConsumedByApplicationID != application.ID || applied.ApprovalRequest.ConsumedAt.IsZero() {
		t.Fatalf("unexpected apply result: %#v", applied)
	}
	loadedApproval, ok, err := repo.GetPermissionPackageApprovalRequest(ctx, approval.ID)
	if err != nil || !ok {
		t.Fatalf("get consumed approval request: ok=%v err=%v", ok, err)
	}
	if loadedApproval.ConsumedByApplicationID != application.ID || loadedApproval.ConsumedAt.IsZero() {
		t.Fatalf("approval request should be consumed once, got %#v", loadedApproval)
	}
	events, err := repo.ListAuditEvents(ctx, AuditEventFilter{Action: "permission_package.applied"})
	if err != nil || len(events) != 1 {
		t.Fatalf("expected one apply audit event, events=%#v err=%v", events, err)
	}

	secondMutation := mutation
	secondApplication := application
	secondApplication.ID = "ppa_apply_second"
	secondMutation.Application = secondApplication
	secondConsumedApproval := consumedApproval
	secondConsumedApproval.ConsumedByApplicationID = secondApplication.ID
	secondMutation.ApprovalRequest = &secondConsumedApproval
	if _, err := repo.ApplyPermissionPackage(ctx, secondMutation); !errors.Is(err, ErrPermissionPackageApprovalNotConsumable) {
		t.Fatalf("expected consumed approval error on retry, got %v", err)
	}
	applications, err := repo.ListPermissionPackageApplications(ctx, PermissionPackageApplicationFilter{})
	if err != nil || len(applications) != 1 || applications[0].ID != application.ID {
		t.Fatalf("retry should not create another application, applications=%#v err=%v", applications, err)
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
