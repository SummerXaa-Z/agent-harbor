import type { ConsoleData } from "./types";
import type { PermissionPackageApplyResult } from "./permissionPackages";

export type PermissionApplyRefreshResult<T> =
  | { ok: true; value: T }
  | { error: unknown; ok: false };

export async function refreshAfterPermissionApply<T>({
  onRefresh
}: {
  onRefresh: () => Promise<T>;
}): Promise<PermissionApplyRefreshResult<T>> {
  try {
    return { ok: true, value: await onRefresh() };
  } catch (error) {
    return { error, ok: false };
  }
}

export function permissionApplyRefreshFailedMessageKey() {
  return "message.permissionPackageAppliedRefreshFailed";
}

export function mergePermissionApplyResultIntoConsoleData(
  current: ConsoleData,
  result: Pick<PermissionPackageApplyResult, "draft" | "instanceAssignments" | "tenantEntitlements" | "workspaceAssignments">
): ConsoleData {
  return {
    ...current,
    capabilities: upsertManyById(current.capabilities, result.draft.allowedCapabilities),
    capabilitiesLoadedFromApi: current.capabilitiesLoadedFromApi || result.draft.allowedCapabilities.length > 0,
    capabilityAssignmentsLoadedFromApi: true,
    grantsLoadedFromApi: true,
    instanceAssignments: upsertManyById(current.instanceAssignments, result.instanceAssignments),
    tenantEntitlements: upsertManyById(current.tenantEntitlements, result.tenantEntitlements),
    workspaceAssignments: upsertManyById(current.workspaceAssignments, result.workspaceAssignments)
  };
}

function upsertManyById<T extends { id: string }>(rows: T[], nextRows: T[]) {
  return nextRows.reduce(upsertById, rows);
}

function upsertById<T extends { id: string }>(rows: T[], next: T) {
  const found = rows.some((row) => row.id === next.id);
  if (!found) return [next, ...rows];
  return rows.map((row) => (row.id === next.id ? next : row));
}
