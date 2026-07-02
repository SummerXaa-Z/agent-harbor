import type {
  PermissionPackageApplication,
  PermissionPackageApprovalEffectiveStatus,
  PermissionPackageApprovalRequest,
  PermissionPackageDraft,
  PermissionPackageProductionReadiness
} from "./permissionPackages";

export type AiAdminProductionConsoleStepKey =
  | "request"
  | "approval"
  | "application"
  | "runtime"
  | "production";

export type AiAdminProductionConsoleStatus = "ready" | "needs_review" | "blocked" | "pending";

export interface AiAdminProductionConsoleStep {
  key: AiAdminProductionConsoleStepKey;
  labelKey: string;
  status: AiAdminProductionConsoleStatus;
  detail: string;
  detailKey?: string;
  metric?: string;
}

export interface AiAdminProductionConsoleSummary {
  status: AiAdminProductionConsoleStatus;
  readyCount: number;
  totalCount: number;
  primaryActionKey: string;
  steps: AiAdminProductionConsoleStep[];
}

export interface AiAdminProductionConsoleInput {
  application: PermissionPackageApplication | null;
  approvalRequest: PermissionPackageApprovalRequest | null;
  draft: PermissionPackageDraft;
  productionReadiness: PermissionPackageProductionReadiness | null;
}

export function buildAiAdminProductionConsoleSummary({
  application,
  approvalRequest,
  draft,
  productionReadiness
}: AiAdminProductionConsoleInput): AiAdminProductionConsoleSummary {
  const approvalRequired = !draft.policyGate.canApplyDirectly;
  const approvalSatisfiedByEvidence = approvalRequired && (Boolean(application) || productionReadiness?.status === "ready");
  const approvalRequestEffectiveStatus = approvalRequest ? approvalEffectiveStatus(approvalRequest) : null;
  const approved = !approvalRequired || approvalRequestEffectiveStatus === "approved" || approvalSatisfiedByEvidence;
  const runtimeEvidenceCount = productionReadiness
    ? Number(Boolean(productionReadiness.runtimeEvidence.allowedTrace)) + Number(Boolean(productionReadiness.runtimeEvidence.deniedTrace))
    : 0;

  const steps: AiAdminProductionConsoleStep[] = [
    {
      key: "request",
      labelKey: "productionConsole.request",
      status: draft.readiness.canApply ? "ready" : "blocked",
      detail: draft.template.id,
      detailKey: draft.readiness.canApply
        ? "productionConsole.requestConfigured"
        : "productionConsole.requestNeedsInput",
      metric: capabilitySummary(draft)
    },
    {
      key: "approval",
      labelKey: "productionConsole.approval",
      status: approvalRequired ? approvalStatus(approvalRequest, approvalSatisfiedByEvidence) : "ready",
      detail: approvalRequired ? approvalRequest?.id ?? application?.id ?? "-" : "-",
      detailKey: approvalRequired
        ? approvalSatisfiedByEvidence
          ? "productionConsole.approvalSatisfied"
          : approvalRequest
          ? `status.approval${capitalize(approvalRequestEffectiveStatus ?? approvalRequest.status)}`
          : "status.approvalNotRequested"
        : "productionConsole.approvalNotRequired",
      metric: approvalRequest?.reviewedBy || approvalRequest?.requestedBy || undefined
    },
    {
      key: "application",
      labelKey: "productionConsole.application",
      status: application ? "ready" : "pending",
      detail: application?.id ?? "-",
      detailKey: application ? "productionConsole.applicationRecorded" : "productionConsole.applicationPending",
      metric: application ? `${application.tenantEntitlementIds.length + application.workspaceAssignmentIds.length + application.instanceAssignmentIds.length}` : undefined
    },
    {
      key: "runtime",
      labelKey: "productionConsole.runtime",
      status: runtimeEvidenceCount === 2 ? "ready" : productionReadiness ? "blocked" : "pending",
      detail: productionReadiness ? `${runtimeEvidenceCount}/2` : "-",
      detailKey: runtimeEvidenceCount === 2 ? "productionConsole.runtimeReady" : "productionConsole.runtimeEvidence",
      metric: `${runtimeEvidenceCount}/2`
    },
    {
      key: "production",
      labelKey: "productionConsole.productionReadiness",
      status: productionReadinessStatus(productionReadiness),
      detail: productionReadiness?.status ?? "-",
      detailKey: productionReadiness?.status === "ready"
        ? "productionConsole.productionReady"
        : productionReadiness ? `status.production${capitalizeProductionStatus(productionReadiness.status)}` : "productionConsole.productionPending",
      metric: productionReadiness ? `${productionReadiness.summary.readyCount}/${productionReadiness.checks.length}` : undefined
    }
  ];

  return {
    steps,
    primaryActionKey: primaryActionKey({ application, approved, approvalRequired, productionReadiness }),
    readyCount: steps.filter((step) => step.status === "ready").length,
    status: overallStatus(steps),
    totalCount: steps.length
  };
}

function approvalStatus(request: PermissionPackageApprovalRequest | null, satisfiedByEvidence = false): AiAdminProductionConsoleStatus {
  if (satisfiedByEvidence) return "ready";
  if (!request) return "pending";
  const effectiveStatus = approvalEffectiveStatus(request);
  if (effectiveStatus === "approved") return "ready";
  if (effectiveStatus === "pending" || effectiveStatus === "expired") return "pending";
  return "blocked";
}

function productionReadinessStatus(readiness: PermissionPackageProductionReadiness | null): AiAdminProductionConsoleStatus {
  if (!readiness) return "pending";
  if (readiness.status === "ready") return "ready";
  if (readiness.status === "needs_review") return "needs_review";
  return "blocked";
}

function approvalEffectiveStatus(request: PermissionPackageApprovalRequest): PermissionPackageApprovalEffectiveStatus {
  return request.effectiveStatus ?? request.status;
}

function capabilitySummary(draft: PermissionPackageDraft) {
  const allowed = draft.allowedCapabilities[0]?.key ?? "0";
  const blocked = draft.blockedCapabilities[0]?.key ?? "0";
  return `${allowed} allowed / ${blocked} blocked`;
}

function overallStatus(steps: AiAdminProductionConsoleStep[]): AiAdminProductionConsoleStatus {
  if (steps.some((step) => step.status === "blocked")) return "blocked";
  if (steps.some((step) => step.status === "pending")) return "pending";
  if (steps.find((step) => step.key === "production")?.status === "ready") return "ready";
  if (steps.some((step) => step.status === "needs_review")) return "needs_review";
  return "ready";
}

function primaryActionKey({
  application,
  approvalRequired,
  approved,
  productionReadiness
}: {
  application: PermissionPackageApplication | null;
  approvalRequired: boolean;
  approved: boolean;
  productionReadiness: PermissionPackageProductionReadiness | null;
}) {
  if (productionReadiness?.status === "ready") return "action.exportProductionEvidence";
  if (!approved && approvalRequired) return "action.createApprovalRequest";
  if (!application) return "action.applyPermissionPackage";
  return "action.checkProductionReadiness";
}

function capitalize(value: string) {
  return `${value.charAt(0).toUpperCase()}${value.slice(1)}`;
}

function capitalizeProductionStatus(value: PermissionPackageProductionReadiness["status"]) {
  if (value === "needs_review") return "NeedsReview";
  if (value === "blocked") return "Blocked";
  return "Ready";
}
