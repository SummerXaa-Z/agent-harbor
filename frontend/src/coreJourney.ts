import type {
  Agent,
  Capability,
  ConsoleData,
  TenantAccessProfileData,
  TenantEntitlement,
  TraceEvent,
} from "./types";

export type CoreJourneyStepKey =
  | "tenantTree"
  | "agentPair"
  | "capabilityDiscovery"
  | "grantChain"
  | "runtimeEvidence"
  | "accessProfile";

export type CoreJourneyStepStatus = "complete" | "partial" | "missing";

export interface CoreJourneyForm {
  workspaceId: string;
  mcpEndpoint: string;
  allowedTool: string;
  deniedTool: string;
}

export interface CoreJourneyConfig extends CoreJourneyForm {
  runId: string;
  rootTenantId: string;
  childTenantId: string;
  grandchildTenantId: string;
  subjectSelector: string;
  subjectId: string;
}

export interface CoreJourneyStep {
  key: CoreJourneyStepKey;
  status: CoreJourneyStepStatus;
  metric: string;
  detail: string;
}

export interface CoreJourneyEvaluation {
  completeCount: number;
  totalCount: number;
  steps: CoreJourneyStep[];
  caller?: Agent;
  target?: Agent;
  allowedCapability?: Capability;
  deniedCapability?: Capability;
  tenantEntitlement?: TenantEntitlement;
}

export const defaultCoreJourneyForm: CoreJourneyForm = {
  allowedTool: "search_customer",
  deniedTool: "export_contracts",
  mcpEndpoint: "http://127.0.0.1:8787/mcp",
  workspaceId: "ws-core-journey",
};

export function createCoreJourneyConfig(
  form: CoreJourneyForm = defaultCoreJourneyForm,
  seed: string = Date.now().toString(36),
): CoreJourneyConfig {
  const suffix = safeIdPart(seed) || Date.now().toString(36);
  const runId = `ui-core-${suffix}`;
  return {
    ...form,
    runId,
    rootTenantId: `tenant-root-${runId}`,
    childTenantId: `tenant-child-${runId}`,
    grandchildTenantId: `tenant-grandchild-${runId}`,
    subjectSelector: `user:${runId}-*`,
    subjectId: `user:${runId}-operator`,
  };
}

export function evaluateCoreJourney(
  data: ConsoleData | null,
  accessProfile: TenantAccessProfileData | null,
  config: CoreJourneyConfig,
): CoreJourneyEvaluation {
  const agents = data?.agents ?? [];
  const capabilities = data?.capabilities ?? [];
  const caller = agents.find(
    (agent) =>
      agent.channelType === "local" &&
      agent.tenantId === config.childTenantId &&
      agent.workspaceId === config.workspaceId,
  );
  const target = agents.find(
    (agent) =>
      agent.channelType === "mcp" &&
      agent.tenantId === config.rootTenantId &&
      agent.workspaceId === config.workspaceId &&
      endpointFor(agent) === config.mcpEndpoint,
  );
  const targetCapabilities = target ? capabilities.filter((capability) => capability.targetId === target.id) : [];
  const allowedCapability = targetCapabilities.find((capability) => capability.key === config.allowedTool);
  const deniedCapability = targetCapabilities.find((capability) => capability.key === config.deniedTool);
  const scopedAllowed = Boolean(
    allowedCapability?.discoveryStatus === "approved" &&
      allowedCapability.dataScopes?.some((scope) => scope.tenantFilter?.includes(config.childTenantId)),
  );
  const tenantEntitlement = data?.tenantEntitlements.find(
    (entitlement) =>
      entitlement.tenantId === config.childTenantId &&
      entitlement.targetId === target?.id &&
      entitlement.capabilityId === allowedCapability?.id &&
      entitlement.effect === "allow" &&
      entitlement.status === "enabled",
  );
  const workspaceAssignment = data?.workspaceAssignments.find(
    (assignment) =>
      assignment.tenantEntitlementId === tenantEntitlement?.id &&
      assignment.workspaceId === config.workspaceId &&
      assignment.effect === "allow" &&
      assignment.status === "enabled",
  );
  const instanceAssignment = data?.instanceAssignments.find(
    (assignment) =>
      assignment.workspaceAssignmentId === workspaceAssignment?.id &&
      assignment.callerInstanceId === caller?.id &&
      assignment.effect === "allow" &&
      assignment.status === "enabled",
  );
  const runtimeEvidence = runtimeEvidenceState(data?.traces ?? [], accessProfile, config);
  const tenantTreeStatus = tenantTreeState(accessProfile, config);
  const profileComplete = Boolean(
    accessProfile?.tenant.id === config.childTenantId &&
      accessProfile.summary.grantCount >= 1 &&
      accessProfile.summary.targetCount >= 1 &&
      accessProfile.summary.capabilityCount >= 1 &&
      accessProfile.summary.workspaceAssignmentCount >= 1 &&
      accessProfile.summary.instanceAssignmentCount >= 1 &&
      accessProfile.summary.recentAllowedTraceCount >= 1 &&
      accessProfile.summary.recentDeniedTraceCount >= 1,
  );
  const profilePartial = Boolean(accessProfile?.tenant.id === config.childTenantId);
  const steps: CoreJourneyStep[] = [
    {
      detail: `${config.rootTenantId} -> ${config.childTenantId} -> ${config.grandchildTenantId}`,
      key: "tenantTree",
      metric: tenantTreeStatus === "complete" ? "3" : accessProfile ? String(accessProfile.scopeTenants.length) : "0",
      status: tenantTreeStatus,
    },
    {
      detail: caller && target ? `${caller.name} -> ${target.name}` : config.mcpEndpoint,
      key: "agentPair",
      metric: `${caller ? 1 : 0}/${target ? 1 : 0}`,
      status: caller && target ? "complete" : caller || target ? "partial" : "missing",
    },
    {
      detail: scopedAllowed ? `${config.allowedTool} scoped, ${config.deniedTool} unassigned` : `${config.allowedTool} / ${config.deniedTool}`,
      key: "capabilityDiscovery",
      metric: `${allowedCapability ? 1 : 0}/${deniedCapability ? 1 : 0}`,
      status: allowedCapability && deniedCapability && scopedAllowed ? "complete" : allowedCapability || deniedCapability ? "partial" : "missing",
    },
    {
      detail: `${config.childTenantId} / ${config.workspaceId} / ${caller?.id ?? "-"}`,
      key: "grantChain",
      metric: `${tenantEntitlement ? 1 : 0}/${workspaceAssignment ? 1 : 0}/${instanceAssignment ? 1 : 0}`,
      status: tenantEntitlement && workspaceAssignment && instanceAssignment ? "complete" : tenantEntitlement || workspaceAssignment || instanceAssignment ? "partial" : "missing",
    },
    {
      detail: `${config.runId} allowed=${runtimeEvidence.allowed} denied=${runtimeEvidence.denied}`,
      key: "runtimeEvidence",
      metric: `${runtimeEvidence.allowed}/${runtimeEvidence.denied}`,
      status: runtimeEvidence.allowed > 0 && runtimeEvidence.denied > 0 ? "complete" : runtimeEvidence.allowed > 0 || runtimeEvidence.denied > 0 ? "partial" : "missing",
    },
    {
      detail: accessProfile ? `${accessProfile.tenant.name} grants=${accessProfile.summary.grantCount}` : config.childTenantId,
      key: "accessProfile",
      metric: accessProfile ? String(accessProfile.summary.grantCount) : "0",
      status: profileComplete ? "complete" : profilePartial ? "partial" : "missing",
    },
  ];

  return {
    allowedCapability,
    caller,
    completeCount: steps.filter((step) => step.status === "complete").length,
    deniedCapability,
    steps,
    target,
    tenantEntitlement,
    totalCount: steps.length,
  };
}

function tenantTreeState(
  accessProfile: TenantAccessProfileData | null,
  config: CoreJourneyConfig,
): CoreJourneyStepStatus {
  if (!accessProfile) return "missing";
  const child = accessProfile.scopeTenants.find((tenant) => tenant.id === config.childTenantId);
  const grandchild = accessProfile.scopeTenants.find((tenant) => tenant.id === config.grandchildTenantId);
  if (
    accessProfile.tenant.id === config.childTenantId &&
    accessProfile.tenant.parentTenantId === config.rootTenantId &&
    child?.parentTenantId === config.rootTenantId &&
    grandchild?.parentTenantId === config.childTenantId
  ) {
    return "complete";
  }
  return child || grandchild ? "partial" : "missing";
}

function runtimeEvidenceState(
  traces: TraceEvent[],
  accessProfile: TenantAccessProfileData | null,
  config: CoreJourneyConfig,
) {
  const runTraces = traces.filter((trace) => trace.runId === config.runId);
  const allowedFromTraces = runTraces.filter((trace) => trace.decision === "allowed").length;
  const deniedFromTraces = runTraces.filter((trace) => trace.decision === "denied").length;
  return {
    allowed: Math.max(allowedFromTraces, accessProfile?.summary.recentAllowedTraceCount ?? 0),
    denied: Math.max(deniedFromTraces, accessProfile?.summary.recentDeniedTraceCount ?? 0),
  };
}

function endpointFor(agent: Agent) {
  const endpoint = agent.channelConfig?.endpoint;
  return typeof endpoint === "string" ? endpoint : "";
}

function safeIdPart(value: string) {
  return value.trim().replace(/[^a-zA-Z0-9]+/g, "-").replace(/^-+|-+$/g, "").toLowerCase();
}
