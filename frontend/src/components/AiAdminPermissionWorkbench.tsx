import { useState } from "react";
import {
  CheckCircle2,
  ClipboardCheck,
  Download,
  FileSearch,
  RefreshCw,
  TriangleAlert,
  Undo2,
  Workflow,
  X
} from "lucide-react";

import {
  accessSubjectOptionForSelectorFrom,
  normalizeAccessSubjectOptions,
  type AccessSubjectOption
} from "../accessSubjects";
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
import type { AiAdminProductionConsoleSummary } from "../aiAdminProductionConsole";
import {
  currentPermissionRequestWizardStep,
  permissionRequestProcessStepStatuses,
  type PermissionRequestWizardStep
} from "../permissionRequestJourney";
import { usePermissionApprovalDecision } from "../hooks/usePermissionApprovalDecision";
import {
  accessDecisionOutcomeLabel,
  accessSubjectDropdownOption,
  formatDate,
  permissionApplicationHealthLabel,
  permissionApplicationHealthRowSummary,
  permissionApprovalRequestBusinessLabel,
  permissionApprovalStatusLabel,
  permissionApprovalStatusTone,
  permissionEntityDisplayName,
  permissionApplyPreflightNextAction,
  permissionInlineMessageTone,
  permissionPackageTemplateName,
  permissionPackageTemplateSummary,
  permissionPolicyGateDetailKey,
  permissionPolicyReasonMessage,
  permissionProductionReadinessCheckLabel,
  permissionProductionReadinessCheckMessage,
  permissionRequestStepSectionId,
  permissionRequestStepTarget,
  permissionTenantPathLabel,
  permissionWorkbenchStepDetailKey,
  permissionWorkbenchStepDisplayDetailCode,
  permissionWorkbenchStepDisplayStatus,
  permissionWorkbenchStepLabelKey,
  permissionWorkspaceDisplayName,
  productionConsoleStatusTone,
  productionReadinessStatusLabel,
  resolvePermissionJourneyStatus,
  shouldShowAdvancedStatusMessage,
  tx,
  uniquePermissionEntityOptions,
  type PermissionRequestStepTarget,
  type Tone,
  type Translator
} from "../permissionWorkbenchPresenters";
import {
  permissionPackageApprovalEffectiveStatus,
  type PermissionPackageApplyPreflight,
  type PermissionPackageApplication,
  type PermissionPackageApplicationHealth,
  type PermissionPackageApplicationImpact,
  type PermissionPackageApprovalRequest,
  type PermissionPackageDraft,
  type PermissionPackageDraftInput,
  type PermissionPackageProductionReadiness,
  type PermissionPackageTemplate,
  type PermissionPackageWorkbenchPreview
} from "../permissionPackages";
import type {
  AccessDecisionExplainResult,
  Agent,
  AuditEvent,
  PermissionChangeHandoffContext,
  Tenant
} from "../types";
import { PermissionChangeDraftSheet } from "./PermissionWorkbenchParts";
import { TechnicalId } from "./TechnicalId";
import { Badge, EmptyRow } from "./ui";

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
  onDismissPermissionHandoff: () => void;
  onWithdrawApprovalRequest: (comment?: string) => void;
  permissionHandoffContext: PermissionChangeHandoffContext | null;
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
    onDismissPermissionHandoff,
    onWithdrawApprovalRequest,
    permissionHandoffContext,
    reviewerQueueLoading,
    reviewerQueueMessage,
    selectedApprovalRequestId,
    templates,
    tenants,
    t
  } = props;
  const {
    beginApprovalDecision, cancelApprovalDecision, confirmPendingApprovalDecision,
    pendingApprovalDecision, updatePendingApprovalComment
  } = usePermissionApprovalDecision({
    onApproveApprovalRequest,
    onRejectApprovalRequest,
    onSelectApprovalRequest, onWithdrawApprovalRequest, t
  });
  const [permissionDraftSheet, setPermissionDraftSheet] = useState<"closed" | "edit">("closed");
  const callers = agents.filter((agent) => agent.status === "active" && agent.channelType === "local");
  const selectedCaller = agents.find((agent) => agent.id === form.callerInstanceId);
  const selectedTarget = mcpTargets.find((agent) => agent.id === form.targetId);
  const permissionHandoffTitle = permissionHandoffContext?.sourceView === "tenants" ? t("text.permissionTenantHandoffTitle") : permissionHandoffContext?.sourceView === "registry" ? t("text.permissionHandoffRegistryTitle") : t("text.permissionHandoffTitle");
  const permissionHandoffDetail = permissionHandoffContext
    ? permissionHandoffContext.sourceView === "tenants"
      ? tx(t, "text.permissionTenantHandoffDetail", {
        tenant: permissionHandoffContext.tenantName ?? permissionHandoffContext.tenantId,
        workspace: permissionHandoffContext.workspaceName ?? permissionHandoffContext.workspaceId
      })
      : tx(t, "text.permissionHandoffDetail", {
        caller: permissionHandoffContext.callerName ?? permissionHandoffContext.callerInstanceId ?? "-",
        target: permissionHandoffContext.targetName ?? permissionHandoffContext.targetId ?? "-"
      })
    : "";
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
  const approvalRequestEffectiveStatus = approvalRequest ? permissionPackageApprovalEffectiveStatus(approvalRequest) : null;
  const hasApprovedRequest = approvalRequestEffectiveStatus === "approved";
  const canApply = draft.readiness.canApply && (draft.policyGate.canApplyDirectly || hasApprovedRequest);
  const reviewerQueueRequests = approvalRequests.filter((request) => permissionPackageApprovalEffectiveStatus(request) === "pending");
  const goLiveReadiness = summarizeAiAdminGoLiveReadiness(approvalJourneyEvaluation);
  const productionReady = productionReadiness?.status === "ready"
    || Boolean(workbenchPreview?.summary.productionReady)
    || productionSummary.status === "ready";
  const goLiveReady = productionReady || goLiveReadiness.status === "ready";
  const requestFormLocked = Boolean(application)
    || goLiveReady
    || approvalRequestEffectiveStatus === "pending"
    || approvalRequestEffectiveStatus === "approved";
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
  const goLivePrerequisitesReady = Boolean(application) || goLiveReady;
  const approvalEffectivelyResolved = !draft.policyGate.canApplyDirectly
    && (approvalRequestEffectiveStatus === "approved" || Boolean(application) || goLiveReady);
  const approvalDisplayStatus = approvalEffectivelyResolved
    ? "approved"
    : approvalRequestEffectiveStatus;
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
    && (!approvalRequest || (approvalRequestEffectiveStatus !== "pending" && approvalRequestEffectiveStatus !== "approved"));
  const showPendingApprovalActions = !application && !goLiveReady && approvalRequestEffectiveStatus === "pending";
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
  const messageTone = permissionInlineMessageTone(message);
  const approvalReadinessMessageTone = permissionInlineMessageTone(approvalReadinessMessage);
  const applyPreflightMessageTone = permissionInlineMessageTone(applyPreflightMessage);
  const applyPreflightNextAction = applyPreflight?.nextActionCodes?.[0]
    ? permissionApplyPreflightNextAction(applyPreflight.nextActions[0] ?? "", t, applyPreflight.nextActionCodes[0])
    : applyPreflight?.nextActions[0]
    ? permissionApplyPreflightNextAction(applyPreflight.nextActions[0], t)
    : "";
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
  function startNewPermissionChangeInSheet() {
    onStartNewPermissionChange();
    setPermissionDraftSheet("edit");
  }
  function scrollToPermissionRequestStep(step: PermissionRequestStepTarget) {
    if (step === "scope" || step === "template") {
      setPermissionDraftSheet("edit");
      return;
    }
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
    if (journeyStatus.nextActionKey === "action.completePermissionRequest") {
      setPermissionDraftSheet("edit");
      return;
    }
    if (journeyStatus.nextActionKey === "action.createApprovalRequest") {
      onCreateApprovalRequest();
      return;
    }
    if (journeyStatus.nextActionKey === "action.startPermissionApproval") {
      startNewPermissionChangeInSheet();
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
      complete: draft.policyGate.canApplyDirectly || approvalRequestEffectiveStatus === "approved",
      detail: draft.policyGate.canApplyDirectly
        ? t("status.directApplyAllowed")
        : approvalRequestEffectiveStatus ? permissionApprovalStatusLabel(approvalRequestEffectiveStatus, t) : t("status.approvalNotRequested"),
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
        : approvalRequestEffectiveStatus ? permissionApprovalStatusLabel(approvalRequestEffectiveStatus, t) : t("status.approvalNotRequested");
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
  const goLivePrimaryActionKey = !goLivePrerequisitesReady
    ? journeyStatus.nextActionKey
    : goLiveReady
    ? "action.exportProductionEvidence"
    : runtimeValidationReady
      ? "action.checkProductionReadiness"
      : "action.runApprovalJourney";
  const goLiveNextActionText = !goLivePrerequisitesReady
    ? t(journeyStatus.nextActionKey)
    : t(goLiveNextKey);
  const runtimeValidationText = runtimeValidationReady
    ? t("text.runtimeValidationResultReady")
    : goLivePrerequisitesReady
      ? t("text.runtimeValidationResultPending")
      : t("text.runtimeValidationBlockedDetail");
  const goLivePrimaryActionIcon = !goLivePrerequisitesReady
    ? <CheckCircle2 size={14} />
    : goLivePrimaryActionKey === "action.exportProductionEvidence"
    ? <Download size={14} />
    : goLivePrimaryActionKey === "action.checkProductionReadiness"
      ? <RefreshCw size={14} />
      : <Workflow size={14} />;
  const goLivePrimaryActionLabel = !goLivePrerequisitesReady
    ? t(journeyStatus.nextActionKey)
    : goLivePrimaryActionKey === "action.exportProductionEvidence" && productionEvidenceExporting
    ? t("action.exportingProductionEvidence")
    : goLivePrimaryActionKey === "action.checkProductionReadiness" && productionReadinessLoading
      ? t("action.checkingProductionReadiness")
      : goLivePrimaryActionKey === "action.runApprovalJourney" && approvalJourneyRunning
        ? t("action.runningApprovalJourney")
        : t(goLivePrimaryActionKey);
  const runGoLivePrimaryAction = () => {
    if (!goLivePrerequisitesReady) {
      runProductionPrimaryAction();
      return;
    }
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

      {permissionHandoffContext ? (
        <section className="permission-handoff-notice" role="status" aria-live="polite">
          <FileSearch size={16} />
          <div>
            <strong>{permissionHandoffTitle}</strong>
            <span>{permissionHandoffDetail}</span>
          </div>
          <button className="secondary-button" onClick={onDismissPermissionHandoff} type="button">
            <X aria-hidden="true" size={14} />
            {t("action.dismiss")}
          </button>
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
          <PermissionChangeDraftSheet
            accessSubjectCatalog={accessSubjectCatalog}
            accessSubjectDropdownOptions={accessSubjectDropdownOptions}
            callerDropdownOptions={callerDropdownOptions}
            draft={draft}
            form={form}
            isActiveLocked={requestFormActiveLocked}
            isLocked={requestFormLocked}
            isOpen={permissionDraftSheet === "edit"}
            lockedDetailKey={requestFormLockedDetailKey}
            onChange={onChange}
            onClose={() => setPermissionDraftSheet("closed")}
            onOpenDraftSheet={() => setPermissionDraftSheet("edit")}
            onStartNewPermissionChange={startNewPermissionChangeInSheet}
            requestFormHelpKey={requestFormHelpKey}
            requestFormTitleKey={requestFormTitleKey}
            selectedAccessSubject={selectedAccessSubject}
            selectedCaller={selectedCaller}
            selectedTarget={selectedTarget}
            statusLabel={t(journeyStatus.labelKey)}
            statusTone={journeyStatus.tone}
            templateDropdownOptions={templateDropdownOptions}
            targetDropdownOptions={targetDropdownOptions}
            tenantDropdownOptions={tenantDropdownOptions}
            tenantPath={tenantPath}
            t={t}
            workspaceName={workspaceName}
          />
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
                  <button className="secondary-button" disabled={permissionRequestBusy} onClick={startNewPermissionChangeInSheet} type="button">
                    <ClipboardCheck size={14} />
                    {t("action.startPermissionApproval")}
                  </button>
                </div>
              </div>
            ) : (
              <>
                <div className="approval-next-line" id={permissionRequestStepSectionId("acceptance")}>
                  <span>{t("text.nextActions")}</span>
                  <strong>{goLiveNextActionText}</strong>
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
              <span>{runtimeValidationText}</span>
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
            {applyPreflightNextAction ? (
              <div className="approval-next-line compact">
                <span>{t("text.nextActions")}</span>
                <strong>{applyPreflightNextAction}</strong>
              </div>
            ) : null}
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
