import { permissionPackageApprovalEffectiveStatus } from "../../permissionPackages.ts";
import type { PermissionPackageApprovalRequestPathFilter } from "../../apiPaths.ts";
import type {
  AccessHandoff,
  PermissionPackageApprovalRequest,
  PermissionPackageDraftInput,
  PermissionPackageProductionReadiness,
  PermissionPackageWorkbenchPreview,
} from "../../permissionPackages.ts";

export type PermissionChangeState =
  | "needsInput"
  | "draft"
  | "noApproval"
  | "submitted"
  | "approved"
  | "rejected"
  | "withdrawn"
  | "expired"
  | "applied"
  | "checked"
  | "needsReview"
  | "blocked"
  | "handoff";

export type PermissionChangeAction =
  | "submit"
  | "submitted"
  | "apply"
  | "withdraw"
  | "reedit"
  | "resubmit"
  | "changeTemplate"
  | "goCheck"
  | "exportReport"
  | "recheck"
  | "viewRequests"
  | "nextAction"
  | "copyConfig"
  | "createToken";

export type PermissionChangeTone = "info" | "success" | "warning" | "danger";
export type PermissionStepState = "done" | "current" | "blocked" | "skipped" | "pending";

export const permissionChangeStepKeys = ["template", "scope", "submit", "approval", "apply"] as const;

export interface PermissionChangeInput {
  // A request bound locally (just created, reconciled after 409, withdrawn,
  // or opened from a notification). The preview never returns expired ones.
  approval?: PermissionPackageApprovalRequest | null;
  capabilityChanged?: boolean;
  handoff?: Pick<AccessHandoff, "status" | "tokens"> | null;
  // Fields the console itself requires before submitting (request text).
  localMissing?: readonly string[];
  preview: PermissionPackageWorkbenchPreview | null;
  readiness?: PermissionPackageProductionReadiness | null;
}

export interface PermissionChangePresentation {
  descKey: string;
  polling: boolean;
  primary: PermissionChangeAction | null;
  primaryDisabled: boolean;
  secondary: PermissionChangeAction[];
  state: PermissionChangeState;
  steps: Record<(typeof permissionChangeStepKeys)[number], PermissionStepState>;
  titleKey: string;
  tone: PermissionChangeTone;
}

// Checks that only turn ready after real calls go through the gateway; right
// after applying they are still blocking, which means "applied, check next",
// not "blocked".
const runtimeOnlyCheckCodes = new Set([
  "runtime_allowed_trace_present",
  "runtime_denied_trace_present",
  "applied_audit_event_present",
]);

export function effectiveApproval(
  fromPreview: PermissionPackageApprovalRequest | null | undefined,
  local: PermissionPackageApprovalRequest | null | undefined,
): PermissionPackageApprovalRequest | null {
  if (!fromPreview) return local ?? null;
  if (!local) return fromPreview;
  if (fromPreview.id === local.id) {
    return Date.parse(local.updatedAt) > Date.parse(fromPreview.updatedAt) ? local : fromPreview;
  }
  const rank = (request: PermissionPackageApprovalRequest) => {
    const status = permissionPackageApprovalEffectiveStatus(request);
    return status === "approved" ? 3 : status === "pending" ? 2 : 1;
  };
  if (rank(fromPreview) !== rank(local)) return rank(fromPreview) > rank(local) ? fromPreview : local;
  return Date.parse(local.createdAt) > Date.parse(fromPreview.createdAt) ? local : fromPreview;
}

export function handoffDelivered(handoff: Pick<AccessHandoff, "status" | "tokens"> | null | undefined): boolean {
  return Boolean(handoff && handoff.status === "ready" && handoff.tokens.some((token) => token.status === "active"));
}

export function awaitingRuntimeChecksOnly(readiness: PermissionPackageProductionReadiness): boolean {
  const blocking = readiness.checks.filter((check) => check.severity === "blocking");
  return blocking.length > 0 && blocking.every((check) => runtimeOnlyCheckCodes.has(check.code));
}

export function permissionChangeState(input: PermissionChangeInput): PermissionChangeState {
  if (input.capabilityChanged) return "blocked";
  const preview = input.preview;
  if (!preview) return "needsInput";
  const summary = preview.summary;
  if (summary.status === "needs_input") return "needsInput";

  const approval = effectiveApproval(preview.approvalRequest, input.approval);
  const readiness = input.readiness ?? preview.productionReadiness ?? null;
  const applied = summary.applied || Boolean(preview.latestApplication) || Boolean(approval?.consumedAt);
  if (applied) {
    if (handoffDelivered(input.handoff)) return "handoff";
    if (!readiness) return "applied";
    if (readiness.status === "ready") return "checked";
    if (readiness.status === "needs_review") return "needsReview";
    return awaitingRuntimeChecksOnly(readiness) ? "applied" : "blocked";
  }
  if (summary.status === "blocked") return "blocked";
  if (preview.draft.policyGate.canApplyDirectly) return "noApproval";
  if (approval) {
    const status = permissionPackageApprovalEffectiveStatus(approval);
    if (status === "pending") return "submitted";
    if (status === "approved") return "approved";
    if (status === "rejected") return "rejected";
    if (status === "withdrawn") return "withdrawn";
    if (status === "expired") return "expired";
  }
  return input.localMissing && input.localMissing.length > 0 ? "needsInput" : "draft";
}

type StepTuple = [PermissionStepState, PermissionStepState, PermissionStepState, PermissionStepState, PermissionStepState];

const allDone: StepTuple = ["done", "done", "done", "done", "done"];

function steps(tuple: StepTuple): PermissionChangePresentation["steps"] {
  return { template: tuple[0], scope: tuple[1], submit: tuple[2], approval: tuple[3], apply: tuple[4] };
}

// Banner, primary and secondary actions and stepper per state (plan §7).
export function permissionChangePresentation(
  state: PermissionChangeState,
  options: { templateSelected: boolean } = { templateSelected: true },
): PermissionChangePresentation {
  const base = { descKey: `rd.pc.${state}.desc`, polling: false, primaryDisabled: false, secondary: [], state, titleKey: `rd.pc.${state}.title` };
  switch (state) {
    case "needsInput":
      return {
        ...base,
        primary: "submit",
        primaryDisabled: true,
        steps: steps(options.templateSelected
          ? ["done", "current", "pending", "pending", "pending"]
          : ["current", "pending", "pending", "pending", "pending"]),
        tone: "info",
      };
    case "draft":
      return { ...base, primary: "submit", secondary: ["changeTemplate"], steps: steps(["done", "done", "current", "pending", "pending"]), tone: "info" };
    case "noApproval":
      return { ...base, primary: "apply", steps: steps(["done", "done", "done", "skipped", "current"]), tone: "info" };
    case "submitted":
      return {
        ...base,
        polling: true,
        primary: "submitted",
        primaryDisabled: true,
        secondary: ["withdraw"],
        steps: steps(["done", "done", "done", "current", "pending"]),
        tone: "warning",
      };
    case "approved":
      return { ...base, primary: "apply", steps: steps(["done", "done", "done", "done", "current"]), tone: "success" };
    case "rejected":
      return { ...base, primary: "reedit", steps: steps(["done", "done", "done", "blocked", "pending"]), tone: "danger" };
    case "withdrawn":
      return { ...base, primary: "reedit", steps: steps(["done", "done", "current", "pending", "pending"]), tone: "info" };
    case "expired":
      return { ...base, primary: "resubmit", steps: steps(["done", "done", "current", "pending", "pending"]), tone: "warning" };
    case "applied":
      return { ...base, primary: "goCheck", steps: steps(allDone), tone: "success" };
    case "checked":
      return { ...base, primary: "exportReport", secondary: ["recheck", "viewRequests"], steps: steps(allDone), tone: "success" };
    case "needsReview":
      return { ...base, primary: "nextAction", secondary: ["recheck"], steps: steps(allDone), tone: "warning" };
    case "blocked":
      return { ...base, primary: "nextAction", steps: steps(allDone), tone: "danger" };
    case "handoff":
      return { ...base, primary: "copyConfig", secondary: ["createToken"], steps: steps(allDone), tone: "success" };
  }
}

// Mirrors the legacy console's inline reconciliation after a duplicate create
// (409 PERMISSION_PACKAGE_APPROVAL_ALREADY_PENDING): re-fetch by scope and bind
// the pending request whose snapshot text matches, else the newest pending one.
export function approvalReconcileFilter(input: PermissionPackageDraftInput): PermissionPackageApprovalRequestPathFilter {
  return {
    callerInstanceId: input.callerInstanceId,
    limit: 8,
    requestedCapabilityId: input.requestedCapabilityId,
    targetId: input.targetId,
    templateId: input.templateId,
    tenantId: input.tenantId,
    workspaceId: input.workspaceId,
  };
}

export function pickPendingApproval(
  rows: readonly PermissionPackageApprovalRequest[],
  input: PermissionPackageDraftInput,
): PermissionPackageApprovalRequest | null {
  const pendingRows = rows.filter((request) => permissionPackageApprovalEffectiveStatus(request) === "pending");
  return pendingRows.find((request) => (
    (request.subjectSelector ?? "") === (input.subjectSelector ?? "")
    && (request.requestText ?? "") === input.requestText
    && (request.region ?? "") === input.region
  )) ?? pendingRows[0] ?? null;
}

interface ApiErrorLike {
  code?: string;
  message?: string;
  status?: number;
}

export function isApprovalAlreadyPendingError(error: unknown): boolean {
  return (error as ApiErrorLike | null)?.code === "PERMISSION_PACKAGE_APPROVAL_ALREADY_PENDING";
}

// Only happens when the policy gate changed between preview and submit.
export function isApprovalNotRequiredError(error: unknown): boolean {
  const candidate = error as ApiErrorLike | null;
  return candidate?.status === 400 && /does not require approval/i.test(candidate.message ?? "");
}

export function isCapabilityChangedError(error: unknown): boolean {
  return (error as ApiErrorLike | null)?.code === "PERMISSION_PACKAGE_CAPABILITY_CHANGED";
}
