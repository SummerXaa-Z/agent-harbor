import type {
  Capability,
  ConsoleData,
  DataScope,
  ManagementScope,
  TenantEntitlement,
  WorkspaceAssignment
} from "./types";

// Shared, framework-free logic for the capability grant chain (租户授权 →
// 工作区分配 → 调用方分配). The legacy console hook and the redesign views
// both build on these functions so the two consoles stay behaviour-identical.

export interface CapabilityGrantForm {
  callerInstanceId: string;
  capabilityId: string;
  subjectSelector: string;
  targetId: string;
  tenantId: string;
  workspaceId: string;
}

export function defaultCapabilityGrantForm(defaultScope: ManagementScope): CapabilityGrantForm {
  return {
    callerInstanceId: "",
    capabilityId: "",
    subjectSelector: "user:support-*",
    targetId: "",
    tenantId: defaultScope.tenantId,
    workspaceId: defaultScope.workspaceId
  };
}

export function shallowEqualCapabilityForm(left: CapabilityGrantForm, right: CapabilityGrantForm) {
  return (
    left.callerInstanceId === right.callerInstanceId &&
    left.capabilityId === right.capabilityId &&
    left.subjectSelector === right.subjectSelector &&
    left.targetId === right.targetId &&
    left.tenantId === right.tenantId &&
    left.workspaceId === right.workspaceId
  );
}

// Default-fills the mutable fields as data arrives: the first MCP target, the
// target's first capability (falling back to any capability), and an active
// local caller matching the current tenant/workspace.
export function normalizeCapabilityGrantForm(current: CapabilityGrantForm, data: ConsoleData): CapabilityGrantForm {
  const mcpTarget = data.agents.find((agent) => agent.channelType === "mcp");
  const targetId = current.targetId || mcpTarget?.id || "";
  const capability = data.capabilities.find((item) => item.id === current.capabilityId && item.targetId === targetId)
    ?? data.capabilities.find((item) => item.targetId === targetId)
    ?? data.capabilities[0];
  const caller = data.agents.find(
    (agent) =>
      agent.status === "active" &&
      agent.tenantId === current.tenantId &&
      agent.workspaceId === current.workspaceId &&
      agent.channelType === "local"
  ) ?? data.agents.find((agent) => agent.status === "active" && agent.channelType === "local");
  const next = {
    ...current,
    callerInstanceId: current.callerInstanceId || caller?.id || "",
    capabilityId: current.capabilityId || capability?.id || "",
    targetId
  };
  return shallowEqualCapabilityForm(current, next) ? current : next;
}

export interface CapabilityGrantChainPlan {
  callerInstanceId: string;
  capability: Capability;
  dataScopes: DataScope[];
  subjectSelector: string;
  tenantId: string;
  workspaceId: string;
}

export type CapabilityGrantValidation = { messageKey: string } | { plan: CapabilityGrantChainPlan };

// Submit-time validation. `"*"` is rejected explicitly so nobody grants a
// capability to every subject at once.
export function validateCapabilityGrantChain(
  form: CapabilityGrantForm,
  capabilities: readonly Capability[],
): CapabilityGrantValidation {
  const capability = capabilities.find((item) => item.id === form.capabilityId);
  const tenantId = form.tenantId.trim();
  const workspaceId = form.workspaceId.trim();
  const callerInstanceId = form.callerInstanceId.trim();
  if (!capability) return { messageKey: "message.validationCapabilityRequired" };
  if (!tenantId || !workspaceId || !callerInstanceId) return { messageKey: "message.validationTenantWorkspaceCaller" };
  const subjectSelector = form.subjectSelector.trim();
  if (!subjectSelector || subjectSelector === "*") return { messageKey: "message.validationSubjectSelectorRequired" };
  return {
    plan: { callerInstanceId, capability, dataScopes: capability.dataScopes ?? [], subjectSelector, tenantId, workspaceId }
  };
}

// Live blocker for the grant sheet; mirrors submit-time validation but breaks
// the caller out for a more specific message. Null means ready to submit.
export function capabilityGrantBlockerKey(
  form: CapabilityGrantForm,
  capability: Capability | null | undefined,
): string | null {
  if (!capability) return "message.validationCapabilityRequired";
  if (!form.tenantId.trim() || !form.workspaceId.trim()) return "message.validationTenantWorkspaceCaller";
  if (!form.callerInstanceId.trim()) return "message.capabilityGrantCallerRequired";
  const subjectSelector = form.subjectSelector.trim();
  if (!subjectSelector || subjectSelector === "*") return "message.validationSubjectSelectorRequired";
  return null;
}

export function mergeCapabilitiesForTarget(existing: Capability[], refreshed: Capability[], targetId: string) {
  return [
    ...existing.filter((capability) => capability.targetId !== targetId),
    ...refreshed
  ];
}

// Demo-mode gate: while capabilities did not come from the API, network-level
// failures fall back to local sample mutations instead of surfacing an error.
export function shouldUseLocalCapabilityFallback(error: unknown, data: ConsoleData | null) {
  if (data?.capabilitiesLoadedFromApi) return false;
  return error instanceof Error && /Failed to fetch|404|not found|Not Found/i.test(error.message);
}

// Local (demo) construction of the three-layer grant chain, idempotent: an
// existing entitlement/workspace/instance record is updated in place instead
// of duplicated, and the capability flips to approved.
export function appendLocalCapabilityGrantChain(
  current: ConsoleData,
  capability: Capability,
  form: CapabilityGrantForm,
  dataScopes: DataScope[],
  defaultScope: ManagementScope
): ConsoleData {
  const now = new Date().toISOString();
  const tenantId = form.tenantId.trim() || defaultScope.tenantId;
  const workspaceId = form.workspaceId.trim() || defaultScope.workspaceId;
  const callerInstanceId = form.callerInstanceId.trim();
  const subjectSelector = form.subjectSelector.trim();

  const existingEntitlement = current.tenantEntitlements.find(
    (item) => item.tenantId === tenantId && item.targetId === capability.targetId && item.capabilityId === capability.id
  );
  const entitlement: TenantEntitlement = existingEntitlement
    ? { ...existingEntitlement, dataScopes, effect: "allow", status: "enabled", updatedAt: now }
    : {
        capabilityId: capability.id,
        createdAt: now,
        dataScopes,
        effect: "allow",
        id: nextLocalId("tent", [tenantId, capability.id], current.tenantEntitlements.map((item) => item.id)),
        priority: 50,
        status: "enabled",
        targetId: capability.targetId,
        tenantId,
        updatedAt: now
      };
  const tenantEntitlements = existingEntitlement
    ? current.tenantEntitlements.map((item) => (item.id === entitlement.id ? entitlement : item))
    : [entitlement, ...current.tenantEntitlements];

  const existingWorkspaceAssignment = current.workspaceAssignments.find(
    (item) => item.tenantEntitlementId === entitlement.id && item.workspaceId === workspaceId
  );
  const workspaceAssignment: WorkspaceAssignment = existingWorkspaceAssignment
    ? { ...existingWorkspaceAssignment, dataScopes, effect: "allow", status: "enabled", updatedAt: now }
    : {
        createdAt: now,
        dataScopes,
        effect: "allow",
        id: nextLocalId("wsa", [workspaceId, entitlement.id], current.workspaceAssignments.map((item) => item.id)),
        status: "enabled",
        tenantId,
        tenantEntitlementId: entitlement.id,
        updatedAt: now,
        workspaceId
      };
  const workspaceAssignments = existingWorkspaceAssignment
    ? current.workspaceAssignments.map((item) => (item.id === workspaceAssignment.id ? workspaceAssignment : item))
    : [workspaceAssignment, ...current.workspaceAssignments];

  const existingInstanceAssignment = current.instanceAssignments.find(
    (item) => item.workspaceAssignmentId === workspaceAssignment.id && item.callerInstanceId === callerInstanceId
  );
  const instanceAssignment = existingInstanceAssignment
    ? { ...existingInstanceAssignment, dataScopes, effect: "allow" as const, status: "enabled" as const, subjectSelector, updatedAt: now }
    : {
        callerInstanceId,
        createdAt: now,
        dataScopes,
        effect: "allow" as const,
        id: nextLocalId("ia", [callerInstanceId, workspaceAssignment.id], current.instanceAssignments.map((item) => item.id)),
        status: "enabled" as const,
        subjectSelector,
        tenantId,
        updatedAt: now,
        workspaceId,
        workspaceAssignmentId: workspaceAssignment.id
      };
  const instanceAssignments = existingInstanceAssignment
    ? current.instanceAssignments.map((item) => (item.id === instanceAssignment.id ? instanceAssignment : item))
    : [instanceAssignment, ...current.instanceAssignments];

  return {
    ...current,
    capabilities: current.capabilities.map((item) =>
      item.id === capability.id ? { ...item, discoveryStatus: "approved", updatedAt: now } : item
    ),
    instanceAssignments,
    tenantEntitlements,
    workspaceAssignments
  };
}

function nextLocalId(prefix: string, parts: string[], existing: string[]) {
  const base = `${prefix}_${parts.map(safeIdPart).filter(Boolean).join("_") || "local"}`;
  let candidate = base;
  let index = 1;
  while (existing.includes(candidate)) {
    index += 1;
    candidate = `${base}_${index}`;
  }
  return candidate;
}

function safeIdPart(value: string) {
  return value.trim().replace(/[^a-zA-Z0-9_-]+/g, "_").replace(/^_+|_+$/g, "").slice(0, 40);
}
