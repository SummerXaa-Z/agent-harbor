import type {
  ConsoleData,
  InstanceAssignment,
  TenantEntitlement,
  WorkspaceAssignment
} from "./types";

export type CapabilityGrantRefreshResult =
  | { ok: true }
  | { error: unknown; ok: false };

export interface CapabilityGrantChainRecords {
  entitlement: TenantEntitlement;
  instanceAssignment: InstanceAssignment;
  workspaceAssignment: WorkspaceAssignment;
}

export async function refreshAfterCapabilityGrantMutation({
  onRefresh
}: {
  onRefresh: () => Promise<void>;
}): Promise<CapabilityGrantRefreshResult> {
  try {
    await onRefresh();
    return { ok: true };
  } catch (error) {
    return { error, ok: false };
  }
}

export function capabilityGrantRefreshFailedMessageKey() {
  return "message.grantChainCreatedRefreshFailed";
}

export function mergeCapabilityGrantChainIntoConsoleData(
  current: ConsoleData,
  records: CapabilityGrantChainRecords
): ConsoleData {
  return {
    ...current,
    capabilityAssignmentsLoadedFromApi: true,
    grantsLoadedFromApi: true,
    instanceAssignments: upsertById(current.instanceAssignments, records.instanceAssignment),
    tenantEntitlements: upsertById(current.tenantEntitlements, records.entitlement),
    workspaceAssignments: upsertById(current.workspaceAssignments, records.workspaceAssignment)
  };
}

function upsertById<T extends { id: string }>(rows: T[], next: T) {
  const found = rows.some((row) => row.id === next.id);
  if (!found) return [next, ...rows];
  return rows.map((row) => (row.id === next.id ? next : row));
}
