import type { AdminIdentity, AdminIdentityRole, AdminIdentitySource, AdminIdentityStatus } from "./types";

export interface AdminAccessSummary {
  active: number;
  bootstrap: number;
  disabled: number;
  scoped: number;
}

export function summarizeAdminIdentities(rows: AdminIdentity[]): AdminAccessSummary {
  return rows.reduce(
    (summary, row) => {
      if (row.status === "active") summary.active += 1;
      if (row.status === "disabled") summary.disabled += 1;
      if (row.source === "bootstrap") summary.bootstrap += 1;
      if (row.tenantId || row.workspaceId) summary.scoped += 1;
      return summary;
    },
    { active: 0, bootstrap: 0, disabled: 0, scoped: 0 }
  );
}

export function adminIdentityScopeText(row: Pick<AdminIdentity, "tenantId" | "workspaceId">) {
  if (row.tenantId && row.workspaceId) return `${row.tenantId} / ${row.workspaceId}`;
  if (row.tenantId) return row.tenantId;
  if (row.workspaceId) return row.workspaceId;
  return "all";
}

export function adminIdentityRoleKey(role: AdminIdentityRole) {
  return `adminAccess.role.${role}`;
}

export function adminIdentitySourceKey(source: AdminIdentitySource) {
  return `adminAccess.source.${source}`;
}

export function adminIdentityStatusKey(status: AdminIdentityStatus) {
  return `adminAccess.status.${status}`;
}

export function adminIdentityStatusTone(status: AdminIdentityStatus) {
  return status === "active" ? "success" : "neutral";
}

export function adminIdentitySourceTone(source: AdminIdentitySource) {
  return source === "managed" ? "info" : "neutral";
}
