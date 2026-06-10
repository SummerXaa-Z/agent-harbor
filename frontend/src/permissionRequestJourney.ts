import type {
  PermissionPackageApplication,
  PermissionPackageApprovalRequest,
  PermissionPackageDraft,
  PermissionPackageProductionReadiness
} from "./permissionPackages";

export type PermissionRequestWizardStep = "scope" | "template" | "approval" | "apply" | "goLive";
export type PermissionRequestProcessStepStatus = "complete" | "current" | "waiting";

export interface PermissionRequestProcessStepInput {
  complete: boolean;
  key: PermissionRequestWizardStep;
}

export interface PermissionRequestWizardStateInput {
  application: PermissionPackageApplication | null;
  approvalRequest: PermissionPackageApprovalRequest | null;
  draft: PermissionPackageDraft;
  productionReadiness: PermissionPackageProductionReadiness | null;
}

export function currentPermissionRequestWizardStep({
  application,
  approvalRequest,
  draft,
  productionReadiness
}: PermissionRequestWizardStateInput): PermissionRequestWizardStep {
  if (!draft.readiness.canApply) {
    return draft.readiness.missingFields.length > 0 ? "scope" : "template";
  }
  if (!draft.policyGate.canApplyDirectly && approvalRequest?.status !== "approved") {
    return "approval";
  }
  if (!application || productionReadiness?.nextActionCode === "apply_permission_package") {
    return "apply";
  }
  return "goLive";
}

export function permissionRequestProcessStepStatuses<TStep extends PermissionRequestProcessStepInput>(
  steps: TStep[],
  currentStep: PermissionRequestWizardStep
): Array<TStep & { status: PermissionRequestProcessStepStatus }> {
  let hasCurrentStep = false;
  return steps.map((step) => {
    if (step.complete) return { ...step, status: "complete" };
    if (!hasCurrentStep && step.key === currentStep) {
      hasCurrentStep = true;
      return { ...step, status: "current" };
    }
    return { ...step, status: "waiting" };
  });
}
