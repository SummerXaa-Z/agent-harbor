package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
		TenantID              string `json:"tenantId"`
		WorkspaceID           string `json:"workspaceId"`
		CallerInstanceID      string `json:"callerInstanceId"`
		TargetID              string `json:"targetId"`
		RequestedCapabilityID string `json:"requestedCapabilityId"`
		SubjectID             string `json:"subjectId"`
		SubjectSelector       string `json:"subjectSelector"`
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
	Tokens []struct {
		ID                  string `json:"id"`
		Status              string `json:"status"`
		ApplicationID       string `json:"applicationId"`
		SubjectSelector     string `json:"subjectSelector"`
		CreatedForHandoffID string `json:"createdForHandoffId"`
	} `json:"tokens"`
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

type accessHandoffTokenCreateResponse struct {
	ID                  string `json:"id"`
	Key                 string `json:"key"`
	Prefix              string `json:"prefix"`
	Status              string `json:"status"`
	ApplicationID       string `json:"applicationId"`
	SubjectSelector     string `json:"subjectSelector"`
	CreatedForHandoffID string `json:"createdForHandoffId"`
}

func TestPermissionPackageAccessHandoffBlocksBeforeApplyAndReturnsReadyArtifacts(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "invalid JSON-RPC request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if requestBody.Method == "tools/list" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      requestBody.ID,
				"result": map[string]any{"tools": []map[string]any{
					{"name": "search_ticket"},
					{"name": "update_ticket"},
					{"name": "export_contracts"},
				}},
			})
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      requestBody.ID,
			"result":  map[string]any{"ok": true},
		})
	}))
	defer upstream.Close()
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, map[string]any{"endpoint": upstream.URL})
	searchTicket := createDirectCapabilityWithAction(t, repo, target.ID, "search_ticket", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	exportContracts := createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId":      caller.ID,
		"region":                "us-east",
		"requestText":           "Allow only ticket updates for this tenant.",
		"requestedCapabilityId": updateTicket.ID,
		"subjectSelector":       "user:support-*",
		"targetId":              target.ID,
		"templateId":            "support-ticket-triage",
		"tenantId":              "tenant-east",
		"workspaceId":           "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, "POST", "/api/v1/permission-packages/approval-requests", input, ""))
	if approval.RequestedCapabilityID != updateTicket.ID || len(approval.AllowedCapabilityIDs) != 1 || approval.AllowedCapabilityIDs[0] != updateTicket.ID {
		t.Fatalf("approval lost exact requested capability boundary: %#v", approval)
	}
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, "POST", "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{"reviewer": "security"}, ""))

	before := decodeData[accessHandoffResponse](t, request(t, router, "GET", permissionPackageAccessHandoffPath(input, approved.ID, "user:support-001"), nil, ""))
	if before.HandoffVersion != "access-handoff/v1" || before.Status != "blocked" || before.Application != nil || before.CopyArtifacts != nil || before.TokenEligibility.Eligible {
		t.Fatalf("expected blocked handoff without usable artifacts before apply, got %#v", before)
	}
	if len(before.TokenEligibility.BlockerCodes) == 0 || before.NextActionCode != "apply_permission_package" {
		t.Fatalf("expected actionable handoff blockers before apply, got %#v", before)
	}
	blockedToken := request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", accessHandoffTokenRequest(input, "handoff:missing", "user:support-001"), "")
	if blockedToken.Code != http.StatusConflict || !strings.Contains(blockedToken.Body.String(), "ACCESS_HANDOFF_NOT_READY") {
		t.Fatalf("expected blocked token issuance before handoff readiness, got status=%d body=%s", blockedToken.Code, blockedToken.Body.String())
	}

	applyInput := map[string]any{
		"approvalRequestId":     approved.ID,
		"callerInstanceId":      caller.ID,
		"region":                input["region"],
		"requestText":           input["requestText"],
		"requestedCapabilityId": input["requestedCapabilityId"],
		"subjectSelector":       input["subjectSelector"],
		"targetId":              target.ID,
		"templateId":            input["templateId"],
		"tenantId":              input["tenantId"],
		"workspaceId":           input["workspaceId"],
	}
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, "POST", "/api/v1/permission-packages:apply", applyInput, ""))
	if applied.Application == nil || applied.Application.RequestedCapabilityID != updateTicket.ID || len(applied.Application.AllowedCapabilityIDs) != 1 || applied.Application.AllowedCapabilityIDs[0] != updateTicket.ID {
		t.Fatalf("expected exact permission package application for requested capability: %#v", applied.Application)
	}
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, exportContracts, "export_contracts", "user:support-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, updateTicket, "update_ticket", "user:support-001", now.Add(2*time.Minute))

	after := decodeData[accessHandoffResponse](t, request(t, router, "GET", permissionPackageAccessHandoffPath(input, "", "user:support-001"), nil, ""))
	if after.Status != "ready" || after.ID != "handoff:"+applied.Application.ID || after.Application == nil || after.Application.ID != applied.Application.ID {
		t.Fatalf("expected ready handoff for latest application, got %#v", after)
	}
	if after.Scope.TenantID != "tenant-east" || after.Scope.WorkspaceID != "ws-support" || after.Scope.CallerInstanceID != caller.ID || after.Scope.TargetID != target.ID || after.Scope.RequestedCapabilityID != updateTicket.ID || after.Scope.SubjectSelector != "user:support-*" {
		t.Fatalf("unexpected handoff scope: %#v", after.Scope)
	}
	if after.Template.ID != "support-ticket-triage" || after.Template.Version != 2 || after.Template.Name == "" || len(after.DataScopes) == 0 {
		t.Fatalf("unexpected handoff template or scopes: template=%#v scopes=%#v", after.Template, after.DataScopes)
	}
	if !accessHandoffCapabilitiesContain(after.AllowedCapabilities, updateTicket.ID, "update_ticket") ||
		!accessHandoffCapabilitiesContain(after.BlockedCapabilities, searchTicket.ID, "search_ticket") ||
		!accessHandoffCapabilitiesContain(after.BlockedCapabilities, exportContracts.ID, "export_contracts") {
		t.Fatalf("unexpected handoff capabilities: allowed=%#v blocked=%#v", after.AllowedCapabilities, after.BlockedCapabilities)
	}
	if !after.TokenEligibility.Eligible || after.TokenEligibility.DefaultExpiresInSeconds != 1800 || after.TokenEligibility.MaxExpiresInSeconds != 3600 || len(after.TokenEligibility.BlockerCodes) != 0 {
		t.Fatalf("expected eligible short-lived token contract, got %#v", after.TokenEligibility)
	}
	if after.CopyArtifacts == nil ||
		!strings.Contains(after.CopyArtifacts.MCPClientConfig, "/api/v1/mcp/agents/"+target.ID+"/rpc") ||
		!strings.Contains(after.CopyArtifacts.MCPClientConfig, "${AGENT_HARBOR_TOKEN}") ||
		!strings.Contains(after.CopyArtifacts.RuntimeRequestExample, "update_ticket") ||
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
	differentSelectorRequest := accessHandoffTokenRequest(input, after.ID, "user:finance-001")
	differentSelectorRequest["subjectSelector"] = "user:finance-*"
	differentSelector := request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", differentSelectorRequest, "")
	if differentSelector.Code != http.StatusConflict || !strings.Contains(differentSelector.Body.String(), "ACCESS_HANDOFF_NOT_READY") {
		t.Fatalf("expected token request with a bounded selector different from the application to be blocked, got status=%d body=%s", differentSelector.Code, differentSelector.Body.String())
	}
	wrongSubjectRequest := request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", accessHandoffTokenRequest(input, after.ID, "user:sales-001"), "")
	if wrongSubjectRequest.Code != http.StatusConflict || !strings.Contains(wrongSubjectRequest.Body.String(), "ACCESS_HANDOFF_NOT_READY") {
		t.Fatalf("expected token request whose subject does not match the selector to be blocked, got status=%d body=%s", wrongSubjectRequest.Code, wrongSubjectRequest.Body.String())
	}

	tokenRequest := accessHandoffTokenRequest(input, after.ID, "user:support-001")
	if tokenRequest["requestedCapabilityId"] != updateTicket.ID {
		t.Fatalf("access handoff token request lost exact requested capability: %#v", tokenRequest)
	}
	createdRecorder, createdRequest := buildRequest(t, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", tokenRequest, "", "", "")
	conflictRecorder, conflictRequest := buildRequest(t, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", tokenRequest, "", "", "")
	var issueWG sync.WaitGroup
	issueWG.Add(2)
	go func() {
		defer issueWG.Done()
		router.ServeHTTP(createdRecorder, createdRequest)
	}()
	go func() {
		defer issueWG.Done()
		router.ServeHTTP(conflictRecorder, conflictRequest)
	}()
	issueWG.Wait()
	if createdRecorder.Code == http.StatusConflict {
		createdRecorder, conflictRecorder = conflictRecorder, createdRecorder
	}
	if createdRecorder.Code != http.StatusCreated || conflictRecorder.Code != http.StatusConflict || !strings.Contains(conflictRecorder.Body.String(), "ACCESS_HANDOFF_TOKEN_ACTIVE") {
		t.Fatalf("expected exactly one concurrent token creation, got first=%d body=%s second=%d body=%s", createdRecorder.Code, createdRecorder.Body.String(), conflictRecorder.Code, conflictRecorder.Body.String())
	}
	created := decodeData[accessHandoffTokenCreateResponse](t, createdRecorder)
	if created.ID == "" || created.Key == "" || created.Prefix == "" || created.Status != "active" || created.ApplicationID != applied.Application.ID || created.SubjectSelector != "user:support-*" || created.CreatedForHandoffID != after.ID {
		t.Fatalf("unexpected access handoff token: %#v", created)
	}
	searchTicket.DiscoveryStatus = domain.CapabilityDiscoveryApproved
	searchTicket.UpdatedAt = now.Add(3 * time.Minute)
	var updated bool
	searchTicket, updated, err := repo.UpdateCapability(t.Context(), searchTicket)
	if err != nil || !updated {
		t.Fatalf("approve extra capability for underlying-access fixture: updated=%v err=%v", updated, err)
	}
	extraEntitlement := createDirectTenantEntitlement(t, repo, "tenant-east", target.ID, searchTicket.ID, nil, now.Add(3*time.Minute))
	extraWorkspace := createDirectWorkspaceAssignment(t, repo, extraEntitlement.ID, "tenant-east", "ws-support", nil, now.Add(3*time.Minute))
	createDirectSubjectInstanceAssignment(t, repo, extraWorkspace.ID, "tenant-east", "ws-support", caller.ID, "user:support-*", nil, now.Add(3*time.Minute))
	applicationWorkspaces, err := repo.ListWorkspaceAssignments(t.Context(), store.AssignmentFilter{EntitlementID: applied.TenantEntitlements[0].ID})
	if err != nil || len(applicationWorkspaces) != 1 {
		t.Fatalf("load applied workspace assignment: rows=%#v err=%v", applicationWorkspaces, err)
	}
	createDirectSubjectInstanceAssignment(t, repo, applicationWorkspaces[0].ID, "tenant-east", "ws-support", caller.ID, "user:sales-*", applicationWorkspaces[0].DataScopes, now.Add(3*time.Minute))
	regularKey := createDirectTestAgentKey(t, repo, caller.ID, now)
	regularExtraCapabilityCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "regular-extra-capability",
		"method":  "tools/call",
		"params":  map[string]any{"name": "search_ticket", "arguments": map[string]any{"query": "T-100"}},
	}, regularKey, "regular-extra-capability", "user:support-001")
	if regularExtraCapabilityCall.Code != http.StatusAccepted {
		t.Fatalf("expected ordinary token to prove the extra capability is otherwise authorized, got status=%d body=%s", regularExtraCapabilityCall.Code, regularExtraCapabilityCall.Body.String())
	}
	regularOtherSubjectCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "regular-other-subject",
		"method":  "tools/call",
		"params":  map[string]any{"name": "update_ticket", "arguments": map[string]any{"ticketId": "T-101"}},
	}, regularKey, "regular-other-subject", "user:sales-001")
	if regularOtherSubjectCall.Code != http.StatusAccepted {
		t.Fatalf("expected ordinary token to prove the other subject is otherwise authorized, got status=%d body=%s", regularOtherSubjectCall.Code, regularOtherSubjectCall.Body.String())
	}
	handoffExtraCapabilityCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-extra-capability",
		"method":  "tools/call",
		"params":  map[string]any{"name": "search_ticket", "arguments": map[string]any{"query": "T-100"}},
	}, created.Key, "handoff-extra-capability", "user:support-001")
	assertAccessHandoffDeniedWithTrace(t, repo, handoffExtraCapabilityCall, "handoff-extra-capability", "access handoff token does not allow this capability")
	allowedCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-allowed",
		"method":  "tools/call",
		"params":  map[string]any{"name": "update_ticket", "arguments": map[string]any{"ticketId": "T-100", "priority": "high"}},
	}, created.Key, "access-handoff-allowed", "user:support-001")
	if allowedCall.Code != http.StatusAccepted {
		t.Fatalf("expected handoff token to call an allowed capability, got status=%d body=%s", allowedCall.Code, allowedCall.Body.String())
	}
	deniedCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-denied",
		"method":  "tools/call",
		"params":  map[string]any{"name": "export_contracts", "arguments": map[string]any{}},
	}, created.Key, "access-handoff-denied", "user:support-001")
	if deniedCall.Code != http.StatusForbidden {
		t.Fatalf("expected handoff token to remain bounded by denied capabilities, got status=%d body=%s", deniedCall.Code, deniedCall.Body.String())
	}
	toolsList := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-tools-list",
		"method":  "tools/list",
	}, created.Key, "access-handoff-tools-list", "user:support-001")
	if toolsList.Code != http.StatusOK {
		t.Fatalf("expected handoff tools/list bounded to application capabilities, got status=%d body=%s", toolsList.Code, toolsList.Body.String())
	}
	toolNames := decodeMCPToolsListNames(t, toolsList)
	if len(toolNames) != 1 || toolNames[0] != "update_ticket" {
		t.Fatalf("expected handoff tools/list to contain only requested capability, got %#v", toolNames)
	}
	wrongSubjectCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-wrong-subject",
		"method":  "tools/call",
		"params":  map[string]any{"name": "update_ticket", "arguments": map[string]any{}},
	}, created.Key, "access-handoff-wrong-subject", "user:sales-001")
	assertAccessHandoffDeniedWithTrace(t, repo, wrongSubjectCall, "access-handoff-wrong-subject", "access handoff token does not allow this subject")
	wrongSubjectToolsList := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-wrong-subject-tools-list",
		"method":  "tools/list",
	}, created.Key, "access-handoff-wrong-subject-tools-list", "user:sales-001")
	assertAccessHandoffDeniedWithTrace(t, repo, wrongSubjectToolsList, "access-handoff-wrong-subject-tools-list", "access handoff token does not allow this subject")
	genericRouteCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-generic-route",
		"method":  "resources/list",
	}, created.Key, "access-handoff-generic-route", "user:support-001")
	if genericRouteCall.Code != http.StatusForbidden {
		t.Fatalf("expected handoff token to reject generic routes, got status=%d body=%s", genericRouteCall.Code, genericRouteCall.Body.String())
	}
	grantRoute(t, router, caller.ID, target.ID, "openapi", "projects/42")
	openAPICall := requestWithRunIDAndSubject(t, router, http.MethodGet, "/api/v1/openapi/agents/"+target.ID+"/projects/42", nil, created.Key, "access-handoff-openapi", "user:support-001")
	if openAPICall.Code != http.StatusForbidden {
		t.Fatalf("expected handoff token to reject OpenAPI routes, got status=%d body=%s", openAPICall.Code, openAPICall.Body.String())
	}
	afterCreate := decodeData[accessHandoffResponse](t, request(t, router, http.MethodGet, permissionPackageAccessHandoffPath(input, "", "user:support-001"), nil, ""))
	if len(afterCreate.Tokens) != 1 || afterCreate.Tokens[0].ID != created.ID || afterCreate.Tokens[0].Status != "active" {
		t.Fatalf("expected active token in handoff projection, got %#v", afterCreate.Tokens)
	}
	duplicate := request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", tokenRequest, "")
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), "ACCESS_HANDOFF_TOKEN_ACTIVE") {
		t.Fatalf("expected duplicate active token rejection, got status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	revoked := decodeData[accessHandoffTokenCreateResponse](t, request(t, router, http.MethodDelete, "/api/v1/permission-packages/access-handoff/tokens/"+created.ID, nil, ""))
	if revoked.ID != created.ID || revoked.Status != "revoked" || revoked.Key != "" {
		t.Fatalf("expected public revoked token without plaintext, got %#v", revoked)
	}
	revokedCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-revoked",
		"method":  "tools/call",
		"params":  map[string]any{"name": "search_ticket", "arguments": map[string]any{"query": "T-100"}},
	}, created.Key, "access-handoff-revoked", "user:support-001")
	if revokedCall.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked handoff token to fail authentication, got status=%d body=%s", revokedCall.Code, revokedCall.Body.String())
	}
	afterRevoke := decodeData[accessHandoffResponse](t, request(t, router, http.MethodGet, permissionPackageAccessHandoffPath(input, "", "user:support-001"), nil, ""))
	if len(afterRevoke.Tokens) != 1 || afterRevoke.Tokens[0].Status != "revoked" {
		t.Fatalf("expected revoked token in handoff projection, got %#v", afterRevoke.Tokens)
	}
	rotatedRecorder := request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", tokenRequest, "")
	if rotatedRecorder.Code != http.StatusCreated {
		t.Fatalf("expected a new token after revocation, got status=%d body=%s", rotatedRecorder.Code, rotatedRecorder.Body.String())
	}
	rotated := decodeData[accessHandoffTokenCreateResponse](t, rotatedRecorder)
	audit := request(t, router, http.MethodGet, "/api/v1/audit/events?resourceId="+created.ID, nil, "")
	if strings.Contains(audit.Body.String(), created.Key) {
		t.Fatal("access handoff token plaintext leaked into audit response")
	}

	application := mustPermissionPackageApplication(t, repo, applied.Application.ID)
	staleApplication := application
	staleApplication.ID = security.NewID("ppa")
	staleApplication.TemplateVersion = application.TemplateVersion - 1
	staleApplication.AppliedAt = application.AppliedAt.Add(-time.Second)
	staleApplication, err = repo.CreatePermissionPackageApplication(t.Context(), staleApplication)
	if err != nil {
		t.Fatalf("create legacy application fixture: %v", err)
	}
	staleTemplateKey := createDirectAccessHandoffKey(t, repo, caller.ID, staleApplication, now, "handoff:"+staleApplication.ID)
	staleTemplateCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-stale-template",
		"method":  "tools/call",
		"params":  map[string]any{"name": "update_ticket", "arguments": map[string]any{"ticketId": "T-102"}},
	}, staleTemplateKey, "handoff-stale-template", "user:support-001")
	assertAccessHandoffDeniedWithTrace(t, repo, staleTemplateCall, "handoff-stale-template", "access handoff token does not allow this capability")
	staleTemplateToolsList := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-stale-template-tools-list",
		"method":  "tools/list",
	}, staleTemplateKey, "handoff-stale-template-tools-list", "user:support-001")
	assertAccessHandoffDeniedWithTrace(t, repo, staleTemplateToolsList, "handoff-stale-template-tools-list", "access handoff token does not allow this capability")

	updateTicket, ok, err := repo.GetCapability(t.Context(), updateTicket.ID)
	if err != nil || !ok {
		t.Fatalf("load applied capability before drift: ok=%v err=%v", ok, err)
	}
	updateTicket.UpdatedAt = application.AppliedAt.Add(time.Second)
	if _, ok, err = repo.UpdateCapability(t.Context(), updateTicket); err != nil || !ok {
		t.Fatalf("drift applied capability: ok=%v err=%v", ok, err)
	}
	driftedHandoff := decodeData[accessHandoffResponse](t, request(t, router, http.MethodGet, permissionPackageAccessHandoffPath(input, "", "user:support-001"), nil, ""))
	if driftedHandoff.Status != "blocked" || !containsString(driftedHandoff.ProductionReadiness.BlockingCheckCodes, "application_capabilities_current") || driftedHandoff.TokenEligibility.Eligible {
		t.Fatalf("expected capability drift after application to block readiness, got %#v", driftedHandoff)
	}
	driftedTokenRequest := request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", tokenRequest, "")
	if driftedTokenRequest.Code != http.StatusConflict || !strings.Contains(driftedTokenRequest.Body.String(), "ACCESS_HANDOFF_NOT_READY") {
		t.Fatalf("expected capability drift to block token issuance, got status=%d body=%s", driftedTokenRequest.Code, driftedTokenRequest.Body.String())
	}
	driftedRuntimeCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-stale-capability",
		"method":  "tools/call",
		"params":  map[string]any{"name": "update_ticket", "arguments": map[string]any{"ticketId": "T-103"}},
	}, rotated.Key, "handoff-stale-capability", "user:support-001")
	assertAccessHandoffDeniedWithTrace(t, repo, driftedRuntimeCall, "handoff-stale-capability", "access handoff token references a stale permission package application")
	driftedToolsList := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-stale-capability-tools-list",
		"method":  "tools/list",
	}, rotated.Key, "handoff-stale-capability-tools-list", "user:support-001")
	assertAccessHandoffDeniedWithTrace(t, repo, driftedToolsList, "handoff-stale-capability-tools-list", "access handoff token references a stale permission package application")
}

func permissionPackageAccessHandoffPath(input map[string]any, approvalRequestID string, subjectID string) string {
	return strings.Replace(permissionPackageProductionReadinessPath(input, approvalRequestID, subjectID), "/production-readiness?", "/access-handoff?", 1)
}

func accessHandoffTokenRequest(input map[string]any, handoffID string, subjectID string) map[string]any {
	return map[string]any{
		"callerInstanceId":      input["callerInstanceId"],
		"expiresInSeconds":      900,
		"handoffId":             handoffID,
		"region":                input["region"],
		"requestText":           input["requestText"],
		"requestedCapabilityId": input["requestedCapabilityId"],
		"subjectId":             subjectID,
		"subjectSelector":       input["subjectSelector"],
		"targetId":              input["targetId"],
		"templateId":            input["templateId"],
		"tenantId":              input["tenantId"],
		"workspaceId":           input["workspaceId"],
	}
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

func createDirectSubjectInstanceAssignment(
	t *testing.T,
	repo store.Repository,
	workspaceAssignmentID string,
	tenantID string,
	workspaceID string,
	callerID string,
	subjectSelector string,
	dataScopes []domain.DataScope,
	now time.Time,
) domain.InstanceAssignment {
	t.Helper()
	assignment, err := repo.CreateInstanceAssignment(t.Context(), domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: workspaceAssignmentID,
		TenantID:              tenantID,
		WorkspaceID:           workspaceID,
		CallerInstanceID:      callerID,
		SubjectSelector:       subjectSelector,
		Effect:                domain.PolicyEffectAllow,
		DataScopes:            domain.CloneDataScopes(dataScopes),
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err != nil {
		t.Fatalf("create subject-specific instance assignment: %v", err)
	}
	return assignment
}

func createDirectTestAgentKey(t *testing.T, repo store.Repository, agentID string, now time.Time) string {
	t.Helper()
	plaintext, prefix := security.NewAgentKey()
	if _, err := repo.CreateAgentKey(t.Context(), domain.AgentKey{
		ID:        security.NewID("key"),
		AgentID:   agentID,
		Name:      "ordinary-runtime-test-key",
		Hash:      security.HashSecret(plaintext),
		Prefix:    prefix,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create ordinary runtime test key: %v", err)
	}
	return plaintext
}

func createDirectAccessHandoffKey(
	t *testing.T,
	repo store.Repository,
	agentID string,
	application domain.PermissionPackageApplication,
	now time.Time,
	handoffID string,
) string {
	t.Helper()
	plaintext, prefix := security.NewAgentKey()
	if _, err := repo.CreateAgentKey(t.Context(), domain.AgentKey{
		ID:                  security.NewID("key"),
		AgentID:             agentID,
		Name:                "legacy-access-handoff-test-key",
		Hash:                security.HashSecret(plaintext),
		Prefix:              prefix,
		ApplicationID:       application.ID,
		TemplateID:          application.TemplateID,
		SubjectSelector:     application.SubjectSelector,
		CreatedForHandoffID: handoffID,
		CreatedAt:           now,
		ExpiresAt:           now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create direct access handoff key: %v", err)
	}
	return plaintext
}

func mustPermissionPackageApplication(t *testing.T, repo store.Repository, id string) domain.PermissionPackageApplication {
	t.Helper()
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{ID: id, Limit: 1})
	if err != nil || len(applications) != 1 {
		t.Fatalf("load permission package application %q: rows=%#v err=%v", id, applications, err)
	}
	return applications[0]
}

func assertAccessHandoffDeniedWithTrace(t *testing.T, repo store.Repository, recorder *httptest.ResponseRecorder, runID string, reason string) {
	t.Helper()
	if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), reason) {
		t.Fatalf("expected handoff gate denial %q, got status=%d body=%s", reason, recorder.Code, recorder.Body.String())
	}
	traces, err := repo.ListTraces(t.Context(), store.TraceFilter{RunID: runID})
	if err != nil || len(traces) != 1 {
		t.Fatalf("load handoff denial trace %q: traces=%#v err=%v", runID, traces, err)
	}
	if traces[0].Decision != domain.TraceDecisionDenied || !strings.Contains(traces[0].Reason, reason) {
		t.Fatalf("expected handoff gate reason in denial trace, got %#v", traces[0])
	}
}

func decodeMCPToolsListNames(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()
	var payload struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode MCP tools/list response: %v body=%s", err, recorder.Body.String())
	}
	names := make([]string, 0, len(payload.Result.Tools))
	for _, tool := range payload.Result.Tools {
		names = append(names, tool.Name)
	}
	return names
}
