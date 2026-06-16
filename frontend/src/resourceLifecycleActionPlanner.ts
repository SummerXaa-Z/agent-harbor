import type { NavKey } from "./consoleNavigation";
import type { ResourceLifecycleItem } from "./resourceLifecycle";
import type { Agent, PermissionChangeHandoffContext, TraceFilters } from "./types";

export type ResourceLifecycleModal = "rotate_credential" | "create_policy" | "create_key";

export type ResourceLifecycleActionPlan =
  | { agentId: string; kind: "open_modal"; modal: ResourceLifecycleModal }
  | { kind: "capability_prefill"; navKey: "capabilities"; targetId: string }
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
  if (item.nextActionKind === "rotate_credential") {
    return { agentId: item.id, kind: "open_modal", modal: "rotate_credential" };
  }
  if (item.nextActionKind === "review_capabilities") {
    return { kind: "capability_prefill", navKey: "capabilities", targetId: item.id };
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
  const caller = item.kind === "caller" ? resourceAgent : sameScopeCaller ?? localCallers[0];
  const target = item.kind === "caller" ? sameScopeTarget ?? mcpTargets[0] : resourceAgent;
  const targetName = target ? formatEntityName(target.name) : item.name;

  return {
    callerInstanceId: caller?.id,
    callerName: caller ? formatEntityName(caller.name) : undefined,
    intentText: formatPermissionIntent(targetName),
    sourceView: "registry",
    targetId: target?.id ?? item.id,
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
