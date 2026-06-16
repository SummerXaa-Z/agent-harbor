import type {
  DataScope,
  Tenant,
  TenantPermissionCenterNextAction,
  TenantPermissionCenterResponse,
  TenantPermissionCenterStatus,
} from "./types";

export interface TenantPermissionCenterViewModel {
  canManageAdministrators: boolean;
  capabilitySummaries: TenantPermissionCenterCapabilitySummary[];
  dataScopeLabels: string[];
  emptyReasons: string[];
  metric: {
    administrators: number;
    allowedCapabilities: number;
    blockedCapabilities: number;
    packages: number;
    workspaces: number;
  };
  operatorBoundaryLabel: string;
  primaryActions: TenantPermissionCenterNextAction[];
  selectedWorkspaceId: string;
  status: TenantPermissionCenterStatus;
  tenantName: string;
  tenantPath: string;
}

export interface TenantPermissionCenterCapabilitySummary {
  capabilityName: string;
  dataScopeLabels: string[];
  effect: "allow" | "deny";
  targetName: string;
  workspaceIds: string[];
}

export function buildTenantPermissionCenterViewModel(
  center: TenantPermissionCenterResponse,
  options: { selectedWorkspaceId?: string } = {},
): TenantPermissionCenterViewModel {
  const administrators = center.administrators ?? [];
  const capabilities = center.capabilities ?? [];
  const permissionPackages = center.permissionPackages ?? [];
  const scopeTenants = center.scopeTenants ?? [];
  const workspaces = center.workspaces ?? [];
  const selectedWorkspaceId = options.selectedWorkspaceId || workspaces[0]?.workspaceId || "";
  const allowedCapabilities = capabilities.filter((capability) => capability.effect === "allow").length;
  const blockedCapabilities = capabilities.filter((capability) => capability.effect === "deny").length;
  const emptyReasons = tenantPermissionCenterEmptyReasons({ capabilities, workspaces });
  return {
    canManageAdministrators: center.operatorBoundary.canManageAdministrators,
    capabilitySummaries: capabilities.slice(0, 4).map((capability) => ({
      capabilityName: capability.capabilityName || capability.capabilityId,
      dataScopeLabels: uniqueDataScopeLabels(capability.dataScopes ?? []),
      effect: capability.effect,
      targetName: capability.targetName || capability.targetId,
      workspaceIds: capability.workspaceIds ?? [],
    })),
    dataScopeLabels: uniqueDataScopeLabels([
      ...permissionPackages.flatMap((item) => item.dataScopes ?? []),
      ...capabilities.flatMap((item) => item.dataScopes ?? []),
    ]),
    emptyReasons,
    metric: {
      administrators: administrators.length,
      allowedCapabilities,
      blockedCapabilities,
      packages: permissionPackages.length,
      workspaces: workspaces.length,
    },
    operatorBoundaryLabel: operatorBoundaryLabel(center),
    primaryActions: center.nextActions ?? [],
    selectedWorkspaceId,
    status: emptyReasons.length > 0 ? "blocked" : strongestStatus(permissionPackages),
    tenantName: center.tenant.name || center.tenant.id,
    tenantPath: tenantPathLabel(scopeTenants, center.tenant),
  };
}

export function tenantPermissionCenterActionTarget(actions: TenantPermissionCenterNextAction[], code: string) {
  return actions.find((action) => action.code === code)?.targetView ?? "";
}

function tenantPathLabel(scopeTenants: Tenant[], tenant: Tenant) {
  const tenantById = new Map(scopeTenants.map((row) => [row.id, row]));
  const path: Tenant[] = [];
  let cursor: Tenant | undefined = tenant;
  while (cursor) {
    path.unshift(cursor);
    cursor = cursor.parentTenantId ? tenantById.get(cursor.parentTenantId) : undefined;
  }
  return path.map((row) => row.name || row.id).join(" / ");
}

function operatorBoundaryLabel(center: TenantPermissionCenterResponse) {
  const tenantName = center.operatorBoundary.tenantId
    ? center.scopeTenants.find((tenant) => tenant.id === center.operatorBoundary.tenantId)?.name || center.operatorBoundary.tenantId
    : "All tenants";
  const workspace = center.operatorBoundary.workspaceId || "All workspaces";
  return `${center.operatorBoundary.role} / ${tenantName} / ${workspace}`;
}

function strongestStatus(packages: TenantPermissionCenterResponse["permissionPackages"]) {
  if (packages.some((item) => item.status === "blocked")) return "blocked";
  if (packages.some((item) => item.status === "needs_review")) return "needs_review";
  return "ready";
}

function tenantPermissionCenterEmptyReasons(center: Pick<TenantPermissionCenterResponse, "workspaces" | "capabilities">) {
  const reasons: string[] = [];
  if (center.workspaces.length === 0) reasons.push("tenantCenter.empty.noWorkspaces");
  if (center.capabilities.length === 0) reasons.push("tenantCenter.empty.noCapabilities");
  return reasons;
}

function uniqueDataScopeLabels(scopes: DataScope[]) {
  const labels = scopes.map(dataScopeLabel).filter(Boolean);
  return Array.from(new Set(labels));
}

function dataScopeLabel(scope: DataScope) {
  return [scope.dataDomain, scope.dataset, scope.region, scope.classification].filter(Boolean).join(" / ");
}
