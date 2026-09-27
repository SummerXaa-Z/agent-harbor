import { useEffect, useState } from "react";
import { fetchAgentKeys, fetchPermissionPackageApprovalRequests } from "../../api";
import type { PermissionPackageApprovalRequest } from "../../permissionPackages";
import type { AgentKey } from "../../types";

export interface UserRecords {
  approvals: PermissionPackageApprovalRequest[];
  failed: boolean;
  keys: AgentKey[];
  loaded: boolean;
}

const emptyRecords: UserRecords = { approvals: [], failed: false, keys: [], loaded: false };

// Approval requests (newest first, filtered to the session actor by the
// caller) and API key metadata. Keys come back as prefixes only.
export function useUserRecords(live: boolean): UserRecords {
  const [records, setRecords] = useState<UserRecords>(emptyRecords);

  useEffect(() => {
    if (!live) return;
    const controller = new AbortController();
    Promise.allSettled([
      fetchPermissionPackageApprovalRequests({ limit: 100 }, "", controller.signal),
      fetchAgentKeys(undefined, "", controller.signal),
    ]).then(([approvals, keys]) => {
      if (controller.signal.aborted) return;
      setRecords({
        approvals: approvals.status === "fulfilled" ? approvals.value : [],
        failed: approvals.status === "rejected" || keys.status === "rejected",
        keys: keys.status === "fulfilled" ? keys.value : [],
        loaded: true,
      });
    });
    return () => controller.abort();
  }, [live]);

  return records;
}
