package httpapi

import (
	"context"
	"fmt"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/permissionpack"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

func (s *Server) listPermissionPackageTemplates(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, permissionpack.Templates())
}

func (s *Server) listPermissionPackageAccessSubjects(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, permissionpack.AccessSubjects())
}

func (s *Server) listPermissionPackageApplications(w http.ResponseWriter, r *http.Request) {
	limit, err := auditLimitFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	requestedCapabilityID := strings.TrimSpace(r.URL.Query().Get("requestedCapabilityId"))
	rows, err := s.repo.ListPermissionPackageApplications(r.Context(), store.PermissionPackageApplicationFilter{
		ManagementScope:       scope,
		TemplateID:            strings.TrimSpace(r.URL.Query().Get("templateId")),
		TargetID:              strings.TrimSpace(r.URL.Query().Get("targetId")),
		CallerInstanceID:      strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		RequestedCapabilityID: requestedCapabilityID,
		Limit:                 limit,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if owned, active, err := s.holderOwnedAgentIDs(r); err != nil {
		writeError(w, err)
		return
	} else if active {
		rows = applicationsOwnedBy(rows, owned)
	}
	rows, err = s.visiblePermissionPackageApplications(r.Context(), rows, scope)
	if err != nil {
		writeError(w, err)
		return
	}
	rows = permissionPackageApplicationsForRequestedCapability(rows, requestedCapabilityID)
	writeJSON(w, http.StatusOK, rows)
}

type permissionPackageApplicationHealthResponse struct {
	Summary      permissionPackageApplicationHealthSummary `json:"summary"`
	Applications []permissionPackageApplicationHealthRow   `json:"applications"`
}

type permissionPackageApplicationHealthSummary struct {
	Total       int `json:"total"`
	Ready       int `json:"ready"`
	Drifted     int `json:"drifted"`
	NeedsReview int `json:"needsReview"`
}

type permissionPackageApplicationHealthRow struct {
	Application        domain.PermissionPackageApplication `json:"application"`
	Status             string                              `json:"status"`
	BlockerCodes       []string                            `json:"blockerCodes"`
	CreatedObjectCount int                                 `json:"createdObjectCount"`
	ActiveObjectCount  int                                 `json:"activeObjectCount"`
	MissingObjectCount int                                 `json:"missingObjectCount"`
	RollbackReady      bool                                `json:"rollbackReady"`
}

func (s *Server) listPermissionPackageApplicationHealth(w http.ResponseWriter, r *http.Request) {
	limit, err := auditLimitFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	requestedCapabilityID := strings.TrimSpace(r.URL.Query().Get("requestedCapabilityId"))
	applications, err := s.repo.ListPermissionPackageApplications(r.Context(), store.PermissionPackageApplicationFilter{
		ManagementScope:       scope,
		TemplateID:            strings.TrimSpace(r.URL.Query().Get("templateId")),
		TargetID:              strings.TrimSpace(r.URL.Query().Get("targetId")),
		CallerInstanceID:      strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		RequestedCapabilityID: requestedCapabilityID,
		Limit:                 limit,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	applications, err = s.visiblePermissionPackageApplications(r.Context(), applications, scope)
	if err != nil {
		writeError(w, err)
		return
	}
	applications = permissionPackageApplicationsForRequestedCapability(applications, requestedCapabilityID)
	response := permissionPackageApplicationHealthResponse{
		Applications: []permissionPackageApplicationHealthRow{},
	}
	for _, application := range applications {
		impact, err := s.permissionPackageApplicationImpact(r.Context(), application)
		if err != nil {
			writeError(w, err)
			return
		}
		status := permissionPackageApplicationHealthStatus(impact)
		switch status {
		case "ready":
			response.Summary.Ready++
		case "drifted":
			response.Summary.Drifted++
		default:
			response.Summary.NeedsReview++
		}
		response.Applications = append(response.Applications, permissionPackageApplicationHealthRow{
			Application:        impact.Application,
			Status:             status,
			BlockerCodes:       append([]string{}, impact.RollbackReview.BlockerCodes...),
			CreatedObjectCount: impact.Summary.CreatedObjectCount,
			ActiveObjectCount:  impact.Summary.ActiveObjectCount,
			MissingObjectCount: impact.Summary.MissingObjectCount,
			RollbackReady:      impact.Summary.RollbackReady,
		})
	}
	response.Summary.Total = len(response.Applications)
	writeJSON(w, http.StatusOK, response)
}

func permissionPackageApplicationHealthStatus(impact permissionPackageApplicationImpactResponse) string {
	if impact.RollbackReview.Ready {
		return "ready"
	}
	if permissionPackageApplicationBlockerCodesContain(impact.RollbackReview.BlockerCodes, "missing_created_objects") ||
		permissionPackageApplicationBlockerCodesContain(impact.RollbackReview.BlockerCodes, "inactive_created_objects") {
		return "drifted"
	}
	return "needs_review"
}

func permissionPackageApplicationBlockerCodesContain(blockerCodes []string, want string) bool {
	for _, blockerCode := range blockerCodes {
		if blockerCode == want {
			return true
		}
	}
	return false
}

type permissionPackageApplicationImpactResponse struct {
	Application       domain.PermissionPackageApplication            `json:"application"`
	Summary           permissionPackageApplicationImpactSummary      `json:"summary"`
	CreatedObjects    []permissionPackageApplicationImpactObject     `json:"createdObjects"`
	CapabilityReviews []permissionPackageApplicationImpactCapability `json:"capabilityReviews"`
	RollbackReview    permissionPackageApplicationRollbackReview     `json:"rollbackReview"`
	RemediationPlan   permissionPackageApplicationRemediationPlan    `json:"remediationPlan"`
	Rehearsal         *permissionPackageApplicationImpactRehearsal   `json:"rehearsal,omitempty"`
}

type permissionPackageApplicationImpactRehearsal struct {
	Enabled  bool   `json:"enabled"`
	Scenario string `json:"scenario"`
}

type permissionPackageApplicationImpactSummary struct {
	CreatedObjectCount int  `json:"createdObjectCount"`
	ActiveObjectCount  int  `json:"activeObjectCount"`
	MissingObjectCount int  `json:"missingObjectCount"`
	RollbackReady      bool `json:"rollbackReady"`
}

type permissionPackageApplicationImpactObject struct {
	ID             string             `json:"id"`
	Type           string             `json:"type"`
	CurrentStatus  string             `json:"currentStatus"`
	RollbackAction string             `json:"rollbackAction"`
	DataScopes     []domain.DataScope `json:"dataScopes,omitempty"`
}

type permissionPackageApplicationImpactCapability struct {
	ID             string `json:"id"`
	Key            string `json:"key,omitempty"`
	CurrentStatus  string `json:"currentStatus"`
	RollbackAction string `json:"rollbackAction"`
}

type permissionPackageApplicationRollbackReview struct {
	Ready        bool     `json:"ready"`
	Blockers     []string `json:"blockers"`
	BlockerCodes []string `json:"blockerCodes"`
	Steps        []string `json:"steps"`
}

type permissionPackageApplicationRemediationPlan struct {
	ExecutionMode string                                          `json:"executionMode"`
	Ready         bool                                            `json:"ready"`
	Blockers      []string                                        `json:"blockers"`
	BlockerCodes  []string                                        `json:"blockerCodes"`
	Actions       []permissionPackageApplicationRemediationAction `json:"actions"`
}

type permissionPackageApplicationRemediationAction struct {
	ID            string `json:"id"`
	Order         int    `json:"order"`
	TargetType    string `json:"targetType"`
	TargetID      string `json:"targetId"`
	Action        string `json:"action"`
	CurrentStatus string `json:"currentStatus,omitempty"`
	Reason        string `json:"reason"`
	ReadOnly      bool   `json:"readOnly"`
}

func (s *Server) getPermissionPackageApplicationImpact(w http.ResponseWriter, r *http.Request) {
	applicationID := strings.TrimSpace(chi.URLParam(r, "id"))
	if applicationID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "permission package application id is required"))
		return
	}
	rehearsal := strings.TrimSpace(r.URL.Query().Get("rehearsal"))
	if rehearsal != "" && rehearsal != "grant_drift" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "rehearsal must be grant_drift when provided"))
		return
	}
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListPermissionPackageApplications(r.Context(), store.PermissionPackageApplicationFilter{
		ID:              applicationID,
		ManagementScope: scope,
		Limit:           1,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if len(rows) == 0 {
		writeError(w, domain.NotFound("permission package application not found"))
		return
	}
	rows, err = s.visiblePermissionPackageApplications(r.Context(), rows, scope)
	if err != nil {
		writeError(w, err)
		return
	}
	if len(rows) == 0 {
		writeError(w, domain.NotFound("permission package application not found"))
		return
	}
	impact, err := s.permissionPackageApplicationImpact(r.Context(), rows[0])
	if err != nil {
		writeError(w, err)
		return
	}
	if rehearsal == "grant_drift" {
		impact = permissionPackageApplicationImpactGrantDriftRehearsal(impact)
	}
	writeJSON(w, http.StatusOK, impact)
}

func (s *Server) permissionPackageApplicationImpact(ctx context.Context, application domain.PermissionPackageApplication) (permissionPackageApplicationImpactResponse, error) {
	createdObjects, err := s.permissionPackageApplicationImpactObjects(ctx, application)
	if err != nil {
		return permissionPackageApplicationImpactResponse{}, err
	}
	capabilityReviews, visibleCapabilityIDs, visibleCapabilityKeys, err := s.permissionPackageApplicationImpactCapabilities(ctx, application)
	if err != nil {
		return permissionPackageApplicationImpactResponse{}, err
	}
	application = permissionPackageApplicationWithVisibleCapabilities(application, visibleCapabilityIDs, visibleCapabilityKeys)
	summary := permissionPackageApplicationImpactSummaryFor(createdObjects)
	rollbackReview := permissionPackageApplicationRollbackReviewFor(application, summary)
	return permissionPackageApplicationImpactResponse{
		Application:       application,
		Summary:           summary,
		CreatedObjects:    createdObjects,
		CapabilityReviews: capabilityReviews,
		RollbackReview:    rollbackReview,
		RemediationPlan:   permissionPackageApplicationRemediationPlanFor(application, createdObjects, capabilityReviews, rollbackReview),
	}, nil
}

func permissionPackageApplicationImpactSummaryFor(createdObjects []permissionPackageApplicationImpactObject) permissionPackageApplicationImpactSummary {
	summary := permissionPackageApplicationImpactSummary{CreatedObjectCount: len(createdObjects)}
	for _, row := range createdObjects {
		switch row.CurrentStatus {
		case string(domain.PolicyStatusEnabled):
			summary.ActiveObjectCount++
		case "missing":
			summary.MissingObjectCount++
		}
	}
	summary.RollbackReady = summary.CreatedObjectCount > 0 &&
		summary.ActiveObjectCount == summary.CreatedObjectCount &&
		summary.MissingObjectCount == 0
	return summary
}

func permissionPackageApplicationImpactGrantDriftRehearsal(impact permissionPackageApplicationImpactResponse) permissionPackageApplicationImpactResponse {
	next := impact
	next.CreatedObjects = append([]permissionPackageApplicationImpactObject(nil), impact.CreatedObjects...)
	for index := range next.CreatedObjects {
		if next.CreatedObjects[index].Type != "workspace_assignment" {
			continue
		}
		next.CreatedObjects[index].CurrentStatus = "missing"
		next.CreatedObjects[index].RollbackAction = "investigate"
		next.CreatedObjects[index].DataScopes = nil
		break
	}
	instanceMarked := false
	for index := range next.CreatedObjects {
		if next.CreatedObjects[index].Type != "instance_assignment" {
			continue
		}
		next.CreatedObjects[index].CurrentStatus = string(domain.PolicyStatusDisabled)
		next.CreatedObjects[index].RollbackAction = "investigate"
		instanceMarked = true
		break
	}
	if !instanceMarked {
		for index := range next.CreatedObjects {
			if next.CreatedObjects[index].Type != "tenant_entitlement" {
				continue
			}
			next.CreatedObjects[index].CurrentStatus = string(domain.PolicyStatusDisabled)
			next.CreatedObjects[index].RollbackAction = "investigate"
			break
		}
	}
	next.Summary = permissionPackageApplicationImpactSummaryFor(next.CreatedObjects)
	next.RollbackReview = permissionPackageApplicationRollbackReviewFor(next.Application, next.Summary)
	next.RemediationPlan = permissionPackageApplicationRemediationPlanFor(next.Application, next.CreatedObjects, next.CapabilityReviews, next.RollbackReview)
	next.Rehearsal = &permissionPackageApplicationImpactRehearsal{
		Enabled:  true,
		Scenario: "grant_drift",
	}
	return next
}

func (s *Server) permissionPackageApplicationImpactObjects(ctx context.Context, application domain.PermissionPackageApplication) ([]permissionPackageApplicationImpactObject, error) {
	objects := make([]permissionPackageApplicationImpactObject, 0,
		len(application.TenantEntitlementIDs)+len(application.WorkspaceAssignmentIDs)+len(application.InstanceAssignmentIDs))

	entitlements, err := s.repo.ListTenantEntitlements(ctx, store.EntitlementFilter{
		ManagementScope: store.ManagementScope{TenantID: application.TenantID},
		TargetID:        application.TargetID,
	})
	if err != nil {
		return nil, err
	}
	entitlementByID := map[string]domain.TenantEntitlement{}
	for _, entitlement := range entitlements {
		entitlementByID[entitlement.ID] = entitlement
	}
	for _, id := range application.TenantEntitlementIDs {
		if entitlement, ok := entitlementByID[id]; ok {
			objects = append(objects, permissionPackageImpactObjectFromGrant("tenant_entitlement", entitlement.ID, string(entitlement.Status), entitlement.DataScopes))
		} else {
			objects = append(objects, missingPermissionPackageImpactObject("tenant_entitlement", id))
		}
	}

	workspaceAssignments, err := s.repo.ListWorkspaceAssignments(ctx, store.AssignmentFilter{
		ManagementScope: store.ManagementScope{TenantID: application.TenantID, WorkspaceID: application.WorkspaceID},
	})
	if err != nil {
		return nil, err
	}
	workspaceAssignmentByID := map[string]domain.WorkspaceAssignment{}
	for _, assignment := range workspaceAssignments {
		workspaceAssignmentByID[assignment.ID] = assignment
	}
	for _, id := range application.WorkspaceAssignmentIDs {
		if assignment, ok := workspaceAssignmentByID[id]; ok {
			objects = append(objects, permissionPackageImpactObjectFromGrant("workspace_assignment", assignment.ID, string(assignment.Status), assignment.DataScopes))
		} else {
			objects = append(objects, missingPermissionPackageImpactObject("workspace_assignment", id))
		}
	}

	instanceAssignments, err := s.repo.ListInstanceAssignments(ctx, store.InstanceAssignmentFilter{
		ManagementScope:  store.ManagementScope{TenantID: application.TenantID, WorkspaceID: application.WorkspaceID},
		CallerInstanceID: application.CallerInstanceID,
	})
	if err != nil {
		return nil, err
	}
	instanceAssignmentByID := map[string]domain.InstanceAssignment{}
	for _, assignment := range instanceAssignments {
		instanceAssignmentByID[assignment.ID] = assignment
	}
	for _, id := range application.InstanceAssignmentIDs {
		if assignment, ok := instanceAssignmentByID[id]; ok {
			objects = append(objects, permissionPackageImpactObjectFromGrant("instance_assignment", assignment.ID, string(assignment.Status), assignment.DataScopes))
		} else {
			objects = append(objects, missingPermissionPackageImpactObject("instance_assignment", id))
		}
	}
	return objects, nil
}

func (s *Server) permissionPackageApplicationImpactCapabilities(ctx context.Context, application domain.PermissionPackageApplication) ([]permissionPackageApplicationImpactCapability, []string, []string, error) {
	rows := make([]permissionPackageApplicationImpactCapability, 0, len(application.AllowedCapabilityIDs))
	visibleCapabilityIDs := []string{}
	visibleCapabilityKeys := []string{}
	for _, id := range application.AllowedCapabilityIDs {
		capability, ok, err := s.repo.GetCapability(ctx, id)
		if err != nil {
			return nil, nil, nil, err
		}
		if ok && capability.TargetID == application.TargetID {
			rows = append(rows, permissionPackageApplicationImpactCapability{
				ID:             capability.ID,
				Key:            capability.Key,
				CurrentStatus:  string(capability.DiscoveryStatus),
				RollbackAction: "manual_review",
			})
			visibleCapabilityIDs = append(visibleCapabilityIDs, capability.ID)
			visibleCapabilityKeys = append(visibleCapabilityKeys, capability.Key)
			continue
		}
		rows = append(rows, permissionPackageApplicationImpactCapability{
			ID:             id,
			CurrentStatus:  "missing",
			RollbackAction: "investigate",
		})
	}
	return rows, visibleCapabilityIDs, visibleCapabilityKeys, nil
}

func (s *Server) visiblePermissionPackageApplications(ctx context.Context, applications []domain.PermissionPackageApplication, scope store.ManagementScope) ([]domain.PermissionPackageApplication, error) {
	if len(applications) == 0 {
		return []domain.PermissionPackageApplication{}, nil
	}
	rows := make([]domain.PermissionPackageApplication, 0, len(applications))
	for _, application := range applications {
		visible, err := s.permissionPackageApplicationVisible(ctx, application, scope)
		if err != nil {
			return nil, err
		}
		if !visible {
			continue
		}
		sanitized, err := s.permissionPackageApplicationWithVisibleCapabilities(ctx, application)
		if err != nil {
			return nil, err
		}
		rows = append(rows, sanitized)
	}
	return rows, nil
}

func (s *Server) permissionPackageApplicationVisible(ctx context.Context, application domain.PermissionPackageApplication, scope store.ManagementScope) (bool, error) {
	if strings.TrimSpace(scope.TenantID) != "" {
		inScope, err := s.tenantCanReceiveTargetEntitlement(ctx, scope.TenantID, application.TenantID)
		if err != nil || !inScope {
			return false, err
		}
	}
	if strings.TrimSpace(scope.WorkspaceID) != "" && application.WorkspaceID != strings.TrimSpace(scope.WorkspaceID) {
		return false, nil
	}

	caller, ok, err := s.repo.GetAgent(ctx, application.CallerInstanceID)
	if err != nil || !ok {
		return false, err
	}
	if caller.TenantID != application.TenantID || caller.WorkspaceID != application.WorkspaceID {
		return false, nil
	}
	callerInScope, err := s.permissionPackageApplicationAgentInScope(ctx, caller, scope)
	if err != nil || !callerInScope {
		return false, err
	}

	target, ok, err := s.repo.GetAgent(ctx, application.TargetID)
	if err != nil || !ok {
		return false, err
	}
	allowedTenant, err := s.tenantCanReceiveTargetEntitlement(ctx, target.TenantID, application.TenantID)
	if err != nil || !allowedTenant {
		return false, err
	}
	if application.WorkspaceID != "" && target.WorkspaceID != application.WorkspaceID {
		return false, nil
	}
	return true, nil
}

func (s *Server) permissionPackageApplicationAgentInScope(ctx context.Context, agent domain.Agent, scope store.ManagementScope) (bool, error) {
	if strings.TrimSpace(scope.TenantID) != "" {
		inScope, err := s.tenantCanReceiveTargetEntitlement(ctx, scope.TenantID, agent.TenantID)
		if err != nil || !inScope {
			return false, err
		}
	}
	if strings.TrimSpace(scope.WorkspaceID) != "" && agent.WorkspaceID != strings.TrimSpace(scope.WorkspaceID) {
		return false, nil
	}
	return true, nil
}

func (s *Server) permissionPackageApplicationWithVisibleCapabilities(ctx context.Context, application domain.PermissionPackageApplication) (domain.PermissionPackageApplication, error) {
	visibleCapabilityIDs := []string{}
	visibleCapabilityKeys := []string{}
	for _, id := range application.AllowedCapabilityIDs {
		capability, ok, err := s.repo.GetCapability(ctx, id)
		if err != nil {
			return domain.PermissionPackageApplication{}, err
		}
		if !ok || capability.TargetID != application.TargetID {
			continue
		}
		visibleCapabilityIDs = append(visibleCapabilityIDs, capability.ID)
		visibleCapabilityKeys = append(visibleCapabilityKeys, capability.Key)
	}
	return permissionPackageApplicationWithVisibleCapabilities(application, visibleCapabilityIDs, visibleCapabilityKeys), nil
}

func permissionPackageApplicationWithVisibleCapabilities(application domain.PermissionPackageApplication, capabilityIDs []string, capabilityKeys []string) domain.PermissionPackageApplication {
	application.AllowedCapabilityIDs = append([]string(nil), capabilityIDs...)
	application.AllowedCapabilityKeys = append([]string(nil), capabilityKeys...)
	if application.RequestedCapabilityID != "" && !stringSliceContains(capabilityIDs, application.RequestedCapabilityID) {
		application.RequestedCapabilityID = ""
	}
	return application
}

func permissionPackageApplicationsForRequestedCapability(applications []domain.PermissionPackageApplication, requestedCapabilityID string) []domain.PermissionPackageApplication {
	requestedCapabilityID = strings.TrimSpace(requestedCapabilityID)
	if requestedCapabilityID == "" {
		return applications
	}
	filtered := applications[:0]
	for _, application := range applications {
		if application.RequestedCapabilityID == requestedCapabilityID {
			filtered = append(filtered, application)
		}
	}
	return filtered
}

func permissionPackageImpactObjectFromGrant(objectType string, id string, currentStatus string, dataScopes []domain.DataScope) permissionPackageApplicationImpactObject {
	rollbackAction := "investigate"
	if currentStatus == string(domain.PolicyStatusEnabled) {
		rollbackAction = "disable"
	}
	return permissionPackageApplicationImpactObject{
		ID:             id,
		Type:           objectType,
		CurrentStatus:  currentStatus,
		RollbackAction: rollbackAction,
		DataScopes:     append([]domain.DataScope(nil), dataScopes...),
	}
}

func missingPermissionPackageImpactObject(objectType string, id string) permissionPackageApplicationImpactObject {
	return permissionPackageApplicationImpactObject{
		ID:             id,
		Type:           objectType,
		CurrentStatus:  "missing",
		RollbackAction: "investigate",
	}
}

func permissionPackageApplicationRollbackReviewFor(application domain.PermissionPackageApplication, summary permissionPackageApplicationImpactSummary) permissionPackageApplicationRollbackReview {
	review := permissionPackageApplicationRollbackReview{
		Ready:        summary.RollbackReady,
		Blockers:     []string{},
		BlockerCodes: []string{},
		Steps: []string{
			"Review capability discovery status manually; shared capabilities are not automatically downgraded by rollback.",
			"Disable recorded instance assignments before workspace assignments.",
			"Disable recorded workspace assignments before tenant entitlements.",
			"Disable recorded tenant entitlements and then verify effective access decisions.",
		},
	}
	if summary.MissingObjectCount > 0 {
		review.Blockers = append(review.Blockers, "Some recorded grant objects are missing; investigate drift before rollback.")
		review.BlockerCodes = append(review.BlockerCodes, "missing_created_objects")
	}
	if summary.ActiveObjectCount != summary.CreatedObjectCount {
		review.Blockers = append(review.Blockers, "Some recorded grant objects are not enabled; review partial rollback or manual changes.")
		review.BlockerCodes = append(review.BlockerCodes, "inactive_created_objects")
	}
	if len(application.AllowedCapabilityIDs) == 0 {
		review.Blockers = append(review.Blockers, "Application has no recorded allowed capabilities.")
		review.BlockerCodes = append(review.BlockerCodes, "no_allowed_capabilities")
	}
	if len(review.Blockers) > 0 {
		review.Ready = false
	}
	return review
}

func permissionPackageApplicationRemediationPlanFor(application domain.PermissionPackageApplication, createdObjects []permissionPackageApplicationImpactObject, capabilityReviews []permissionPackageApplicationImpactCapability, rollbackReview permissionPackageApplicationRollbackReview) permissionPackageApplicationRemediationPlan {
	plan := permissionPackageApplicationRemediationPlan{
		ExecutionMode: "read_only",
		Ready:         rollbackReview.Ready,
		Blockers:      append([]string{}, rollbackReview.Blockers...),
		BlockerCodes:  append([]string{}, rollbackReview.BlockerCodes...),
		Actions:       []permissionPackageApplicationRemediationAction{},
	}
	for _, capability := range capabilityReviews {
		action := capability.RollbackAction
		reason := "shared_capability_manual_review"
		if action == "investigate" {
			reason = "capability_drift_investigation"
		}
		plan.addAction("capability", capability.ID, action, capability.CurrentStatus, reason)
	}
	for _, objectType := range []string{"instance_assignment", "workspace_assignment", "tenant_entitlement"} {
		for _, object := range createdObjects {
			if object.Type != objectType || object.RollbackAction == "disable" {
				continue
			}
			plan.addAction(object.Type, object.ID, "investigate", object.CurrentStatus, "grant_drift_investigation")
		}
	}
	for _, objectType := range []string{"instance_assignment", "workspace_assignment", "tenant_entitlement"} {
		for _, object := range createdObjects {
			if object.Type != objectType || object.RollbackAction != "disable" {
				continue
			}
			plan.addAction(object.Type, object.ID, "disable", object.CurrentStatus, permissionPackageDisableRemediationReason(object.Type))
		}
	}
	plan.addAction("access_decision", application.ID, "verify", "", "verify_effective_access")
	if len(plan.Blockers) > 0 {
		plan.Ready = false
	}
	return plan
}

func (plan *permissionPackageApplicationRemediationPlan) addAction(targetType string, targetID string, action string, currentStatus string, reason string) {
	order := len(plan.Actions) + 1
	plan.Actions = append(plan.Actions, permissionPackageApplicationRemediationAction{
		ID:            fmt.Sprintf("remediation:%02d:%s:%s:%s", order, targetType, targetID, action),
		Order:         order,
		TargetType:    targetType,
		TargetID:      targetID,
		Action:        action,
		CurrentStatus: currentStatus,
		Reason:        reason,
		ReadOnly:      true,
	})
}

func permissionPackageDisableRemediationReason(objectType string) string {
	switch objectType {
	case "instance_assignment":
		return "disable_instance_assignment"
	case "workspace_assignment":
		return "disable_workspace_assignment"
	case "tenant_entitlement":
		return "disable_tenant_entitlement"
	default:
		return "grant_drift_investigation"
	}
}
