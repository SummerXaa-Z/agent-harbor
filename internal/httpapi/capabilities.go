package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/permissionpack"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (s *Server) refreshTargetCapabilities(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "targetId"))
	target, ok, err := s.repo.GetAgent(r.Context(), targetID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, target); err != nil {
		writeError(w, err)
		return
	}
	if target.ChannelType != "mcp" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "capability refresh currently supports mcp targets only"))
		return
	}
	discovered, err := s.discoverMCPCapabilities(r.Context(), target)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, target.TenantID, target.WorkspaceID, "capabilities.refreshed", "agent", target.ID, "Capabilities refreshed", map[string]any{
		"targetId":        target.ID,
		"capabilityCount": len(discovered),
	})); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, discovered)
}

func (s *Server) listCapabilities(w http.ResponseWriter, r *http.Request) {
	status := domain.CapabilityDiscoveryStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validCapabilityDiscoveryStatus(status) {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "status must be pending_review, approved, deprecated, or removed"))
		return
	}
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListCapabilities(r.Context(), store.CapabilityFilter{
		ManagementScope: scope,
		TargetID:        strings.TrimSpace(r.URL.Query().Get("targetId")),
		Status:          status,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) updateCapability(w http.ResponseWriter, r *http.Request) {
	existing, ok, err := s.repo.GetCapability(r.Context(), strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("capability not found"))
		return
	}
	if err := s.requireCapabilityManagementScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	var req domain.UpdateCapabilityRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	updated := existing
	if req.DiscoveryStatus != nil {
		if !validCapabilityDiscoveryStatus(*req.DiscoveryStatus) {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "discoveryStatus must be pending_review, approved, deprecated, or removed"))
			return
		}
		updated.DiscoveryStatus = *req.DiscoveryStatus
	}
	if req.Sensitivity != nil {
		if !validCapabilitySensitivity(*req.Sensitivity) {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "sensitivity must be public, internal, confidential, or restricted"))
			return
		}
		updated.Sensitivity = *req.Sensitivity
	}
	if req.RiskLevel != nil {
		if !validCapabilityRisk(*req.RiskLevel) {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "riskLevel must be low, medium, high, or critical"))
			return
		}
		updated.RiskLevel = *req.RiskLevel
	}
	if req.DataScopes != nil {
		updated.DataScopes = req.DataScopes
	}
	if req.DataDomains != nil {
		updated.DataDomains = normalizedCapabilityDataDomains(req.DataDomains)
		if !validPermissionPackageDataDomains(updated.DataDomains) {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "dataDomains must use a supported permission package data domain"))
			return
		}
	}
	if !capabilityDataDomainsConsistent(updated.DataDomains, updated.DataScopes) {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "dataDomains must match the dataDomain values in dataScopes"))
		return
	}
	updated.UpdatedAt = s.now()
	saved, ok, err := s.repo.UpdateCapability(r.Context(), updated)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("capability not found"))
		return
	}
	if target, ok, err := s.repo.GetAgent(r.Context(), saved.TargetID); err != nil {
		writeError(w, err)
		return
	} else if ok {
		if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, target.TenantID, target.WorkspaceID, "capability.updated", "capability", saved.ID, "Capability updated", map[string]any{
			"targetId":        saved.TargetID,
			"capabilityKey":   saved.Key,
			"discoveryStatus": saved.DiscoveryStatus,
			"dataDomains":     append([]string(nil), saved.DataDomains...),
			"dataScopes":      domain.CloneDataScopes(saved.DataScopes),
			"sensitivity":     saved.Sensitivity,
			"riskLevel":       saved.RiskLevel,
		})); err != nil {
			writeError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, saved)
}

var errMCPToolsListRequestPrepare = errors.New("mcp tools/list request could not be prepared")

func mcpTargetEndpoint(target domain.Agent) string {
	endpoint, _ := target.ChannelConfig["endpoint"].(string)
	return strings.TrimSpace(endpoint)
}

// newMCPToolsListRequest builds the tools/list call shared by capability
// discovery and the read-only target probe, so a probe always exercises the
// exact headers and credentials a capability refresh would send.
func newMCPToolsListRequest(ctx context.Context, target domain.Agent, endpoint string, requestID string) (*http.Request, error) {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if err != nil {
		return nil, errMCPToolsListRequestPrepare
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errMCPToolsListRequestPrepare
	}
	req.Header.Set("Content-Type", "application/json")
	if err := copyConfiguredHeaders(req.Header, target.ChannelConfig); err != nil {
		return nil, err
	}
	if err := copyCredentialHeaders(req.Header, target.ChannelConfig, target.Credentials); err != nil {
		return nil, err
	}
	setMCPUpstreamHeaders(req.Header)
	return req, nil
}

type mcpToolsListResponse struct {
	Result struct {
		Tools []mcpToolDescription `json:"tools"`
	} `json:"result"`
}

type mcpToolDescription struct {
	Name         string         `json:"name"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema"`
}

func (s *Server) discoverMCPCapabilities(ctx context.Context, target domain.Agent) ([]domain.Capability, error) {
	endpoint := mcpTargetEndpoint(target)
	if endpoint == "" {
		return nil, domain.BadRequest("VALIDATION_FAILED", "mcp target requires channelConfig.endpoint for capability discovery")
	}
	timeout, err := proxyTimeoutFromConfig(target.ChannelConfig)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := newMCPToolsListRequest(ctx, target, endpoint, "capability-discovery")
	if errors.Is(err, errMCPToolsListRequestPrepare) {
		return nil, domain.UpstreamError("capability discovery request could not be prepared")
	}
	if err != nil {
		return nil, err
	}
	resp, err := doUpstreamRequest(req)
	if err != nil {
		return nil, classifyUpstreamError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, domain.UpstreamError("capability discovery upstream returned non-2xx status")
	}
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxProxyBodyBytes+1))
	if err != nil {
		return nil, domain.UpstreamError("capability discovery response could not be read")
	}
	if len(responseBody) > maxProxyBodyBytes {
		return nil, domain.PayloadTooLarge("capability discovery response exceeds 4MiB")
	}
	var result mcpToolsListResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, domain.BadRequest("VALIDATION_FAILED", "capability discovery response must be valid MCP tools/list JSON")
	}
	existing, err := s.repo.ListCapabilities(ctx, store.CapabilityFilter{TargetID: target.ID})
	if err != nil {
		return nil, err
	}
	existingByKey := map[string]domain.Capability{}
	for _, capability := range existing {
		if capability.Type == domain.CapabilityTypeMCPTool {
			existingByKey[capability.Key] = capability
		}
	}
	now := s.now()
	seen := map[string]struct{}{}
	out := make([]domain.Capability, 0, len(result.Result.Tools))
	for _, tool := range result.Result.Tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
		capability := capabilityFromMCPTool(target.ID, tool, now)
		if current, ok := existingByKey[name]; ok {
			capability.ID = current.ID
			capability.DiscoveredAt = current.DiscoveredAt
			capability.Version = current.Version
			capability.DiscoveryStatus = current.DiscoveryStatus
			capability.DataDomains = append([]string(nil), current.DataDomains...)
			capability.DataScopes = append([]domain.DataScope(nil), current.DataScopes...)
			capability.Sensitivity = current.Sensitivity
			capability.RiskLevel = current.RiskLevel
			if mcpCapabilityChanged(current, capability) {
				capability.Version++
				capability.DiscoveryStatus = domain.CapabilityDiscoveryPendingReview
			} else {
				capability.UpdatedAt = current.UpdatedAt
			}
		}
		saved, err := s.repo.UpsertCapability(ctx, capability)
		if err != nil {
			return nil, err
		}
		out = append(out, saved)
	}
	for key, current := range existingByKey {
		if _, ok := seen[key]; ok || current.DiscoveryStatus == domain.CapabilityDiscoveryRemoved {
			continue
		}
		current.DiscoveryStatus = domain.CapabilityDiscoveryRemoved
		current.UpdatedAt = now
		saved, ok, err := s.repo.UpdateCapability(ctx, current)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, saved)
		}
	}
	return out, nil
}

func capabilityFromMCPTool(targetID string, tool mcpToolDescription, now time.Time) domain.Capability {
	name := strings.TrimSpace(tool.Name)
	displayName := strings.TrimSpace(tool.Title)
	if displayName == "" {
		displayName = name
	}
	action := inferCapabilityAction(name)
	return domain.Capability{
		ID:              security.NewID("cap"),
		TargetID:        targetID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             name,
		DisplayName:     displayName,
		Description:     strings.TrimSpace(tool.Description),
		Action:          action,
		InputSchema:     nonNilMap(tool.InputSchema),
		OutputSchema:    nonNilMap(tool.OutputSchema),
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       riskForCapabilityAction(action),
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryPendingReview,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
}

func mcpCapabilityChanged(left domain.Capability, right domain.Capability) bool {
	return left.DisplayName != right.DisplayName ||
		left.Description != right.Description ||
		left.Action != right.Action ||
		jsonStable(left.InputSchema) != jsonStable(right.InputSchema) ||
		jsonStable(left.OutputSchema) != jsonStable(right.OutputSchema)
}

func normalizedCapabilityDataDomains(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func validPermissionPackageDataDomains(values []string) bool {
	allowed := map[string]struct{}{}
	for _, template := range permissionpack.Templates() {
		if value := strings.TrimSpace(template.DefaultDataDomain); value != "" {
			allowed[value] = struct{}{}
		}
	}
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return false
		}
	}
	return true
}

func capabilityDataDomainsConsistent(dataDomains []string, dataScopes []domain.DataScope) bool {
	domainSet := map[string]struct{}{}
	for _, value := range dataDomains {
		if value = strings.TrimSpace(value); value != "" {
			domainSet[value] = struct{}{}
		}
	}
	scopeSet := map[string]struct{}{}
	for _, scope := range dataScopes {
		if value := strings.TrimSpace(scope.DataDomain); value != "" {
			scopeSet[value] = struct{}{}
		}
	}
	if len(domainSet) == 0 || len(scopeSet) == 0 {
		return true
	}
	if len(domainSet) != len(scopeSet) {
		return false
	}
	for value := range domainSet {
		if _, ok := scopeSet[value]; !ok {
			return false
		}
	}
	return true
}

func inferCapabilityAction(name string) domain.CapabilityAction {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"search", "list", "get", "query", "read"} {
		if strings.HasPrefix(lower, prefix) {
			return domain.CapabilityActionRead
		}
	}
	if strings.HasPrefix(lower, "export") || strings.Contains(lower, "export") {
		return domain.CapabilityActionExport
	}
	for _, prefix := range []string{"delete", "remove"} {
		if strings.HasPrefix(lower, prefix) {
			return domain.CapabilityActionDelete
		}
	}
	for _, prefix := range []string{"update", "create", "write", "patch"} {
		if strings.HasPrefix(lower, prefix) {
			return domain.CapabilityActionWrite
		}
	}
	return domain.CapabilityActionExecute
}

func riskForCapabilityAction(action domain.CapabilityAction) domain.CapabilityRisk {
	switch action {
	case domain.CapabilityActionRead:
		return domain.CapabilityRiskLow
	case domain.CapabilityActionExport, domain.CapabilityActionDelete, domain.CapabilityActionAdmin:
		return domain.CapabilityRiskHigh
	default:
		return domain.CapabilityRiskMedium
	}
}

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func jsonStable(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func validCapabilityDiscoveryStatus(status domain.CapabilityDiscoveryStatus) bool {
	switch status {
	case domain.CapabilityDiscoveryPendingReview, domain.CapabilityDiscoveryApproved, domain.CapabilityDiscoveryDeprecated, domain.CapabilityDiscoveryRemoved:
		return true
	default:
		return false
	}
}

func validCapabilitySensitivity(sensitivity domain.CapabilitySensitivity) bool {
	switch sensitivity {
	case domain.CapabilitySensitivityPublic, domain.CapabilitySensitivityInternal, domain.CapabilitySensitivityConfidential, domain.CapabilitySensitivityRestricted:
		return true
	default:
		return false
	}
}

func validCapabilityRisk(risk domain.CapabilityRisk) bool {
	switch risk {
	case domain.CapabilityRiskLow, domain.CapabilityRiskMedium, domain.CapabilityRiskHigh, domain.CapabilityRiskCritical:
		return true
	default:
		return false
	}
}

func validPermissionPackageApprovalStatus(status domain.PermissionPackageApprovalStatus) bool {
	switch status {
	case domain.PermissionPackageApprovalStatusPending,
		domain.PermissionPackageApprovalStatusApproved,
		domain.PermissionPackageApprovalStatusRejected,
		domain.PermissionPackageApprovalStatusWithdrawn:
		return true
	default:
		return false
	}
}
