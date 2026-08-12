import type { PermissionPackageDraftInput, PermissionPackageTemplate } from "./permissionPackages";
import type { Translator } from "./consolePresenters";
import type {
  AccessDecisionExplainRequest,
  AccessDecisionExplainResult,
  AccessDecisionExplainEvidence,
  Agent,
  Capability,
  ConsoleData,
  PermissionChangeHandoffContext,
  Tenant
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

export interface PermissionChangeHandoffOptions {
  decisionResult?: AccessDecisionExplainResult
  templates?: PermissionPackageTemplate[]
  translateIntent?: (key: string, values: Record<string, string>) => string
}

export interface AskAccessInventory {
  agents: Agent[]
  capabilities: Capability[]
  tenants: Tenant[]
}

export interface AskAccessScopeOptions {
  callers: Agent[]
  capabilities: Capability[]
  targets: Agent[]
  tenants: Tenant[]
  workspaceIds: string[]
}

export type AccessDecisionPrimaryAction =
  | { kind: "access_profile"; labelKey: "action.openAccessProfile" | "action.reviewPermissionBoundary" | "action.reviewPlatformManagedTarget" }
  | { kind: "capability_review"; labelKey: "action.reviewCapabilityApproval" | "action.classifyCapability" }
  | { kind: "permission_change"; labelKey: "action.fixAccessDecision" }

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

export function askAccessScopeOptions(
  inventory: AskAccessInventory,
  selection: AskAccessSelection
): AskAccessScopeOptions {
  const tenantId = normalizeOptional(selection.tenantId);
  const workspaceId = normalizeOptional(selection.workspaceId);
  const targetId = normalizeOptional(selection.targetId);
  const activeTenants = inventory.tenants.filter((tenant) => tenant.status === "active");
  const activeTenantIds = new Set(activeTenants.map((tenant) => tenant.id));
  const activeAgents = inventory.agents.filter((agent) => (
    agent.status === "active" && activeTenantIds.has(agent.tenantId)
  ));
  const targetIdsWithCapabilities = new Set(inventory.capabilities.map((capability) => capability.targetId));
  const tenantAgents = activeAgents.filter((agent) => !tenantId || agent.tenantId === tenantId);
  const scopedAgents = tenantAgents.filter((agent) => !workspaceId || agent.workspaceId === workspaceId);
  const targets = activeAgents.filter((agent) => (
    targetIdsWithCapabilities.has(agent.id)
    && askAccessTargetVisibleToScope(agent, tenantId, workspaceId, inventory.tenants)
  ));
  const visibleTargetIds = new Set(targets.map((agent) => agent.id));

  return {
    callers: scopedAgents.filter((agent) => agent.channelType === "local"),
    capabilities: targetId && visibleTargetIds.has(targetId)
      ? inventory.capabilities.filter((capability) => capability.targetId === targetId)
      : [],
    targets,
    tenants: activeTenants,
    workspaceIds: uniqueStrings(tenantAgents.map((agent) => agent.workspaceId))
  };
}

export function askAccessTargetVisibleToScope(
  target: Agent,
  tenantId: string,
  workspaceId: string,
  tenants: Tenant[]
): boolean {
  const normalizedTenantId = normalizeOptional(tenantId);
  const normalizedWorkspaceId = normalizeOptional(workspaceId);
  if (
    !normalizedTenantId
    || !normalizedWorkspaceId
    || target.status !== "active"
    || target.workspaceId !== normalizedWorkspaceId
  ) {
    return false;
  }

  const tenantsById = new Map(tenants.map((tenant) => [tenant.id, tenant]));
  const visitedTenantIds = new Set<string>();
  let currentTenant = tenantsById.get(normalizedTenantId);
  while (currentTenant && currentTenant.status === "active" && !visitedTenantIds.has(currentTenant.id)) {
    if (currentTenant.id === target.tenantId) return true;
    visitedTenantIds.add(currentTenant.id);
    currentTenant = currentTenant.parentTenantId
      ? tenantsById.get(currentTenant.parentTenantId)
      : undefined;
  }
  return false;
}

export function resolveAskAccessSelection(
  selection: AskAccessSelection,
  inventory: AskAccessInventory
): AskAccessSelection {
  const activeTenants = inventory.tenants.filter((tenant) => tenant.status === "active");
  const activeTenantIds = new Set(activeTenants.map((tenant) => tenant.id));
  const activeAgents = inventory.agents.filter((agent) => (
    agent.status === "active" && activeTenantIds.has(agent.tenantId)
  ));
  const targetIdsWithCapabilities = new Set(inventory.capabilities.map((capability) => capability.targetId));
  const usableScopes = activeAgents
    .filter((agent) => agent.channelType === "local")
    .map((caller) => ({ tenantId: caller.tenantId, workspaceId: caller.workspaceId }))
    .filter((scope) => activeAgents.some((agent) => (
      targetIdsWithCapabilities.has(agent.id)
      && askAccessTargetVisibleToScope(agent, scope.tenantId, scope.workspaceId, inventory.tenants)
    )));
  const selectedCaller = activeAgents.find((agent) => (
    agent.id === selection.callerInstanceId && agent.channelType === "local"
  ));
  const explicitTenant = hasSelectionField(selection, "tenantId");
  const explicitWorkspace = hasSelectionField(selection, "workspaceId");
  const explicitCaller = hasSelectionField(selection, "callerInstanceId");
  const explicitTarget = hasSelectionField(selection, "targetId");
  const explicitCapability = hasSelectionField(selection, "capabilityId");
  const preferredScope = selectedCaller && !explicitTenant && !explicitWorkspace
    ? { tenantId: selectedCaller.tenantId, workspaceId: selectedCaller.workspaceId }
    : usableScopes.find((scope) => (
    (!selection.tenantId || scope.tenantId === selection.tenantId)
    && (!selection.workspaceId || scope.workspaceId === selection.workspaceId)
  )) ?? (selectedCaller ? {
    tenantId: selectedCaller.tenantId,
    workspaceId: selectedCaller.workspaceId
  } : usableScopes[0]);
  const selectedTenantId = normalizeOptional(selection.tenantId);
  const tenantId = explicitTenant
    ? activeTenants.some((tenant) => tenant.id === selectedTenantId) ? selectedTenantId : ""
    : preferredScope?.tenantId ?? activeTenants[0]?.id ?? "";
  if (!tenantId) return clearedAskAccessSelection(selection.subjectId);
  const tenantAgents = activeAgents.filter((agent) => agent.tenantId === tenantId);
  const workspaceIds = uniqueStrings(tenantAgents.map((agent) => agent.workspaceId));
  const selectedWorkspaceId = normalizeOptional(selection.workspaceId);
  const workspaceId = explicitWorkspace
    ? workspaceIds.includes(selectedWorkspaceId) ? selectedWorkspaceId : ""
    : preferredScope?.tenantId === tenantId && workspaceIds.includes(preferredScope.workspaceId)
      ? preferredScope.workspaceId
      : workspaceIds[0] ?? "";
  if (!workspaceId) return { ...clearedAskAccessSelection(selection.subjectId), tenantId };
  const scopedAgents = tenantAgents.filter((agent) => agent.workspaceId === workspaceId);
  const callers = scopedAgents.filter((agent) => agent.channelType === "local");
  const targets = activeAgents.filter((agent) => (
    targetIdsWithCapabilities.has(agent.id)
    && askAccessTargetVisibleToScope(agent, tenantId, workspaceId, inventory.tenants)
  ));
  const caller = callers.find((agent) => agent.id === selection.callerInstanceId)
    ?? (explicitCaller ? undefined : callers[0]);
  const target = targets.find((agent) => agent.id === selection.targetId)
    ?? (explicitTarget ? undefined : targets[0]);
  const capabilities = inventory.capabilities.filter((capability) => capability.targetId === target?.id);
  const capability = capabilities.find((item) => item.id === selection.capabilityId)
    ?? (explicitCapability ? undefined : capabilities[0]);

  return {
    capabilityId: capability?.id ?? "",
    callerInstanceId: caller?.id ?? "",
    subjectId: selection.subjectId ?? "",
    targetId: target?.id ?? "",
    tenantId,
    workspaceId
  };
}

function hasSelectionField(selection: AskAccessSelection, field: keyof AskAccessSelection) {
  return Object.prototype.hasOwnProperty.call(selection, field);
}

function clearedAskAccessSelection(subjectId?: string): AskAccessSelection {
  return {
    capabilityId: "",
    callerInstanceId: "",
    subjectId: subjectId ?? "",
    targetId: "",
    tenantId: "",
    workspaceId: ""
  };
}

export function accessDecisionPrimaryAction(
  result: AccessDecisionExplainResult,
  capabilities: Capability[],
  templates: PermissionPackageTemplate[],
  permissionChangeAvailable = true
): AccessDecisionPrimaryAction | null {
  if (result.outcome === "allowed") return null;
  const code = result.nextActionCodes?.[0] ?? legacyAccessDecisionActionCode(result);
  if (["approve_capability", "refresh_capabilities"].includes(code)) {
    return { kind: "capability_review", labelKey: "action.reviewCapabilityApproval" };
  }
  if (["create_caller_assignment", "create_workspace_assignment", "use_permission_package"].includes(code)) {
    const capability = capabilities.find((item) => item.id === result.request.capabilityId);
    if (capability && !hasExplicitCapabilityDomain(capability)) {
      return { kind: "capability_review", labelKey: "action.classifyCapability" };
    }
    if (capability && findTemplateForCapability(capability, templates)) {
      if (!permissionChangeAvailable) {
        return { kind: "access_profile", labelKey: "action.reviewPlatformManagedTarget" };
      }
      return { kind: "permission_change", labelKey: "action.fixAccessDecision" };
    }
    return { kind: "access_profile", labelKey: "action.reviewPermissionBoundary" };
  }
  return { kind: "access_profile", labelKey: "action.openAccessProfile" };
}

export function canStartPermissionChangeForAdmin(
  adminRole: string | undefined,
  request: AccessDecisionExplainRequest | undefined,
  agents: Agent[]
): boolean {
  if (!request) return false;
  const role = normalizeOptional(adminRole) || "platform_admin";
  if (role === "platform_admin") return true;
  const target = agents.find((agent) => agent.id === request.targetId);
  return Boolean(target && target.tenantId === request.tenantId);
}

function hasExplicitCapabilityDomain(capability: Capability) {
  return uniqueStrings([
    ...(capability.dataDomains ?? []),
    ...(capability.dataScopes ?? []).map((scope) => scope.dataDomain ?? "")
  ]).length > 0;
}

function legacyAccessDecisionActionCode(result: AccessDecisionExplainResult) {
  const reason = result.decision.reason.toLowerCase();
  if (reason.includes("not registered")) return "refresh_capabilities";
  if (reason.includes("not approved")) return "approve_capability";
  if (reason.includes("tenant has no entitlement")) return "use_permission_package";
  if (reason.includes("workspace has no assignment")) return "create_workspace_assignment";
  if (reason.includes("caller instance has no assignment")) return "create_caller_assignment";
  return "inspect_access_profile";
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
  const brokenRecord = options.decisionResult
    ? decisionRecordRows(options.decisionResult).find((row) => row.isBroken)
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
    ...(brokenRecord?.layer ? { brokenLayer: brokenRecord.layer } : {}),
    ...(options.decisionResult?.decision.reason ? { decisionReason: options.decisionResult.decision.reason } : {}),
    ...(options.decisionResult?.decision.source ? { decisionSource: options.decisionResult.decision.source } : {}),
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

export function permissionChangeHandoffDraftInput(
  context: PermissionChangeHandoffContext,
  current: PermissionPackageDraftInput,
): PermissionPackageDraftInput {
  return {
    ...current,
    callerInstanceId: context.callerInstanceId ?? current.callerInstanceId,
    requestText: context.intentText ?? current.requestText,
    requestedCapabilityId: context.sourceView === "ask" ? context.capabilityId : undefined,
    subjectSelector: context.sourceView === "ask"
      ? context.subjectId ?? ""
      : context.subjectId ?? current.subjectSelector,
    targetId: context.targetId ?? current.targetId,
    templateId: context.sourceView === "ask"
      ? context.templateId ?? ""
      : context.templateId ?? current.templateId,
    tenantId: context.tenantId,
    workspaceId: context.workspaceId,
  };
}

export function decisionRecordRows(result: AccessDecisionExplainResult): AskDecisionRecordRow[] {
  const brokenIndex = result.outcome === "denied"
    ? result.evidence.findIndex((record) => recordTone(record) === "danger")
    : -1;

  return result.evidence.map((record, index) => ({
    id: record.id,
    isBroken: index === brokenIndex,
    layer: record.layer,
    layerKey: `ask.recordLayer.${record.layer}`,
    message: record.message,
    status: record.status,
    tone: recordTone(record)
  }));
}

export function accessDecisionSummaryLabel(result: AccessDecisionExplainResult, t: Translator) {
  const reason = accessDecisionReasonLabel(result.decision.reason, t);
  return tx(t, result.outcome === "allowed" ? "ask.summaryAllowed" : "ask.summaryDenied", { reason });
}

export function accessDecisionRecordMessageLabel(row: Pick<AskDecisionRecordRow, "message">, t: Translator) {
  const key = knownDecisionRecordMessages[row.message];
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

function recordTone(record: AccessDecisionExplainEvidence): AskDecisionRecordRow["tone"] {
  if (record.status === "matched") return "success";
  if (["blocking", "denied", "missing", "mismatch", "not_approved"].includes(record.status)) return "danger";
  if (["inactive", "pending_review"].includes(record.status)) return "warning";
  return "neutral";
}

function findTemplateForCapability(
  capability: Capability,
  templates: PermissionPackageTemplate[]
) {
  return templates.find((template) => templateAllowsCapability(capability, template));
}

function templateAllowsCapability(capability: Capability, template: PermissionPackageTemplate) {
  const capabilityDomains = uniqueStrings([
    ...(capability.dataDomains ?? []),
    ...(capability.dataScopes ?? []).map((scope) => scope.dataDomain ?? "")
  ]);
  return (
    capabilityDomains.length > 0 &&
    capabilityDomains.every((domain) => domain === template.defaultDataDomain) &&
    template.allowedActions.includes(capability.action) &&
    !template.blockedActions.includes(capability.action) &&
    !template.blockedRisks.includes(capability.riskLevel) &&
    !template.blockedSensitivities.includes(capability.sensitivity) &&
    !template.guardrails.some((guardrail) => (
      guardrail.capabilityKey === capability.key && guardrail.expectedDecision === "deny"
    ))
  );
}

function uniqueStrings(values: string[]) {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean)));
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

const knownDecisionRecordMessages: Record<string, string> = {
  "Caller instance assignment matched.": "ask.recordMessage.instanceMatched",
  "Caller instance assignment is missing or blocking this capability.": "ask.recordMessage.instanceBlocked",
  "Tenant entitlement matched.": "ask.recordMessage.tenantMatched",
  "Tenant entitlement is missing or blocking this capability.": "ask.recordMessage.tenantBlocked",
  "Workspace assignment matched.": "ask.recordMessage.workspaceMatched",
  "Workspace assignment is missing or blocking this capability.": "ask.recordMessage.workspaceBlocked"
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
