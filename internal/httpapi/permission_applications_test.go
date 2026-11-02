package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestPermissionPackageApplicationImpactReportsDriftBlockers(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	target := createDirectAgent(t, repo, "Drift MCP Target", "tenant-east", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	caller := createDirectAgent(t, repo, "Drift Caller", "tenant-east", "ws-sales", "local", domain.AgentStatusActive, nil)
	entitlement, err := repo.CreateTenantEntitlement(t.Context(), domain.TenantEntitlement{
		ID:           "ent-disabled-drift",
		TenantID:     "tenant-east",
		TargetID:     target.ID,
		CapabilityID: "cap-drift",
		Effect:       domain.PolicyEffectAllow,
		Status:       domain.PolicyStatusDisabled,
		DataScopes:   []domain.DataScope{{DataDomain: "crm", Region: "华东"}},
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create disabled entitlement: %v", err)
	}
	workspaceAssignment, err := repo.CreateWorkspaceAssignment(t.Context(), domain.WorkspaceAssignment{
		ID:                  "wsa-disabled-drift",
		TenantEntitlementID: entitlement.ID,
		TenantID:            "tenant-east",
		WorkspaceID:         "ws-sales",
		Effect:              domain.PolicyEffectAllow,
		Status:              domain.PolicyStatusDisabled,
		DataScopes:          []domain.DataScope{{DataDomain: "crm", Region: "华东"}},
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		t.Fatalf("create disabled workspace assignment: %v", err)
	}
	instanceAssignment, err := repo.CreateInstanceAssignment(t.Context(), domain.InstanceAssignment{
		ID:                    "ina-disabled-drift",
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-sales",
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:sales-*",
		Effect:                domain.PolicyEffectAllow,
		Status:                domain.PolicyStatusDisabled,
		DataScopes:            []domain.DataScope{{DataDomain: "crm", Region: "华东"}},
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err != nil {
		t.Fatalf("create disabled instance assignment: %v", err)
	}
	application, err := repo.CreatePermissionPackageApplication(t.Context(), domain.PermissionPackageApplication{
		ID:                     "ppa-drift",
		DraftID:                "draft-drift",
		TemplateID:             "sales-readonly",
		TemplateVersion:        1,
		TenantID:               "tenant-east",
		WorkspaceID:            "ws-sales",
		TargetID:               target.ID,
		CallerInstanceID:       caller.ID,
		SubjectSelector:        "user:sales-*",
		RequestText:            "drift review",
		Region:                 "华东",
		DataScopes:             []domain.DataScope{{DataDomain: "crm", Region: "华东"}},
		AllowedCapabilityIDs:   []string{},
		AllowedCapabilityKeys:  []string{},
		TenantEntitlementIDs:   []string{entitlement.ID},
		WorkspaceAssignmentIDs: []string{"wsa-missing-drift", workspaceAssignment.ID},
		InstanceAssignmentIDs:  []string{instanceAssignment.ID},
		AppliedAt:              now,
	})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	impact := decodeData[permissionPackageApplicationImpactResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales", nil, ""))
	if impact.Summary.CreatedObjectCount != 4 || impact.Summary.ActiveObjectCount != 0 ||
		impact.Summary.MissingObjectCount != 1 || impact.Summary.RollbackReady {
		t.Fatalf("unexpected drift impact summary: %#v", impact.Summary)
	}
	health := decodeData[permissionPackageApplicationHealthResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-root&workspaceId=ws-sales&limit=10", nil, ""))
	if health.Summary.Total != 1 || health.Summary.Ready != 0 || health.Summary.Drifted != 1 || health.Summary.NeedsReview != 0 {
		t.Fatalf("expected drifted application health summary, got %#v", health.Summary)
	}
	if len(health.Applications) != 1 {
		t.Fatalf("expected one drifted application health row, got %#v", health.Applications)
	}
	healthRow := health.Applications[0]
	if healthRow.Application.ID != application.ID || healthRow.Status != "drifted" || healthRow.CreatedObjectCount != 4 ||
		healthRow.ActiveObjectCount != 0 || healthRow.MissingObjectCount != 1 || healthRow.RollbackReady {
		t.Fatalf("unexpected drifted application health row: %#v", healthRow)
	}
	for _, code := range []string{"missing_created_objects", "inactive_created_objects"} {
		if !containsString(healthRow.BlockerCodes, code) {
			t.Fatalf("expected drift health blocker code %q, got %#v", code, healthRow.BlockerCodes)
		}
	}
	for _, code := range []string{"missing_created_objects", "inactive_created_objects", "no_allowed_capabilities"} {
		if !containsString(impact.RollbackReview.BlockerCodes, code) {
			t.Fatalf("expected rollback blocker code %q, got %#v", code, impact.RollbackReview.BlockerCodes)
		}
		if !containsString(impact.RemediationPlan.BlockerCodes, code) {
			t.Fatalf("expected remediation blocker code %q, got %#v", code, impact.RemediationPlan.BlockerCodes)
		}
	}
	if impact.RollbackReview.Ready || impact.RemediationPlan.Ready {
		t.Fatalf("drift impact should not be ready: rollback=%#v remediation=%#v", impact.RollbackReview, impact.RemediationPlan)
	}
	if impact.RollbackReview.BlockerCodes == nil || impact.RemediationPlan.BlockerCodes == nil {
		t.Fatalf("expected blocker codes to encode as arrays: rollback=%#v remediation=%#v", impact.RollbackReview.BlockerCodes, impact.RemediationPlan.BlockerCodes)
	}
	if !remediationActionsContain(impact.RemediationPlan.Actions, "tenant_entitlement", entitlement.ID, "investigate") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "workspace_assignment", "wsa-missing-drift", "investigate") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "workspace_assignment", workspaceAssignment.ID, "investigate") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "instance_assignment", instanceAssignment.ID, "investigate") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "access_decision", application.ID, "verify") {
		t.Fatalf("expected drift remediation investigate and verify actions, got %#v", impact.RemediationPlan.Actions)
	}
	for _, action := range impact.RemediationPlan.Actions {
		if !action.ReadOnly {
			t.Fatalf("expected drift remediation actions to remain read-only, got %#v", action)
		}
	}
}

func TestPermissionPackageApplicationImpactRedactsOutOfScopeCapabilityHydration(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	application, err := repo.CreatePermissionPackageApplication(t.Context(), domain.PermissionPackageApplication{
		ID:                    "ppa-cross-capability",
		DraftID:               "draft-cross-capability",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              supportTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		SubjectSelector:       "role:support",
		RequestText:           "dirty application review",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{"export_invoices"},
		AppliedAt:             now,
	})
	if err != nil {
		t.Fatalf("create dirty application: %v", err)
	}

	impact := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+application.ID+"/impact?tenantId=tenant-east&workspaceId=ws-support", nil, "", "support-key")
	if impact.Code != http.StatusOK {
		t.Fatalf("scoped admin should read own application impact, got %d body=%s", impact.Code, impact.Body.String())
	}
	health := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-east&workspaceId=ws-support&limit=10", nil, "", "support-key")
	if health.Code != http.StatusOK {
		t.Fatalf("scoped admin should read own application health, got %d body=%s", health.Code, health.Body.String())
	}
	for _, resp := range []*httptest.ResponseRecorder{impact, health} {
		body := resp.Body.String()
		if strings.Contains(body, "export_invoices") || strings.Contains(body, "Finance MCP") {
			t.Fatalf("scoped application response leaked out-of-scope hydrated capability details: %s", body)
		}
	}
}

func TestPermissionPackageApplicationReadsRedactDirtyApplicationCapabilityScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	dirtyCapabilityApplication, err := repo.CreatePermissionPackageApplication(t.Context(), domain.PermissionPackageApplication{
		ID:                    "ppa-dirty-capability",
		DraftID:               "draft-dirty-capability",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              supportTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		SubjectSelector:       "role:support",
		RequestText:           "dirty capability application",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{financeCapability.Key},
		AppliedAt:             now,
	})
	if err != nil {
		t.Fatalf("create dirty capability application: %v", err)
	}
	dirtyTargetApplication, err := repo.CreatePermissionPackageApplication(t.Context(), domain.PermissionPackageApplication{
		ID:                    "ppa-dirty-target",
		DraftID:               "draft-dirty-target",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              financeTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		SubjectSelector:       "role:support",
		RequestText:           "dirty target application",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "finance", Region: "eu-west"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{financeCapability.Key},
		AppliedAt:             now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("create dirty target application: %v", err)
	}
	if _, err := repo.AppendAuditEvent(t.Context(), domain.AuditEvent{
		ID:           "aud-dirty-application",
		TenantID:     "tenant-east",
		WorkspaceID:  "ws-support",
		Actor:        "platform",
		Action:       "permission_package.applied",
		ResourceType: "permission_package",
		ResourceID:   dirtyCapabilityApplication.ID,
		Summary:      "Permission package applied",
		Metadata: map[string]any{
			"applicationId":          dirtyCapabilityApplication.ID,
			"draftId":                dirtyCapabilityApplication.DraftID,
			"templateId":             dirtyCapabilityApplication.TemplateID,
			"templateVersion":        dirtyCapabilityApplication.TemplateVersion,
			"targetId":               financeTarget.ID,
			"callerInstanceId":       supportCaller.ID,
			"requestedCapabilityId":  financeCapability.ID,
			"subjectSelector":        dirtyCapabilityApplication.SubjectSelector,
			"allowedCapabilityIds":   []string{financeCapability.ID},
			"allowedCapabilityKeys":  []string{financeCapability.Key},
			"tenantEntitlementIds":   []string{"tte-cross-scope"},
			"workspaceAssignmentIds": []string{"wsa-cross-scope"},
			"instanceAssignmentIds":  []string{"ias-cross-scope"},
		},
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("append dirty applied audit event: %v", err)
	}

	leaks := []string{financeTarget.ID, financeTarget.Name, financeCapability.ID, financeCapability.Key, "ws-finance"}
	listPath := "/api/v1/permission-packages/applications?tenantId=tenant-east&workspaceId=ws-support&templateId=support-ticket-triage&targetId=" + supportTarget.ID + "&callerInstanceId=" + supportCaller.ID + "&limit=10"
	listResp := requestWithAdmin(t, router, http.MethodGet, listPath, nil, "", "support-key")
	if listResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read in-scope application list, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	assertResponseDoesNotContain(t, listResp.Body.String(), leaks...)
	applications := decodeData[[]permissionPackageApplicationResponse](t, listResp)
	if len(applications) != 1 || applications[0].ID != dirtyCapabilityApplication.ID {
		t.Fatalf("expected scoped application list to keep the in-scope application only, got %#v", applications)
	}
	if len(applications[0].AllowedCapabilityIDs) != 0 || len(applications[0].AllowedCapabilityKeys) != 0 {
		t.Fatalf("expected dirty out-of-scope capabilities to be redacted from application list, got %#v", applications[0])
	}
	dirtyRequestedList := decodeData[[]permissionPackageApplicationResponse](t, requestWithAdmin(t, router, http.MethodGet, listPath+"&requestedCapabilityId="+url.QueryEscape(financeCapability.ID), nil, "", "support-key"))
	if len(dirtyRequestedList) != 0 {
		t.Fatalf("requested capability filter must not reveal a redacted capability reference: %#v", dirtyRequestedList)
	}

	mcpList := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "dirty-application-list",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_applications",
			"arguments": map[string]any{
				"callerInstanceId": supportCaller.ID,
				"limit":            10,
				"targetId":         supportTarget.ID,
				"templateId":       "support-ticket-triage",
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-support",
			},
		},
	}, "", "support-key"))
	assertResponseDoesNotContain(t, string(mcpList.Result.StructuredContent), leaks...)
	var mcpApplications []permissionPackageApplicationResponse
	if err := json.Unmarshal(mcpList.Result.StructuredContent, &mcpApplications); err != nil {
		t.Fatalf("decode management MCP application list: %v", err)
	}
	if len(mcpApplications) != 1 || mcpApplications[0].ID != dirtyCapabilityApplication.ID ||
		len(mcpApplications[0].AllowedCapabilityIDs) != 0 || len(mcpApplications[0].AllowedCapabilityKeys) != 0 {
		t.Fatalf("expected MCP application list to redact dirty out-of-scope capabilities, got %#v", mcpApplications)
	}
	dirtyRequestedMCP := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "dirty-requested-application-list",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_applications",
			"arguments": map[string]any{
				"callerInstanceId":      supportCaller.ID,
				"limit":                 10,
				"requestedCapabilityId": financeCapability.ID,
				"targetId":              supportTarget.ID,
				"templateId":            "support-ticket-triage",
				"tenantId":              "tenant-east",
				"workspaceId":           "ws-support",
			},
		},
	}, "", "support-key"))
	var dirtyRequestedMCPApplications []permissionPackageApplicationResponse
	if err := json.Unmarshal(dirtyRequestedMCP.Result.StructuredContent, &dirtyRequestedMCPApplications); err != nil || len(dirtyRequestedMCPApplications) != 0 {
		t.Fatalf("MCP requested capability filter must not reveal a redacted reference: rows=%#v err=%v", dirtyRequestedMCPApplications, err)
	}

	healthResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-east&workspaceId=ws-support&limit=10", nil, "", "support-key")
	if healthResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read application health, got %d body=%s", healthResp.Code, healthResp.Body.String())
	}
	assertResponseDoesNotContain(t, healthResp.Body.String(), append(leaks, dirtyTargetApplication.ID)...)
	health := decodeData[permissionPackageApplicationHealthResponse](t, healthResp)
	if health.Summary.Total != 1 || len(health.Applications) != 1 || health.Applications[0].Application.ID != dirtyCapabilityApplication.ID {
		t.Fatalf("expected application health to hide dirty target applications, got %#v", health)
	}
	dirtyRequestedHealth := decodeData[permissionPackageApplicationHealthResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-east&workspaceId=ws-support&requestedCapabilityId="+url.QueryEscape(financeCapability.ID), nil, "", "support-key"))
	if dirtyRequestedHealth.Summary.Total != 0 || len(dirtyRequestedHealth.Applications) != 0 {
		t.Fatalf("application health filter must not reveal a redacted capability reference: %#v", dirtyRequestedHealth)
	}

	input := map[string]any{
		"callerInstanceId": supportCaller.ID,
		"region":           "us-east",
		"requestText":      "validate support package status",
		"subjectSelector":  "role:support",
		"targetId":         supportTarget.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	readinessResp := requestWithAdmin(t, router, http.MethodGet, permissionPackageProductionReadinessPath(input, "", "user:support-001"), nil, "", "support-key")
	if readinessResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read production readiness, got %d body=%s", readinessResp.Code, readinessResp.Body.String())
	}
	assertResponseDoesNotContain(t, readinessResp.Body.String(), leaks...)
	readiness := decodeData[permissionPackageProductionReadinessResponse](t, readinessResp)
	if readiness.LatestApplication == nil || readiness.LatestApplication.ID != dirtyCapabilityApplication.ID {
		t.Fatalf("expected readiness to keep the in-scope latest application, got %#v", readiness.LatestApplication)
	}
	if len(readiness.LatestApplication.AllowedCapabilityIDs) != 0 || len(readiness.LatestApplication.AllowedCapabilityKeys) != 0 {
		t.Fatalf("expected readiness latest application to redact out-of-scope capabilities, got %#v", readiness.LatestApplication)
	}

	reportResp := requestWithAdmin(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(input, "", "user:support-001"), nil, "", "support-key")
	if reportResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read production report, got %d body=%s", reportResp.Code, reportResp.Body.String())
	}
	assertResponseDoesNotContain(t, reportResp.Body.String(), leaks...)
	report := decodeData[permissionPackageProductionEvidenceReportResponse](t, reportResp)
	if report.Evidence.Application.ID != dirtyCapabilityApplication.ID || len(report.Evidence.Application.AllowedCapabilityIDs) != 0 {
		t.Fatalf("expected production report application evidence to redact out-of-scope capabilities, got %#v", report.Evidence.Application)
	}

	auditResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/audit/events?tenantId=tenant-east&workspaceId=ws-support&action=permission_package.applied&resourceId="+dirtyCapabilityApplication.ID, nil, "", "support-key")
	if auditResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read applied audit event, got %d body=%s", auditResp.Code, auditResp.Body.String())
	}
	assertResponseDoesNotContain(t, auditResp.Body.String(), leaks...)
	auditEvents := decodeData[[]auditEventResponse](t, auditResp)
	if len(auditEvents) != 1 {
		t.Fatalf("expected one scoped applied audit event, got %#v", auditEvents)
	}
	if auditEvents[0].Metadata["targetId"] != supportTarget.ID || auditEvents[0].Metadata["callerInstanceId"] != supportCaller.ID {
		t.Fatalf("expected applied audit metadata to be rebuilt from visible application scope, got %#v", auditEvents[0].Metadata)
	}
	if raw, ok := auditEvents[0].Metadata["allowedCapabilityIds"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("expected applied audit metadata to redact out-of-scope capability ids, got %#v", auditEvents[0].Metadata)
	}
	if raw, ok := auditEvents[0].Metadata["allowedCapabilityKeys"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("expected applied audit metadata to redact out-of-scope capability keys, got %#v", auditEvents[0].Metadata)
	}
	if readiness.AuditEvidence.AppliedEvent == nil {
		t.Fatalf("expected readiness to include scoped applied audit event")
	}
	if readiness.AuditEvidence.AppliedEvent.Metadata["targetId"] != supportTarget.ID {
		t.Fatalf("expected readiness applied audit event to use visible target, got %#v", readiness.AuditEvidence.AppliedEvent.Metadata)
	}
	if raw, ok := readiness.AuditEvidence.AppliedEvent.Metadata["allowedCapabilityIds"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("expected readiness applied audit event to redact out-of-scope capability ids, got %#v", readiness.AuditEvidence.AppliedEvent.Metadata)
	}
	mcpReadiness := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "dirty-production-readiness",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "check_permission_package_production_readiness",
			"arguments": input,
		},
	}, "", "support-key"))
	assertResponseDoesNotContain(t, string(mcpReadiness.Result.StructuredContent), leaks...)

	targetImpactResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+dirtyTargetApplication.ID+"/impact?tenantId=tenant-east&workspaceId=ws-support", nil, "", "support-key")
	if targetImpactResp.Code != http.StatusNotFound || strings.Contains(targetImpactResp.Body.String(), financeTarget.ID) ||
		strings.Contains(targetImpactResp.Body.String(), financeCapability.ID) || strings.Contains(targetImpactResp.Body.String(), financeCapability.Key) {
		t.Fatalf("dirty target application impact should be hidden without leaking finance details, got %d body=%s", targetImpactResp.Code, targetImpactResp.Body.String())
	}
}
