import type {
  Capability,
  CapabilityAction,
  CapabilityRisk,
  CapabilitySensitivity,
  DataScope,
  InstanceAssignment,
  TenantEntitlement,
  WorkspaceAssignment,
  AuditEvent,
  TenantAccessProfile,
  TraceEvent,
} from "./types";

export type PermissionPackageDecision = "allow" | "deny";
export type PermissionPackagePolicyDecision = "allow" | "approval_required";
export type PermissionPackageApprovalStatus = "pending" | "approved" | "rejected" | "withdrawn";

export interface PermissionPackageTemplate {
  id: string;
  version: number;
  name: string;
  summary: string;
  allowedActions: CapabilityAction[];
  blockedActions: CapabilityAction[];
  blockedRisks: CapabilityRisk[];
  blockedSensitivities: CapabilitySensitivity[];
  defaultDataDomain: string;
  guardrails: PermissionPackageGuardrail[];
}

export interface PermissionPackageGuardrail {
  capabilityKey: string;
  expectedDecision: PermissionPackageDecision;
  reason: string;
  reasonKey: string;
}

export interface PermissionPackageDraftInput {
  callerInstanceId: string;
  region: string;
  requestText: string;
  subjectSelector?: string;
  targetId: string;
  templateId: string;
  tenantId: string;
  workspaceId: string;
}

export interface PermissionPackageApplyInput extends PermissionPackageDraftInput {
  approvalRequestId?: string;
}

export interface PermissionPackageDraft {
  id: string;
  input: PermissionPackageDraftInput;
  template: PermissionPackageTemplate;
  allowedCapabilities: Capability[];
  blockedCapabilities: Capability[];
  dataScopes: DataScope[];
  readiness: PermissionPackageReadiness;
  policyGate: PermissionPackagePolicyGate;
  simulationRows: PermissionPackageSimulationRow[];
}

export interface PermissionPackageApplyResult {
  draft: PermissionPackageDraft;
  tenantEntitlements: TenantEntitlement[];
  workspaceAssignments: WorkspaceAssignment[];
  instanceAssignments: InstanceAssignment[];
  application?: PermissionPackageApplication;
}

export type PermissionPackageApplyPreflightSeverity = "passed" | "info" | "warning" | "blocking";

export interface PermissionPackageApplyPreflight {
  draft: PermissionPackageDraft;
  summary: PermissionPackageApplyPreflightSummary;
  checks: PermissionPackageApplyPreflightCheck[];
  planned: PermissionPackageApplyPreflightPlannedChanges;
  existingGrants: PermissionPackageApplyPreflightExistingGrant[];
  nextActions: string[];
}

export interface PermissionPackageApplyPreflightSummary {
  canApply: boolean;
  blockingCount: number;
  warningCount: number;
  plannedCapabilityCount: number;
  plannedTenantEntitlementCount: number;
  plannedWorkspaceAssignmentCount: number;
  plannedInstanceAssignmentCount: number;
  existingGrantCount: number;
  requiresApproval: boolean;
  approvalReady: boolean;
}

export interface PermissionPackageApplyPreflightCheck {
  code: string;
  severity: PermissionPackageApplyPreflightSeverity;
  message: string;
  capabilityId?: string;
  capabilityKey?: string;
}

export interface PermissionPackageApplyPreflightPlannedChanges {
  capabilities: Capability[];
  tenantEntitlements: TenantEntitlement[];
  workspaceAssignments: WorkspaceAssignment[];
  instanceAssignments: InstanceAssignment[];
}

export interface PermissionPackageApplyPreflightExistingGrant {
  capabilityId: string;
  capabilityKey: string;
  tenantEntitlementId: string;
  workspaceAssignmentId: string;
  instanceAssignmentId: string;
}

export interface PermissionPackageApplication {
  id: string;
  draftId: string;
  templateId: string;
  templateVersion: number;
  tenantId: string;
  workspaceId: string;
  targetId: string;
  callerInstanceId: string;
  subjectSelector?: string;
  requestText?: string;
  region?: string;
  dataScopes?: DataScope[];
  allowedCapabilityIds: string[];
  allowedCapabilityKeys: string[];
  tenantEntitlementIds: string[];
  workspaceAssignmentIds: string[];
  instanceAssignmentIds: string[];
  appliedAt: string;
}

export interface PermissionPackageApplicationImpact {
  application: PermissionPackageApplication;
  summary: PermissionPackageImpactSummary;
  createdObjects: PermissionPackageImpactObject[];
  capabilityReviews: PermissionPackageImpactCapability[];
  rollbackReview: PermissionPackageRollbackReview;
  remediationPlan: PermissionPackageRemediationPlan;
  rehearsal?: PermissionPackageImpactRehearsal;
}

export type PermissionPackageApplicationHealthStatus = "ready" | "drifted" | "needs_review";

export interface PermissionPackageApplicationHealth {
  summary: PermissionPackageApplicationHealthSummary;
  applications: PermissionPackageApplicationHealthRow[];
}

export interface PermissionPackageApplicationHealthSummary {
  total: number;
  ready: number;
  drifted: number;
  needsReview: number;
}

export interface PermissionPackageApplicationHealthRow {
  application: PermissionPackageApplication;
  status: PermissionPackageApplicationHealthStatus;
  blockerCodes?: string[];
  createdObjectCount: number;
  activeObjectCount: number;
  missingObjectCount: number;
  rollbackReady: boolean;
}

export type PermissionPackageProductionReadinessStatus = "ready" | "needs_review" | "blocked";

export interface PermissionPackageProductionReadinessFilter {
  approvalRequestId?: string;
  callerInstanceId: string;
  region?: string;
  requestText?: string;
  subjectId?: string;
  subjectSelector?: string;
  targetId: string;
  templateId: string;
  tenantId: string;
  traceLimit?: number;
  workspaceId: string;
}

export interface PermissionPackageProductionReadiness {
  status: PermissionPackageProductionReadinessStatus;
  summary: PermissionPackageProductionReadinessSummary;
  checks: PermissionPackageProductionReadinessCheck[];
  latestApplication?: PermissionPackageApplication;
  preflight?: PermissionPackageApplyPreflight;
  applicationHealth?: PermissionPackageApplicationHealthRow;
  applicationImpact?: PermissionPackageApplicationImpact;
  accessProfile?: TenantAccessProfile;
  runtimeEvidence: PermissionPackageRuntimeEvidence;
  auditEvidence: PermissionPackageAuditEvidence;
  nextActionCode?: PermissionPackageProductionNextActionCode;
  nextActions: string[];
  generatedAt: string;
}

export type PermissionPackageProductionNextActionCode =
  | "resolve_preflight_blockers"
  | "apply_permission_package"
  | "review_application_scope"
  | "review_application_health"
  | "resolve_impact_blockers"
  | "verify_access_profile"
  | "run_allowed_runtime_call"
  | "run_denied_runtime_call"
  | "verify_applied_audit"
  | "export_production_evidence";

export interface PermissionPackageProductionReadinessSummary {
  readyCount: number;
  warningCount: number;
  blockingCount: number;
  hasApplication: boolean;
  hasAllowedTrace: boolean;
  hasDeniedTrace: boolean;
  hasAppliedAudit: boolean;
  accessProfileReady: boolean;
}

export interface PermissionPackageProductionReadinessCheck {
  code: string;
  severity: PermissionPackageApplyPreflightSeverity;
  message: string;
  evidenceId?: string;
}

export interface PermissionPackageRuntimeEvidence {
  allowedTrace?: TraceEvent;
  deniedTrace?: TraceEvent;
}

export interface PermissionPackageAuditEvidence {
  appliedEvent?: AuditEvent;
}

export interface PermissionPackageProductionEvidenceReport {
  reportVersion: string;
  generatedAt: string;
  scope: PermissionPackageProductionEvidenceScope;
  status: PermissionPackageProductionReadinessStatus;
  summary: PermissionPackageProductionReadinessSummary;
  checks: PermissionPackageProductionReadinessCheck[];
  evidence: PermissionPackageProductionEvidenceRefs;
  nextActionCode?: PermissionPackageProductionNextActionCode;
  nextActions: string[];
  readinessGeneratedAt: string;
}

export interface PermissionPackageProductionEvidenceScope {
  tenantId: string;
  workspaceId: string;
  templateId: string;
  targetId: string;
  callerInstanceId: string;
  subjectId?: string;
  region?: string;
  subjectSelector?: string;
}

export interface PermissionPackageProductionEvidenceRefs {
  application: PermissionPackageProductionApplicationEvidence;
  runtime: PermissionPackageProductionRuntimeEvidence;
  audit: PermissionPackageProductionAuditEvidence;
  accessProfile: PermissionPackageProductionEvidenceState;
  applicationHealth: PermissionPackageProductionEvidenceState;
  applicationImpact: PermissionPackageProductionEvidenceState;
}

export interface PermissionPackageProductionApplicationEvidence {
  present: boolean;
  id?: string;
  draftId?: string;
  templateVersion?: number;
  appliedAt?: string;
  allowedCapabilityIds?: string[];
  allowedCapabilityKeys?: string[];
  dataScopes?: DataScope[];
}

export interface PermissionPackageProductionRuntimeEvidence {
  allowedTraceId?: string;
  deniedTraceId?: string;
}

export interface PermissionPackageProductionAuditEvidence {
  appliedEventId?: string;
}

export interface PermissionPackageProductionEvidenceState {
  present: boolean;
  status?: string;
}

export type PermissionPackageWorkbenchStatus =
  | "needs_input"
  | "awaiting_approval"
  | "ready_to_apply"
  | "validating"
  | "production_ready"
  | "blocked";

export type PermissionPackageWorkbenchActionCode =
  | "complete_request"
  | "create_approval_request"
  | "review_approval_request"
  | "apply_permission_package"
  | "run_runtime_validation"
  | "export_production_evidence";

export type PermissionPackageWorkbenchStepKey =
  | "request"
  | "approval"
  | "apply"
  | "validation"
  | "acceptance";

export type PermissionPackageWorkbenchStepStatus = "complete" | "current" | "waiting" | "blocked";

export interface PermissionPackageWorkbenchPreview {
  draft: PermissionPackageDraft;
  approvalRequest?: PermissionPackageApprovalRequest;
  latestApplication?: PermissionPackageApplication;
  productionReadiness?: PermissionPackageProductionReadiness;
  summary: PermissionPackageWorkbenchSummary;
  generatedAt: string;
}

export interface PermissionPackageWorkbenchSummary {
  status: PermissionPackageWorkbenchStatus;
  primaryActionCode: PermissionPackageWorkbenchActionCode;
  nextActionCode?: PermissionPackageProductionNextActionCode;
  approvalRequired: boolean;
  canApply: boolean;
  applied: boolean;
  runtimeEvidenceReady: boolean;
  productionReady: boolean;
  allowedCapabilityCount: number;
  blockedCapabilityCount: number;
  plannedObjectCount: number;
  readinessReadyCount: number;
  readinessTotalCount: number;
  blockingCount: number;
  warningCount: number;
  steps: PermissionPackageWorkbenchStep[];
}

export interface PermissionPackageWorkbenchStep {
  key: PermissionPackageWorkbenchStepKey;
  status: PermissionPackageWorkbenchStepStatus;
  detailCode: string;
  count?: number;
  total?: number;
}

export interface PermissionPackageImpactRehearsal {
  enabled: boolean;
  scenario?: string;
}

export interface PermissionPackageImpactSummary {
  createdObjectCount: number;
  activeObjectCount: number;
  missingObjectCount: number;
  rollbackReady: boolean;
}

export interface PermissionPackageImpactObject {
  id: string;
  type: string;
  currentStatus: string;
  rollbackAction: string;
  dataScopes?: DataScope[];
}

export interface PermissionPackageImpactCapability {
  id: string;
  key?: string;
  currentStatus: string;
  rollbackAction: string;
}

export interface PermissionPackageRollbackReview {
  ready: boolean;
  blockers: string[];
  blockerCodes?: string[];
  steps: string[];
}

export interface PermissionPackageRemediationPlan {
  executionMode: string;
  ready: boolean;
  blockers: string[];
  blockerCodes?: string[];
  actions: PermissionPackageRemediationAction[];
}

export interface PermissionPackageRemediationAction {
  id: string;
  order: number;
  targetType: string;
  targetId: string;
  action: string;
  currentStatus?: string;
  reason: string;
  readOnly: boolean;
}

export type PermissionPackageApprovalEffectiveStatus = PermissionPackageApprovalStatus | "expired";

export interface PermissionPackageApprovalRequest {
  id: string;
  draftId: string;
  templateId: string;
  templateVersion: number;
  policyVersion: number;
  tenantId: string;
  workspaceId: string;
  targetId: string;
  callerInstanceId: string;
  subjectSelector?: string;
  requestText?: string;
  region?: string;
  dataScopes?: DataScope[];
  allowedCapabilityIds: string[];
  allowedCapabilityKeys: string[];
  allowedCapabilityFingerprints: string[];
  policyGate: PermissionPackagePolicyGate;
  status: PermissionPackageApprovalStatus;
  effectiveStatus?: PermissionPackageApprovalEffectiveStatus;
  isExpired?: boolean;
  requestedBy?: string;
  reviewedBy?: string;
  reviewComment?: string;
  createdAt: string;
  updatedAt: string;
  resolvedAt?: string;
  expiresAt: string;
  consumedAt?: string;
  consumedByApplicationId?: string;
}

export function permissionPackageApprovalEffectiveStatus(
  request: PermissionPackageApprovalRequest
): PermissionPackageApprovalEffectiveStatus {
  return request.effectiveStatus ?? request.status;
}

export interface PermissionPackageReadiness {
  canApply: boolean;
  missingFields: string[];
  warnings: string[];
}

export interface PermissionPackagePolicyGate {
  decision: PermissionPackagePolicyDecision;
  canApplyDirectly: boolean;
  policyVersion: number;
  reasons: PermissionPackagePolicyReason[];
  nextActions: string[];
}

export interface PermissionPackagePolicyReason {
  id: string;
  capabilityId?: string;
  capabilityKey?: string;
  severity: string;
  message: string;
  reasonKey?: string;
  reasonValues?: Record<string, string>;
}

export interface PermissionPackageSimulationRow {
  id: string;
  capabilityId?: string;
  capabilityKey: string;
  expectedDecision: PermissionPackageDecision;
  reason: string;
  reasonKey?: string;
  reasonValues?: Record<string, string>;
}

export interface PermissionPackageInventory {
  capabilities: Capability[];
}

const policyGateVersion = 1;

export const permissionPackageTemplates: PermissionPackageTemplate[] = [
  {
    id: "sales-readonly",
    version: 1,
    name: "Sales read-only",
    summary: "Allow CRM reads for a scoped sales tenant while blocking exports, deletes, admin actions, and restricted data.",
    allowedActions: ["read"],
    blockedActions: ["export", "delete", "admin"],
    blockedRisks: ["high", "critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "crm",
    guardrails: [
      {
        capabilityKey: "cross-region-data",
        expectedDecision: "deny",
        reason: "Data scope keeps the package inside the selected region.",
        reasonKey: "permissionSimulation.guardrailRegion",
      },
      {
        capabilityKey: "sensitive-finance-fields",
        expectedDecision: "deny",
        reason: "Sales read-only does not include finance field access.",
        reasonKey: "permissionSimulation.guardrailFinance",
      },
    ],
  },
  {
    id: "support-ticket-triage",
    version: 1,
    name: "Support ticket triage",
    summary: "Allow ticket reads and bounded updates while blocking exports, deletes, and admin operations.",
    allowedActions: ["read", "write"],
    blockedActions: ["export", "delete", "admin"],
    blockedRisks: ["critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "support",
    guardrails: [
      {
        capabilityKey: "delete-ticket",
        expectedDecision: "deny",
        reason: "Support triage does not include destructive operations.",
        reasonKey: "permissionSimulation.guardrailDeleteTicket",
      },
    ],
  },
  {
    id: "analytics-sandbox",
    version: 1,
    name: "Analytics sandbox",
    summary: "Allow read and execute capabilities for sandbox analysis while blocking writes, exports, and production admin actions.",
    allowedActions: ["read", "execute"],
    blockedActions: ["write", "export", "delete", "admin"],
    blockedRisks: ["high", "critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "analytics",
    guardrails: [
      {
        capabilityKey: "production-write",
        expectedDecision: "deny",
        reason: "Analytics sandbox cannot write production data.",
        reasonKey: "permissionSimulation.guardrailProductionWrite",
      },
    ],
  },
  {
    id: "audit-readonly",
    version: 1,
    name: "Audit read-only",
    summary: "Allow low-risk reads for audit review while blocking mutations, exports, and restricted data.",
    allowedActions: ["read"],
    blockedActions: ["write", "export", "delete", "admin"],
    blockedRisks: ["high", "critical"],
    blockedSensitivities: ["restricted"],
    defaultDataDomain: "audit",
    guardrails: [
      {
        capabilityKey: "audit-export",
        expectedDecision: "deny",
        reason: "Audit read-only requires a separate export approval.",
        reasonKey: "permissionSimulation.guardrailAuditExport",
      },
    ],
  },
];

export const defaultPermissionPackageDraftInput: PermissionPackageDraftInput = {
  callerInstanceId: "",
  region: "华东",
  requestText: "给客服助手开通当前租户的工单查询和有限更新权限，禁止导出合同、删除工单和管理操作。",
  subjectSelector: "user:support-*",
  targetId: "",
  templateId: "support-ticket-triage",
  tenantId: "default",
  workspaceId: "workspace-sandbox"
};

export function subjectIdExampleFromSelector(subjectSelector?: string): string | undefined {
  const selector = subjectSelector?.trim() ?? "";
  if (!selector) return undefined;
  if (!selector.endsWith("*")) return selector;
  return `${selector.slice(0, -1)}example`;
}

export function createPermissionPackageDraft(
  input: PermissionPackageDraftInput,
  inventory: PermissionPackageInventory,
): PermissionPackageDraft {
  const template = permissionPackageTemplates.find((item) => item.id === input.templateId) ?? permissionPackageTemplates[0];
  const targetCapabilities = inventory.capabilities.filter((capability) => capability.targetId === input.targetId);
  const blockedCapabilities = targetCapabilities.filter((capability) => isBlockedByTemplate(capability, template));
  const allowedCapabilities = targetCapabilities.filter(
    (capability) => template.allowedActions.includes(capability.action) && !isBlockedByTemplate(capability, template),
  );
  const dataScopes = buildDraftDataScopes(input, template, allowedCapabilities);
  const readiness = buildReadiness(input, allowedCapabilities);
  const policyGate = buildPolicyGate(allowedCapabilities);
  return {
    allowedCapabilities,
    blockedCapabilities,
    dataScopes,
    id: draftId(input),
    input,
    policyGate,
    readiness,
    simulationRows: buildSimulationRows(allowedCapabilities, blockedCapabilities, template),
    template,
  };
}

function isBlockedByTemplate(capability: Capability, template: PermissionPackageTemplate): boolean {
  return (
    template.blockedActions.includes(capability.action) ||
    template.blockedRisks.includes(capability.riskLevel) ||
    template.blockedSensitivities.includes(capability.sensitivity)
  );
}

function buildDraftDataScopes(
  input: PermissionPackageDraftInput,
  template: PermissionPackageTemplate,
  allowedCapabilities: Capability[],
): DataScope[] {
  const dataDomain =
    allowedCapabilities.flatMap((capability) => capability.dataDomains ?? [])[0] ||
    allowedCapabilities.flatMap((capability) => capability.dataScopes ?? []).find((scope) => scope.dataDomain)?.dataDomain ||
    template.defaultDataDomain;
  return [
    {
      dataDomain,
      region: input.region.trim() || undefined,
      tenantFilter: input.tenantId.trim() ? `tenant_id = '${input.tenantId.trim()}'` : undefined,
    },
  ];
}

function buildReadiness(
  input: PermissionPackageDraftInput,
  allowedCapabilities: Capability[],
): PermissionPackageReadiness {
  const missingFields = [
    ["tenantId", input.tenantId],
    ["workspaceId", input.workspaceId],
    ["callerInstanceId", input.callerInstanceId],
    ["targetId", input.targetId],
  ]
    .filter(([, value]) => !value.trim())
    .map(([field]) => field);
  if (!input.subjectSelector?.trim() || input.subjectSelector.trim() === "*") {
    missingFields.push("subjectSelector");
  }
  const warnings = allowedCapabilities.length === 0 ? ["No matching allowed capabilities for the selected target."] : [];
  return {
    canApply: missingFields.length === 0 && warnings.length === 0,
    missingFields,
    warnings,
  };
}

function buildPolicyGate(allowedCapabilities: Capability[]): PermissionPackagePolicyGate {
  const reasons = allowedCapabilities.flatMap((capability) => policyReasonsForCapability(capability));
  if (reasons.length === 0) {
    return {
      canApplyDirectly: true,
      decision: "allow",
      nextActions: [],
      policyVersion: policyGateVersion,
      reasons,
    };
  }
  return {
    canApplyDirectly: false,
    decision: "approval_required",
    nextActions: ["Request approval before applying this permission request."],
    policyVersion: policyGateVersion,
    reasons,
  };
}

function policyReasonsForCapability(capability: Capability): PermissionPackagePolicyReason[] {
  const reasons: PermissionPackagePolicyReason[] = [];
  if (actionRequiresApproval(capability.action)) {
    reasons.push(policyReason(
      capability,
      "action",
      "high",
      `Capability ${capability.key} uses ${capability.action} and requires approval before direct apply.`,
      "permissionPolicy.actionApprovalRequired",
      {
        action: capability.action,
        capability: capability.key,
      },
    ));
  }
  if (capability.action === "execute" && capability.riskLevel !== "low") {
    reasons.push(policyReason(
      capability,
      "execute-risk",
      "medium",
      `Capability ${capability.key} executes work with ${capability.riskLevel} risk and requires approval.`,
      "permissionPolicy.executeApprovalRequired",
      {
        capability: capability.key,
        risk: capability.riskLevel,
      },
    ));
  }
  if (capability.riskLevel === "high" || capability.riskLevel === "critical") {
    reasons.push(policyReason(
      capability,
      "risk",
      "high",
      `Capability ${capability.key} is ${capability.riskLevel} risk and requires approval.`,
      "permissionPolicy.riskApprovalRequired",
      {
        capability: capability.key,
        risk: capability.riskLevel,
      },
    ));
  }
  if (capability.sensitivity === "confidential" || capability.sensitivity === "restricted") {
    reasons.push(policyReason(
      capability,
      "sensitivity",
      "high",
      `Capability ${capability.key} touches ${capability.sensitivity} data and requires approval.`,
      "permissionPolicy.sensitivityApprovalRequired",
      {
        capability: capability.key,
        sensitivity: capability.sensitivity,
      },
    ));
  }
  return reasons;
}

function actionRequiresApproval(action: CapabilityAction): boolean {
  return action === "write" || action === "export" || action === "delete" || action === "admin";
}

function policyReason(
  capability: Capability,
  reasonId: string,
  severity: string,
  message: string,
  reasonKey: string,
  reasonValues: Record<string, string>,
): PermissionPackagePolicyReason {
  return {
    capabilityId: capability.id,
    capabilityKey: capability.key,
    id: `policy:${capability.id}:${reasonId}`,
    message,
    reasonKey,
    reasonValues,
    severity,
  };
}

function buildSimulationRows(
  allowedCapabilities: Capability[],
  blockedCapabilities: Capability[],
  template: PermissionPackageTemplate,
): PermissionPackageSimulationRow[] {
  return [
    ...allowedCapabilities.map((capability) => ({
      capabilityId: capability.id,
      capabilityKey: capability.key,
      expectedDecision: "allow" as const,
      id: `allow:${capability.id}`,
      reason: `${template.name} allows ${capability.action} capability ${capability.key}.`,
      reasonKey: "permissionSimulation.allowCapability",
      reasonValues: {
        action: capability.action,
        capability: capability.key,
        packageId: template.id,
      },
    })),
    ...blockedCapabilities.map((capability) => ({
      capabilityId: capability.id,
      capabilityKey: capability.key,
      expectedDecision: "deny" as const,
      id: `deny:${capability.id}`,
      reason: `${template.name} blocks ${capability.action} capability ${capability.key}.`,
      reasonKey: "permissionSimulation.blockCapability",
      reasonValues: {
        action: capability.action,
        capability: capability.key,
        packageId: template.id,
      },
    })),
    ...template.guardrails.map((guardrail) => ({
      ...guardrail,
      id: `guardrail:${guardrail.capabilityKey}`,
    })),
  ];
}

function draftId(input: PermissionPackageDraftInput): string {
  return [
    "pkgdraft",
    input.templateId,
    input.tenantId,
    input.workspaceId,
    input.callerInstanceId,
    input.targetId,
  ]
    .map((part) => part.trim().replace(/[^a-zA-Z0-9]+/g, "-").replace(/^-+|-+$/g, "").toLowerCase())
    .filter(Boolean)
    .join("-");
}
