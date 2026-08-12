package permissionpack

import (
	"slices"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

func TestBuildDraftPolicyGateAllowsLowRiskRead(t *testing.T) {
	draft, err := BuildDraft(permissionPackageTestInput("sales-readonly"), []domain.Capability{
		permissionPackageTestCapability("cap-read", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal),
	})
	if err != nil {
		t.Fatalf("build draft: %v", err)
	}

	if draft.PolicyGate.Decision != domain.PermissionPackagePolicyDecisionAllow {
		t.Fatalf("expected allow policy decision, got %q", draft.PolicyGate.Decision)
	}
	if !draft.PolicyGate.CanApplyDirectly {
		t.Fatalf("expected low-risk read package to be directly applicable: %#v", draft.PolicyGate)
	}
	if draft.PolicyGate.PolicyVersion != 1 {
		t.Fatalf("expected policy version 1, got %d", draft.PolicyGate.PolicyVersion)
	}
	if len(draft.PolicyGate.Reasons) != 0 {
		t.Fatalf("expected no policy gate reasons, got %#v", draft.PolicyGate.Reasons)
	}
}

func TestBuildDraftPolicyGateRequiresApprovalForRiskyAllowedCapability(t *testing.T) {
	draft, err := BuildDraft(permissionPackageTestInput("support-ticket-triage"), []domain.Capability{
		permissionPackageTestCapabilityWithDomain("cap-write", "support", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential),
	})
	if err != nil {
		t.Fatalf("build draft: %v", err)
	}

	if !draft.Readiness.CanApply {
		t.Fatalf("expected structurally ready draft before policy approval check: %#v", draft.Readiness)
	}
	if draft.PolicyGate.Decision != domain.PermissionPackagePolicyDecisionApprovalRequired {
		t.Fatalf("expected approval required policy decision, got %q", draft.PolicyGate.Decision)
	}
	if draft.PolicyGate.CanApplyDirectly {
		t.Fatalf("expected risky package to require approval: %#v", draft.PolicyGate)
	}
	if draft.PolicyGate.PolicyVersion != 1 {
		t.Fatalf("expected policy version 1, got %d", draft.PolicyGate.PolicyVersion)
	}
	if len(draft.PolicyGate.Reasons) == 0 {
		t.Fatalf("expected policy gate reasons")
	}
	if draft.PolicyGate.Reasons[0].CapabilityID != "cap-write" {
		t.Fatalf("expected capability-level reason, got %#v", draft.PolicyGate.Reasons[0])
	}
	if len(draft.PolicyGate.NextActions) == 0 {
		t.Fatalf("expected policy gate next actions")
	}
	if !slices.Contains(draft.PolicyGate.NextActionCodes, domain.PermissionPackagePolicyNextCreateApproval) {
		t.Fatalf("expected policy gate next action code, got %#v", draft.PolicyGate.NextActionCodes)
	}
}

func TestBuildDraftRejectsUnboundedSubjectSelector(t *testing.T) {
	for _, subjectSelector := range []string{"", " ", "*"} {
		input := permissionPackageTestInput("support-ticket-triage")
		input.SubjectSelector = subjectSelector
		draft, err := BuildDraft(input, []domain.Capability{
			permissionPackageTestCapabilityWithDomain("cap-write", "support", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential),
		})
		if err != nil {
			t.Fatalf("build draft with subjectSelector %q: %v", subjectSelector, err)
		}
		if draft.Readiness.CanApply {
			t.Fatalf("subjectSelector %q should not produce an applicable draft: %#v", subjectSelector, draft.Readiness)
		}
		if !containsString(draft.Readiness.MissingFields, "subjectSelector") {
			t.Fatalf("subjectSelector %q should be reported as missing, got %#v", subjectSelector, draft.Readiness)
		}
	}
}

func TestBuildDraftBlocksCapabilitiesOutsideTemplateDomain(t *testing.T) {
	draft, err := BuildDraft(permissionPackageTestInput("support-ticket-triage"), []domain.Capability{
		permissionPackageTestCapabilityWithDomain("cap-support", "support", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal),
		permissionPackageTestCapabilityWithDomain("cap-crm", "crm", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal),
	})
	if err != nil {
		t.Fatalf("build draft: %v", err)
	}
	if len(draft.AllowedCapabilities) != 1 || draft.AllowedCapabilities[0].ID != "cap-support" {
		t.Fatalf("expected only support capability allowed, got %#v", draft.AllowedCapabilities)
	}
	if len(draft.BlockedCapabilities) != 1 || draft.BlockedCapabilities[0].ID != "cap-crm" {
		t.Fatalf("expected CRM capability blocked, got %#v", draft.BlockedCapabilities)
	}
}

func TestBuildDraftBlocksCapabilityWithoutDomain(t *testing.T) {
	capability := permissionPackageTestCapabilityWithDomain("cap-unknown-domain", "", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal)
	draft, err := BuildDraft(permissionPackageTestInput("sales-readonly"), []domain.Capability{capability})
	if err != nil {
		t.Fatalf("build draft: %v", err)
	}
	if len(draft.AllowedCapabilities) != 0 || len(draft.BlockedCapabilities) != 1 {
		t.Fatalf("expected unknown-domain capability to fail closed, got allowed=%#v blocked=%#v", draft.AllowedCapabilities, draft.BlockedCapabilities)
	}
	if draft.Readiness.CanApply {
		t.Fatal("expected unknown-domain draft not ready")
	}
}

func TestBuildDraftEnforcesDenyGuardrail(t *testing.T) {
	capability := permissionPackageTestCapabilityWithDomain("cap-delete-ticket", "support", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal)
	capability.Key = "delete-ticket"
	draft, err := BuildDraft(permissionPackageTestInput("support-ticket-triage"), []domain.Capability{capability})
	if err != nil {
		t.Fatalf("build draft: %v", err)
	}
	if len(draft.AllowedCapabilities) != 0 || len(draft.BlockedCapabilities) != 1 {
		t.Fatalf("expected deny guardrail capability blocked, got allowed=%#v blocked=%#v", draft.AllowedCapabilities, draft.BlockedCapabilities)
	}
}

func TestBuildDraftNarrowsRequestedCapabilityToExactAllowBoundary(t *testing.T) {
	input := permissionPackageTestInput("support-ticket-triage")
	input.RequestedCapabilityID = "cap-update-ticket"
	draft, err := BuildDraft(input, []domain.Capability{
		permissionPackageTestCapabilityWithDomain("cap-read-ticket", "support", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal),
		permissionPackageTestCapabilityWithDomain("cap-update-ticket", "support", domain.CapabilityActionWrite, domain.CapabilityRiskMedium, domain.CapabilitySensitivityInternal),
	})
	if err != nil {
		t.Fatalf("build exact capability draft: %v", err)
	}
	if !draft.Readiness.CanApply {
		t.Fatalf("expected requested capability draft to be structurally ready: %#v", draft.Readiness)
	}
	if len(draft.AllowedCapabilities) != 1 || draft.AllowedCapabilities[0].ID != input.RequestedCapabilityID {
		t.Fatalf("expected only requested capability allowed, got %#v", draft.AllowedCapabilities)
	}
	if len(draft.BlockedCapabilities) != 1 || draft.BlockedCapabilities[0].ID != "cap-read-ticket" {
		t.Fatalf("expected sibling capability retained as blocked evidence, got %#v", draft.BlockedCapabilities)
	}
}

func TestBuildDraftFailsClosedWhenRequestedCapabilityIsNotSafelyCovered(t *testing.T) {
	tests := []struct {
		name       string
		capability domain.Capability
	}{
		{
			name:       "missing",
			capability: permissionPackageTestCapabilityWithDomain("cap-other", "support", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal),
		},
		{
			name:       "wrong domain",
			capability: permissionPackageTestCapabilityWithDomain("cap-export", "crm", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal),
		},
		{
			name:       "disallowed action",
			capability: permissionPackageTestCapabilityWithDomain("cap-export", "support", domain.CapabilityActionExport, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := permissionPackageTestInput("support-ticket-triage")
			input.RequestedCapabilityID = "cap-export"
			draft, err := BuildDraft(input, []domain.Capability{test.capability})
			if err != nil {
				t.Fatalf("build exact capability draft: %v", err)
			}
			if draft.Readiness.CanApply || len(draft.AllowedCapabilities) != 0 {
				t.Fatalf("unsafe requested capability must fail closed: %#v", draft)
			}
			if !containsString(draft.Readiness.Warnings, "The requested capability is not safely covered by this permission package.") {
				t.Fatalf("expected exact capability blocker, got %#v", draft.Readiness.Warnings)
			}
		})
	}
}

func permissionPackageTestInput(templateID string) domain.PermissionPackageDraftRequest {
	return domain.PermissionPackageDraftRequest{
		CallerInstanceID: "inst-sales",
		Region:           "us-east",
		RequestText:      "Grant access for a scoped tenant.",
		SubjectSelector:  "user:sales-*",
		TargetID:         "target-crm",
		TemplateID:       templateID,
		TenantID:         "tenant-east",
		WorkspaceID:      "workspace-sales",
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func permissionPackageTestCapability(id string, action domain.CapabilityAction, risk domain.CapabilityRisk, sensitivity domain.CapabilitySensitivity) domain.Capability {
	return permissionPackageTestCapabilityWithDomain(id, "crm", action, risk, sensitivity)
}

func permissionPackageTestCapabilityWithDomain(id string, dataDomain string, action domain.CapabilityAction, risk domain.CapabilityRisk, sensitivity domain.CapabilitySensitivity) domain.Capability {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	return domain.Capability{
		ID:              id,
		TargetID:        "target-crm",
		Type:            domain.CapabilityTypeMCPTool,
		Key:             id + "-key",
		DisplayName:     id,
		Action:          action,
		DataDomains:     []string{dataDomain},
		DataScopes:      []domain.DataScope{{DataDomain: dataDomain, Region: "us-east", TenantFilter: "tenant_id = 'tenant-east'"}},
		Sensitivity:     sensitivity,
		RiskLevel:       risk,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
}
