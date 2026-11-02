package httpapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestManagementMCPAdminIdentityTools(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east"},
	})

	tools := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "tools",
		"method":  "tools/list",
	}, "", "platform-key"))
	for _, name := range []string{"list_admin_identities", "create_admin_identity", "rotate_admin_identity_key", "disable_admin_identity"} {
		if !mcpToolNamesContain(tools.Result.Tools, name) {
			t.Fatalf("admin identity tool %q missing from tools/list: %#v", name, tools.Result.Tools)
		}
	}

	create := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "create-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "create_admin_identity",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{"actor": "mcp-east", "role": "tenant_admin", "tenantId": "tenant-east"}),
		},
	}, "", "platform-key"))
	var created createAdminIdentityResponse
	if err := json.Unmarshal(create.Result.StructuredContent, &created); err != nil {
		t.Fatalf("decode create_admin_identity result: %v", err)
	}
	if created.Identity.Actor != "mcp-east" || !strings.HasPrefix(created.Key, "ahadm_") {
		t.Fatalf("unexpected create_admin_identity result: %#v", created)
	}

	rotate := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "rotate-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "rotate_admin_identity_key",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{"id": created.Identity.ID}),
		},
	}, "", "platform-key"))
	var rotated rotateAdminIdentityKeyResponse
	if err := json.Unmarshal(rotate.Result.StructuredContent, &rotated); err != nil {
		t.Fatalf("decode rotate_admin_identity_key result: %v", err)
	}
	if rotated.Key == "" || rotated.Key == created.Key || rotated.Identity.KeyPrefix == created.Identity.KeyPrefix {
		t.Fatalf("unexpected rotate_admin_identity_key result: %#v", rotated)
	}

	disable := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "disable-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "disable_admin_identity",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{"id": created.Identity.ID}),
		},
	}, "", "platform-key"))
	var disabled adminIdentityResponse
	if err := json.Unmarshal(disable.Result.StructuredContent, &disabled); err != nil {
		t.Fatalf("decode disable_admin_identity result: %v", err)
	}
	if disabled.Status != "disabled" {
		t.Fatalf("unexpected disable_admin_identity result: %#v", disabled)
	}

	rotateDisabled := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "rotate-disabled-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "rotate_admin_identity_key",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{"id": created.Identity.ID}),
		},
	}, "", "platform-key")
	if rotateDisabled.Code != http.StatusOK {
		t.Fatalf("expected MCP HTTP 200 for disabled rotation tool error, got %d body=%s", rotateDisabled.Code, rotateDisabled.Body.String())
	}
	var rotateDisabledPayload mcpEnvelopeResponse
	if err := json.Unmarshal(rotateDisabled.Body.Bytes(), &rotateDisabledPayload); err != nil {
		t.Fatalf("decode disabled rotation MCP response: %v body=%s", err, rotateDisabled.Body.String())
	}
	if rotateDisabledPayload.Error == nil || rotateDisabledPayload.Error.Data == nil ||
		rotateDisabledPayload.Error.Data.AppCode != "ADMIN_IDENTITY_DISABLED" ||
		rotateDisabledPayload.Error.Data.HTTPStatus != http.StatusConflict {
		t.Fatalf("disabled admin rotation MCP error should include app code and status, got %#v body=%s", rotateDisabledPayload, rotateDisabled.Body.String())
	}

	disableAgain := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "disable-disabled-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "disable_admin_identity",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{"id": created.Identity.ID}),
		},
	}, "", "platform-key")
	if disableAgain.Code != http.StatusOK {
		t.Fatalf("expected MCP HTTP 200 for disabled disable tool error, got %d body=%s", disableAgain.Code, disableAgain.Body.String())
	}
	var disableAgainPayload mcpEnvelopeResponse
	if err := json.Unmarshal(disableAgain.Body.Bytes(), &disableAgainPayload); err != nil {
		t.Fatalf("decode disabled disable MCP response: %v body=%s", err, disableAgain.Body.String())
	}
	if disableAgainPayload.Error == nil || disableAgainPayload.Error.Data == nil ||
		disableAgainPayload.Error.Data.AppCode != "ADMIN_IDENTITY_DISABLED" ||
		disableAgainPayload.Error.Data.HTTPStatus != http.StatusConflict {
		t.Fatalf("disabled admin disable MCP error should include app code and status, got %#v body=%s", disableAgainPayload, disableAgain.Body.String())
	}

	forbidden := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "forbidden",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_admin_identities",
			"arguments": map[string]any{},
		},
	}, "", "east-key")
	if forbidden.Code != http.StatusOK {
		t.Fatalf("expected MCP HTTP 200 for tool error, got %d body=%s", forbidden.Code, forbidden.Body.String())
	}
	var forbiddenPayload mcpEnvelopeResponse
	if err := json.Unmarshal(forbidden.Body.Bytes(), &forbiddenPayload); err != nil {
		t.Fatalf("decode forbidden MCP response: %v body=%s", err, forbidden.Body.String())
	}
	if forbiddenPayload.Error == nil || !strings.Contains(forbiddenPayload.Error.Message, "platform administrator is required") {
		t.Fatalf("scoped admin MCP call should be rejected, got %#v body=%s", forbiddenPayload, forbidden.Body.String())
	}
}

func TestExplainAccessDecisionHonorsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastCaller := createDirectAgent(t, repo, "East caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	westCapability := createDirectCapabilityWithAction(t, repo, westTarget.ID, "west_tool", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	path := "/api/v1/access-decisions:explain?tenantId=tenant-east&workspaceId=ws-support&callerInstanceId=" + eastCaller.ID + "&subjectId=role:support&targetId=" + westTarget.ID + "&capabilityId=" + westCapability.ID
	resp := requestWithAdmin(t, router, http.MethodGet, path, nil, "", "east-key")
	if resp.Code != http.StatusForbidden || strings.Contains(resp.Body.String(), "West MCP") || strings.Contains(resp.Body.String(), "west_tool") {
		t.Fatalf("scoped admin should not explain outside target metadata, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestManagementMCPInheritsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastAgent := createDirectAgent(t, repo, "East caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westCaller := createDirectAgent(t, repo, "West caller", "tenant-west", "ws-finance", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	westCapability := createDirectCapabilityWithAction(t, repo, westTarget.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	listCall := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "scoped-list-agents",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_agents",
			"arguments": map[string]any{},
		},
	}, "", "east-key"))
	var agents []agentResponse
	if err := json.Unmarshal(listCall.Result.StructuredContent, &agents); err != nil {
		t.Fatalf("decode scoped MCP agents: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != eastAgent.ID {
		t.Fatalf("scoped management MCP should only list own agents, got %#v", agents)
	}

	widen := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "scoped-list-west",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_agents",
			"arguments": map[string]any{
				"tenantId":    "tenant-west",
				"workspaceId": "ws-finance",
			},
		},
	}, "", "east-key")
	var widenEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(widen.Body.Bytes(), &widenEnvelope); err != nil {
		t.Fatalf("decode scoped MCP widen envelope: %v body=%s", err, widen.Body.String())
	}
	if widenEnvelope.Error == nil || !strings.Contains(widenEnvelope.Error.Message, "outside authenticated admin scope") {
		t.Fatalf("expected scoped MCP widen rejection, got %#v body=%s", widenEnvelope, widen.Body.String())
	}

	draftOutside := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "scoped-draft-west",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "draft_permission_package",
			"arguments": map[string]any{
				"callerInstanceId": westCaller.ID,
				"region":           "us-east",
				"requestText":      "Allow sales read access.",
				"subjectSelector":  "role:sales",
				"targetId":         westTarget.ID,
				"templateId":       "sales-readonly",
				"tenantId":         "tenant-west",
				"workspaceId":      "ws-finance",
			},
		},
	}, "", "east-key")
	var draftEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(draftOutside.Body.Bytes(), &draftEnvelope); err != nil {
		t.Fatalf("decode scoped MCP draft envelope: %v body=%s", err, draftOutside.Body.String())
	}
	if draftEnvelope.Error == nil || !strings.Contains(draftEnvelope.Error.Message, "outside authenticated admin scope") {
		t.Fatalf("expected scoped MCP draft rejection, got %#v body=%s", draftEnvelope, draftOutside.Body.String())
	}

	explainOutside := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "scoped-explain-west",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "explain_access_decision",
			"arguments": map[string]any{
				"capabilityId":     westCapability.ID,
				"callerInstanceId": eastAgent.ID,
				"subjectId":        "role:support",
				"targetId":         westTarget.ID,
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-support",
			},
		},
	}, "", "east-key")
	var explainEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(explainOutside.Body.Bytes(), &explainEnvelope); err != nil {
		t.Fatalf("decode scoped MCP explain envelope: %v body=%s", err, explainOutside.Body.String())
	}
	if explainEnvelope.Error == nil || !strings.Contains(explainEnvelope.Error.Message, "outside") ||
		strings.Contains(explainOutside.Body.String(), "West MCP") || strings.Contains(explainOutside.Body.String(), "search_customer") {
		t.Fatalf("expected scoped MCP explain rejection without outside metadata, got %#v body=%s", explainEnvelope, explainOutside.Body.String())
	}
}

func TestManagementMCPApprovalReviewerUsesAuthenticatedAdminIdentity(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 45, 0, 0, time.UTC)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-auth-east-support", "tenant-east", "ws-support", now)
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	impersonatedList := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-impersonation",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"reviewer": "security",
				"status":   "pending",
				"limit":    10,
			},
		},
	}, "", "requester-key")
	var impersonatedListEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(impersonatedList.Body.Bytes(), &impersonatedListEnvelope); err != nil {
		t.Fatalf("decode impersonated MCP list envelope: %v body=%s", err, impersonatedList.Body.String())
	}
	if impersonatedListEnvelope.Error == nil || !strings.Contains(impersonatedListEnvelope.Error.Message, "authenticated admin identity") {
		t.Fatalf("expected MCP reviewer list impersonation rejection, got %#v body=%s", impersonatedListEnvelope, impersonatedList.Body.String())
	}

	impersonatedApprove := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve-impersonation",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"id":       eastSupport.ID,
				"reviewer": "security",
			}),
		},
	}, "", "requester-key")
	var impersonatedApproveEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(impersonatedApprove.Body.Bytes(), &impersonatedApproveEnvelope); err != nil {
		t.Fatalf("decode impersonated MCP approve envelope: %v body=%s", err, impersonatedApprove.Body.String())
	}
	if impersonatedApproveEnvelope.Error == nil || !strings.Contains(impersonatedApproveEnvelope.Error.Message, "authenticated admin identity") {
		t.Fatalf("expected MCP reviewer approve impersonation rejection, got %#v body=%s", impersonatedApproveEnvelope, impersonatedApprove.Body.String())
	}
	stillPending, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), eastSupport.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after MCP impersonation attempts: ok=%v err=%v", ok, err)
	}
	if stillPending.Status != domain.PermissionPackageApprovalStatusPending || stillPending.ReviewedBy != "" {
		t.Fatalf("MCP reviewer impersonation should not mutate approval request: %#v", stillPending)
	}

	approveCall := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve-authenticated",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"reviewer": "security",
				"id":       eastSupport.ID,
			}),
		},
	}, "", "security-key"))
	var approved permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(approveCall.Result.StructuredContent, &approved); err != nil {
		t.Fatalf("decode authenticated MCP approval: %v", err)
	}
	if approved.Status != "approved" || approved.ReviewedBy != "security" {
		t.Fatalf("authenticated MCP reviewer should approve as themselves, got %#v", approved)
	}
}

func TestManagementMCPWriteToolsRequireConfirmation(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	read := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "read-admins",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_admin_identities",
			"arguments": map[string]any{},
		},
	}, "", "platform-key"))
	if read.Error != nil {
		t.Fatalf("read-only management MCP tool should not require write confirmation: %#v", read.Error)
	}

	missing := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "create-admin-missing-confirmation",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "create_admin_identity",
			"arguments": map[string]any{"actor": "mcp-east", "role": "tenant_admin", "tenantId": "tenant-east"},
		},
	}, "", "platform-key")
	var missingEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(missing.Body.Bytes(), &missingEnvelope); err != nil {
		t.Fatalf("decode missing confirmation MCP response: %v body=%s", err, missing.Body.String())
	}
	if missingEnvelope.Error == nil || missingEnvelope.Error.Data == nil ||
		missingEnvelope.Error.Data.AppCode != "VALIDATION_FAILED" ||
		!strings.Contains(missingEnvelope.Error.Message, "confirmation.confirmed") {
		t.Fatalf("write tool without confirmation should fail validation, got %#v body=%s", missingEnvelope, missing.Body.String())
	}

	blankReason := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "create-admin-blank-confirmation-reason",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "create_admin_identity",
			"arguments": map[string]any{
				"actor":    "mcp-blank-reason",
				"role":     "tenant_admin",
				"tenantId": "tenant-east",
				"confirmation": map[string]any{
					"confirmed": true,
					"reason":    "   ",
				},
			},
		},
	}, "", "platform-key")
	var blankReasonEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(blankReason.Body.Bytes(), &blankReasonEnvelope); err != nil {
		t.Fatalf("decode blank confirmation reason MCP response: %v body=%s", err, blankReason.Body.String())
	}
	if blankReasonEnvelope.Error == nil || blankReasonEnvelope.Error.Data == nil ||
		blankReasonEnvelope.Error.Data.AppCode != "VALIDATION_FAILED" ||
		!strings.Contains(blankReasonEnvelope.Error.Message, "confirmation.reason") {
		t.Fatalf("write tool with blank confirmation reason should fail validation, got %#v body=%s", blankReasonEnvelope, blankReason.Body.String())
	}

	longReason := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "create-admin-long-confirmation-reason",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "create_admin_identity",
			"arguments": map[string]any{
				"actor":    "mcp-long-reason",
				"role":     "tenant_admin",
				"tenantId": "tenant-east",
				"confirmation": map[string]any{
					"confirmed": true,
					"reason":    strings.Repeat("a", 501),
				},
			},
		},
	}, "", "platform-key")
	var longReasonEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(longReason.Body.Bytes(), &longReasonEnvelope); err != nil {
		t.Fatalf("decode long confirmation reason MCP response: %v body=%s", err, longReason.Body.String())
	}
	if longReasonEnvelope.Error == nil || longReasonEnvelope.Error.Data == nil ||
		longReasonEnvelope.Error.Data.AppCode != "VALIDATION_FAILED" ||
		!strings.Contains(longReasonEnvelope.Error.Message, "confirmation.reason") {
		t.Fatalf("write tool with long confirmation reason should fail validation, got %#v body=%s", longReasonEnvelope, longReason.Body.String())
	}

	confirmed := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "create-admin-confirmed",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "create_admin_identity",
			"arguments": map[string]any{
				"actor":    "mcp-east",
				"role":     "tenant_admin",
				"tenantId": "tenant-east",
				"confirmation": map[string]any{
					"confirmed": true,
					"reason":    "Create scoped tenant admin for the MCP write gate test.",
				},
			},
		},
	}, "", "platform-key"))
	if confirmed.Error != nil {
		t.Fatalf("confirmed write tool should pass confirmation gate: %#v", confirmed.Error)
	}

	adminEvents := decodeData[[]auditEventResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/audit/events?action=admin_identity.created", nil, "", "platform-key"))
	if len(adminEvents) != 1 {
		t.Fatalf("expected one admin identity audit event, got %#v", adminEvents)
	}
	event := adminEvents[0]
	if event.Actor != "platform" || event.ResourceType != "admin_identity" ||
		event.Metadata["managementMcpTool"] != "create_admin_identity" ||
		event.Metadata["managementMcpConfirmationReason"] != "Create scoped tenant admin for the MCP write gate test." {
		t.Fatalf("unexpected management MCP confirmation audit event: %#v", event)
	}
}

func TestManagementMCPWriteConfirmationAuditRedactsSecrets(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	secretReason := "Operator approved with Bearer confirmation-secret, apiToken=console-secret, admin key ahadm_abc123secret."
	confirmed := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "create-admin-secret-confirmation",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "create_admin_identity",
			"arguments": map[string]any{
				"actor":    "mcp-redacted",
				"role":     "tenant_admin",
				"tenantId": "tenant-east",
				"confirmation": map[string]any{
					"confirmed": true,
					"reason":    secretReason,
				},
			},
		},
	}, "", "platform-key"))
	if confirmed.Error != nil {
		t.Fatalf("confirmed write tool should pass confirmation gate: %#v", confirmed.Error)
	}

	events := decodeData[[]auditEventResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/audit/events?action=admin_identity.created", nil, "", "platform-key"))
	if len(events) != 1 {
		t.Fatalf("expected one admin identity audit event, got %#v", events)
	}
	reason, ok := events[0].Metadata["managementMcpConfirmationReason"].(string)
	if !ok || strings.Contains(reason, "confirmation-secret") ||
		strings.Contains(reason, "console-secret") ||
		strings.Contains(reason, "ahadm_abc123secret") ||
		!strings.Contains(reason, "[redacted]") {
		t.Fatalf("confirmation audit reason should redact secrets, got %#v", events[0].Metadata)
	}
}

func TestManagementMCPApprovalReviewerQueueDefaultsToAuthenticatedReviewer(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 50, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-default-east-support", "tenant-east", "ws-support", now)
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-default-east-finance", "tenant-east", "ws-finance", now.Add(time.Minute))
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "security", Key: "security-key", Role: "security_reviewer", TenantID: "tenant-east"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	securityCall := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-default-reviewer",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"status": "pending",
				"limit":  10,
			},
		},
	}, "", "security-key"))
	var securityQueue []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(securityCall.Result.StructuredContent, &securityQueue); err != nil {
		t.Fatalf("decode authenticated MCP default reviewer queue: %v", err)
	}
	if len(securityQueue) != 1 || securityQueue[0].ID != eastSupport.ID {
		t.Fatalf("authenticated MCP reviewer default queue should be routed, got %#v", securityQueue)
	}

	platformCall := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-default-platform",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"status": "pending",
				"limit":  10,
			},
		},
	}, "", "platform-key"))
	var platformQueue []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(platformCall.Result.StructuredContent, &platformQueue); err != nil {
		t.Fatalf("decode platform MCP default approval queue: %v", err)
	}
	if len(platformQueue) != 2 {
		t.Fatalf("platform admin should keep all-queue MCP visibility when reviewer is omitted, got %#v", platformQueue)
	}
}

func TestManagementMCPApprovalPendingListSkipsExpiredRequestsBeforeLimit(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 55, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	active := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-active-pending-list", "tenant-east", "ws-support", now)
	expired := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-expired-pending-list", "tenant-east", "ws-support", now.Add(time.Minute))
	expired.ExpiresAt = now.Add(-time.Minute)
	expired.UpdatedAt = expired.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), expired); err != nil || !ok {
		t.Fatalf("expire MCP approval request: ok=%v err=%v", ok, err)
	}
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "security", Key: "security-key", Role: "security_reviewer", TenantID: "tenant-east"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	call := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-expiry-filter",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"status": "pending",
				"limit":  1,
			},
		},
	}, "", "security-key"))
	var approvals []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(call.Result.StructuredContent, &approvals); err != nil {
		t.Fatalf("decode MCP expiry-filtered approvals structured content: %v", err)
	}
	if len(approvals) != 1 || approvals[0].ID != active.ID {
		t.Fatalf("MCP pending queue should skip expired request before limit, got %#v", approvals)
	}
}

func TestManagementMCPPermissionPackageApprovalReviewerRouting(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 11, 0, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-east-support", "tenant-east", "ws-support", now)
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-west-support", "tenant-west", "ws-support", now.Add(time.Minute))
	router := newRouterWithRepoAndApprovalReviewers(repo, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security-east", TenantID: "tenant-east", WorkspaceID: "ws-support"},
		{Reviewer: "security-west", TenantID: "tenant-west", WorkspaceID: "ws-support"},
	})

	listCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-routed",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"reviewer": "security-east",
				"status":   "pending",
				"limit":    10,
			},
		},
	}, ""))
	var approvals []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(listCall.Result.StructuredContent, &approvals); err != nil {
		t.Fatalf("decode routed approvals structured content: %v", err)
	}
	if len(approvals) != 1 || approvals[0].ID != eastSupport.ID {
		t.Fatalf("expected routed MCP approval queue, got %#v", approvals)
	}

	unauthorized := request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve-denied",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"id":       eastSupport.ID,
				"reviewer": "security-west",
			}),
		},
	}, "")
	var unauthorizedEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(unauthorized.Body.Bytes(), &unauthorizedEnvelope); err != nil {
		t.Fatalf("decode unauthorized MCP envelope: %v body=%s", err, unauthorized.Body.String())
	}
	if unauthorizedEnvelope.Error == nil || !strings.Contains(unauthorizedEnvelope.Error.Message, "not allowed") {
		t.Fatalf("expected routed MCP approval rejection, got %#v body=%s", unauthorizedEnvelope, unauthorized.Body.String())
	}

	approveCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve-routed",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"id":       eastSupport.ID,
				"reviewer": "security-east",
				"comment":  "approved through routed MCP queue",
			}),
		},
	}, ""))
	var approved permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(approveCall.Result.StructuredContent, &approved); err != nil {
		t.Fatalf("decode routed MCP approval result: %v", err)
	}
	if approved.Status != "approved" || approved.ReviewedBy != "security-east" {
		t.Fatalf("unexpected routed MCP approval result: %#v", approved)
	}
}

func TestManagementMCPToolsListAndPermissionPackageCalls(t *testing.T) {
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
	exportContracts := createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	tools := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "tools-list",
		"method":  "tools/list",
	}, ""))
	if tools.Result.MetadataVersion != 4 {
		t.Fatalf("management MCP tools/list metadataVersion=%d want 4", tools.Result.MetadataVersion)
	}
	if len(tools.Result.CatalogDigest) != 64 {
		t.Fatalf("management MCP tools/list should include catalog digest, got %q", tools.Result.CatalogDigest)
	}
	info := decodeData[systemInfoResponse](t, request(t, router, http.MethodGet, "/api/v1/system/info", nil, ""))
	if info.ManagementMcpToolCatalog.CatalogDigest != tools.Result.CatalogDigest {
		t.Fatalf("system info catalog digest %q does not match tools/list digest %q", info.ManagementMcpToolCatalog.CatalogDigest, tools.Result.CatalogDigest)
	}
	if !mcpToolNamesContain(tools.Result.Tools, "draft_permission_package") ||
		!mcpToolNamesContain(tools.Result.Tools, "preflight_permission_package") ||
		!mcpToolNamesContain(tools.Result.Tools, "apply_permission_package") ||
		!mcpToolNamesContain(tools.Result.Tools, "create_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "list_permission_package_approval_requests") ||
		!mcpToolNamesContain(tools.Result.Tools, "approve_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "reject_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "withdraw_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "list_permission_package_applications") ||
		!mcpToolNamesContain(tools.Result.Tools, "check_permission_package_production_readiness") ||
		!mcpToolNamesContain(tools.Result.Tools, "export_permission_package_production_report") ||
		!mcpToolNamesContain(tools.Result.Tools, "export_permission_package_production_evidence") {
		t.Fatalf("management MCP tools missing permission package tools: %#v", tools.Result.Tools)
	}
	for _, toolName := range []string{
		"draft_permission_package",
		"preflight_permission_package",
		"apply_permission_package",
		"create_permission_package_approval_request",
		"list_permission_package_approval_requests",
		"list_permission_package_applications",
		"check_permission_package_production_readiness",
		"export_permission_package_production_report",
	} {
		assertMCPToolInputSchemaHasProperty(t, tools.Result.Tools, toolName, "requestedCapabilityId")
	}
	for _, tool := range tools.Result.Tools {
		if tool.Safety.OperationType == "" || tool.Safety.OperationType == "unspecified" ||
			tool.Safety.ApprovalMode == "" || tool.Safety.ApprovalMode == "unspecified" {
			t.Fatalf("management MCP tool %q missing safety metadata: %#v", tool.Name, tool)
		}
		if tool.Access.RequiredRole == "" || tool.Access.RequiredRole == "unspecified" ||
			tool.Access.ScopeBoundary == "" || tool.Access.ScopeBoundary == "unspecified" {
			t.Fatalf("management MCP tool %q missing access metadata: %#v", tool.Name, tool)
		}
		if tool.Lifecycle.Status == "" || tool.Lifecycle.Status == "unspecified" {
			t.Fatalf("management MCP tool %q missing lifecycle metadata: %#v", tool.Name, tool)
		}
		if tool.Execution.Idempotency == "" || tool.Execution.Idempotency == "unspecified" {
			t.Fatalf("management MCP tool %q missing execution metadata: %#v", tool.Name, tool)
		}
		if tool.Execution.ConfirmationRequired {
			assertMCPToolInputSchemaRequiresConfirmation(t, tool)
		} else {
			assertMCPToolInputSchemaDoesNotExposeConfirmation(t, tool)
		}
	}
	assertMCPToolSafety(t, tools.Result.Tools, "explain_access_decision", mcpToolSafety{
		OperationType: "read",
		ReadOnly:      true,
		MutatesState:  false,
		ApprovalMode:  "none",
	})
	assertMCPToolSafety(t, tools.Result.Tools, "draft_permission_package", mcpToolSafety{
		OperationType: "preview",
		ReadOnly:      true,
		MutatesState:  false,
		ApprovalMode:  "none",
	})
	assertMCPToolSafety(t, tools.Result.Tools, "apply_permission_package", mcpToolSafety{
		OperationType: "write",
		ReadOnly:      false,
		MutatesState:  true,
		ApprovalMode:  "conditional",
	})
	assertMCPToolSafety(t, tools.Result.Tools, "approve_permission_package_approval_request", mcpToolSafety{
		OperationType: "write",
		ReadOnly:      false,
		MutatesState:  true,
		ApprovalMode:  "reviewer",
	})
	assertMCPToolAccess(t, tools.Result.Tools, "list_admin_identities", mcpToolAccess{
		RequiredRole:  "platform_admin",
		ScopeBoundary: "platform",
		ReviewerBound: false,
	})
	assertMCPToolAccess(t, tools.Result.Tools, "draft_permission_package", mcpToolAccess{
		RequiredRole:  "authenticated_admin",
		ScopeBoundary: "requested_scope",
		ReviewerBound: false,
	})
	assertMCPToolAccess(t, tools.Result.Tools, "approve_permission_package_approval_request", mcpToolAccess{
		RequiredRole:  "authenticated_admin",
		ScopeBoundary: "reviewer_route",
		ReviewerBound: true,
	})
	assertMCPToolLifecycle(t, tools.Result.Tools, "draft_permission_package", mcpToolLifecycle{
		Status: "active",
	})
	assertMCPToolLifecycle(t, tools.Result.Tools, "export_permission_package_production_evidence", mcpToolLifecycle{
		Status:        "compatibility_alias",
		PreferredName: "export_permission_package_production_report",
	})
	assertMCPToolExecution(t, tools.Result.Tools, "explain_access_decision", mcpToolExecution{
		Idempotency: "safe_repeat",
	})
	assertMCPToolExecution(t, tools.Result.Tools, "create_admin_identity", mcpToolExecution{
		Idempotency:          "not_idempotent",
		ConfirmationRequired: true,
		AuditResourceType:    "admin_identity",
		ReturnsSecret:        true,
	})
	assertMCPToolExecution(t, tools.Result.Tools, "rotate_admin_identity_key", mcpToolExecution{
		Idempotency:          "not_idempotent",
		ConfirmationRequired: true,
		AuditResourceType:    "admin_identity",
		ReturnsSecret:        true,
	})
	assertMCPToolExecution(t, tools.Result.Tools, "apply_permission_package", mcpToolExecution{
		Idempotency:          "conditional_repeat",
		ConfirmationRequired: true,
		PreflightTool:        "preflight_permission_package",
		AuditResourceType:    "permission_package_application",
	})
	assertMCPToolExecution(t, tools.Result.Tools, "approve_permission_package_approval_request", mcpToolExecution{
		Idempotency:          "conditional_repeat",
		ConfirmationRequired: true,
		AuditResourceType:    "permission_package_approval_request",
	})
	assertMCPToolDescriptionDoesNotContain(t, tools.Result.Tools, "export_permission_package_production_report", "evidence", "证据")
	assertMCPToolDescriptionDoesNotContain(t, tools.Result.Tools, "export_permission_package_production_evidence", "evidence", "证据")

	args := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	draftCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "draft",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "draft_permission_package",
			"arguments": args,
		},
	}, ""))
	var draft permissionPackageDraftResponse
	if err := json.Unmarshal(draftCall.Result.StructuredContent, &draft); err != nil {
		t.Fatalf("decode draft structured content: %v", err)
	}
	if !draft.Readiness.CanApply || len(draft.AllowedCapabilities) != 1 || draft.AllowedCapabilities[0].ID != search.ID {
		t.Fatalf("unexpected management MCP draft: %#v", draft)
	}

	preflightCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "preflight",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "preflight_permission_package",
			"arguments": args,
		},
	}, ""))
	var preflight permissionPackageApplyPreflightResponse
	if err := json.Unmarshal(preflightCall.Result.StructuredContent, &preflight); err != nil {
		t.Fatalf("decode preflight structured content: %v", err)
	}
	if !preflight.Summary.CanApply || preflight.Summary.BlockingCount != 0 ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "planned_changes", "info") {
		t.Fatalf("unexpected management MCP preflight: %#v", preflight)
	}
	if !slices.Contains(preflight.NextActionCodes, "apply_permission_package") {
		t.Fatalf("expected management MCP preflight next action code, got %#v", preflight.NextActionCodes)
	}
	entitlements, err := repo.ListTenantEntitlements(t.Context(), store.EntitlementFilter{})
	if err != nil {
		t.Fatalf("list entitlements after MCP preflight: %v", err)
	}
	if len(entitlements) != 0 {
		t.Fatalf("management MCP preflight must not create entitlements: %#v", entitlements)
	}

	appliedCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "apply",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "apply_permission_package",
			"arguments": withManagementMCPWriteConfirmation(args),
		},
	}, ""))
	var applied permissionPackageApplyResponse
	if err := json.Unmarshal(appliedCall.Result.StructuredContent, &applied); err != nil {
		t.Fatalf("decode apply structured content: %v", err)
	}
	if len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != search.ID {
		t.Fatalf("expected one applied entitlement, got %#v", applied.TenantEntitlements)
	}
	applicationsCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "applications",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_applications",
			"arguments": map[string]any{
				"tenantId":         "tenant-root",
				"workspaceId":      "ws-sales",
				"templateId":       "sales-readonly",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"limit":            1,
			},
		},
	}, ""))
	var applications []permissionPackageApplicationResponse
	if err := json.Unmarshal(applicationsCall.Result.StructuredContent, &applications); err != nil {
		t.Fatalf("decode applications structured content: %v", err)
	}
	if applied.Application == nil || len(applications) != 1 || applications[0].ID != applied.Application.ID || applications[0].TemplateVersion != 2 {
		t.Fatalf("unexpected management MCP applications: applied=%#v rows=%#v", applied.Application, applications)
	}
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, exportContracts, "export_contracts", "user:sales-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, search, "search_customer", "user:sales-001", now.Add(2*time.Minute))
	readinessCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "production-readiness",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "check_permission_package_production_readiness",
			"arguments": map[string]any{
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-sales",
				"templateId":       "sales-readonly",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"subjectId":        "user:sales-001",
			},
		},
	}, ""))
	var readiness permissionPackageProductionReadinessResponse
	if err := json.Unmarshal(readinessCall.Result.StructuredContent, &readiness); err != nil {
		t.Fatalf("decode production readiness structured content: %v", err)
	}
	if readiness.Status != "ready" || readiness.LatestApplication == nil || readiness.LatestApplication.ID != applied.Application.ID ||
		!permissionPackageProductionReadinessHasCheck(readiness.Checks, "runtime_allowed_trace_present", "passed") ||
		!permissionPackageProductionReadinessHasCheck(readiness.Checks, "applied_audit_event_present", "passed") {
		t.Fatalf("unexpected production readiness MCP result: %#v", readiness)
	}
	reportCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "production-report",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "export_permission_package_production_report",
			"arguments": map[string]any{
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-sales",
				"templateId":       "sales-readonly",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"subjectId":        "user:sales-001",
			},
		},
	}, ""))
	var report permissionPackageProductionEvidenceReportResponse
	if err := json.Unmarshal(reportCall.Result.StructuredContent, &report); err != nil {
		t.Fatalf("decode production evidence report structured content: %v", err)
	}
	if report.ReportVersion != "production-readiness-report/v1" || report.Status != "ready" ||
		report.Evidence.Application.ID != applied.Application.ID ||
		report.Evidence.Runtime.AllowedTraceID == "" ||
		report.Evidence.Audit.AppliedEventID == "" {
		t.Fatalf("unexpected production evidence MCP report: %#v", report)
	}
	if report.PlatformContract.APIVersion != "2026-06-15" ||
		report.PlatformContract.ManagementMcpToolCatalog.MetadataVersion != tools.Result.MetadataVersion ||
		report.PlatformContract.ManagementMcpToolCatalog.CatalogDigest != tools.Result.CatalogDigest {
		t.Fatalf("unexpected production report platform contract from MCP: report=%#v tools=%#v", report.PlatformContract, tools.Result)
	}
	if report.GeneratedBy != "local-dev" {
		t.Fatalf("unexpected production report generatedBy from MCP: %q", report.GeneratedBy)
	}
	assertProductionReportDigestFromRaw(t, reportCall.Result.StructuredContent, report.ReportDigest)
	legacyReportCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "legacy-production-report",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "export_permission_package_production_evidence",
			"arguments": map[string]any{
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-sales",
				"templateId":       "sales-readonly",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"subjectId":        "user:sales-001",
			},
		},
	}, ""))
	var legacyReport permissionPackageProductionEvidenceReportResponse
	if err := json.Unmarshal(legacyReportCall.Result.StructuredContent, &legacyReport); err != nil {
		t.Fatalf("decode legacy production report structured content: %v", err)
	}
	if legacyReport.ReportVersion != report.ReportVersion || legacyReport.Status != report.Status ||
		legacyReport.Evidence.Application.ID != report.Evidence.Application.ID {
		t.Fatalf("legacy production report alias changed report semantics: new=%#v legacy=%#v", report, legacyReport)
	}
	events := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?action=permission_package.applied", nil, ""))
	if len(events) != 1 || events[0].ResourceType != "permission_package" {
		t.Fatalf("expected permission package audit event, got %#v", events)
	}
}

func TestManagementMCPPermissionPackageApprovalRequestFlow(t *testing.T) {
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

	args := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	draftCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "draft",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "draft_permission_package",
			"arguments": args,
		},
	}, ""))
	var draft permissionPackageDraftResponse
	if err := json.Unmarshal(draftCall.Result.StructuredContent, &draft); err != nil {
		t.Fatalf("decode draft structured content: %v", err)
	}
	if !draft.Readiness.CanApply || draft.PolicyGate.Decision != "approval_required" || draft.PolicyGate.CanApplyDirectly {
		t.Fatalf("expected approval-required management MCP draft, got %#v", draft)
	}

	missingApprovalPreflightCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "preflight-missing-approval",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "preflight_permission_package",
			"arguments": args,
		},
	}, ""))
	var missingApprovalPreflight permissionPackageApplyPreflightResponse
	if err := json.Unmarshal(missingApprovalPreflightCall.Result.StructuredContent, &missingApprovalPreflight); err != nil {
		t.Fatalf("decode missing approval preflight structured content: %v", err)
	}
	if missingApprovalPreflight.Summary.CanApply || !permissionPackagePreflightHasCheck(missingApprovalPreflight.Checks, "approval_request_missing", "blocking") {
		t.Fatalf("expected missing approval MCP preflight to block, got %#v", missingApprovalPreflight)
	}

	createCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-create",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "create_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(args),
		},
	}, ""))
	var approval permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(createCall.Result.StructuredContent, &approval); err != nil {
		t.Fatalf("decode approval structured content: %v", err)
	}
	if approval.Status != "pending" || approval.TemplateID != "support-ticket-triage" ||
		len(approval.AllowedCapabilityIDs) != 1 || approval.AllowedCapabilityIDs[0] != updateTicket.ID {
		t.Fatalf("unexpected management MCP approval request: %#v", approval)
	}
	duplicateCreateCall := request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-create-duplicate",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "create_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(args),
		},
	}, "")
	if duplicateCreateCall.Code != http.StatusOK {
		t.Fatalf("expected MCP HTTP 200 for duplicate approval tool error, got %d body=%s", duplicateCreateCall.Code, duplicateCreateCall.Body.String())
	}
	var duplicateCreatePayload mcpEnvelopeResponse
	if err := json.Unmarshal(duplicateCreateCall.Body.Bytes(), &duplicateCreatePayload); err != nil {
		t.Fatalf("decode duplicate approval MCP response: %v body=%s", err, duplicateCreateCall.Body.String())
	}
	if duplicateCreatePayload.Error == nil || duplicateCreatePayload.Error.Data == nil ||
		duplicateCreatePayload.Error.Data.AppCode != "PERMISSION_PACKAGE_APPROVAL_ALREADY_PENDING" ||
		duplicateCreatePayload.Error.Data.HTTPStatus != http.StatusConflict {
		t.Fatalf("duplicate approval MCP error should include app code and status, got %#v body=%s", duplicateCreatePayload, duplicateCreateCall.Body.String())
	}

	listCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"tenantId":         "tenant-root",
				"workspaceId":      "ws-support",
				"templateId":       "support-ticket-triage",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"status":           "pending",
				"limit":            1,
			},
		},
	}, ""))
	var approvals []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(listCall.Result.StructuredContent, &approvals); err != nil {
		t.Fatalf("decode approvals structured content: %v", err)
	}
	if len(approvals) != 1 || approvals[0].ID != approval.ID {
		t.Fatalf("unexpected management MCP approval request list: %#v", approvals)
	}

	withdrawArgs := map[string]any{}
	for key, value := range args {
		withdrawArgs[key] = value
	}
	withdrawArgs["requestText"] = "Allow replacement support triage updates for this tenant."
	withdrawCreateCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-create-withdraw",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "create_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(withdrawArgs),
		},
	}, ""))
	var withdrawApproval permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(withdrawCreateCall.Result.StructuredContent, &withdrawApproval); err != nil {
		t.Fatalf("decode withdraw approval structured content: %v", err)
	}
	withdrawCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-withdraw",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "withdraw_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"id":      withdrawApproval.ID,
				"comment": "replaced by narrower request",
			}),
		},
	}, ""))
	var withdrawn permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(withdrawCall.Result.StructuredContent, &withdrawn); err != nil {
		t.Fatalf("decode withdrawn structured content: %v", err)
	}
	if withdrawn.Status != "withdrawn" || withdrawn.ReviewedBy != "local-dev" || withdrawn.ReviewComment != "replaced by narrower request" {
		t.Fatalf("unexpected management MCP withdrawn request: %#v", withdrawn)
	}

	approveCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": withManagementMCPWriteConfirmation(map[string]any{
				"id":       approval.ID,
				"reviewer": "security",
				"comment":  "approved via MCP",
			}),
		},
	}, ""))
	var approved permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(approveCall.Result.StructuredContent, &approved); err != nil {
		t.Fatalf("decode approved structured content: %v", err)
	}
	if approved.Status != "approved" || approved.ReviewedBy != "security" {
		t.Fatalf("unexpected management MCP approved request: %#v", approved)
	}

	applyArgs := map[string]any{
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
	approvedPreflightCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "preflight-approved",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "preflight_permission_package",
			"arguments": applyArgs,
		},
	}, ""))
	var approvedPreflight permissionPackageApplyPreflightResponse
	if err := json.Unmarshal(approvedPreflightCall.Result.StructuredContent, &approvedPreflight); err != nil {
		t.Fatalf("decode approved preflight structured content: %v", err)
	}
	if !approvedPreflight.Summary.CanApply || !approvedPreflight.Summary.ApprovalReady ||
		!permissionPackagePreflightHasCheck(approvedPreflight.Checks, "approval_request_ready", "passed") {
		t.Fatalf("expected approved MCP preflight to pass, got %#v", approvedPreflight)
	}
	loadedApproval, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after MCP preflight: ok=%v err=%v", ok, err)
	}
	if !loadedApproval.ConsumedAt.IsZero() {
		t.Fatalf("MCP preflight must not consume approval, got %#v", loadedApproval)
	}

	appliedCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "apply",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "apply_permission_package",
			"arguments": withManagementMCPWriteConfirmation(applyArgs),
		},
	}, ""))
	var applied permissionPackageApplyResponse
	if err := json.Unmarshal(appliedCall.Result.StructuredContent, &applied); err != nil {
		t.Fatalf("decode apply structured content: %v", err)
	}
	if len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != updateTicket.ID ||
		applied.Application == nil || applied.Application.TemplateID != "support-ticket-triage" {
		t.Fatalf("expected approved management MCP apply, got %#v", applied)
	}
	events := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?action=permission_package.applied", nil, ""))
	if len(events) != 1 || events[0].Metadata["approvalRequestId"] != approval.ID {
		t.Fatalf("expected management MCP applied audit with approval request id, got %#v", events)
	}
}

func TestManagementMCPReadTools(t *testing.T) {
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
	capability := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	agentsCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "agents",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_agents",
			"arguments": map[string]any{
				"tenantId": "tenant-east",
			},
		},
	}, ""))
	var agents []agentResponse
	if err := json.Unmarshal(agentsCall.Result.StructuredContent, &agents); err != nil {
		t.Fatalf("decode agents structured content: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != caller.ID {
		t.Fatalf("unexpected agent list: %#v", agents)
	}

	capabilitiesCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "capabilities",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_capabilities",
			"arguments": map[string]any{
				"targetId": target.ID,
			},
		},
	}, ""))
	var capabilities []capabilityResponse
	if err := json.Unmarshal(capabilitiesCall.Result.StructuredContent, &capabilities); err != nil {
		t.Fatalf("decode capabilities structured content: %v", err)
	}
	if len(capabilities) != 1 || capabilities[0].ID != capability.ID {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}

	profileCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "profile",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_tenant_access_profile",
			"arguments": map[string]any{
				"tenantId":    "tenant-east",
				"workspaceId": "ws-sales",
			},
		},
	}, ""))
	var profile struct {
		Summary struct {
			TenantCount int `json:"tenantCount"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(profileCall.Result.StructuredContent, &profile); err != nil {
		t.Fatalf("decode profile structured content: %v", err)
	}
	if profile.Summary.TenantCount != 1 {
		t.Fatalf("unexpected profile summary: %#v", profile.Summary)
	}
}

func TestManagementMCPExplainPermissionPackageDraft(t *testing.T) {
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
	createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	call := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "explain-draft",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "explain_permission_package_draft",
			"arguments": map[string]any{
				"callerInstanceId": caller.ID,
				"region":           "eu-west",
				"requestText":      "给销售助手开通客户只读。",
				"subjectSelector":  "user:sales-*",
				"targetId":         target.ID,
				"templateId":       "sales-readonly",
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-sales",
			},
		},
	}, ""))
	var explanation managementMCPExplainPermissionPackageResponse
	if err := json.Unmarshal(call.Result.StructuredContent, &explanation); err != nil {
		t.Fatalf("decode draft explanation: %v", err)
	}
	if explanation.Outcome != "blocked" || explanation.Readiness.CanApply {
		t.Fatalf("expected blocked draft explanation, got %#v", explanation)
	}
	if !containsString(explanation.Readiness.Warnings, "Permission package data scopes exceed capability boundary for search_customer.") {
		t.Fatalf("expected data-scope warning, got %#v", explanation.Readiness.Warnings)
	}
	if len(explanation.BlockedSimulationRows) == 0 || len(explanation.NextActions) == 0 || explanation.Summary == "" {
		t.Fatalf("expected blocked rows, next actions, and summary, got %#v", explanation)
	}
	if !slices.Contains(explanation.NextActionCodes, "narrow_data_scope") {
		t.Fatalf("expected narrow data-scope next action code, got %#v", explanation.NextActionCodes)
	}
}

func TestManagementMCPExplainPermissionPackageDraftRequiresApproval(t *testing.T) {
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

	call := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "explain-approval",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "explain_permission_package_draft",
			"arguments": map[string]any{
				"callerInstanceId": caller.ID,
				"region":           "us-east",
				"requestText":      "Allow support triage updates for this tenant.",
				"subjectSelector":  "user:support-*",
				"targetId":         target.ID,
				"templateId":       "support-ticket-triage",
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-support",
			},
		},
	}, ""))
	var explanation managementMCPExplainPermissionPackageResponse
	if err := json.Unmarshal(call.Result.StructuredContent, &explanation); err != nil {
		t.Fatalf("decode approval draft explanation: %v", err)
	}
	if explanation.Outcome != "approval_required" || !explanation.Readiness.CanApply {
		t.Fatalf("expected approval-required ready draft explanation, got %#v", explanation)
	}
	if explanation.PolicyGate.Decision != "approval_required" || explanation.PolicyGate.CanApplyDirectly || len(explanation.PolicyGate.Reasons) == 0 {
		t.Fatalf("expected policy gate in MCP explanation, got %#v", explanation.PolicyGate)
	}
	if !slices.Contains(explanation.PolicyGate.NextActionCodes, "create_approval_request") {
		t.Fatalf("expected policy gate next action code in MCP explanation, got %#v", explanation.PolicyGate.NextActionCodes)
	}
	if !strings.Contains(explanation.Summary, "requires approval") {
		t.Fatalf("expected approval summary, got %q", explanation.Summary)
	}
	if !strings.Contains(strings.Join(explanation.NextActions, " "), "approval") {
		t.Fatalf("expected approval next action, got %#v", explanation.NextActions)
	}
	if !slices.Contains(explanation.NextActionCodes, "create_approval_request") {
		t.Fatalf("expected approval next action code, got %#v", explanation.NextActionCodes)
	}
	if slices.Contains(explanation.NextActionCodes, "apply_permission_package") {
		t.Fatalf("approval-required draft should not suggest direct apply code, got %#v", explanation.NextActionCodes)
	}
	if strings.Contains(strings.Join(explanation.NextActions, " "), "Apply the permission package") {
		t.Fatalf("approval-required draft should not suggest direct apply, got %#v", explanation.NextActions)
	}
}

func TestManagementMCPExplainAccessDecision(t *testing.T) {
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
	search.DiscoveryStatus = domain.CapabilityDiscoveryApproved
	search.UpdatedAt = now
	if _, ok, err := repo.UpdateCapability(t.Context(), search); err != nil || !ok {
		t.Fatalf("approve capability: ok=%v err=%v", ok, err)
	}

	explainArgs := map[string]any{
		"callerInstanceId": caller.ID,
		"capabilityId":     search.ID,
		"subjectId":        "user:sales-001",
		"targetId":         target.ID,
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	deniedCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "explain-denied",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "explain_access_decision",
			"arguments": explainArgs,
		},
	}, ""))
	var denied managementMCPExplainAccessResponse
	if err := json.Unmarshal(deniedCall.Result.StructuredContent, &denied); err != nil {
		t.Fatalf("decode denied explanation: %v", err)
	}
	if denied.Outcome != "denied" || denied.Decision.Allowed || denied.Decision.Reason != "tenant has no entitlement for capability" {
		t.Fatalf("unexpected denied explanation: %#v", denied)
	}
	if !strings.Contains(strings.Join(denied.NextActions, " "), "permission package") {
		t.Fatalf("expected permission package next action, got %#v", denied.NextActions)
	}
	if !slices.Contains(denied.NextActionCodes, "use_permission_package") {
		t.Fatalf("expected permission package next action code, got %#v", denied.NextActionCodes)
	}
	if slices.Contains(denied.NextActionCodes, "no_change_required") {
		t.Fatalf("denied decision should not suggest no-change code, got %#v", denied.NextActionCodes)
	}

	packageArgs := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	_ = decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "apply",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "apply_permission_package",
			"arguments": withManagementMCPWriteConfirmation(packageArgs),
		},
	}, ""))

	allowedCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "explain-allowed",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "explain_access_decision",
			"arguments": explainArgs,
		},
	}, ""))
	var allowed managementMCPExplainAccessResponse
	if err := json.Unmarshal(allowedCall.Result.StructuredContent, &allowed); err != nil {
		t.Fatalf("decode allowed explanation: %v", err)
	}
	if allowed.Outcome != "allowed" || !allowed.Decision.Allowed || len(allowed.DataScopes) == 0 {
		t.Fatalf("unexpected allowed explanation: %#v", allowed)
	}
	if !slices.Contains(allowed.NextActionCodes, "no_change_required") {
		t.Fatalf("expected no-change next action code, got %#v", allowed.NextActionCodes)
	}
	if !explainEvidenceContains(allowed.Evidence, "tenant_entitlement") ||
		!explainEvidenceContains(allowed.Evidence, "workspace_assignment") ||
		!explainEvidenceContains(allowed.Evidence, "instance_assignment") {
		t.Fatalf("expected entitlement/workspace/instance evidence, got %#v", allowed.Evidence)
	}
}

func TestAccessDecisionExplain(t *testing.T) {
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
	search.DiscoveryStatus = domain.CapabilityDiscoveryApproved
	search.UpdatedAt = now
	if _, ok, err := repo.UpdateCapability(t.Context(), search); err != nil || !ok {
		t.Fatalf("approve capability: ok=%v err=%v", ok, err)
	}

	explainPath := "/api/v1/access-decisions:explain?tenantId=tenant-east&workspaceId=ws-sales&callerInstanceId=" + caller.ID + "&targetId=" + target.ID + "&capabilityId=" + search.ID + "&subjectId=user:sales-001"
	denied := decodeData[managementMCPExplainAccessResponse](t, request(t, router, http.MethodGet, explainPath, nil, ""))
	if denied.Outcome != "denied" || denied.Decision.Allowed || denied.Decision.Reason != "tenant has no entitlement for capability" {
		t.Fatalf("unexpected denied explanation: %#v", denied)
	}
	if !strings.Contains(strings.Join(denied.NextActions, " "), "permission package") {
		t.Fatalf("expected permission package next action, got %#v", denied.NextActions)
	}
	if !slices.Contains(denied.NextActionCodes, "use_permission_package") {
		t.Fatalf("expected permission package next action code, got %#v", denied.NextActionCodes)
	}

	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}, ""))
	if applied.Application == nil {
		t.Fatalf("expected permission package application evidence")
	}

	allowed := decodeData[managementMCPExplainAccessResponse](t, request(t, router, http.MethodGet, explainPath, nil, ""))
	if allowed.Outcome != "allowed" || !allowed.Decision.Allowed || len(allowed.DataScopes) == 0 {
		t.Fatalf("unexpected allowed explanation: %#v", allowed)
	}
	if !slices.Contains(allowed.NextActionCodes, "no_change_required") {
		t.Fatalf("expected no-change next action code, got %#v", allowed.NextActionCodes)
	}
	if !explainEvidenceContains(allowed.Evidence, "tenant_entitlement") ||
		!explainEvidenceContains(allowed.Evidence, "workspace_assignment") ||
		!explainEvidenceContains(allowed.Evidence, "instance_assignment") {
		t.Fatalf("expected entitlement/workspace/instance evidence, got %#v", allowed.Evidence)
	}
}

func TestAccessDecisionExplainRequiresSubject(t *testing.T) {
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

	// A subjectless query can never match subject-scoped assignments, so it must be
	// rejected up front instead of returning a confident-looking deny.
	path := "/api/v1/access-decisions:explain?tenantId=tenant-east&workspaceId=ws-sales&callerInstanceId=" + caller.ID + "&targetId=" + target.ID + "&capabilityId=" + search.ID
	resp := request(t, router, http.MethodGet, path, nil, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("subjectless explain should fail validation, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "subjectId") {
		t.Fatalf("validation error should name the missing subjectId, body=%s", resp.Body.String())
	}
}
