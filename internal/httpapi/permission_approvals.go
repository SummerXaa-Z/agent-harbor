package httpapi

import (
	"context"
	"errors"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/permissionpack"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/go-chi/chi/v5"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (s *Server) listPermissionPackageApprovalRequests(w http.ResponseWriter, r *http.Request) {
	limit, err := auditLimitFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	status := domain.PermissionPackageApprovalStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validPermissionPackageApprovalStatus(status) {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "status must be pending, approved, rejected, or withdrawn"))
		return
	}
	reviewer := strings.TrimSpace(r.URL.Query().Get("reviewer"))
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter := store.PermissionPackageApprovalRequestFilter{
		ManagementScope:       scope,
		TemplateID:            strings.TrimSpace(r.URL.Query().Get("templateId")),
		TargetID:              strings.TrimSpace(r.URL.Query().Get("targetId")),
		CallerInstanceID:      strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		RequestedCapabilityID: strings.TrimSpace(r.URL.Query().Get("requestedCapabilityId")),
		Status:                status,
		Limit:                 limit,
	}
	rows, err := s.listPermissionPackageApprovalRequestsForRequest(r.Context(), r, filter, reviewer, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	if owned, active, err := s.holderOwnedAgentIDs(r); err != nil {
		writeError(w, err)
		return
	} else if active {
		rows = approvalRequestsOwnedBy(rows, owned)
	}
	writeJSON(w, http.StatusOK, permissionPackageApprovalRequestResponses(rows, s.now()))
}

func (s *Server) createPermissionPackageDraft(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageDraftRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageDraftScope(r, req); err != nil {
		writeError(w, err)
		return
	}
	draft, err := s.buildPermissionPackageDraft(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (s *Server) preflightPermissionPackage(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageDraftScope(r, req.PermissionPackageDraftRequest); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.preflightPermissionPackageRequest(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) preflightPermissionPackageRequest(ctx context.Context, req domain.PermissionPackageApplyRequest) (domain.PermissionPackageApplyPreflightResponse, error) {
	req.ApprovalRequestID = strings.TrimSpace(req.ApprovalRequestID)
	draft, err := s.buildPermissionPackageDraft(ctx, req.PermissionPackageDraftRequest)
	if err != nil {
		return domain.PermissionPackageApplyPreflightResponse{}, err
	}

	result := domain.PermissionPackageApplyPreflightResponse{
		Draft:  draft,
		Checks: []domain.PermissionPackageApplyPreflightCheck{},
		Planned: domain.PermissionPackageApplyPreflightPlannedChanges{
			Capabilities:         []domain.Capability{},
			TenantEntitlements:   []domain.TenantEntitlement{},
			WorkspaceAssignments: []domain.WorkspaceAssignment{},
			InstanceAssignments:  []domain.InstanceAssignment{},
		},
		ExistingGrants:  []domain.PermissionPackageApplyPreflightExistingGrant{},
		NextActionCodes: []domain.PermissionPackagePreflightNextActionCode{},
		NextActions:     []string{},
	}

	if draft.Readiness.CanApply {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("draft_ready", domain.PermissionPackagePreflightPassed, "Permission package draft is ready to evaluate.", "", ""))
	} else {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("draft_not_ready", domain.PermissionPackagePreflightBlocking, "Permission package draft is not ready to apply.", "", ""))
		permissionPackagePreflightAddNextAction(&result, domain.PermissionPackagePreflightNextFixDraftReadiness, "Fix draft readiness blockers before applying this permission request.")
	}

	requiresApproval := !draft.PolicyGate.CanApplyDirectly
	approvalReady := false
	if requiresApproval {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("policy_gate", domain.PermissionPackagePreflightInfo, "Policy gate requires an approved permission package approval request.", "", ""))
		if req.ApprovalRequestID == "" {
			result.Checks = append(result.Checks, permissionPackagePreflightCheck("approval_request_missing", domain.PermissionPackagePreflightBlocking, "Permission package requires approval before apply.", "", ""))
			permissionPackagePreflightAddNextAction(&result, domain.PermissionPackagePreflightNextCreateApproval, "Create and approve an approval request for this permission request, then preflight again with approvalRequestId.")
		} else {
			approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(ctx, req.ApprovalRequestID)
			if err != nil {
				return domain.PermissionPackageApplyPreflightResponse{}, err
			}
			if !ok {
				result.Checks = append(result.Checks, permissionPackagePreflightCheck("approval_request_invalid", domain.PermissionPackagePreflightBlocking, "Approval request was not found.", "", ""))
				permissionPackagePreflightAddNextAction(&result, domain.PermissionPackagePreflightNextUseApprovedRequest, "Use an approved approvalRequestId that matches the current draft.")
			} else if err := validatePermissionPackageApprovalForDraft(approval, draft, s.now()); err != nil {
				result.Checks = append(result.Checks, permissionPackagePreflightCheck("approval_request_invalid", domain.PermissionPackagePreflightBlocking, err.Error(), "", ""))
				permissionPackagePreflightAddNextAction(&result, domain.PermissionPackagePreflightNextRefreshApproval, "Refresh approval or create a new approval request for the current draft.")
			} else {
				approvalReady = true
				result.Checks = append(result.Checks, permissionPackagePreflightCheck("approval_request_ready", domain.PermissionPackagePreflightPassed, "Approval request is approved and matches the current draft.", "", ""))
			}
		}
	} else {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("policy_gate", domain.PermissionPackagePreflightPassed, "Policy gate allows direct apply.", "", ""))
	}

	dataScopeConflictCount := 0
	for _, capability := range draft.AllowedCapabilities {
		effectiveScopes, ok := domain.EffectiveDataScopes(capability.DataScopes, draft.DataScopes)
		if !ok {
			dataScopeConflictCount++
			result.Checks = append(result.Checks, permissionPackagePreflightCheck("data_scope_fit", domain.PermissionPackagePreflightBlocking, "Permission package dataScopes exceed capability dataScopes.", capability.ID, capability.Key))
			permissionPackagePreflightAddNextAction(&result, domain.PermissionPackagePreflightNextNarrowDataScope, "Narrow region or data scopes so the package stays inside every capability boundary.")
			continue
		}
		result.Planned.Capabilities = append(result.Planned.Capabilities, permissionPackagePreflightPlannedCapability(capability, effectiveScopes))
		entitlement, workspaceAssignment, instanceAssignment := permissionPackagePreflightPlannedGrantChain(draft, capability, effectiveScopes)
		result.Planned.TenantEntitlements = append(result.Planned.TenantEntitlements, entitlement)
		result.Planned.WorkspaceAssignments = append(result.Planned.WorkspaceAssignments, workspaceAssignment)
		result.Planned.InstanceAssignments = append(result.Planned.InstanceAssignments, instanceAssignment)

		existingGrants, err := s.permissionPackagePreflightExistingGrants(ctx, draft, capability)
		if err != nil {
			return domain.PermissionPackageApplyPreflightResponse{}, err
		}
		for _, existing := range existingGrants {
			result.ExistingGrants = append(result.ExistingGrants, existing)
			result.Checks = append(result.Checks, permissionPackagePreflightCheck("existing_grant_chain", domain.PermissionPackagePreflightWarning, "An enabled grant chain already exists for this tenant, workspace, caller, and capability.", capability.ID, capability.Key))
			permissionPackagePreflightAddNextAction(&result, domain.PermissionPackagePreflightNextReviewExistingGrants, "Review existing grant chains before applying another permission request for the same caller and capability.")
		}
	}
	if dataScopeConflictCount == 0 {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("data_scope_fit", domain.PermissionPackagePreflightPassed, "Permission package dataScopes fit all allowed capability boundaries.", "", ""))
	}
	alreadyApplied, err := s.permissionPackagePreflightAlreadyAppliedApplications(ctx, draft)
	if err != nil {
		return domain.PermissionPackageApplyPreflightResponse{}, err
	}
	if len(alreadyApplied) > 0 {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("application_already_applied", domain.PermissionPackagePreflightBlocking, "A matching permission request has already been applied.", "", ""))
		permissionPackagePreflightAddNextAction(&result, domain.PermissionPackagePreflightNextReviewCurrentApplication, "Review the latest permission request status before applying the same permission request again.")
	}
	result.Checks = append(result.Checks, permissionPackagePreflightCheck("planned_changes", domain.PermissionPackagePreflightInfo, "Preflight planned grant objects without writing them.", "", ""))

	result.Summary = permissionPackagePreflightSummary(result, requiresApproval, approvalReady)
	if result.Summary.CanApply {
		permissionPackagePreflightAddNextAction(&result, domain.PermissionPackagePreflightNextApplyPermissionPackage, "Apply this permission request when the reviewer is ready.")
	}
	return result, nil
}

func (s *Server) createPermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageDraftRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageDraftScope(r, req); err != nil {
		writeError(w, err)
		return
	}
	created, err := s.createPermissionPackageApprovalRequestRecord(r.Context(), req, managementActor(r), s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "permission_package.approval_requested", "permission_package_approval_request", created.ID, "Permission package approval requested", permissionPackageApprovalAuditMetadata(created))); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, permissionPackageApprovalRequestResponse(created, s.now()))
}

func (s *Server) approvePermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request) {
	s.resolvePermissionPackageApprovalRequest(w, r, domain.PermissionPackageApprovalStatusApproved)
}

func (s *Server) rejectPermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request) {
	s.resolvePermissionPackageApprovalRequest(w, r, domain.PermissionPackageApprovalStatusRejected)
}

func (s *Server) withdrawPermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, domain.NotFound("approval request not found"))
		return
	}
	var req domain.PermissionPackageApprovalResolutionRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
	}
	existing, ok, err := s.repo.GetPermissionPackageApprovalRequest(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("approval request not found"))
		return
	}
	if err := s.requirePermissionPackageApprovalRequestScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	requester := managementActor(r)
	saved, err := s.withdrawPermissionPackageApprovalRequestRecord(r.Context(), existing, requester, req.Comment, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, saved.TenantID, saved.WorkspaceID, "permission_package.approval_withdrawn", "permission_package_approval_request", saved.ID, "Permission package approval withdrawn", permissionPackageApprovalAuditMetadata(saved))); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, permissionPackageApprovalRequestResponse(saved, s.now()))
}

func (s *Server) resolvePermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request, status domain.PermissionPackageApprovalStatus) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, domain.NotFound("approval request not found"))
		return
	}
	var req domain.PermissionPackageApprovalResolutionRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
	}
	existing, ok, err := s.repo.GetPermissionPackageApprovalRequest(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("approval request not found"))
		return
	}
	if err := s.requirePermissionPackageApprovalRequestScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	reviewer, err := reviewerFromRequest(req.Reviewer, r)
	if err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	if err := s.validatePermissionPackageApprovalReviewer(r.Context(), reviewer, existing); err != nil {
		writeError(w, err)
		return
	}
	saved, err := s.resolvePermissionPackageApprovalRequestRecord(r.Context(), existing, status, reviewer, req.Comment, now)
	if err != nil {
		writeError(w, err)
		return
	}
	action := "permission_package.approval_approved"
	summary := "Permission package approval approved"
	if status == domain.PermissionPackageApprovalStatusRejected {
		action = "permission_package.approval_rejected"
		summary = "Permission package approval rejected"
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, saved.TenantID, saved.WorkspaceID, action, "permission_package_approval_request", saved.ID, summary, permissionPackageApprovalAuditMetadata(saved))); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, permissionPackageApprovalRequestResponse(saved, s.now()))
}

func (s *Server) applyPermissionPackage(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.applyPermissionPackageRequest(r, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) applyPermissionPackageRequest(r *http.Request, req domain.PermissionPackageApplyRequest) (domain.PermissionPackageApplyResponse, error) {
	req.ApprovalRequestID = strings.TrimSpace(req.ApprovalRequestID)
	if err := s.requirePermissionPackageDraftScope(r, req.PermissionPackageDraftRequest); err != nil {
		return domain.PermissionPackageApplyResponse{}, err
	}
	draft, err := s.buildPermissionPackageDraft(r.Context(), req.PermissionPackageDraftRequest)
	if err != nil {
		return domain.PermissionPackageApplyResponse{}, err
	}
	if !draft.Readiness.CanApply {
		return domain.PermissionPackageApplyResponse{}, domain.BadRequest("VALIDATION_FAILED", "permission package draft is not ready to apply")
	}
	approvalRequestID := ""
	var approvalForApply *domain.PermissionPackageApprovalRequest
	if !draft.PolicyGate.CanApplyDirectly {
		if req.ApprovalRequestID == "" {
			return domain.PermissionPackageApplyResponse{}, domain.BadRequest("VALIDATION_FAILED", "permission package requires approval before apply")
		}
		approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(r.Context(), req.ApprovalRequestID)
		if err != nil {
			return domain.PermissionPackageApplyResponse{}, err
		}
		if !ok {
			return domain.PermissionPackageApplyResponse{}, domain.NotFound("approval request not found")
		}
		if err := validatePermissionPackageApprovalForDraft(approval, draft, s.now()); err != nil {
			return domain.PermissionPackageApplyResponse{}, err
		}
		approvalRequestID = approval.ID
		approvalForApply = &approval
	}

	now := s.now()
	result := domain.PermissionPackageApplyResponse{
		Draft:                draft,
		TenantEntitlements:   []domain.TenantEntitlement{},
		WorkspaceAssignments: []domain.WorkspaceAssignment{},
		InstanceAssignments:  []domain.InstanceAssignment{},
	}
	appliedCapabilityIDs := make([]string, 0, len(draft.AllowedCapabilities))
	appliedCapabilityKeys := make([]string, 0, len(draft.AllowedCapabilities))
	tenantEntitlementIDs := make([]string, 0, len(draft.AllowedCapabilities))
	workspaceAssignmentIDs := make([]string, 0, len(draft.AllowedCapabilities))
	instanceAssignmentIDs := make([]string, 0, len(draft.AllowedCapabilities))
	capabilityMutations := make([]store.PermissionPackageApplyCapabilityMutation, 0, len(draft.AllowedCapabilities))
	tenantEntitlements := make([]domain.TenantEntitlement, 0, len(draft.AllowedCapabilities))
	workspaceAssignments := make([]domain.WorkspaceAssignment, 0, len(draft.AllowedCapabilities))
	instanceAssignments := make([]domain.InstanceAssignment, 0, len(draft.AllowedCapabilities))

	for _, capability := range draft.AllowedCapabilities {
		effectiveScopes, ok := domain.EffectiveDataScopes(capability.DataScopes, draft.DataScopes)
		if !ok {
			return domain.PermissionPackageApplyResponse{}, domain.BadRequest("VALIDATION_FAILED", "permission package dataScopes exceed capability dataScopes")
		}
		updatedCapability := capability
		updatedCapability.DiscoveryStatus = domain.CapabilityDiscoveryApproved
		updatedCapability.DataScopes = effectiveScopes
		updatedCapability.UpdatedAt = now
		capabilityMutations = append(capabilityMutations, store.PermissionPackageApplyCapabilityMutation{
			ExpectedFingerprint: permissionpack.CapabilityFingerprint(capability),
			Capability:          updatedCapability,
		})

		entitlement := domain.TenantEntitlement{
			ID:           security.NewID("ent"),
			TenantID:     draft.Input.TenantID,
			TargetID:     draft.Input.TargetID,
			CapabilityID: updatedCapability.ID,
			Effect:       domain.PolicyEffectAllow,
			DataScopes:   effectiveScopes,
			Status:       domain.PolicyStatusEnabled,
			Priority:     40,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		workspaceAssignment := domain.WorkspaceAssignment{
			ID:                  security.NewID("wsa"),
			TenantEntitlementID: entitlement.ID,
			TenantID:            draft.Input.TenantID,
			WorkspaceID:         draft.Input.WorkspaceID,
			Effect:              domain.PolicyEffectAllow,
			DataScopes:          effectiveScopes,
			Status:              domain.PolicyStatusEnabled,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		instanceAssignment := domain.InstanceAssignment{
			ID:                    security.NewID("ina"),
			WorkspaceAssignmentID: workspaceAssignment.ID,
			TenantID:              draft.Input.TenantID,
			WorkspaceID:           draft.Input.WorkspaceID,
			CallerInstanceID:      draft.Input.CallerInstanceID,
			SubjectSelector:       draft.Input.SubjectSelector,
			Effect:                domain.PolicyEffectAllow,
			DataScopes:            effectiveScopes,
			Status:                domain.PolicyStatusEnabled,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		tenantEntitlements = append(tenantEntitlements, entitlement)
		workspaceAssignments = append(workspaceAssignments, workspaceAssignment)
		instanceAssignments = append(instanceAssignments, instanceAssignment)
		appliedCapabilityIDs = append(appliedCapabilityIDs, updatedCapability.ID)
		appliedCapabilityKeys = append(appliedCapabilityKeys, updatedCapability.Key)
		tenantEntitlementIDs = append(tenantEntitlementIDs, entitlement.ID)
		workspaceAssignmentIDs = append(workspaceAssignmentIDs, workspaceAssignment.ID)
		instanceAssignmentIDs = append(instanceAssignmentIDs, instanceAssignment.ID)
	}

	application := domain.PermissionPackageApplication{
		ID:                     security.NewID("ppa"),
		DraftID:                draft.ID,
		TemplateID:             draft.Template.ID,
		TemplateVersion:        draft.Template.Version,
		TenantID:               draft.Input.TenantID,
		WorkspaceID:            draft.Input.WorkspaceID,
		TargetID:               draft.Input.TargetID,
		CallerInstanceID:       draft.Input.CallerInstanceID,
		RequestedCapabilityID:  draft.Input.RequestedCapabilityID,
		SubjectSelector:        draft.Input.SubjectSelector,
		RequestText:            draft.Input.RequestText,
		Region:                 draft.Input.Region,
		DataScopes:             draft.DataScopes,
		AllowedCapabilityIDs:   appliedCapabilityIDs,
		AllowedCapabilityKeys:  appliedCapabilityKeys,
		TenantEntitlementIDs:   tenantEntitlementIDs,
		WorkspaceAssignmentIDs: workspaceAssignmentIDs,
		InstanceAssignmentIDs:  instanceAssignmentIDs,
		AppliedAt:              now,
	}
	var consumedApproval *domain.PermissionPackageApprovalRequest
	if approvalForApply != nil {
		approval := *approvalForApply
		approval.ConsumedAt = now
		approval.ConsumedByApplicationID = application.ID
		approval.UpdatedAt = now
		consumedApproval = &approval
	}

	auditMetadata := map[string]any{
		"applicationId":          application.ID,
		"draftId":                draft.ID,
		"templateId":             draft.Template.ID,
		"templateVersion":        draft.Template.Version,
		"targetId":               draft.Input.TargetID,
		"callerInstanceId":       draft.Input.CallerInstanceID,
		"requestedCapabilityId":  draft.Input.RequestedCapabilityID,
		"subjectSelector":        draft.Input.SubjectSelector,
		"allowedCapabilityIds":   appliedCapabilityIDs,
		"allowedCapabilityKeys":  appliedCapabilityKeys,
		"tenantEntitlementIds":   tenantEntitlementIDs,
		"workspaceAssignmentIds": workspaceAssignmentIDs,
		"instanceAssignmentIds":  instanceAssignmentIDs,
	}
	if consumedApproval != nil {
		auditMetadata["approvalRequestId"] = consumedApproval.ID
		auditMetadata["approvalExpiresAt"] = consumedApproval.ExpiresAt
		auditMetadata["approvalConsumedAt"] = consumedApproval.ConsumedAt
		auditMetadata["approvalConsumedByApplicationId"] = consumedApproval.ConsumedByApplicationID
	} else if approvalRequestID != "" {
		auditMetadata["approvalRequestId"] = approvalRequestID
	}
	applyResult, err := s.repo.ApplyPermissionPackage(r.Context(), store.PermissionPackageApplyMutation{
		Capabilities:         capabilityMutations,
		TenantEntitlements:   tenantEntitlements,
		WorkspaceAssignments: workspaceAssignments,
		InstanceAssignments:  instanceAssignments,
		Application:          application,
		ApprovalRequest:      consumedApproval,
		ExpectedApproval:     approvalForApply,
		AuditEvent:           s.managementAuditEvent(r, draft.Input.TenantID, draft.Input.WorkspaceID, "permission_package.applied", "permission_package", application.ID, "Permission package applied", auditMetadata),
	})
	if err != nil {
		if errors.Is(err, store.ErrPermissionPackageApprovalNotConsumable) {
			return domain.PermissionPackageApplyResponse{}, s.permissionPackageApprovalNotConsumableError(r.Context(), approvalRequestID, draft, now)
		}
		if errors.Is(err, store.ErrPermissionPackageApplicationAlreadyApplied) {
			return domain.PermissionPackageApplyResponse{}, permissionPackageApplicationAlreadyAppliedError()
		}
		if errors.Is(err, store.ErrPermissionPackageCapabilitySnapshotChanged) {
			return domain.PermissionPackageApplyResponse{}, domain.Conflict("PERMISSION_PACKAGE_CAPABILITY_CHANGED", "capability changed after this permission request was reviewed; rebuild the draft and request approval again")
		}
		return domain.PermissionPackageApplyResponse{}, err
	}
	result.TenantEntitlements = applyResult.TenantEntitlements
	result.WorkspaceAssignments = applyResult.WorkspaceAssignments
	result.InstanceAssignments = applyResult.InstanceAssignments
	result.Application = &applyResult.Application
	return result, nil
}

func permissionPackagePreflightCheck(code string, severity domain.PermissionPackagePreflightSeverity, message string, capabilityID string, capabilityKey string) domain.PermissionPackageApplyPreflightCheck {
	return domain.PermissionPackageApplyPreflightCheck{
		Code:          code,
		Severity:      severity,
		Message:       message,
		CapabilityID:  capabilityID,
		CapabilityKey: capabilityKey,
	}
}

func permissionPackagePreflightAddNextAction(result *domain.PermissionPackageApplyPreflightResponse, code domain.PermissionPackagePreflightNextActionCode, message string) {
	for _, existing := range result.NextActionCodes {
		if existing == code {
			result.NextActions = appendUniqueString(result.NextActions, message)
			return
		}
	}
	result.NextActionCodes = append(result.NextActionCodes, code)
	result.NextActions = appendUniqueString(result.NextActions, message)
}

func permissionPackagePreflightSummary(result domain.PermissionPackageApplyPreflightResponse, requiresApproval bool, approvalReady bool) domain.PermissionPackageApplyPreflightSummary {
	summary := domain.PermissionPackageApplyPreflightSummary{
		CanApply:                        true,
		PlannedCapabilityCount:          len(result.Planned.Capabilities),
		PlannedTenantEntitlementCount:   len(result.Planned.TenantEntitlements),
		PlannedWorkspaceAssignmentCount: len(result.Planned.WorkspaceAssignments),
		PlannedInstanceAssignmentCount:  len(result.Planned.InstanceAssignments),
		ExistingGrantCount:              len(result.ExistingGrants),
		RequiresApproval:                requiresApproval,
		ApprovalReady:                   approvalReady,
	}
	for _, check := range result.Checks {
		switch check.Severity {
		case domain.PermissionPackagePreflightBlocking:
			summary.BlockingCount++
		case domain.PermissionPackagePreflightWarning:
			summary.WarningCount++
		}
	}
	summary.CanApply = summary.BlockingCount == 0
	return summary
}

func permissionPackagePreflightPlannedCapability(capability domain.Capability, dataScopes []domain.DataScope) domain.Capability {
	planned := capability
	planned.DiscoveryStatus = domain.CapabilityDiscoveryApproved
	planned.DataScopes = append([]domain.DataScope(nil), dataScopes...)
	return planned
}

func permissionPackagePreflightPlannedGrantChain(draft domain.PermissionPackageDraft, capability domain.Capability, dataScopes []domain.DataScope) (domain.TenantEntitlement, domain.WorkspaceAssignment, domain.InstanceAssignment) {
	entitlementID := "planned:ent:" + capability.ID
	workspaceAssignmentID := "planned:wsa:" + capability.ID
	return domain.TenantEntitlement{
			ID:           entitlementID,
			TenantID:     draft.Input.TenantID,
			TargetID:     draft.Input.TargetID,
			CapabilityID: capability.ID,
			Effect:       domain.PolicyEffectAllow,
			DataScopes:   append([]domain.DataScope(nil), dataScopes...),
			Status:       domain.PolicyStatusEnabled,
			Priority:     40,
		}, domain.WorkspaceAssignment{
			ID:                  workspaceAssignmentID,
			TenantEntitlementID: entitlementID,
			TenantID:            draft.Input.TenantID,
			WorkspaceID:         draft.Input.WorkspaceID,
			Effect:              domain.PolicyEffectAllow,
			DataScopes:          append([]domain.DataScope(nil), dataScopes...),
			Status:              domain.PolicyStatusEnabled,
		}, domain.InstanceAssignment{
			ID:                    "planned:ina:" + capability.ID,
			WorkspaceAssignmentID: workspaceAssignmentID,
			TenantID:              draft.Input.TenantID,
			WorkspaceID:           draft.Input.WorkspaceID,
			CallerInstanceID:      draft.Input.CallerInstanceID,
			SubjectSelector:       draft.Input.SubjectSelector,
			Effect:                domain.PolicyEffectAllow,
			DataScopes:            append([]domain.DataScope(nil), dataScopes...),
			Status:                domain.PolicyStatusEnabled,
		}
}

func (s *Server) permissionPackagePreflightAlreadyAppliedApplications(ctx context.Context, draft domain.PermissionPackageDraft) ([]domain.PermissionPackageApplication, error) {
	candidate := permissionPackagePreflightApplicationCandidate(draft)
	applications, err := s.repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
		ManagementScope: store.ManagementScope{
			TenantID:    draft.Input.TenantID,
			WorkspaceID: draft.Input.WorkspaceID,
		},
		TemplateID:       draft.Template.ID,
		TargetID:         draft.Input.TargetID,
		CallerInstanceID: draft.Input.CallerInstanceID,
	})
	if err != nil {
		return nil, err
	}
	matches := []domain.PermissionPackageApplication{}
	for _, application := range applications {
		if store.PermissionPackageApplicationsShareDuplicateKey(application, candidate) {
			matches = append(matches, application)
		}
	}
	return matches, nil
}

func permissionPackagePreflightApplicationCandidate(draft domain.PermissionPackageDraft) domain.PermissionPackageApplication {
	allowedCapabilityIDs := make([]string, 0, len(draft.AllowedCapabilities))
	allowedCapabilityKeys := make([]string, 0, len(draft.AllowedCapabilities))
	for _, capability := range draft.AllowedCapabilities {
		allowedCapabilityIDs = append(allowedCapabilityIDs, capability.ID)
		allowedCapabilityKeys = append(allowedCapabilityKeys, capability.Key)
	}
	return domain.PermissionPackageApplication{
		DraftID:               draft.ID,
		TemplateID:            draft.Template.ID,
		TemplateVersion:       draft.Template.Version,
		TenantID:              draft.Input.TenantID,
		WorkspaceID:           draft.Input.WorkspaceID,
		TargetID:              draft.Input.TargetID,
		CallerInstanceID:      draft.Input.CallerInstanceID,
		RequestedCapabilityID: draft.Input.RequestedCapabilityID,
		SubjectSelector:       draft.Input.SubjectSelector,
		Region:                draft.Input.Region,
		DataScopes:            draft.DataScopes,
		AllowedCapabilityIDs:  allowedCapabilityIDs,
		AllowedCapabilityKeys: allowedCapabilityKeys,
	}
}

func (s *Server) permissionPackagePreflightExistingGrants(ctx context.Context, draft domain.PermissionPackageDraft, capability domain.Capability) ([]domain.PermissionPackageApplyPreflightExistingGrant, error) {
	entitlements, err := s.repo.ListTenantEntitlements(ctx, store.EntitlementFilter{
		ManagementScope: store.ManagementScope{TenantID: draft.Input.TenantID},
		TargetID:        draft.Input.TargetID,
		CapabilityID:    capability.ID,
	})
	if err != nil {
		return nil, err
	}
	grants := []domain.PermissionPackageApplyPreflightExistingGrant{}
	for _, entitlement := range entitlements {
		if entitlement.TenantID != draft.Input.TenantID ||
			entitlement.TargetID != draft.Input.TargetID ||
			entitlement.CapabilityID != capability.ID ||
			entitlement.Effect != domain.PolicyEffectAllow ||
			entitlement.Status != domain.PolicyStatusEnabled {
			continue
		}
		workspaceAssignments, err := s.repo.ListWorkspaceAssignments(ctx, store.AssignmentFilter{
			ManagementScope: store.ManagementScope{
				TenantID:    draft.Input.TenantID,
				WorkspaceID: draft.Input.WorkspaceID,
			},
			EntitlementID: entitlement.ID,
		})
		if err != nil {
			return nil, err
		}
		for _, workspaceAssignment := range workspaceAssignments {
			if workspaceAssignment.TenantID != draft.Input.TenantID ||
				workspaceAssignment.WorkspaceID != draft.Input.WorkspaceID ||
				workspaceAssignment.TenantEntitlementID != entitlement.ID ||
				workspaceAssignment.Effect != domain.PolicyEffectAllow ||
				workspaceAssignment.Status != domain.PolicyStatusEnabled {
				continue
			}
			instanceAssignments, err := s.repo.ListInstanceAssignments(ctx, store.InstanceAssignmentFilter{
				ManagementScope: store.ManagementScope{
					TenantID:    draft.Input.TenantID,
					WorkspaceID: draft.Input.WorkspaceID,
				},
				CallerInstanceID: draft.Input.CallerInstanceID,
				CapabilityID:     capability.ID,
			})
			if err != nil {
				return nil, err
			}
			for _, instanceAssignment := range instanceAssignments {
				if instanceAssignment.TenantID != draft.Input.TenantID ||
					instanceAssignment.WorkspaceID != draft.Input.WorkspaceID ||
					instanceAssignment.WorkspaceAssignmentID != workspaceAssignment.ID ||
					instanceAssignment.CallerInstanceID != draft.Input.CallerInstanceID ||
					instanceAssignment.Effect != domain.PolicyEffectAllow ||
					instanceAssignment.Status != domain.PolicyStatusEnabled ||
					!permissionPackageSubjectSelectorsOverlap(instanceAssignment.SubjectSelector, draft.Input.SubjectSelector) {
					continue
				}
				grants = append(grants, domain.PermissionPackageApplyPreflightExistingGrant{
					CapabilityID:          capability.ID,
					CapabilityKey:         capability.Key,
					TenantEntitlementID:   entitlement.ID,
					WorkspaceAssignmentID: workspaceAssignment.ID,
					InstanceAssignmentID:  instanceAssignment.ID,
				})
			}
		}
	}
	return grants, nil
}

func permissionPackageSubjectSelectorsOverlap(existing string, requested string) bool {
	existing = strings.TrimSpace(existing)
	requested = strings.TrimSpace(requested)
	return existing == "" || requested == "" || existing == requested
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func (s *Server) requirePermissionPackageDraftScope(r *http.Request, req domain.PermissionPackageDraftRequest) error {
	req = trimPermissionPackageDraftRequest(req)
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: req.TenantID, WorkspaceID: req.WorkspaceID}); err != nil {
		return err
	}
	if req.CallerInstanceID != "" {
		caller, ok, err := s.repo.GetAgent(r.Context(), req.CallerInstanceID)
		if err != nil {
			return err
		}
		if !ok {
			return domain.NotFound("caller instance not found")
		}
		if err := s.requireAgentManagementScope(r, caller); err != nil {
			return err
		}
		if err := s.requireHolderScope(r, req.CallerInstanceID); err != nil {
			return err
		}
	}
	if req.TargetID != "" {
		target, ok, err := s.repo.GetAgent(r.Context(), req.TargetID)
		if err != nil {
			return err
		}
		if !ok {
			return domain.NotFound("target agent not found")
		}
		if err := s.requireAgentManagementScope(r, target); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) requirePermissionPackageQueryScope(r *http.Request, query permissionPackageProductionReadinessQuery) error {
	return s.requirePermissionPackageDraftScope(r, domain.PermissionPackageDraftRequest{
		CallerInstanceID: query.CallerInstanceID,
		TargetID:         query.TargetID,
		TenantID:         query.TenantID,
		WorkspaceID:      query.WorkspaceID,
	})
}

func (s *Server) requirePermissionPackageApprovalRequestScope(r *http.Request, approval domain.PermissionPackageApprovalRequest) error {
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: approval.TenantID, WorkspaceID: approval.WorkspaceID}); err != nil {
		return err
	}
	return s.requirePermissionPackageApprovalRequestResourceScope(r, approval)
}

func (s *Server) requirePermissionPackageApprovalRequestResourceScope(r *http.Request, approval domain.PermissionPackageApprovalRequest) error {
	if strings.TrimSpace(approval.CallerInstanceID) != "" {
		caller, ok, err := s.repo.GetAgent(r.Context(), approval.CallerInstanceID)
		if err != nil {
			return err
		}
		if !ok {
			return domain.NotFound("approval request caller not found")
		}
		if caller.TenantID != approval.TenantID || caller.WorkspaceID != approval.WorkspaceID {
			return domain.PermissionDenied("approval request caller is outside authenticated admin scope")
		}
		if err := s.requireAgentManagementScope(r, caller); err != nil {
			return err
		}
	}
	if strings.TrimSpace(approval.TargetID) != "" {
		target, ok, err := s.repo.GetAgent(r.Context(), approval.TargetID)
		if err != nil {
			return err
		}
		if !ok {
			return domain.NotFound("approval request target not found")
		}
		allowedTenant, err := s.tenantCanReceiveTargetEntitlement(r.Context(), target.TenantID, approval.TenantID)
		if err != nil {
			return err
		}
		if !allowedTenant || (approval.WorkspaceID != "" && target.WorkspaceID != approval.WorkspaceID) {
			return domain.PermissionDenied("approval request target is outside authenticated admin scope")
		}
		if err := s.requireAgentManagementScope(r, target); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) buildPermissionPackageDraft(ctx context.Context, req domain.PermissionPackageDraftRequest) (domain.PermissionPackageDraft, error) {
	req = trimPermissionPackageDraftRequest(req)
	if req.TargetID != "" {
		target, ok, err := s.repo.GetAgent(ctx, req.TargetID)
		if err != nil {
			return domain.PermissionPackageDraft{}, err
		}
		if !ok {
			return domain.PermissionPackageDraft{}, domain.NotFound("target agent not found")
		}
		if req.TenantID != "" {
			allowedTenant, err := s.tenantCanReceiveTargetEntitlement(ctx, target.TenantID, req.TenantID)
			if err != nil {
				return domain.PermissionPackageDraft{}, err
			}
			if !allowedTenant {
				return domain.PermissionPackageDraft{}, domain.BadRequest("VALIDATION_FAILED", "tenantId must match target tenantId or be a descendant tenant")
			}
		}
	}
	if req.CallerInstanceID != "" {
		caller, ok, err := s.repo.GetAgent(ctx, req.CallerInstanceID)
		if err != nil {
			return domain.PermissionPackageDraft{}, err
		}
		if !ok {
			return domain.PermissionPackageDraft{}, domain.NotFound("caller instance not found")
		}
		if req.TenantID != "" && caller.TenantID != req.TenantID {
			return domain.PermissionPackageDraft{}, domain.BadRequest("VALIDATION_FAILED", "caller instance must match permission package tenantId")
		}
		if req.WorkspaceID != "" && caller.WorkspaceID != req.WorkspaceID {
			return domain.PermissionPackageDraft{}, domain.BadRequest("VALIDATION_FAILED", "caller instance must match permission package workspaceId")
		}
	}
	capabilities := []domain.Capability{}
	if req.TargetID != "" {
		rows, err := s.repo.ListCapabilities(ctx, store.CapabilityFilter{TargetID: req.TargetID})
		if err != nil {
			return domain.PermissionPackageDraft{}, err
		}
		capabilities = rows
	}
	return permissionpack.BuildDraft(req, capabilities)
}

func trimPermissionPackageDraftRequest(req domain.PermissionPackageDraftRequest) domain.PermissionPackageDraftRequest {
	req.CallerInstanceID = strings.TrimSpace(req.CallerInstanceID)
	req.Region = strings.TrimSpace(req.Region)
	req.RequestText = strings.TrimSpace(req.RequestText)
	req.RequestedCapabilityID = strings.TrimSpace(req.RequestedCapabilityID)
	req.SubjectSelector = strings.TrimSpace(req.SubjectSelector)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.TemplateID = strings.TrimSpace(req.TemplateID)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	return req
}

func (s *Server) validatePermissionPackageApprovalReviewer(ctx context.Context, reviewer string, approval domain.PermissionPackageApprovalRequest) error {
	allowed, err := s.permissionPackageApprovalReviewerCanReview(ctx, reviewer, approval)
	if err != nil {
		return err
	}
	if !allowed {
		return domain.PermissionDenied("reviewer is not allowed to review this approval request")
	}
	return nil
}

func (s *Server) listPermissionPackageApprovalRequestsForReviewer(ctx context.Context, filter store.PermissionPackageApprovalRequestFilter, reviewer string, limit int) ([]domain.PermissionPackageApprovalRequest, error) {
	reviewer = strings.TrimSpace(reviewer)
	if len(s.approvalReviewers) == 0 {
		filter.Limit = permissionPackageApprovalRequestRepositoryLimit(filter.Status, limit)
		return s.repo.ListPermissionPackageApprovalRequests(ctx, filter)
	}
	if reviewer == "" {
		return nil, nil
	}
	repoLimit := permissionPackageApprovalRequestRepositoryLimit(filter.Status, limit)
	seen := map[string]struct{}{}
	rows := []domain.PermissionPackageApprovalRequest{}
	for _, rule := range s.approvalReviewers {
		if strings.TrimSpace(rule.Reviewer) != reviewer {
			continue
		}
		ruleFilter, ok, err := s.permissionPackageApprovalReviewerListFilter(ctx, filter, rule, repoLimit)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		ruleRows, err := s.repo.ListPermissionPackageApprovalRequests(ctx, ruleFilter)
		if err != nil {
			return nil, err
		}
		for _, row := range ruleRows {
			if _, exists := seen[row.ID]; exists {
				continue
			}
			allowed, err := s.permissionPackageApprovalReviewerCanReview(ctx, reviewer, row)
			if err != nil {
				return nil, err
			}
			if !allowed {
				continue
			}
			seen[row.ID] = struct{}{}
			rows = append(rows, row)
		}
	}
	sortPermissionPackageApprovalRequests(rows)
	return rows, nil
}

func (s *Server) listPermissionPackageApprovalRequestsForRequest(ctx context.Context, r *http.Request, filter store.PermissionPackageApprovalRequestFilter, reviewer string, limit int) ([]domain.PermissionPackageApprovalRequest, error) {
	reviewer, scoped, err := s.permissionPackageApprovalListReviewer(r, reviewer)
	if err != nil {
		return nil, err
	}
	var rows []domain.PermissionPackageApprovalRequest
	if scoped {
		rows, err = s.listPermissionPackageApprovalRequestsForReviewer(ctx, filter, reviewer, limit)
	} else {
		filter.Limit = permissionPackageApprovalRequestRepositoryLimit(filter.Status, limit)
		rows, err = s.repo.ListPermissionPackageApprovalRequests(ctx, filter)
	}
	if err != nil {
		return nil, err
	}
	rows, err = s.visiblePermissionPackageApprovalRequests(ctx, rows, filter.ManagementScope)
	if err != nil {
		return nil, err
	}
	rows = permissionPackageApprovalRequestsForRequestedCapability(rows, filter.RequestedCapabilityID)
	rows = permissionPackageApprovalRequestsWithoutExpiredPending(rows, filter.Status, s.now())
	return limitPermissionPackageApprovalRequests(rows, limit), nil
}

func permissionPackageApprovalRequestsForRequestedCapability(rows []domain.PermissionPackageApprovalRequest, requestedCapabilityID string) []domain.PermissionPackageApprovalRequest {
	requestedCapabilityID = strings.TrimSpace(requestedCapabilityID)
	if requestedCapabilityID == "" {
		return rows
	}
	filtered := rows[:0]
	for _, row := range rows {
		if row.RequestedCapabilityID == requestedCapabilityID {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func (s *Server) permissionPackageApprovalListReviewer(r *http.Request, reviewer string) (string, bool, error) {
	reviewer = strings.TrimSpace(reviewer)
	if reviewer != "" {
		resolved, err := reviewerFromRequest(reviewer, r)
		return resolved, true, err
	}
	if len(s.approvalReviewers) == 0 {
		return "", false, nil
	}
	principal, ok := requestAdminPrincipal(r)
	if !ok || principal.Role == adminRolePlatformAdmin {
		return "", false, nil
	}
	resolved, err := reviewerFromRequest("", r)
	return resolved, true, err
}

func (s *Server) permissionPackageApprovalReviewerListFilter(ctx context.Context, filter store.PermissionPackageApprovalRequestFilter, rule domain.PermissionPackageApprovalReviewer, limit int) (store.PermissionPackageApprovalRequestFilter, bool, error) {
	tenantID, ok, err := s.intersectApprovalReviewerTenantScope(ctx, filter.TenantID, rule.TenantID)
	if err != nil || !ok {
		return store.PermissionPackageApprovalRequestFilter{}, ok, err
	}
	workspaceID, ok := intersectApprovalReviewerWorkspaceScope(filter.WorkspaceID, rule.WorkspaceID)
	if !ok {
		return store.PermissionPackageApprovalRequestFilter{}, false, nil
	}
	filter.TenantID = tenantID
	filter.WorkspaceID = workspaceID
	filter.Limit = limit
	return filter, true, nil
}

func (s *Server) intersectApprovalReviewerTenantScope(ctx context.Context, requestTenantID string, ruleTenantID string) (string, bool, error) {
	requestTenantID = strings.TrimSpace(requestTenantID)
	ruleTenantID = strings.TrimSpace(ruleTenantID)
	if ruleTenantID == "" || ruleTenantID == "*" {
		return requestTenantID, true, nil
	}
	if requestTenantID == "" {
		return ruleTenantID, true, nil
	}
	if requestTenantID == ruleTenantID {
		return requestTenantID, true, nil
	}
	requestWithinRule, err := s.approvalReviewerTenantMatches(ctx, ruleTenantID, requestTenantID)
	if err != nil || requestWithinRule {
		return requestTenantID, requestWithinRule, err
	}
	ruleWithinRequest, err := s.approvalReviewerTenantMatches(ctx, requestTenantID, ruleTenantID)
	if err != nil || ruleWithinRequest {
		return ruleTenantID, ruleWithinRequest, err
	}
	return "", false, nil
}

func intersectApprovalReviewerWorkspaceScope(requestWorkspaceID string, ruleWorkspaceID string) (string, bool) {
	requestWorkspaceID = strings.TrimSpace(requestWorkspaceID)
	ruleWorkspaceID = strings.TrimSpace(ruleWorkspaceID)
	if ruleWorkspaceID == "" || ruleWorkspaceID == "*" {
		return requestWorkspaceID, true
	}
	if requestWorkspaceID == "" || requestWorkspaceID == ruleWorkspaceID {
		return ruleWorkspaceID, true
	}
	return "", false
}

func (s *Server) permissionPackageApprovalReviewerCanReview(ctx context.Context, reviewer string, approval domain.PermissionPackageApprovalRequest) (bool, error) {
	reviewer = strings.TrimSpace(reviewer)
	if len(s.approvalReviewers) == 0 {
		return reviewer != "", nil
	}
	if reviewer == "" {
		return false, nil
	}
	for _, rule := range s.approvalReviewers {
		if strings.TrimSpace(rule.Reviewer) != reviewer {
			continue
		}
		if !approvalReviewerWorkspaceMatches(rule.WorkspaceID, approval.WorkspaceID) {
			continue
		}
		matches, err := s.approvalReviewerTenantMatches(ctx, rule.TenantID, approval.TenantID)
		if err != nil {
			return false, err
		}
		if matches {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) approvalReviewerTenantMatches(ctx context.Context, ruleTenantID string, approvalTenantID string) (bool, error) {
	ruleTenantID = strings.TrimSpace(ruleTenantID)
	approvalTenantID = strings.TrimSpace(approvalTenantID)
	if ruleTenantID == "" || ruleTenantID == "*" {
		return approvalTenantID != "", nil
	}
	if ruleTenantID == approvalTenantID {
		return true, nil
	}
	tenants, err := s.repo.ListTenants(ctx, store.TenantFilter{TenantID: ruleTenantID})
	if err != nil {
		return false, err
	}
	for _, tenant := range tenants {
		if tenant.ID == approvalTenantID {
			return true, nil
		}
	}
	return false, nil
}

func approvalReviewerWorkspaceMatches(ruleWorkspaceID string, approvalWorkspaceID string) bool {
	ruleWorkspaceID = strings.TrimSpace(ruleWorkspaceID)
	if ruleWorkspaceID == "" || ruleWorkspaceID == "*" {
		return true
	}
	return ruleWorkspaceID == strings.TrimSpace(approvalWorkspaceID)
}

func sortPermissionPackageApprovalRequests(rows []domain.PermissionPackageApprovalRequest) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
}

func limitPermissionPackageApprovalRequests(rows []domain.PermissionPackageApprovalRequest, limit int) []domain.PermissionPackageApprovalRequest {
	if limit > 0 && len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func permissionPackageApprovalRequestRepositoryLimit(status domain.PermissionPackageApprovalStatus, limit int) int {
	if status == domain.PermissionPackageApprovalStatusPending {
		return 0
	}
	return limit
}

func permissionPackageApprovalRequestsWithoutExpiredPending(rows []domain.PermissionPackageApprovalRequest, status domain.PermissionPackageApprovalStatus, now time.Time) []domain.PermissionPackageApprovalRequest {
	if status != domain.PermissionPackageApprovalStatusPending || len(rows) == 0 {
		return rows
	}
	filtered := rows[:0]
	for _, row := range rows {
		if permissionPackageApprovalRequestExpired(row, now) {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func (s *Server) createPermissionPackageApprovalRequestRecord(ctx context.Context, req domain.PermissionPackageDraftRequest, requestedBy string, now time.Time) (domain.PermissionPackageApprovalRequest, error) {
	draft, err := s.buildPermissionPackageDraft(ctx, req)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if !draft.Readiness.CanApply {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "permission package draft is not ready to request approval")
	}
	if draft.PolicyGate.CanApplyDirectly {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "permission package does not require approval")
	}
	if err := s.rejectDuplicateActivePermissionPackageApprovalRequest(ctx, draft, now); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	approval := permissionPackageApprovalRequestFromDraft(draft, requestedBy, now)
	created, err := s.repo.CreatePermissionPackageApprovalRequest(ctx, approval)
	if errors.Is(err, store.ErrPermissionPackageApprovalAlreadyPending) {
		return domain.PermissionPackageApprovalRequest{}, permissionPackageApprovalAlreadyPendingError()
	}
	return created, err
}

func (s *Server) rejectDuplicateActivePermissionPackageApprovalRequest(ctx context.Context, draft domain.PermissionPackageDraft, now time.Time) error {
	rows, err := s.repo.ListPermissionPackageApprovalRequests(ctx, store.PermissionPackageApprovalRequestFilter{
		ManagementScope:            store.ManagementScope{TenantID: draft.Input.TenantID, WorkspaceID: draft.Input.WorkspaceID},
		TemplateID:                 draft.Template.ID,
		TargetID:                   draft.Input.TargetID,
		CallerInstanceID:           draft.Input.CallerInstanceID,
		RequestedCapabilityID:      draft.Input.RequestedCapabilityID,
		MatchRequestedCapabilityID: true,
		Status:                     domain.PermissionPackageApprovalStatusPending,
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if permissionPackageApprovalRequestExpired(row, now) {
			continue
		}
		if permissionPackageApprovalRequestMatchesDraftSnapshot(row, draft) {
			return permissionPackageApprovalAlreadyPendingError()
		}
	}
	return nil
}

func permissionPackageApprovalAlreadyPendingError() domain.AppError {
	return domain.Conflict("PERMISSION_PACKAGE_APPROVAL_ALREADY_PENDING", "a matching permission package approval request is already pending")
}

func (s *Server) resolvePermissionPackageApprovalRequestRecord(ctx context.Context, existing domain.PermissionPackageApprovalRequest, status domain.PermissionPackageApprovalStatus, reviewer string, comment string, now time.Time) (domain.PermissionPackageApprovalRequest, error) {
	if existing.Status != domain.PermissionPackageApprovalStatusPending {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request is already resolved")
	}
	if permissionPackageApprovalRequestExpired(existing, now) {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request has expired")
	}
	if strings.TrimSpace(existing.RequestedBy) != "" && strings.TrimSpace(reviewer) == strings.TrimSpace(existing.RequestedBy) {
		return domain.PermissionPackageApprovalRequest{}, domain.PermissionDenied("reviewer cannot resolve their own permission package approval request")
	}
	updated := existing
	updated.Status = status
	updated.ReviewedBy = strings.TrimSpace(reviewer)
	updated.ReviewComment = strings.TrimSpace(comment)
	updated.UpdatedAt = now
	updated.ResolvedAt = now
	saved, ok, err := s.repo.TransitionPermissionPackageApprovalRequest(ctx, updated, now)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if !ok {
		return domain.PermissionPackageApprovalRequest{}, s.permissionPackageApprovalTransitionUnavailableError(ctx, existing.ID, now)
	}
	return saved, nil
}

func (s *Server) withdrawPermissionPackageApprovalRequestRecord(ctx context.Context, existing domain.PermissionPackageApprovalRequest, requester string, comment string, now time.Time) (domain.PermissionPackageApprovalRequest, error) {
	if existing.Status != domain.PermissionPackageApprovalStatusPending {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request is already resolved")
	}
	if !existing.ConsumedAt.IsZero() || strings.TrimSpace(existing.ConsumedByApplicationID) != "" {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request is already consumed")
	}
	if permissionPackageApprovalRequestExpired(existing, now) {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request has expired")
	}
	requester = strings.TrimSpace(requester)
	if strings.TrimSpace(existing.RequestedBy) != "" && requester != strings.TrimSpace(existing.RequestedBy) {
		return domain.PermissionPackageApprovalRequest{}, domain.PermissionDenied("only the original requester can withdraw this approval request")
	}
	updated := existing
	updated.Status = domain.PermissionPackageApprovalStatusWithdrawn
	updated.ReviewedBy = requester
	updated.ReviewComment = strings.TrimSpace(comment)
	updated.UpdatedAt = now
	updated.ResolvedAt = now
	saved, ok, err := s.repo.TransitionPermissionPackageApprovalRequest(ctx, updated, now)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if !ok {
		return domain.PermissionPackageApprovalRequest{}, s.permissionPackageApprovalTransitionUnavailableError(ctx, existing.ID, now)
	}
	return saved, nil
}

func (s *Server) permissionPackageApprovalTransitionUnavailableError(ctx context.Context, id string, now time.Time) error {
	current, ok, err := s.repo.GetPermissionPackageApprovalRequest(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return domain.NotFound("approval request not found")
	}
	if current.Status != domain.PermissionPackageApprovalStatusPending {
		return domain.BadRequest("VALIDATION_FAILED", "approval request is already resolved")
	}
	if !current.ConsumedAt.IsZero() || strings.TrimSpace(current.ConsumedByApplicationID) != "" {
		return domain.BadRequest("VALIDATION_FAILED", "approval request is already consumed")
	}
	if permissionPackageApprovalRequestExpired(current, now) {
		return domain.BadRequest("VALIDATION_FAILED", "approval request has expired")
	}
	return domain.BadRequest("VALIDATION_FAILED", "permission package approval request is no longer available")
}

func permissionPackageApprovalRequestFromDraft(draft domain.PermissionPackageDraft, requestedBy string, now time.Time) domain.PermissionPackageApprovalRequest {
	allowedCapabilityIDs, allowedCapabilityKeys := permissionPackageCapabilityIDsAndKeys(draft.AllowedCapabilities)
	allowedCapabilityFingerprints := permissionPackageCapabilityFingerprints(draft.AllowedCapabilities)
	return domain.PermissionPackageApprovalRequest{
		ID:                            security.NewID("ppar"),
		DraftID:                       draft.ID,
		TemplateID:                    draft.Template.ID,
		TemplateVersion:               draft.Template.Version,
		PolicyVersion:                 draft.PolicyGate.PolicyVersion,
		TenantID:                      draft.Input.TenantID,
		WorkspaceID:                   draft.Input.WorkspaceID,
		TargetID:                      draft.Input.TargetID,
		CallerInstanceID:              draft.Input.CallerInstanceID,
		RequestedCapabilityID:         draft.Input.RequestedCapabilityID,
		SubjectSelector:               draft.Input.SubjectSelector,
		RequestText:                   draft.Input.RequestText,
		Region:                        draft.Input.Region,
		DataScopes:                    append([]domain.DataScope(nil), draft.DataScopes...),
		AllowedCapabilityIDs:          allowedCapabilityIDs,
		AllowedCapabilityKeys:         allowedCapabilityKeys,
		AllowedCapabilityFingerprints: allowedCapabilityFingerprints,
		PolicyGate:                    draft.PolicyGate,
		Status:                        domain.PermissionPackageApprovalStatusPending,
		RequestedBy:                   requestedBy,
		CreatedAt:                     now,
		UpdatedAt:                     now,
		ExpiresAt:                     now.Add(defaultPermissionPackageApprovalTTL),
	}
}

func permissionPackageApprovalRequestExpired(approval domain.PermissionPackageApprovalRequest, now time.Time) bool {
	return !approval.ExpiresAt.IsZero() && !now.Before(approval.ExpiresAt)
}

type permissionPackageApprovalRequestAPIResponse struct {
	domain.PermissionPackageApprovalRequest
	EffectiveStatus string `json:"effectiveStatus"`
	IsExpired       bool   `json:"isExpired"`
}

func permissionPackageApprovalRequestResponse(approval domain.PermissionPackageApprovalRequest, now time.Time) permissionPackageApprovalRequestAPIResponse {
	effectiveStatus := string(approval.Status)
	isExpired := permissionPackageApprovalRequestEffectivelyExpired(approval, now)
	if isExpired {
		effectiveStatus = "expired"
	}
	return permissionPackageApprovalRequestAPIResponse{
		PermissionPackageApprovalRequest: approval,
		EffectiveStatus:                  effectiveStatus,
		IsExpired:                        isExpired,
	}
}

func permissionPackageApprovalRequestResponses(rows []domain.PermissionPackageApprovalRequest, now time.Time) []permissionPackageApprovalRequestAPIResponse {
	out := make([]permissionPackageApprovalRequestAPIResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, permissionPackageApprovalRequestResponse(row, now))
	}
	return out
}

func permissionPackageApprovalRequestEffectivelyExpired(approval domain.PermissionPackageApprovalRequest, now time.Time) bool {
	if !permissionPackageApprovalRequestExpired(approval, now) {
		return false
	}
	switch approval.Status {
	case domain.PermissionPackageApprovalStatusPending, domain.PermissionPackageApprovalStatusApproved:
		return true
	default:
		return false
	}
}

func validatePermissionPackageApprovalForDraft(approval domain.PermissionPackageApprovalRequest, draft domain.PermissionPackageDraft, now time.Time) error {
	if approval.Status != domain.PermissionPackageApprovalStatusApproved {
		return domain.BadRequest("VALIDATION_FAILED", "permission package approval request must be approved before apply")
	}
	if !approval.ConsumedAt.IsZero() {
		return permissionPackageApprovalAlreadyConsumedError()
	}
	if permissionPackageApprovalRequestExpired(approval, now) {
		return domain.BadRequest("VALIDATION_FAILED", "permission package approval request has expired")
	}
	allowedCapabilityIDs, allowedCapabilityKeys := permissionPackageCapabilityIDsAndKeys(draft.AllowedCapabilities)
	allowedCapabilityFingerprints := permissionPackageCapabilityFingerprints(draft.AllowedCapabilities)
	if approval.DraftID != draft.ID ||
		approval.TemplateID != draft.Template.ID ||
		approval.TemplateVersion != draft.Template.Version ||
		approval.PolicyVersion != draft.PolicyGate.PolicyVersion ||
		approval.TenantID != draft.Input.TenantID ||
		approval.WorkspaceID != draft.Input.WorkspaceID ||
		approval.TargetID != draft.Input.TargetID ||
		approval.CallerInstanceID != draft.Input.CallerInstanceID ||
		approval.RequestedCapabilityID != draft.Input.RequestedCapabilityID ||
		approval.SubjectSelector != draft.Input.SubjectSelector ||
		approval.RequestText != draft.Input.RequestText ||
		approval.Region != draft.Input.Region ||
		!samePermissionPackageDataScopes(approval.DataScopes, draft.DataScopes) ||
		!sameStringSet(approval.AllowedCapabilityIDs, allowedCapabilityIDs) ||
		!sameStringSet(approval.AllowedCapabilityKeys, allowedCapabilityKeys) ||
		!sameStringSet(approval.AllowedCapabilityFingerprints, allowedCapabilityFingerprints) {
		return domain.BadRequest("VALIDATION_FAILED", "approved permission package approval request does not match current draft")
	}
	return nil
}

func (s *Server) permissionPackageApprovalNotConsumableError(ctx context.Context, approvalRequestID string, draft domain.PermissionPackageDraft, now time.Time) error {
	approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(ctx, approvalRequestID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.NotFound("approval request not found")
	}
	if !approval.ConsumedAt.IsZero() {
		return permissionPackageApprovalAlreadyConsumedError()
	}
	if err := validatePermissionPackageApprovalForDraft(approval, draft, now); err != nil {
		return err
	}
	return domain.BadRequest("VALIDATION_FAILED", "permission package approval request is no longer available")
}

func permissionPackageApprovalAlreadyConsumedError() domain.AppError {
	return domain.BadRequest("PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED", "permission package approval request is already consumed")
}

func permissionPackageApplicationAlreadyAppliedError() domain.AppError {
	return domain.Conflict("PERMISSION_PACKAGE_ALREADY_APPLIED", "a matching permission package application has already been applied")
}

func permissionPackageCapabilityIDsAndKeys(capabilities []domain.Capability) ([]string, []string) {
	ids := make([]string, 0, len(capabilities))
	keys := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		ids = append(ids, capability.ID)
		keys = append(keys, capability.Key)
	}
	return ids, keys
}

func permissionPackageCapabilityFingerprints(capabilities []domain.Capability) []string {
	fingerprints := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		fingerprints = append(fingerprints, permissionpack.CapabilityFingerprint(capability))
	}
	sort.Strings(fingerprints)
	return fingerprints
}

func samePermissionPackageDataScopes(left []domain.DataScope, right []domain.DataScope) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func permissionPackageApprovalAuditMetadata(request domain.PermissionPackageApprovalRequest) map[string]any {
	return map[string]any{
		"approvalRequestId":       request.ID,
		"draftId":                 request.DraftID,
		"templateId":              request.TemplateID,
		"templateVersion":         request.TemplateVersion,
		"policyVersion":           request.PolicyVersion,
		"targetId":                request.TargetID,
		"callerInstanceId":        request.CallerInstanceID,
		"requestedCapabilityId":   request.RequestedCapabilityID,
		"status":                  request.Status,
		"requestedBy":             request.RequestedBy,
		"reviewedBy":              request.ReviewedBy,
		"reasonCount":             len(request.PolicyGate.Reasons),
		"allowedCapabilityIds":    request.AllowedCapabilityIDs,
		"expiresAt":               request.ExpiresAt,
		"consumedAt":              request.ConsumedAt,
		"consumedByApplicationId": request.ConsumedByApplicationID,
	}
}
