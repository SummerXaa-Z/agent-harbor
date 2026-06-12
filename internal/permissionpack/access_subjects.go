package permissionpack

import "github.com/SummerXaa-Z/agent-harbor/internal/domain"

var accessSubjects = []domain.PermissionPackageAccessSubject{
	{
		ID:              "role:support-agent",
		Kind:            "role",
		LabelKey:        "accessSubject.supportAgent.name",
		DetailKey:       "accessSubject.supportAgent.detail",
		SubjectSelector: "user:support-*",
	},
	{
		ID:              "role:support-lead",
		Kind:            "role",
		LabelKey:        "accessSubject.supportLead.name",
		DetailKey:       "accessSubject.supportLead.detail",
		SubjectSelector: "user:support-lead-*",
	},
	{
		ID:              "role:security-reviewer",
		Kind:            "role",
		LabelKey:        "accessSubject.securityReviewer.name",
		DetailKey:       "accessSubject.securityReviewer.detail",
		SubjectSelector: "user:security-*",
	},
	{
		ID:              "department:customer-service",
		Kind:            "department",
		LabelKey:        "accessSubject.customerService.name",
		DetailKey:       "accessSubject.customerService.detail",
		SubjectSelector: "user:cs-*",
	},
	{
		ID:              "member:support-001",
		Kind:            "member",
		LabelKey:        "accessSubject.support001.name",
		DetailKey:       "accessSubject.support001.detail",
		SubjectSelector: "user:support-001",
	},
}

func AccessSubjects() []domain.PermissionPackageAccessSubject {
	next := make([]domain.PermissionPackageAccessSubject, len(accessSubjects))
	copy(next, accessSubjects)
	return next
}
