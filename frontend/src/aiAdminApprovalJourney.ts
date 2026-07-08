import type {
  Agent,
  AuditEvent,
  Capability,
  ConsoleData,
  TenantAccessProfileData,
  TraceEvent,
} from "./types";
import type {
  PermissionPackageApplication,
  PermissionPackageApprovalEffectiveStatus,
  PermissionPackageApprovalRequest,
} from "./permissionPackages";

export type AiAdminApprovalJourneyStepKey =
  | "tenantTree"
  | "agentPair"
  | "capabilityDiscovery"
  | "approvalRequest"
  | "approvedApply"
  | "runtimeEvidence"
  | "accessProfile"
  | "auditEvidence";

export type AiAdminApprovalJourneyStepStatus = "complete" | "partial" | "missing";

export interface AiAdminApprovalJourneyConfig {
  runId: string;
  rootTenantId: string;
  childTenantId: string;
  grandchildTenantId: string;
  workspaceId: string;
  mcpEndpoint: string;
  readTool: string;
  writeTool: string;
  deniedTool: string;
  region: string;
  subjectSelector: string;
  subjectId: string;
  templateId: string;
  requestText: string;
}

export interface AiAdminApprovalJourneyResult {
  allowedStatus: number;
  applicationId: string;
  approvalRequestId: string;
  callerId: string;
  deniedStatus: number;
  targetId: string;
  toolListStatus: number;
}

export interface AiAdminApprovalJourneyStep {
  key: AiAdminApprovalJourneyStepKey;
  status: AiAdminApprovalJourneyStepStatus;
  metric: string;
  detail: string;
}

export interface AiAdminApprovalJourneyEvaluation {
  completeCount: number;
  totalCount: number;
  steps: AiAdminApprovalJourneyStep[];
  caller?: Agent;
  target?: Agent;
  readCapability?: Capability;
  writeCapability?: Capability;
  deniedCapability?: Capability;
}

export interface AiAdminGoLiveReadinessSummary {
  status: "ready" | "waiting";
  remainingCount: number;
  totalCount: number;
  nextStep?: AiAdminApprovalJourneyStep;
}

export function createAiAdminApprovalJourneyConfig(
  seed: string = Date.now().toString(36),
): AiAdminApprovalJourneyConfig {
  const suffix = safeIdPart(seed) || Date.now().toString(36);
  const runId = `ui-approval-${suffix}`;
  return {
    runId,
    rootTenantId: `tenant-root-${runId}`,
    childTenantId: `tenant-child-${runId}`,
    grandchildTenantId: `tenant-grandchild-${runId}`,
    workspaceId: "ws-ai-admin-approval",
    mcpEndpoint: "http://127.0.0.1:8787/mcp",
    readTool: "search_customer",
    writeTool: "update_ticket",
    deniedTool: "export_contracts",
    region: "us-east",
    subjectSelector: "user:support-*",
    subjectId: "user:support-001",
    templateId: "support-ticket-triage",
    requestText: "Allow support triage reads and bounded ticket updates for this tenant.",
  };
}

export function evaluateAiAdminApprovalJourney({
  accessProfile,
  application,
  approvalRequest,
  auditEvent,
  config,
  data,
  result,
}: {
  accessProfile: TenantAccessProfileData | null;
  application: PermissionPackageApplication | null;
  approvalRequest: PermissionPackageApprovalRequest | null;
  auditEvent: AuditEvent | null;
  config: AiAdminApprovalJourneyConfig;
  data: ConsoleData | null;
  result: AiAdminApprovalJourneyResult | null;
}): AiAdminApprovalJourneyEvaluation {
  const approvalRequestEffectiveStatus = approvalRequest ? approvalEffectiveStatus(approvalRequest) : null;
  const agents = data?.agents ?? [];
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
  const targetCapabilities = target
    ? (data?.capabilities ?? []).filter((capability) => capability.targetId === target.id)
    : [];
  const readCapability = targetCapabilities.find((capability) => capability.key === config.readTool);
  const writeCapability = targetCapabilities.find((capability) => capability.key === config.writeTool);
  const deniedCapability = targetCapabilities.find((capability) => capability.key === config.deniedTool);
  const scopedAllowedCount = [readCapability, writeCapability].filter((capability) =>
    Boolean(
      capability?.discoveryStatus === "approved" &&
        capability.dataScopes?.some((scope) => scope.tenantFilter?.includes(config.childTenantId)),
    ),
  ).length;
  const runtimeRecord = runtimeRecordState(data?.traces ?? [], accessProfile, config);
  const profileComplete = Boolean(
    accessProfile?.tenant.id === config.childTenantId &&
      accessProfile.summary.grantCount >= 2 &&
      accessProfile.summary.targetCount >= 1 &&
      accessProfile.summary.capabilityCount >= 2 &&
      accessProfile.summary.workspaceAssignmentCount >= 2 &&
      accessProfile.summary.instanceAssignmentCount >= 2 &&
      accessProfile.summary.recentAllowedTraceCount >= 1 &&
      accessProfile.summary.recentDeniedTraceCount >= 1,
  );
  const auditComplete = Boolean(
    auditEvent?.action === "permission_package.applied" &&
      auditEvent.resourceId === application?.id &&
      auditEvent.metadata?.applicationId === application?.id &&
      auditEvent.metadata?.approvalRequestId === approvalRequest?.id,
  );
  const appliedComplete = Boolean(
    application &&
      result?.applicationId === application.id &&
      result.approvalRequestId === approvalRequest?.id &&
      application.templateId === config.templateId &&
      application.allowedCapabilityKeys.includes(config.readTool) &&
      application.allowedCapabilityKeys.includes(config.writeTool) &&
      !application.allowedCapabilityKeys.includes(config.deniedTool),
  );
  const runtimeStatusComplete = Boolean(
    result &&
      result.toolListStatus >= 200 &&
      result.toolListStatus < 300 &&
      result.deniedStatus === 403 &&
      result.allowedStatus >= 200 &&
      result.allowedStatus < 300 &&
      runtimeRecord.allowed > 0 &&
      runtimeRecord.denied > 0,
  );
  const steps: AiAdminApprovalJourneyStep[] = [
    {
      detail: `${config.rootTenantId} -> ${config.childTenantId} -> ${config.grandchildTenantId}`,
      key: "tenantTree",
      metric: accessProfile ? String(accessProfile.scopeTenants.length + 1) : "0",
      status: tenantTreeState(accessProfile, config),
    },
    {
      detail: caller && target ? `${caller.name} -> ${target.name}` : config.mcpEndpoint,
      key: "agentPair",
      metric: `${caller ? 1 : 0}/${target ? 1 : 0}`,
      status: caller && target ? "complete" : caller || target ? "partial" : "missing",
    },
    {
      detail: scopedAllowedCount === 2
        ? `${config.readTool}, ${config.writeTool} scoped; ${config.deniedTool} blocked`
        : `${config.readTool} / ${config.writeTool} / ${config.deniedTool}`,
      key: "capabilityDiscovery",
      metric: `${[readCapability, writeCapability, deniedCapability].filter(Boolean).length}/3`,
      status: readCapability && writeCapability && deniedCapability && scopedAllowedCount === 2
        ? "complete"
        : readCapability || writeCapability || deniedCapability
          ? "partial"
          : "missing",
    },
    {
      detail: approvalRequest?.id ?? config.templateId,
      key: "approvalRequest",
      metric: approvalRequestEffectiveStatus ?? "0",
      status: approvalRequestEffectiveStatus === "approved" ? "complete" : approvalRequest ? "partial" : "missing",
    },
    {
      detail: application?.id ?? config.templateId,
      key: "approvedApply",
      metric: String(application?.allowedCapabilityIds.length ?? 0),
      status: appliedComplete ? "complete" : application ? "partial" : "missing",
    },
    {
      detail: `${config.runId} allowed=${runtimeRecord.allowed} denied=${runtimeRecord.denied}`,
      key: "runtimeEvidence",
      metric: result ? `${result.toolListStatus}/${result.deniedStatus}/${result.allowedStatus}` : `${runtimeRecord.allowed}/${runtimeRecord.denied}`,
      status: runtimeStatusComplete ? "complete" : runtimeRecord.allowed > 0 || runtimeRecord.denied > 0 || result ? "partial" : "missing",
    },
    {
      detail: accessProfile ? `${accessProfile.tenant.name} grants=${accessProfile.summary.grantCount}` : config.childTenantId,
      key: "accessProfile",
      metric: accessProfile ? String(accessProfile.summary.grantCount) : "0",
      status: profileComplete ? "complete" : accessProfile ? "partial" : "missing",
    },
    {
      detail: auditEvent?.id ?? "permission_package.applied",
      key: "auditEvidence",
      metric: auditComplete ? "1" : "0",
      status: auditComplete ? "complete" : auditEvent ? "partial" : "missing",
    },
  ];
  return {
    caller,
    completeCount: steps.filter((step) => step.status === "complete").length,
    deniedCapability,
    readCapability,
    steps,
    target,
    totalCount: steps.length,
    writeCapability,
  };
}

export function summarizeAiAdminGoLiveReadiness(
  evaluation: AiAdminApprovalJourneyEvaluation,
): AiAdminGoLiveReadinessSummary {
  const incompleteSteps = evaluation.steps.filter((step) => step.status !== "complete");
  return {
    nextStep: incompleteSteps[0],
    remainingCount: incompleteSteps.length,
    status: incompleteSteps.length === 0 ? "ready" : "waiting",
    totalCount: evaluation.totalCount,
  };
}

function tenantTreeState(
  accessProfile: TenantAccessProfileData | null,
  config: AiAdminApprovalJourneyConfig,
): AiAdminApprovalJourneyStepStatus {
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

function runtimeRecordState(
  traces: TraceEvent[],
  accessProfile: TenantAccessProfileData | null,
  config: AiAdminApprovalJourneyConfig,
) {
  const runTraces = traces.filter((trace) => trace.runId === config.runId);
  return {
    allowed: Math.max(
      runTraces.filter((trace) => trace.decision === "allowed").length,
      accessProfile?.summary.recentAllowedTraceCount ?? 0,
    ),
    denied: Math.max(
      runTraces.filter((trace) => trace.decision === "denied").length,
      accessProfile?.summary.recentDeniedTraceCount ?? 0,
    ),
  };
}

function approvalEffectiveStatus(request: PermissionPackageApprovalRequest): PermissionPackageApprovalEffectiveStatus {
  return request.effectiveStatus ?? request.status;
}

function endpointFor(agent: Agent) {
  const endpoint = agent.channelConfig?.endpoint;
  return typeof endpoint === "string" ? endpoint : "";
}

function safeIdPart(value: string) {
  return value.trim().replace(/[^a-zA-Z0-9]+/g, "-").replace(/^-+|-+$/g, "").toLowerCase();
}
