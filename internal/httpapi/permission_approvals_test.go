package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestPermissionPackageDraftAndApplyManagement(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-biz", "tenant-root", "Business tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-biz", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	export := createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	templates := decodeData[[]permissionPackageTemplateResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/templates", nil, ""))
	if len(templates) == 0 || templates[0].ID != "sales-readonly" || templates[0].Version != 2 {
		t.Fatalf("expected sales-readonly template first, got %#v", templates)
	}
	accessSubjects := decodeData[[]domain.PermissionPackageAccessSubject](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/access-subjects", nil, ""))
	if len(accessSubjects) == 0 || accessSubjects[0].ID != "role:support-agent" || accessSubjects[0].SubjectSelector != "user:support-*" {
		t.Fatalf("expected support-agent access subject first, got %#v", accessSubjects)
	}
	var registeredMember domain.PermissionPackageAccessSubject
	for _, subject := range accessSubjects {
		if subject.ID == "member:support-002" {
			registeredMember = subject
			break
		}
	}
	if registeredMember.SubjectSelector != "user:support-002" || registeredMember.WorkspaceID != "ws-permission-package-approval" {
		t.Fatalf("expected registered support member with workspace scope, got %#v", registeredMember)
	}

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读，禁止导出合同。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	draft := decodeData[permissionPackageDraftResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, ""))
	if !draft.Readiness.CanApply || draft.Template.ID != "sales-readonly" || draft.Template.Version != 2 {
		t.Fatalf("expected applicable sales-readonly draft, got %#v", draft)
	}
	if draft.PolicyGate.Decision != "allow" || !draft.PolicyGate.CanApplyDirectly || draft.PolicyGate.PolicyVersion != 1 {
		t.Fatalf("expected direct-apply policy gate, got %#v", draft.PolicyGate)
	}
	directApprovalResp := request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "")
	if directApprovalResp.Code != http.StatusBadRequest {
		t.Fatalf("direct-apply package should not create approval request, status=%d body=%s", directApprovalResp.Code, directApprovalResp.Body.String())
	}
	if len(draft.AllowedCapabilities) != 1 || draft.AllowedCapabilities[0].ID != search.ID {
		t.Fatalf("expected search_customer allowed, got %#v", draft.AllowedCapabilities)
	}
	if len(draft.BlockedCapabilities) != 1 || draft.BlockedCapabilities[0].ID != export.ID {
		t.Fatalf("expected export_contracts blocked, got %#v", draft.BlockedCapabilities)
	}
	if len(draft.DataScopes) != 1 || draft.DataScopes[0].Region != "华东" || draft.DataScopes[0].TenantFilter != "tenant_id = 'tenant-east'" {
		t.Fatalf("unexpected draft data scopes: %#v", draft.DataScopes)
	}
	if len(draft.SimulationRows) != 4 || draft.SimulationRows[0].ExpectedDecision != "allow" || draft.SimulationRows[1].ExpectedDecision != "deny" {
		t.Fatalf("unexpected simulation rows: %#v", draft.SimulationRows)
	}

	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, ""))
	if len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != search.ID {
		t.Fatalf("expected one applied tenant entitlement for search, got %#v", applied.TenantEntitlements)
	}
	if len(applied.WorkspaceAssignments) != 1 || applied.WorkspaceAssignments[0].WorkspaceID != "ws-sales" {
		t.Fatalf("expected one workspace assignment, got %#v", applied.WorkspaceAssignments)
	}
	if len(applied.InstanceAssignments) != 1 || applied.InstanceAssignments[0].CallerInstanceID != caller.ID {
		t.Fatalf("expected one caller instance assignment, got %#v", applied.InstanceAssignments)
	}
	if applied.Application == nil || applied.Application.DraftID != applied.Draft.ID ||
		applied.Application.TemplateID != "sales-readonly" || applied.Application.TemplateVersion != 2 ||
		applied.Application.TenantID != "tenant-east" || applied.Application.WorkspaceID != "ws-sales" ||
		applied.Application.TargetID != target.ID || applied.Application.CallerInstanceID != caller.ID ||
		len(applied.Application.AllowedCapabilityIDs) != 1 || applied.Application.AllowedCapabilityIDs[0] != search.ID ||
		len(applied.Application.TenantEntitlementIDs) != 1 || applied.Application.TenantEntitlementIDs[0] != applied.TenantEntitlements[0].ID ||
		len(applied.Application.WorkspaceAssignmentIDs) != 1 || applied.Application.WorkspaceAssignmentIDs[0] != applied.WorkspaceAssignments[0].ID ||
		len(applied.Application.InstanceAssignmentIDs) != 1 || applied.Application.InstanceAssignmentIDs[0] != applied.InstanceAssignments[0].ID ||
		len(applied.Application.DataScopes) != 1 || applied.Application.DataScopes[0].Region != "华东" {
		t.Fatalf("unexpected permission package application: %#v", applied.Application)
	}
	applications := decodeData[[]permissionPackageApplicationResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?tenantId=tenant-root&workspaceId=ws-sales&templateId=sales-readonly&targetId="+target.ID+"&callerInstanceId="+caller.ID+"&limit=1", nil, ""))
	if len(applications) != 1 || applications[0].ID != applied.Application.ID || applications[0].DraftID != applied.Draft.ID {
		t.Fatalf("expected listed application record, got %#v", applications)
	}
	preflightAfterApply := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if preflightAfterApply.Summary.CanApply || preflightAfterApply.Summary.BlockingCount == 0 ||
		!permissionPackagePreflightHasCheck(preflightAfterApply.Checks, "application_already_applied", "blocking") {
		t.Fatalf("expected already-applied preflight to block direct reapply, got %#v", preflightAfterApply)
	}
	if !slices.Contains(preflightAfterApply.NextActionCodes, "review_current_application") {
		t.Fatalf("expected already-applied preflight next action code, got %#v", preflightAfterApply.NextActionCodes)
	}
	applicationsAfterPreflight := decodeData[[]permissionPackageApplicationResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?tenantId=tenant-root&workspaceId=ws-sales&templateId=sales-readonly&targetId="+target.ID+"&callerInstanceId="+caller.ID, nil, ""))
	if len(applicationsAfterPreflight) != 1 || applicationsAfterPreflight[0].ID != applied.Application.ID {
		t.Fatalf("already-applied preflight must not create another application, got %#v", applicationsAfterPreflight)
	}
	duplicateApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, "")
	if duplicateApply.Code != http.StatusConflict {
		t.Fatalf("duplicate direct apply should be rejected, status=%d body=%s", duplicateApply.Code, duplicateApply.Body.String())
	}
	var duplicateApplyErr apiEnvelope
	if err := json.Unmarshal(duplicateApply.Body.Bytes(), &duplicateApplyErr); err != nil {
		t.Fatalf("decode duplicate apply error: %v body=%s", err, duplicateApply.Body.String())
	}
	if duplicateApplyErr.Error != "PERMISSION_PACKAGE_ALREADY_APPLIED" {
		t.Fatalf("duplicate direct apply should return stable app code, got %#v", duplicateApplyErr)
	}
	applicationsAfterDuplicate := decodeData[[]permissionPackageApplicationResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?tenantId=tenant-root&workspaceId=ws-sales&templateId=sales-readonly&targetId="+target.ID+"&callerInstanceId="+caller.ID, nil, ""))
	if len(applicationsAfterDuplicate) != 1 || applicationsAfterDuplicate[0].ID != applied.Application.ID {
		t.Fatalf("duplicate direct apply should not create another application, got %#v", applicationsAfterDuplicate)
	}
	appliedEvents, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.applied"})
	if err != nil || len(appliedEvents) != 1 || appliedEvents[0].ResourceID != applied.Application.ID {
		t.Fatalf("duplicate direct apply should not create another audit event, events=%#v err=%v", appliedEvents, err)
	}
	health := decodeData[permissionPackageApplicationHealthResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-root&workspaceId=ws-sales&templateId=sales-readonly&targetId="+target.ID+"&callerInstanceId="+caller.ID+"&limit=10", nil, ""))
	if health.Summary.Total != 1 || health.Summary.Ready != 1 || health.Summary.Drifted != 0 || health.Summary.NeedsReview != 0 {
		t.Fatalf("expected ready application health summary, got %#v", health.Summary)
	}
	if len(health.Applications) != 1 {
		t.Fatalf("expected one application health row, got %#v", health.Applications)
	}
	healthRow := health.Applications[0]
	if healthRow.Application.ID != applied.Application.ID || healthRow.Application.DraftID != applied.Draft.ID ||
		healthRow.Status != "ready" || healthRow.CreatedObjectCount != 3 || healthRow.ActiveObjectCount != 3 ||
		healthRow.MissingObjectCount != 0 || !healthRow.RollbackReady {
		t.Fatalf("unexpected ready application health row: %#v", healthRow)
	}
	if healthRow.BlockerCodes == nil || len(healthRow.BlockerCodes) != 0 {
		t.Fatalf("expected ready health blocker codes to encode as an empty array, got %#v", healthRow.BlockerCodes)
	}
	impact := decodeData[permissionPackageApplicationImpactResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+applied.Application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales", nil, ""))
	if impact.Application.ID != applied.Application.ID || impact.Application.DraftID != applied.Draft.ID {
		t.Fatalf("expected impact for applied application, got %#v", impact.Application)
	}
	if impact.Rehearsal != nil {
		t.Fatalf("real impact should not include rehearsal metadata, got %#v", impact.Rehearsal)
	}
	if impact.Summary.CreatedObjectCount != 3 || impact.Summary.ActiveObjectCount != 3 ||
		impact.Summary.MissingObjectCount != 0 || !impact.Summary.RollbackReady {
		t.Fatalf("unexpected application impact summary: %#v", impact.Summary)
	}
	if !impactObjectsContain(impact.CreatedObjects, "tenant_entitlement", applied.TenantEntitlements[0].ID, "enabled", "disable") ||
		!impactObjectsContain(impact.CreatedObjects, "workspace_assignment", applied.WorkspaceAssignments[0].ID, "enabled", "disable") ||
		!impactObjectsContain(impact.CreatedObjects, "instance_assignment", applied.InstanceAssignments[0].ID, "enabled", "disable") {
		t.Fatalf("expected created grant objects in impact review, got %#v", impact.CreatedObjects)
	}
	if len(impact.CapabilityReviews) != 1 || impact.CapabilityReviews[0].ID != search.ID ||
		impact.CapabilityReviews[0].CurrentStatus != string(domain.CapabilityDiscoveryApproved) ||
		impact.CapabilityReviews[0].RollbackAction != "manual_review" {
		t.Fatalf("expected capability manual review row, got %#v", impact.CapabilityReviews)
	}
	if !impact.RollbackReview.Ready || len(impact.RollbackReview.Blockers) != 0 || len(impact.RollbackReview.Steps) == 0 {
		t.Fatalf("expected ready rollback review steps, got %#v", impact.RollbackReview)
	}
	if impact.RollbackReview.Blockers == nil {
		t.Fatalf("expected rollback blockers to encode as an empty array, got nil")
	}
	if impact.RollbackReview.BlockerCodes == nil || len(impact.RollbackReview.BlockerCodes) != 0 {
		t.Fatalf("expected rollback blocker codes to encode as an empty array, got %#v", impact.RollbackReview.BlockerCodes)
	}
	if impact.RemediationPlan.ExecutionMode != "read_only" || !impact.RemediationPlan.Ready {
		t.Fatalf("expected ready read-only remediation plan, got %#v", impact.RemediationPlan)
	}
	if impact.RemediationPlan.Blockers == nil || len(impact.RemediationPlan.Blockers) != 0 {
		t.Fatalf("expected remediation blockers to encode as an empty array, got %#v", impact.RemediationPlan.Blockers)
	}
	if impact.RemediationPlan.BlockerCodes == nil || len(impact.RemediationPlan.BlockerCodes) != 0 {
		t.Fatalf("expected remediation blocker codes to encode as an empty array, got %#v", impact.RemediationPlan.BlockerCodes)
	}
	if len(impact.RemediationPlan.Actions) == 0 {
		t.Fatalf("expected remediation actions, got %#v", impact.RemediationPlan)
	}
	for _, action := range impact.RemediationPlan.Actions {
		if !action.ReadOnly {
			t.Fatalf("expected all remediation actions to be read-only, got %#v in %#v", action, impact.RemediationPlan.Actions)
		}
	}
	if !remediationActionsContain(impact.RemediationPlan.Actions, "capability", search.ID, "manual_review") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "instance_assignment", applied.InstanceAssignments[0].ID, "disable") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "workspace_assignment", applied.WorkspaceAssignments[0].ID, "disable") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "tenant_entitlement", applied.TenantEntitlements[0].ID, "disable") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "access_decision", applied.Application.ID, "verify") {
		t.Fatalf("expected complete remediation action sequence, got %#v", impact.RemediationPlan.Actions)
	}
	instanceOrder := remediationActionOrder(impact.RemediationPlan.Actions, "instance_assignment", applied.InstanceAssignments[0].ID, "disable")
	workspaceOrder := remediationActionOrder(impact.RemediationPlan.Actions, "workspace_assignment", applied.WorkspaceAssignments[0].ID, "disable")
	tenantOrder := remediationActionOrder(impact.RemediationPlan.Actions, "tenant_entitlement", applied.TenantEntitlements[0].ID, "disable")
	if instanceOrder == 0 || workspaceOrder == 0 || tenantOrder == 0 || !(instanceOrder < workspaceOrder && workspaceOrder < tenantOrder) {
		t.Fatalf("expected instance before workspace before tenant remediation order, got actions %#v", impact.RemediationPlan.Actions)
	}
	driftRehearsal := decodeData[permissionPackageApplicationImpactResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+applied.Application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales&rehearsal=grant_drift", nil, ""))
	if driftRehearsal.Rehearsal == nil || !driftRehearsal.Rehearsal.Enabled || driftRehearsal.Rehearsal.Scenario != "grant_drift" {
		t.Fatalf("expected grant_drift rehearsal metadata, got %#v", driftRehearsal.Rehearsal)
	}
	if driftRehearsal.Summary.CreatedObjectCount != 3 || driftRehearsal.Summary.ActiveObjectCount >= driftRehearsal.Summary.CreatedObjectCount ||
		driftRehearsal.Summary.MissingObjectCount != 1 || driftRehearsal.Summary.RollbackReady {
		t.Fatalf("unexpected rehearsal impact summary: %#v", driftRehearsal.Summary)
	}
	for _, code := range []string{"missing_created_objects", "inactive_created_objects"} {
		if !containsString(driftRehearsal.RollbackReview.BlockerCodes, code) {
			t.Fatalf("expected rehearsal rollback blocker code %q, got %#v", code, driftRehearsal.RollbackReview.BlockerCodes)
		}
		if !containsString(driftRehearsal.RemediationPlan.BlockerCodes, code) {
			t.Fatalf("expected rehearsal remediation blocker code %q, got %#v", code, driftRehearsal.RemediationPlan.BlockerCodes)
		}
	}
	if driftRehearsal.RollbackReview.Ready || driftRehearsal.RemediationPlan.Ready {
		t.Fatalf("rehearsal drift should not be ready: rollback=%#v remediation=%#v", driftRehearsal.RollbackReview, driftRehearsal.RemediationPlan)
	}
	if !remediationActionsContain(driftRehearsal.RemediationPlan.Actions, "workspace_assignment", applied.WorkspaceAssignments[0].ID, "investigate") ||
		!remediationActionsContain(driftRehearsal.RemediationPlan.Actions, "instance_assignment", applied.InstanceAssignments[0].ID, "investigate") ||
		!remediationActionsContain(driftRehearsal.RemediationPlan.Actions, "access_decision", applied.Application.ID, "verify") {
		t.Fatalf("expected rehearsal investigate and verify actions, got %#v", driftRehearsal.RemediationPlan.Actions)
	}
	for _, action := range driftRehearsal.RemediationPlan.Actions {
		if !action.ReadOnly {
			t.Fatalf("expected rehearsal actions to be read-only, got %#v in %#v", action, driftRehearsal.RemediationPlan.Actions)
		}
	}
	realImpactAfterRehearsal := decodeData[permissionPackageApplicationImpactResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+applied.Application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales", nil, ""))
	if realImpactAfterRehearsal.Rehearsal != nil {
		t.Fatalf("real impact after rehearsal should not include rehearsal metadata, got %#v", realImpactAfterRehearsal.Rehearsal)
	}
	if realImpactAfterRehearsal.Summary.CreatedObjectCount != 3 || realImpactAfterRehearsal.Summary.ActiveObjectCount != 3 ||
		realImpactAfterRehearsal.Summary.MissingObjectCount != 0 || !realImpactAfterRehearsal.Summary.RollbackReady {
		t.Fatalf("rehearsal should not persist grant drift, got real summary %#v", realImpactAfterRehearsal.Summary)
	}
	badRehearsal := request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+applied.Application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales&rehearsal=unknown", nil, "")
	if badRehearsal.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid rehearsal to fail, status=%d body=%s", badRehearsal.Code, badRehearsal.Body.String())
	}
	badHealthLimit := request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?limit=0", nil, "")
	if badHealthLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid application health limit to fail, status=%d body=%s", badHealthLimit.Code, badHealthLimit.Body.String())
	}
	badLimit := request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?limit=0", nil, "")
	if badLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid application limit to fail, status=%d body=%s", badLimit.Code, badLimit.Body.String())
	}
	updated, ok, err := repo.GetCapability(t.Context(), search.ID)
	if err != nil || !ok {
		t.Fatalf("get updated capability: ok=%v err=%v", ok, err)
	}
	if updated.DiscoveryStatus != domain.CapabilityDiscoveryApproved || len(updated.DataScopes) != 1 || updated.DataScopes[0].Region != "华东" {
		t.Fatalf("expected package apply to approve and scope capability, got %#v", updated)
	}
	events := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?action=permission_package.applied", nil, ""))
	if len(events) != 1 || events[0].TenantID != "tenant-east" || events[0].ResourceID != applied.Application.ID ||
		events[0].Metadata["applicationId"] != applied.Application.ID || events[0].Metadata["draftId"] != applied.Draft.ID ||
		events[0].Metadata["templateVersion"] != float64(2) {
		t.Fatalf("expected permission_package.applied audit event, got %#v", events)
	}
}

func TestPermissionPackageRequestedCapabilityStaysExactThroughApprovalAndApplication(t *testing.T) {
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
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	assignTicket := createDirectCapabilityWithAction(t, repo, target.ID, "assign_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId":      caller.ID,
		"region":                "us-east",
		"requestText":           "Repair only the denied ticket update capability.",
		"requestedCapabilityId": updateTicket.ID,
		"subjectSelector":       "user:support-*",
		"targetId":              target.ID,
		"templateId":            "support-ticket-triage",
		"tenantId":              "tenant-east",
		"workspaceId":           "ws-support",
	}
	draft := decodeData[permissionPackageDraftResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, ""))
	if !draft.Readiness.CanApply || len(draft.AllowedCapabilities) != 1 || draft.AllowedCapabilities[0].ID != updateTicket.ID {
		t.Fatalf("requested capability should be the only allowed capability: %#v", draft)
	}
	if len(draft.BlockedCapabilities) != 1 || draft.BlockedCapabilities[0].ID != assignTicket.ID {
		t.Fatalf("sibling capability should remain blocked evidence: %#v", draft.BlockedCapabilities)
	}

	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	if approval.RequestedCapabilityID != updateTicket.ID || len(approval.AllowedCapabilityIDs) != 1 || approval.AllowedCapabilityIDs[0] != updateTicket.ID {
		t.Fatalf("approval snapshot lost exact capability boundary: %#v", approval)
	}
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{"reviewer": "security"}, ""))

	mismatchedInput := map[string]any{
		"approvalRequestId":     approved.ID,
		"callerInstanceId":      caller.ID,
		"region":                input["region"],
		"requestText":           input["requestText"],
		"requestedCapabilityId": assignTicket.ID,
		"subjectSelector":       input["subjectSelector"],
		"targetId":              target.ID,
		"templateId":            input["templateId"],
		"tenantId":              input["tenantId"],
		"workspaceId":           input["workspaceId"],
	}
	mismatch := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", mismatchedInput, "")
	if mismatch.Code != http.StatusBadRequest || !strings.Contains(mismatch.Body.String(), "does not match current draft") {
		t.Fatalf("approval must not be reusable for another requested capability, status=%d body=%s", mismatch.Code, mismatch.Body.String())
	}

	applyInput := map[string]any{}
	for key, value := range input {
		applyInput[key] = value
	}
	applyInput["approvalRequestId"] = approved.ID
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, ""))
	if applied.Application == nil || applied.Application.RequestedCapabilityID != updateTicket.ID ||
		len(applied.Application.AllowedCapabilityIDs) != 1 || applied.Application.AllowedCapabilityIDs[0] != updateTicket.ID ||
		len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != updateTicket.ID {
		t.Fatalf("application lost exact capability boundary: %#v", applied)
	}

	requested := url.QueryEscape(updateTicket.ID)
	applications := decodeData[[]permissionPackageApplicationResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?tenantId=tenant-east&workspaceId=ws-support&requestedCapabilityId="+requested, nil, ""))
	if len(applications) != 1 || applications[0].ID != applied.Application.ID {
		t.Fatalf("application exact capability filter mismatch: %#v", applications)
	}
	otherApplications := decodeData[[]permissionPackageApplicationResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?tenantId=tenant-east&workspaceId=ws-support&requestedCapabilityId="+url.QueryEscape(assignTicket.ID), nil, ""))
	if len(otherApplications) != 0 {
		t.Fatalf("application filter leaked another requested capability: %#v", otherApplications)
	}
	mcpApplicationsResult := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "exact-capability-applications",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_applications",
			"arguments": map[string]any{
				"tenantId":              "tenant-east",
				"workspaceId":           "ws-support",
				"requestedCapabilityId": updateTicket.ID,
			},
		},
	}, ""))
	var mcpApplications []permissionPackageApplicationResponse
	if err := json.Unmarshal(mcpApplicationsResult.Result.StructuredContent, &mcpApplications); err != nil || len(mcpApplications) != 1 || mcpApplications[0].ID != applied.Application.ID {
		t.Fatalf("management MCP application filter lost exact capability: rows=%#v err=%v", mcpApplications, err)
	}
	mcpApprovalsResult := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "exact-capability-approvals",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"tenantId":              "tenant-east",
				"workspaceId":           "ws-support",
				"requestedCapabilityId": updateTicket.ID,
			},
		},
	}, ""))
	var mcpApprovals []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(mcpApprovalsResult.Result.StructuredContent, &mcpApprovals); err != nil || len(mcpApprovals) != 1 || mcpApprovals[0].ID != approved.ID {
		t.Fatalf("management MCP approval filter lost exact capability: rows=%#v err=%v", mcpApprovals, err)
	}
	health := decodeData[permissionPackageApplicationHealthResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-east&workspaceId=ws-support&requestedCapabilityId="+requested, nil, ""))
	if health.Summary.Total != 1 || len(health.Applications) != 1 || health.Applications[0].Application.RequestedCapabilityID != updateTicket.ID {
		t.Fatalf("application health lost exact capability filter: %#v", health)
	}

	readiness := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, permissionPackageProductionReadinessPath(input, "", "user:support-001"), nil, ""))
	if readiness.LatestApplication == nil || readiness.LatestApplication.ID != applied.Application.ID || readiness.LatestApplication.RequestedCapabilityID != updateTicket.ID {
		t.Fatalf("readiness did not reload exact capability application: %#v", readiness.LatestApplication)
	}
	report := decodeData[permissionPackageProductionEvidenceReportResponse](t, request(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(input, "", "user:support-001"), nil, ""))
	if report.Scope.RequestedCapabilityID != updateTicket.ID || report.Evidence.Application.RequestedCapabilityID != updateTicket.ID {
		t.Fatalf("production report lost exact capability scope: %#v", report)
	}
}

func TestPermissionPackageApplyRejectsCapabilityDriftAtRepositoryBoundary(t *testing.T) {
	memory := store.NewMemory()
	now := time.Now().UTC()
	createDirectTenant(t, memory, "tenant-east", "", "East tenant", now)
	caller := createDirectAgent(t, memory, "Support Assistant", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	target := createDirectAgent(t, memory, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	capability := createDirectCapabilityWithAction(t, memory, target.ID, "search_ticket", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	repo := &capabilityDriftBeforeApplyRepository{Repository: memory, capabilityID: capability.ID}
	router := newRouterWithRepo(repo)
	input := map[string]any{
		"callerInstanceId":      caller.ID,
		"region":                "us-east",
		"requestText":           "Apply an exact ticket search capability.",
		"requestedCapabilityId": capability.ID,
		"subjectSelector":       "user:support-*",
		"targetId":              target.ID,
		"templateId":            "support-ticket-triage",
		"tenantId":              "tenant-east",
		"workspaceId":           "ws-support",
	}
	resp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, "")
	if resp.Code != http.StatusConflict || !strings.Contains(resp.Body.String(), "PERMISSION_PACKAGE_CAPABILITY_CHANGED") {
		t.Fatalf("capability drift at apply boundary should fail closed, status=%d body=%s", resp.Code, resp.Body.String())
	}
	current, ok, err := memory.GetCapability(t.Context(), capability.ID)
	if err != nil || !ok || current.RiskLevel != domain.CapabilityRiskCritical || current.Version != capability.Version+1 || current.DiscoveryStatus != domain.CapabilityDiscoveryPendingReview {
		t.Fatalf("concurrent capability drift must remain intact: ok=%v capability=%#v err=%v", ok, current, err)
	}
	applications, err := memory.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil || len(applications) != 0 {
		t.Fatalf("conflicted apply must not create an application: rows=%#v err=%v", applications, err)
	}
	entitlements, err := memory.ListTenantEntitlements(t.Context(), store.EntitlementFilter{ManagementScope: store.ManagementScope{TenantID: "tenant-east"}})
	if err != nil || len(entitlements) != 0 {
		t.Fatalf("conflicted apply must not create grants: rows=%#v err=%v", entitlements, err)
	}
}

func TestPermissionPackagePreflightDirectApplyIsReadOnly(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读，禁止导出合同。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	preflight := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if !preflight.Summary.CanApply || preflight.Summary.BlockingCount != 0 || preflight.Summary.WarningCount != 0 ||
		preflight.Summary.PlannedCapabilityCount != 1 || preflight.Summary.PlannedTenantEntitlementCount != 1 ||
		preflight.Summary.PlannedWorkspaceAssignmentCount != 1 || preflight.Summary.PlannedInstanceAssignmentCount != 1 ||
		preflight.Summary.RequiresApproval || preflight.Summary.ApprovalReady {
		t.Fatalf("expected ready direct preflight summary, got %#v", preflight.Summary)
	}
	if !permissionPackagePreflightHasCheck(preflight.Checks, "draft_ready", "passed") ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "policy_gate", "passed") ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "data_scope_fit", "passed") ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "planned_changes", "info") {
		t.Fatalf("expected passed preflight checks, got %#v", preflight.Checks)
	}
	if len(preflight.Planned.Capabilities) != 1 || preflight.Planned.Capabilities[0].ID != search.ID ||
		len(preflight.Planned.TenantEntitlements) != 1 || preflight.Planned.TenantEntitlements[0].CapabilityID != search.ID ||
		len(preflight.Planned.WorkspaceAssignments) != 1 || preflight.Planned.WorkspaceAssignments[0].WorkspaceID != "ws-sales" ||
		len(preflight.Planned.InstanceAssignments) != 1 || preflight.Planned.InstanceAssignments[0].CallerInstanceID != caller.ID {
		t.Fatalf("unexpected planned changes: %#v", preflight.Planned)
	}
	entitlements, err := repo.ListTenantEntitlements(t.Context(), store.EntitlementFilter{})
	if err != nil {
		t.Fatalf("list entitlements: %v", err)
	}
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.applied"})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(entitlements) != 0 || len(applications) != 0 || len(events) != 0 {
		t.Fatalf("preflight must not write records: entitlements=%#v applications=%#v events=%#v", entitlements, applications, events)
	}
}

func TestPermissionPackagePreflightApprovalRequiredStates(t *testing.T) {
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
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

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
	missingApproval := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if missingApproval.Summary.CanApply || missingApproval.Summary.BlockingCount == 0 || !missingApproval.Summary.RequiresApproval ||
		missingApproval.Summary.ApprovalReady || !permissionPackagePreflightHasCheck(missingApproval.Checks, "approval_request_missing", "blocking") {
		t.Fatalf("expected missing approval to block preflight, got %#v", missingApproval)
	}
	if !slices.Contains(missingApproval.NextActionCodes, "create_approval_request") {
		t.Fatalf("expected missing approval next action code, got %#v", missingApproval.NextActionCodes)
	}

	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	if approved.Status != "approved" {
		t.Fatalf("expected approved request, got %#v", approved)
	}
	approvedInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	approvedPreflight := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", approvedInput, ""))
	if !approvedPreflight.Summary.CanApply || approvedPreflight.Summary.BlockingCount != 0 ||
		!approvedPreflight.Summary.RequiresApproval || !approvedPreflight.Summary.ApprovalReady ||
		!permissionPackagePreflightHasCheck(approvedPreflight.Checks, "approval_request_ready", "passed") ||
		len(approvedPreflight.Planned.Capabilities) != 1 || approvedPreflight.Planned.Capabilities[0].ID != updateTicket.ID {
		t.Fatalf("expected approved preflight to be ready, got %#v", approvedPreflight)
	}
	if !slices.Contains(approvedPreflight.NextActionCodes, "apply_permission_package") {
		t.Fatalf("expected approved preflight next action code, got %#v", approvedPreflight.NextActionCodes)
	}
	loadedApproval, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approved.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after preflight: ok=%v err=%v", ok, err)
	}
	if !loadedApproval.ConsumedAt.IsZero() || loadedApproval.ConsumedByApplicationID != "" {
		t.Fatalf("preflight must not consume approval, got %#v", loadedApproval)
	}
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if len(applications) != 0 {
		t.Fatalf("preflight must not create applications, got %#v", applications)
	}

	mismatchedInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            "eu-west",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	mismatched := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", mismatchedInput, ""))
	if mismatched.Summary.CanApply || !permissionPackagePreflightHasCheck(mismatched.Checks, "approval_request_invalid", "blocking") {
		t.Fatalf("expected mismatched approval to block preflight, got %#v", mismatched)
	}
	if !slices.Contains(mismatched.NextActionCodes, "refresh_approval_request") {
		t.Fatalf("expected mismatched approval next action code, got %#v", mismatched.NextActionCodes)
	}
}

func TestPermissionPackagePreflightDetectsDataScopeConflict(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithActionAndScopes(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, []domain.DataScope{{
		DataDomain:   "crm",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-east'",
	}}, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "eu-west",
		"requestText":      "给销售助手开通客户只读。",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	preflight := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if preflight.Summary.CanApply || preflight.Summary.BlockingCount == 0 ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "data_scope_fit", "blocking") {
		t.Fatalf("expected data-scope conflict to block preflight, got %#v", preflight)
	}
	if !slices.Contains(preflight.NextActionCodes, "narrow_data_scope") {
		t.Fatalf("expected data-scope next action code, got %#v", preflight.NextActionCodes)
	}
}

func TestPermissionPackagePreflightWarnsAboutExistingGrantChain(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	entitlement, err := repo.CreateTenantEntitlement(t.Context(), domain.TenantEntitlement{
		ID:           security.NewID("ent"),
		TenantID:     "tenant-east",
		TargetID:     target.ID,
		CapabilityID: search.ID,
		Effect:       domain.PolicyEffectAllow,
		DataScopes: []domain.DataScope{{
			DataDomain:   "crm",
			Region:       "华东",
			TenantFilter: "tenant_id = 'tenant-east'",
		}},
		Status:    domain.PolicyStatusEnabled,
		Priority:  40,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	workspaceAssignment, err := repo.CreateWorkspaceAssignment(t.Context(), domain.WorkspaceAssignment{
		ID:                  security.NewID("wsa"),
		TenantEntitlementID: entitlement.ID,
		TenantID:            "tenant-east",
		WorkspaceID:         "ws-sales",
		Effect:              domain.PolicyEffectAllow,
		DataScopes:          entitlement.DataScopes,
		Status:              domain.PolicyStatusEnabled,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		t.Fatalf("create workspace assignment: %v", err)
	}
	instanceAssignment, err := repo.CreateInstanceAssignment(t.Context(), domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-sales",
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:sales-*",
		Effect:                domain.PolicyEffectAllow,
		DataScopes:            entitlement.DataScopes,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err != nil {
		t.Fatalf("create instance assignment: %v", err)
	}
	preflight := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if !preflight.Summary.CanApply || preflight.Summary.WarningCount == 0 || preflight.Summary.ExistingGrantCount != 1 ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "existing_grant_chain", "warning") ||
		len(preflight.ExistingGrants) != 1 || preflight.ExistingGrants[0].CapabilityID != search.ID ||
		preflight.ExistingGrants[0].TenantEntitlementID != entitlement.ID ||
		preflight.ExistingGrants[0].WorkspaceAssignmentID != workspaceAssignment.ID ||
		preflight.ExistingGrants[0].InstanceAssignmentID != instanceAssignment.ID {
		t.Fatalf("expected existing grant warning, got %#v", preflight)
	}
	if !slices.Contains(preflight.NextActionCodes, "review_existing_grants") {
		t.Fatalf("expected existing grant next action code, got %#v", preflight.NextActionCodes)
	}
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if len(applications) != 0 {
		t.Fatalf("preflight should not create an application, got %#v", applications)
	}
}

func TestPermissionPackageApprovalAuditMetadataRedactsDirtyScope(t *testing.T) {
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
	approval, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), domain.PermissionPackageApprovalRequest{
		ID:                    "ppar-dirty-audit",
		DraftID:               "draft-dirty-approval",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              supportTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		RequestedCapabilityID: financeCapability.ID,
		SubjectSelector:       "role:support",
		RequestText:           "approve support ticket updates",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{financeCapability.Key},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
			Reasons: []domain.PermissionPackagePolicyReason{{
				ID:            "policy:dirty-audit",
				CapabilityID:  financeCapability.ID,
				CapabilityKey: financeCapability.Key,
				Severity:      "high",
				Message:       "Approval is required.",
				ReasonKey:     "permissionPolicy.actionApprovalRequired",
			}},
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "support-admin",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create dirty approval request: %v", err)
	}
	if _, err := repo.AppendAuditEvent(t.Context(), domain.AuditEvent{
		ID:           "aud-dirty-approval-request",
		TenantID:     "tenant-east",
		WorkspaceID:  "ws-support",
		Actor:        "support-admin",
		Action:       "permission_package.approval_requested",
		ResourceType: "permission_package_approval_request",
		ResourceID:   approval.ID,
		Summary:      "Permission package approval requested",
		Metadata: map[string]any{
			"approvalRequestId":     approval.ID,
			"draftId":               approval.DraftID,
			"templateId":            approval.TemplateID,
			"templateVersion":       approval.TemplateVersion,
			"policyVersion":         approval.PolicyVersion,
			"targetId":              financeTarget.ID,
			"callerInstanceId":      supportCaller.ID,
			"requestedCapabilityId": financeCapability.ID,
			"status":                approval.Status,
			"requestedBy":           approval.RequestedBy,
			"reviewedBy":            approval.ReviewedBy,
			"reasonCount":           1,
			"allowedCapabilityIds":  []string{financeCapability.ID},
			"expiresAt":             approval.ExpiresAt,
		},
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("append dirty approval audit event: %v", err)
	}

	leaks := []string{financeTarget.ID, financeTarget.Name, financeCapability.ID, financeCapability.Key, "ws-finance"}
	auditResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/audit/events?tenantId=tenant-east&workspaceId=ws-support&action=permission_package.approval_requested&resourceId="+approval.ID, nil, "", "support-key")
	if auditResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read approval audit event, got %d body=%s", auditResp.Code, auditResp.Body.String())
	}
	assertResponseDoesNotContain(t, auditResp.Body.String(), leaks...)
	auditEvents := decodeData[[]auditEventResponse](t, auditResp)
	if len(auditEvents) != 1 {
		t.Fatalf("expected one scoped approval audit event, got %#v", auditEvents)
	}
	if auditEvents[0].Metadata["targetId"] != supportTarget.ID || auditEvents[0].Metadata["callerInstanceId"] != supportCaller.ID {
		t.Fatalf("expected approval audit metadata to be rebuilt from visible approval request scope, got %#v", auditEvents[0].Metadata)
	}
	if raw, ok := auditEvents[0].Metadata["allowedCapabilityIds"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("expected approval audit metadata to redact out-of-scope capability ids, got %#v", auditEvents[0].Metadata)
	}
}

func TestPermissionPackageApprovalListRedactsDirtyCapabilityScope(t *testing.T) {
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
	approval, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), domain.PermissionPackageApprovalRequest{
		ID:                    "ppar-dirty-list",
		DraftID:               "draft-dirty-list",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              supportTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		RequestedCapabilityID: financeCapability.ID,
		SubjectSelector:       "role:support",
		RequestText:           "approve support ticket updates",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{financeCapability.Key},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
			Reasons: []domain.PermissionPackagePolicyReason{{
				ID:            "policy:dirty-list",
				CapabilityID:  financeCapability.ID,
				CapabilityKey: financeCapability.Key,
				Severity:      "high",
				Message:       "Approval is required.",
				ReasonKey:     "permissionPolicy.actionApprovalRequired",
			}},
			NextActions: []string{"Review approval scope."},
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "support-admin",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create dirty approval request: %v", err)
	}

	listResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-east&workspaceId=ws-support&status=pending", nil, "", "support-key")
	if listResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read approval request list, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	assertResponseDoesNotContain(t, listResp.Body.String(), financeTarget.ID, financeTarget.Name, financeCapability.ID, financeCapability.Key, "ws-finance")
	approvals := decodeData[[]permissionPackageApprovalRequestResponse](t, listResp)
	if len(approvals) != 1 || approvals[0].ID != approval.ID {
		t.Fatalf("expected one scoped approval request, got %#v", approvals)
	}
	if len(approvals[0].AllowedCapabilityIDs) != 0 || len(approvals[0].AllowedCapabilityKeys) != 0 {
		t.Fatalf("expected dirty approval capabilities to be redacted, got ids=%#v keys=%#v", approvals[0].AllowedCapabilityIDs, approvals[0].AllowedCapabilityKeys)
	}
	if len(approvals[0].PolicyGate.Reasons) != 0 {
		t.Fatalf("expected dirty approval policy reasons to be redacted, got %#v", approvals[0].PolicyGate.Reasons)
	}
	dirtyRequested := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-east&workspaceId=ws-support&status=pending&requestedCapabilityId="+url.QueryEscape(financeCapability.ID), nil, "", "support-key"))
	if len(dirtyRequested) != 0 {
		t.Fatalf("approval filter must not reveal a redacted capability reference: %#v", dirtyRequested)
	}
}

func TestScopedAdminCannotResolveDirtyPermissionPackageApprovalRequest(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	approval, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), domain.PermissionPackageApprovalRequest{
		ID:               "ppar-dirty-resolution",
		DraftID:          "draft-dirty-resolution",
		TemplateID:       "support-ticket-triage",
		TemplateVersion:  1,
		PolicyVersion:    1,
		TenantID:         "tenant-east",
		WorkspaceID:      "ws-support",
		TargetID:         financeTarget.ID,
		CallerInstanceID: supportCaller.ID,
		SubjectSelector:  "role:support",
		RequestText:      "approve support ticket updates",
		Region:           "us-east",
		DataScopes:       []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "requester",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create dirty approval request: %v", err)
	}

	approveResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "", "support-key")
	if approveResp.Code != http.StatusForbidden {
		t.Fatalf("dirty approval request resolution should be forbidden, got %d body=%s", approveResp.Code, approveResp.Body.String())
	}
	withdrawResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/withdraw", map[string]any{
		"comment": "withdraw dirty request",
	}, "", "support-key")
	if withdrawResp.Code != http.StatusForbidden {
		t.Fatalf("dirty approval request withdraw should be forbidden, got %d body=%s", withdrawResp.Code, withdrawResp.Body.String())
	}
	mcpApprove := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "dirty-approval-approve",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"id": approval.ID,
			}),
		},
	}, "", "support-key")
	var envelope mcpEnvelopeResponse
	if err := json.Unmarshal(mcpApprove.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode MCP dirty approval envelope: %v body=%s", err, mcpApprove.Body.String())
	}
	if envelope.Error == nil || !strings.Contains(envelope.Error.Message, "outside authenticated admin scope") {
		t.Fatalf("dirty MCP approval request resolution should be rejected, got %#v body=%s", envelope, mcpApprove.Body.String())
	}
	stored, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get dirty approval request after rejected operations: ok=%v err=%v", ok, err)
	}
	if stored.Status != domain.PermissionPackageApprovalStatusPending || stored.ReviewedBy != "" || !stored.ResolvedAt.IsZero() {
		t.Fatalf("dirty approval request must remain pending after rejected operations, got %#v", stored)
	}
}

func TestApprovalResolutionRequiresExistingApprovalResources(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	approval, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), domain.PermissionPackageApprovalRequest{
		ID:               "ppar-missing-resolution",
		DraftID:          "draft-missing-resolution",
		TemplateID:       "support-ticket-triage",
		TemplateVersion:  1,
		PolicyVersion:    1,
		TenantID:         "tenant-east",
		WorkspaceID:      "ws-support",
		TargetID:         "agt_missing_target",
		CallerInstanceID: "agt_missing_caller",
		SubjectSelector:  "role:support",
		RequestText:      "approve support ticket updates",
		Region:           "us-east",
		DataScopes:       []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "requester",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create missing-resource approval request: %v", err)
	}

	approveResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "", "support-key")
	if approveResp.Code != http.StatusNotFound || !strings.Contains(approveResp.Body.String(), "caller not found") {
		t.Fatalf("missing-resource approval request should be rejected, got %d body=%s", approveResp.Code, approveResp.Body.String())
	}
	mcpApprove := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "missing-approval-approve",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"id": approval.ID,
			}),
		},
	}, "", "support-key")
	var envelope mcpEnvelopeResponse
	if err := json.Unmarshal(mcpApprove.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode MCP missing-resource approval envelope: %v body=%s", err, mcpApprove.Body.String())
	}
	if envelope.Error == nil || !strings.Contains(envelope.Error.Message, "caller not found") {
		t.Fatalf("missing-resource MCP approval request should be rejected, got %#v body=%s", envelope, mcpApprove.Body.String())
	}
	stored, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get missing-resource approval request after rejected operations: ok=%v err=%v", ok, err)
	}
	if stored.Status != domain.PermissionPackageApprovalStatusPending || stored.ReviewedBy != "" || !stored.ResolvedAt.IsZero() {
		t.Fatalf("missing-resource approval request must remain pending after rejected operations, got %#v", stored)
	}
}

func TestApprovalResolutionRejectsExpiredApprovalRequest(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "security", Key: "security-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	approval := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-expired-resolution", "tenant-east", "ws-support", now)
	approval.ExpiresAt = now.Add(-time.Minute)
	approval.UpdatedAt = approval.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), approval); err != nil || !ok {
		t.Fatalf("expire approval request: ok=%v err=%v", ok, err)
	}

	approveResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "", "security-key")
	if approveResp.Code != http.StatusBadRequest || !strings.Contains(approveResp.Body.String(), "expired") {
		t.Fatalf("expired approval request approve should be rejected, got %d body=%s", approveResp.Code, approveResp.Body.String())
	}
	mcpReject := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "expired-approval-reject",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "reject_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"id":      approval.ID,
				"comment": "expired",
			}),
		},
	}, "", "security-key")
	var envelope mcpEnvelopeResponse
	if err := json.Unmarshal(mcpReject.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode MCP expired approval envelope: %v body=%s", err, mcpReject.Body.String())
	}
	if envelope.Error == nil || !strings.Contains(envelope.Error.Message, "expired") {
		t.Fatalf("expired MCP approval request resolution should be rejected, got %#v body=%s", envelope, mcpReject.Body.String())
	}
	stored, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get expired approval request after rejected operations: ok=%v err=%v", ok, err)
	}
	if stored.Status != domain.PermissionPackageApprovalStatusPending || stored.ReviewedBy != "" || !stored.ResolvedAt.IsZero() {
		t.Fatalf("expired approval request must remain pending after rejected operations, got %#v", stored)
	}
}

func TestApprovalPendingListSkipsExpiredRequestsBeforeLimit(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	active := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-active-pending-list", "tenant-east", "ws-support", now)
	expired := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-expired-pending-list", "tenant-east", "ws-support", now.Add(time.Minute))
	expired.ExpiresAt = now.Add(-time.Minute)
	expired.UpdatedAt = expired.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), expired); err != nil || !ok {
		t.Fatalf("expire approval request: ok=%v err=%v", ok, err)
	}

	rows := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-east&workspaceId=ws-support&status=pending&limit=1", nil, ""))
	if len(rows) != 1 || rows[0].ID != active.ID {
		t.Fatalf("expected pending queue to skip expired request before limit, got %#v", rows)
	}
}

func TestApprovalRequestResponsesExposeExpiredEffectiveStatus(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	active := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-active-effective-status", "tenant-east", "ws-support", now)
	expired := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-expired-effective-status", "tenant-east", "ws-support", now.Add(time.Minute))
	expired.ExpiresAt = now.Add(-time.Minute)
	expired.UpdatedAt = expired.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), expired); err != nil || !ok {
		t.Fatalf("expire approval request: ok=%v err=%v", ok, err)
	}

	rows := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-east&workspaceId=ws-support&limit=10", nil, ""))
	byID := map[string]permissionPackageApprovalRequestResponse{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	expiredRow := byID[expired.ID]
	if expiredRow.Status != "pending" || expiredRow.EffectiveStatus != "expired" || !expiredRow.IsExpired {
		t.Fatalf("expired approval response should preserve status and expose effective expiry, got %#v", expiredRow)
	}
	activeRow := byID[active.ID]
	if activeRow.Status != "pending" || activeRow.EffectiveStatus != "pending" || activeRow.IsExpired {
		t.Fatalf("active approval response should remain pending, got %#v", activeRow)
	}

	call := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-effective-status",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"tenantId":    "tenant-east",
				"workspaceId": "ws-support",
				"limit":       10,
			},
		},
	}, ""))
	var mcpRows []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(call.Result.StructuredContent, &mcpRows); err != nil {
		t.Fatalf("decode MCP approval responses: %v", err)
	}
	mcpByID := map[string]permissionPackageApprovalRequestResponse{}
	for _, row := range mcpRows {
		mcpByID[row.ID] = row
	}
	mcpExpiredRow := mcpByID[expired.ID]
	if mcpExpiredRow.Status != "pending" || mcpExpiredRow.EffectiveStatus != "expired" || !mcpExpiredRow.IsExpired {
		t.Fatalf("MCP expired approval response should preserve status and expose effective expiry, got %#v", mcpExpiredRow)
	}
}

func TestPermissionPackageApplyRequiresApprovalForPolicyGatedDraft(t *testing.T) {
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
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

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
	draft := decodeData[permissionPackageDraftResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, ""))
	if !draft.Readiness.CanApply {
		t.Fatalf("expected ready draft before policy approval check, got %#v", draft.Readiness)
	}
	if draft.PolicyGate.Decision != "approval_required" || draft.PolicyGate.CanApplyDirectly || len(draft.PolicyGate.Reasons) == 0 {
		t.Fatalf("expected approval-required policy gate, got %#v", draft.PolicyGate)
	}
	if !slices.Contains(draft.PolicyGate.NextActionCodes, "create_approval_request") {
		t.Fatalf("expected approval policy next action code, got %#v", draft.PolicyGate.NextActionCodes)
	}

	applyResp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, "")
	if applyResp.Code != http.StatusBadRequest {
		t.Fatalf("expected approval-required apply to fail, status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
	if !strings.Contains(applyResp.Body.String(), "requires approval") {
		t.Fatalf("expected approval error message, body=%s", applyResp.Body.String())
	}
	entitlements, err := repo.ListTenantEntitlements(t.Context(), store.EntitlementFilter{})
	if err != nil {
		t.Fatalf("list entitlements: %v", err)
	}
	workspaceAssignments, err := repo.ListWorkspaceAssignments(t.Context(), store.AssignmentFilter{})
	if err != nil {
		t.Fatalf("list workspace assignments: %v", err)
	}
	instanceAssignments, err := repo.ListInstanceAssignments(t.Context(), store.InstanceAssignmentFilter{})
	if err != nil {
		t.Fatalf("list instance assignments: %v", err)
	}
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.applied"})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	updated, ok, err := repo.GetCapability(t.Context(), updateTicket.ID)
	if err != nil || !ok {
		t.Fatalf("get capability: ok=%v err=%v", ok, err)
	}
	if len(entitlements) != 0 || len(workspaceAssignments) != 0 || len(instanceAssignments) != 0 || len(applications) != 0 || len(events) != 0 {
		t.Fatalf("approval-required package should not write records: entitlements=%#v workspace=%#v instances=%#v applications=%#v events=%#v", entitlements, workspaceAssignments, instanceAssignments, applications, events)
	}
	if updated.DiscoveryStatus != domain.CapabilityDiscoveryPendingReview {
		t.Fatalf("approval-required package should not update capability, got %#v", updated)
	}

	firstApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	if firstApproval.Status != "pending" || firstApproval.TemplateID != "support-ticket-triage" ||
		firstApproval.TemplateVersion != 2 || firstApproval.PolicyVersion != 1 ||
		firstApproval.TargetID != target.ID || firstApproval.CallerInstanceID != caller.ID ||
		len(firstApproval.AllowedCapabilityIDs) != 1 || firstApproval.AllowedCapabilityIDs[0] != updateTicket.ID ||
		len(firstApproval.PolicyGate.Reasons) == 0 {
		t.Fatalf("unexpected created approval request: %#v", firstApproval)
	}
	if firstApproval.ExpiresAt.IsZero() || !firstApproval.ExpiresAt.After(firstApproval.CreatedAt) {
		t.Fatalf("approval request should expose a future expiry: %#v", firstApproval)
	}
	duplicateApprovalResp := request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "")
	if duplicateApprovalResp.Code != http.StatusConflict {
		t.Fatalf("duplicate pending approval request should be rejected, status=%d body=%s", duplicateApprovalResp.Code, duplicateApprovalResp.Body.String())
	}
	var duplicateApprovalErr apiEnvelope
	if err := json.Unmarshal(duplicateApprovalResp.Body.Bytes(), &duplicateApprovalErr); err != nil {
		t.Fatalf("decode duplicate approval error: %v body=%s", err, duplicateApprovalResp.Body.String())
	}
	if duplicateApprovalErr.Error != "PERMISSION_PACKAGE_APPROVAL_ALREADY_PENDING" {
		t.Fatalf("duplicate approval should return stable app code, got %#v", duplicateApprovalErr)
	}
	expiredPending, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), firstApproval.ID)
	if err != nil || !ok {
		t.Fatalf("get pending approval for duplicate expiry test: ok=%v err=%v", ok, err)
	}
	expiredPending.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	expiredPending.UpdatedAt = expiredPending.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), expiredPending); err != nil || !ok {
		t.Fatalf("expire pending approval request: ok=%v err=%v", ok, err)
	}
	replacementApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	if replacementApproval.ID == firstApproval.ID || replacementApproval.Status != "pending" {
		t.Fatalf("expired pending approval should allow a fresh request, old=%#v new=%#v", firstApproval, replacementApproval)
	}
	firstApproval = replacementApproval
	listedApprovals := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-root&workspaceId=ws-support&templateId=support-ticket-triage&targetId="+target.ID+"&callerInstanceId="+caller.ID+"&status=pending&limit=1", nil, ""))
	if len(listedApprovals) != 1 || listedApprovals[0].ID != firstApproval.ID {
		t.Fatalf("expected listed pending approval request, got %#v", listedApprovals)
	}

	pendingApplyInput := map[string]any{
		"approvalRequestId": firstApproval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	pendingApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", pendingApplyInput, "")
	if pendingApply.Code != http.StatusBadRequest || !strings.Contains(pendingApply.Body.String(), "approved") {
		t.Fatalf("pending approval request should not authorize apply, status=%d body=%s", pendingApply.Code, pendingApply.Body.String())
	}
	rejectedApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+firstApproval.ID+"/reject", map[string]any{
		"reviewer": "security",
		"comment":  "too broad",
	}, ""))
	if rejectedApproval.Status != "rejected" || rejectedApproval.ReviewedBy != "security" || rejectedApproval.ReviewComment != "too broad" {
		t.Fatalf("unexpected rejected approval request: %#v", rejectedApproval)
	}
	rejectedApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", pendingApplyInput, "")
	if rejectedApply.Code != http.StatusBadRequest || !strings.Contains(rejectedApply.Body.String(), "approved") {
		t.Fatalf("rejected approval request should not authorize apply, status=%d body=%s", rejectedApply.Code, rejectedApply.Body.String())
	}

	secondApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approvedApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+secondApproval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	if approvedApproval.Status != "approved" || approvedApproval.ReviewedBy != "security" {
		t.Fatalf("unexpected approved approval request: %#v", approvedApproval)
	}
	mismatchedApplyInput := map[string]any{
		"approvalRequestId": secondApproval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "eu-west",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	mismatchedApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", mismatchedApplyInput, "")
	if mismatchedApply.Code != http.StatusBadRequest || !strings.Contains(mismatchedApply.Body.String(), "does not match") {
		t.Fatalf("mismatched approval request should not authorize apply, status=%d body=%s", mismatchedApply.Code, mismatchedApply.Body.String())
	}

	expiredApproval, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), secondApproval.ID)
	if err != nil || !ok {
		t.Fatalf("get approved approval for expiry test: ok=%v err=%v", ok, err)
	}
	expiredApproval.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	expiredApproval.UpdatedAt = expiredApproval.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), expiredApproval); err != nil || !ok {
		t.Fatalf("expire approval request: ok=%v err=%v", ok, err)
	}
	approvedApplyInput := map[string]any{
		"approvalRequestId": secondApproval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	expiredApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", approvedApplyInput, "")
	if expiredApply.Code != http.StatusBadRequest || !strings.Contains(expiredApply.Body.String(), "expired") {
		t.Fatalf("expired approval request should not authorize apply, status=%d body=%s", expiredApply.Code, expiredApply.Body.String())
	}

	thirdApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	thirdApproval = decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+thirdApproval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	approvedApplyInput["approvalRequestId"] = thirdApproval.ID
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", approvedApplyInput, ""))
	if len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != updateTicket.ID ||
		len(applied.WorkspaceAssignments) != 1 || len(applied.InstanceAssignments) != 1 ||
		applied.Application == nil || applied.Application.TemplateID != "support-ticket-triage" {
		t.Fatalf("expected approved package apply to write records, got %#v", applied)
	}
	consumedApproval, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), thirdApproval.ID)
	if err != nil || !ok {
		t.Fatalf("get consumed approval request: ok=%v err=%v", ok, err)
	}
	if consumedApproval.ConsumedAt.IsZero() || consumedApproval.ConsumedByApplicationID != applied.Application.ID {
		t.Fatalf("approval request should be consumed by application %s, got %#v", applied.Application.ID, consumedApproval)
	}
	reusedApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", approvedApplyInput, "")
	if reusedApply.Code != http.StatusBadRequest ||
		!strings.Contains(reusedApply.Body.String(), "PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED") ||
		!strings.Contains(reusedApply.Body.String(), "already consumed") {
		t.Fatalf("consumed approval request should not authorize apply, status=%d body=%s", reusedApply.Code, reusedApply.Body.String())
	}
	updated, ok, err = repo.GetCapability(t.Context(), updateTicket.ID)
	if err != nil || !ok {
		t.Fatalf("get applied capability: ok=%v err=%v", ok, err)
	}
	if updated.DiscoveryStatus != domain.CapabilityDiscoveryApproved {
		t.Fatalf("approved package should update capability, got %#v", updated)
	}
	events, err = repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.applied"})
	if err != nil {
		t.Fatalf("list applied audit events: %v", err)
	}
	if len(events) != 1 || events[0].Metadata["approvalRequestId"] != thirdApproval.ID ||
		events[0].Metadata["approvalConsumedAt"] == nil || events[0].Metadata["approvalExpiresAt"] == nil {
		t.Fatalf("expected applied audit event with approval request id, got %#v", events)
	}
	applications, err = repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications after consumed retry: %v", err)
	}
	if len(applications) != 1 {
		t.Fatalf("consumed approval retry should not write duplicate applications: %#v", applications)
	}
}

func TestPermissionPackageApprovalRejectsCapabilityDriftAfterApproval(t *testing.T) {
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
	updateTicket := createDirectCapabilityWithActionAndScopes(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, []domain.DataScope{{
		DataDomain:   "support",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-east'",
	}}, now)

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

	changed := decodeData[capabilityResponse](t, request(t, router, http.MethodPatch, "/api/v1/capabilities/"+updateTicket.ID, map[string]any{
		"dataScopes": []map[string]any{{"dataDomain": "support"}},
	}, ""))
	if len(changed.DataScopes) != 1 || changed.DataScopes[0].Region != "" {
		t.Fatalf("expected capability boundary to be widened, got %#v", changed.DataScopes)
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
	applyResp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, "")
	if applyResp.Code != http.StatusBadRequest || !strings.Contains(applyResp.Body.String(), "does not match") {
		t.Fatalf("capability drift should invalidate approval request, status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
}

func TestPermissionPackageApprovalRejectsTemplateAndPolicyVersionDrift(t *testing.T) {
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
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	stored, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approved.ID)
	if err != nil || !ok {
		t.Fatalf("get approval request for version drift: ok=%v err=%v", ok, err)
	}
	stored.TemplateVersion++
	stored.PolicyVersion++
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), stored); err != nil || !ok {
		t.Fatalf("update approval request version drift: ok=%v err=%v", ok, err)
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
	applyResp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, "")
	if applyResp.Code != http.StatusBadRequest || !strings.Contains(applyResp.Body.String(), "does not match") {
		t.Fatalf("template or policy version drift should invalidate approval request, status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
}

func TestPermissionPackageApprovalRejectsSelfApproval(t *testing.T) {
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
	resp := request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "")
	if resp.Code != http.StatusForbidden || !strings.Contains(resp.Body.String(), "own permission package approval request") {
		t.Fatalf("self approval should be rejected, status=%d body=%s", resp.Code, resp.Body.String())
	}
	loaded, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after rejected self approval: ok=%v err=%v", ok, err)
	}
	if loaded.Status != domain.PermissionPackageApprovalStatusPending || loaded.ReviewedBy != "" {
		t.Fatalf("self approval rejection must leave request pending, got %#v", loaded)
	}
}

func TestPermissionPackageApprovalRequestWithdraw(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
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
	approval := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "", "requester-key"))
	impersonation := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/withdraw", map[string]any{
		"comment": "not my request",
	}, "", "security-key")
	if impersonation.Code != http.StatusForbidden || !strings.Contains(impersonation.Body.String(), "requester can withdraw") {
		t.Fatalf("non-requester withdraw should be rejected, status=%d body=%s", impersonation.Code, impersonation.Body.String())
	}

	withdrawn := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/withdraw", map[string]any{
		"comment": "wrong scope",
	}, "", "requester-key"))
	if withdrawn.Status != "withdrawn" || withdrawn.ReviewedBy != "requester" || withdrawn.ReviewComment != "wrong scope" || withdrawn.ResolvedAt.IsZero() {
		t.Fatalf("unexpected withdrawn approval request: %#v", withdrawn)
	}
	pending := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-root&workspaceId=ws-support&status=pending&limit=10", nil, "", "requester-key"))
	if len(pending) != 0 {
		t.Fatalf("withdrawn approval should leave pending queue, got %#v", pending)
	}
	withdrawnRows := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-root&workspaceId=ws-support&status=withdrawn&limit=10", nil, "", "requester-key"))
	if len(withdrawnRows) != 1 || withdrawnRows[0].ID != approval.ID {
		t.Fatalf("expected withdrawn approval in withdrawn list, got %#v", withdrawnRows)
	}
	withdrawnApplyInput := map[string]any{
		"approvalRequestId": approval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	withdrawnApply := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages:apply", withdrawnApplyInput, "", "requester-key")
	if withdrawnApply.Code != http.StatusBadRequest || !strings.Contains(withdrawnApply.Body.String(), "approved") {
		t.Fatalf("withdrawn approval request should not authorize apply, status=%d body=%s", withdrawnApply.Code, withdrawnApply.Body.String())
	}
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.approval_withdrawn"})
	if err != nil {
		t.Fatalf("list withdraw audit events: %v", err)
	}
	if len(events) != 1 || events[0].Actor != "requester" || events[0].ResourceID != approval.ID || events[0].Metadata["status"] != domain.PermissionPackageApprovalStatusWithdrawn {
		t.Fatalf("expected withdrawn audit event, got %#v", events)
	}

	approvedApproval := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "", "requester-key"))
	approvedApproval = decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approvedApproval.ID+"/approve", nil, "", "security-key"))
	withdrawApproved := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approvedApproval.ID+"/withdraw", map[string]any{
		"comment": "too late",
	}, "", "requester-key")
	if withdrawApproved.Code != http.StatusBadRequest || !strings.Contains(withdrawApproved.Body.String(), "already resolved") {
		t.Fatalf("approved approval should not be withdrawable, status=%d body=%s", withdrawApproved.Code, withdrawApproved.Body.String())
	}
	loaded, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approvedApproval.ID)
	if err != nil || !ok {
		t.Fatalf("get approved approval after withdraw attempt: ok=%v err=%v", ok, err)
	}
	if loaded.Status != domain.PermissionPackageApprovalStatusApproved {
		t.Fatalf("approved approval should remain approved, got %#v", loaded)
	}
}

func TestPermissionPackageApprovalReviewerUsesAuthenticatedAdminIdentity(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
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
	approval := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "", "requester-key"))
	if approval.RequestedBy != "requester" {
		t.Fatalf("approval request should be attributed to authenticated requester, got %#v", approval)
	}
	impersonation := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "requester",
	}, "", "security-key")
	if impersonation.Code != http.StatusForbidden || !strings.Contains(impersonation.Body.String(), "authenticated admin identity") {
		t.Fatalf("reviewer impersonation should be rejected, status=%d body=%s", impersonation.Code, impersonation.Body.String())
	}
	approved := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "", "security-key"))
	if approved.Status != "approved" || approved.ReviewedBy != "security" {
		t.Fatalf("approval should use authenticated reviewer identity, got %#v", approved)
	}
}

func TestPermissionPackageApprovalReviewerQueueUsesAuthenticatedAdminIdentity(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 30, 0, 0, time.UTC)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-auth-east-support", "tenant-east", "ws-support", now)
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	impersonatedQueue := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security&limit=10", nil, "", "requester-key")
	if impersonatedQueue.Code != http.StatusForbidden || !strings.Contains(impersonatedQueue.Body.String(), "authenticated admin identity") {
		t.Fatalf("reviewer queue impersonation should be rejected, status=%d body=%s", impersonatedQueue.Code, impersonatedQueue.Body.String())
	}

	securityQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security&limit=10", nil, "", "security-key"))
	if len(securityQueue) != 1 || securityQueue[0].ID != eastSupport.ID {
		t.Fatalf("authenticated reviewer should see routed queue, got %#v", securityQueue)
	}
}

func TestPermissionPackageApprovalReviewerQueueDefaultsToAuthenticatedReviewer(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 40, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-default-east-support", "tenant-east", "ws-support", now)
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-default-east-finance", "tenant-east", "ws-finance", now.Add(time.Minute))
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "security", Key: "security-key", Role: "security_reviewer", TenantID: "tenant-east"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	securityQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&limit=10", nil, "", "security-key"))
	if len(securityQueue) != 1 || securityQueue[0].ID != eastSupport.ID {
		t.Fatalf("authenticated reviewer default queue should be routed, got %#v", securityQueue)
	}

	platformQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&limit=10", nil, "", "platform-key"))
	if len(platformQueue) != 2 {
		t.Fatalf("platform admin should keep all-queue visibility when reviewer is omitted, got %#v", platformQueue)
	}
}

func TestPermissionPackageApprovalReviewerRouting(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-east-support", "tenant-east", "ws-support", now)
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-east-sales", "tenant-east", "ws-sales", now.Add(time.Minute))
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-west-support", "tenant-west", "ws-support", now.Add(2*time.Minute))
	router := newRouterWithRepoAndApprovalReviewers(repo, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security-east", TenantID: "tenant-east", WorkspaceID: "ws-support"},
		{Reviewer: "security-root", TenantID: "tenant-root", WorkspaceID: "*"},
		{Reviewer: "security-root", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	eastQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security-east&limit=10", nil, ""))
	if len(eastQueue) != 1 || eastQueue[0].ID != eastSupport.ID {
		t.Fatalf("expected security-east to see only east support approvals, got %#v", eastQueue)
	}
	eastSalesQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security-east&workspaceId=ws-sales&limit=10", nil, ""))
	if len(eastSalesQueue) != 0 {
		t.Fatalf("expected security-east workspace route to exclude east sales approvals, got %#v", eastSalesQueue)
	}
	rootQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security-root&limit=10", nil, ""))
	if len(rootQueue) != 3 {
		t.Fatalf("expected root reviewer to see deduplicated tenant subtree approvals, got %#v", rootQueue)
	}
	rootEastQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-east&status=pending&reviewer=security-root&limit=10", nil, ""))
	if len(rootEastQueue) != 2 {
		t.Fatalf("expected root reviewer tenant query to narrow to east approvals, got %#v", rootEastQueue)
	}
	limitedRootQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security-root&limit=2", nil, ""))
	if len(limitedRootQueue) != 2 || limitedRootQueue[0].ID != "ppar-west-support" || limitedRootQueue[1].ID != "ppar-east-sales" {
		t.Fatalf("expected root reviewer queue to be sorted and limited after dedupe, got %#v", limitedRootQueue)
	}

	unauthorized := request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+eastSupport.ID+"/approve", map[string]any{
		"reviewer": "security-root-ws-sales",
		"comment":  "wrong workspace",
	}, "")
	if unauthorized.Code != http.StatusForbidden || !strings.Contains(unauthorized.Body.String(), "not allowed") {
		t.Fatalf("expected unauthorized reviewer rejection, status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	stillPending, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), eastSupport.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after unauthorized review: ok=%v err=%v", ok, err)
	}
	if stillPending.Status != domain.PermissionPackageApprovalStatusPending || stillPending.ReviewedBy != "" {
		t.Fatalf("unauthorized review should not mutate approval request: %#v", stillPending)
	}

	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+eastSupport.ID+"/approve", map[string]any{
		"reviewer": "security-east",
		"comment":  "approved by routed reviewer",
	}, ""))
	if approved.Status != "approved" || approved.ReviewedBy != "security-east" || approved.ReviewComment != "approved by routed reviewer" {
		t.Fatalf("unexpected routed approval result: %#v", approved)
	}
}

func TestPermissionPackageDraftDetectsDataScopeConflicts(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithActionAndScopes(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, []domain.DataScope{{
		DataDomain:   "crm",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-east'",
	}}, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "eu-west",
		"requestText":      "给销售助手开通客户只读。",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	draft := decodeData[permissionPackageDraftResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, ""))
	if draft.Readiness.CanApply || len(draft.Readiness.Warnings) == 0 {
		t.Fatalf("expected data-scope conflict warning, got %#v", draft.Readiness)
	}
	applyResp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, "")
	if applyResp.Code != http.StatusBadRequest {
		t.Fatalf("expected apply to reject conflicting data scopes, status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
}
