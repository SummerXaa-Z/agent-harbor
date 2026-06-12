export type AccessSubjectKind = "role" | "department" | "member" | "custom";

export interface AccessSubjectOption {
  id: string;
  kind: AccessSubjectKind;
  labelKey: string;
  detailKey: string;
  subjectSelector: string;
  email?: string;
  status?: string;
  tenantId?: string;
  workspaceId?: string;
}

export const customAccessSubjectOption: AccessSubjectOption = {
  detailKey: "accessSubject.custom.detail",
  id: "custom",
  kind: "custom",
  labelKey: "accessSubject.custom.name",
  subjectSelector: ""
};

export const accessSubjectOptions: AccessSubjectOption[] = [
  {
    detailKey: "accessSubject.supportAgent.detail",
    id: "role:support-agent",
    kind: "role",
    labelKey: "accessSubject.supportAgent.name",
    subjectSelector: "user:support-*"
  },
  {
    detailKey: "accessSubject.supportLead.detail",
    id: "role:support-lead",
    kind: "role",
    labelKey: "accessSubject.supportLead.name",
    subjectSelector: "user:support-lead-*"
  },
  {
    detailKey: "accessSubject.securityReviewer.detail",
    id: "role:security-reviewer",
    kind: "role",
    labelKey: "accessSubject.securityReviewer.name",
    subjectSelector: "user:security-*"
  },
  {
    detailKey: "accessSubject.customerService.detail",
    id: "department:customer-service",
    kind: "department",
    labelKey: "accessSubject.customerService.name",
    subjectSelector: "user:cs-*"
  },
  {
    detailKey: "accessSubject.support001.detail",
    id: "member:support-001",
    kind: "member",
    labelKey: "accessSubject.support001.name",
    subjectSelector: "user:support-001",
    workspaceId: "ws-permission-package-approval",
    email: "support001@example.com",
    status: "active"
  },
  {
    detailKey: "accessSubject.support002.detail",
    id: "member:support-002",
    kind: "member",
    labelKey: "accessSubject.support002.name",
    subjectSelector: "user:support-002",
    workspaceId: "ws-permission-package-approval",
    email: "support002@example.com",
    status: "active"
  },
  {
    detailKey: "accessSubject.supportLead001.detail",
    id: "member:support-lead-001",
    kind: "member",
    labelKey: "accessSubject.supportLead001.name",
    subjectSelector: "user:support-lead-001",
    workspaceId: "ws-permission-package-approval",
    email: "support-lead001@example.com",
    status: "active"
  },
  {
    detailKey: "accessSubject.securityReviewer001.detail",
    id: "member:security-reviewer-001",
    kind: "member",
    labelKey: "accessSubject.securityReviewer001.name",
    subjectSelector: "user:security-reviewer-001",
    workspaceId: "ws-permission-package-approval",
    email: "security-reviewer001@example.com",
    status: "active"
  }
];

export function accessSubjectOptionForSelector(subjectSelector?: string): AccessSubjectOption {
  const selector = subjectSelector?.trim() ?? "";
  return accessSubjectOptions.find((option) => option.subjectSelector === selector) ?? {
    ...customAccessSubjectOption,
    subjectSelector: selector
  };
}

export function accessSubjectOptionForId(id: string): AccessSubjectOption | undefined {
  if (id === customAccessSubjectOption.id) return customAccessSubjectOption;
  return accessSubjectOptions.find((option) => option.id === id);
}

export function accessSubjectOptionForSelectorFrom(
  options: AccessSubjectOption[],
  subjectSelector?: string
): AccessSubjectOption {
  const selector = subjectSelector?.trim() ?? "";
  return options.find((option) => option.subjectSelector === selector) ?? {
    ...customAccessSubjectOption,
    subjectSelector: selector
  };
}

export function accessSubjectOptionForIdFrom(options: AccessSubjectOption[], id: string): AccessSubjectOption | undefined {
  if (id === customAccessSubjectOption.id) return customAccessSubjectOption;
  return options.find((option) => option.id === id);
}

export function normalizeAccessSubjectOptions(options: AccessSubjectOption[] | undefined): AccessSubjectOption[] {
  const normalized = (options ?? []).filter((option) =>
    option.id.trim()
    && option.subjectSelector.trim()
    && ["role", "department", "member"].includes(option.kind)
  );
  return normalized.length > 0 ? normalized : accessSubjectOptions;
}

export function accessSubjectsForWorkspace(
  options: AccessSubjectOption[] | undefined,
  workspaceId: string
): AccessSubjectOption[] {
  const normalized = normalizeAccessSubjectOptions(options);
  const scopedWorkspaceId = workspaceId.trim();
  if (!scopedWorkspaceId) return normalized;

  const scoped = normalized.filter((option) => !option.workspaceId || option.workspaceId === scopedWorkspaceId);
  return scoped.length > 0 ? scoped : normalized;
}
