package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestPermissionPackageExactSingleCapabilityDoesNotRequireImpossibleDeniedTrace(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Single-tool Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	searchTicket := createDirectCapabilityWithAction(t, repo, target.ID, "search_ticket", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	input := map[string]any{
		"callerInstanceId":      caller.ID,
		"region":                "us-east",
		"requestText":           "Allow only the single safe ticket search capability.",
		"requestedCapabilityId": searchTicket.ID,
		"subjectSelector":       "user:support-*",
		"targetId":              target.ID,
		"templateId":            "support-ticket-triage",
		"tenantId":              "tenant-east",
		"workspaceId":           "ws-support",
	}
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, ""))
	if applied.Application == nil || len(applied.Draft.BlockedCapabilities) != 0 || len(applied.Application.AllowedCapabilityIDs) != 1 {
		t.Fatalf("expected exact single-capability application, got %#v", applied)
	}
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, searchTicket, "search_ticket", "user:support-001", time.Now().UTC().Add(time.Second))
	readiness := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, permissionPackageProductionReadinessPath(input, "", "user:support-001"), nil, ""))
	if readiness.Status != "ready" || readiness.Summary.BlockingCount != 0 || !readiness.Summary.HasAllowedTrace || readiness.Summary.HasDeniedTrace {
		t.Fatalf("single-capability readiness should not require impossible denied evidence: %#v", readiness)
	}
	if !permissionPackageProductionReadinessHasCheck(readiness.Checks, "runtime_denied_trace_not_applicable", "passed") {
		t.Fatalf("expected explicit denied-evidence not-applicable check: %#v", readiness.Checks)
	}
}

func TestPermissionPackageExactReadinessAcceptsEquivalentLegacySingleCapabilityApplication(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-east", "", "East tenant", now)
	caller := createDirectAgent(t, repo, "Support Assistant", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	target := createDirectAgent(t, repo, "Legacy single-tool MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	searchTicket := createDirectCapabilityWithAction(t, repo, target.ID, "search_ticket", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	legacyInput := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Legacy package request whose effective boundary is one capability.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", legacyInput, ""))
	if applied.Application == nil || applied.Application.RequestedCapabilityID != "" || len(applied.Application.AllowedCapabilityIDs) != 1 || applied.Application.AllowedCapabilityIDs[0] != searchTicket.ID {
		t.Fatalf("expected one-capability legacy application provenance: %#v", applied.Application)
	}
	exactInput := map[string]any{}
	for key, value := range legacyInput {
		exactInput[key] = value
	}
	exactInput["requestedCapabilityId"] = searchTicket.ID
	readiness := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, permissionPackageProductionReadinessPath(exactInput, "", "user:support-001"), nil, ""))
	if readiness.LatestApplication == nil || readiness.LatestApplication.ID != applied.Application.ID || readiness.LatestApplication.RequestedCapabilityID != "" {
		t.Fatalf("exact readiness should reuse equivalent legacy provenance without rewriting it: %#v", readiness.LatestApplication)
	}
	if !permissionPackageProductionReadinessHasCheck(readiness.Checks, "legacy_exact_equivalent", "info") || !permissionPackageProductionReadinessHasCheck(readiness.Checks, "application_scope_match", "passed") {
		t.Fatalf("expected transparent legacy exact equivalence checks: %#v", readiness.Checks)
	}
	report := decodeData[permissionPackageProductionEvidenceReportResponse](t, request(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(exactInput, "", "user:support-001"), nil, ""))
	if report.Scope.RequestedCapabilityID != searchTicket.ID || report.Evidence.Application.RequestedCapabilityID != "" {
		t.Fatalf("report must preserve exact query scope and legacy application provenance separately: %#v", report)
	}
	duplicate := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", exactInput, "")
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), "PERMISSION_PACKAGE_ALREADY_APPLIED") {
		t.Fatalf("equivalent exact request must not create duplicate grants, status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	applications := decodeData[[]permissionPackageApplicationResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?tenantId=tenant-east&workspaceId=ws-support", nil, ""))
	if len(applications) != 1 || applications[0].RequestedCapabilityID != "" {
		t.Fatalf("legacy exact compatibility must not add or rewrite application provenance: %#v", applications)
	}
}

func TestPermissionPackageWorkbenchPreviewSummarizesPrimaryJourney(t *testing.T) {
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
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))

	beforeApply := decodeData[permissionPackageWorkbenchPreviewResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/workbench:preview", input, ""))
	if beforeApply.ApprovalRequest == nil || beforeApply.ApprovalRequest.ID != approved.ID {
		t.Fatalf("expected workbench preview to select the approved request, got %#v", beforeApply.ApprovalRequest)
	}
	if beforeApply.Summary.Status != "ready_to_apply" || beforeApply.Summary.PrimaryActionCode != "apply_permission_package" ||
		!beforeApply.Summary.ApprovalRequired || !beforeApply.Summary.CanApply || beforeApply.Summary.Applied ||
		beforeApply.Summary.AllowedCapabilityCount != 1 || beforeApply.Summary.BlockedCapabilityCount != 2 ||
		beforeApply.Summary.PlannedObjectCount != 3 || beforeApply.ProductionReadiness == nil ||
		beforeApply.ProductionReadiness.NextActionCode != "apply_permission_package" {
		t.Fatalf("expected ready-to-apply workbench summary, got %#v", beforeApply.Summary)
	}
	if !permissionPackageWorkbenchHasStep(beforeApply.Summary.Steps, "approval", "complete", "approval_approved") ||
		!permissionPackageWorkbenchHasStep(beforeApply.Summary.Steps, "apply", "current", "apply_ready") ||
		!permissionPackageWorkbenchHasStep(beforeApply.Summary.Steps, "validation", "waiting", "validation_waiting") {
		t.Fatalf("expected approval complete and apply current steps, got %#v", beforeApply.Summary.Steps)
	}
	if requestStep, ok := permissionPackageWorkbenchStepByKey(beforeApply.Summary.Steps, "request"); !ok || requestStep.Count != 0 || requestStep.Total != 0 {
		t.Fatalf("expected request step to avoid capability count noise, got step=%#v ok=%v", requestStep, ok)
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
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, ""))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, exportContracts, "export_contracts", "user:support-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, searchCustomer, "search_customer", "user:support-001", now.Add(2*time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, updateTicket, "update_ticket", "user:support-001", now.Add(3*time.Minute))

	afterEvidence := decodeData[permissionPackageWorkbenchPreviewResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/workbench:preview", input, ""))
	if afterEvidence.LatestApplication == nil || afterEvidence.LatestApplication.ID != applied.Application.ID ||
		afterEvidence.Summary.Status != "production_ready" || afterEvidence.Summary.PrimaryActionCode != "export_acceptance_report" ||
		!afterEvidence.Summary.Applied || !afterEvidence.Summary.RuntimeEvidenceReady || !afterEvidence.Summary.ProductionReady {
		t.Fatalf("expected production-ready workbench summary after evidence, got summary=%#v app=%#v", afterEvidence.Summary, afterEvidence.LatestApplication)
	}
	if !permissionPackageWorkbenchHasStep(afterEvidence.Summary.Steps, "approval", "complete", "approval_approved") ||
		!permissionPackageWorkbenchHasStep(afterEvidence.Summary.Steps, "acceptance", "complete", "acceptance_ready") {
		t.Fatalf("expected completed approval and acceptance after evidence, got %#v", afterEvidence.Summary.Steps)
	}

	secondApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	afterNewRequest := decodeData[permissionPackageWorkbenchPreviewResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/workbench:preview", input, ""))
	if afterNewRequest.ApprovalRequest == nil || afterNewRequest.ApprovalRequest.ID != secondApproval.ID {
		t.Fatalf("expected workbench preview to select the new pending request, got %#v", afterNewRequest.ApprovalRequest)
	}
	if afterNewRequest.LatestApplication != nil || afterNewRequest.ProductionReadiness != nil {
		t.Fatalf("expected pending request to suppress historical production evidence, got app=%#v readiness=%#v", afterNewRequest.LatestApplication, afterNewRequest.ProductionReadiness)
	}
	if afterNewRequest.Summary.Status != "awaiting_approval" || afterNewRequest.Summary.PrimaryActionCode != "review_approval_request" ||
		afterNewRequest.Summary.Applied || afterNewRequest.Summary.ProductionReady || afterNewRequest.Summary.CanApply {
		t.Fatalf("expected pending request to restart the approval step, got %#v", afterNewRequest.Summary)
	}
	if !permissionPackageWorkbenchHasStep(afterNewRequest.Summary.Steps, "approval", "current", "approval_pending") ||
		!permissionPackageWorkbenchHasStep(afterNewRequest.Summary.Steps, "apply", "waiting", "apply_waiting") {
		t.Fatalf("expected pending approval and waiting apply steps, got %#v", afterNewRequest.Summary.Steps)
	}
}

func TestPermissionPackageWorkbenchIgnoresExpiredPendingApprovalRequest(t *testing.T) {
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
	createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

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
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	expired, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get approval for expiry: ok=%v err=%v", ok, err)
	}
	expired.ExpiresAt = now.Add(-time.Minute)
	expired.UpdatedAt = expired.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), expired); err != nil || !ok {
		t.Fatalf("expire approval request: ok=%v err=%v", ok, err)
	}

	preview := decodeData[permissionPackageWorkbenchPreviewResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/workbench:preview", input, ""))
	if preview.ApprovalRequest != nil {
		t.Fatalf("expected expired pending approval to be ignored, got %#v", preview.ApprovalRequest)
	}
	if preview.Summary.Status != "awaiting_approval" || preview.Summary.PrimaryActionCode != "create_approval_request" ||
		preview.Summary.CanApply || preview.Summary.Applied || preview.Summary.ProductionReady {
		t.Fatalf("expected workbench to ask for a fresh approval request, got %#v", preview.Summary)
	}
	if !permissionPackageWorkbenchHasStep(preview.Summary.Steps, "approval", "current", "approval_required") {
		t.Fatalf("expected approval step to request a fresh approval, got %#v", preview.Summary.Steps)
	}
}

func TestPermissionPackageProductionReadinessBlocksBeforeApplyAndReadyAfterEvidence(t *testing.T) {
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
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	if approved.Status != "approved" {
		t.Fatalf("expected approved request, got %#v", approved)
	}

	before := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, permissionPackageProductionReadinessPath(input, approved.ID, "user:support-001"), nil, ""))
	if before.Status != "blocked" || before.Summary.BlockingCount == 0 || before.Summary.HasApplication {
		t.Fatalf("expected production readiness to block before apply, got %#v", before)
	}
	if before.NextActionCode != "apply_permission_package" {
		t.Fatalf("expected apply next action before application evidence, got %q", before.NextActionCode)
	}
	if !permissionPackageProductionReadinessHasCheck(before.Checks, "preflight_ready", "passed") ||
		!permissionPackageProductionReadinessHasCheck(before.Checks, "application_present", "blocking") {
		t.Fatalf("expected ready preflight and missing application blocker, got %#v", before.Checks)
	}
	beforeReportResp := request(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(input, approved.ID, "user:support-001"), nil, "")
	beforeReport := decodeData[permissionPackageProductionEvidenceReportResponse](t, beforeReportResp)
	if beforeReport.ReportVersion != "production-readiness-report/v1" || beforeReport.Status != "blocked" ||
		beforeReport.Scope.TenantID != "tenant-east" || beforeReport.Scope.SubjectID != "user:support-001" ||
		beforeReport.Evidence.Application.Present || beforeReport.Summary.BlockingCount == 0 ||
		beforeReport.NextActionCode != "apply_permission_package" ||
		!permissionPackageProductionReadinessHasCheck(beforeReport.Checks, "application_present", "blocking") {
		t.Fatalf("expected blocked production evidence report before apply, got %#v", beforeReport)
	}
	if beforeReport.PlatformContract.APIVersion != "2026-06-15" ||
		beforeReport.PlatformContract.ManagementMcpToolCatalog.MetadataVersion != 4 ||
		!isSHA256Hex(beforeReport.PlatformContract.ManagementMcpToolCatalog.CatalogDigest) {
		t.Fatalf("expected blocked production report platform contract, got %#v", beforeReport.PlatformContract)
	}
	if beforeReport.GeneratedBy != "local-dev" {
		t.Fatalf("expected blocked production report generatedBy=%q, got %q", "local-dev", beforeReport.GeneratedBy)
	}
	assertProductionReportDigest(t, beforeReportResp, beforeReport.ReportDigest)

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
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, ""))
	if applied.Application == nil || applied.Application.TemplateID != "support-ticket-triage" {
		t.Fatalf("expected applied support package application, got %#v", applied)
	}
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, exportContracts, "export_contracts", "user:support-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, updateTicket, "update_ticket", "user:support-001", now.Add(2*time.Minute))

	after := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, permissionPackageProductionReadinessPath(input, "", "user:support-001"), nil, ""))
	if after.Status != "ready" || after.Summary.BlockingCount != 0 || !after.Summary.HasApplication ||
		!after.Summary.HasAllowedTrace || !after.Summary.HasDeniedTrace || !after.Summary.HasAppliedAudit ||
		!after.Summary.AccessProfileReady || after.LatestApplication == nil || after.LatestApplication.ID != applied.Application.ID {
		t.Fatalf("expected production readiness after evidence, got %#v", after)
	}
	if after.NextActionCode != "export_acceptance_report" {
		t.Fatalf("expected acceptance report export next action after readiness, got %q", after.NextActionCode)
	}
	if after.RuntimeEvidence.AllowedTrace == nil || after.RuntimeEvidence.AllowedTrace.CapabilityID != updateTicket.ID ||
		after.RuntimeEvidence.DeniedTrace == nil || after.RuntimeEvidence.DeniedTrace.CapabilityID != exportContracts.ID ||
		after.AuditEvidence.AppliedEvent == nil || after.AuditEvidence.AppliedEvent.ResourceID != applied.Application.ID {
		t.Fatalf("expected runtime and audit evidence, got runtime=%#v audit=%#v", after.RuntimeEvidence, after.AuditEvidence)
	}
	afterReportResp := request(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(input, "", "user:support-001"), nil, "")
	afterReport := decodeData[permissionPackageProductionEvidenceReportResponse](t, afterReportResp)
	if afterReport.ReportVersion != "production-readiness-report/v1" || afterReport.Status != "ready" ||
		afterReport.Evidence.Application.ID != applied.Application.ID ||
		afterReport.Evidence.Application.TemplateVersion != 2 ||
		len(afterReport.Evidence.Application.AllowedCapabilityIDs) != len(applied.Application.AllowedCapabilityIDs) ||
		afterReport.Evidence.Runtime.AllowedTraceID == "" || afterReport.Evidence.Runtime.DeniedTraceID == "" ||
		afterReport.Evidence.Audit.AppliedEventID == "" ||
		afterReport.NextActionCode != "export_acceptance_report" ||
		afterReport.Evidence.AccessProfile.Present != true ||
		afterReport.ReadinessGeneratedAt == "" {
		t.Fatalf("expected ready production evidence report after evidence, got %#v", afterReport)
	}
	if afterReport.PlatformContract.APIVersion != beforeReport.PlatformContract.APIVersion ||
		afterReport.PlatformContract.ManagementMcpToolCatalog.MetadataVersion != beforeReport.PlatformContract.ManagementMcpToolCatalog.MetadataVersion ||
		afterReport.PlatformContract.ManagementMcpToolCatalog.CatalogDigest != beforeReport.PlatformContract.ManagementMcpToolCatalog.CatalogDigest {
		t.Fatalf("expected stable production report platform contract, before=%#v after=%#v", beforeReport.PlatformContract, afterReport.PlatformContract)
	}
	if afterReport.GeneratedBy != beforeReport.GeneratedBy {
		t.Fatalf("expected stable production report generatedBy, before=%q after=%q", beforeReport.GeneratedBy, afterReport.GeneratedBy)
	}
	assertProductionReportDigest(t, afterReportResp, afterReport.ReportDigest)
	for _, check := range []string{
		"application_present",
		"application_health_ready",
		"impact_ready",
		"access_profile_chain_present",
		"runtime_allowed_trace_present",
		"runtime_denied_trace_present",
		"applied_audit_event_present",
	} {
		if !permissionPackageProductionReadinessHasCheck(after.Checks, check, "passed") {
			t.Fatalf("expected readiness check %q to pass, got %#v", check, after.Checks)
		}
	}
	limitedTracePath := permissionPackageProductionReadinessPath(input, "", "user:support-001") + "&traceLimit=1"
	limitedTrace := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, limitedTracePath, nil, ""))
	if limitedTrace.Status != "blocked" || limitedTrace.RuntimeEvidence.AllowedTrace == nil || limitedTrace.RuntimeEvidence.DeniedTrace != nil ||
		limitedTrace.NextActionCode != "run_denied_runtime_call" ||
		!permissionPackageProductionReadinessHasCheck(limitedTrace.Checks, "runtime_denied_trace_present", "blocking") {
		t.Fatalf("expected traceLimit=1 to inspect only latest trace and block on missing denied evidence, got %#v", limitedTrace)
	}
	allowedIDs := map[string]struct{}{}
	for _, capability := range applied.Draft.AllowedCapabilities {
		allowedIDs[capability.ID] = struct{}{}
	}
	if len(allowedIDs) != 1 {
		t.Fatalf("expected support package to include one domain-matched capability, got %#v", applied.Draft.AllowedCapabilities)
	}
	if _, ok := allowedIDs[searchCustomer.ID]; ok {
		t.Fatalf("expected support package to block CRM search capability, got %#v", applied.Draft.AllowedCapabilities)
	}
	if _, ok := allowedIDs[updateTicket.ID]; !ok {
		t.Fatalf("expected support package to include update capability, got %#v", applied.Draft.AllowedCapabilities)
	}
}

func TestPermissionPackageProductionReportStampsAuthenticatedAdminActor(t *testing.T) {
	repo := store.NewMemory()
	const platformKey = "platform-operator-key"
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{
			Actor: "platform-operator",
			Key:   platformKey,
			Role:  string(domain.AdminIdentityRolePlatformAdmin),
		},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

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
	reportResp := requestWithAdmin(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(input, "", "user:support-001"), nil, "", platformKey)
	report := decodeData[permissionPackageProductionEvidenceReportResponse](t, reportResp)
	if report.GeneratedBy != "platform-operator" {
		t.Fatalf("expected production report generatedBy=%q, got %q", "platform-operator", report.GeneratedBy)
	}
	assertProductionReportDigest(t, reportResp, report.ReportDigest)
}
