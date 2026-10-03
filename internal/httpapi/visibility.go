package httpapi

import (
	"context"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"strings"
)

func (s *Server) visibleAuditEvents(ctx context.Context, events []domain.AuditEvent, scope store.ManagementScope) ([]domain.AuditEvent, error) {
	rows := make([]domain.AuditEvent, 0, len(events))
	for _, event := range events {
		visible, ok, err := s.visibleAuditEvent(ctx, event, scope)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		rows = append(rows, visible)
	}
	return rows, nil
}

func (s *Server) visibleAuditEvent(ctx context.Context, event domain.AuditEvent, scope store.ManagementScope) (domain.AuditEvent, bool, error) {
	if isPermissionPackageAppliedAuditEvent(event) {
		return s.visiblePermissionPackageAppliedAuditEvent(ctx, event, scope)
	}
	if isPermissionPackageApprovalAuditEvent(event) {
		return s.visiblePermissionPackageApprovalAuditEvent(ctx, event, scope)
	}
	event.Metadata = cloneAuditMetadata(event.Metadata)
	return event, true, nil
}

func (s *Server) visiblePermissionPackageAppliedAuditEvent(ctx context.Context, event domain.AuditEvent, scope store.ManagementScope) (domain.AuditEvent, bool, error) {
	applications, err := s.repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
		ManagementScope: scope,
		ID:              event.ResourceID,
		Limit:           1,
	})
	if err != nil {
		return domain.AuditEvent{}, false, err
	}
	applications, err = s.visiblePermissionPackageApplications(ctx, applications, scope)
	if err != nil {
		return domain.AuditEvent{}, false, err
	}
	if len(applications) == 0 {
		return domain.AuditEvent{}, false, nil
	}
	event.Metadata = permissionPackageAppliedAuditMetadataForVisibleApplication(applications[0], event.Metadata)
	return event, true, nil
}

func (s *Server) visiblePermissionPackageApprovalAuditEvent(ctx context.Context, event domain.AuditEvent, scope store.ManagementScope) (domain.AuditEvent, bool, error) {
	approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(ctx, event.ResourceID)
	if err != nil || !ok {
		return domain.AuditEvent{}, false, err
	}
	visible, err := s.permissionPackageApprovalRequestVisible(ctx, approval, scope)
	if err != nil || !visible {
		return domain.AuditEvent{}, false, err
	}
	metadata, err := s.permissionPackageApprovalAuditMetadataForVisibleRequest(ctx, approval)
	if err != nil {
		return domain.AuditEvent{}, false, err
	}
	event.Metadata = metadata
	return event, true, nil
}

func isPermissionPackageAppliedAuditEvent(event domain.AuditEvent) bool {
	return event.Action == "permission_package.applied" && event.ResourceType == "permission_package"
}

func isPermissionPackageApprovalAuditEvent(event domain.AuditEvent) bool {
	if event.ResourceType != "permission_package_approval_request" {
		return false
	}
	switch event.Action {
	case "permission_package.approval_requested", "permission_package.approval_approved", "permission_package.approval_rejected", "permission_package.approval_withdrawn":
		return true
	default:
		return false
	}
}

func cloneAuditMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func permissionPackageAppliedAuditMetadataForVisibleApplication(application domain.PermissionPackageApplication, original map[string]any) map[string]any {
	metadata := map[string]any{
		"applicationId":          application.ID,
		"draftId":                application.DraftID,
		"templateId":             application.TemplateID,
		"templateVersion":        application.TemplateVersion,
		"targetId":               application.TargetID,
		"callerInstanceId":       application.CallerInstanceID,
		"requestedCapabilityId":  application.RequestedCapabilityID,
		"subjectSelector":        application.SubjectSelector,
		"allowedCapabilityIds":   auditMetadataStrings(application.AllowedCapabilityIDs),
		"allowedCapabilityKeys":  auditMetadataStrings(application.AllowedCapabilityKeys),
		"tenantEntitlementIds":   auditMetadataStrings(application.TenantEntitlementIDs),
		"workspaceAssignmentIds": auditMetadataStrings(application.WorkspaceAssignmentIDs),
		"instanceAssignmentIds":  auditMetadataStrings(application.InstanceAssignmentIDs),
	}
	for _, key := range []string{"approvalRequestId", "approvalExpiresAt", "approvalConsumedAt", "approvalConsumedByApplicationId"} {
		if value, ok := original[key]; ok {
			metadata[key] = value
		}
	}
	return metadata
}

func (s *Server) permissionPackageApprovalRequestVisible(ctx context.Context, approval domain.PermissionPackageApprovalRequest, scope store.ManagementScope) (bool, error) {
	if strings.TrimSpace(scope.TenantID) != "" {
		inScope, err := s.tenantCanReceiveTargetEntitlement(ctx, scope.TenantID, approval.TenantID)
		if err != nil || !inScope {
			return false, err
		}
	}
	if strings.TrimSpace(scope.WorkspaceID) != "" && approval.WorkspaceID != strings.TrimSpace(scope.WorkspaceID) {
		return false, nil
	}

	caller, ok, err := s.repo.GetAgent(ctx, approval.CallerInstanceID)
	if err != nil || !ok {
		return false, err
	}
	if caller.TenantID != approval.TenantID || caller.WorkspaceID != approval.WorkspaceID {
		return false, nil
	}
	callerInScope, err := s.permissionPackageApplicationAgentInScope(ctx, caller, scope)
	if err != nil || !callerInScope {
		return false, err
	}

	target, ok, err := s.repo.GetAgent(ctx, approval.TargetID)
	if err != nil || !ok {
		return false, err
	}
	allowedTenant, err := s.tenantCanReceiveTargetEntitlement(ctx, target.TenantID, approval.TenantID)
	if err != nil || !allowedTenant {
		return false, err
	}
	if approval.WorkspaceID != "" && target.WorkspaceID != approval.WorkspaceID {
		return false, nil
	}
	return true, nil
}

func (s *Server) permissionPackageApprovalAuditMetadataForVisibleRequest(ctx context.Context, approval domain.PermissionPackageApprovalRequest) (map[string]any, error) {
	allowedCapabilityIDs := []string{}
	for _, id := range approval.AllowedCapabilityIDs {
		capability, ok, err := s.repo.GetCapability(ctx, id)
		if err != nil {
			return nil, err
		}
		if !ok || capability.TargetID != approval.TargetID {
			continue
		}
		allowedCapabilityIDs = append(allowedCapabilityIDs, capability.ID)
	}
	requestedCapabilityID := approval.RequestedCapabilityID
	if requestedCapabilityID != "" && !stringSliceContains(allowedCapabilityIDs, requestedCapabilityID) {
		requestedCapabilityID = ""
	}
	return map[string]any{
		"approvalRequestId":       approval.ID,
		"draftId":                 approval.DraftID,
		"templateId":              approval.TemplateID,
		"templateVersion":         approval.TemplateVersion,
		"policyVersion":           approval.PolicyVersion,
		"targetId":                approval.TargetID,
		"callerInstanceId":        approval.CallerInstanceID,
		"requestedCapabilityId":   requestedCapabilityID,
		"status":                  approval.Status,
		"requestedBy":             approval.RequestedBy,
		"reviewedBy":              approval.ReviewedBy,
		"reasonCount":             len(approval.PolicyGate.Reasons),
		"allowedCapabilityIds":    auditMetadataStrings(allowedCapabilityIDs),
		"expiresAt":               approval.ExpiresAt,
		"consumedAt":              approval.ConsumedAt,
		"consumedByApplicationId": approval.ConsumedByApplicationID,
	}, nil
}

func (s *Server) visiblePermissionPackageApprovalRequests(ctx context.Context, approvals []domain.PermissionPackageApprovalRequest, scope store.ManagementScope) ([]domain.PermissionPackageApprovalRequest, error) {
	if len(approvals) == 0 {
		return []domain.PermissionPackageApprovalRequest{}, nil
	}
	rows := make([]domain.PermissionPackageApprovalRequest, 0, len(approvals))
	for _, approval := range approvals {
		visible, err := s.permissionPackageApprovalRequestListVisible(ctx, approval, scope)
		if err != nil {
			return nil, err
		}
		if !visible {
			continue
		}
		sanitized, err := s.permissionPackageApprovalRequestWithVisibleCapabilities(ctx, approval)
		if err != nil {
			return nil, err
		}
		rows = append(rows, sanitized)
	}
	return rows, nil
}

func (s *Server) permissionPackageApprovalRequestListVisible(ctx context.Context, approval domain.PermissionPackageApprovalRequest, scope store.ManagementScope) (bool, error) {
	if strings.TrimSpace(scope.TenantID) != "" {
		inScope, err := s.tenantCanReceiveTargetEntitlement(ctx, scope.TenantID, approval.TenantID)
		if err != nil || !inScope {
			return false, err
		}
	}
	if strings.TrimSpace(scope.WorkspaceID) != "" && approval.WorkspaceID != strings.TrimSpace(scope.WorkspaceID) {
		return false, nil
	}

	caller, ok, err := s.repo.GetAgent(ctx, approval.CallerInstanceID)
	if err != nil {
		return false, err
	}
	if ok {
		if caller.TenantID != approval.TenantID || caller.WorkspaceID != approval.WorkspaceID {
			return false, nil
		}
		callerInScope, err := s.permissionPackageApplicationAgentInScope(ctx, caller, scope)
		if err != nil || !callerInScope {
			return false, err
		}
	}

	target, ok, err := s.repo.GetAgent(ctx, approval.TargetID)
	if err != nil {
		return false, err
	}
	if ok {
		allowedTenant, err := s.tenantCanReceiveTargetEntitlement(ctx, target.TenantID, approval.TenantID)
		if err != nil || !allowedTenant {
			return false, err
		}
		if approval.WorkspaceID != "" && target.WorkspaceID != approval.WorkspaceID {
			return false, nil
		}
	}
	return true, nil
}

func (s *Server) permissionPackageApprovalRequestWithVisibleCapabilities(ctx context.Context, approval domain.PermissionPackageApprovalRequest) (domain.PermissionPackageApprovalRequest, error) {
	visibleCapabilities := []domain.Capability{}
	for _, id := range approval.AllowedCapabilityIDs {
		capability, ok, err := s.repo.GetCapability(ctx, id)
		if err != nil {
			return domain.PermissionPackageApprovalRequest{}, err
		}
		if !ok || capability.TargetID != approval.TargetID {
			continue
		}
		visibleCapabilities = append(visibleCapabilities, capability)
	}
	allowedCapabilityIDs, allowedCapabilityKeys := permissionPackageCapabilityIDsAndKeys(visibleCapabilities)
	approval.AllowedCapabilityIDs = allowedCapabilityIDs
	approval.AllowedCapabilityKeys = allowedCapabilityKeys
	approval.AllowedCapabilityFingerprints = permissionPackageCapabilityFingerprints(visibleCapabilities)
	approval.PolicyGate = permissionPackagePolicyGateWithVisibleCapabilities(approval.PolicyGate, visibleCapabilities)
	if approval.RequestedCapabilityID != "" && !stringSliceContains(allowedCapabilityIDs, approval.RequestedCapabilityID) {
		approval.RequestedCapabilityID = ""
	}
	return approval, nil
}

func permissionPackagePolicyGateWithVisibleCapabilities(gate domain.PermissionPackagePolicyGate, capabilities []domain.Capability) domain.PermissionPackagePolicyGate {
	visibleByID := make(map[string]domain.Capability, len(capabilities))
	for _, capability := range capabilities {
		visibleByID[capability.ID] = capability
	}
	gate.NextActions = append([]string(nil), gate.NextActions...)
	gate.Reasons = make([]domain.PermissionPackagePolicyReason, 0, len(gate.Reasons))
	for _, reason := range gate.Reasons {
		reason.ReasonValues = cloneStringMap(reason.ReasonValues)
		if strings.TrimSpace(reason.CapabilityID) == "" {
			gate.Reasons = append(gate.Reasons, reason)
			continue
		}
		capability, ok := visibleByID[reason.CapabilityID]
		if !ok {
			continue
		}
		reason.CapabilityID = capability.ID
		reason.CapabilityKey = capability.Key
		gate.Reasons = append(gate.Reasons, reason)
	}
	return gate
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func auditMetadataStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}
