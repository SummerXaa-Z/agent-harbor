import type { Agent, Tenant, TenantEntitlement, WorkspaceAssignment } from "../../types";

// Tenant directory view-model: one row per (tenant, workspace) pair actually
// in use. Workspaces are not entities (plan F8), so the pairs are derived
// from registered agents plus workspace assignments of the grant chain.

export interface TenantDirectoryRow {
  agentCount: number;
  allowCount: number;
  denyCount: number;
  tenantId: string;
  tenantName: string;
  tenantStatus: "active" | "disabled";
  workspaceId: string;
}

export interface TenantDirectoryInput {
  agents: readonly Agent[];
  entitlements: readonly TenantEntitlement[];
  tenants: readonly Tenant[];
  workspaceAssignments: readonly WorkspaceAssignment[];
}

export function tenantDirectoryRows({
  agents,
  entitlements,
  tenants,
  workspaceAssignments
}: TenantDirectoryInput): TenantDirectoryRow[] {
  const tenantNameById = new Map(tenants.map((tenant) => [tenant.id, tenant.name]));
  const tenantStatusById = new Map(tenants.map((tenant) => [tenant.id, tenant.status]));

  const workspaceIds = new Set<string>();
  const tenantIdByWorkspaceKey = new Map<string, string>();
  const track = (tenantId: string, workspaceId: string) => {
    if (!workspaceId) return;
    workspaceIds.add(workspaceId);
    const key = `${tenantId}\u0000${workspaceId}`;
    if (!tenantIdByWorkspaceKey.has(key)) tenantIdByWorkspaceKey.set(key, tenantId);
  };
  for (const agent of agents) track(agent.tenantId ?? "", agent.workspaceId ?? "");
  const entitlementTenantById = new Map(entitlements.map((entitlement) => [entitlement.id, entitlement]));
  for (const assignment of workspaceAssignments) {
    const tenantId = assignment.tenantId ?? entitlementTenantById.get(assignment.tenantEntitlementId)?.tenantId ?? "";
    track(tenantId, assignment.workspaceId ?? "");
  }

  const agentCounts = new Map<string, number>();
  for (const agent of agents) {
    const key = `${agent.tenantId ?? ""}\u0000${agent.workspaceId ?? ""}`;
    agentCounts.set(key, (agentCounts.get(key) ?? 0) + 1);
  }

  const grantCounts = new Map<string, { allow: number; deny: number }>();
  for (const entitlement of entitlements) {
    const current = grantCounts.get(entitlement.tenantId) ?? { allow: 0, deny: 0 };
    if (entitlement.effect === "deny") current.deny += 1;
    else current.allow += 1;
    grantCounts.set(entitlement.tenantId, current);
  }

  const rows: TenantDirectoryRow[] = [];
  const seenTenants = new Set<string>();
  for (const [key, tenantId] of tenantIdByWorkspaceKey) {
    const workspaceId = key.slice(tenantId.length + 1);
    seenTenants.add(tenantId);
    const grants = grantCounts.get(tenantId) ?? { allow: 0, deny: 0 };
    rows.push({
      agentCount: agentCounts.get(key) ?? 0,
      allowCount: grants.allow,
      denyCount: grants.deny,
      tenantId,
      tenantName: tenantNameById.get(tenantId) ?? tenantId,
      tenantStatus: tenantStatusById.get(tenantId) === "disabled" ? "disabled" : "active",
      workspaceId
    });
  }
  // A freshly created tenant has no workspace or agent yet; it still deserves
  // a directory row, otherwise it is invisible right after creation.
  for (const tenant of tenants) {
    if (seenTenants.has(tenant.id)) continue;
    rows.push({
      agentCount: 0,
      allowCount: 0,
      denyCount: 0,
      tenantId: tenant.id,
      tenantName: tenant.name,
      tenantStatus: tenant.status === "disabled" ? "disabled" : "active",
      workspaceId: ""
    });
  }
  return rows.sort((a, b) => a.tenantName.localeCompare(b.tenantName) || a.workspaceId.localeCompare(b.workspaceId));
}

export interface TenantDirectorySummary {
  subjects: number;
  tenants: number;
  workspaces: number;
}

export function tenantDirectorySummary(
  rows: readonly TenantDirectoryRow[],
  subjectCount: number,
): TenantDirectorySummary {
  return {
    subjects: subjectCount,
    tenants: new Set(rows.map((row) => row.tenantId)).size,
    workspaces: rows.filter((row) => row.workspaceId !== "").length
  };
}

export type TenantDetailTab = "org" | "center" | "profile";

const tenantDetailTabs: readonly TenantDetailTab[] = ["org", "center", "profile"];

export function normalizeTenantDetailTab(value: string | undefined): TenantDetailTab {
  return tenantDetailTabs.includes(value as TenantDetailTab) ? (value as TenantDetailTab) : "org";
}
