package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	adminRolePlatformAdmin    = "platform_admin"
	adminRoleTenantAdmin      = "tenant_admin"
	adminRoleSecurityReviewer = "security_reviewer"
)

type adminPrincipal struct {
	Actor       string
	Role        string
	TenantID    string
	WorkspaceID string
}

func normalizeAdminIdentity(identity AdminIdentity) (AdminIdentity, bool) {
	normalized := AdminIdentity{
		Actor:       strings.TrimSpace(identity.Actor),
		Key:         strings.TrimSpace(identity.Key),
		Role:        normalizeAdminRole(identity.Role),
		TenantID:    strings.TrimSpace(identity.TenantID),
		WorkspaceID: strings.TrimSpace(identity.WorkspaceID),
	}
	return normalized, normalized.Actor != "" && normalized.Key != ""
}

func normalizeAdminRole(role string) string {
	switch strings.TrimSpace(role) {
	case "", adminRolePlatformAdmin:
		return adminRolePlatformAdmin
	case adminRoleTenantAdmin:
		return adminRoleTenantAdmin
	case adminRoleSecurityReviewer:
		return adminRoleSecurityReviewer
	default:
		return strings.TrimSpace(role)
	}
}

func (identity AdminIdentity) principal() adminPrincipal {
	return adminPrincipal{
		Actor:       strings.TrimSpace(identity.Actor),
		Role:        normalizeAdminRole(identity.Role),
		TenantID:    strings.TrimSpace(identity.TenantID),
		WorkspaceID: strings.TrimSpace(identity.WorkspaceID),
	}
}

func platformAdminPrincipal(actor string) adminPrincipal {
	return adminPrincipal{Actor: strings.TrimSpace(actor), Role: adminRolePlatformAdmin}
}

func requestAdminPrincipal(r *http.Request) (adminPrincipal, bool) {
	principal, ok := r.Context().Value(adminActorContextKey{}).(adminPrincipal)
	if !ok {
		return adminPrincipal{}, false
	}
	principal = normalizeAdminPrincipal(principal)
	if principal.Actor == "" {
		return adminPrincipal{}, false
	}
	return principal, true
}

func normalizeAdminPrincipal(principal adminPrincipal) adminPrincipal {
	return adminPrincipal{
		Actor:       strings.TrimSpace(principal.Actor),
		Role:        normalizeAdminRole(principal.Role),
		TenantID:    strings.TrimSpace(principal.TenantID),
		WorkspaceID: strings.TrimSpace(principal.WorkspaceID),
	}
}

func (s *Server) effectiveManagementScope(ctx context.Context, requested store.ManagementScope, principal adminPrincipal) (store.ManagementScope, error) {
	requested.TenantID = strings.TrimSpace(requested.TenantID)
	requested.WorkspaceID = strings.TrimSpace(requested.WorkspaceID)
	principal = normalizeAdminPrincipal(principal)
	if principal.Role == adminRolePlatformAdmin || principal.TenantID == "" {
		return requested, nil
	}
	tenantID, err := s.intersectAdminTenantScope(ctx, requested.TenantID, principal.TenantID)
	if err != nil {
		return store.ManagementScope{}, err
	}
	workspaceID, ok := intersectAdminWorkspaceScope(requested.WorkspaceID, principal.WorkspaceID)
	if !ok {
		return store.ManagementScope{}, domain.PermissionDenied("requested workspace is outside authenticated admin scope")
	}
	return store.ManagementScope{TenantID: tenantID, WorkspaceID: workspaceID}, nil
}

func (s *Server) requireRequestedScopeAllowed(r *http.Request, requested store.ManagementScope) error {
	requested.TenantID = strings.TrimSpace(requested.TenantID)
	requested.WorkspaceID = strings.TrimSpace(requested.WorkspaceID)
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return nil
	}
	principal = normalizeAdminPrincipal(principal)
	if principal.Role == adminRolePlatformAdmin || principal.TenantID == "" {
		return nil
	}
	if requested.TenantID == "" {
		return domain.PermissionDenied("resource tenant is outside authenticated admin scope")
	}
	effective, err := s.effectiveManagementScope(r.Context(), requested, principal)
	if err != nil {
		return err
	}
	if effective.TenantID != requested.TenantID {
		return domain.PermissionDenied("resource tenant is outside authenticated admin scope")
	}
	if requested.WorkspaceID != "" && effective.WorkspaceID != requested.WorkspaceID {
		return domain.PermissionDenied("resource workspace is outside authenticated admin scope")
	}
	return nil
}

func (s *Server) requireAgentManagementScope(r *http.Request, agent domain.Agent) error {
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{
		TenantID:    agent.TenantID,
		WorkspaceID: agent.WorkspaceID,
	})
}

func (s *Server) requireTenantManagementScope(r *http.Request, tenantID string) error {
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: tenantID})
}

func (s *Server) requireCapabilityManagementScope(r *http.Request, capability domain.Capability) error {
	target, ok, err := s.repo.GetAgent(r.Context(), capability.TargetID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.NotFound("target agent not found")
	}
	return s.requireAgentManagementScope(r, target)
}

func (s *Server) intersectAdminTenantScope(ctx context.Context, requestedTenantID string, principalTenantID string) (string, error) {
	requestedTenantID = strings.TrimSpace(requestedTenantID)
	principalTenantID = strings.TrimSpace(principalTenantID)
	if principalTenantID == "" {
		return requestedTenantID, nil
	}
	if requestedTenantID == "" || requestedTenantID == principalTenantID {
		return principalTenantID, nil
	}
	principalSubtree, err := s.tenantSubtreeIDs(ctx, principalTenantID)
	if err != nil {
		return "", err
	}
	if _, ok := principalSubtree[requestedTenantID]; ok {
		return requestedTenantID, nil
	}
	requestedSubtree, err := s.tenantSubtreeIDs(ctx, requestedTenantID)
	if err != nil {
		return "", err
	}
	if _, ok := requestedSubtree[principalTenantID]; ok {
		return principalTenantID, nil
	}
	return "", domain.PermissionDenied("requested tenant is outside authenticated admin scope")
}

func (s *Server) tenantSubtreeIDs(ctx context.Context, tenantID string) (map[string]struct{}, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, nil
	}
	rows, err := s.repo.ListTenants(ctx, store.TenantFilter{TenantID: tenantID})
	if err != nil {
		return nil, err
	}
	result := map[string]struct{}{}
	if len(rows) == 0 {
		result[tenantID] = struct{}{}
		return result, nil
	}
	for _, row := range rows {
		result[row.ID] = struct{}{}
	}
	return result, nil
}

func intersectAdminWorkspaceScope(requestedWorkspaceID string, principalWorkspaceID string) (string, bool) {
	requestedWorkspaceID = strings.TrimSpace(requestedWorkspaceID)
	principalWorkspaceID = strings.TrimSpace(principalWorkspaceID)
	if principalWorkspaceID == "" {
		return requestedWorkspaceID, true
	}
	if requestedWorkspaceID == "" || requestedWorkspaceID == principalWorkspaceID {
		return principalWorkspaceID, true
	}
	return "", false
}
