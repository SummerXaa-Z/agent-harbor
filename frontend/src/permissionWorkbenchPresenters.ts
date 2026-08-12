import type { AccessSubjectOption } from "./accessSubjects";
import type { AiAdminProductionConsoleStatus } from "./aiAdminProductionConsole";
import type { MetricTone } from "./consoleMetrics";
import { capabilityKeyDisplayName } from "./consolePresenters";
import { createTranslator } from "./i18n";
import type { PermissionRequestWizardStep } from "./permissionRequestJourney";
import type {
  PermissionPackageApplyPreflight,
  PermissionPackageApplyPreflightCheck,
  PermissionPackageApplyPreflightNextActionCode,
  PermissionPackageApplicationHealthRow,
  PermissionPackageApplicationHealthStatus,
  PermissionPackageApprovalEffectiveStatus,
  PermissionPackageApprovalRequest,
  PermissionPackageDraft,
  PermissionPackagePolicyNextActionCode,
  PermissionPackageProductionReadinessCheck,
  PermissionPackageProductionReadinessStatus,
  PermissionPackageTemplate,
  PermissionPackageWorkbenchPreview,
  PermissionPackageWorkbenchStepKey,
  PermissionPackageWorkbenchStepStatus
} from "./permissionPackages";
import type {
  AccessDecisionExplainResult,
  Agent,
  Tenant
} from "./types";

export type Tone = MetricTone;
export type Translator = ReturnType<typeof createTranslator>;
export type PermissionRequestStepTarget = PermissionRequestWizardStep | "validation" | "acceptance";

const defaultWorkspaceId = "workspace-sandbox";

export function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}

function txKnown(t: Translator, key: string, values: Record<string, string | number>, fallbackKey: string) {
  const template = t(key);
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    template === key ? t(fallbackKey) : template
  );
}

function knownLabel(t: Translator, key: string, fallbackKey: string) {
  const value = t(key);
  return value === key ? t(fallbackKey) : value;
}

export function resourcePermissionIntent(targetName: string, t: Translator) {
  return tx(t, "resource.permissionIntent", { target: targetName });
}

export function accessSubjectDropdownOption(option: AccessSubjectOption, t: Translator) {
  return {
    label: `${t(accessSubjectKindLabelKey(option.kind))} · ${t(option.labelKey)}`,
    value: option.id
  };
}

export function accessSubjectKindLabelKey(kind: AccessSubjectOption["kind"]) {
  if (kind === "department") return "accessSubject.kind.department";
  if (kind === "member") return "accessSubject.kind.member";
  if (kind === "custom") return "accessSubject.kind.custom";
  return "accessSubject.kind.role";
}

export function accessDecisionOutcomeLabel(outcome: AccessDecisionExplainResult["outcome"], t: Translator) {
  return outcome === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied");
}

export function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  });
}

export function permissionTenantPathLabel(tenantId: string, tenants: Tenant[], t: Translator): { path: string; primary: string } {
  const normalizedTenantId = tenantId.trim();
  if (!normalizedTenantId) return { path: "-", primary: t("text.unresolvedTenant") };
  const tenantById = tenants.reduce<Record<string, Tenant>>((acc, tenant) => {
    acc[tenant.id] = tenant;
    return acc;
  }, {});
  const selected = tenantById[normalizedTenantId];
  if (!selected) {
    if (normalizedTenantId === "default") {
      const defaultTenantName = t("text.defaultTenantName");
      return { path: defaultTenantName, primary: defaultTenantName };
    }
    return {
      path: tx(t, "text.unresolvedTenantDetail", { id: normalizedTenantId }),
      primary: t("text.unresolvedTenant")
    };
  }

  const path: Tenant[] = [];
  const visited = new Set<string>();
  let current: Tenant | undefined = selected;
  while (current && !visited.has(current.id)) {
    path.unshift(current);
    visited.add(current.id);
    current = current.parentTenantId ? tenantById[current.parentTenantId] : undefined;
  }
  const names = path.map((tenant) => permissionEntityDisplayName(tenant.name.trim() || tenant.id, t));
  return {
    path: names.join(" > "),
    primary: permissionEntityDisplayName(selected.name.trim() || selected.id, t)
  };
}

export function permissionWorkspaceDisplayName(workspaceId: string, agents: Agent[], t: Translator) {
  const normalizedWorkspaceId = workspaceId.trim();
  if (!normalizedWorkspaceId) return "-";
  const agentInWorkspace = agents.find((agent) => agent.workspaceId === normalizedWorkspaceId);
  if (normalizedWorkspaceId === defaultWorkspaceId || agentInWorkspace?.workspaceId === defaultWorkspaceId) {
    return t("text.defaultWorkspaceName");
  }
  if (/permission[-_]?(request|package)[-_]?approval/i.test(normalizedWorkspaceId)) {
    return t("demo.permissionRequestWorkspace");
  }
  if (/core[-_]?journey/i.test(normalizedWorkspaceId)) {
    return t("demo.coreJourneyWorkspace");
  }
  return permissionEntityDisplayName(readableIdentifierLabel(normalizedWorkspaceId), t);
}

export function uniquePermissionEntityOptions<T extends { id: string }>(
  entities: T[],
  selectedId: string,
  labelFor: (entity: T) => string
): Array<{ id: string; label: string }> {
  const selected = entities.find((entity) => entity.id === selectedId);
  const ordered = selected ? [selected, ...entities.filter((entity) => entity.id !== selected.id)] : entities;
  const seen = new Set<string>();
  const options: Array<{ id: string; label: string }> = [];

  for (const entity of ordered) {
    const label = labelFor(entity).trim() || entity.id;
    const key = label.toLocaleLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    options.push({ id: entity.id, label });
  }

  return options;
}

export function readableIdentifierLabel(value: string) {
  const normalized = value.trim().replace(/^ws[-_]/, "").replace(/^workspace[-_]/, "");
  if (!normalized) return value;
  return normalized
    .split(/[-_]+/)
    .filter(Boolean)
    .map((part) => {
      const upper = part.toUpperCase();
      if (["AI", "API", "CRM", "MCP", "UI"].includes(upper)) return upper;
      return `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`;
    })
    .join(" ");
}

export function permissionEntityDisplayName(value: string, t: Translator) {
  const normalized = value.trim();
  if (!normalized) return value;
  const demoNames: Record<string, string> = {
    "Core Journey Caller": t("demo.coreJourneyCaller"),
    "Core Journey MCP Target": t("demo.coreJourneyTarget"),
    "Core Journey Project": t("demo.coreJourneyProject"),
    "Core Journey Root": t("demo.coreJourneyRoot"),
    "Core Journey Team": t("demo.coreJourneyTeam"),
    "Permission Package Approval": t("demo.permissionRequestApproval"),
    "Permission Package Approval Root": t("demo.permissionRequestApprovalRoot"),
    "Permission Package Approval Team": t("demo.permissionRequestApprovalTeam"),
    "Permission Package Approval Project": t("demo.permissionRequestApprovalProject"),
    "Permission Package Approval Caller": t("demo.permissionRequestApprovalCaller"),
    "Permission Package Approval MCP Target": t("demo.permissionRequestApprovalTarget"),
    "Permission Request Approval": t("demo.permissionRequestApproval"),
    "Permission Request Approval Root": t("demo.permissionRequestApprovalRoot"),
    "Permission Request Approval Team": t("demo.permissionRequestApprovalTeam"),
    "Permission Request Approval Project": t("demo.permissionRequestApprovalProject"),
    "Permission Request Approval Caller": t("demo.permissionRequestApprovalCaller"),
    "Permission Request Approval MCP Target": t("demo.permissionRequestApprovalTarget"),
    "Security Reviewer": t("accessSubject.securityReviewer.name")
  };
  if (demoNames[normalized]) return demoNames[normalized];
  if (normalized.startsWith("MCP Capability Caller")) return t("demo.mcpCapabilityCaller");
  if (normalized.startsWith("MCP Capability MCP Target")) return t("demo.mcpCapabilityTarget");
  if (normalized.includes("Permission Package Approval")) {
    return normalized.replaceAll("Permission Package Approval", t("demo.permissionRequestApproval"));
  }
  return normalized;
}

export function permissionPackageTemplateName(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.name`, template.name);
}

export function permissionPackageTemplateNameById(templateId: string, t: Translator) {
  return t(`permissionPackage.${templateId}.name`, templateId);
}

export function permissionApprovalRequestBusinessLabel(
  request: PermissionPackageApprovalRequest,
  templates: PermissionPackageTemplate[],
  tenants: Tenant[],
  agents: Agent[],
  t: Translator
) {
  const template = templates.find((item) => item.id === request.templateId);
  const tenant = tenants.find((item) => item.id === request.tenantId);
  const caller = agents.find((item) => item.id === request.callerInstanceId);
  const target = agents.find((item) => item.id === request.targetId);
  return {
    caller: caller ? permissionEntityDisplayName(caller.name, t) : t("text.unknownCaller"),
    target: target ? permissionEntityDisplayName(target.name, t) : t("text.unknownTarget"),
    template: template ? permissionPackageTemplateName(template, t) : t("text.unknownPermissionPackage"),
    tenant: tenant ? permissionEntityDisplayName(tenant.name, t) : t("text.unknownTenant")
  };
}

export function permissionPackageTemplateSummary(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.summary`, template.summary);
}

export function permissionRequestStepSectionId(step: PermissionRequestStepTarget) {
  return `permission-request-step-${step}`;
}

export function permissionRequestStepTarget(step: PermissionPackageWorkbenchStepKey | PermissionRequestWizardStep): PermissionRequestStepTarget {
  if (step === "request") return "scope";
  if (step === "validation") return "validation";
  if (step === "acceptance") return "acceptance";
  return step;
}

export function resolvePermissionJourneyStatus(args: {
  approvalRequest: PermissionPackageApprovalRequest | null;
  canApply: boolean;
  draft: PermissionPackageDraft;
  goLiveReady: boolean;
  productionStatus: AiAdminProductionConsoleStatus;
  workbenchStatus?: PermissionPackageWorkbenchPreview["summary"]["status"];
}): { labelKey: string; detailKey: string; tone: Tone; nextActionKey: string } {
  const approvalStatus = args.approvalRequest ? approvalEffectiveStatus(args.approvalRequest) : null;
  if (args.goLiveReady) {
    return {
      detailKey: "permissionJourney.statusDetail.ready",
      labelKey: "permissionJourney.status.ready",
      nextActionKey: "action.exportAcceptanceReport",
      tone: "success"
    };
  }
  if (approvalStatus === "pending") {
    return {
      detailKey: "permissionJourney.statusDetail.awaitingApproval",
      labelKey: "permissionJourney.status.awaitingApproval",
      nextActionKey: "action.refreshReviewerQueue",
      tone: "warning"
    };
  }
  if (approvalStatus === "rejected") {
    return {
      detailKey: "permissionJourney.statusDetail.rejected",
      labelKey: "permissionJourney.status.rejected",
      nextActionKey: "action.startPermissionApproval",
      tone: "danger"
    };
  }
  if (approvalStatus === "expired") {
    return {
      detailKey: "permissionJourney.statusDetail.needsApproval",
      labelKey: "permissionJourney.status.needsApproval",
      nextActionKey: "action.startPermissionApproval",
      tone: "warning"
    };
  }
  if (args.canApply) {
    return {
      detailKey: "permissionJourney.statusDetail.readyToApply",
      labelKey: "permissionJourney.status.readyToApply",
      nextActionKey: "action.applyPermissionPackage",
      tone: "info"
    };
  }
  if (args.draft.readiness.missingFields.length > 0 || args.workbenchStatus === "needs_input") {
    return {
      detailKey: "permissionJourney.statusDetail.needsInput",
      labelKey: "permissionJourney.status.needsInput",
      nextActionKey: "action.completePermissionRequest",
      tone: "warning"
    };
  }
  if (args.workbenchStatus === "blocked" || args.productionStatus === "blocked") {
    return {
      detailKey: "permissionJourney.statusDetail.blocked",
      labelKey: "permissionJourney.status.blocked",
      nextActionKey: "action.checkPreflight",
      tone: "danger"
    };
  }
  return {
    detailKey: "permissionJourney.statusDetail.needsApproval",
    labelKey: "permissionJourney.status.needsApproval",
    nextActionKey: "action.createApprovalRequest",
    tone: "warning"
  };
}

function approvalEffectiveStatus(request: PermissionPackageApprovalRequest): PermissionPackageApprovalEffectiveStatus {
  return request.effectiveStatus ?? request.status;
}

export function permissionDraftStatus(draft: PermissionPackageDraft): { labelKey: string; tone: Tone } {
  if (!draft.readiness.canApply) {
    return { labelKey: "status.needsReview", tone: "warning" };
  }
  if (!draft.policyGate.canApplyDirectly) {
    return { labelKey: "status.approvalPending", tone: "warning" };
  }
  return { labelKey: "status.readyToApply", tone: "success" };
}

export function permissionPolicyGateDetailKey(
  canApplyDirectly: boolean,
  approvalStatus: PermissionPackageApprovalEffectiveStatus | null,
) {
  if (canApplyDirectly) return "text.policyGateDirectDetail";
  if (approvalStatus === "approved") return "text.policyGateApprovedDetail";
  if (approvalStatus === "rejected") return "text.policyGateRejectedDetail";
  if (approvalStatus === "withdrawn") return "text.policyGateWithdrawnDetail";
  if (approvalStatus === "expired") return "text.policyGateApprovalDetail";
  return "text.policyGateApprovalDetail";
}

export function permissionPolicyGateNextActionByCode(code: string, t: Translator) {
  const keyByCode: Record<PermissionPackagePolicyNextActionCode, string> = {
    create_approval_request: "permissionPolicy.next.createApproval"
  };
  return Object.prototype.hasOwnProperty.call(keyByCode, code)
    ? t(keyByCode[code as PermissionPackagePolicyNextActionCode])
    : t("permissionPolicy.next.unknown");
}

export function permissionPolicyGateMessages(policyGate: PermissionPackageDraft["policyGate"], t: Translator) {
  if (policyGate.canApplyDirectly) {
    return [t("text.policyGateDirectDetail")];
  }
  const nextActionCode = policyGate.nextActionCodes?.[0];
  const messages = policyGate.reasons.length > 0
    ? policyGate.reasons.map((reason) => permissionPolicyReasonMessage(reason, t))
    : [t("text.policyGateApprovalDetail")];
  return nextActionCode
    ? [...messages, permissionPolicyGateNextActionByCode(nextActionCode, t)]
    : messages;
}

export function permissionApprovalStatusLabel(status: PermissionPackageApprovalEffectiveStatus, t: Translator) {
  if (status === "approved") return t("status.approvalApproved");
  if (status === "rejected") return t("status.approvalRejected");
  if (status === "withdrawn") return t("status.approvalWithdrawn");
  if (status === "expired") return t("status.approvalExpired");
  return t("status.approvalPending");
}

export function permissionApprovalStatusTone(status: PermissionPackageApprovalEffectiveStatus): Tone {
  if (status === "approved") return "success";
  if (status === "rejected") return "danger";
  if (status === "withdrawn") return "warning";
  if (status === "expired") return "warning";
  return "warning";
}

export function permissionPolicyReasonMessage(
  reason: PermissionPackageDraft["policyGate"]["reasons"][number],
  t: Translator,
) {
  if (!reason.reasonKey) return t("permissionPolicy.unknownReason");
  const values = Object.entries(reason.reasonValues ?? {}).reduce<Record<string, string>>((acc, [key, value]) => {
    if (key === "capability") {
      acc[key] = capabilityKeyDisplayName(value, t);
    } else if (key === "action" || key === "risk" || key === "sensitivity") {
      acc[key] = translatedValue(t, value);
    } else {
      acc[key] = value;
    }
    return acc;
  }, {});
  return txKnown(t, reason.reasonKey, values, "permissionPolicy.unknownReason");
}

export function permissionReadinessMessages(readiness: PermissionPackageDraft["readiness"], t: Translator) {
  const fieldLabels: Record<string, string> = {
    callerInstanceId: t("form.caller"),
    subjectSelector: t("form.accessSubject"),
    templateId: t("form.permissionPackage"),
    targetId: t("form.target"),
    tenantId: t("form.tenant"),
    workspaceId: t("form.workspace")
  };
  const warningLabels: Record<string, string> = {
    "No matching allowed capabilities for the selected target.": t("message.noMatchingAllowedCapabilities"),
    "The requested capability is not safely covered by this permission package.": t("message.requestedCapabilityNotCovered")
  };
  return [
    ...readiness.missingFields.map((field) =>
      tx(t, "message.permissionPackageMissingField", { field: fieldLabels[field] ?? t("form.requiredFieldFallback") })
    ),
    ...readiness.warnings.map((warning) => warningLabels[warning] ?? t("message.permissionPackageReadinessWarning"))
  ];
}

export function translatedValue(t: Translator, value: string) {
  return t(`value.${value}`, value);
}

export function permissionInlineMessageTone(message: string): "error" | "info" | "success" | "warning" {
  const normalized = message.trim().toLowerCase();
  if (!normalized) return "info";
  if (
    normalized.includes("unable") ||
    normalized.includes("failed") ||
    normalized.includes("error") ||
    normalized.includes("requires") ||
    normalized.includes("missing") ||
    normalized.includes("blocked") ||
    normalized.includes("rejected") ||
    normalized.includes("失败") ||
    normalized.includes("错误") ||
    normalized.includes("需要") ||
    normalized.includes("缺少") ||
    normalized.includes("阻断") ||
    normalized.includes("拒绝")
  ) {
    return "error";
  }
  if (
    normalized.includes("ready") ||
    normalized.includes("loaded") ||
    normalized.includes("exported") ||
    normalized.includes("applied") ||
    normalized.includes("approved") ||
    normalized.includes("已") ||
    normalized.includes("可上线") ||
    normalized.includes("完成")
  ) {
    return "success";
  }
  if (normalized.includes("pending") || normalized.includes("waiting") || normalized.includes("待")) {
    return "warning";
  }
  return "info";
}

export function shouldShowAdvancedStatusMessage(tone: ReturnType<typeof permissionInlineMessageTone>) {
  return tone === "error" || tone === "warning";
}

export function permissionApplicationHealthLabel(status: PermissionPackageApplicationHealthStatus, t: Translator) {
  if (status === "ready") return t("status.applicationHealthReady");
  if (status === "drifted") return t("status.applicationHealthDrifted");
  return t("status.applicationHealthNeedsReview");
}

export function permissionApplicationHealthRowSummary(row: PermissionPackageApplicationHealthRow, t: Translator) {
  if (row.status === "ready") return t("text.applicationHealthReadyDetail");
  if (row.status === "drifted") return t("text.applicationHealthDriftedDetail");
  return t("text.applicationHealthNeedsReviewDetail");
}

export function productionReadinessStatusLabel(status: PermissionPackageProductionReadinessStatus | undefined, t: Translator) {
  if (status === "ready") return t("status.productionReady");
  if (status === "needs_review") return t("status.productionNeedsReview");
  if (status === "blocked") return t("status.productionBlocked");
  return t("status.preflightPending");
}

export function permissionProductionReadinessCheckLabel(code: string, t: Translator) {
  return knownLabel(t, `productionCheck.${code}`, "productionCheck.unknown");
}

export function permissionProductionReadinessCheckMessage(check: PermissionPackageProductionReadinessCheck, t: Translator) {
  return knownLabel(t, `productionCheck.detail.${check.code}`, "productionCheck.detail.unknown");
}

export function firstBlockingApplyPreflightCheck(preflight: PermissionPackageApplyPreflight): PermissionPackageApplyPreflightCheck | null {
  return preflight.checks.find((check) => check.severity === "blocking") ?? preflight.checks[0] ?? null;
}

export function permissionApplyPreflightCheckLabel(code: string, t: Translator) {
  return knownLabel(t, `permissionPreflight.${code}`, "permissionPreflight.unknown");
}

export function permissionApplyPreflightCheckMessage(check: PermissionPackageApplyPreflightCheck | null, t: Translator) {
  if (!check) return t("message.permissionPackagePreflightNoDetail");
  return knownLabel(t, `permissionPreflight.detail.${check.code}`, "permissionPreflight.detail.unknown");
}

export function permissionApplyPreflightNextActionByCode(code: string, t: Translator) {
  const keyByCode: Record<PermissionPackageApplyPreflightNextActionCode, string> = {
    apply_permission_package: "permissionPreflight.next.applyWhenReady",
    create_approval_request: "permissionPreflight.next.createApproval",
    fix_draft_readiness: "permissionPreflight.next.fixDraft",
    narrow_data_scope: "permissionPreflight.next.narrowScope",
    refresh_approval_request: "permissionPreflight.next.refreshApproval",
    review_current_application: "permissionPreflight.next.reviewAlreadyApplied",
    review_existing_grants: "permissionPreflight.next.reviewExistingGrants",
    use_approved_request: "permissionPreflight.next.useApprovedRequest"
  };
  return Object.prototype.hasOwnProperty.call(keyByCode, code)
    ? t(keyByCode[code as PermissionPackageApplyPreflightNextActionCode])
    : t("permissionPreflight.next.unknown");
}

export function permissionApplyPreflightNextAction(action: string, t: Translator, code?: string) {
  if (code) return permissionApplyPreflightNextActionByCode(code, t);
  const keyByAction: Record<string, string> = {
    "Apply this permission package when the reviewer is ready.": "permissionPreflight.next.applyWhenReady",
    "Apply this permission request when the reviewer is ready.": "permissionPreflight.next.applyWhenReady",
    "Create and approve a permission package approval request, then preflight again with approvalRequestId.": "permissionPreflight.next.createApproval",
    "Create and approve an approval request for this permission request, then preflight again with approvalRequestId.": "permissionPreflight.next.createApproval",
    "Fix draft readiness blockers before applying this permission package.": "permissionPreflight.next.fixDraft",
    "Fix draft readiness blockers before applying this permission request.": "permissionPreflight.next.fixDraft",
    "Narrow region or data scopes so the package stays inside every capability boundary.": "permissionPreflight.next.narrowScope",
    "Refresh approval or create a new approval request for the current draft.": "permissionPreflight.next.refreshApproval",
    "Review the latest permission request status before applying the same permission request again.": "permissionPreflight.next.reviewAlreadyApplied",
    "Review existing grant chains before applying another permission package for the same caller and capability.": "permissionPreflight.next.reviewExistingGrants",
    "Review existing grant chains before applying another permission request for the same caller and capability.": "permissionPreflight.next.reviewExistingGrants",
    "Use an approved approvalRequestId that matches the current draft.": "permissionPreflight.next.useApprovedRequest"
  };
  return keyByAction[action] ? t(keyByAction[action]) : t("permissionPreflight.next.unknown");
}

export function permissionPackageApprovalRouteLabel(request: PermissionPackageApprovalRequest) {
  return `${request.tenantId} / ${request.workspaceId} / ${request.callerInstanceId}`;
}

export function productionConsoleStatusTone(status: AiAdminProductionConsoleStatus): Tone {
  if (status === "ready") return "success";
  if (status === "needs_review") return "warning";
  if (status === "blocked") return "danger";
  return "neutral";
}

export function isAcceptanceReportActionCode(code: string | undefined) {
  return code === "export_acceptance_report" || code === "export_production_evidence";
}

export function permissionWorkbenchActionKey(code: string | undefined, fallback: string) {
  if (isAcceptanceReportActionCode(code)) return "action.exportAcceptanceReport";
  switch (code) {
    case "complete_request":
      return "action.completePermissionRequest";
    case "create_approval_request":
      return "action.createApprovalRequest";
    case "review_approval_request":
      return "action.reviewApprovalRequest";
    case "apply_permission_package":
      return "action.applyPermissionPackage";
    case "run_runtime_validation":
      return "action.runApprovalJourney";
    default:
      return fallback;
  }
}

export function permissionWorkbenchStatusLabelKey(status: string | undefined) {
  if (!status) return "permissionWorkbench.status.pending";
  return `permissionWorkbench.status.${status}`;
}

export function permissionWorkbenchStatusDetailKey(status: string | undefined) {
  if (!status) return "permissionWorkbench.statusDetail.pending";
  return `permissionWorkbench.statusDetail.${status}`;
}

export function permissionWorkbenchStatusTone(status: string | undefined, fallback: AiAdminProductionConsoleStatus): Tone {
  if (status === "production_ready") return "success";
  if (status === "blocked") return "danger";
  if (status === "ready_to_apply" || status === "validating" || status === "awaiting_approval") return "warning";
  if (status === "needs_input") return "neutral";
  return productionConsoleStatusTone(fallback);
}

export function permissionWorkbenchStepLabelKey(key: string) {
  return `permissionWorkbench.step.${key}`;
}

export function permissionWorkbenchStepDetailKey(detailCode: string) {
  return `permissionWorkbench.detail.${detailCode}`;
}

export function permissionWorkbenchStepDisplayDetailCode(
  step: {
    detailCode: string;
    key: PermissionPackageWorkbenchStepKey;
  },
  args: {
    approvalRequired: boolean;
    approvalStatus: PermissionPackageApprovalEffectiveStatus | null;
    applicationReady: boolean;
    goLiveReady: boolean;
    runtimeValidationReady: boolean;
  }
) {
  if (args.goLiveReady) {
    if (step.key === "approval") return args.approvalRequired ? "approval_approved" : "approval_not_required";
    if (step.key === "apply") return "apply_done";
    if (step.key === "validation") return "validation_ready";
    if (step.key === "acceptance") return "acceptance_ready";
  }
  if (step.key === "approval" && args.approvalStatus === "approved") {
    return args.approvalRequired ? "approval_approved" : "approval_not_required";
  }
  if (step.key === "apply" && args.applicationReady) return "apply_done";
  if (step.key === "validation" && args.runtimeValidationReady) return "validation_ready";
  return step.detailCode;
}

export function permissionWorkbenchStepDisplayStatus(
  step: {
    key: PermissionPackageWorkbenchStepKey;
    status: PermissionPackageWorkbenchStepStatus;
  },
  args: {
    approvalComplete: boolean;
    applicationReady: boolean;
    goLiveReady: boolean;
    runtimeValidationReady: boolean;
  }
): PermissionPackageWorkbenchStepStatus {
  if (args.goLiveReady) return "complete";
  if (step.key === "approval" && args.approvalComplete) return "complete";
  if (step.key === "apply" && args.applicationReady) return "complete";
  if (step.key === "validation" && args.runtimeValidationReady) return "complete";
  return step.status;
}
