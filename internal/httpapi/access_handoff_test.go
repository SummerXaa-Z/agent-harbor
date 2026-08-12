package httpapi_test

import (
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
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"handoff-call","result":{"ok":true}}`))
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
	blockedToken := request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", accessHandoffTokenRequest(input, "handoff:missing", "user:support-001"), "")
	if blockedToken.Code != http.StatusConflict || !strings.Contains(blockedToken.Body.String(), "ACCESS_HANDOFF_NOT_READY") {
		t.Fatalf("expected blocked token issuance before handoff readiness, got status=%d body=%s", blockedToken.Code, blockedToken.Body.String())
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

	tokenRequest := accessHandoffTokenRequest(input, after.ID, "user:support-001")
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
	allowedCall := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "handoff-allowed",
		"method":  "tools/call",
		"params":  map[string]any{"name": "search_customer", "arguments": map[string]any{"query": "Acme"}},
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
		"params":  map[string]any{"name": "search_customer", "arguments": map[string]any{"query": "Acme"}},
	}, created.Key, "access-handoff-revoked", "user:support-001")
	if revokedCall.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked handoff token to fail authentication, got status=%d body=%s", revokedCall.Code, revokedCall.Body.String())
	}
	afterRevoke := decodeData[accessHandoffResponse](t, request(t, router, http.MethodGet, permissionPackageAccessHandoffPath(input, "", "user:support-001"), nil, ""))
	if len(afterRevoke.Tokens) != 1 || afterRevoke.Tokens[0].Status != "revoked" {
		t.Fatalf("expected revoked token in handoff projection, got %#v", afterRevoke.Tokens)
	}
	rotated := request(t, router, http.MethodPost, "/api/v1/permission-packages/access-handoff/tokens", tokenRequest, "")
	if rotated.Code != http.StatusCreated {
		t.Fatalf("expected a new token after revocation, got status=%d body=%s", rotated.Code, rotated.Body.String())
	}
	audit := request(t, router, http.MethodGet, "/api/v1/audit/events?resourceId="+created.ID, nil, "")
	if strings.Contains(audit.Body.String(), created.Key) {
		t.Fatal("access handoff token plaintext leaked into audit response")
	}
}

func permissionPackageAccessHandoffPath(input map[string]any, approvalRequestID string, subjectID string) string {
	return strings.Replace(permissionPackageProductionReadinessPath(input, approvalRequestID, subjectID), "/production-readiness?", "/access-handoff?", 1)
}

func accessHandoffTokenRequest(input map[string]any, handoffID string, subjectID string) map[string]any {
	return map[string]any{
		"callerInstanceId": input["callerInstanceId"],
		"expiresInSeconds": 900,
		"handoffId":        handoffID,
		"region":           input["region"],
		"requestText":      input["requestText"],
		"subjectId":        subjectID,
		"subjectSelector":  input["subjectSelector"],
		"targetId":         input["targetId"],
		"templateId":       input["templateId"],
		"tenantId":         input["tenantId"],
		"workspaceId":      input["workspaceId"],
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
