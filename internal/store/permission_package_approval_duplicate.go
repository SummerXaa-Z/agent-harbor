package store

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

func permissionPackageApprovalRequestDuplicateReferenceTime(request domain.PermissionPackageApprovalRequest) time.Time {
	if !request.CreatedAt.IsZero() {
		return request.CreatedAt
	}
	if !request.UpdatedAt.IsZero() {
		return request.UpdatedAt
	}
	return time.Now().UTC()
}

func permissionPackageApprovalRequestActivePendingAt(request domain.PermissionPackageApprovalRequest, now time.Time) bool {
	if request.Status != domain.PermissionPackageApprovalStatusPending {
		return false
	}
	return request.ExpiresAt.IsZero() || now.Before(request.ExpiresAt)
}

func permissionPackageApprovalRequestCanTransition(request domain.PermissionPackageApprovalRequest, now time.Time) bool {
	return permissionPackageApprovalRequestActivePendingAt(request, now) &&
		request.ConsumedAt.IsZero() &&
		request.ConsumedByApplicationID == ""
}

func permissionPackageApprovalRequestsShareDuplicateKey(left domain.PermissionPackageApprovalRequest, right domain.PermissionPackageApprovalRequest) bool {
	return left.DraftID == right.DraftID &&
		left.TemplateID == right.TemplateID &&
		left.TemplateVersion == right.TemplateVersion &&
		left.PolicyVersion == right.PolicyVersion &&
		left.TenantID == right.TenantID &&
		left.WorkspaceID == right.WorkspaceID &&
		left.TargetID == right.TargetID &&
		left.CallerInstanceID == right.CallerInstanceID &&
		left.RequestedCapabilityID == right.RequestedCapabilityID &&
		left.SubjectSelector == right.SubjectSelector &&
		left.RequestText == right.RequestText &&
		left.Region == right.Region &&
		samePermissionPackageApprovalDataScopes(left.DataScopes, right.DataScopes) &&
		samePermissionPackageApprovalStringSet(left.AllowedCapabilityIDs, right.AllowedCapabilityIDs) &&
		samePermissionPackageApprovalStringSet(left.AllowedCapabilityKeys, right.AllowedCapabilityKeys) &&
		samePermissionPackageApprovalStringSet(left.AllowedCapabilityFingerprints, right.AllowedCapabilityFingerprints)
}

func permissionPackageApprovalRequestDuplicateLockKey(request domain.PermissionPackageApprovalRequest) string {
	payload := struct {
		DraftID                       string             `json:"draftId"`
		TemplateID                    string             `json:"templateId"`
		TemplateVersion               int                `json:"templateVersion"`
		PolicyVersion                 int                `json:"policyVersion"`
		TenantID                      string             `json:"tenantId"`
		WorkspaceID                   string             `json:"workspaceId"`
		TargetID                      string             `json:"targetId"`
		CallerInstanceID              string             `json:"callerInstanceId"`
		RequestedCapabilityID         string             `json:"requestedCapabilityId"`
		SubjectSelector               string             `json:"subjectSelector"`
		RequestText                   string             `json:"requestText"`
		Region                        string             `json:"region"`
		DataScopes                    []domain.DataScope `json:"dataScopes"`
		AllowedCapabilityIDs          []string           `json:"allowedCapabilityIds"`
		AllowedCapabilityKeys         []string           `json:"allowedCapabilityKeys"`
		AllowedCapabilityFingerprints []string           `json:"allowedCapabilityFingerprints"`
	}{
		DraftID:                       request.DraftID,
		TemplateID:                    request.TemplateID,
		TemplateVersion:               request.TemplateVersion,
		PolicyVersion:                 request.PolicyVersion,
		TenantID:                      request.TenantID,
		WorkspaceID:                   request.WorkspaceID,
		TargetID:                      request.TargetID,
		CallerInstanceID:              request.CallerInstanceID,
		RequestedCapabilityID:         request.RequestedCapabilityID,
		SubjectSelector:               request.SubjectSelector,
		RequestText:                   request.RequestText,
		Region:                        request.Region,
		DataScopes:                    sortedPermissionPackageApprovalDataScopes(request.DataScopes),
		AllowedCapabilityIDs:          sortedPermissionPackageApprovalStrings(request.AllowedCapabilityIDs),
		AllowedCapabilityKeys:         sortedPermissionPackageApprovalStrings(request.AllowedCapabilityKeys),
		AllowedCapabilityFingerprints: sortedPermissionPackageApprovalStrings(request.AllowedCapabilityFingerprints),
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func samePermissionPackageApprovalDataScopes(left []domain.DataScope, right []domain.DataScope) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := sortedPermissionPackageApprovalDataScopes(left)
	rightCopy := sortedPermissionPackageApprovalDataScopes(right)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func sortedPermissionPackageApprovalDataScopes(values []domain.DataScope) []domain.DataScope {
	out := append([]domain.DataScope(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		return permissionPackageApprovalDataScopeSortKey(out[i]) < permissionPackageApprovalDataScopeSortKey(out[j])
	})
	return out
}

func permissionPackageApprovalDataScopeSortKey(scope domain.DataScope) string {
	data, _ := json.Marshal(scope)
	return string(data)
}

func samePermissionPackageApprovalStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := sortedPermissionPackageApprovalStrings(left)
	rightCopy := sortedPermissionPackageApprovalStrings(right)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func sortedPermissionPackageApprovalStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
