import { permissionPackageApprovalEffectiveStatus } from "../../permissionPackages.ts";
import type {
  PermissionPackageApplication,
  PermissionPackageApprovalEffectiveStatus,
  PermissionPackageApprovalRequest,
} from "../../permissionPackages.ts";
import type {
  Agent,
  AgentKey,
  Capability,
  ConsoleSession,
  DataScope,
  TenantAccessProfile,
  TraceEvent,
} from "../../types.ts";

// Demo mode runs every request as this actor, so "my requests" still filters.
export const DEMO_ACTOR = "local-dev";

export function workbenchActor(session: ConsoleSession | null): string {
  const actor = session?.actor?.trim() ?? "";
  if (actor) return actor;
  return session?.requiresLogin === false ? DEMO_ACTOR : "";
}

export const onboardingStepKeys = ["connect", "register", "authorize", "handoff"] as const;
export type OnboardingStepKey = (typeof onboardingStepKeys)[number];

export interface OnboardingProgress {
  doneCount: number;
  nextKey: OnboardingStepKey | null;
  steps: { done: boolean; key: OnboardingStepKey }[];
  total: number;
}

export function onboardingProgress({
  agents,
  applications,
  capabilities,
  keys,
  liveData,
  now = Date.now(),
}: {
  agents: readonly Agent[];
  applications: readonly PermissionPackageApplication[];
  capabilities: readonly Capability[];
  keys: readonly AgentKey[];
  liveData: boolean;
  now?: number;
}): OnboardingProgress {
  const targetIds = new Set(capabilities.map((capability) => capability.targetId));
  const done: Record<OnboardingStepKey, boolean> = {
    connect: liveData,
    register: agents.some((agent) => agent.channelType === "local" && agent.status === "active")
      && agents.some((agent) => targetIds.has(agent.id) && agent.status === "active"),
    authorize: applications.length > 0,
    handoff: keys.some((key) => Boolean(key.createdForHandoffId) && keyStatus(key, now) === "active"),
  };
  const steps = onboardingStepKeys.map((key) => ({ done: done[key], key }));
  return {
    doneCount: steps.filter((step) => step.done).length,
    nextKey: steps.find((step) => !step.done)?.key ?? null,
    steps,
    total: steps.length,
  };
}

export type MyRequestGroup = "pending" | "approved" | "rejected";

export interface MyRequestRow {
  allowedCount: number;
  blockedCount: number;
  group: MyRequestGroup;
  request: PermissionPackageApprovalRequest;
  status: PermissionPackageApprovalEffectiveStatus;
}

const groupForStatus: Partial<Record<PermissionPackageApprovalEffectiveStatus, MyRequestGroup>> = {
  approved: "approved",
  pending: "pending",
  rejected: "rejected",
};

// Only requests this actor filed, newest first; withdrawn and expired ones
// drop out because they need no follow-up.
export function myRequests(
  requests: readonly PermissionPackageApprovalRequest[],
  actor: string,
  capabilities: readonly Capability[],
): MyRequestRow[] {
  const normalizedActor = actor.trim();
  if (!normalizedActor) return [];
  return requests
    .filter((request) => (request.requestedBy ?? "").trim() === normalizedActor)
    .map((request) => {
      const status = permissionPackageApprovalEffectiveStatus(request);
      const targetCount = capabilities.filter((capability) => capability.targetId === request.targetId).length;
      const allowedCount = request.allowedCapabilityIds.length;
      return {
        allowedCount,
        blockedCount: Math.max(targetCount - allowedCount, 0),
        group: groupForStatus[status],
        request,
        status,
      };
    })
    .filter((row): row is MyRequestRow => row.group !== undefined)
    .sort((a, b) => Date.parse(b.request.createdAt) - Date.parse(a.request.createdAt));
}

export function myRequestCounts(rows: readonly MyRequestRow[]): Record<MyRequestGroup, number> {
  const counts: Record<MyRequestGroup, number> = { approved: 0, pending: 0, rejected: 0 };
  for (const row of rows) counts[row.group] += 1;
  return counts;
}

export type ResourceRole = "caller" | "target";

export interface MyResourceRow {
  agent: Agent;
  lastActivity: string;
  role: ResourceRole;
}

export function myResources(
  agents: readonly Agent[],
  capabilities: readonly Capability[],
  traces: readonly TraceEvent[],
): MyResourceRow[] {
  const targetIds = new Set(capabilities.map((capability) => capability.targetId));
  return agents
    .filter((agent) => agent.channelType === "local" || targetIds.has(agent.id))
    .map((agent) => {
      const role: ResourceRole = agent.channelType === "local" ? "caller" : "target";
      const latestTrace = traces
        .filter((trace) => (role === "caller"
          ? trace.callerInstanceId === agent.id || trace.callerAgentId === agent.id
          : trace.targetAgentId === agent.id))
        .reduce<string>((latest, trace) => (Date.parse(trace.createdAt) > Date.parse(latest || "0") ? trace.createdAt : latest), "");
      return { agent, lastActivity: latestTrace || agent.updatedAt, role };
    })
    .sort((a, b) => (a.role === b.role ? a.agent.name.localeCompare(b.agent.name) : a.role === "caller" ? -1 : 1));
}

export type KeyStatus = "active" | "expired" | "revoked";

export function keyStatus(key: Pick<AgentKey, "expiresAt" | "revokedAt">, now = Date.now()): KeyStatus {
  if (key.revokedAt) return "revoked";
  return Date.parse(key.expiresAt) > now ? "active" : "expired";
}

export interface PermissionRow {
  approvalId: string;
  capability: Capability;
  dataScopes: DataScope[];
  decision: "allowed" | "blocked";
  id: string;
}

export interface MyPermissions {
  kpis: { activeTokens: number; allowed: number; blocked: number; historicalTokens: number };
  rows: PermissionRow[];
}

// Rows come from the caller-filtered access profile: a capability counts as
// allowed only when every layer allows, is enabled and stays inside its
// parent's data scope. Other capabilities on the same targets are blocked.
// The approval ID is the approved request consumed by the application that
// created the caller assignment.
export function myPermissions({
  approvals,
  applications,
  callerInstanceId,
  capabilities,
  keys,
  now = Date.now(),
  profile,
}: {
  approvals: readonly PermissionPackageApprovalRequest[];
  applications: readonly PermissionPackageApplication[];
  callerInstanceId: string;
  capabilities: readonly Capability[];
  keys: readonly AgentKey[];
  now?: number;
  profile: TenantAccessProfile | null;
}): MyPermissions {
  const allowedRows = new Map<string, PermissionRow>();
  const grantedTargets = new Set<string>();
  for (const grant of profile?.grants ?? []) {
    const entitlement = grant.tenantEntitlement;
    for (const workspace of grant.workspaceAssignments ?? []) {
      for (const instance of workspace.instanceAssignments ?? []) {
        const assignment = instance.instanceAssignment;
        if (assignment.callerInstanceId !== callerInstanceId) continue;
        grantedTargets.add(entitlement.targetId);
        const capability = grant.capability ?? capabilities.find((item) => item.id === entitlement.capabilityId);
        if (!capability) continue;
        const allowed = [entitlement, workspace.workspaceAssignment, assignment].every((layer) => layer.effect === "allow" && layer.status === "enabled")
          && [grant.scopeStatus, workspace.scopeStatus, instance.scopeStatus].every((status) => status === "valid")
          && capability.discoveryStatus === "approved";
        if (!allowed || allowedRows.has(capability.id)) continue;
        const application = applications.find((item) => item.instanceAssignmentIds.includes(assignment.id));
        const approval = application ? approvals.find((item) => item.consumedByApplicationId === application.id) : undefined;
        allowedRows.set(capability.id, {
          approvalId: approval?.id ?? "",
          capability,
          dataScopes: instance.effectiveInstanceDataScopes ?? assignment.dataScopes ?? [],
          decision: "allowed",
          id: capability.id,
        });
      }
    }
  }
  const blockedRows: PermissionRow[] = capabilities
    .filter((capability) => grantedTargets.has(capability.targetId) && !allowedRows.has(capability.id))
    .map((capability) => ({ approvalId: "", capability, dataScopes: capability.dataScopes ?? [], decision: "blocked", id: capability.id }));
  const callerKeys = keys.filter((key) => key.agentId === callerInstanceId);
  const activeTokens = callerKeys.filter((key) => keyStatus(key, now) === "active").length;
  const rows = [...allowedRows.values(), ...blockedRows];
  return {
    kpis: {
      activeTokens,
      allowed: allowedRows.size,
      blocked: blockedRows.length,
      historicalTokens: callerKeys.length - activeTokens,
    },
    rows,
  };
}
