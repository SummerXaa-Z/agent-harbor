import type {
  Agent,
  Capability,
  InstanceAssignment,
  ManagementScope,
  RoutePolicy,
  TenantEntitlement,
  TraceEvent,
  WorkspaceAssignment
} from "./types";

export type ResourceLifecycleKind = "caller" | "mcp_target" | "agent";
export type ResourceLifecycleSetupGapKind = "caller" | "target" | "capability";
export type ResourceLifecycleActionKind =
  | "create_key"
  | "create_policy"
  | "rotate_credential"
  | "review_capabilities"
  | "start_permission_change"
  | "review_runtime"
  | "review_resource";
export type ResourceLifecycleStatus =
  | "ready"
  | "needs_credentials"
  | "needs_capabilities"
  | "needs_approval"
  | "needs_runtime"
  | "disabled";

export type ResourceLifecycleNextActionHash = "#registry" | "#capabilities" | "#ai-admin" | "#traces";

export interface ResourceLifecycleItem {
  approvedCapabilityCount: number;
  capabilityCount: number;
  credentialVersion: number;
  detailKey: string;
  grantCount: number;
  id: string;
  kind: ResourceLifecycleKind;
  kindKey: string;
  name: string;
  nextActionKind: ResourceLifecycleActionKind;
  nextActionHash: ResourceLifecycleNextActionHash;
  nextActionKey: string;
  runtimeDecisionCount: number;
  status: ResourceLifecycleStatus;
  statusKey: string;
  tenantId: string;
  workspaceId: string;
}

export interface ResourceLifecycleSetupGap {
  actionHash: ResourceLifecycleNextActionHash;
  actionKey: string;
  detailKey: string;
  kind: ResourceLifecycleSetupGapKind;
  titleKey: string;
}

export interface ResourceLifecycleSummary {
  activeResources: number;
  callers: number;
  items: ResourceLifecycleItem[];
  mcpTargets: number;
  needsAttention: number;
  readyResources: number;
  setupGaps: ResourceLifecycleSetupGap[];
  totalResources: number;
}

export interface ResourceLifecycleInput {
  agents: Agent[];
  capabilities: Capability[];
  instanceAssignments: InstanceAssignment[];
  routePolicies: RoutePolicy[];
  scope?: ManagementScope;
  tenantEntitlements: TenantEntitlement[];
  traces: TraceEvent[];
  workspaceAssignments: WorkspaceAssignment[];
}

export function buildResourceLifecycleSummary(input: ResourceLifecycleInput): ResourceLifecycleSummary {
  const items = input.agents.map((agent) => buildResourceLifecycleItem(agent, input));
  const setupGaps = resourceSetupGaps(items, input);

  return {
    activeResources: items.filter((item) => item.status !== "disabled").length,
    callers: items.filter((item) => item.kind === "caller").length,
    items,
    mcpTargets: items.filter((item) => item.kind === "mcp_target").length,
    needsAttention: items.filter((item) => item.status !== "ready").length,
    readyResources: items.filter((item) => item.status === "ready").length,
    setupGaps,
    totalResources: items.length
  };
}

function buildResourceLifecycleItem(agent: Agent, input: ResourceLifecycleInput): ResourceLifecycleItem {
  const capabilities = input.capabilities.filter((capability) => capability.targetId === agent.id);
  const approvedCapabilities = capabilities.filter((capability) => capability.discoveryStatus === "approved");
  const entitlements = input.tenantEntitlements.filter((entitlement) => entitlement.targetId === agent.id && entitlement.status === "enabled");
  const workspaceAssignments = enabledWorkspaceAssignmentsForEntitlements(entitlements, input.workspaceAssignments);
  const instanceAssignments = enabledInstanceAssignmentsForWorkspaces(workspaceAssignments, input.instanceAssignments);
  const callerInstanceAssignments = input.instanceAssignments.filter(
    (assignment) => assignment.status === "enabled" && assignment.callerInstanceId === agent.id
  );
  const routePolicies = input.routePolicies.filter(
    (policy) => policy.status === "enabled" && (policy.callerAgentId === agent.id || policy.targetAgentId === agent.id)
  );
  const runtimeDecisionCount = input.traces.filter(
    (trace) => trace.callerAgentId === agent.id || trace.callerInstanceId === agent.id || trace.targetAgentId === agent.id
  ).length;
  const kind = resourceKind(agent);
  const grantCount = kind === "caller"
    ? callerInstanceAssignments.length + routePolicies.filter((policy) => policy.callerAgentId === agent.id).length
    : entitlements.length + routePolicies.filter((policy) => policy.targetAgentId === agent.id).length;
  const status = resolveResourceLifecycleStatus({
    agent,
    approvedCapabilityCount: approvedCapabilities.length,
    capabilityCount: capabilities.length,
    grantCount,
    kind,
    runtimeDecisionCount
  });

  return {
    approvedCapabilityCount: approvedCapabilities.length,
    capabilityCount: capabilities.length,
    credentialVersion: agent.credentialVersion,
    detailKey: detailKey(status),
    grantCount,
    id: agent.id,
    kind,
    kindKey: resourceKindKey(kind),
    name: agent.name,
    nextActionKind: nextActionKind(status),
    nextActionHash: nextActionHash(status),
    nextActionKey: nextActionKey(status),
    runtimeDecisionCount,
    status,
    statusKey: statusKey(status),
    tenantId: agent.tenantId,
    workspaceId: agent.workspaceId
  };
}

function resourceKind(agent: Agent): ResourceLifecycleKind {
  if (agent.channelType === "local") return "caller";
  if (agent.channelType === "mcp") return "mcp_target";
  return "agent";
}

function enabledWorkspaceAssignmentsForEntitlements(entitlements: TenantEntitlement[], assignments: WorkspaceAssignment[]) {
  const entitlementIds = new Set(entitlements.map((entitlement) => entitlement.id));
  return assignments.filter((assignment) => assignment.status === "enabled" && entitlementIds.has(assignment.tenantEntitlementId));
}

function enabledInstanceAssignmentsForWorkspaces(workspaceAssignments: WorkspaceAssignment[], assignments: InstanceAssignment[]) {
  const workspaceAssignmentIds = new Set(workspaceAssignments.map((assignment) => assignment.id));
  return assignments.filter((assignment) => assignment.status === "enabled" && workspaceAssignmentIds.has(assignment.workspaceAssignmentId));
}

function resolveResourceLifecycleStatus({
  agent,
  approvedCapabilityCount,
  capabilityCount,
  grantCount,
  kind,
  runtimeDecisionCount
}: {
  agent: Agent;
  approvedCapabilityCount: number;
  capabilityCount: number;
  grantCount: number;
  kind: ResourceLifecycleKind;
  runtimeDecisionCount: number;
}): ResourceLifecycleStatus {
  if (agent.status === "disabled") return "disabled";
  if (kind !== "caller" && agent.credentialVersion <= 0) return "needs_credentials";
  if (kind !== "caller" && capabilityCount === 0) return "needs_capabilities";
  if (kind !== "caller" && approvedCapabilityCount === 0) return "needs_approval";
  if (grantCount === 0) return "needs_approval";
  if (runtimeDecisionCount === 0) return "needs_runtime";
  return "ready";
}

function resourceKindKey(kind: ResourceLifecycleKind) {
  if (kind === "caller") return "resource.kind.caller";
  if (kind === "mcp_target") return "resource.kind.mcpTarget";
  return "resource.kind.agent";
}

function statusKey(status: ResourceLifecycleStatus) {
  const keys: Record<ResourceLifecycleStatus, string> = {
    disabled: "resource.status.disabled",
    needs_approval: "resource.status.needsApproval",
    needs_capabilities: "resource.status.needsCapabilities",
    needs_credentials: "resource.status.needsCredentials",
    needs_runtime: "resource.status.needsRuntime",
    ready: "resource.status.ready"
  };
  return keys[status];
}

function nextActionKey(status: ResourceLifecycleStatus) {
  const keys: Record<ResourceLifecycleStatus, string> = {
    disabled: "resource.nextAction.registry",
    needs_approval: "resource.nextAction.aiAdmin",
    needs_capabilities: "resource.nextAction.capabilities",
    needs_credentials: "resource.nextAction.registry",
    needs_runtime: "resource.nextAction.traces",
    ready: "resource.nextAction.reviewRuntime"
  };
  return keys[status];
}

function detailKey(status: ResourceLifecycleStatus) {
  const keys: Record<ResourceLifecycleStatus, string> = {
    disabled: "resource.detail.disabled",
    needs_approval: "resource.detail.needsApproval",
    needs_capabilities: "resource.detail.needsCapabilities",
    needs_credentials: "resource.detail.needsCredentials",
    needs_runtime: "resource.detail.needsRuntime",
    ready: "resource.detail.ready"
  };
  return keys[status];
}

function nextActionKind(status: ResourceLifecycleStatus): ResourceLifecycleActionKind {
  const actions: Record<ResourceLifecycleStatus, ResourceLifecycleActionKind> = {
    disabled: "review_resource",
    needs_approval: "start_permission_change",
    needs_capabilities: "review_capabilities",
    needs_credentials: "rotate_credential",
    needs_runtime: "review_runtime",
    ready: "review_runtime"
  };
  return actions[status];
}

function nextActionHash(status: ResourceLifecycleStatus): ResourceLifecycleNextActionHash {
  const hashes: Record<ResourceLifecycleStatus, ResourceLifecycleNextActionHash> = {
    disabled: "#registry",
    needs_approval: "#ai-admin",
    needs_capabilities: "#capabilities",
    needs_credentials: "#registry",
    needs_runtime: "#traces",
    ready: "#traces"
  };
  return hashes[status];
}

function resourceSetupGaps(
  items: ResourceLifecycleItem[],
  input: ResourceLifecycleInput
): ResourceLifecycleSetupGap[] {
  const setupScope = input.scope;
  const scopedItems = setupScope ? items.filter((item) => resourceInScope(item, setupScope)) : items;
  const activeItems = scopedItems.filter((item) => item.status !== "disabled");
  const activeTargetIds = new Set(
    activeItems
      .filter((item) => item.kind !== "caller")
      .map((item) => item.id)
  );
  const hasCaller = activeItems.some((item) => item.kind === "caller");
  const hasTarget = activeTargetIds.size > 0;
  const hasTargetCapability = input.capabilities.some((capability) => activeTargetIds.has(capability.targetId));
  const gaps: ResourceLifecycleSetupGap[] = [];

  if (!hasCaller) {
    gaps.push(setupGap("caller"));
  }
  if (!hasTarget) {
    gaps.push(setupGap("target"));
  } else if (!hasTargetCapability) {
    gaps.push(setupGap("capability"));
  }
  return gaps;
}

function resourceInScope(item: ResourceLifecycleItem, scope: ManagementScope) {
  const tenantId = scope.tenantId.trim();
  const workspaceId = scope.workspaceId.trim();
  return (!tenantId || item.tenantId === tenantId) && (!workspaceId || item.workspaceId === workspaceId);
}

function setupGap(kind: ResourceLifecycleSetupGapKind): ResourceLifecycleSetupGap {
  const gaps: Record<ResourceLifecycleSetupGapKind, ResourceLifecycleSetupGap> = {
    caller: {
      actionHash: "#registry",
      actionKey: "resource.setupGap.caller.action",
      detailKey: "resource.setupGap.caller.detail",
      kind: "caller",
      titleKey: "resource.setupGap.caller.title"
    },
    capability: {
      actionHash: "#capabilities",
      actionKey: "resource.setupGap.capability.action",
      detailKey: "resource.setupGap.capability.detail",
      kind: "capability",
      titleKey: "resource.setupGap.capability.title"
    },
    target: {
      actionHash: "#registry",
      actionKey: "resource.setupGap.target.action",
      detailKey: "resource.setupGap.target.detail",
      kind: "target",
      titleKey: "resource.setupGap.target.title"
    }
  };
  return gaps[kind];
}
