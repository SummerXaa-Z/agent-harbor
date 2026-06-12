import type { PermissionPackageTemplate } from "./permissionPackages";
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

export interface AskEvidenceChainRow {
  id?: string
  isBroken: boolean
  layer: string
  layerKey: string
  message: string
  status: string
  tone: "danger" | "neutral" | "success" | "warning"
}

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

export function evidenceChainRows(result: AccessDecisionExplainResult): AskEvidenceChainRow[] {
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

function normalizeOptional(value?: string) {
  return value?.trim() ?? "";
}

function evidenceTone(evidence: AccessDecisionExplainEvidence): AskEvidenceChainRow["tone"] {
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
