import { useCallback, useEffect, useRef, useState } from "react";
import {
  approvePermissionPackageApprovalRequest,
  fetchAdminIdentities,
  fetchPermissionPackageApprovalRequests,
  rejectPermissionPackageApprovalRequest,
} from "../../api";
import type { PermissionPackageApprovalRequest } from "../../permissionPackages";
import type { AdminIdentity, ConsoleSession } from "../../types";
import {
  DEFAULT_DEMO_REVIEWER,
  DEMO_REVIEWER_STORAGE_KEY,
  resolveReviewer,
  reviewerOptions,
} from "../model/demoRole";

export type ApprovalDecision = "approve" | "reject";

export interface ApprovalActResult {
  error: unknown;
  ok: boolean;
  request: PermissionPackageApprovalRequest | null;
}

export interface ApprovalsState {
  act: (id: string, decision: ApprovalDecision, comment: string) => Promise<ApprovalActResult>;
  failed: boolean;
  loading: boolean;
  reload: () => Promise<void>;
  requests: readonly PermissionPackageApprovalRequest[];
  reviewer: string;
  reviewerOptions: string[];
  reviewerSelectable: boolean;
  setReviewer: (value: string) => void;
}

// The admin approval queue. Approve / reject go through the real endpoints
// with the resolved reviewer (session actor when signed in, the demo picker
// otherwise); every outcome also reloads the list, so a 409 "already
// resolved" from another reviewer surfaces as fresh state, not a dead end.
export function useApprovals(live: boolean, session: ConsoleSession | null): ApprovalsState {
  const [requests, setRequests] = useState<readonly PermissionPackageApprovalRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const [admins, setAdmins] = useState<readonly AdminIdentity[]>([]);
  const [reviewerInput, setReviewerInput] = useState("");
  const loadedRef = useRef(false);

  const resolution = resolveReviewer(session, reviewerInput || readStoredReviewer());

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const rows = await fetchPermissionPackageApprovalRequests({ limit: 100 });
      setRequests(rows);
      setFailed(false);
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!live) return;
    const controller = new AbortController();
    void reload();
    fetchAdminIdentities(undefined, controller.signal)
      .then((rows) => {
        if (!controller.signal.aborted) setAdmins(rows);
      })
      .catch(() => undefined);
    loadedRef.current = true;
    return () => controller.abort();
  }, [live, reload]);

  const setReviewer = useCallback((value: string) => {
    setReviewerInput(value);
    try {
      window.localStorage.setItem(DEMO_REVIEWER_STORAGE_KEY, value.trim() || DEFAULT_DEMO_REVIEWER);
    } catch {
      // The picker still works for this session without storage.
    }
  }, []);

  const act = useCallback(
    async (id: string, decision: ApprovalDecision, comment: string): Promise<ApprovalActResult> => {
      const body = resolution.selectable && resolution.reviewer ? { comment, reviewer: resolution.reviewer } : { comment };
      try {
        const request = decision === "approve"
          ? await approvePermissionPackageApprovalRequest(id, body)
          : await rejectPermissionPackageApprovalRequest(id, body);
        await reload();
        return { error: null, ok: true, request };
      } catch (error) {
        // Conflict or network failure: refresh so the queue reflects whatever
        // happened on the server instead of stranding the stale row.
        await reload().catch(() => undefined);
        return { error, ok: false, request: null };
      }
    },
    [reload, resolution.reviewer, resolution.selectable],
  );

  return {
    act,
    failed,
    loading,
    reload,
    requests,
    reviewer: resolution.reviewer,
    reviewerOptions: resolution.selectable ? reviewerOptions(admins) : [],
    reviewerSelectable: resolution.selectable,
    setReviewer,
  };
}

function readStoredReviewer(): string | null {
  try {
    return window.localStorage.getItem(DEMO_REVIEWER_STORAGE_KEY);
  } catch {
    return null;
  }
}
