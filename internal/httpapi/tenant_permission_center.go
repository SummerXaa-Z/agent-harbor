package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/permissionpack"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	tenantCenterStatusReady       = "ready"
	tenantCenterStatusNeedsReview = "needs_review"
	tenantCenterStatusBlocked     = "blocked"

	tenantCenterActionStartPermissionChange = "start_permission_change"
	tenantCenterActionOpenAccessProfile     = "open_access_profile"
	tenantCenterActionManageAdministrators  = "manage_administrators"
	tenantCenterActionCompleteSetup         = "complete_setup"
)

type tenantPermissionCenterResponse struct {
	Tenant           domain.Tenant                          `json:"tenant"`
	ScopeTenants     []domain.Tenant                        `json:"scopeTenants"`
	OperatorBoundary tenantPermissionCenterOperatorBoundary `json:"operatorBoundary"`
	Administrators   []tenantPermissionCenterAdministrator  `json:"administrators"`
	Workspaces       []tenantPermissionCenterWorkspace      `json:"workspaces"`
	PermissionPacks  []tenantPermissionCenterPackage        `json:"permissionPackages"`
	Capabilities     []tenantPermissionCenterCapability     `json:"capabilities"`
	NextActions      []tenantPermissionCenterNextAction     `json:"nextActions"`
	GeneratedAt      time.Time                              `json:"generatedAt"`
}

type tenantPermissionCenterOperatorBoundary struct {
	Actor                   string `json:"actor"`
	Role                    string `json:"role"`
	TenantID                string `json:"tenantId,omitempty"`
	WorkspaceID             string `json:"workspaceId,omitempty"`
	CanManageAdministrators bool   `json:"canManageAdministrators"`
}

type tenantPermissionCenterAdministrator struct {
	ID          string `json:"id"`
	Actor       string `json:"actor"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	TenantID    string `json:"tenantId,omitempty"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	Status      string `json:"status"`
	Source      string `json:"source"`
}

type tenantPermissionCenterWorkspace struct {
	WorkspaceID     string `json:"workspaceId"`
	CallerCount     int    `json:"callerCount"`
	TargetCount     int    `json:"targetCount"`
	AssignmentCount int    `json:"assignmentCount"`
}

type tenantPermissionCenterPackage struct {
	TemplateID             string             `json:"templateId"`
	TemplateName           string             `json:"templateName"`
	Status                 string             `json:"status"`
	AllowedCapabilityCount int                `json:"allowedCapabilityCount"`
	BlockedCapabilityCount int                `json:"blockedCapabilityCount"`
	DataScopes             []domain.DataScope `json:"dataScopes,omitempty"`
	LatestApplicationID    string             `json:"latestApplicationId,omitempty"`
}

type tenantPermissionCenterCapability struct {
	TargetID       string             `json:"targetId"`
	TargetName     string             `json:"targetName"`
	CapabilityID   string             `json:"capabilityId"`
	CapabilityName string             `json:"capabilityName"`
	Effect         string             `json:"effect"`
	DataScopes     []domain.DataScope `json:"dataScopes,omitempty"`
	WorkspaceIDs   []string           `json:"workspaceIds"`
}

type tenantPermissionCenterNextAction struct {
	Code       string `json:"code"`
	TargetView string `json:"targetView"`
}

func (s *Server) getTenantPermissionCenter(w http.ResponseWriter, r *http.Request) {
	response, err := s.buildTenantPermissionCenterForRequest(r, chi.URLParam(r, "id"), r.URL.Query().Get("workspaceId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) buildTenantPermissionCenterForRequest(r *http.Request, tenantID string, workspaceID string) (tenantPermissionCenterResponse, error) {
	tenantID = strings.TrimSpace(tenantID)
	workspaceID = strings.TrimSpace(workspaceID)
	tenant, ok, err := s.repo.GetTenant(r.Context(), tenantID)
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	if !ok {
		return tenantPermissionCenterResponse{}, domain.NotFound("tenant not found")
	}
	requestedScope := store.ManagementScope{TenantID: tenant.ID, WorkspaceID: workspaceID}
	if err := s.requireRequestedScopeAllowed(r, requestedScope); err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	principal, _ := requestAdminPrincipal(r)
	effectiveScope, err := s.effectiveManagementScope(r.Context(), requestedScope, principal)
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	return s.buildTenantPermissionCenter(r.Context(), tenant, effectiveScope.WorkspaceID, principal)
}

func (s *Server) buildTenantPermissionCenter(ctx context.Context, tenant domain.Tenant, workspaceID string, principal adminPrincipal) (tenantPermissionCenterResponse, error) {
	scopeTenants, err := s.repo.ListTenants(ctx, store.TenantFilter{TenantID: tenant.ID})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	if len(scopeTenants) == 0 {
		scopeTenants = []domain.Tenant{tenant}
	}
	scope := store.ManagementScope{TenantID: tenant.ID, WorkspaceID: workspaceID}
	agents, err := s.repo.ListAgents(ctx, store.AgentFilter{ManagementScope: scope})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	entitlements, err := s.repo.ListTenantEntitlements(ctx, store.EntitlementFilter{ManagementScope: store.ManagementScope{TenantID: tenant.ID}})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	assignments, err := s.repo.ListWorkspaceAssignments(ctx, store.AssignmentFilter{ManagementScope: scope})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	visibleEntitlements := tenantPermissionCenterVisibleEntitlements(entitlements, assignments, workspaceID)
	applications, err := s.repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{ManagementScope: store.ManagementScope{TenantID: tenant.ID, WorkspaceID: workspaceID}})
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	admins, err := s.tenantPermissionCenterAdministrators(ctx, tenant.ID, workspaceID, principal)
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	capabilities, err := s.tenantPermissionCenterCapabilities(ctx, visibleEntitlements, assignments)
	if err != nil {
		return tenantPermissionCenterResponse{}, err
	}
	response := tenantPermissionCenterResponse{
		Tenant:           tenant,
		ScopeTenants:     scopeTenants,
		OperatorBoundary: tenantPermissionCenterOperatorBoundaryFromPrincipal(principal),
		Administrators:   admins,
		Workspaces:       tenantPermissionCenterWorkspaces(agents, assignments),
		PermissionPacks:  s.tenantPermissionCenterPackages(applications, visibleEntitlements),
		Capabilities:     capabilities,
		GeneratedAt:      s.now(),
	}
	response.NextActions = tenantPermissionCenterNextActions(response)
	return response, nil
}

func tenantPermissionCenterVisibleEntitlements(entitlements []domain.TenantEntitlement, assignments []domain.WorkspaceAssignment, workspaceID string) []domain.TenantEntitlement {
	if strings.TrimSpace(workspaceID) == "" {
		return entitlements
	}
	visibleEntitlementIDs := map[string]struct{}{}
	for _, assignment := range assignments {
		visibleEntitlementIDs[assignment.TenantEntitlementID] = struct{}{}
	}
	rows := make([]domain.TenantEntitlement, 0, len(entitlements))
	for _, entitlement := range entitlements {
		if _, ok := visibleEntitlementIDs[entitlement.ID]; !ok {
			continue
		}
		rows = append(rows, entitlement)
	}
	return rows
}

func tenantPermissionCenterOperatorBoundaryFromPrincipal(principal adminPrincipal) tenantPermissionCenterOperatorBoundary {
	principal = normalizeAdminPrincipal(principal)
	return tenantPermissionCenterOperatorBoundary{
		Actor:                   principal.Actor,
		Role:                    principal.Role,
		TenantID:                principal.TenantID,
		WorkspaceID:             principal.WorkspaceID,
		CanManageAdministrators: principal.Role == adminRolePlatformAdmin || principal.Role == "",
	}
}

func (s *Server) tenantPermissionCenterAdministrators(ctx context.Context, tenantID string, workspaceID string, principal adminPrincipal) ([]tenantPermissionCenterAdministrator, error) {
	if normalizeAdminRole(principal.Role) != adminRolePlatformAdmin {
		return []tenantPermissionCenterAdministrator{}, nil
	}
	rows := append([]domain.AdminIdentity{}, s.bootstrapAdminIdentities()...)
	managed, err := s.repo.ListAdminIdentities(ctx)
	if err != nil {
		return nil, err
	}
	rows = append(rows, managed...)
	result := []tenantPermissionCenterAdministrator{}
	for _, row := range rows {
		if !tenantPermissionCenterAdminMatches(row, tenantID, workspaceID) {
			continue
		}
		result = append(result, tenantPermissionCenterAdministrator{
			ID:          row.ID,
			Actor:       row.Actor,
			DisplayName: row.DisplayName,
			Role:        string(row.Role),
			TenantID:    row.TenantID,
			WorkspaceID: row.WorkspaceID,
			Status:      string(row.Status),
			Source:      string(row.Source),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Actor < result[j].Actor })
	return result, nil
}

func tenantPermissionCenterAdminMatches(identity domain.AdminIdentity, tenantID string, workspaceID string) bool {
	if identity.Role == domain.AdminIdentityRolePlatformAdmin {
		return false
	}
	if identity.TenantID != "" && identity.TenantID != tenantID {
		return false
	}
	if workspaceID != "" && identity.WorkspaceID != "" && identity.WorkspaceID != workspaceID {
		return false
	}
	return identity.TenantID == tenantID || identity.WorkspaceID == workspaceID
}

func tenantPermissionCenterWorkspaces(agents []domain.Agent, assignments []domain.WorkspaceAssignment) []tenantPermissionCenterWorkspace {
	byID := map[string]*tenantPermissionCenterWorkspace{}
	ensure := func(workspaceID string) *tenantPermissionCenterWorkspace {
		if byID[workspaceID] == nil {
			byID[workspaceID] = &tenantPermissionCenterWorkspace{WorkspaceID: workspaceID}
		}
		return byID[workspaceID]
	}
	for _, agent := range agents {
		row := ensure(agent.WorkspaceID)
		if agent.ChannelType == "local" {
			row.CallerCount++
		} else {
			row.TargetCount++
		}
	}
	for _, assignment := range assignments {
		ensure(assignment.WorkspaceID).AssignmentCount++
	}
	rows := make([]tenantPermissionCenterWorkspace, 0, len(byID))
	for _, row := range byID {
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].WorkspaceID < rows[j].WorkspaceID })
	return rows
}

func (s *Server) tenantPermissionCenterPackages(applications []domain.PermissionPackageApplication, entitlements []domain.TenantEntitlement) []tenantPermissionCenterPackage {
	latestByTemplate := map[string]domain.PermissionPackageApplication{}
	for _, application := range applications {
		current, ok := latestByTemplate[application.TemplateID]
		if !ok || application.AppliedAt.After(current.AppliedAt) {
			latestByTemplate[application.TemplateID] = application
		}
	}
	rows := []tenantPermissionCenterPackage{}
	for _, template := range permissionpack.Templates() {
		application, hasApplication := latestByTemplate[template.ID]
		allowed := 0
		denied := 0
		scopes := []domain.DataScope{}
		for _, entitlement := range entitlements {
			if entitlement.Effect == domain.PolicyEffectAllow {
				allowed++
				scopes = append(scopes, entitlement.DataScopes...)
			}
			if entitlement.Effect == domain.PolicyEffectDeny {
				denied++
			}
		}
		status := tenantCenterStatusNeedsReview
		latestID := ""
		if hasApplication {
			status = tenantCenterStatusReady
			latestID = application.ID
		}
		if allowed == 0 && denied == 0 {
			status = tenantCenterStatusBlocked
		}
		rows = append(rows, tenantPermissionCenterPackage{
			TemplateID:             template.ID,
			TemplateName:           template.Name,
			Status:                 status,
			AllowedCapabilityCount: allowed,
			BlockedCapabilityCount: denied,
			DataScopes:             scopes,
			LatestApplicationID:    latestID,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].TemplateID < rows[j].TemplateID })
	return rows
}

func (s *Server) tenantPermissionCenterCapabilities(ctx context.Context, entitlements []domain.TenantEntitlement, assignments []domain.WorkspaceAssignment) ([]tenantPermissionCenterCapability, error) {
	workspaceIDsByEntitlement := map[string][]string{}
	for _, assignment := range assignments {
		workspaceIDsByEntitlement[assignment.TenantEntitlementID] = appendUniqueString(workspaceIDsByEntitlement[assignment.TenantEntitlementID], assignment.WorkspaceID)
	}
	rows := []tenantPermissionCenterCapability{}
	for _, entitlement := range entitlements {
		targetName := entitlement.TargetID
		target, ok, err := s.repo.GetAgent(ctx, entitlement.TargetID)
		if err != nil {
			return nil, err
		}
		if ok && strings.TrimSpace(target.Name) != "" {
			targetName = target.Name
		}
		capabilityName := entitlement.CapabilityID
		capability, ok, err := s.repo.GetCapability(ctx, entitlement.CapabilityID)
		if err != nil {
			return nil, err
		}
		if ok && strings.TrimSpace(capability.DisplayName) != "" {
			capabilityName = capability.DisplayName
		}
		rows = append(rows, tenantPermissionCenterCapability{
			TargetID:       entitlement.TargetID,
			TargetName:     targetName,
			CapabilityID:   entitlement.CapabilityID,
			CapabilityName: capabilityName,
			Effect:         string(entitlement.Effect),
			DataScopes:     entitlement.DataScopes,
			WorkspaceIDs:   workspaceIDsByEntitlement[entitlement.ID],
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TargetName == rows[j].TargetName {
			return rows[i].CapabilityName < rows[j].CapabilityName
		}
		return rows[i].TargetName < rows[j].TargetName
	})
	return rows, nil
}

func tenantPermissionCenterNextActions(center tenantPermissionCenterResponse) []tenantPermissionCenterNextAction {
	actions := []tenantPermissionCenterNextAction{
		{Code: tenantCenterActionStartPermissionChange, TargetView: "ai-admin"},
		{Code: tenantCenterActionOpenAccessProfile, TargetView: "access"},
	}
	if center.OperatorBoundary.CanManageAdministrators {
		actions = append(actions, tenantPermissionCenterNextAction{Code: tenantCenterActionManageAdministrators, TargetView: "admin-access"})
	}
	if len(center.Workspaces) == 0 || len(center.Capabilities) == 0 {
		actions = append(actions, tenantPermissionCenterNextAction{Code: tenantCenterActionCompleteSetup, TargetView: "getting-started"})
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Code < actions[j].Code })
	return actions
}
