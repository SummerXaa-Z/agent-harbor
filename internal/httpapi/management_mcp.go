package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/permissionpack"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type managementMCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"params"`
}

type managementMCPResponse struct {
	JSONRPC string              `json:"jsonrpc"`
	ID      json.RawMessage     `json:"id"`
	Result  any                 `json:"result,omitempty"`
	Error   *managementMCPError `json:"error,omitempty"`
}

type managementMCPError struct {
	Code    int                     `json:"code"`
	Message string                  `json:"message"`
	Data    *managementMCPErrorData `json:"data,omitempty"`
}

type managementMCPErrorData struct {
	AppCode    string `json:"appCode,omitempty"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
}

type managementMCPToolsListResult struct {
	Tools []managementMCPTool `json:"tools"`
}

type managementMCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type managementMCPCallResult struct {
	Content           []managementMCPContentItem `json:"content"`
	StructuredContent any                        `json:"structuredContent,omitempty"`
}

type managementMCPContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type managementMCPScopeArgs struct {
	TenantID    string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
}

type managementMCPCreateAdminIdentityArgs struct {
	Actor       string `json:"actor"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	TenantID    string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
}

type managementMCPAdminIdentityIDArgs struct {
	ID string `json:"id"`
}

type managementMCPCapabilityArgs struct {
	TenantID    string                           `json:"tenantId"`
	WorkspaceID string                           `json:"workspaceId"`
	TargetID    string                           `json:"targetId"`
	Status      domain.CapabilityDiscoveryStatus `json:"status"`
}

type managementMCPPermissionPackageApplicationArgs struct {
	TenantID         string `json:"tenantId"`
	WorkspaceID      string `json:"workspaceId"`
	TemplateID       string `json:"templateId"`
	TargetID         string `json:"targetId"`
	CallerInstanceID string `json:"callerInstanceId"`
	Limit            *int   `json:"limit"`
}

type managementMCPPermissionPackageProductionReadinessArgs struct {
	TenantID          string `json:"tenantId"`
	WorkspaceID       string `json:"workspaceId"`
	TemplateID        string `json:"templateId"`
	TargetID          string `json:"targetId"`
	CallerInstanceID  string `json:"callerInstanceId"`
	SubjectID         string `json:"subjectId"`
	Region            string `json:"region"`
	RequestText       string `json:"requestText"`
	SubjectSelector   string `json:"subjectSelector"`
	ApprovalRequestID string `json:"approvalRequestId"`
	TraceLimit        *int   `json:"traceLimit"`
}

type managementMCPPermissionPackageApprovalRequestArgs struct {
	TenantID         string                                 `json:"tenantId"`
	WorkspaceID      string                                 `json:"workspaceId"`
	TemplateID       string                                 `json:"templateId"`
	TargetID         string                                 `json:"targetId"`
	CallerInstanceID string                                 `json:"callerInstanceId"`
	Status           domain.PermissionPackageApprovalStatus `json:"status"`
	Reviewer         string                                 `json:"reviewer"`
	Limit            *int                                   `json:"limit"`
}

type managementMCPApprovalResolutionArgs struct {
	ID       string `json:"id"`
	Reviewer string `json:"reviewer"`
	Comment  string `json:"comment"`
}

type managementMCPAccessProfileArgs struct {
	TenantID         string `json:"tenantId"`
	WorkspaceID      string `json:"workspaceId"`
	TargetID         string `json:"targetId"`
	CapabilityID     string `json:"capabilityId"`
	CallerInstanceID string `json:"callerInstanceId"`
	TraceLimit       *int   `json:"traceLimit"`
}

type managementMCPExplainAccessArgs struct {
	TenantID         string `json:"tenantId"`
	WorkspaceID      string `json:"workspaceId"`
	CallerInstanceID string `json:"callerInstanceId"`
	SubjectID        string `json:"subjectId"`
	TargetID         string `json:"targetId"`
	CapabilityID     string `json:"capabilityId"`
}

type managementMCPExplainEvidence struct {
	Layer   string `json:"layer"`
	Status  string `json:"status"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message"`
}

type managementMCPExplainPermissionPackageResult struct {
	Outcome                string                                  `json:"outcome"`
	Summary                string                                  `json:"summary"`
	DraftID                string                                  `json:"draftId"`
	Input                  domain.PermissionPackageDraftRequest    `json:"input"`
	Readiness              domain.PermissionPackageReadiness       `json:"readiness"`
	PolicyGate             domain.PermissionPackagePolicyGate      `json:"policyGate"`
	AllowedCapabilityCount int                                     `json:"allowedCapabilityCount"`
	BlockedCapabilityCount int                                     `json:"blockedCapabilityCount"`
	BlockedSimulationRows  []domain.PermissionPackageSimulationRow `json:"blockedSimulationRows"`
	DataScopes             []domain.DataScope                      `json:"dataScopes,omitempty"`
	NextActionCodes        []string                                `json:"nextActionCodes"`
	NextActions            []string                                `json:"nextActions"`
}

type managementMCPExplainAccessResult struct {
	Outcome         string                          `json:"outcome"`
	Summary         string                          `json:"summary"`
	Request         managementMCPExplainAccessArgs  `json:"request"`
	Decision        domain.CapabilityAccessDecision `json:"decision"`
	Evidence        []managementMCPExplainEvidence  `json:"evidence"`
	DataScopes      []domain.DataScope              `json:"dataScopes,omitempty"`
	NextActionCodes []string                        `json:"nextActionCodes"`
	NextActions     []string                        `json:"nextActions"`
}

func (s *Server) managementMCP(w http.ResponseWriter, r *http.Request) {
	req, err := managementMCPRequestFromHTTP(r)
	if err != nil {
		writeManagementMCPError(w, managementMCPResponseID(nil), -32700, err.Error())
		return
	}
	switch req.Method {
	case "tools/list":
		writeManagementMCPResult(w, req.ID, managementMCPToolsListResult{Tools: managementMCPTools()})
	case "tools/call":
		result, err := s.callManagementMCPTool(r, req)
		if err != nil {
			writeManagementMCPAppError(w, req.ID, err)
			return
		}
		writeManagementMCPResult(w, req.ID, result)
	default:
		writeManagementMCPError(w, req.ID, -32601, fmt.Sprintf("method %q is not supported", req.Method))
	}
}

func managementMCPRequestFromHTTP(r *http.Request) (managementMCPRequest, error) {
	body, err := readProxyBody(r.Body)
	if err != nil {
		return managementMCPRequest{}, err
	}
	var req managementMCPRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return managementMCPRequest{}, errors.New("request body must be valid MCP JSON-RPC")
	}
	req.Method = strings.TrimSpace(req.Method)
	req.Params.Name = strings.TrimSpace(req.Params.Name)
	if len(req.ID) == 0 {
		req.ID = managementMCPResponseID(nil)
	}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return managementMCPRequest{}, errors.New("jsonrpc must be 2.0")
	}
	if req.Method == "" {
		return managementMCPRequest{}, errors.New("method is required")
	}
	return req, nil
}

func (s *Server) callManagementMCPTool(r *http.Request, req managementMCPRequest) (managementMCPCallResult, error) {
	switch req.Params.Name {
	case "list_admin_identities":
		rows, err := s.adminIdentityRowsForPlatform(r)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(rows), nil
	case "create_admin_identity":
		args, err := decodeManagementMCPArguments[managementMCPCreateAdminIdentityArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		created, err := s.createManagedAdminIdentity(r, domain.CreateAdminIdentityRequest{
			Actor:       args.Actor,
			DisplayName: args.DisplayName,
			Role:        domain.AdminIdentityRole(args.Role),
			TenantID:    args.TenantID,
			WorkspaceID: args.WorkspaceID,
		})
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(created), nil
	case "rotate_admin_identity_key":
		args, err := decodeManagementMCPArguments[managementMCPAdminIdentityIDArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		rotated, err := s.rotateManagedAdminIdentityKey(r, args.ID)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(rotated), nil
	case "disable_admin_identity":
		args, err := decodeManagementMCPArguments[managementMCPAdminIdentityIDArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		disabled, err := s.disableManagedAdminIdentity(r, args.ID)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(disabled), nil
	case "list_permission_package_templates":
		return managementMCPResult(permissionpack.Templates()), nil
	case "draft_permission_package":
		args, err := decodeManagementMCPArguments[domain.PermissionPackageDraftRequest](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if err := s.requirePermissionPackageDraftScope(r, args); err != nil {
			return managementMCPCallResult{}, err
		}
		draft, err := s.buildPermissionPackageDraft(r.Context(), args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(draft), nil
	case "preflight_permission_package":
		args, err := decodeManagementMCPArguments[domain.PermissionPackageApplyRequest](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if err := s.requirePermissionPackageDraftScope(r, args.PermissionPackageDraftRequest); err != nil {
			return managementMCPCallResult{}, err
		}
		preflight, err := s.preflightPermissionPackageRequest(r.Context(), args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(preflight), nil
	case "apply_permission_package":
		args, err := decodeManagementMCPArguments[domain.PermissionPackageApplyRequest](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		applied, err := s.applyPermissionPackageRequest(r, args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(applied), nil
	case "create_permission_package_approval_request":
		args, err := decodeManagementMCPArguments[domain.PermissionPackageDraftRequest](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if err := s.requirePermissionPackageDraftScope(r, args); err != nil {
			return managementMCPCallResult{}, err
		}
		created, err := s.createPermissionPackageApprovalRequestRecord(r.Context(), args, managementActor(r), s.now())
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "permission_package.approval_requested", "permission_package_approval_request", created.ID, "Permission package approval requested", permissionPackageApprovalAuditMetadata(created))); err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(permissionPackageApprovalRequestResponse(created, s.now())), nil
	case "list_permission_package_approval_requests":
		args, err := decodeManagementMCPArguments[managementMCPPermissionPackageApprovalRequestArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		filter, err := permissionPackageApprovalRequestFilterFromMCPArgs(args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		filter.ManagementScope, err = s.effectiveManagementScopeForRequest(r, filter.ManagementScope)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		rows, err := s.listPermissionPackageApprovalRequestsForRequest(r.Context(), r, filter, args.Reviewer, filter.Limit)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(permissionPackageApprovalRequestResponses(rows, s.now())), nil
	case "approve_permission_package_approval_request":
		args, err := decodeManagementMCPArguments[managementMCPApprovalResolutionArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		approved, err := s.resolveManagementMCPPermissionPackageApprovalRequest(r, args, domain.PermissionPackageApprovalStatusApproved)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(permissionPackageApprovalRequestResponse(approved, s.now())), nil
	case "reject_permission_package_approval_request":
		args, err := decodeManagementMCPArguments[managementMCPApprovalResolutionArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		rejected, err := s.resolveManagementMCPPermissionPackageApprovalRequest(r, args, domain.PermissionPackageApprovalStatusRejected)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(permissionPackageApprovalRequestResponse(rejected, s.now())), nil
	case "withdraw_permission_package_approval_request":
		args, err := decodeManagementMCPArguments[managementMCPApprovalResolutionArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		withdrawn, err := s.withdrawManagementMCPPermissionPackageApprovalRequest(r, args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(permissionPackageApprovalRequestResponse(withdrawn, s.now())), nil
	case "list_permission_package_applications":
		args, err := decodeManagementMCPArguments[managementMCPPermissionPackageApplicationArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		filter, err := permissionPackageApplicationFilterFromMCPArgs(args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		filter.ManagementScope, err = s.effectiveManagementScopeForRequest(r, filter.ManagementScope)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		rows, err := s.repo.ListPermissionPackageApplications(r.Context(), filter)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		rows, err = s.visiblePermissionPackageApplications(r.Context(), rows, filter.ManagementScope)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(rows), nil
	case "check_permission_package_production_readiness":
		args, err := decodeManagementMCPArguments[managementMCPPermissionPackageProductionReadinessArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		query, err := permissionPackageProductionReadinessQueryFromMCPArgs(args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if err := s.requirePermissionPackageQueryScope(r, query); err != nil {
			return managementMCPCallResult{}, err
		}
		readiness, err := s.permissionPackageProductionReadiness(r.Context(), query)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(readiness), nil
	case "export_permission_package_production_evidence":
		args, err := decodeManagementMCPArguments[managementMCPPermissionPackageProductionReadinessArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		query, err := permissionPackageProductionReadinessQueryFromMCPArgs(args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if err := s.requirePermissionPackageQueryScope(r, query); err != nil {
			return managementMCPCallResult{}, err
		}
		report, err := s.permissionPackageProductionEvidenceReport(r.Context(), query)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(report), nil
	case "explain_permission_package_draft":
		args, err := decodeManagementMCPArguments[domain.PermissionPackageDraftRequest](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if err := s.requirePermissionPackageDraftScope(r, args); err != nil {
			return managementMCPCallResult{}, err
		}
		explanation, err := s.explainManagementMCPPermissionPackageDraft(r, args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(explanation), nil
	case "explain_access_decision":
		args, err := decodeManagementMCPArguments[managementMCPExplainAccessArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: strings.TrimSpace(args.TenantID), WorkspaceID: strings.TrimSpace(args.WorkspaceID)}); err != nil {
			return managementMCPCallResult{}, err
		}
		explanation, err := s.explainManagementMCPAccessDecision(r, args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(explanation), nil
	case "get_tenant_access_profile":
		args, err := decodeManagementMCPArguments[managementMCPAccessProfileArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: strings.TrimSpace(args.TenantID), WorkspaceID: strings.TrimSpace(args.WorkspaceID)}); err != nil {
			return managementMCPCallResult{}, err
		}
		profile, err := s.managementMCPAccessProfile(r, args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(profile), nil
	case "list_agents":
		args, err := decodeManagementMCPArguments[managementMCPScopeArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		scope, err := s.effectiveManagementScopeForRequest(r, store.ManagementScope{
			TenantID:    strings.TrimSpace(args.TenantID),
			WorkspaceID: strings.TrimSpace(args.WorkspaceID),
		})
		if err != nil {
			return managementMCPCallResult{}, err
		}
		rows, err := s.repo.ListAgents(r.Context(), store.AgentFilter{ManagementScope: scope})
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(rows), nil
	case "list_capabilities":
		args, err := decodeManagementMCPArguments[managementMCPCapabilityArgs](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		if args.Status != "" && !validCapabilityDiscoveryStatus(args.Status) {
			return managementMCPCallResult{}, domain.BadRequest("VALIDATION_FAILED", "status must be pending_review, approved, deprecated, or removed")
		}
		scope, err := s.effectiveManagementScopeForRequest(r, store.ManagementScope{
			TenantID:    strings.TrimSpace(args.TenantID),
			WorkspaceID: strings.TrimSpace(args.WorkspaceID),
		})
		if err != nil {
			return managementMCPCallResult{}, err
		}
		rows, err := s.repo.ListCapabilities(r.Context(), store.CapabilityFilter{
			ManagementScope: scope,
			TargetID:        strings.TrimSpace(args.TargetID),
			Status:          args.Status,
		})
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(rows), nil
	default:
		return managementMCPCallResult{}, domain.BadRequest("VALIDATION_FAILED", fmt.Sprintf("management MCP tool %q is not supported", req.Params.Name))
	}
}

func managementMCPTools() []managementMCPTool {
	return []managementMCPTool{
		{
			Name:        "list_admin_identities",
			Description: "List bootstrap and managed administrator identities, including roles, status, and tenant/workspace boundaries. Requires platform administrator.",
			InputSchema: objectSchema(map[string]any{}, []string{}),
		},
		{
			Name:        "create_admin_identity",
			Description: "Create a managed administrator identity and return its one-time key. Requires platform administrator.",
			InputSchema: objectSchema(map[string]any{
				"actor":       stringSchema("Unique administrator actor."),
				"displayName": stringSchema("Business-readable administrator name."),
				"role":        map[string]any{"type": "string", "enum": []string{"platform_admin", "tenant_admin", "security_reviewer"}},
				"tenantId":    stringSchema("Required for tenant_admin or security_reviewer."),
				"workspaceId": stringSchema("Optional scoped workspace."),
			}, []string{"actor", "role"}),
		},
		{
			Name:        "rotate_admin_identity_key",
			Description: "Rotate a managed administrator key and return the new one-time key. Requires platform administrator.",
			InputSchema: objectSchema(map[string]any{
				"id": stringSchema("Managed administrator identity id."),
			}, []string{"id"}),
		},
		{
			Name:        "disable_admin_identity",
			Description: "Disable a managed administrator identity. Requires platform administrator.",
			InputSchema: objectSchema(map[string]any{
				"id": stringSchema("Managed administrator identity id."),
			}, []string{"id"}),
		},
		{
			Name:        "list_permission_package_templates",
			Description: "List deterministic permission package templates an admin agent can use for low-friction permission changes.",
			InputSchema: objectSchema(map[string]any{}, []string{}),
		},
		{
			Name:        "draft_permission_package",
			Description: "Build a permission package draft with allowed capabilities, blocked capabilities, data scopes, readiness, and simulation rows.",
			InputSchema: permissionPackageDraftSchema(),
		},
		{
			Name:        "preflight_permission_package",
			Description: "Run a read-only permission package apply preflight with blockers, warnings, planned changes, approval readiness, and existing grant-chain records.",
			InputSchema: permissionPackageApplySchema(),
		},
		{
			Name:        "apply_permission_package",
			Description: "Apply a ready permission package draft by approving allowed capabilities, creating tenant/workspace/caller assignments, and recording audit events. Approval-required drafts need approvalRequestId.",
			InputSchema: permissionPackageApplySchema(),
		},
		{
			Name:        "create_permission_package_approval_request",
			Description: "Create a pending approval request snapshot for a ready permission package draft that policy gate says cannot be applied directly.",
			InputSchema: permissionPackageDraftSchema(),
		},
		{
			Name:        "list_permission_package_approval_requests",
			Description: "List permission package approval requests by scope, template, target, caller, status, and limit.",
			InputSchema: permissionPackageApprovalRequestListSchema(),
		},
		{
			Name:        "approve_permission_package_approval_request",
			Description: "Approve a pending permission package approval request so a matching apply_permission_package call can use its approvalRequestId.",
			InputSchema: approvalResolutionSchema(),
		},
		{
			Name:        "reject_permission_package_approval_request",
			Description: "Reject a pending permission package approval request and record reviewer audit details.",
			InputSchema: approvalResolutionSchema(),
		},
		{
			Name:        "withdraw_permission_package_approval_request",
			Description: "Withdraw a pending permission package approval request as its original requester before review or apply.",
			InputSchema: approvalResolutionSchema(),
		},
		{
			Name:        "list_permission_package_applications",
			Description: "List permission package application records so an admin agent can review template version, scope, created assignments, and data-scope records.",
			InputSchema: permissionPackageApplicationListSchema(),
		},
		{
			Name:        "check_permission_package_production_readiness",
			Description: "Check the read-only production go/no-go gate for a tenant-scoped permission package using preflight, application, health, impact, access-profile, runtime, and audit records.",
			InputSchema: permissionPackageProductionReadinessSchema(),
		},
		{
			Name:        "export_permission_package_production_evidence",
			Description: "Export a read-only JSON acceptance report for a tenant-scoped permission package production readiness decision.",
			InputSchema: permissionPackageProductionReadinessSchema(),
		},
		{
			Name:        "explain_permission_package_draft",
			Description: "Explain whether a permission package draft is ready, which simulation rows are blocked, and what an admin agent should fix next.",
			InputSchema: permissionPackageDraftSchema(),
		},
		{
			Name:        "explain_access_decision",
			Description: "Explain the current capability access decision for a tenant, workspace, caller, target, and capability without changing permissions.",
			InputSchema: explainAccessDecisionSchema(),
		},
		{
			Name:        "get_tenant_access_profile",
			Description: "Get tenant access-profile records after permission changes, including effective grants, assignments, data scopes, and recent traces.",
			InputSchema: objectSchema(map[string]any{
				"tenantId":         stringSchema("Tenant to inspect."),
				"workspaceId":      stringSchema("Optional workspace filter."),
				"targetId":         stringSchema("Optional target agent filter."),
				"capabilityId":     stringSchema("Optional capability filter."),
				"callerInstanceId": stringSchema("Optional caller instance filter."),
				"traceLimit":       map[string]any{"type": "integer", "minimum": 0, "maximum": maxAccessProfileTraceLimit, "description": "Recent trace count to include."},
			}, []string{"tenantId"}),
		},
		{
			Name:        "list_agents",
			Description: "List registered caller agents and targets, optionally scoped by tenant and workspace.",
			InputSchema: scopedListSchema(),
		},
		{
			Name:        "list_capabilities",
			Description: "List discovered MCP capabilities, optionally scoped by tenant/workspace, target, or discovery status.",
			InputSchema: objectSchema(map[string]any{
				"tenantId":    stringSchema("Optional tenant scope."),
				"workspaceId": stringSchema("Optional workspace scope."),
				"targetId":    stringSchema("Optional target agent id."),
				"status":      map[string]any{"type": "string", "enum": []string{"pending_review", "approved", "deprecated", "removed"}, "description": "Optional discovery status filter."},
			}, []string{}),
		},
	}
}

func permissionPackageApplicationFilterFromMCPArgs(args managementMCPPermissionPackageApplicationArgs) (store.PermissionPackageApplicationFilter, error) {
	limit := defaultAuditLimit
	if args.Limit != nil {
		if *args.Limit < 1 || *args.Limit > maxAuditLimit {
			return store.PermissionPackageApplicationFilter{}, domain.BadRequest("VALIDATION_FAILED", "limit must be between 1 and 500")
		}
		limit = *args.Limit
	}
	return store.PermissionPackageApplicationFilter{
		ManagementScope: store.ManagementScope{
			TenantID:    strings.TrimSpace(args.TenantID),
			WorkspaceID: strings.TrimSpace(args.WorkspaceID),
		},
		TemplateID:       strings.TrimSpace(args.TemplateID),
		TargetID:         strings.TrimSpace(args.TargetID),
		CallerInstanceID: strings.TrimSpace(args.CallerInstanceID),
		Limit:            limit,
	}, nil
}

func permissionPackageProductionReadinessQueryFromMCPArgs(args managementMCPPermissionPackageProductionReadinessArgs) (permissionPackageProductionReadinessQuery, error) {
	query := permissionPackageProductionReadinessQuery{
		TenantID:          strings.TrimSpace(args.TenantID),
		WorkspaceID:       strings.TrimSpace(args.WorkspaceID),
		TemplateID:        strings.TrimSpace(args.TemplateID),
		TargetID:          strings.TrimSpace(args.TargetID),
		CallerInstanceID:  strings.TrimSpace(args.CallerInstanceID),
		SubjectID:         strings.TrimSpace(args.SubjectID),
		Region:            strings.TrimSpace(args.Region),
		RequestText:       strings.TrimSpace(args.RequestText),
		SubjectSelector:   strings.TrimSpace(args.SubjectSelector),
		ApprovalRequestID: strings.TrimSpace(args.ApprovalRequestID),
		TraceLimit:        defaultAccessProfileTraceLimit,
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "tenantId", value: query.TenantID},
		{name: "workspaceId", value: query.WorkspaceID},
		{name: "templateId", value: query.TemplateID},
		{name: "targetId", value: query.TargetID},
		{name: "callerInstanceId", value: query.CallerInstanceID},
	} {
		if required.value == "" {
			return permissionPackageProductionReadinessQuery{}, domain.BadRequest("VALIDATION_FAILED", required.name+" is required")
		}
	}
	if args.TraceLimit != nil {
		if *args.TraceLimit < 0 || *args.TraceLimit > maxAccessProfileTraceLimit {
			return permissionPackageProductionReadinessQuery{}, domain.BadRequest("VALIDATION_FAILED", "traceLimit must be between 0 and 100")
		}
		query.TraceLimit = *args.TraceLimit
	}
	return query, nil
}

func permissionPackageApprovalRequestFilterFromMCPArgs(args managementMCPPermissionPackageApprovalRequestArgs) (store.PermissionPackageApprovalRequestFilter, error) {
	limit := defaultAuditLimit
	if args.Limit != nil {
		if *args.Limit < 1 || *args.Limit > maxAuditLimit {
			return store.PermissionPackageApprovalRequestFilter{}, domain.BadRequest("VALIDATION_FAILED", "limit must be between 1 and 500")
		}
		limit = *args.Limit
	}
	if args.Status != "" && !validPermissionPackageApprovalStatus(args.Status) {
		return store.PermissionPackageApprovalRequestFilter{}, domain.BadRequest("VALIDATION_FAILED", "status must be pending, approved, rejected, or withdrawn")
	}
	return store.PermissionPackageApprovalRequestFilter{
		ManagementScope: store.ManagementScope{
			TenantID:    strings.TrimSpace(args.TenantID),
			WorkspaceID: strings.TrimSpace(args.WorkspaceID),
		},
		TemplateID:       strings.TrimSpace(args.TemplateID),
		TargetID:         strings.TrimSpace(args.TargetID),
		CallerInstanceID: strings.TrimSpace(args.CallerInstanceID),
		Status:           args.Status,
		Limit:            limit,
	}, nil
}

func (s *Server) resolveManagementMCPPermissionPackageApprovalRequest(r *http.Request, args managementMCPApprovalResolutionArgs, status domain.PermissionPackageApprovalStatus) (domain.PermissionPackageApprovalRequest, error) {
	id := strings.TrimSpace(args.ID)
	if id == "" {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "id is required")
	}
	existing, ok, err := s.repo.GetPermissionPackageApprovalRequest(r.Context(), id)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if !ok {
		return domain.PermissionPackageApprovalRequest{}, domain.NotFound("approval request not found")
	}
	if err := s.requirePermissionPackageApprovalRequestScope(r, existing); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	reviewer, err := reviewerFromRequest(args.Reviewer, r)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if err := s.validatePermissionPackageApprovalReviewer(r.Context(), reviewer, existing); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	saved, err := s.resolvePermissionPackageApprovalRequestRecord(r.Context(), existing, status, reviewer, args.Comment, s.now())
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	action := "permission_package.approval_approved"
	summary := "Permission package approval approved"
	if status == domain.PermissionPackageApprovalStatusRejected {
		action = "permission_package.approval_rejected"
		summary = "Permission package approval rejected"
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, saved.TenantID, saved.WorkspaceID, action, "permission_package_approval_request", saved.ID, summary, permissionPackageApprovalAuditMetadata(saved))); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	return saved, nil
}

func (s *Server) withdrawManagementMCPPermissionPackageApprovalRequest(r *http.Request, args managementMCPApprovalResolutionArgs) (domain.PermissionPackageApprovalRequest, error) {
	id := strings.TrimSpace(args.ID)
	if id == "" {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "id is required")
	}
	existing, ok, err := s.repo.GetPermissionPackageApprovalRequest(r.Context(), id)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if !ok {
		return domain.PermissionPackageApprovalRequest{}, domain.NotFound("approval request not found")
	}
	if err := s.requirePermissionPackageApprovalRequestScope(r, existing); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	saved, err := s.withdrawPermissionPackageApprovalRequestRecord(r.Context(), existing, managementActor(r), args.Comment, s.now())
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, saved.TenantID, saved.WorkspaceID, "permission_package.approval_withdrawn", "permission_package_approval_request", saved.ID, "Permission package approval withdrawn", permissionPackageApprovalAuditMetadata(saved))); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	return saved, nil
}

func (s *Server) managementMCPAccessProfile(r *http.Request, args managementMCPAccessProfileArgs) (tenantAccessProfileResponse, error) {
	traceLimit := defaultAccessProfileTraceLimit
	if args.TraceLimit != nil {
		if *args.TraceLimit < 0 || *args.TraceLimit > maxAccessProfileTraceLimit {
			return tenantAccessProfileResponse{}, domain.BadRequest("VALIDATION_FAILED", "traceLimit must be between 0 and 100")
		}
		traceLimit = *args.TraceLimit
	}
	return s.buildTenantAccessProfileForRequest(r, strings.TrimSpace(args.TenantID), accessProfileQuery{
		WorkspaceID:      strings.TrimSpace(args.WorkspaceID),
		TargetID:         strings.TrimSpace(args.TargetID),
		CapabilityID:     strings.TrimSpace(args.CapabilityID),
		CallerInstanceID: strings.TrimSpace(args.CallerInstanceID),
		TraceLimit:       traceLimit,
	})
}

func (s *Server) explainAccessDecision(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	result, err := s.explainManagementMCPAccessDecision(r, managementMCPExplainAccessArgs{
		TenantID:         query.Get("tenantId"),
		WorkspaceID:      query.Get("workspaceId"),
		CallerInstanceID: query.Get("callerInstanceId"),
		SubjectID:        query.Get("subjectId"),
		TargetID:         query.Get("targetId"),
		CapabilityID:     query.Get("capabilityId"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) explainManagementMCPPermissionPackageDraft(r *http.Request, args domain.PermissionPackageDraftRequest) (managementMCPExplainPermissionPackageResult, error) {
	draft, err := s.buildPermissionPackageDraft(r.Context(), args)
	if err != nil {
		return managementMCPExplainPermissionPackageResult{}, err
	}
	outcome := "blocked"
	if draft.Readiness.CanApply {
		outcome = "ready"
		if !draft.PolicyGate.CanApplyDirectly {
			outcome = "approval_required"
		}
	}
	blockedRows := managementMCPBlockedSimulationRows(draft.SimulationRows)
	result := managementMCPExplainPermissionPackageResult{
		Outcome:                outcome,
		Summary:                managementMCPPermissionPackageSummary(draft),
		DraftID:                draft.ID,
		Input:                  draft.Input,
		Readiness:              draft.Readiness,
		PolicyGate:             draft.PolicyGate,
		AllowedCapabilityCount: len(draft.AllowedCapabilities),
		BlockedCapabilityCount: len(draft.BlockedCapabilities),
		BlockedSimulationRows:  blockedRows,
		DataScopes:             draft.DataScopes,
		NextActionCodes:        managementMCPPermissionPackageNextActionCodes(draft, blockedRows),
		NextActions:            managementMCPPermissionPackageNextActions(draft, blockedRows),
	}
	return result, nil
}

func managementMCPBlockedSimulationRows(rows []domain.PermissionPackageSimulationRow) []domain.PermissionPackageSimulationRow {
	blocked := make([]domain.PermissionPackageSimulationRow, 0, len(rows))
	for _, row := range rows {
		if row.ExpectedDecision == domain.PermissionPackageDecisionDeny {
			blocked = append(blocked, row)
		}
	}
	return blocked
}

func managementMCPPermissionPackageSummary(draft domain.PermissionPackageDraft) string {
	if draft.Readiness.CanApply {
		if !draft.PolicyGate.CanApplyDirectly {
			return fmt.Sprintf("Permission package %s requires approval before apply with %d policy gate reasons.", draft.Template.ID, len(draft.PolicyGate.Reasons))
		}
		return fmt.Sprintf("Permission package %s is ready to apply with %d allowed capabilities and %d blocked capabilities.", draft.Template.ID, len(draft.AllowedCapabilities), len(draft.BlockedCapabilities))
	}
	if len(draft.Readiness.MissingFields) > 0 {
		return fmt.Sprintf("Permission package %s is blocked because required fields are missing: %s.", draft.Template.ID, strings.Join(draft.Readiness.MissingFields, ", "))
	}
	if len(draft.Readiness.Warnings) > 0 {
		return fmt.Sprintf("Permission package %s is blocked by readiness warnings: %s", draft.Template.ID, strings.Join(draft.Readiness.Warnings, "; "))
	}
	return fmt.Sprintf("Permission package %s is blocked and needs review before applying.", draft.Template.ID)
}

func managementMCPPermissionPackageNextActions(draft domain.PermissionPackageDraft, blockedRows []domain.PermissionPackageSimulationRow) []string {
	actions := []string{}
	if len(draft.Readiness.MissingFields) > 0 {
		actions = append(actions, "Provide required fields: "+strings.Join(draft.Readiness.MissingFields, ", ")+".")
	}
	for _, warning := range draft.Readiness.Warnings {
		lower := strings.ToLower(warning)
		switch {
		case strings.Contains(lower, "no matching allowed capabilities"):
			actions = append(actions, "Refresh target capabilities or choose a template whose allowed actions match this target.")
		case strings.Contains(lower, "data scopes exceed"):
			actions = append(actions, "Narrow the requested region or data scope to fit the capability boundary, or pick a capability with the required boundary.")
		default:
			actions = append(actions, "Resolve readiness warning: "+warning)
		}
	}
	if draft.Readiness.CanApply && !draft.PolicyGate.CanApplyDirectly {
		if len(draft.PolicyGate.NextActions) > 0 {
			actions = append(actions, draft.PolicyGate.NextActions...)
		} else {
			actions = append(actions, "Request approval before applying this permission request.")
		}
	}
	if len(blockedRows) > 0 {
		actions = append(actions, "Keep denied simulation rows blocked unless a separate approval policy explicitly allows them.")
	}
	if len(actions) == 0 {
		actions = append(actions, "Apply the permission package after the administrator confirms the preview.")
	}
	return dedupeStrings(actions)
}

func managementMCPPermissionPackageNextActionCodes(draft domain.PermissionPackageDraft, blockedRows []domain.PermissionPackageSimulationRow) []string {
	codes := []string{}
	if len(draft.Readiness.MissingFields) > 0 {
		codes = append(codes, "fix_draft_readiness")
	}
	for _, warning := range draft.Readiness.Warnings {
		lower := strings.ToLower(warning)
		switch {
		case strings.Contains(lower, "no matching allowed capabilities"):
			codes = append(codes, "refresh_capabilities")
		case strings.Contains(lower, "data scopes exceed"):
			codes = append(codes, "narrow_data_scope")
		default:
			codes = append(codes, "fix_draft_readiness")
		}
	}
	if draft.Readiness.CanApply && !draft.PolicyGate.CanApplyDirectly {
		if len(draft.PolicyGate.NextActionCodes) > 0 {
			for _, code := range draft.PolicyGate.NextActionCodes {
				codes = append(codes, string(code))
			}
		} else {
			codes = append(codes, "create_approval_request")
		}
	}
	if len(blockedRows) > 0 {
		codes = append(codes, "review_blocked_simulation_rows")
	}
	if len(codes) == 0 {
		codes = append(codes, "apply_permission_package")
	}
	return dedupeStrings(codes)
}

func (s *Server) explainManagementMCPAccessDecision(r *http.Request, args managementMCPExplainAccessArgs) (managementMCPExplainAccessResult, error) {
	args = trimManagementMCPExplainAccessArgs(args)
	if err := validateManagementMCPExplainAccessArgs(args); err != nil {
		return managementMCPExplainAccessResult{}, err
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: args.TenantID, WorkspaceID: args.WorkspaceID}); err != nil {
		return managementMCPExplainAccessResult{}, err
	}
	if err := s.requireAccessDecisionResourceScope(r, args); err != nil {
		return managementMCPExplainAccessResult{}, err
	}
	decision, err := s.repo.EvaluateCapabilityAccess(r.Context(), store.CapabilityAccessRequest{
		TenantID:         args.TenantID,
		WorkspaceID:      args.WorkspaceID,
		CallerInstanceID: args.CallerInstanceID,
		SubjectID:        args.SubjectID,
		TargetID:         args.TargetID,
		CapabilityID:     args.CapabilityID,
		Now:              s.now(),
	})
	if err != nil {
		return managementMCPExplainAccessResult{}, err
	}
	evidence, err := s.managementMCPAccessEvidence(r.Context(), args, decision)
	if err != nil {
		return managementMCPExplainAccessResult{}, err
	}
	outcome := "denied"
	if decision.Allowed {
		outcome = "allowed"
	}
	return managementMCPExplainAccessResult{
		Outcome:         outcome,
		Summary:         managementMCPAccessSummary(decision),
		Request:         args,
		Decision:        decision,
		Evidence:        evidence,
		DataScopes:      decision.DataScopes,
		NextActionCodes: managementMCPAccessNextActionCodes(decision),
		NextActions:     managementMCPAccessNextActions(decision),
	}, nil
}

func (s *Server) requireAccessDecisionResourceScope(r *http.Request, args managementMCPExplainAccessArgs) error {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return nil
	}
	principal = normalizeAdminPrincipal(principal)
	if principal.Role == adminRolePlatformAdmin || principal.TenantID == "" {
		return nil
	}

	caller, ok, err := s.repo.GetAgent(r.Context(), args.CallerInstanceID)
	if err != nil {
		return err
	}
	if ok {
		if !agentInRequestedScope(caller, args.TenantID, args.WorkspaceID) {
			return domain.PermissionDenied("access decision resource is outside requested scope")
		}
		if err := s.requireAgentManagementScope(r, caller); err != nil {
			return err
		}
	}

	target, ok, err := s.repo.GetAgent(r.Context(), args.TargetID)
	if err != nil {
		return err
	}
	if ok {
		if !agentInRequestedScope(target, args.TenantID, args.WorkspaceID) {
			return domain.PermissionDenied("access decision resource is outside requested scope")
		}
		if err := s.requireAgentManagementScope(r, target); err != nil {
			return err
		}
	}

	capability, ok, err := s.repo.GetCapability(r.Context(), args.CapabilityID)
	if err != nil {
		return err
	}
	if ok {
		if capability.TargetID != args.TargetID {
			return domain.PermissionDenied("access decision resource is outside requested scope")
		}
		if err := s.requireCapabilityManagementScope(r, capability); err != nil {
			return err
		}
	}
	return nil
}

func agentInRequestedScope(agent domain.Agent, tenantID string, workspaceID string) bool {
	return agent.TenantID == strings.TrimSpace(tenantID) && agent.WorkspaceID == strings.TrimSpace(workspaceID)
}

func trimManagementMCPExplainAccessArgs(args managementMCPExplainAccessArgs) managementMCPExplainAccessArgs {
	args.TenantID = strings.TrimSpace(args.TenantID)
	args.WorkspaceID = strings.TrimSpace(args.WorkspaceID)
	args.CallerInstanceID = strings.TrimSpace(args.CallerInstanceID)
	args.SubjectID = strings.TrimSpace(args.SubjectID)
	args.TargetID = strings.TrimSpace(args.TargetID)
	args.CapabilityID = strings.TrimSpace(args.CapabilityID)
	return args
}

func validateManagementMCPExplainAccessArgs(args managementMCPExplainAccessArgs) error {
	missing := []string{}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "tenantId", value: args.TenantID},
		{name: "workspaceId", value: args.WorkspaceID},
		{name: "callerInstanceId", value: args.CallerInstanceID},
		{name: "targetId", value: args.TargetID},
		{name: "capabilityId", value: args.CapabilityID},
	} {
		if field.value == "" {
			missing = append(missing, field.name)
		}
	}
	if len(missing) > 0 {
		return domain.BadRequest("VALIDATION_FAILED", "missing required tool arguments: "+strings.Join(missing, ", "))
	}
	return nil
}

func (s *Server) managementMCPAccessEvidence(ctx context.Context, args managementMCPExplainAccessArgs, decision domain.CapabilityAccessDecision) ([]managementMCPExplainEvidence, error) {
	evidence := []managementMCPExplainEvidence{}
	caller, ok, err := s.repo.GetAgent(ctx, args.CallerInstanceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "caller_instance", Status: "missing", Message: "Caller instance was not found."})
	} else if caller.TenantID != args.TenantID || caller.WorkspaceID != args.WorkspaceID {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "caller_instance", Status: "mismatch", ID: caller.ID, Message: "Caller instance tenant or workspace does not match the requested scope."})
	} else {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "caller_instance", Status: "matched", ID: caller.ID, Message: fmt.Sprintf("Caller instance %s matches tenant %s and workspace %s.", caller.Name, args.TenantID, args.WorkspaceID)})
	}

	target, ok, err := s.repo.GetAgent(ctx, args.TargetID)
	if err != nil {
		return nil, err
	}
	if !ok {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "target", Status: "missing", Message: "Target agent was not found."})
	} else if target.Status != domain.AgentStatusActive {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "target", Status: "inactive", ID: target.ID, Message: "Target agent is not active."})
	} else {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "target", Status: "matched", ID: target.ID, Message: fmt.Sprintf("Target %s is active.", target.Name)})
	}

	capability, ok, err := s.repo.GetCapability(ctx, args.CapabilityID)
	if err != nil {
		return nil, err
	}
	if !ok {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "capability", Status: "missing", Message: "Capability was not found."})
	} else if capability.TargetID != args.TargetID {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "capability", Status: "mismatch", ID: capability.ID, Message: "Capability is registered on a different target."})
	} else if capability.DiscoveryStatus != domain.CapabilityDiscoveryApproved {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "capability", Status: "not_approved", ID: capability.ID, Message: fmt.Sprintf("Capability %s is not approved.", capability.Key)})
	} else {
		evidence = append(evidence, managementMCPExplainEvidence{Layer: "capability", Status: "matched", ID: capability.ID, Message: fmt.Sprintf("Capability %s is approved for the target.", capability.Key)})
	}

	evidence = appendDecisionEvidence(evidence, "tenant_entitlement", decision.EntitlementID, decision.Source, decision.Allowed, "Tenant entitlement matched.", "Tenant entitlement is missing or blocking this capability.")
	evidence = appendDecisionEvidence(evidence, "workspace_assignment", decision.WorkspaceAssignmentID, decision.Source, decision.Allowed, "Workspace assignment matched.", "Workspace assignment is missing or blocking this capability.")
	evidence = appendDecisionEvidence(evidence, "instance_assignment", decision.InstanceAssignmentID, decision.Source, decision.Allowed, "Caller instance assignment matched.", "Caller instance assignment is missing or blocking this capability.")
	return evidence, nil
}

func appendDecisionEvidence(rows []managementMCPExplainEvidence, layer string, id string, source string, allowed bool, matchedMessage string, blockedMessage string) []managementMCPExplainEvidence {
	if id != "" {
		status := "matched"
		message := matchedMessage
		if !allowed && source == layer {
			status = "blocking"
			message = blockedMessage
		}
		return append(rows, managementMCPExplainEvidence{Layer: layer, Status: status, ID: id, Message: message})
	}
	if source == layer {
		return append(rows, managementMCPExplainEvidence{Layer: layer, Status: "missing", Message: blockedMessage})
	}
	return rows
}

func managementMCPAccessSummary(decision domain.CapabilityAccessDecision) string {
	reason := strings.TrimSpace(decision.Reason)
	if reason == "" {
		reason = "no detailed reason was returned"
	}
	if decision.Allowed {
		return "Allowed: " + reason + "."
	}
	return "Denied: " + reason + "."
}

func managementMCPAccessNextActions(decision domain.CapabilityAccessDecision) []string {
	if decision.Allowed {
		return []string{"No permission change is required. Review the returned dataScopes before broadening access."}
	}
	reason := strings.ToLower(decision.Reason)
	switch {
	case strings.Contains(reason, "not registered"):
		return []string{"Refresh the target MCP capabilities, then choose a registered capability from list_capabilities."}
	case strings.Contains(reason, "not approved"):
		return []string{"Approve the capability or apply a permission package that approves the selected capability."}
	case strings.Contains(reason, "tenant has no entitlement"):
		return []string{"Use the permission package flow with draft_permission_package and apply_permission_package to create the tenant entitlement, workspace assignment, and caller assignment together."}
	case strings.Contains(reason, "workspace has no assignment"):
		return []string{"Apply a permission package or create a workspace assignment for this tenant entitlement and workspace."}
	case strings.Contains(reason, "caller instance has no assignment"):
		return []string{"Apply a permission package or create an instance assignment for this caller instance."}
	case strings.Contains(reason, "data scopes exceed"):
		return []string{"Narrow the child dataScopes so they stay inside the parent capability, tenant, workspace, or instance boundary."}
	case strings.Contains(reason, "denies"):
		return []string{"Review the deny effect on the matching entitlement or assignment before granting broader access."}
	default:
		return []string{"Inspect get_tenant_access_profile for this tenant/workspace/caller/capability and repair the first missing decision layer."}
	}
}

func managementMCPAccessNextActionCodes(decision domain.CapabilityAccessDecision) []string {
	if decision.Allowed {
		return []string{"no_change_required"}
	}
	reason := strings.ToLower(decision.Reason)
	switch {
	case strings.Contains(reason, "not registered"):
		return []string{"refresh_capabilities"}
	case strings.Contains(reason, "not approved"):
		return []string{"approve_capability"}
	case strings.Contains(reason, "tenant has no entitlement"):
		return []string{"use_permission_package"}
	case strings.Contains(reason, "workspace has no assignment"):
		return []string{"create_workspace_assignment"}
	case strings.Contains(reason, "caller instance has no assignment"):
		return []string{"create_caller_assignment"}
	case strings.Contains(reason, "data scopes exceed"):
		return []string{"narrow_data_scope"}
	case strings.Contains(reason, "denies"):
		return []string{"review_deny"}
	default:
		return []string{"inspect_access_profile"}
	}
}

func decodeManagementMCPArguments[T any](raw json.RawMessage) (T, error) {
	var out T
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		raw = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return out, domain.BadRequest("VALIDATION_FAILED", "tool arguments are invalid: "+err.Error())
	}
	return out, nil
}

func managementMCPResult(data any) managementMCPCallResult {
	text := managementMCPText(data)
	return managementMCPCallResult{
		Content: []managementMCPContentItem{{
			Type: "text",
			Text: text,
		}},
		StructuredContent: data,
	}
}

func managementMCPText(data any) string {
	raw, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	const maxTextBytes = 12000
	if len(raw) > maxTextBytes {
		truncated := raw[:maxTextBytes]
		for !utf8.Valid(truncated) {
			truncated = truncated[:len(truncated)-1]
		}
		return string(truncated) + "... truncated"
	}
	return string(raw)
}

func writeManagementMCPResult(w http.ResponseWriter, id json.RawMessage, result any) {
	writeManagementMCPResponse(w, managementMCPResponse{
		JSONRPC: "2.0",
		ID:      managementMCPResponseID(id),
		Result:  result,
	})
}

func writeManagementMCPAppError(w http.ResponseWriter, id json.RawMessage, err error) {
	var appErr domain.AppError
	if errors.As(err, &appErr) {
		writeManagementMCPErrorWithData(w, id, -32602, appErr.Message, &managementMCPErrorData{AppCode: appErr.Code, HTTPStatus: appErr.Status})
		return
	}
	writeManagementMCPError(w, id, -32603, "internal server error")
}

func writeManagementMCPError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	writeManagementMCPErrorWithData(w, id, code, message, nil)
}

func writeManagementMCPErrorWithData(w http.ResponseWriter, id json.RawMessage, code int, message string, data *managementMCPErrorData) {
	writeManagementMCPResponse(w, managementMCPResponse{
		JSONRPC: "2.0",
		ID:      managementMCPResponseID(id),
		Error: &managementMCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

func writeManagementMCPResponse(w http.ResponseWriter, response managementMCPResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func managementMCPResponseID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}

func permissionPackageDraftSchema() map[string]any {
	return objectSchema(permissionPackageDraftProperties(), permissionPackageDraftRequiredFields())
}

func permissionPackageApplySchema() map[string]any {
	properties := permissionPackageDraftProperties()
	properties["approvalRequestId"] = stringSchema("Approved permission package approval request id required when policyGate.canApplyDirectly is false.")
	return objectSchema(properties, permissionPackageDraftRequiredFields())
}

func permissionPackageDraftProperties() map[string]any {
	return map[string]any{
		"callerInstanceId": stringSchema("Caller agent instance that will receive the package."),
		"region":           stringSchema("Region data-scope value, for example us-east."),
		"requestText":      stringSchema("Administrator natural-language request for audit context."),
		"subjectSelector":  stringSchema("Required subject selector for the access object, for example user:sales-*. Empty and bare * are rejected."),
		"targetId":         stringSchema("Target MCP agent id."),
		"templateId":       stringSchema("Permission package template id, for example sales-readonly."),
		"tenantId":         stringSchema("Tenant that receives the entitlement."),
		"workspaceId":      stringSchema("Workspace that receives the assignment."),
	}
}

func permissionPackageDraftRequiredFields() []string {
	return []string{"callerInstanceId", "subjectSelector", "targetId", "templateId", "tenantId", "workspaceId"}
}

func permissionPackageApplicationListSchema() map[string]any {
	return objectSchema(map[string]any{
		"tenantId":         stringSchema("Optional tenant scope. Tenant subtree is included when the tenant is registered."),
		"workspaceId":      stringSchema("Optional workspace scope."),
		"templateId":       stringSchema("Optional permission package template id."),
		"targetId":         stringSchema("Optional target MCP agent id."),
		"callerInstanceId": stringSchema("Optional caller agent instance id."),
		"limit":            map[string]any{"type": "integer", "minimum": 1, "maximum": maxAuditLimit, "description": "Maximum application records to return."},
	}, []string{})
}

func permissionPackageProductionReadinessSchema() map[string]any {
	return objectSchema(map[string]any{
		"tenantId":          stringSchema("Tenant that receives the entitlement."),
		"workspaceId":       stringSchema("Workspace that receives the assignment."),
		"templateId":        stringSchema("Permission package template id, for example sales-readonly."),
		"targetId":          stringSchema("Target MCP agent id."),
		"callerInstanceId":  stringSchema("Caller agent instance that will receive the package."),
		"subjectId":         stringSchema("Optional production subject id used to filter runtime records."),
		"region":            stringSchema("Optional region data-scope value. Defaults from the latest application when available."),
		"requestText":       stringSchema("Optional administrator request text. Defaults from the latest application when available."),
		"subjectSelector":   stringSchema("Optional subject selector. Defaults from the latest application when available."),
		"approvalRequestId": stringSchema("Optional approved request id for pre-apply readiness checks."),
		"traceLimit":        map[string]any{"type": "integer", "minimum": 0, "maximum": maxAccessProfileTraceLimit, "description": "Recent trace count to include."},
	}, []string{"tenantId", "workspaceId", "templateId", "targetId", "callerInstanceId"})
}

func permissionPackageApprovalRequestListSchema() map[string]any {
	return objectSchema(map[string]any{
		"tenantId":         stringSchema("Optional tenant scope. Tenant subtree is included when the tenant is registered."),
		"workspaceId":      stringSchema("Optional workspace scope."),
		"templateId":       stringSchema("Optional permission package template id."),
		"targetId":         stringSchema("Optional target MCP agent id."),
		"callerInstanceId": stringSchema("Optional caller agent instance id."),
		"status":           map[string]any{"type": "string", "enum": []string{"pending", "approved", "rejected", "withdrawn"}, "description": "Optional approval request status filter."},
		"reviewer":         stringSchema("Optional reviewer identity for reviewable approval queue filtering."),
		"limit":            map[string]any{"type": "integer", "minimum": 1, "maximum": maxAuditLimit, "description": "Maximum approval requests to return."},
	}, []string{})
}

func approvalResolutionSchema() map[string]any {
	return objectSchema(map[string]any{
		"id":       stringSchema("Permission package approval request id."),
		"reviewer": stringSchema("Optional reviewer identity. Defaults to the management actor."),
		"comment":  stringSchema("Optional review or withdraw comment for audit context."),
	}, []string{"id"})
}

func explainAccessDecisionSchema() map[string]any {
	return objectSchema(map[string]any{
		"tenantId":         stringSchema("Tenant scope to evaluate."),
		"workspaceId":      stringSchema("Workspace scope to evaluate."),
		"callerInstanceId": stringSchema("Caller agent instance requesting access."),
		"subjectId":        stringSchema("Optional subject id for subject-specific assignments."),
		"targetId":         stringSchema("Target agent id."),
		"capabilityId":     stringSchema("Capability id to evaluate."),
	}, []string{"tenantId", "workspaceId", "callerInstanceId", "targetId", "capabilityId"})
}

func scopedListSchema() map[string]any {
	return objectSchema(map[string]any{
		"tenantId":    stringSchema("Optional tenant scope."),
		"workspaceId": stringSchema("Optional workspace scope."),
	}, []string{})
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func stringSchema(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
