import { useState } from "react";

import type { Translator } from "../permissionWorkbenchPresenters";

export type ApprovalDecisionAction = "approve" | "reject" | "withdraw";

export interface PendingApprovalDecision {
  action: ApprovalDecisionAction;
  requestId?: string;
  comment: string;
  error: string;
}

interface UsePermissionApprovalDecisionArgs {
  onApproveApprovalRequest: (requestId?: string, comment?: string) => void;
  onRejectApprovalRequest: (requestId?: string, comment?: string) => void;
  onSelectApprovalRequest: (requestId: string) => void;
  onWithdrawApprovalRequest: (comment?: string) => void;
  t: Translator;
}

export function usePermissionApprovalDecision({
  onApproveApprovalRequest,
  onRejectApprovalRequest,
  onSelectApprovalRequest,
  onWithdrawApprovalRequest,
  t
}: UsePermissionApprovalDecisionArgs) {
  const [pendingApprovalDecision, setPendingApprovalDecision] = useState<PendingApprovalDecision | null>(null);

  function beginApprovalDecision(action: ApprovalDecisionAction, requestId?: string) {
    if (requestId) onSelectApprovalRequest(requestId);
    setPendingApprovalDecision({
      action,
      requestId,
      comment: action === "approve" ? t("text.approvalApproveDefaultComment") : "",
      error: ""
    });
  }

  function cancelApprovalDecision() {
    setPendingApprovalDecision(null);
  }

  function updatePendingApprovalComment(comment: string) {
    setPendingApprovalDecision((current) => current ? { ...current, comment, error: "" } : current);
  }

  function confirmPendingApprovalDecision() {
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
  }

  return {
    beginApprovalDecision,
    cancelApprovalDecision,
    confirmPendingApprovalDecision,
    pendingApprovalDecision,
    updatePendingApprovalComment
  };
}
