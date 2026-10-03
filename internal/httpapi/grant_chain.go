package httpapi

import (
	"context"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

func (s *Server) createTenantEntitlement(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTenantEntitlementRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.CapabilityID = strings.TrimSpace(req.CapabilityID)
	if req.TenantID == "" || req.TargetID == "" || req.CapabilityID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "tenantId, targetId, and capabilityId are required"))
		return
	}
	effect, err := normalizePolicyEffect(req.Effect, domain.PolicyEffectAllow)
	if err != nil {
		writeError(w, err)
		return
	}
	status, err := normalizePolicyStatus(req.Status, domain.PolicyStatusEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	priority, err := normalizePolicyPriority(req.Priority)
	if err != nil {
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
	allowedTenant, err := s.tenantCanReceiveTargetEntitlement(r.Context(), target.TenantID, req.TenantID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !allowedTenant {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "tenantId must match target tenantId or be a descendant tenant"))
		return
	}
	capability, ok, err := s.repo.GetCapability(r.Context(), req.CapabilityID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok || capability.TargetID != target.ID {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "capabilityId must belong to targetId"))
		return
	}
	if err := s.requireCapabilityManagementScope(r, capability); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requireTenantManagementScope(r, req.TenantID); err != nil {
		writeError(w, err)
		return
	}
	if _, ok := domain.EffectiveDataScopes(capability.DataScopes, req.DataScopes); !ok {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "dataScopes must be equal to or narrower than capability dataScopes"))
		return
	}
	now := s.now()
	entitlement := domain.TenantEntitlement{
		ID:           security.NewID("ent"),
		TenantID:     req.TenantID,
		TargetID:     req.TargetID,
		CapabilityID: req.CapabilityID,
		Effect:       effect,
		DataScopes:   req.DataScopes,
		Status:       status,
		Priority:     priority,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	created, err := s.repo.CreateTenantEntitlement(r.Context(), entitlement)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, target.TenantID, target.WorkspaceID, "tenant_entitlement.created", "tenant_entitlement", created.ID, "Tenant entitlement created", map[string]any{
		"targetId":      created.TargetID,
		"capabilityId":  created.CapabilityID,
		"capabilityKey": capability.Key,
		"effect":        created.Effect,
		"status":        created.Status,
		"priority":      created.Priority,
	})); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listTenantEntitlements(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{
		ManagementScope: scope,
		TargetID:        strings.TrimSpace(r.URL.Query().Get("targetId")),
		CapabilityID:    strings.TrimSpace(r.URL.Query().Get("capabilityId")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) requireTenantEntitlementManagementScope(r *http.Request, entitlement domain.TenantEntitlement) (domain.Agent, domain.Capability, error) {
	if err := s.requireTenantManagementScope(r, entitlement.TenantID); err != nil {
		return domain.Agent{}, domain.Capability{}, err
	}
	target, ok, err := s.repo.GetAgent(r.Context(), entitlement.TargetID)
	if err != nil {
		return domain.Agent{}, domain.Capability{}, err
	}
	if !ok {
		return domain.Agent{}, domain.Capability{}, domain.NotFound("target agent not found")
	}
	if err := s.requireAgentManagementScope(r, target); err != nil {
		return domain.Agent{}, domain.Capability{}, err
	}
	capability, ok, err := s.repo.GetCapability(r.Context(), entitlement.CapabilityID)
	if err != nil {
		return domain.Agent{}, domain.Capability{}, err
	}
	if !ok || capability.TargetID != entitlement.TargetID {
		return domain.Agent{}, domain.Capability{}, domain.BadRequest("VALIDATION_FAILED", "tenant entitlement capability is not registered for target")
	}
	return target, capability, nil
}

func (s *Server) createWorkspaceAssignment(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateWorkspaceAssignmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.TenantEntitlementID = strings.TrimSpace(req.TenantEntitlementID)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	if req.TenantEntitlementID == "" || req.WorkspaceID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "tenantEntitlementId and workspaceId are required"))
		return
	}
	effect, err := normalizePolicyEffect(req.Effect, domain.PolicyEffectAllow)
	if err != nil {
		writeError(w, err)
		return
	}
	status, err := normalizePolicyStatus(req.Status, domain.PolicyStatusEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	entitlements, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	entitlement, ok := findTenantEntitlement(entitlements, req.TenantEntitlementID)
	if !ok {
		writeError(w, domain.NotFound("tenant entitlement not found"))
		return
	}
	target, _, err := s.requireTenantEntitlementManagementScope(r, entitlement)
	if err != nil {
		writeError(w, err)
		return
	}
	entitlementScopes, err := s.effectiveTenantEntitlementDataScopes(r.Context(), entitlement)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, ok := domain.EffectiveDataScopes(entitlementScopes, req.DataScopes); !ok {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "dataScopes must be equal to or narrower than tenant entitlement dataScopes"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: entitlement.TenantID, WorkspaceID: req.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	assignment := domain.WorkspaceAssignment{
		ID:                  security.NewID("wsa"),
		TenantEntitlementID: entitlement.ID,
		TenantID:            entitlement.TenantID,
		WorkspaceID:         req.WorkspaceID,
		Effect:              effect,
		DataScopes:          req.DataScopes,
		Status:              status,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	created, err := s.repo.CreateWorkspaceAssignment(r.Context(), assignment)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "workspace_assignment.created", "workspace_assignment", created.ID, "Workspace assignment created", map[string]any{
		"tenantEntitlementId": created.TenantEntitlementID,
		"targetId":            target.ID,
		"capabilityId":        entitlement.CapabilityID,
		"effect":              created.Effect,
		"status":              created.Status,
	})); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listWorkspaceAssignments(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListWorkspaceAssignments(r.Context(), store.AssignmentFilter{
		ManagementScope: scope,
		EntitlementID:   strings.TrimSpace(r.URL.Query().Get("entitlementId")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) createInstanceAssignment(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateInstanceAssignmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.WorkspaceAssignmentID = strings.TrimSpace(req.WorkspaceAssignmentID)
	req.CallerInstanceID = strings.TrimSpace(req.CallerInstanceID)
	req.SubjectSelector = strings.TrimSpace(req.SubjectSelector)
	if req.WorkspaceAssignmentID == "" || req.CallerInstanceID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "workspaceAssignmentId and callerInstanceId are required"))
		return
	}
	if domain.IsUnboundedSubjectSelector(req.SubjectSelector) {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "subjectSelector is required and cannot be *"))
		return
	}
	effect, err := normalizePolicyEffect(req.Effect, domain.PolicyEffectAllow)
	if err != nil {
		writeError(w, err)
		return
	}
	status, err := normalizePolicyStatus(req.Status, domain.PolicyStatusEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	workspaceAssignments, err := s.repo.ListWorkspaceAssignments(r.Context(), store.AssignmentFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	workspaceAssignment, ok := findWorkspaceAssignment(workspaceAssignments, req.WorkspaceAssignmentID)
	if !ok {
		writeError(w, domain.NotFound("workspace assignment not found"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: workspaceAssignment.TenantID, WorkspaceID: workspaceAssignment.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	entitlements, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	entitlement, ok := findTenantEntitlement(entitlements, workspaceAssignment.TenantEntitlementID)
	if !ok {
		writeError(w, domain.NotFound("tenant entitlement not found"))
		return
	}
	if _, _, err := s.requireTenantEntitlementManagementScope(r, entitlement); err != nil {
		writeError(w, err)
		return
	}
	workspaceScopes, err := s.effectiveWorkspaceAssignmentDataScopes(r.Context(), entitlement, workspaceAssignment)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, ok := domain.EffectiveDataScopes(workspaceScopes, req.DataScopes); !ok {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "dataScopes must be equal to or narrower than workspace assignment dataScopes"))
		return
	}
	caller, ok, err := s.repo.GetAgent(r.Context(), req.CallerInstanceID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("caller instance not found"))
		return
	}
	if caller.TenantID != workspaceAssignment.TenantID || caller.WorkspaceID != workspaceAssignment.WorkspaceID {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "caller instance must match workspace assignment tenant and workspace"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: workspaceAssignment.TenantID, WorkspaceID: workspaceAssignment.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	assignment := domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              workspaceAssignment.TenantID,
		WorkspaceID:           workspaceAssignment.WorkspaceID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       req.SubjectSelector,
		Effect:                effect,
		DataScopes:            req.DataScopes,
		Status:                status,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	created, err := s.repo.CreateInstanceAssignment(r.Context(), assignment)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "instance_assignment.created", "instance_assignment", created.ID, "Instance assignment created", map[string]any{
		"workspaceAssignmentId": created.WorkspaceAssignmentID,
		"callerInstanceId":      created.CallerInstanceID,
		"effect":                created.Effect,
		"status":                created.Status,
	})); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listInstanceAssignments(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListInstanceAssignments(r.Context(), store.InstanceAssignmentFilter{
		ManagementScope:  scope,
		CallerInstanceID: strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		CapabilityID:     strings.TrimSpace(r.URL.Query().Get("capabilityId")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// Grant-chain removal (issue #413): DELETE maps to a status transition, not a
// row deletion — applied permission packages keep the row ids they recorded,
// the audit trail survives, and runtime decisions already skip disabled rows
// at every level. Parents refuse (409) while enabled children still reference
// them, so narrowing always proceeds leaves-first and nothing is left silently
// inert.
const grantChainChildrenActiveCode = "GRANT_CHAIN_CHILDREN_ACTIVE"

func (s *Server) deleteTenantEntitlement(w http.ResponseWriter, r *http.Request) {
	entitlementID := chi.URLParam(r, "id")
	entitlements, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	entitlement, ok := findTenantEntitlement(entitlements, entitlementID)
	if !ok {
		writeError(w, domain.NotFound("tenant entitlement not found"))
		return
	}
	target, capability, err := s.requireTenantEntitlementManagementScope(r, entitlement)
	if err != nil {
		writeError(w, err)
		return
	}
	children, err := s.repo.ListWorkspaceAssignments(r.Context(), store.AssignmentFilter{EntitlementID: entitlement.ID})
	if err != nil {
		writeError(w, err)
		return
	}
	if hasEnabledWorkspaceAssignment(children) {
		writeError(w, domain.Conflict(grantChainChildrenActiveCode, "tenant entitlement still has enabled workspace assignments; disable them first"))
		return
	}
	disabled, ok, err := s.repo.DisableTenantEntitlementWithAudit(r.Context(), entitlement.ID, s.now(), func(disabled domain.TenantEntitlement) domain.AuditEvent {
		return s.managementAuditEvent(r, target.TenantID, target.WorkspaceID, "tenant_entitlement.disabled", "tenant_entitlement", disabled.ID, "Tenant entitlement disabled", map[string]any{
			"targetId":      disabled.TargetID,
			"capabilityId":  disabled.CapabilityID,
			"capabilityKey": capability.Key,
			"effect":        disabled.Effect,
			"status":        disabled.Status,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("tenant entitlement not found"))
		return
	}
	writeJSON(w, http.StatusOK, disabled)
}

func (s *Server) deleteWorkspaceAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "id")
	assignments, err := s.repo.ListWorkspaceAssignments(r.Context(), store.AssignmentFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	assignment, ok := findWorkspaceAssignment(assignments, assignmentID)
	if !ok {
		writeError(w, domain.NotFound("workspace assignment not found"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: assignment.TenantID, WorkspaceID: assignment.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	entitlements, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	entitlement, ok := findTenantEntitlement(entitlements, assignment.TenantEntitlementID)
	if !ok {
		writeError(w, domain.NotFound("tenant entitlement not found"))
		return
	}
	target, capability, err := s.requireTenantEntitlementManagementScope(r, entitlement)
	if err != nil {
		writeError(w, err)
		return
	}
	children, err := s.repo.ListInstanceAssignments(r.Context(), store.InstanceAssignmentFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	if hasEnabledInstanceAssignmentForWorkspaceAssignment(children, assignment.ID) {
		writeError(w, domain.Conflict(grantChainChildrenActiveCode, "workspace assignment still has enabled instance assignments; disable them first"))
		return
	}
	disabled, ok, err := s.repo.DisableWorkspaceAssignmentWithAudit(r.Context(), assignment.ID, s.now(), func(disabled domain.WorkspaceAssignment) domain.AuditEvent {
		return s.managementAuditEvent(r, disabled.TenantID, disabled.WorkspaceID, "workspace_assignment.disabled", "workspace_assignment", disabled.ID, "Workspace assignment disabled", map[string]any{
			"tenantEntitlementId": disabled.TenantEntitlementID,
			"targetId":            target.ID,
			"capabilityId":        entitlement.CapabilityID,
			"capabilityKey":       capability.Key,
			"effect":              disabled.Effect,
			"status":              disabled.Status,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("workspace assignment not found"))
		return
	}
	writeJSON(w, http.StatusOK, disabled)
}

func (s *Server) deleteInstanceAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "id")
	assignments, err := s.repo.ListInstanceAssignments(r.Context(), store.InstanceAssignmentFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	assignment, ok := findInstanceAssignment(assignments, assignmentID)
	if !ok {
		writeError(w, domain.NotFound("instance assignment not found"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: assignment.TenantID, WorkspaceID: assignment.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	workspaceAssignments, err := s.repo.ListWorkspaceAssignments(r.Context(), store.AssignmentFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	workspaceAssignment, ok := findWorkspaceAssignment(workspaceAssignments, assignment.WorkspaceAssignmentID)
	if !ok {
		writeError(w, domain.NotFound("workspace assignment not found"))
		return
	}
	entitlements, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	entitlement, ok := findTenantEntitlement(entitlements, workspaceAssignment.TenantEntitlementID)
	if !ok {
		writeError(w, domain.NotFound("tenant entitlement not found"))
		return
	}
	if _, _, err := s.requireTenantEntitlementManagementScope(r, entitlement); err != nil {
		writeError(w, err)
		return
	}
	disabled, ok, err := s.repo.DisableInstanceAssignmentWithAudit(r.Context(), assignment.ID, s.now(), func(disabled domain.InstanceAssignment) domain.AuditEvent {
		return s.managementAuditEvent(r, disabled.TenantID, disabled.WorkspaceID, "instance_assignment.disabled", "instance_assignment", disabled.ID, "Instance assignment disabled", map[string]any{
			"workspaceAssignmentId": disabled.WorkspaceAssignmentID,
			"callerInstanceId":      disabled.CallerInstanceID,
			"effect":                disabled.Effect,
			"status":                disabled.Status,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("instance assignment not found"))
		return
	}
	writeJSON(w, http.StatusOK, disabled)
}

func hasEnabledWorkspaceAssignment(rows []domain.WorkspaceAssignment) bool {
	for _, row := range rows {
		if row.Status == domain.PolicyStatusEnabled {
			return true
		}
	}
	return false
}

func hasEnabledInstanceAssignmentForWorkspaceAssignment(rows []domain.InstanceAssignment, workspaceAssignmentID string) bool {
	for _, row := range rows {
		if row.WorkspaceAssignmentID == workspaceAssignmentID && row.Status == domain.PolicyStatusEnabled {
			return true
		}
	}
	return false
}

func findTenantEntitlement(rows []domain.TenantEntitlement, id string) (domain.TenantEntitlement, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return domain.TenantEntitlement{}, false
}

func findWorkspaceAssignment(rows []domain.WorkspaceAssignment, id string) (domain.WorkspaceAssignment, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return domain.WorkspaceAssignment{}, false
}

func findInstanceAssignment(rows []domain.InstanceAssignment, id string) (domain.InstanceAssignment, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return domain.InstanceAssignment{}, false
}

func (s *Server) effectiveTenantEntitlementDataScopes(ctx context.Context, entitlement domain.TenantEntitlement) ([]domain.DataScope, error) {
	capability, ok, err := s.repo.GetCapability(ctx, entitlement.CapabilityID)
	if err != nil {
		return nil, err
	}
	if !ok || capability.TargetID != entitlement.TargetID {
		return nil, domain.BadRequest("VALIDATION_FAILED", "tenant entitlement capability is not registered for target")
	}
	scopes, ok := domain.EffectiveDataScopes(capability.DataScopes, entitlement.DataScopes)
	if !ok {
		return nil, domain.BadRequest("VALIDATION_FAILED", "tenant entitlement dataScopes exceed capability dataScopes")
	}
	return scopes, nil
}

func (s *Server) effectiveWorkspaceAssignmentDataScopes(ctx context.Context, entitlement domain.TenantEntitlement, assignment domain.WorkspaceAssignment) ([]domain.DataScope, error) {
	entitlementScopes, err := s.effectiveTenantEntitlementDataScopes(ctx, entitlement)
	if err != nil {
		return nil, err
	}
	scopes, ok := domain.EffectiveDataScopes(entitlementScopes, assignment.DataScopes)
	if !ok {
		return nil, domain.BadRequest("VALIDATION_FAILED", "workspace assignment dataScopes exceed tenant entitlement dataScopes")
	}
	return scopes, nil
}

func (s *Server) tenantCanReceiveTargetEntitlement(ctx context.Context, targetTenantID string, granteeTenantID string) (bool, error) {
	targetTenantID = strings.TrimSpace(targetTenantID)
	granteeTenantID = strings.TrimSpace(granteeTenantID)
	if targetTenantID == "" || granteeTenantID == "" {
		return false, nil
	}
	if targetTenantID == granteeTenantID {
		return true, nil
	}
	current, ok, err := s.repo.GetTenant(ctx, granteeTenantID)
	if err != nil || !ok {
		return false, err
	}
	for current.ParentTenantID != "" {
		if current.ParentTenantID == targetTenantID {
			return true, nil
		}
		parent, ok, err := s.repo.GetTenant(ctx, current.ParentTenantID)
		if err != nil || !ok {
			return false, err
		}
		current = parent
	}
	return false, nil
}
