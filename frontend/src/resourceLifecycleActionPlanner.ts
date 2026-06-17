import type { NavKey } from "./consoleNavigation";
import type { ResourceLifecycleItem } from "./resourceLifecycle";
import type { Agent, CapabilityGovernanceHandoffContext, PermissionChangeHandoffContext, TraceFilters } from "./types";

export type ResourceLifecycleModal = "rotate_credential" | "create_policy" | "create_key";

export interface ResourceLifecycleActionContext {
  resourceKindKey: string;
  resourceName: string;
  tenantName: string;
  workspaceName: string;
}

export type ResourceLifecycleActionPlan =
  | {
      agentId?: string;
      callerAgentId?: string;
      context: ResourceLifecycleActionContext;
      kind: "open_modal";
      modal: ResourceLifecycleModal;
      targetAgentId?: string;
    }
  | { context: CapabilityGovernanceHandoffContext; kind: "capability_prefill"; navKey: "capabilities"; targetId: string }
  | { context: PermissionChangeHandoffContext; kind: "permission_handoff" }
  | { kind: "runtime_filters"; navKey: "traces"; traceFilters: TraceFilters }
  | { kind: "navigate"; navKey: NavKey };

export interface ResourceLifecycleActionPlanInput {
  agents: Agent[];
  formatEntityName: (name: string) => string;
  formatPermissionIntent: (targetName: string) => string;
  formatTenantName: (tenantId: string) => string;
  formatWorkspaceName: (workspaceId: string) => string;
  item: ResourceLifecycleItem;
  localCallers: Agent[];
  mcpTargets: Agent[];
}

export function planResourceLifecycleAction({
  agents,
  formatEntityName,
  formatPermissionIntent,
  formatTenantName,
  formatWorkspaceName,
  item,
  localCallers,
  mcpTargets
}: ResourceLifecycleActionPlanInput): ResourceLifecycleActionPlan {
  const context = resourceActionContext({
    formatEntityName,
    formatTenantName,
    formatWorkspaceName,
    item
  });
  const sameScopeCaller = localCallers.find((agent) => sameScope(agent, item));
  const sameScopeTarget = mcpTargets.find((agent) => sameScope(agent, item));

  if (item.nextActionKind === "rotate_credential") {
    return { agentId: item.id, context, kind: "open_modal", modal: "rotate_credential" };
  }
  if (item.nextActionKind === "create_key") {
    return {
      agentId: item.kind === "caller" ? item.id : sameScopeCaller?.id,
      context,
      kind: "open_modal",
      modal: "create_key"
    };
  }
  if (item.nextActionKind === "create_policy") {
    return {
      callerAgentId: item.kind === "caller" ? item.id : sameScopeCaller?.id,
      context,
      kind: "open_modal",
      modal: "create_policy",
      targetAgentId: item.kind === "caller" ? sameScopeTarget?.id : item.id
    };
  }
  if (item.nextActionKind === "review_capabilities") {
    return {
      context: capabilityHandoffContext({
        formatEntityName,
        formatTenantName,
        formatWorkspaceName,
        item
      }),
      kind: "capability_prefill",
      navKey: "capabilities",
      targetId: item.id
    };
  }
  if (item.nextActionKind === "start_permission_change") {
    return {
      context: permissionHandoffContext({
        agents,
        formatEntityName,
        formatPermissionIntent,
        formatTenantName,
        formatWorkspaceName,
        item,
        localCallers,
        mcpTargets
      }),
      kind: "permission_handoff"
    };
  }
  if (item.nextActionKind === "review_runtime") {
    return {
      kind: "runtime_filters",
      navKey: "traces",
      traceFilters: item.kind === "caller" ? { callerAgentId: item.id } : { targetAgentId: item.id }
    };
  }
  return { kind: "navigate", navKey: "registry" };
}

function resourceActionContext({
  formatEntityName,
  formatTenantName,
  formatWorkspaceName,
  item
}: {
  formatEntityName: (name: string) => string;
  formatTenantName: (tenantId: string) => string;
  formatWorkspaceName: (workspaceId: string) => string;
  item: ResourceLifecycleItem;
}): ResourceLifecycleActionContext {
  return {
    resourceKindKey: item.kindKey,
    resourceName: formatEntityName(item.name),
    tenantName: formatTenantName(item.tenantId),
    workspaceName: formatWorkspaceName(item.workspaceId)
  };
}

function capabilityHandoffContext({
  formatEntityName,
  formatTenantName,
  formatWorkspaceName,
  item
}: {
  formatEntityName: (name: string) => string;
  formatTenantName: (tenantId: string) => string;
  formatWorkspaceName: (workspaceId: string) => string;
  item: ResourceLifecycleItem;
}): CapabilityGovernanceHandoffContext {
  return {
    sourceView: "registry",
    targetId: item.id,
    targetName: formatEntityName(item.name),
    tenantId: item.tenantId,
    tenantName: formatTenantName(item.tenantId),
    workspaceId: item.workspaceId,
    workspaceName: formatWorkspaceName(item.workspaceId)
  };
}

function permissionHandoffContext({
  agents,
  formatEntityName,
  formatPermissionIntent,
  formatTenantName,
  formatWorkspaceName,
  item,
  localCallers,
  mcpTargets
}: ResourceLifecycleActionPlanInput): PermissionChangeHandoffContext {
  const resourceAgent = agents.find((agent) => agent.id === item.id);
  const sameScopeCaller = localCallers.find((agent) => sameScope(agent, item));
  const sameScopeTarget = mcpTargets.find((agent) => sameScope(agent, item));
  const caller = item.kind === "caller" ? resourceAgent : sameScopeCaller;
  const target = item.kind === "caller" ? sameScopeTarget : resourceAgent;
  const targetName = target ? formatEntityName(target.name) : undefined;

  return {
    callerInstanceId: caller?.id,
    callerName: caller ? formatEntityName(caller.name) : undefined,
    intentText: targetName ? formatPermissionIntent(targetName) : undefined,
    sourceView: "registry",
    targetId: target?.id,
    targetName,
    tenantId: item.tenantId,
    tenantName: formatTenantName(item.tenantId),
    workspaceId: item.workspaceId,
    workspaceName: formatWorkspaceName(item.workspaceId)
  };
}

function sameScope(agent: Agent, item: ResourceLifecycleItem) {
  return agent.tenantId === item.tenantId && agent.workspaceId === item.workspaceId;
}
