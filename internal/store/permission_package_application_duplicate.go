package store

import (
	"encoding/json"
	"sort"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

type permissionPackageApplicationDuplicatePayload struct {
	DraftID               string             `json:"draftId"`
	TemplateID            string             `json:"templateId"`
	TemplateVersion       int                `json:"templateVersion"`
	TenantID              string             `json:"tenantId"`
	WorkspaceID           string             `json:"workspaceId"`
	TargetID              string             `json:"targetId"`
	CallerInstanceID      string             `json:"callerInstanceId"`
	SubjectSelector       string             `json:"subjectSelector"`
	Region                string             `json:"region"`
	DataScopes            []domain.DataScope `json:"dataScopes"`
	AllowedCapabilityIDs  []string           `json:"allowedCapabilityIds"`
	AllowedCapabilityKeys []string           `json:"allowedCapabilityKeys"`
}

func permissionPackageApplicationsShareDuplicateKey(left domain.PermissionPackageApplication, right domain.PermissionPackageApplication) bool {
	return PermissionPackageApplicationsShareDuplicateKey(left, right)
}

// PermissionPackageApplicationsShareDuplicateKey reports whether two application records
// represent the same semantic permission-package apply request.
func PermissionPackageApplicationsShareDuplicateKey(left domain.PermissionPackageApplication, right domain.PermissionPackageApplication) bool {
	return left.DraftID == right.DraftID &&
		left.TemplateID == right.TemplateID &&
		left.TemplateVersion == right.TemplateVersion &&
		left.TenantID == right.TenantID &&
		left.WorkspaceID == right.WorkspaceID &&
		left.TargetID == right.TargetID &&
		left.CallerInstanceID == right.CallerInstanceID &&
		left.SubjectSelector == right.SubjectSelector &&
		left.Region == right.Region &&
		samePermissionPackageApplicationDataScopes(left.DataScopes, right.DataScopes) &&
		samePermissionPackageApplicationStringSet(left.AllowedCapabilityIDs, right.AllowedCapabilityIDs) &&
		samePermissionPackageApplicationStringSet(left.AllowedCapabilityKeys, right.AllowedCapabilityKeys)
}

func permissionPackageApplicationDuplicateLockKey(application domain.PermissionPackageApplication) string {
	payload := permissionPackageApplicationDuplicatePayload{
		DraftID:               application.DraftID,
		TemplateID:            application.TemplateID,
		TemplateVersion:       application.TemplateVersion,
		TenantID:              application.TenantID,
		WorkspaceID:           application.WorkspaceID,
		TargetID:              application.TargetID,
		CallerInstanceID:      application.CallerInstanceID,
		SubjectSelector:       application.SubjectSelector,
		Region:                application.Region,
		DataScopes:            sortedPermissionPackageApplicationDataScopes(application.DataScopes),
		AllowedCapabilityIDs:  sortedPermissionPackageApplicationStrings(application.AllowedCapabilityIDs),
		AllowedCapabilityKeys: sortedPermissionPackageApplicationStrings(application.AllowedCapabilityKeys),
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func samePermissionPackageApplicationDataScopes(left []domain.DataScope, right []domain.DataScope) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := sortedPermissionPackageApplicationDataScopes(left)
	rightCopy := sortedPermissionPackageApplicationDataScopes(right)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func sortedPermissionPackageApplicationDataScopes(values []domain.DataScope) []domain.DataScope {
	out := append([]domain.DataScope(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		left, _ := json.Marshal(out[i])
		right, _ := json.Marshal(out[j])
		return string(left) < string(right)
	})
	return out
}

func samePermissionPackageApplicationStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := sortedPermissionPackageApplicationStrings(left)
	rightCopy := sortedPermissionPackageApplicationStrings(right)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func sortedPermissionPackageApplicationStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
