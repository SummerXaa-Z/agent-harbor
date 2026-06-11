import { useState } from "react";
import {
  CheckCircle2,
  ClipboardCheck,
  Download,
  FileSearch,
  RefreshCw,
  TriangleAlert,
  Undo2,
  Workflow
} from "lucide-react";

import {
  accessSubjectOptionForSelectorFrom,
  normalizeAccessSubjectOptions,
  type AccessSubjectOption
} from "../accessSubjects";
import { summarizeDataScopes } from "../accessProfile";
import {
  aiAdminApprovalReadinessRows,
  type AiAdminApprovalReadinessState
} from "../aiAdminApprovalReadiness";
import {
  summarizeAiAdminGoLiveReadiness,
  type AiAdminApprovalJourneyEvaluation,
  type AiAdminApprovalJourneyConfig,
  type AiAdminApprovalJourneyResult
} from "../aiAdminApprovalJourney";
import type {
  AiAdminProductionConsoleStatus,
  AiAdminProductionConsoleSummary
} from "../aiAdminProductionConsole";
import type { MetricTone } from "../consoleMetrics";
import { capabilityDisplayName, capabilityKeyDisplayName, dataScopeValueLabels } from "../consolePresenters";
import { createTranslator } from "../i18n";
import {
  currentPermissionRequestWizardStep,
  permissionRequestProcessStepStatuses,
  type PermissionRequestWizardStep
} from "../permissionRequestJourney";
import { applyPermissionRequestAccessSubject } from "../permissionRequestForm";
import type {
  PermissionPackageApplyPreflight,
  PermissionPackageApplication,
  PermissionPackageApplicationHealth,
  PermissionPackageApplicationHealthRow,
  PermissionPackageApplicationHealthStatus,
  PermissionPackageApplicationImpact,
  PermissionPackageApprovalRequest,
  PermissionPackageDraft,
  PermissionPackageDraftInput,
  PermissionPackageProductionReadiness,
  PermissionPackageProductionReadinessCheck,
  PermissionPackageProductionReadinessStatus,
  PermissionPackageTemplate,
  PermissionPackageWorkbenchPreview,
  PermissionPackageWorkbenchStepKey,
  PermissionPackageWorkbenchStepStatus
} from "../permissionPackages";
import type {
  AccessDecisionExplainResult,
  Agent,
  AuditEvent,
  Capability,
  Tenant
} from "../types";
import { ApprovalDropdown, type ApprovalDropdownOption } from "./ApprovalDropdown";
import { TechnicalId } from "./TechnicalId";
import { Badge, EmptyRow } from "./ui";

type Tone = MetricTone;
type Translator = ReturnType<typeof createTranslator>;
const defaultWorkspaceId = "workspace-sandbox";

type ApprovalDecisionAction = "approve" | "reject" | "withdraw";
type PermissionRequestStepTarget = PermissionRequestWizardStep | "validation" | "acceptance";

interface PendingApprovalDecision {
  action: ApprovalDecisionAction;
  requestId?: string;
  comment: string;
  error: string;
}

interface AiAdminPermissionWorkbenchProps {
  accessSubjects: AccessSubjectOption[];
  agents: Agent[];
  application: PermissionPackageApplication | null;
  approvalAction: "" | "create" | "approve" | "reject" | "withdraw";
  approvalAuditEvent: AuditEvent | null;
  approvalJourneyConfig: AiAdminApprovalJourneyConfig;
  approvalJourneyEvaluation: AiAdminApprovalJourneyEvaluation;
  approvalJourneyMessage: string;
  approvalJourneyResult: AiAdminApprovalJourneyResult | null;
  approvalJourneyRunning: boolean;
  approvalReadiness: AiAdminApprovalReadinessState;
  approvalReadinessChecking: boolean;
  approvalReadinessMessage: string;
  approvalResolutionBlocked: boolean;
  approvalRequest: PermissionPackageApprovalRequest | null;
  approvalRequests: PermissionPackageApprovalRequest[];
  approvalReviewer: string;
  workbenchPreview: PermissionPackageWorkbenchPreview | null;
  productionSummary: AiAdminProductionConsoleSummary;
  applyPreflight: PermissionPackageApplyPreflight | null;
  applyPreflightLoading: boolean;
  applyPreflightMessage: string;
  applicationHealth: PermissionPackageApplicationHealth | null;
  applicationHealthLoading: boolean;
  applicationHealthMessage: string;
  applicationImpact: PermissionPackageApplicationImpact | null;
  applicationImpactLoading: boolean;
  applicationImpactMessage: string;
  productionReadiness: PermissionPackageProductionReadiness | null;
  productionEvidenceExporting: boolean;
  productionReadinessLoading: boolean;
  productionReadinessMessage: string;
  accessDecisionExplanation: AccessDecisionExplainResult | null;
  accessDecisionExplanationLoading: boolean;
  accessDecisionExplanationMessage: string;
  applying: boolean;
  draft: PermissionPackageDraft;
  form: PermissionPackageDraftInput;
  liveDataAvailable: boolean;
  message: string;
  mcpTargets: Agent[];
  onApply: () => void;
  onApprovalReviewerChange: (reviewer: string) => void;
  onApproveApprovalRequest: (requestId?: string, comment?: string) => void;
  onChange: (form: PermissionPackageDraftInput) => void;
  onCreateApprovalRequest: () => void;
  onExplainAccessDecision: () => void;
  onOpenAccessProfile: () => void;
  onRefreshApplyPreflight: () => void;
  onRefreshApprovalReadiness: () => void;
  onRefreshApplicationHealth: () => void;
  onExportProductionEvidence: () => void;
  onRefreshProductionReadiness: () => void;
  onRefreshReviewerQueue: () => void;
  onRejectApprovalRequest: (requestId?: string, comment?: string) => void;
  onRehearseApplicationDrift: () => void;
  onReviewApplicationHealthRow: (application: PermissionPackageApplication) => void;
  onReviewApplicationImpact: () => void;
  onRunApprovalJourney: () => void;
  onSelectApprovalRequest: (requestId: string) => void;
  onStartNewPermissionChange: () => void;
  onWithdrawApprovalRequest: (comment?: string) => void;
  reviewerQueueLoading: boolean;
  reviewerQueueMessage: string;
  selectedApprovalRequestId: string;
  templates: PermissionPackageTemplate[];
  tenants: Tenant[];
  t: Translator;
}

export function AiAdminPermissionWorkbench(props: AiAdminPermissionWorkbenchProps) {
  const {
    accessSubjects,
    agents,
    application,
    approvalAction,
    approvalAuditEvent,
    approvalJourneyEvaluation,
    approvalJourneyMessage,
    approvalJourneyResult,
    approvalJourneyRunning,
    approvalReadiness,
    approvalReadinessChecking,
    approvalReadinessMessage,
    approvalResolutionBlocked,
    approvalRequest,
    approvalRequests,
    approvalReviewer,
    workbenchPreview,
    productionSummary,
    applyPreflight,
    applyPreflightLoading,
    applyPreflightMessage,
    applicationHealth,
    applicationHealthLoading,
    applicationHealthMessage,
    applicationImpact,
    applicationImpactLoading,
    applicationImpactMessage,
    productionReadiness,
    productionEvidenceExporting,
    productionReadinessLoading,
    productionReadinessMessage,
    accessDecisionExplanation,
    accessDecisionExplanationLoading,
    accessDecisionExplanationMessage,
    applying,
    draft,
    form,
    liveDataAvailable,
    message,
    mcpTargets,
    onApply,
    onApprovalReviewerChange,
    onApproveApprovalRequest,
    onChange,
    onCreateApprovalRequest,
    onExplainAccessDecision,
    onOpenAccessProfile,
    onRefreshApplyPreflight,
    onRefreshApprovalReadiness,
    onRefreshApplicationHealth,
    onExportProductionEvidence,
    onRefreshProductionReadiness,
    onRefreshReviewerQueue,
    onRejectApprovalRequest,
    onRehearseApplicationDrift,
    onReviewApplicationHealthRow,
    onReviewApplicationImpact,
    onRunApprovalJourney,
    onSelectApprovalRequest,
    onStartNewPermissionChange,
    onWithdrawApprovalRequest,
    reviewerQueueLoading,
    reviewerQueueMessage,
    selectedApprovalRequestId,
    templates,
    tenants,
    t
  } = props;
  const [pendingApprovalDecision, setPendingApprovalDecision] = useState<PendingApprovalDecision | null>(null);
  const callers = agents.filter((agent) => agent.status === "active" && agent.channelType === "local");
  const selectedCaller = agents.find((agent) => agent.id === form.callerInstanceId);
  const selectedTarget = mcpTargets.find((agent) => agent.id === form.targetId);
  const tenantPath = permissionTenantPathLabel(form.tenantId, tenants, t);
  const workspaceName = permissionWorkspaceDisplayName(form.workspaceId, agents, t);
  const callerName = selectedCaller ? permissionEntityDisplayName(selectedCaller.name, t) : t("form.selectCaller");
  const tenantOptions = uniquePermissionEntityOptions(tenants, form.tenantId, (tenant) => permissionEntityDisplayName(tenant.name, t));
  const callerOptions = uniquePermissionEntityOptions(callers, form.callerInstanceId, (agent) => permissionEntityDisplayName(agent.name, t));
  const targetOptions = uniquePermissionEntityOptions(mcpTargets, form.targetId, (agent) => permissionEntityDisplayName(agent.name, t));
  const tenantDropdownOptions = [
    ...(tenants.some((tenant) => tenant.id === form.tenantId) ? [] : [{ value: form.tenantId, label: tenantPath.primary }]),
    ...tenantOptions.map((tenant) => ({ value: tenant.id, label: tenant.label }))
  ].filter((option) => option.value);
  const callerDropdownOptions = [
    { value: "", label: t("form.selectCaller") },
    ...callerOptions.map((agent) => ({ value: agent.id, label: agent.label }))
  ];
  const targetDropdownOptions = [
    { value: "", label: t("form.allMcpTargets") },
    ...targetOptions.map((agent) => ({ value: agent.id, label: agent.label }))
  ];
  const templateDropdownOptions = templates.map((template) => ({
    value: template.id,
    label: permissionPackageTemplateName(template, t)
  }));
  const hasApprovedRequest = approvalRequest?.status === "approved";
  const canApply = draft.readiness.canApply && (draft.policyGate.canApplyDirectly || hasApprovedRequest);
  const reviewerQueueRequests = approvalRequests.filter((request) => request.status === "pending");
  const goLiveReadiness = summarizeAiAdminGoLiveReadiness(approvalJourneyEvaluation);
  const productionReady = productionReadiness?.status === "ready"
    || Boolean(workbenchPreview?.summary.productionReady)
    || productionSummary.status === "ready";
  const goLiveReady = productionReady || goLiveReadiness.status === "ready";
  const requestFormLocked = Boolean(application)
    || goLiveReady
    || approvalRequest?.status === "pending"
    || approvalRequest?.status === "approved";
  const requestFormActiveLocked = Boolean(application) || goLiveReady;
  const requestFormLockedDetailKey = requestFormActiveLocked
    ? "text.permissionRequestLockedActiveDetail"
    : "text.permissionRequestLockedApprovalDetail";
  const requestFormTitleKey = requestFormLocked ? "section.permissionRequestReview" : "section.permissionRequestForm";
  const requestFormHelpKey = requestFormLocked ? "text.permissionRequestReviewHelp" : "text.permissionRequestScopeHelp";
  const reviewerQueueReadOnly = Boolean(application) || goLiveReady;
  const reviewerQueueTitleKey = reviewerQueueReadOnly ? "section.permissionApprovalTrace" : "section.permissionReviewerQueue";
  const reviewerQueueRefreshKey = reviewerQueueReadOnly ? "action.refreshApprovalTrace" : "action.refreshReviewerQueue";
  const runtimeValidationReady = Boolean(approvalJourneyResult) || goLiveReady;
  const approvalEffectivelyResolved = !draft.policyGate.canApplyDirectly
    && (approvalRequest?.status === "approved" || Boolean(application) || goLiveReady);
  const approvalDisplayStatus: PermissionPackageApprovalRequest["status"] | null = approvalEffectivelyResolved
    ? "approved"
    : approvalRequest?.status ?? null;
  const approvalDisplayTone = draft.policyGate.canApplyDirectly
    ? "success"
    : approvalDisplayStatus ? permissionApprovalStatusTone(approvalDisplayStatus) : "warning";
  const approvalDisplayLabel = draft.policyGate.canApplyDirectly
    ? t("status.directApplyAllowed")
    : approvalDisplayStatus ? permissionApprovalStatusLabel(approvalDisplayStatus, t) : t("status.approvalNotRequested");
  const approvalGateDetailKey = permissionPolicyGateDetailKey(draft.policyGate.canApplyDirectly, approvalDisplayStatus);
  const showPolicyGateReasons = !draft.policyGate.canApplyDirectly
    && (!approvalDisplayStatus || approvalDisplayStatus === "pending");
  const showCreateApprovalAction = !application && !goLiveReady
    && (!approvalRequest || (approvalRequest.status !== "pending" && approvalRequest.status !== "approved"));
  const showPendingApprovalActions = !application && !goLiveReady && approvalRequest?.status === "pending";
  const currentWizardStep = currentPermissionRequestWizardStep({
    application,
    approvalRequest,
    draft,
    productionReadiness
  });
  const goLiveNextKey = goLiveReady
    ? "journey.aiAdmin.next.complete"
    : `journey.aiAdmin.next.${goLiveReadiness.nextStep?.key ?? "tenantTree"}`;
  const goLiveCompletedAt = productionReadiness?.generatedAt
    ? tx(t, "text.permissionChangeCompletedAt", { date: formatDate(productionReadiness.generatedAt) })
    : "";
  const dataScopeLabels = dataScopeValueLabels(t);
  const messageTone = permissionInlineMessageTone(message);
  const approvalReadinessMessageTone = permissionInlineMessageTone(approvalReadinessMessage);
  const applyPreflightMessageTone = permissionInlineMessageTone(applyPreflightMessage);
  const reviewerQueueMessageTone = permissionInlineMessageTone(reviewerQueueMessage);
  const productionReadinessMessageTone = permissionInlineMessageTone(productionReadinessMessage);
  const accessDecisionExplanationMessageTone = permissionInlineMessageTone(accessDecisionExplanationMessage);
  const applicationHealthMessageTone = permissionInlineMessageTone(applicationHealthMessage);
  const applicationImpactMessageTone = permissionInlineMessageTone(applicationImpactMessage);
  const permissionRequestBusy = approvalJourneyRunning
    || applying
    || Boolean(approvalAction)
    || approvalResolutionBlocked
    || approvalReadinessChecking
    || applyPreflightLoading
    || productionEvidenceExporting
    || productionReadinessLoading
    || accessDecisionExplanationLoading
    || applicationHealthLoading
    || applicationImpactLoading
    || reviewerQueueLoading;
  const reviewerIdentity = approvalReviewer.trim()
    ? permissionEntityDisplayName(approvalReviewer.trim(), t)
    : t("text.approvalReviewerFallback");
  const beginApprovalDecision = (action: ApprovalDecisionAction, requestId?: string) => {
    if (requestId) onSelectApprovalRequest(requestId);
    setPendingApprovalDecision({
      action,
      requestId,
      comment: action === "approve" ? t("text.approvalApproveDefaultComment") : "",
      error: ""
    });
  };
  const cancelApprovalDecision = () => setPendingApprovalDecision(null);
  const updatePendingApprovalComment = (comment: string) => {
    setPendingApprovalDecision((current) => current ? { ...current, comment, error: "" } : current);
  };
  function scrollToPermissionRequestStep(step: PermissionRequestStepTarget) {
    document.getElementById(permissionRequestStepSectionId(step))?.scrollIntoView({
      behavior: "smooth",
      block: "start"
    });
  }
  function scrollToAcceptanceDetails() {
    document.getElementById("permission-request-acceptance-details")?.scrollIntoView({
      behavior: "smooth",
      block: "start"
    });
  }
  const confirmPendingApprovalDecision = () => {
    if (!pendingApprovalDecision) return;
    const comment = pendingApprovalDecision.comment.trim();
    if (pendingApprovalDecision.action === "reject" && !comment) {
      setPendingApprovalDecision({
        ...pendingApprovalDecision,
        error: t("message.permissionApprovalRejectReasonRequired")
      });
      return;
    }
    if (pendingApprovalDecision.action === "approve") {
      onApproveApprovalRequest(pendingApprovalDecision.requestId, comment || t("text.approvalApproveDefaultComment"));
    } else if (pendingApprovalDecision.action === "withdraw") {
      onWithdrawApprovalRequest(comment);
    } else {
      onRejectApprovalRequest(pendingApprovalDecision.requestId, comment);
    }
    setPendingApprovalDecision(null);
  };
  const runProductionPrimaryAction = () => {
    if (journeyStatus.nextActionKey === "action.refreshReviewerQueue") {
      onRefreshReviewerQueue();
      return;
    }
    if (journeyStatus.nextActionKey === "action.applyPermissionPackage") {
      onApply();
      return;
    }
    if (journeyStatus.nextActionKey === "action.exportProductionEvidence") {
      onExportProductionEvidence();
      return;
    }
    if (journeyStatus.nextActionKey === "action.checkPreflight") {
      onRefreshApplyPreflight();
      return;
    }
    if (journeyStatus.nextActionKey === "action.createApprovalRequest") {
      onCreateApprovalRequest();
      return;
    }
    if (journeyStatus.nextActionKey === "action.startPermissionApproval") {
      onStartNewPermissionChange();
      return;
    }
    if (primaryActionCode === "create_approval_request" || productionSummary.primaryActionKey === "action.createApprovalRequest") {
      onCreateApprovalRequest();
      return;
    }
    if (primaryActionCode === "review_approval_request") {
      onRefreshReviewerQueue();
      return;
    }
    if (primaryActionCode === "apply_permission_package" || productionSummary.primaryActionKey === "action.applyPermissionPackage") {
      onApply();
      return;
    }
    if (primaryActionCode === "run_runtime_validation") {
      onRunApprovalJourney();
      return;
    }
    if (primaryActionCode === "export_production_evidence" || productionSummary.primaryActionKey === "action.exportProductionEvidence") {
      onExportProductionEvidence();
      return;
    }
    onRefreshProductionReadiness();
  };
  const flowSteps: Array<{
    key: PermissionRequestWizardStep;
    labelKey: string;
    detail: string;
    complete: boolean;
  }> = [
    {
      complete: draft.readiness.missingFields.length === 0,
      detail: tenantPath.primary,
      key: "scope",
      labelKey: "section.permissionWizardScope"
    },
    {
      complete: draft.allowedCapabilities.length > 0,
      detail: permissionPackageTemplateName(draft.template, t),
      key: "template",
      labelKey: "section.permissionWizardTemplate"
    },
    {
      complete: draft.policyGate.canApplyDirectly || approvalRequest?.status === "approved",
      detail: draft.policyGate.canApplyDirectly
        ? t("status.directApplyAllowed")
        : approvalRequest ? permissionApprovalStatusLabel(approvalRequest.status, t) : t("status.approvalNotRequested"),
      key: "approval",
      labelKey: "section.permissionWizardApproval"
    },
    {
      complete: Boolean(application),
      detail: application ? t("status.stepComplete") : canApply ? t("status.readyToApply") : t("status.stepMissing"),
      key: "apply",
      labelKey: "section.permissionWizardApply"
    },
    {
      complete: goLiveReady,
      detail: goLiveReady ? t("status.productionReady") : tx(t, "text.aiAdminGoLiveRemainingBadge", { count: goLiveReadiness.remainingCount }),
      key: "goLive",
      labelKey: "section.permissionWizardGoLive"
    }
  ];
  const readinessRows = aiAdminApprovalReadinessRows(approvalReadiness);
  const preflightSummary = applyPreflight?.summary;
  const productionChecks = productionReadiness?.checks ?? [];
  const healthRows = applicationHealth?.applications ?? [];
  const impactSummary = applicationImpact?.summary;
  const productionStepDetail = (step: AiAdminProductionConsoleSummary["steps"][number]) => {
    if (step.key === "request") {
      return `${draft.allowedCapabilities.length} ${t("detail.allowed")} / ${draft.blockedCapabilities.length} ${t("detail.denied")}`;
    }
    if (step.key === "approval") {
      return draft.policyGate.canApplyDirectly
        ? t("productionConsole.approvalNotRequired")
        : approvalRequest ? permissionApprovalStatusLabel(approvalRequest.status, t) : t("status.approvalNotRequested");
    }
    if (step.key === "application") {
      return application ? t("status.stepComplete") : t("status.stepMissing");
    }
    if (step.key === "runtime") {
      return step.metric ?? "0/2";
    }
    if (step.key === "production") {
      return productionReadinessStatusLabel(productionReadiness?.status, t);
    }
    return step.metric ?? step.detail;
  };
  const primaryActionCode = workbenchPreview?.summary.primaryActionCode;
  const journeyStatus = resolvePermissionJourneyStatus({
    approvalRequest,
    canApply,
    draft,
    goLiveReady,
    productionStatus: productionSummary.status,
    workbenchStatus: workbenchPreview?.summary.status
  });
  const readinessReadyCount = workbenchPreview?.summary.readinessReadyCount ?? productionSummary.readyCount;
  const readinessTotalCount = workbenchPreview?.summary.readinessTotalCount || productionSummary.totalCount;
  const fallbackProcessSteps = permissionRequestProcessStepStatuses(flowSteps, currentWizardStep);
  const processSteps = workbenchPreview?.summary.steps.map((step) => {
    const detailCode = permissionWorkbenchStepDisplayDetailCode(step, {
      approvalRequired: !draft.policyGate.canApplyDirectly,
      approvalStatus: approvalDisplayStatus,
      applicationReady: Boolean(application),
      goLiveReady,
      runtimeValidationReady
    });
    return {
      count: step.key === "request" ? undefined : step.count,
      detail: t(permissionWorkbenchStepDetailKey(detailCode), detailCode),
      key: step.key,
      labelKey: permissionWorkbenchStepLabelKey(step.key),
      status: permissionWorkbenchStepDisplayStatus(step, {
        approvalComplete: approvalDisplayStatus === "approved" || draft.policyGate.canApplyDirectly,
        applicationReady: Boolean(application),
        goLiveReady,
        runtimeValidationReady
      }),
      targetStep: permissionRequestStepTarget(step.key),
      total: step.key === "request" ? undefined : step.total
    };
  }) ?? fallbackProcessSteps.map((step) => ({
    detail: step.detail,
    key: step.key,
    labelKey: step.labelKey,
    status: step.complete ? "complete" : currentWizardStep === step.key ? "current" : "waiting",
    targetStep: step.key,
    count: undefined,
    total: undefined
  }));
  const accessSubjectCatalog = normalizeAccessSubjectOptions(accessSubjects);
  const selectedAccessSubject = accessSubjectOptionForSelectorFrom(accessSubjectCatalog, form.subjectSelector);
  const accessSubjectChoices = selectedAccessSubject.kind === "custom"
    ? [...accessSubjectCatalog, selectedAccessSubject]
    : accessSubjectCatalog;
  const accessSubjectDropdownOptions = accessSubjectChoices.map((option) => accessSubjectDropdownOption(option, t));
  const liveDataBlocked = !liveDataAvailable;
  const quickSecondaryActionLabel = goLiveReady
    ? t("action.openAccessProfile")
    : runtimeValidationReady ? t("action.openAcceptanceDetails") : t("action.openProcessDetails");
  const quickSecondaryActionDisabled = goLiveReady ? permissionRequestBusy : false;
  const runQuickSecondaryAction = goLiveReady
    ? onOpenAccessProfile
    : runtimeValidationReady ? scrollToAcceptanceDetails : () => scrollToPermissionRequestStep(currentWizardStep);
  const goLivePrimaryActionKey = goLiveReady
    ? "action.exportProductionEvidence"
    : runtimeValidationReady
      ? "action.checkProductionReadiness"
      : "action.runApprovalJourney";
  const goLivePrimaryActionIcon = goLivePrimaryActionKey === "action.exportProductionEvidence"
    ? <Download size={14} />
    : goLivePrimaryActionKey === "action.checkProductionReadiness"
      ? <RefreshCw size={14} />
      : <Workflow size={14} />;
  const goLivePrimaryActionLabel = goLivePrimaryActionKey === "action.exportProductionEvidence" && productionEvidenceExporting
    ? t("action.exportingProductionEvidence")
    : goLivePrimaryActionKey === "action.checkProductionReadiness" && productionReadinessLoading
      ? t("action.checkingProductionReadiness")
      : goLivePrimaryActionKey === "action.runApprovalJourney" && approvalJourneyRunning
        ? t("action.runningApprovalJourney")
        : t(goLivePrimaryActionKey);
  const runGoLivePrimaryAction = () => {
    if (goLivePrimaryActionKey === "action.exportProductionEvidence") {
      onExportProductionEvidence();
      return;
    }
    if (goLivePrimaryActionKey === "action.checkProductionReadiness") {
      onRefreshProductionReadiness();
      return;
    }
    onRunApprovalJourney();
  };
  const approvalDecisionConfirmation = pendingApprovalDecision ? (
    <div className="approval-decision-confirmation" role="group" aria-label={t("text.approvalDecisionConfirmTitle")}>
      <div>
        <strong>
          {pendingApprovalDecision.action === "approve"
            ? t("action.confirmApprovePermissionRequest")
            : pendingApprovalDecision.action === "withdraw"
              ? t("action.confirmWithdrawPermissionRequest")
              : t("action.confirmRejectPermissionRequest")}
        </strong>
        <span>{pendingApprovalDecision.action === "withdraw"
          ? t("text.approvalWithdrawLine")
          : tx(t, "text.approvalDecisionReviewerLine", { reviewer: reviewerIdentity })}
        </span>
      </div>
      <label>
        {pendingApprovalDecision.action === "reject"
          ? t("form.approvalRejectReason")
          : pendingApprovalDecision.action === "withdraw"
            ? t("form.approvalWithdrawReason")
            : t("form.approvalDecisionComment")}
        <textarea
          rows={2}
          value={pendingApprovalDecision.comment}
          onChange={(event) => updatePendingApprovalComment(event.target.value)}
        />
        <small>
          {pendingApprovalDecision.action === "reject"
            ? t("text.approvalRejectReasonHelp")
            : pendingApprovalDecision.action === "withdraw"
              ? t("text.approvalWithdrawHelp")
              : t("text.approvalApproveCommentHelp")}
        </small>
      </label>
      {pendingApprovalDecision.error ? <span className="approval-inline-message status-error">{pendingApprovalDecision.error}</span> : null}
      <div className="approval-actions">
        <button
          className={`approval-action-button ${pendingApprovalDecision.action === "approve" ? "is-primary" : pendingApprovalDecision.action === "reject" ? "is-danger" : ""}`}
          disabled={liveDataBlocked || permissionRequestBusy}
          onClick={confirmPendingApprovalDecision}
          type="button"
        >
          {pendingApprovalDecision.action === "approve"
            ? <CheckCircle2 size={14} />
            : pendingApprovalDecision.action === "withdraw"
              ? <Undo2 size={14} />
              : <TriangleAlert size={14} />}
          {pendingApprovalDecision.action === "approve"
            ? t("action.confirmApprovePermissionRequest")
            : pendingApprovalDecision.action === "withdraw"
              ? t("action.confirmWithdrawPermissionRequest")
              : t("action.confirmRejectPermissionRequest")}
        </button>
        <button className="secondary-button" disabled={Boolean(approvalAction)} onClick={cancelApprovalDecision} type="button">
          {t("action.cancelApprovalDecision")}
        </button>
      </div>
    </div>
  ) : null;
  return (
    <div className={`approval-studio status-${productionSummary.status}`} id="permission-package-workbench">
      <section className="approval-header">
        <div className="approval-title">
          <span>{t("section.permissionRequestTask")}</span>
          <h2>{t("text.permissionRequestTaskTitle")}</h2>
          <p>{t("text.permissionRequestTaskBody")}</p>
        </div>
        <div className="approval-command">
          <span>{t("text.permissionRequestActions")}</span>
          <div className="approval-actions">
            <button
              className="primary-button"
              data-primary-approval-action
              disabled={liveDataBlocked || permissionRequestBusy}
              onClick={runProductionPrimaryAction}
              type="button"
            >
              <CheckCircle2 size={14} />
              {t(journeyStatus.nextActionKey)}
            </button>
            <button className="secondary-button" disabled={quickSecondaryActionDisabled} onClick={runQuickSecondaryAction} type="button">
              {goLiveReady ? <FileSearch size={14} /> : <Workflow size={14} />}
              {quickSecondaryActionLabel}
            </button>
          </div>
        </div>
      </section>

      {liveDataBlocked ? (
        <section className="approval-live-warning" role="status" aria-live="polite">
          <TriangleAlert size={16} />
          <div>
            <strong>{t("message.fallbackDataModeTitle")}</strong>
            <span>{t("message.fallbackDataModeDetail")}</span>
          </div>
        </section>
      ) : null}

      <section className="approval-context-bar" aria-label={t("text.currentWorkspaceContext")}>
        <div>
          <span>{t("form.businessTenant")}</span>
          <strong>{tenantPath.primary}</strong>
        </div>
        <div>
          <span>{t("form.businessWorkspace")}</span>
          <strong>{workspaceName}</strong>
        </div>
        <div>
          <span>{t("form.businessCaller")}</span>
          <strong>{callerName}</strong>
        </div>
      </section>

      <section className="approval-overview" aria-label={t("text.permissionJourneyStatus")}>
        <div className="approval-task-strip">
          <article>
            <span>{t("text.currentStatus")}</span>
            <strong>{t(journeyStatus.labelKey)}</strong>
            <small>{t(journeyStatus.detailKey)}</small>
          </article>
          <article>
            <span>{t("form.permissionPackage")}</span>
            <strong>{permissionPackageTemplateName(draft.template, t)}</strong>
            <small>
              {draft.allowedCapabilities.length} {t("detail.allowed")} / {draft.blockedCapabilities.length} {t("detail.denied")}
            </small>
          </article>
          <article>
            <span>{t("text.permissionRequestNextAction")}</span>
            <strong>{t(journeyStatus.nextActionKey)}</strong>
            <small>{readinessReadyCount}/{readinessTotalCount} {t("text.checks")}</small>
          </article>
        </div>
      </section>

      <div className="approval-flow-layout">
        <main className="approval-request-panel">
          <section className={`approval-section approval-request-form-section ${requestFormLocked ? "is-read-only" : ""}`} id={permissionRequestStepSectionId("scope")}>
            <header>
              <div>
                <strong>{t(requestFormTitleKey)}</strong>
                <p>{t(requestFormHelpKey)}</p>
              </div>
              <Badge tone={journeyStatus.tone}>{t(journeyStatus.labelKey)}</Badge>
            </header>
            {requestFormLocked ? (
              <div className="approval-lock-notice">
                <div>
                  <strong>{t("text.permissionRequestLockedTitle")}</strong>
                  <span>{t(requestFormLockedDetailKey)}</span>
                </div>
                {requestFormActiveLocked ? (
                  <button className="secondary-button" disabled={permissionRequestBusy} onClick={onStartNewPermissionChange} type="button">
                    <ClipboardCheck size={14} />
                    {t("action.startPermissionApproval")}
                  </button>
                ) : null}
              </div>
            ) : null}
            <details className="approval-concept-guide">
              <summary>{t("section.permissionConceptGuide")}</summary>
              <div className="approval-concept-grid">
                <article>
                  <strong>{t("concept.tenant")}</strong>
                  <span>{t("concept.tenant.detail")}</span>
                </article>
                <article>
                  <strong>{t("concept.caller")}</strong>
                  <span>{t("concept.caller.detail")}</span>
                </article>
                <article>
                  <strong>{t("concept.permissionPackage")}</strong>
                  <span>{t("concept.permissionPackage.detail")}</span>
                </article>
                <article>
                  <strong>{t("concept.evidence")}</strong>
                  <span>{t("concept.evidence.detail")}</span>
                </article>
              </div>
            </details>
            <label className="approval-request">
              {t("form.adminRequest")}
              <textarea
                disabled={requestFormLocked}
                rows={3}
                value={form.requestText}
                onChange={(event) => onChange({ ...form, requestText: event.target.value })}
              />
            </label>
            <div className="approval-form-grid">
              <div className="approval-field is-wide">
                <span className="approval-field-label">{t("form.businessTenant")}</span>
                <ApprovalDropdown
                  disabled={requestFormLocked}
                  label={t("form.businessTenant")}
                  options={tenantDropdownOptions}
                  value={form.tenantId}
                  onChange={(value) => onChange({ ...form, tenantId: value })}
                />
                <span>{tenantPath.path}</span>
              </div>
              <div className="approval-readonly-field is-wide">
                <span>{t("form.businessWorkspace")}</span>
                <strong>{workspaceName}</strong>
                <small>{t("text.workspaceResolvedDetail")}</small>
              </div>
              <div className="approval-field">
                <span className="approval-field-label">{t("form.businessCaller")}</span>
                <ApprovalDropdown
                  disabled={requestFormLocked}
                  label={t("form.businessCaller")}
                  options={callerDropdownOptions}
                  value={form.callerInstanceId}
                  onChange={(value) => onChange({ ...form, callerInstanceId: value })}
                />
              </div>
              <div className="approval-field">
                <span className="approval-field-label">{t("form.target")}</span>
                <ApprovalDropdown
                  disabled={requestFormLocked}
                  label={t("form.target")}
                  options={targetDropdownOptions}
                  value={form.targetId}
                  onChange={(value) => onChange({ ...form, targetId: value })}
                />
              </div>
              <div className="approval-field approval-subject-field is-wide">
                <span className="approval-field-label">{t("form.accessSubject")}</span>
                <ApprovalDropdown
                  disabled={requestFormLocked}
                  label={t("form.accessSubject")}
                  options={accessSubjectDropdownOptions}
                  value={selectedAccessSubject.id}
                  onChange={(value) => onChange(applyPermissionRequestAccessSubject(form, accessSubjectCatalog, value))}
                />
                <small>{t(selectedAccessSubject.detailKey)}</small>
              </div>
              <label>
                {t("form.region")}
                <input disabled={requestFormLocked} value={form.region} onChange={(event) => onChange({ ...form, region: event.target.value })} />
              </label>
              <div className="approval-select is-wide">
                <span className="approval-field-label">{t("form.permissionPackage")}</span>
                <ApprovalDropdown
                  disabled={requestFormLocked}
                  label={t("form.permissionPackage")}
                  options={templateDropdownOptions}
                  value={form.templateId}
                  onChange={(value) => onChange({ ...form, templateId: value })}
                />
              </div>
            </div>
            <div className="approval-package-preview" id={permissionRequestStepSectionId("template")}>
              <div>
                <span>{t("section.permissionWizardTemplate")}</span>
                <strong>{permissionPackageTemplateName(draft.template, t)}</strong>
                <p>{permissionPackageTemplateSummary(draft.template, t)}</p>
              </div>
              <div className="approval-capability-columns">
                <CapabilityChipList
                  capabilities={draft.allowedCapabilities}
                  emptyLabel={t("empty.permissionAllowed.detail")}
                  label={t("section.allowedByPackage")}
                  tone="success"
                  t={t}
                />
                <CapabilityChipList
                  capabilities={draft.blockedCapabilities}
                  emptyLabel={t("empty.permissionBlocked.detail")}
                  label={t("section.blockedByPackage")}
                  tone="danger"
                  t={t}
                />
              </div>
              <div className="approval-scope">
                <span>{t("section.dataScope")}</span>
                <code>{summarizeDataScopes(draft.dataScopes, t("text.noDataScope"), dataScopeLabels)}</code>
              </div>
            </div>
            <details className="approval-details">
              <summary>{t("text.technicalOverrides")}</summary>
              <label>
                {t("form.workspaceId")}
                <input disabled={requestFormLocked} value={form.workspaceId} onChange={(event) => onChange({ ...form, workspaceId: event.target.value })} />
              </label>
              <label>
                {t("form.subjectSelector")}
                <input
                  disabled={requestFormLocked}
                  placeholder={t("form.subjectSelectorPlaceholder")}
                  value={form.subjectSelector ?? ""}
                  onChange={(event) => onChange({ ...form, subjectSelector: event.target.value })}
                />
                <small>{t("text.subjectSelectorAdvancedHelp")}</small>
              </label>
              <dl>
                <div>
                  <dt>{t("form.tenantId")}</dt>
                  <dd>{form.tenantId || "-"}</dd>
                </div>
                <div>
                  <dt>{t("form.workspaceId")}</dt>
                  <dd>{form.workspaceId || "-"}</dd>
                </div>
                <div>
                  <dt>{t("form.callerInstance")}</dt>
                  <dd>{selectedCaller?.id || form.callerInstanceId || "-"}</dd>
                </div>
                <div>
                  <dt>{t("form.target")}</dt>
                  <dd>{selectedTarget?.id || form.targetId || "-"}</dd>
                </div>
              </dl>
            </details>
          </section>
        </main>

        <aside className="approval-process-panel" aria-label={t("section.permissionRequestProcess")}>
          <div className="approval-process-header">
            <div>
              <span>{t("section.permissionRequestProcess")}</span>
              <strong>{t(journeyStatus.labelKey)}</strong>
            </div>
          </div>
          <div className="approval-process-list">
            {processSteps.map((step, index) => {
              const stepLabel = t(step.labelKey);
              return (
                <button
                  aria-current={step.status === "current" ? "step" : undefined}
                  aria-label={tx(t, "text.permissionProcessStepAria", { detail: step.detail, index: index + 1, label: stepLabel })}
                  className={`approval-process-step status-${step.status}`}
                  data-step-target={step.targetStep}
                  key={step.key}
                  onClick={() => scrollToPermissionRequestStep(step.targetStep)}
                  type="button"
                >
                  <span>{index + 1}</span>
                  <div>
                    <strong>{stepLabel}</strong>
                    <small>{step.detail}</small>
                  </div>
                  {typeof step.count === "number" && typeof step.total === "number" ? <em>{step.count}/{step.total}</em> : null}
                </button>
              );
            })}
          </div>
          <section className="approval-process-block" id={permissionRequestStepSectionId("approval")}>
            <header>
              <strong>{t("section.permissionWizardApproval")}</strong>
              <Badge tone={approvalDisplayTone}>{approvalDisplayLabel}</Badge>
            </header>
            <div className="approval-decision">
              <strong>{t(approvalGateDetailKey)}</strong>
              {showPolicyGateReasons && draft.policyGate.reasons.length > 0 ? (
                <ul>
                  {draft.policyGate.reasons.slice(0, 2).map((reason) => (
                    <li key={reason.id}>{permissionPolicyReasonMessage(reason, t)}</li>
                  ))}
                </ul>
              ) : null}
            </div>
            {!draft.policyGate.canApplyDirectly ? (
              <div className="approval-reviewer-context">
                <span>{tx(t, "text.approvalReviewerIdentity", { reviewer: reviewerIdentity })}</span>
                <small>{t("text.approvalReviewerSeparationDetail")}</small>
              </div>
            ) : null}
            {!draft.policyGate.canApplyDirectly ? (
              <div className="approval-request-state">
                <div>
                  <span>{t("section.permissionApprovalRequest")}</span>
                  <strong>{approvalDisplayLabel}</strong>
                  {approvalRequest?.expiresAt && !approvalEffectivelyResolved ? <small>{tx(t, "text.approvalExpiresAt", { date: formatDate(approvalRequest.expiresAt) })}</small> : null}
                </div>
                <div className="approval-actions">
                  {showCreateApprovalAction ? (
                    <button
                      className="approval-action-button is-primary"
                      disabled={liveDataBlocked || permissionRequestBusy || !draft.readiness.canApply}
                      onClick={onCreateApprovalRequest}
                      type="button"
                    >
                      <ClipboardCheck size={14} />
                      {approvalAction === "create" ? t("action.creatingApprovalRequest") : t("action.createApprovalRequest")}
                    </button>
                  ) : null}
                  {showPendingApprovalActions ? (
                    <>
                      <button className="approval-action-button is-primary" disabled={liveDataBlocked || permissionRequestBusy} onClick={() => beginApprovalDecision("approve")} type="button">
                        <CheckCircle2 size={14} />
                        {approvalAction === "approve" ? t("action.approving") : t("action.approvePermissionRequest")}
                      </button>
                      <button className="approval-action-button is-danger" disabled={liveDataBlocked || permissionRequestBusy} onClick={() => beginApprovalDecision("reject")} type="button">
                        <TriangleAlert size={14} />
                        {approvalAction === "reject" ? t("action.rejecting") : t("action.rejectPermissionRequest")}
                      </button>
                      <button className="approval-action-button" disabled={liveDataBlocked || permissionRequestBusy} onClick={() => beginApprovalDecision("withdraw")} type="button">
                        <Undo2 size={14} />
                        {approvalAction === "withdraw" ? t("action.withdrawing") : t("action.withdrawPermissionRequest")}
                      </button>
                    </>
                  ) : null}
                </div>
              </div>
            ) : null}
            {approvalDecisionConfirmation}
          </section>
          <section className="approval-process-block" id={permissionRequestStepSectionId("apply")}>
            <div className="approval-apply">
              <div>
                <strong>{application ? t("text.permissionAppliedTitle") : canApply ? t("text.permissionApplyReadyTitle") : t("text.permissionApplyWaitingTitle")}</strong>
                <span>{application
                  ? tx(t, "text.permissionAppliedDetail", {
                    count: application.tenantEntitlementIds.length + application.workspaceAssignmentIds.length + application.instanceAssignmentIds.length
                  })
                  : canApply ? t("text.permissionApplyReadyDetail") : t("text.permissionApplyWaitingDetail")}
                </span>
              </div>
              {application ? (
                <span className="approval-action-status is-complete">
                  <CheckCircle2 size={14} />
                  {t("action.permissionPackageApplied")}
                </span>
              ) : (
                <button className="primary-button" disabled={liveDataBlocked || permissionRequestBusy || !canApply} onClick={onApply} type="button">
                  <CheckCircle2 size={14} />
                  {applying ? t("action.applyingPermissionPackage") : t("action.applyPermissionPackage")}
                </button>
              )}
            </div>
            {message ? <span className={`approval-inline-message status-${messageTone}`}>{message}</span> : null}
          </section>

          <section className="approval-process-block approval-go-live-block" id={permissionRequestStepSectionId("goLive")}>
            <header>
              <strong>{t("section.permissionWizardGoLive")}</strong>
              <Badge tone={goLiveReady ? "success" : "warning"}>
                {goLiveReady ? t("status.productionReady") : tx(t, "text.aiAdminGoLiveRemainingBadge", { count: goLiveReadiness.remainingCount })}
              </Badge>
            </header>
            <p>{goLiveReady ? t("text.aiAdminGoLiveReadyDetail") : t("text.aiAdminGoLiveWaitingDetail")}</p>
            {goLiveReady ? (
              <div className="approval-completion" id={permissionRequestStepSectionId("acceptance")}>
                <div>
                  <CheckCircle2 size={18} />
                  <div>
                    <strong>{t("text.permissionChangeCompleteTitle")}</strong>
                    <span>{t("text.permissionChangeCompleteDetail")}</span>
                    {goLiveCompletedAt ? <small>{goLiveCompletedAt}</small> : null}
                  </div>
                </div>
                <div className="approval-completion-actions">
                  <button className="secondary-button" disabled={liveDataBlocked || permissionRequestBusy} onClick={onExportProductionEvidence} type="button">
                    <Download size={14} />
                    {productionEvidenceExporting ? t("action.exportingProductionEvidence") : t("action.downloadAcceptanceReport")}
                  </button>
                  <button className="secondary-button" disabled={permissionRequestBusy} onClick={onOpenAccessProfile} type="button">
                    <FileSearch size={14} />
                    {t("action.openAccessProfile")}
                  </button>
                  <button className="secondary-button" disabled={permissionRequestBusy} onClick={onStartNewPermissionChange} type="button">
                    <ClipboardCheck size={14} />
                    {t("action.startPermissionApproval")}
                  </button>
                </div>
              </div>
            ) : (
              <>
                <div className="approval-next-line" id={permissionRequestStepSectionId("acceptance")}>
                  <span>{t("text.nextActions")}</span>
                  <strong>{t(goLiveNextKey)}</strong>
                </div>
                <div className="approval-actions">
                  <button className="primary-button" disabled={liveDataBlocked || permissionRequestBusy} onClick={runGoLivePrimaryAction} type="button">
                    {goLivePrimaryActionIcon}
                    {goLivePrimaryActionLabel}
                  </button>
                </div>
              </>
            )}
            <div className="approval-runtime" id={permissionRequestStepSectionId("validation")}>
              <strong>{t("text.runtimeValidationResultTitle")}</strong>
              <span>{runtimeValidationReady ? t("text.runtimeValidationResultReady") : t("text.runtimeValidationResultPending")}</span>
              {approvalJourneyMessage ? <em>{approvalJourneyMessage}</em> : null}
            </div>
          </section>
        </aside>
      </div>

      <details className="approval-evidence" id="permission-request-acceptance-details">
        <summary>
          <div>
            <strong>{t("section.permissionAdvancedChecks")}</strong>
            <span>{t("text.permissionRequestAdvancedSummary")}</span>
          </div>
          <Badge tone={productionConsoleStatusTone(productionSummary.status)}>
            {productionSummary.readyCount}/{productionSummary.totalCount}
          </Badge>
        </summary>
        <div className="approval-evidence-grid">
          <section>
            <header>
              <strong>{t("section.aiAdminReadiness")}</strong>
              <button className="secondary-button" disabled={permissionRequestBusy} onClick={onRefreshApprovalReadiness} type="button">
                <RefreshCw size={14} />
                {approvalReadinessChecking ? t("action.checkingApprovalReadiness") : t("action.checkApprovalReadiness")}
              </button>
            </header>
            {approvalReadinessMessage && shouldShowAdvancedStatusMessage(approvalReadinessMessageTone) ? <span className={`approval-inline-message status-${approvalReadinessMessageTone}`}>{approvalReadinessMessage}</span> : null}
            <div className="approval-mini-list">
              {readinessRows.map((row) => (
                <div className={`approval-mini-row status-${row.status}`} key={row.key}>
                  <strong>{t(row.titleKey)}</strong>
                  <span>{t(row.detailKey)}</span>
                </div>
              ))}
            </div>
          </section>

          <section>
            <header>
              <strong>{t("section.permissionApplyPreflight")}</strong>
              <button className="secondary-button" disabled={liveDataBlocked || permissionRequestBusy} onClick={onRefreshApplyPreflight} type="button">
                <RefreshCw size={14} />
                {applyPreflightLoading ? t("action.checkingPreflight") : t("action.checkPreflight")}
              </button>
            </header>
            {applyPreflightMessage && shouldShowAdvancedStatusMessage(applyPreflightMessageTone) ? <span className={`approval-inline-message status-${applyPreflightMessageTone}`}>{applyPreflightMessage}</span> : null}
            <div className="approval-metrics">
              <div>
                <span>{t("metric.preflightBlockers")}</span>
                <strong>{preflightSummary?.blockingCount ?? 0}</strong>
              </div>
              <div>
                <span>{t("metric.preflightWarnings")}</span>
                <strong>{preflightSummary?.warningCount ?? 0}</strong>
              </div>
              <div>
                <span>{t("metric.preflightPlannedObjects")}</span>
                <strong>
                  {preflightSummary
                    ? preflightSummary.plannedTenantEntitlementCount + preflightSummary.plannedWorkspaceAssignmentCount + preflightSummary.plannedInstanceAssignmentCount
                    : 0}
                </strong>
              </div>
            </div>
          </section>

          <section className="approval-reviewer-queue">
            <header>
              <strong>{t(reviewerQueueTitleKey)}</strong>
              <button className="secondary-button" disabled={liveDataBlocked || permissionRequestBusy} onClick={onRefreshReviewerQueue} type="button">
                <RefreshCw size={14} />
                {reviewerQueueLoading ? t("action.loading") : t(reviewerQueueRefreshKey)}
              </button>
            </header>
            <label className="approval-select">
              {t("form.approvalReviewer")}
              <input value={approvalReviewer} onChange={(event) => onApprovalReviewerChange(event.target.value)} />
            </label>
            {reviewerQueueMessage && shouldShowAdvancedStatusMessage(reviewerQueueMessageTone) ? <span className={`approval-inline-message status-${reviewerQueueMessageTone}`}>{reviewerQueueMessage}</span> : null}
            {reviewerQueueReadOnly ? <span className="approval-inline-message status-info">{t("text.reviewerQueueReadOnlyDetail")}</span> : null}
            <div className="approval-mini-list">
              {reviewerQueueRequests.length === 0 ? (
                <EmptyRow title={t(reviewerQueueTitleKey)} detail={t("empty.reviewerQueue.detail")} />
              ) : null}
              {reviewerQueueRequests.map((request) => {
                const queueLabel = permissionApprovalRequestBusinessLabel(request, templates, tenants, agents, t);
                return (
                  <article className={`approval-review-row ${request.id === selectedApprovalRequestId ? "is-selected" : ""}`} key={request.id}>
                    <button onClick={() => onSelectApprovalRequest(request.id)} type="button">
                      <span className="approval-review-row-main">
                        <strong>{queueLabel.template}</strong>
                        <small>{tx(t, "text.permissionQueueBusinessScope", { caller: queueLabel.caller, target: queueLabel.target, tenant: queueLabel.tenant })}</small>
                      </span>
                      <span className="approval-review-row-meta">{tx(t, "text.permissionQueueExpires", { date: formatDate(request.expiresAt) })}</span>
                    </button>
                    {reviewerQueueReadOnly ? (
                      <span className="approval-review-row-state">{t("text.reviewerQueueReadOnlyAction")}</span>
                    ) : (
                      <div>
                        <button className="approval-action-button is-primary" disabled={liveDataBlocked || permissionRequestBusy} onClick={() => beginApprovalDecision("approve", request.id)} type="button">
                          <CheckCircle2 size={13} />
                          {t("action.approvePermissionRequest")}
                        </button>
                        <button className="approval-action-button is-danger" disabled={liveDataBlocked || permissionRequestBusy} onClick={() => beginApprovalDecision("reject", request.id)} type="button">
                          <TriangleAlert size={13} />
                          {t("action.rejectPermissionRequest")}
                        </button>
                      </div>
                    )}
                    <details className="approval-details">
                      <summary>{t("text.reviewerQueueTraceDetails")}</summary>
                      <div className="approval-review-row-technical">
                        <TechnicalId label={t("table.request")} value={request.id} />
                        <TechnicalId label={t("form.tenantId")} value={request.tenantId} />
                        <TechnicalId label={t("form.workspaceId")} value={request.workspaceId} />
                        <TechnicalId label={t("form.callerInstance")} value={request.callerInstanceId} />
                        <TechnicalId label={t("form.target")} value={request.targetId} />
                      </div>
                    </details>
                  </article>
                );
              })}
            </div>
          </section>

          <section>
            <header>
              <strong>{t("section.permissionProductionReadiness")}</strong>
              <div className="approval-actions">
                <button className="secondary-button" disabled={liveDataBlocked || permissionRequestBusy} onClick={onRefreshProductionReadiness} type="button">
                  <RefreshCw size={14} />
                  {productionReadinessLoading ? t("action.checkingProductionReadiness") : t("action.checkProductionReadiness")}
                </button>
                <button className="secondary-button" disabled={liveDataBlocked || !productionReadiness || permissionRequestBusy} onClick={onExportProductionEvidence} type="button">
                  <Download size={14} />
                  {productionEvidenceExporting ? t("action.exportingProductionEvidence") : t("action.exportProductionEvidence")}
                </button>
              </div>
            </header>
            {productionReadinessMessage && shouldShowAdvancedStatusMessage(productionReadinessMessageTone) ? <span className={`approval-inline-message status-${productionReadinessMessageTone}`}>{productionReadinessMessage}</span> : null}
            <div className="approval-mini-list">
              {productionChecks.length === 0 ? (
                <EmptyRow title={t("section.permissionProductionReadiness")} detail={t("empty.permissionProductionReadiness.detail")} />
              ) : null}
              {productionChecks.map((check) => (
                <div className={`approval-mini-row severity-${check.severity}`} key={check.code}>
                  <strong>{permissionProductionReadinessCheckLabel(check.code, t)}</strong>
                  <span>{permissionProductionReadinessCheckMessage(check, t)}</span>
                </div>
              ))}
            </div>
          </section>

          <section>
            <header>
              <strong>{t("section.accessDecisionExplain")}</strong>
              <button className="secondary-button" disabled={liveDataBlocked || permissionRequestBusy} onClick={onExplainAccessDecision} type="button">
                <RefreshCw size={14} />
                {accessDecisionExplanationLoading ? t("action.loading") : t("action.explainAccessDecision")}
              </button>
            </header>
            {accessDecisionExplanationMessage && shouldShowAdvancedStatusMessage(accessDecisionExplanationMessageTone) ? <span className={`approval-inline-message status-${accessDecisionExplanationMessageTone}`}>{accessDecisionExplanationMessage}</span> : null}
            <div className="approval-metrics">
              <div>
                <span>{t("table.decision")}</span>
                <strong>{accessDecisionExplanation ? accessDecisionOutcomeLabel(accessDecisionExplanation.outcome, t) : "-"}</strong>
              </div>
              <div>
                <span>{t("section.permissionApplicationHealth")}</span>
                <strong>{healthRows.length}</strong>
              </div>
              <div>
                <span>{t("section.permissionApplicationImpact")}</span>
                <strong>{impactSummary ? impactSummary.createdObjectCount + impactSummary.activeObjectCount : 0}</strong>
              </div>
            </div>
            <div className="approval-actions">
              <button className="secondary-button" disabled={liveDataBlocked || permissionRequestBusy} onClick={onRefreshApplicationHealth} type="button">
                <RefreshCw size={14} />
                {applicationHealthLoading ? t("action.loading") : t("action.refreshApplicationHealth")}
              </button>
              <button className="secondary-button" disabled={liveDataBlocked || permissionRequestBusy || !application} onClick={onReviewApplicationImpact} type="button">
                <ClipboardCheck size={14} />
                {applicationImpactLoading ? t("action.loading") : t("action.reviewApplicationImpact")}
              </button>
              <button className="secondary-button" disabled={liveDataBlocked || permissionRequestBusy || !application} onClick={onRehearseApplicationDrift} type="button">
                <TriangleAlert size={14} />
                {t("action.rehearseApplicationDrift")}
              </button>
            </div>
            {applicationHealthMessage && shouldShowAdvancedStatusMessage(applicationHealthMessageTone) ? <span className={`approval-inline-message status-${applicationHealthMessageTone}`}>{applicationHealthMessage}</span> : null}
            {applicationImpactMessage && shouldShowAdvancedStatusMessage(applicationImpactMessageTone) ? <span className={`approval-inline-message status-${applicationImpactMessageTone}`}>{applicationImpactMessage}</span> : null}
            {healthRows.length > 0 ? (
              <div className="approval-mini-list">
                {healthRows.slice(0, 3).map((row) => (
                  <button className={`approval-mini-row status-${row.status}`} key={row.application.id} onClick={() => onReviewApplicationHealthRow(row.application)} type="button">
                    <strong>{permissionApplicationHealthLabel(row.status, t)}</strong>
                    <span>{permissionApplicationHealthRowSummary(row, t)}</span>
                  </button>
                ))}
              </div>
            ) : null}
          </section>

          <section>
            <header>
              <strong>{t("section.aiAdminApprovalJourney")}</strong>
              <Badge tone={approvalJourneyEvaluation.completeCount === approvalJourneyEvaluation.totalCount ? "success" : "neutral"}>
                {approvalJourneyEvaluation.completeCount}/{approvalJourneyEvaluation.totalCount}
              </Badge>
            </header>
            <div className="approval-mini-list">
              {approvalJourneyEvaluation.steps.map((step) => (
                <div className={`approval-mini-row status-${step.status}`} key={step.key}>
                  <strong>{t(`journey.aiAdmin.step.${step.key}`)}</strong>
                  <span>{step.status === "complete" ? t("journey.aiAdmin.evidence.ready") : t(`journey.aiAdmin.next.${step.key}`)}</span>
                </div>
              ))}
            </div>
            {approvalAuditEvent ? (
              <details className="approval-details">
                <summary>{t("text.aiAdminGoLiveTechDetails")}</summary>
                <code>{approvalAuditEvent.id}</code>
              </details>
            ) : null}
          </section>
        </div>
      </details>
    </div>
  );
}

function tx(t: Translator, key: string, values: Record<string, string | number>) {
  return Object.entries(values).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    t(key)
  );
}

function accessSubjectDropdownOption(option: AccessSubjectOption, t: Translator): ApprovalDropdownOption {
  return {
    label: `${t(accessSubjectKindLabelKey(option.kind))} · ${t(option.labelKey)}`,
    value: option.id
  };
}

function accessSubjectKindLabelKey(kind: AccessSubjectOption["kind"]) {
  if (kind === "department") return "accessSubject.kind.department";
  if (kind === "member") return "accessSubject.kind.member";
  if (kind === "custom") return "accessSubject.kind.custom";
  return "accessSubject.kind.role";
}

function CapabilityChipList({
  capabilities,
  emptyLabel,
  label,
  tone,
  t
}: {
  capabilities: Capability[];
  emptyLabel: string;
  label: string;
  tone: Tone;
  t: Translator;
}) {
  return (
    <div className={`approval-capability-list tone-${tone}`}>
      <strong>{label}</strong>
      {capabilities.length === 0 ? <span>{emptyLabel}</span> : null}
      <div>
        {capabilities.map((capability) => (
          <span key={capability.id}>
            {capabilityDisplayName(capability, t)} · {t(`value.${capability.action}`, capability.action)}
          </span>
        ))}
      </div>
    </div>
  );
}

function accessDecisionOutcomeLabel(outcome: AccessDecisionExplainResult["outcome"], t: Translator) {
  return outcome === "allowed" ? t("text.decisionAllowed") : t("text.decisionDenied");
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  });
}


function permissionTenantPathLabel(tenantId: string, tenants: Tenant[], t: Translator): { path: string; primary: string } {
  const normalizedTenantId = tenantId.trim();
  if (!normalizedTenantId) return { path: "-", primary: t("text.unresolvedTenant") };
  const tenantById = tenants.reduce<Record<string, Tenant>>((acc, tenant) => {
    acc[tenant.id] = tenant;
    return acc;
  }, {});
  const selected = tenantById[normalizedTenantId];
  if (!selected) {
    if (normalizedTenantId === "default") {
      const defaultTenantName = t("text.defaultTenantName");
      return { path: defaultTenantName, primary: defaultTenantName };
    }
    return {
      path: tx(t, "text.unresolvedTenantDetail", { id: normalizedTenantId }),
      primary: t("text.unresolvedTenant")
    };
  }

  const path: Tenant[] = [];
  const visited = new Set<string>();
  let current: Tenant | undefined = selected;
  while (current && !visited.has(current.id)) {
    path.unshift(current);
    visited.add(current.id);
    current = current.parentTenantId ? tenantById[current.parentTenantId] : undefined;
  }
  const names = path.map((tenant) => permissionEntityDisplayName(tenant.name.trim() || tenant.id, t));
  return {
    path: names.join(" > "),
    primary: permissionEntityDisplayName(selected.name.trim() || selected.id, t)
  };
}

function permissionWorkspaceDisplayName(workspaceId: string, agents: Agent[], t: Translator) {
  const normalizedWorkspaceId = workspaceId.trim();
  if (!normalizedWorkspaceId) return "-";
  const agentInWorkspace = agents.find((agent) => agent.workspaceId === normalizedWorkspaceId);
  if (normalizedWorkspaceId === defaultWorkspaceId || agentInWorkspace?.workspaceId === defaultWorkspaceId) {
    return t("text.defaultWorkspaceName");
  }
  if (/permission[-_]?(request|package)[-_]?approval/i.test(normalizedWorkspaceId)) {
    return t("demo.permissionRequestWorkspace");
  }
  if (/core[-_]?journey/i.test(normalizedWorkspaceId)) {
    return t("demo.coreJourneyWorkspace");
  }
  return permissionEntityDisplayName(readableIdentifierLabel(normalizedWorkspaceId), t);
}

function uniquePermissionEntityOptions<T extends { id: string }>(
  entities: T[],
  selectedId: string,
  labelFor: (entity: T) => string
): Array<{ id: string; label: string }> {
  const selected = entities.find((entity) => entity.id === selectedId);
  const ordered = selected ? [selected, ...entities.filter((entity) => entity.id !== selected.id)] : entities;
  const seen = new Set<string>();
  const options: Array<{ id: string; label: string }> = [];

  for (const entity of ordered) {
    const label = labelFor(entity).trim() || entity.id;
    const key = label.toLocaleLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    options.push({ id: entity.id, label });
  }

  return options;
}

function readableIdentifierLabel(value: string) {
  const normalized = value.trim().replace(/^ws[-_]/, "").replace(/^workspace[-_]/, "");
  if (!normalized) return value;
  return normalized
    .split(/[-_]+/)
    .filter(Boolean)

    .map((part) => {
      const upper = part.toUpperCase();
      if (["AI", "API", "CRM", "MCP", "UI"].includes(upper)) return upper;
      return `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`;
    })
    .join(" ");
}

function permissionEntityDisplayName(value: string, t: Translator) {
  const normalized = value.trim();
  if (!normalized) return value;
  const demoNames: Record<string, string> = {
    "Core Journey Caller": t("demo.coreJourneyCaller"),
    "Core Journey MCP Target": t("demo.coreJourneyTarget"),
    "Core Journey Project": t("demo.coreJourneyProject"),
    "Core Journey Root": t("demo.coreJourneyRoot"),
    "Core Journey Team": t("demo.coreJourneyTeam"),
    "Permission Package Approval": t("demo.permissionRequestApproval"),
    "Permission Package Approval Root": t("demo.permissionRequestApprovalRoot"),
    "Permission Package Approval Team": t("demo.permissionRequestApprovalTeam"),
    "Permission Package Approval Project": t("demo.permissionRequestApprovalProject"),
    "Permission Package Approval Caller": t("demo.permissionRequestApprovalCaller"),
    "Permission Package Approval MCP Target": t("demo.permissionRequestApprovalTarget"),
    "Permission Request Approval": t("demo.permissionRequestApproval"),
    "Permission Request Approval Root": t("demo.permissionRequestApprovalRoot"),
    "Permission Request Approval Team": t("demo.permissionRequestApprovalTeam"),
    "Permission Request Approval Project": t("demo.permissionRequestApprovalProject"),
    "Permission Request Approval Caller": t("demo.permissionRequestApprovalCaller"),
    "Permission Request Approval MCP Target": t("demo.permissionRequestApprovalTarget"),
    "Security Reviewer": t("accessSubject.securityReviewer.name")
  };
  if (demoNames[normalized]) return demoNames[normalized];
  if (normalized.includes("Permission Package Approval")) {
    return normalized.replaceAll("Permission Package Approval", t("demo.permissionRequestApproval"));
  }
  return normalized;
}

function permissionPackageTemplateName(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.name`, template.name);
}

function permissionPackageTemplateNameById(templateId: string, t: Translator) {
  return t(`permissionPackage.${templateId}.name`, templateId);
}

function permissionApprovalRequestBusinessLabel(
  request: PermissionPackageApprovalRequest,
  templates: PermissionPackageTemplate[],
  tenants: Tenant[],
  agents: Agent[],
  t: Translator
) {
  const template = templates.find((item) => item.id === request.templateId);
  const tenant = tenants.find((item) => item.id === request.tenantId);
  const caller = agents.find((item) => item.id === request.callerInstanceId);
  const target = agents.find((item) => item.id === request.targetId);
  return {
    caller: caller ? permissionEntityDisplayName(caller.name, t) : t("text.unknownCaller"),
    target: target ? permissionEntityDisplayName(target.name, t) : t("text.unknownTarget"),
    template: template ? permissionPackageTemplateName(template, t) : t("text.unknownPermissionPackage"),
    tenant: tenant ? permissionEntityDisplayName(tenant.name, t) : t("text.unknownTenant")
  };
}

function permissionPackageTemplateSummary(template: PermissionPackageTemplate, t: Translator) {
  return t(`permissionPackage.${template.id}.summary`, template.summary);
}

function permissionRequestStepSectionId(step: PermissionRequestStepTarget) {
  return `permission-request-step-${step}`;
}

function permissionRequestStepTarget(step: PermissionPackageWorkbenchStepKey | PermissionRequestWizardStep): PermissionRequestStepTarget {
  if (step === "request") return "scope";
  if (step === "validation") return "validation";
  if (step === "acceptance") return "acceptance";
  return step;
}

function resolvePermissionJourneyStatus(args: {
  approvalRequest: PermissionPackageApprovalRequest | null;
  canApply: boolean;
  draft: PermissionPackageDraft;
  goLiveReady: boolean;
  productionStatus: AiAdminProductionConsoleStatus;
  workbenchStatus?: PermissionPackageWorkbenchPreview["summary"]["status"];
}): { labelKey: string; detailKey: string; tone: Tone; nextActionKey: string } {
  if (args.goLiveReady) {
    return {
      detailKey: "permissionJourney.statusDetail.ready",
      labelKey: "permissionJourney.status.ready",
      nextActionKey: "action.exportProductionEvidence",
      tone: "success"
    };
  }
  if (args.approvalRequest?.status === "pending") {
    return {
      detailKey: "permissionJourney.statusDetail.awaitingApproval",
      labelKey: "permissionJourney.status.awaitingApproval",
      nextActionKey: "action.refreshReviewerQueue",
      tone: "warning"
    };
  }
  if (args.approvalRequest?.status === "rejected") {
    return {
      detailKey: "permissionJourney.statusDetail.rejected",
      labelKey: "permissionJourney.status.rejected",
      nextActionKey: "action.startPermissionApproval",
      tone: "danger"
    };
  }
  if (args.canApply) {
    return {
      detailKey: "permissionJourney.statusDetail.readyToApply",
      labelKey: "permissionJourney.status.readyToApply",
      nextActionKey: "action.applyPermissionPackage",
      tone: "info"
    };
  }
  if (args.draft.readiness.missingFields.length > 0 || args.workbenchStatus === "needs_input") {
    return {
      detailKey: "permissionJourney.statusDetail.needsInput",
      labelKey: "permissionJourney.status.needsInput",
      nextActionKey: "action.createApprovalRequest",
      tone: "warning"
    };
  }
  if (args.workbenchStatus === "blocked" || args.productionStatus === "blocked") {
    return {
      detailKey: "permissionJourney.statusDetail.blocked",
      labelKey: "permissionJourney.status.blocked",
      nextActionKey: "action.checkPreflight",
      tone: "danger"
    };
  }
  return {
    detailKey: "permissionJourney.statusDetail.needsApproval",
    labelKey: "permissionJourney.status.needsApproval",
    nextActionKey: "action.createApprovalRequest",
    tone: "warning"
  };
}

function permissionDraftStatus(draft: PermissionPackageDraft): { labelKey: string; tone: Tone } {
  if (!draft.readiness.canApply) {
    return { labelKey: "status.needsReview", tone: "warning" };
  }
  if (!draft.policyGate.canApplyDirectly) {
    return { labelKey: "status.approvalPending", tone: "warning" };
  }
  return { labelKey: "status.readyToApply", tone: "success" };
}

function permissionPolicyGateDetailKey(
  canApplyDirectly: boolean,
  approvalStatus: PermissionPackageApprovalRequest["status"] | null,
) {
  if (canApplyDirectly) return "text.policyGateDirectDetail";
  if (approvalStatus === "approved") return "text.policyGateApprovedDetail";
  if (approvalStatus === "rejected") return "text.policyGateRejectedDetail";
  if (approvalStatus === "withdrawn") return "text.policyGateWithdrawnDetail";
  return "text.policyGateApprovalDetail";
}

function permissionApprovalStatusLabel(status: PermissionPackageApprovalRequest["status"], t: Translator) {
  if (status === "approved") return t("status.approvalApproved");
  if (status === "rejected") return t("status.approvalRejected");
  if (status === "withdrawn") return t("status.approvalWithdrawn");
  return t("status.approvalPending");
}

function permissionApprovalStatusTone(status: PermissionPackageApprovalRequest["status"]): Tone {
  if (status === "approved") return "success";
  if (status === "rejected") return "danger";
  if (status === "withdrawn") return "warning";
  return "warning";
}

function permissionPolicyReasonMessage(
  reason: PermissionPackageDraft["policyGate"]["reasons"][number],
  t: Translator,
) {
  if (!reason.reasonKey) return reason.message;
  const values = Object.entries(reason.reasonValues ?? {}).reduce<Record<string, string>>((acc, [key, value]) => {
    if (key === "capability") {
      acc[key] = capabilityKeyDisplayName(value, t);
    } else if (key === "action" || key === "risk" || key === "sensitivity") {
      acc[key] = translatedValue(t, value);
    } else {
      acc[key] = value;
    }
    return acc;
  }, {});
  return tx(t, reason.reasonKey, values);
}

function translatedValue(t: Translator, value: string) {
  return t(`value.${value}`, value);
}

function permissionInlineMessageTone(message: string): "error" | "info" | "success" | "warning" {
  const normalized = message.trim().toLowerCase();
  if (!normalized) return "info";
  if (
    normalized.includes("unable") ||
    normalized.includes("failed") ||
    normalized.includes("error") ||
    normalized.includes("requires") ||
    normalized.includes("missing") ||
    normalized.includes("blocked") ||
    normalized.includes("rejected") ||
    normalized.includes("失败") ||
    normalized.includes("错误") ||
    normalized.includes("需要") ||
    normalized.includes("缺少") ||
    normalized.includes("阻断") ||
    normalized.includes("拒绝")
  ) {
    return "error";
  }
  if (
    normalized.includes("ready") ||
    normalized.includes("loaded") ||
    normalized.includes("exported") ||
    normalized.includes("applied") ||
    normalized.includes("approved") ||
    normalized.includes("已") ||
    normalized.includes("可上线") ||
    normalized.includes("完成")
  ) {
    return "success";
  }
  if (normalized.includes("pending") || normalized.includes("waiting") || normalized.includes("待")) {
    return "warning";
  }
  return "info";
}

function shouldShowAdvancedStatusMessage(tone: ReturnType<typeof permissionInlineMessageTone>) {
  return tone === "error" || tone === "warning";
}

function permissionApplicationHealthLabel(status: PermissionPackageApplicationHealthStatus, t: Translator) {
  if (status === "ready") return t("status.applicationHealthReady");
  if (status === "drifted") return t("status.applicationHealthDrifted");
  return t("status.applicationHealthNeedsReview");
}

function permissionApplicationHealthRowSummary(row: PermissionPackageApplicationHealthRow, t: Translator) {
  if (row.status === "ready") return t("text.applicationHealthReadyDetail");
  if (row.status === "drifted") return t("text.applicationHealthDriftedDetail");
  return t("text.applicationHealthNeedsReviewDetail");
}

function productionReadinessStatusLabel(status: PermissionPackageProductionReadinessStatus | undefined, t: Translator) {
  if (status === "ready") return t("status.productionReady");
  if (status === "needs_review") return t("status.productionNeedsReview");
  if (status === "blocked") return t("status.productionBlocked");
  return t("status.preflightPending");
}


function permissionProductionReadinessCheckLabel(code: string, t: Translator) {
  return t(`productionCheck.${code}`, code.replaceAll("_", " "));
}

function permissionProductionReadinessCheckMessage(check: PermissionPackageProductionReadinessCheck, t: Translator) {
  return t(`productionCheck.detail.${check.code}`, check.message);
}

function permissionPackageApprovalRouteLabel(request: PermissionPackageApprovalRequest) {
  return `${request.tenantId} / ${request.workspaceId} / ${request.callerInstanceId}`;
}

function productionConsoleStatusTone(status: AiAdminProductionConsoleStatus): Tone {
  if (status === "ready") return "success";
  if (status === "needs_review") return "warning";
  if (status === "blocked") return "danger";
  return "neutral";
}

function permissionWorkbenchActionKey(code: string | undefined, fallback: string) {
  switch (code) {
    case "complete_request":
      return "action.completePermissionRequest";
    case "create_approval_request":
      return "action.createApprovalRequest";
    case "review_approval_request":
      return "action.reviewApprovalRequest";
    case "apply_permission_package":
      return "action.applyPermissionPackage";
    case "run_runtime_validation":
      return "action.runApprovalJourney";
    case "export_production_evidence":
      return "action.exportProductionEvidence";
    default:
      return fallback;
  }
}

function permissionWorkbenchStatusLabelKey(status: string | undefined) {
  if (!status) return "permissionWorkbench.status.pending";
  return `permissionWorkbench.status.${status}`;
}

function permissionWorkbenchStatusDetailKey(status: string | undefined) {
  if (!status) return "permissionWorkbench.statusDetail.pending";
  return `permissionWorkbench.statusDetail.${status}`;
}

function permissionWorkbenchStatusTone(status: string | undefined, fallback: AiAdminProductionConsoleStatus): Tone {
  if (status === "production_ready") return "success";
  if (status === "blocked") return "danger";
  if (status === "ready_to_apply" || status === "validating" || status === "awaiting_approval") return "warning";
  if (status === "needs_input") return "neutral";
  return productionConsoleStatusTone(fallback);
}

function permissionWorkbenchStepLabelKey(key: string) {
  return `permissionWorkbench.step.${key}`;
}

function permissionWorkbenchStepDetailKey(detailCode: string) {
  return `permissionWorkbench.detail.${detailCode}`;
}

function permissionWorkbenchStepDisplayDetailCode(
  step: {
    detailCode: string;
    key: PermissionPackageWorkbenchStepKey;
  },
  args: {
    approvalRequired: boolean;
    approvalStatus: PermissionPackageApprovalRequest["status"] | null;
    applicationReady: boolean;
    goLiveReady: boolean;
    runtimeValidationReady: boolean;
  }
) {
  if (args.goLiveReady) {
    if (step.key === "approval") return args.approvalRequired ? "approval_approved" : "approval_not_required";
    if (step.key === "apply") return "apply_done";
    if (step.key === "validation") return "validation_ready";
    if (step.key === "acceptance") return "acceptance_ready";
  }
  if (step.key === "approval" && args.approvalStatus === "approved") {
    return args.approvalRequired ? "approval_approved" : "approval_not_required";
  }
  if (step.key === "apply" && args.applicationReady) return "apply_done";
  if (step.key === "validation" && args.runtimeValidationReady) return "validation_ready";
  return step.detailCode;
}

function permissionWorkbenchStepDisplayStatus(
  step: {
    key: PermissionPackageWorkbenchStepKey;
    status: PermissionPackageWorkbenchStepStatus;
  },
  args: {
    approvalComplete: boolean;
    applicationReady: boolean;
    goLiveReady: boolean;
    runtimeValidationReady: boolean;
  }
): PermissionPackageWorkbenchStepStatus {
  if (args.goLiveReady) return "complete";
  if (step.key === "approval" && args.approvalComplete) return "complete";
  if (step.key === "apply" && args.applicationReady) return "complete";
  if (step.key === "validation" && args.runtimeValidationReady) return "complete";
  return step.status;
}
