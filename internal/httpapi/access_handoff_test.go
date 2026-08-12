package httpapi_test

import (
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type accessHandoffResponse struct {
	HandoffVersion string `json:"handoffVersion"`
	ID             string `json:"id"`
	Status         string `json:"status"`
	Scope          struct {
		TenantID         string `json:"tenantId"`
		WorkspaceID      string `json:"workspaceId"`
		CallerInstanceID string `json:"callerInstanceId"`
		TargetID         string `json:"targetId"`
		SubjectID        string `json:"subjectId"`
		SubjectSelector  string `json:"subjectSelector"`
	} `json:"scope"`
	Template struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
		Name    string `json:"name"`
	} `json:"template"`
	Application *struct {
		ID string `json:"id"`
	} `json:"application"`
	AllowedCapabilities []struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	} `json:"allowedCapabilities"`
	BlockedCapabilities []struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	} `json:"blockedCapabilities"`
	DataScopes          []domain.DataScope `json:"dataScopes"`
	ProductionReadiness struct {
		Status             string   `json:"status"`
		BlockingCheckCodes []string `json:"blockingCheckCodes"`
		BlockingCount      int      `json:"blockingCount"`
	} `json:"productionReadiness"`
	CopyArtifacts *struct {
		MCPClientConfig           string `json:"mcpClientConfig"`
		RuntimeRequestExample     string `json:"runtimeRequestExample"`
		PromptTemplate            string `json:"promptTemplate"`
		PermissionBoundarySummary string `json:"permissionBoundarySummary"`
	} `json:"copyArtifacts"`
	TokenEligibility struct {
		Eligible                bool     `json:"eligible"`
		DefaultExpiresInSeconds int64    `json:"defaultExpiresInSeconds"`
		MaxExpiresInSeconds     int64    `json:"maxExpiresInSeconds"`
		BlockerCodes            []string `json:"blockerCodes"`
	} `json:"tokenEligibility"`
	AuditRefs struct {
		ApplicationID          string `json:"applicationId"`
		ApprovalRequestID      string `json:"approvalRequestId"`
		AppliedAuditEventID    string `json:"appliedAuditEventId"`
		AllowedTraceID         string `json:"allowedTraceId"`
		DeniedTraceID          string `json:"deniedTraceId"`
		AcceptanceReportDigest string `json:"acceptanceReportDigest"`
	} `json:"auditRefs"`
	NextActionCode string `json:"nextActionCode"`
}

func TestPermissionPackageAccessHandoffBlocksBeforeApplyAndReturnsReadyArtifacts(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	searchCustomer := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	exportContracts := createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, "POST", "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, "POST", "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{"reviewer": "security"}, ""))

	before := decodeData[accessHandoffResponse](t, request(t, router, "GET", permissionPackageAccessHandoffPath(input, approved.ID, "user:support-001"), nil, ""))
	if before.HandoffVersion != "access-handoff/v1" || before.Status != "blocked" || before.Application != nil || before.CopyArtifacts != nil || before.TokenEligibility.Eligible {
		t.Fatalf("expected blocked handoff without usable artifacts before apply, got %#v", before)
	}
	if len(before.TokenEligibility.BlockerCodes) == 0 || before.NextActionCode != "apply_permission_package" {
		t.Fatalf("expected actionable handoff blockers before apply, got %#v", before)
	}

	applyInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            input["region"],
		"requestText":       input["requestText"],
		"subjectSelector":   input["subjectSelector"],
		"targetId":          target.ID,
		"templateId":        input["templateId"],
		"tenantId":          input["tenantId"],
		"workspaceId":       input["workspaceId"],
	}
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, "POST", "/api/v1/permission-packages:apply", applyInput, ""))
	if applied.Application == nil {
		t.Fatal("expected permission package application")
	}
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, exportContracts, "export_contracts", "user:support-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, updateTicket, "update_ticket", "user:support-001", now.Add(2*time.Minute))

	after := decodeData[accessHandoffResponse](t, request(t, router, "GET", permissionPackageAccessHandoffPath(input, "", "user:support-001"), nil, ""))
	if after.Status != "ready" || after.ID != "handoff:"+applied.Application.ID || after.Application == nil || after.Application.ID != applied.Application.ID {
		t.Fatalf("expected ready handoff for latest application, got %#v", after)
	}
	if after.Scope.TenantID != "tenant-east" || after.Scope.WorkspaceID != "ws-support" || after.Scope.CallerInstanceID != caller.ID || after.Scope.TargetID != target.ID || after.Scope.SubjectSelector != "user:support-*" {
		t.Fatalf("unexpected handoff scope: %#v", after.Scope)
	}
	if after.Template.ID != "support-ticket-triage" || after.Template.Version != 1 || after.Template.Name == "" || len(after.DataScopes) == 0 {
		t.Fatalf("unexpected handoff template or scopes: template=%#v scopes=%#v", after.Template, after.DataScopes)
	}
	if !accessHandoffCapabilitiesContain(after.AllowedCapabilities, searchCustomer.ID, "search_customer") ||
		!accessHandoffCapabilitiesContain(after.AllowedCapabilities, updateTicket.ID, "update_ticket") ||
		!accessHandoffCapabilitiesContain(after.BlockedCapabilities, exportContracts.ID, "export_contracts") {
		t.Fatalf("unexpected handoff capabilities: allowed=%#v blocked=%#v", after.AllowedCapabilities, after.BlockedCapabilities)
	}
	if !after.TokenEligibility.Eligible || after.TokenEligibility.DefaultExpiresInSeconds != 1800 || after.TokenEligibility.MaxExpiresInSeconds != 3600 || len(after.TokenEligibility.BlockerCodes) != 0 {
		t.Fatalf("expected eligible short-lived token contract, got %#v", after.TokenEligibility)
	}
	if after.CopyArtifacts == nil ||
		!strings.Contains(after.CopyArtifacts.MCPClientConfig, "/api/v1/mcp/agents/"+target.ID+"/rpc") ||
		!strings.Contains(after.CopyArtifacts.MCPClientConfig, "${AGENT_HARBOR_TOKEN}") ||
		!strings.Contains(after.CopyArtifacts.RuntimeRequestExample, "search_customer") ||
		!strings.Contains(after.CopyArtifacts.PromptTemplate, "never bypass AgentHarbor") ||
		after.CopyArtifacts.PermissionBoundarySummary == "" {
		t.Fatalf("expected safe copy artifacts, got %#v", after.CopyArtifacts)
	}
	if after.AuditRefs.ApplicationID != applied.Application.ID || after.AuditRefs.ApprovalRequestID != approved.ID || after.AuditRefs.AppliedAuditEventID == "" || after.AuditRefs.AllowedTraceID == "" || after.AuditRefs.DeniedTraceID == "" || !isSHA256Hex(after.AuditRefs.AcceptanceReportDigest) {
		t.Fatalf("expected complete handoff audit references, got %#v", after.AuditRefs)
	}
	if after.NextActionCode != "copy_access_config" || after.ProductionReadiness.Status != "ready" || after.ProductionReadiness.BlockingCount != 0 || len(after.ProductionReadiness.BlockingCheckCodes) != 0 {
		t.Fatalf("expected ready handoff next action, got %#v", after)
	}
}

func permissionPackageAccessHandoffPath(input map[string]any, approvalRequestID string, subjectID string) string {
	return strings.Replace(permissionPackageProductionReadinessPath(input, approvalRequestID, subjectID), "/production-readiness?", "/access-handoff?", 1)
}

func accessHandoffCapabilitiesContain(capabilities []struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}, id string, key string) bool {
	for _, capability := range capabilities {
		if capability.ID == id && capability.Key == key {
			return true
		}
	}
	return false
}
