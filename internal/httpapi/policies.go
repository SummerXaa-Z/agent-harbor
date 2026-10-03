package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
	"time"
)

func (s *Server) createAccessGrant(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateAccessGrantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.CallerID = strings.TrimSpace(req.CallerID)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.RouteType = strings.TrimSpace(req.RouteType)
	req.RouteKey = strings.TrimSpace(req.RouteKey)
	if req.CallerID == "" || req.TargetID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "callerAgentId and targetAgentId are required"))
		return
	}
	caller, ok, err := s.repo.GetAgent(r.Context(), req.CallerID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("caller agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, caller); err != nil {
		writeError(w, err)
		return
	}
	target, ok, err := s.repo.GetAgent(r.Context(), req.TargetID)
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
	grant := domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  req.CallerID,
		TargetID:  req.TargetID,
		RouteType: req.RouteType,
		RouteKey:  req.RouteKey,
		CreatedAt: s.now(),
	}
	created, err := s.repo.CreateAccessGrantWithAudit(r.Context(), grant, func(created domain.AccessGrant) domain.AuditEvent {
		return s.managementAuditEvent(r, caller.TenantID, caller.WorkspaceID, "access_grant.created", "access_grant", created.ID, "Access grant created", map[string]any{
			"callerAgentId": created.CallerID,
			"targetAgentId": created.TargetID,
			"routeType":     created.RouteType,
			"routeKey":      created.RouteKey,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listAccessGrants(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListAccessGrants(r.Context(), scope)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) revokeAccessGrant(w http.ResponseWriter, r *http.Request) {
	grantID := chi.URLParam(r, "id")
	tenantID, workspaceID := "", ""
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	foundForAudit := false
	grants, err := s.repo.ListAccessGrants(r.Context(), scope)
	if err != nil {
		writeError(w, err)
		return
	}
	for _, existing := range grants {
		if existing.ID != grantID {
			continue
		}
		foundForAudit = true
		if caller, ok, err := s.repo.GetAgent(r.Context(), existing.CallerID); err != nil {
			writeError(w, err)
			return
		} else if ok {
			tenantID = caller.TenantID
			workspaceID = caller.WorkspaceID
		}
		break
	}
	if !foundForAudit {
		writeError(w, domain.NotFound("access grant not found"))
		return
	}
	now := s.now()
	grant, ok, err := s.repo.RevokeAccessGrantWithAudit(r.Context(), grantID, now, func(revoked domain.AccessGrant) domain.AuditEvent {
		return s.managementAuditEvent(r, tenantID, workspaceID, "access_grant.revoked", "access_grant", revoked.ID, "Access grant revoked", map[string]any{
			"callerAgentId": revoked.CallerID,
			"targetAgentId": revoked.TargetID,
			"routeType":     revoked.RouteType,
			"routeKey":      revoked.RouteKey,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("access grant not found"))
		return
	}
	writeJSON(w, http.StatusOK, grant)
}

func (s *Server) createRoutePolicy(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateRoutePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CallerID = strings.TrimSpace(req.CallerID)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.RouteType = strings.TrimSpace(req.RouteType)
	req.RouteKey = strings.TrimSpace(req.RouteKey)
	if req.CallerID == "" || req.TargetID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "callerAgentId and targetAgentId are required"))
		return
	}
	if req.RouteType == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "routeType is required"))
		return
	}
	effect, err := normalizeRoutePolicyEffect(req.Effect, domain.RoutePolicyEffectAllow)
	if err != nil {
		writeError(w, err)
		return
	}
	status, err := normalizeRoutePolicyStatus(req.Status, domain.RoutePolicyStatusEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	priority, err := normalizeRoutePolicyPriority(req.Priority)
	if err != nil {
		writeError(w, err)
		return
	}
	retry, err := normalizeRoutePolicyRetry(req.Retry, "retry")
	if err != nil {
		writeError(w, err)
		return
	}
	caller, ok, err := s.repo.GetAgent(r.Context(), req.CallerID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("caller agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, caller); err != nil {
		writeError(w, err)
		return
	}
	target, ok, err := s.repo.GetAgent(r.Context(), req.TargetID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return
	}
	if caller.TenantID != target.TenantID || caller.WorkspaceID != target.WorkspaceID {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "caller and target agents must be in the same tenant and workspace for route policies"))
		return
	}
	if req.Name == "" {
		req.Name = defaultRoutePolicyName(req.RouteType, req.RouteKey, effect)
	}
	now := s.now()
	policy := domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    caller.TenantID,
		WorkspaceID: caller.WorkspaceID,
		Name:        req.Name,
		CallerID:    caller.ID,
		TargetID:    target.ID,
		RouteType:   req.RouteType,
		RouteKey:    req.RouteKey,
		Effect:      effect,
		Status:      status,
		Priority:    priority,
		Retry:       retry,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.rejectDuplicateRoutePolicy(r.Context(), policy); err != nil {
		writeError(w, err)
		return
	}
	created, err := s.repo.CreateRoutePolicyWithAudit(r.Context(), policy, func(created domain.RoutePolicy) domain.AuditEvent {
		return s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "route_policy.created", "route_policy", created.ID, "Route policy created", routePolicyAuditMetadata(created))
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listRoutePolicies(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListRoutePolicies(r.Context(), scope)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) requireRoutePolicyManagementScope(r *http.Request, policy domain.RoutePolicy) error {
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: policy.TenantID, WorkspaceID: policy.WorkspaceID}); err != nil {
		return err
	}
	caller, ok, err := s.repo.GetAgent(r.Context(), policy.CallerID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.NotFound("route policy not found")
	}
	target, ok, err := s.repo.GetAgent(r.Context(), policy.TargetID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.NotFound("route policy not found")
	}
	if caller.TenantID != policy.TenantID || target.TenantID != policy.TenantID ||
		caller.WorkspaceID != policy.WorkspaceID || target.WorkspaceID != policy.WorkspaceID {
		return domain.NotFound("route policy not found")
	}
	return nil
}

func (s *Server) updateRoutePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "id")
	existing, ok, err := s.repo.GetRoutePolicy(r.Context(), policyID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("route policy not found"))
		return
	}
	if err := s.requireRoutePolicyManagementScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	var req domain.UpdateRoutePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	policy := existing
	if req.Name != nil {
		policy.Name = strings.TrimSpace(*req.Name)
	}
	if req.RouteType != nil {
		policy.RouteType = strings.TrimSpace(*req.RouteType)
	}
	if req.RouteKey != nil {
		policy.RouteKey = strings.TrimSpace(*req.RouteKey)
	}
	if policy.RouteType == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "routeType is required"))
		return
	}
	if req.Effect != nil {
		effect, err := normalizeRoutePolicyEffect(*req.Effect, existing.Effect)
		if err != nil {
			writeError(w, err)
			return
		}
		policy.Effect = effect
	}
	if req.Status != nil {
		status, err := normalizeRoutePolicyStatus(*req.Status, existing.Status)
		if err != nil {
			writeError(w, err)
			return
		}
		policy.Status = status
	}
	if req.Priority != nil {
		if *req.Priority < 0 {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "priority must be zero or greater"))
			return
		}
		policy.Priority = *req.Priority
	}
	if req.Retry != nil {
		retry, err := routePolicyRetryFromPatch(req.Retry)
		if err != nil {
			writeError(w, err)
			return
		}
		policy.Retry = retry
	}
	if policy.Name == "" {
		policy.Name = defaultRoutePolicyName(policy.RouteType, policy.RouteKey, policy.Effect)
	}
	policy.UpdatedAt = s.now()
	updated, ok, err := s.repo.UpdateRoutePolicyWithAudit(r.Context(), policy, func(updated domain.RoutePolicy) domain.AuditEvent {
		return s.managementAuditEvent(r, updated.TenantID, updated.WorkspaceID, "route_policy.updated", "route_policy", updated.ID, "Route policy updated", routePolicyAuditMetadata(updated))
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("route policy not found"))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) disableRoutePolicy(w http.ResponseWriter, r *http.Request) {
	existing, ok, err := s.repo.GetRoutePolicy(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("route policy not found"))
		return
	}
	if err := s.requireRoutePolicyManagementScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	policy, ok, err := s.repo.DisableRoutePolicyWithAudit(r.Context(), existing.ID, now, func(disabled domain.RoutePolicy) domain.AuditEvent {
		return s.managementAuditEvent(r, disabled.TenantID, disabled.WorkspaceID, "route_policy.disabled", "route_policy", disabled.ID, "Route policy disabled", routePolicyAuditMetadata(disabled))
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("route policy not found"))
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func normalizePolicyEffect(value domain.PolicyEffect, fallback domain.PolicyEffect) (domain.PolicyEffect, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.PolicyEffectAllow && value != domain.PolicyEffectDeny {
		return "", domain.BadRequest("VALIDATION_FAILED", "effect must be allow or deny")
	}
	return value, nil
}

func normalizePolicyStatus(value domain.PolicyStatus, fallback domain.PolicyStatus) (domain.PolicyStatus, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.PolicyStatusEnabled && value != domain.PolicyStatusDisabled {
		return "", domain.BadRequest("VALIDATION_FAILED", "status must be enabled or disabled")
	}
	return value, nil
}

func normalizePolicyPriority(value *int) (int, error) {
	if value == nil {
		return 100, nil
	}
	if *value < 0 {
		return 0, domain.BadRequest("VALIDATION_FAILED", "priority must be zero or greater")
	}
	return *value, nil
}

func normalizeRoutePolicyEffect(value domain.RoutePolicyEffect, fallback domain.RoutePolicyEffect) (domain.RoutePolicyEffect, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.RoutePolicyEffectAllow && value != domain.RoutePolicyEffectDeny {
		return "", domain.BadRequest("VALIDATION_FAILED", "effect must be allow or deny")
	}
	return value, nil
}

func normalizeRoutePolicyStatus(value domain.RoutePolicyStatus, fallback domain.RoutePolicyStatus) (domain.RoutePolicyStatus, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.RoutePolicyStatusEnabled && value != domain.RoutePolicyStatusDisabled {
		return "", domain.BadRequest("VALIDATION_FAILED", "status must be enabled or disabled")
	}
	return value, nil
}

func normalizeRoutePolicyPriority(value *int) (int, error) {
	if value == nil {
		return 100, nil
	}
	if *value < 0 {
		return 0, domain.BadRequest("VALIDATION_FAILED", "priority must be zero or greater")
	}
	return *value, nil
}

func normalizeRoutePolicyRetry(req *domain.RoutePolicyRetryRequest, fieldPrefix string) (*domain.RoutePolicyRetry, error) {
	if req == nil {
		return nil, nil
	}
	maxAttempts := 1
	if req.MaxAttempts != nil {
		if *req.MaxAttempts < 1 || *req.MaxAttempts > maxRetryAttempts {
			return nil, domain.BadRequest("VALIDATION_FAILED", fieldPrefix+".maxAttempts must be between 1 and 4")
		}
		maxAttempts = *req.MaxAttempts
	}
	backoffMs := 0
	if req.BackoffMs != nil {
		if *req.BackoffMs < 0 || *req.BackoffMs > int(maxRetryBackoff/time.Millisecond) {
			return nil, domain.BadRequest("VALIDATION_FAILED", fieldPrefix+".backoffMs must be between 0 and 1000")
		}
		backoffMs = *req.BackoffMs
	}
	statusCodes := []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout}
	if req.StatusCodes != nil {
		statusCodes = append([]int(nil), req.StatusCodes...)
	}
	for _, statusCode := range statusCodes {
		if statusCode < 500 || statusCode > 599 {
			return nil, domain.BadRequest("VALIDATION_FAILED", fieldPrefix+".statusCodes must contain 5xx status codes")
		}
	}
	return &domain.RoutePolicyRetry{
		MaxAttempts: maxAttempts,
		BackoffMs:   backoffMs,
		StatusCodes: statusCodes,
	}, nil
}

func routePolicyRetryFromPatch(raw json.RawMessage) (*domain.RoutePolicyRetry, error) {
	if string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	var req domain.RoutePolicyRetryRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, domain.BadRequest("INVALID_JSON", "retry must be an object or null")
	}
	return normalizeRoutePolicyRetry(&req, "retry")
}

func defaultRoutePolicyName(routeType string, routeKey string, effect domain.RoutePolicyEffect) string {
	key := routeKey
	if key == "" {
		key = "*"
	}
	return string(effect) + " " + routeType + ":" + key
}

func routePolicyAuditMetadata(policy domain.RoutePolicy) map[string]any {
	metadata := map[string]any{
		"callerAgentId": policy.CallerID,
		"targetAgentId": policy.TargetID,
		"routeType":     policy.RouteType,
		"routeKey":      policy.RouteKey,
		"effect":        policy.Effect,
		"status":        policy.Status,
		"priority":      policy.Priority,
	}
	if policy.Retry != nil {
		metadata["retry"] = map[string]any{
			"maxAttempts": policy.Retry.MaxAttempts,
			"backoffMs":   policy.Retry.BackoffMs,
			"statusCodes": policy.Retry.StatusCodes,
		}
	}
	return metadata
}
