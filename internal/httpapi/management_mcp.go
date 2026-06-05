package httpapi

import (
	"bytes"
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
	Code    int    `json:"code"`
	Message string `json:"message"`
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

type managementMCPCapabilityArgs struct {
	TenantID    string                           `json:"tenantId"`
	WorkspaceID string                           `json:"workspaceId"`
	TargetID    string                           `json:"targetId"`
	Status      domain.CapabilityDiscoveryStatus `json:"status"`
}

type managementMCPAccessProfileArgs struct {
	TenantID         string `json:"tenantId"`
	WorkspaceID      string `json:"workspaceId"`
	TargetID         string `json:"targetId"`
	CapabilityID     string `json:"capabilityId"`
	CallerInstanceID string `json:"callerInstanceId"`
	TraceLimit       *int   `json:"traceLimit"`
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
	case "list_permission_package_templates":
		return managementMCPResult(permissionpack.Templates()), nil
	case "draft_permission_package":
		args, err := decodeManagementMCPArguments[domain.PermissionPackageDraftRequest](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		draft, err := s.buildPermissionPackageDraft(r.Context(), args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(draft), nil
	case "apply_permission_package":
		args, err := decodeManagementMCPArguments[domain.PermissionPackageDraftRequest](req.Params.Arguments)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		applied, err := s.applyPermissionPackageRequest(r, args)
		if err != nil {
			return managementMCPCallResult{}, err
		}
		return managementMCPResult(applied), nil
	case "get_tenant_access_profile":
		args, err := decodeManagementMCPArguments[managementMCPAccessProfileArgs](req.Params.Arguments)
		if err != nil {
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
		rows, err := s.repo.ListAgents(r.Context(), store.AgentFilter{ManagementScope: store.ManagementScope{
			TenantID:    strings.TrimSpace(args.TenantID),
			WorkspaceID: strings.TrimSpace(args.WorkspaceID),
		}})
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
		rows, err := s.repo.ListCapabilities(r.Context(), store.CapabilityFilter{
			ManagementScope: store.ManagementScope{
				TenantID:    strings.TrimSpace(args.TenantID),
				WorkspaceID: strings.TrimSpace(args.WorkspaceID),
			},
			TargetID: strings.TrimSpace(args.TargetID),
			Status:   args.Status,
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
			Name:        "apply_permission_package",
			Description: "Apply a ready permission package draft by approving allowed capabilities, creating tenant/workspace/caller assignments, and recording audit evidence.",
			InputSchema: permissionPackageDraftSchema(),
		},
		{
			Name:        "get_tenant_access_profile",
			Description: "Get tenant access-profile evidence after permission changes, including effective grants, assignments, data scopes, and recent traces.",
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

func (s *Server) managementMCPAccessProfile(r *http.Request, args managementMCPAccessProfileArgs) (tenantAccessProfileResponse, error) {
	traceLimit := defaultAccessProfileTraceLimit
	if args.TraceLimit != nil {
		if *args.TraceLimit < 0 || *args.TraceLimit > maxAccessProfileTraceLimit {
			return tenantAccessProfileResponse{}, domain.BadRequest("VALIDATION_FAILED", "traceLimit must be between 0 and 100")
		}
		traceLimit = *args.TraceLimit
	}
	return s.buildTenantAccessProfile(r.Context(), strings.TrimSpace(args.TenantID), accessProfileQuery{
		WorkspaceID:      strings.TrimSpace(args.WorkspaceID),
		TargetID:         strings.TrimSpace(args.TargetID),
		CapabilityID:     strings.TrimSpace(args.CapabilityID),
		CallerInstanceID: strings.TrimSpace(args.CallerInstanceID),
		TraceLimit:       traceLimit,
	})
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
		writeManagementMCPError(w, id, -32602, appErr.Message)
		return
	}
	writeManagementMCPError(w, id, -32603, "internal server error")
}

func writeManagementMCPError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	writeManagementMCPResponse(w, managementMCPResponse{
		JSONRPC: "2.0",
		ID:      managementMCPResponseID(id),
		Error: &managementMCPError{
			Code:    code,
			Message: message,
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
	return objectSchema(map[string]any{
		"callerInstanceId": stringSchema("Caller agent instance that will receive the package."),
		"region":           stringSchema("Region data-scope value, for example us-east."),
		"requestText":      stringSchema("Administrator natural-language request for audit context."),
		"subjectSelector":  stringSchema("Optional subject selector, for example user:sales-*."),
		"targetId":         stringSchema("Target MCP agent id."),
		"templateId":       stringSchema("Permission package template id, for example sales-readonly."),
		"tenantId":         stringSchema("Tenant that receives the entitlement."),
		"workspaceId":      stringSchema("Workspace that receives the assignment."),
	}, []string{"callerInstanceId", "targetId", "templateId", "tenantId", "workspaceId"})
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
