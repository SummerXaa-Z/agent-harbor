import type { PermissionPackageTemplate } from "./permissionPackages";
import type { Translator } from "./consolePresenters";
import type {
  AccessDecisionExplainRequest,
  AccessDecisionExplainResult,
  AccessDecisionExplainEvidence,
  Capability,
  ConsoleData,
  PermissionChangeHandoffContext
} from "./types";

export interface AskAccessSelection {
  tenantId?: string
  workspaceId?: string
  callerInstanceId?: string
  targetId?: string
  capabilityId?: string
  subjectId?: string
}

export interface ExplainRequestBuildResult {
  complete: boolean
  missingFields: Array<keyof AccessDecisionExplainRequest>
  request: AccessDecisionExplainRequest | null
}

export interface AskDecisionRecordRow {
  id?: string
  isBroken: boolean
  layer: string
  layerKey: string
  message: string
  status: string
  tone: "danger" | "neutral" | "success" | "warning"
}

export type AskEvidenceChainRow = AskDecisionRecordRow;

export interface PermissionChangeHandoffOptions {
  templates?: PermissionPackageTemplate[]
  translateIntent?: (key: string, values: Record<string, string>) => string
}

const requiredExplainFields: Array<keyof AccessDecisionExplainRequest> = [
  "tenantId",
  "workspaceId",
  "callerInstanceId",
  "targetId",
  "capabilityId"
];

export function buildExplainRequest(selection: AskAccessSelection): ExplainRequestBuildResult {
  const normalized = {
    capabilityId: normalizeOptional(selection.capabilityId),
    callerInstanceId: normalizeOptional(selection.callerInstanceId),
    subjectId: normalizeOptional(selection.subjectId),
    targetId: normalizeOptional(selection.targetId),
    tenantId: normalizeOptional(selection.tenantId),
    workspaceId: normalizeOptional(selection.workspaceId)
  };
  const missingFields = requiredExplainFields.filter((field) => !normalized[field]);
  if (missingFields.length > 0) {
    return {
      complete: false,
      missingFields,
      request: null
    };
  }

  return {
    complete: true,
    missingFields: [],
    request: {
      capabilityId: normalized.capabilityId,
      callerInstanceId: normalized.callerInstanceId,
      targetId: normalized.targetId,
      tenantId: normalized.tenantId,
      workspaceId: normalized.workspaceId,
      ...(normalized.subjectId ? { subjectId: normalized.subjectId } : {})
    }
  };
}

export function buildPermissionChangeHandoff(
  request: AccessDecisionExplainRequest,
  consoleData: ConsoleData,
  options: PermissionChangeHandoffOptions = {}
): PermissionChangeHandoffContext {
  const caller = consoleData.agents.find((agent) => agent.id === request.callerInstanceId);
  const target = consoleData.agents.find((agent) => agent.id === request.targetId);
  const tenant = consoleData.tenants.find((item) => item.id === request.tenantId);
  const capability = consoleData.capabilities.find((item) => item.id === request.capabilityId);
  const template = capability
    ? findTemplateForCapability(capability, options.templates ?? [])
    : undefined;
  const values = {
    callerName: caller?.name ?? request.callerInstanceId,
    capabilityName: capability?.displayName ?? capability?.key ?? request.capabilityId,
    targetName: target?.name ?? request.targetId,
    tenantName: tenant?.name ?? request.tenantId,
    workspaceName: workspaceNameFromId(request.workspaceId)
  };

  return {
    callerInstanceId: request.callerInstanceId,
    callerName: caller?.name,
    capabilityId: request.capabilityId,
    capabilityName: capability?.displayName ?? capability?.key,
    intentText: (options.translateIntent ?? defaultTranslateIntent)("ask.intent.openAccess", values),
    sourceView: "ask",
    subjectId: request.subjectId,
    targetId: request.targetId,
    targetName: target?.name,
    templateId: template?.id,
    tenantId: request.tenantId,
    tenantName: tenant?.name,
    workspaceId: request.workspaceId,
    workspaceName: values.workspaceName
  };
}

export function decisionRecordRows(result: AccessDecisionExplainResult): AskDecisionRecordRow[] {
  const brokenIndex = result.outcome === "denied"
    ? result.evidence.findIndex((evidence) => evidenceTone(evidence) === "danger")
    : -1;

  return result.evidence.map((evidence, index) => ({
    id: evidence.id,
    isBroken: index === brokenIndex,
    layer: evidence.layer,
    layerKey: `ask.evidenceLayer.${evidence.layer}`,
    message: evidence.message,
    status: evidence.status,
    tone: evidenceTone(evidence)
  }));
}

export const evidenceChainRows = decisionRecordRows;

export function accessDecisionSummaryLabel(result: AccessDecisionExplainResult, t: Translator) {
  const reason = accessDecisionReasonLabel(result.decision.reason, t);
  return tx(t, result.outcome === "allowed" ? "ask.summaryAllowed" : "ask.summaryDenied", { reason });
}

export function accessEvidenceMessageLabel(row: Pick<AskDecisionRecordRow, "message">, t: Translator) {
  const key = knownEvidenceMessages[row.message];
  return key ? t(key) : sanitizeAccessGuidance(row.message);
}

export function accessNextActionLabel(action: string, t: Translator) {
  const key = knownNextActions[action];
  return key ? t(key) : sanitizeAccessGuidance(action);
}

export function accessDecisionReasonLabel(reason: string, t: Translator) {
  const normalized = reason.trim();
  const key = knownDecisionReasons[normalized];
  return key ? t(key) : sanitizeAccessGuidance(normalized || t("ask.reason.noDetail"));
}

function normalizeOptional(value?: string) {
  return value?.trim() ?? "";
}

function evidenceTone(evidence: AccessDecisionExplainEvidence): AskDecisionRecordRow["tone"] {
  if (evidence.status === "matched") return "success";
  if (["blocking", "denied", "missing", "mismatch", "not_approved"].includes(evidence.status)) return "danger";
  if (["inactive", "pending_review"].includes(evidence.status)) return "warning";
  return "neutral";
}

function findTemplateForCapability(
  capability: Capability,
  templates: PermissionPackageTemplate[]
) {
  return templates.find((template) => templateAllowsCapability(capability, template));
}

function templateAllowsCapability(capability: Capability, template: PermissionPackageTemplate) {
  return (
    template.allowedActions.includes(capability.action) &&
    !template.blockedActions.includes(capability.action) &&
    !template.blockedRisks.includes(capability.riskLevel) &&
    !template.blockedSensitivities.includes(capability.sensitivity)
  );
}

function workspaceNameFromId(workspaceId: string) {
  const normalized = workspaceId.trim().replace(/^ws[-_]/, "").replace(/^workspace[-_]/, "");
  if (!normalized) return workspaceId;
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

function defaultTranslateIntent(_key: string, values: Record<string, string>) {
  return `Open ${values.capabilityName} access from ${values.callerName} to ${values.targetName}`;
}

const knownDecisionReasons: Record<string, string> = {
  "access grant matched": "ask.reason.accessGrantMatched",
  "caller has no route policy or access grant for target route": "ask.reason.routeGrantMissing",
  "caller instance assignment data scopes exceed workspace assignment boundary": "ask.reason.instanceScopeExceeded",
  "caller instance assignment denies capability": "ask.reason.instanceDenied",
  "caller instance has no assignment for capability": "ask.reason.instanceMissing",
  "capability assignment matched": "ask.reason.capabilityAssignmentMatched",
  "capability is not approved": "ask.reason.capabilityNotApproved",
  "capability is not registered for target": "ask.reason.capabilityNotRegistered",
  "tenant entitlement data scopes exceed capability boundary": "ask.reason.tenantScopeExceeded",
  "tenant entitlement denies capability": "ask.reason.tenantDenied",
  "tenant has no entitlement for capability": "ask.reason.tenantMissing",
  "workspace assignment data scopes exceed tenant entitlement boundary": "ask.reason.workspaceScopeExceeded",
  "workspace assignment denies capability": "ask.reason.workspaceDenied",
  "workspace has no assignment for capability": "ask.reason.workspaceMissing"
}

const knownEvidenceMessages: Record<string, string> = {
  "Caller instance assignment matched.": "ask.evidenceMessage.instanceMatched",
  "Caller instance assignment is missing or blocking this capability.": "ask.evidenceMessage.instanceBlocked",
  "Tenant entitlement matched.": "ask.evidenceMessage.tenantMatched",
  "Tenant entitlement is missing or blocking this capability.": "ask.evidenceMessage.tenantBlocked",
  "Workspace assignment matched.": "ask.evidenceMessage.workspaceMatched",
  "Workspace assignment is missing or blocking this capability.": "ask.evidenceMessage.workspaceBlocked"
}

const knownNextActions: Record<string, string> = {
  "Apply a permission package or create a workspace assignment for this tenant entitlement and workspace.": "ask.nextAction.createWorkspaceAssignment",
  "Apply a permission package or create an instance assignment for this caller instance.": "ask.nextAction.createCallerAssignment",
  "Approve the capability or apply a permission package that approves the selected capability.": "ask.nextAction.approveCapability",
  "Inspect get_tenant_access_profile for this tenant/workspace/caller/capability and repair the first missing decision layer.": "ask.nextAction.inspectProfile",
  "Narrow the child dataScopes so they stay inside the parent capability, tenant, workspace, or instance boundary.": "ask.nextAction.narrowDataScope",
  "No permission change is required. Review the returned dataScopes before broadening access.": "ask.nextAction.noChangeRequired",
  "Refresh the target MCP capabilities, then choose a registered capability from list_capabilities.": "ask.nextAction.refreshCapabilities",
  "Review the deny effect on the matching entitlement or assignment before granting broader access.": "ask.nextAction.reviewDeny",
  "Use the permission package flow with draft_permission_package and apply_permission_package to create the tenant entitlement, workspace assignment, and caller assignment together.": "ask.nextAction.usePermissionPackage"
}

function sanitizeAccessGuidance(value: string) {
  return value
    .replaceAll("draft_permission_package", "permission package draft")
    .replaceAll("apply_permission_package", "permission package apply")
    .replaceAll("get_tenant_access_profile", "tenant access profile")
    .replaceAll("list_capabilities", "capability list")
    .replace(/\bdataScopes\b/g, "data scopes")
    .replace(/_/g, " ")
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}
