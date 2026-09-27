import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  applyPermissionPackage,
  createPermissionPackageApprovalRequest,
  fetchPermissionPackageAcceptanceReport,
  fetchPermissionPackageApprovalRequests,
  preflightPermissionPackage,
  previewPermissionPackageWorkbench,
  withdrawPermissionPackageApprovalRequest,
} from "../../api";
import { tx } from "../../localizedMessages";
import {
  permissionPackageApprovalEffectiveStatus,
  type PermissionPackageApprovalRequest,
  type PermissionPackageWorkbenchPreview,
} from "../../permissionPackages";
import { firstBlockingApplyPreflightCheck, permissionApplyPreflightCheckMessage } from "../../permissionWorkbenchPresenters";
import { productionAcceptanceReportFilename } from "../../productionAcceptance";
import { accessContextComplete, accessContextDraftInput, readinessFilterFromContext, type AccessContext } from "../model/accessContext";
import { apiErrorPresentation } from "../model/apiErrorCategory";
import {
  approvalReconcileFilter,
  effectiveApproval,
  isApprovalAlreadyPendingError,
  isApprovalNotRequiredError,
  isCapabilityChangedError,
  permissionChangePresentation,
  permissionChangeState,
  pickPendingApproval,
  type PermissionChangePresentation,
} from "../model/approvalStateMachine";
import { useToast } from "../ui/Toast";
import { useRedesignI18n } from "./useRedesignI18n";

export type PermissionChangeBusy = "" | "submit" | "withdraw" | "apply" | "report";

export interface PermissionChangeFlow {
  apply: () => Promise<void>;
  approval: PermissionPackageApprovalRequest | null;
  // Binds a request opened by ID (deep link, notification); the preview omits
  // expired ones, so this is the only way they show up.
  bindApproval: (approval: PermissionPackageApprovalRequest) => void;
  busy: PermissionChangeBusy;
  exportReport: () => Promise<void>;
  presentation: PermissionChangePresentation;
  preview: PermissionPackageWorkbenchPreview | null;
  previewError: unknown;
  previewLoading: boolean;
  refresh: () => void;
  reedit: () => void;
  submit: () => Promise<void>;
  withdraw: () => Promise<void>;
}

const previewPollMs = 10_000;
const previewDebounceMs = 300;
const reeditableStatuses = new Set(["rejected", "withdrawn", "expired"]);

function approvalScopeKey(scope: Pick<AccessContext, "tenantId" | "workspaceId" | "callerInstanceId" | "targetId" | "templateId">) {
  return [scope.tenantId, scope.workspaceId, scope.callerInstanceId, scope.targetId, scope.templateId].join("|");
}

function downloadJson(value: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(value, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

// Drives Request access from the server's workbench preview. Submit, withdraw and
// apply follow the legacy console: request text is required before
// submitting, a duplicate create binds the existing pending request, a policy
// change that removes the approval gate refreshes into "no approval needed",
// and apply runs preflight first.
export function usePermissionChangeFlow({
  context,
  live,
  onApplied,
}: {
  context: AccessContext;
  live: boolean;
  onApplied?: () => void;
}): PermissionChangeFlow {
  const { language, t } = useRedesignI18n();
  const showToast = useToast();
  const [preview, setPreview] = useState<PermissionPackageWorkbenchPreview | null>(null);
  const [previewError, setPreviewError] = useState<unknown>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [bound, setBound] = useState<{ approval: PermissionPackageApprovalRequest; scopeKey: string } | null>(null);
  const [capabilityChanged, setCapabilityChanged] = useState(false);
  const [reediting, setReediting] = useState(false);
  const [busy, setBusy] = useState<PermissionChangeBusy>("");
  const [refreshKey, setRefreshKey] = useState(0);
  const busyRef = useRef(false);

  const draftInput = useMemo(() => accessContextDraftInput(context), [context]);
  const inputKey = JSON.stringify(draftInput);
  const ready = live && accessContextComplete(context);
  const scopeKey = approvalScopeKey(context);
  const localApproval = bound?.scopeKey === scopeKey ? bound.approval : null;
  const setLocalApproval = useCallback((approval: PermissionPackageApprovalRequest) => {
    setBound({ approval, scopeKey: approvalScopeKey(approval) });
  }, []);

  useEffect(() => {
    setCapabilityChanged(false);
    setReediting(false);
  }, [scopeKey]);

  const silentRef = useRef(false);
  useEffect(() => {
    if (!ready) {
      setPreview(null);
      setPreviewError(null);
      return;
    }
    const controller = new AbortController();
    const silent = silentRef.current;
    silentRef.current = false;
    const timer = window.setTimeout(() => {
      if (!silent) setPreviewLoading(true);
      previewPermissionPackageWorkbench(JSON.parse(inputKey), "", controller.signal)
        .then((next) => {
          setPreview(next);
          setPreviewError(null);
        })
        .catch((error: unknown) => {
          if (controller.signal.aborted) return;
          setPreviewError(error);
          if (!silent) setPreview(null);
        })
        .finally(() => {
          if (!controller.signal.aborted) setPreviewLoading(false);
        });
    }, silent ? 0 : previewDebounceMs);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [ready, inputKey, refreshKey]);

  const refresh = useCallback(() => setRefreshKey((key) => key + 1), []);
  const silentRefresh = useCallback(() => {
    silentRef.current = true;
    setRefreshKey((key) => key + 1);
  }, []);

  const approval = effectiveApproval(preview?.approvalRequest, localApproval);
  const hideSettledApproval = reediting && approval !== null && reeditableStatuses.has(permissionPackageApprovalEffectiveStatus(approval));
  const stateInput = hideSettledApproval && preview ? { ...preview, approvalRequest: undefined } : preview;
  const localMissing = context.requestText.trim() ? [] : ["requestText"];
  const state = permissionChangeState({
    approval: hideSettledApproval ? null : localApproval,
    capabilityChanged,
    localMissing,
    preview: stateInput,
  });
  const presentation = permissionChangePresentation(state, { templateSelected: Boolean(context.templateId) });

  useEffect(() => {
    if (!presentation.polling || !ready) return;
    const timer = window.setInterval(silentRefresh, previewPollMs);
    return () => window.clearInterval(timer);
  }, [presentation.polling, ready, silentRefresh]);

  function errorDetail(error: unknown, fallbackKey: string) {
    return apiErrorPresentation(t, language, error, fallbackKey).detail;
  }

  function begin(action: PermissionChangeBusy) {
    if (busyRef.current) return false;
    busyRef.current = true;
    setBusy(action);
    return true;
  }

  function end() {
    busyRef.current = false;
    setBusy("");
  }

  async function reconcilePending() {
    try {
      const rows = await fetchPermissionPackageApprovalRequests(approvalReconcileFilter(draftInput), "");
      const existing = pickPendingApproval(rows, draftInput);
      if (!existing) {
        showToast(t("message.permissionApprovalAlreadyPending"), "warning");
        return;
      }
      setLocalApproval(existing);
      setReediting(false);
      showToast(tx(t, "message.permissionApprovalAlreadyPendingReconciled", { id: existing.id }), "warning");
    } catch {
      showToast(t("message.permissionApprovalAlreadyPending"), "warning");
    }
  }

  async function submit() {
    if (!live) {
      showToast(t("message.permissionApprovalRequiresLiveApi"), "warning");
      return;
    }
    if (!draftInput.requestText.trim()) {
      showToast(t("message.permissionApprovalRequestTextRequired"), "warning");
      return;
    }
    if (!hideSettledApproval && approval && permissionPackageApprovalEffectiveStatus(approval) === "pending") {
      showToast(t("message.permissionApprovalAlreadyPending"), "warning");
      return;
    }
    if (!begin("submit")) return;
    try {
      const request = await createPermissionPackageApprovalRequest(draftInput, "");
      setLocalApproval(request);
      setReediting(false);
      showToast(tx(t, "message.permissionApprovalCreated", { id: request.id }));
    } catch (error) {
      if (isApprovalAlreadyPendingError(error)) {
        await reconcilePending();
      } else if (isApprovalNotRequiredError(error)) {
        showToast(t("rd.apply.toast.noApprovalNeeded"), "warning");
      } else {
        showToast(errorDetail(error, "error.createApprovalRequest"), "danger");
      }
    } finally {
      end();
      silentRefresh();
    }
  }

  async function withdraw() {
    if (!approval || permissionPackageApprovalEffectiveStatus(approval) !== "pending") {
      showToast(t("message.permissionApprovalWithdrawUnavailable"), "warning");
      return;
    }
    if (!begin("withdraw")) return;
    try {
      const request = await withdrawPermissionPackageApprovalRequest(approval.id, {}, "");
      setLocalApproval(request);
      showToast(t("message.permissionApprovalWithdrawn"));
    } catch (error) {
      showToast(errorDetail(error, "error.withdrawRequest"), "danger");
    } finally {
      end();
      silentRefresh();
    }
  }

  async function apply() {
    if (!live) {
      showToast(t("message.fallbackDataModeActionBlocked"), "warning");
      return;
    }
    if (!begin("apply")) return;
    const approvedId = approval && permissionPackageApprovalEffectiveStatus(approval) === "approved" ? approval.id : "";
    const applyInput = approvedId ? { ...draftInput, approvalRequestId: approvedId } : draftInput;
    try {
      const preflight = await preflightPermissionPackage(applyInput, "");
      if (!preflight.summary.canApply) {
        showToast(tx(t, "message.permissionPackagePreflightApplyBlocked", {
          detail: permissionApplyPreflightCheckMessage(firstBlockingApplyPreflightCheck(preflight), t),
        }), "danger");
        return;
      }
      await applyPermissionPackage(applyInput, "");
      showToast(t("rd.apply.toast.applied"));
      onApplied?.();
    } catch (error) {
      if (isCapabilityChangedError(error)) setCapabilityChanged(true);
      showToast(errorDetail(error, "error.applyPermissionPackage"), "danger");
    } finally {
      end();
      silentRefresh();
    }
  }

  async function exportReport() {
    if (!live) {
      showToast(t("message.acceptanceReportRequiresLiveApi"), "warning");
      return;
    }
    if (!begin("report")) return;
    try {
      const report = await fetchPermissionPackageAcceptanceReport(readinessFilterFromContext(context, approval?.id), "");
      downloadJson(report, productionAcceptanceReportFilename(report));
      showToast(tx(t, "message.acceptanceReportExportedBy", { actor: report.generatedBy }));
    } catch (error) {
      showToast(errorDetail(error, "error.exportAcceptanceReport"), "danger");
    } finally {
      end();
    }
  }

  const reedit = useCallback(() => {
    setCapabilityChanged(false);
    setReediting(true);
  }, []);

  return {
    apply,
    approval: hideSettledApproval ? null : approval,
    bindApproval: setLocalApproval,
    busy,
    exportReport,
    presentation,
    preview,
    previewError,
    previewLoading,
    refresh,
    reedit,
    submit,
    withdraw,
  };
}
