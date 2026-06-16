import type {
  DataScope,
  Tenant,
  TenantPermissionCenterNextAction,
  TenantPermissionCenterResponse,
  TenantPermissionCenterStatus,
} from "./types";

export interface TenantPermissionCenterViewModel {
  canManageAdministrators: boolean;
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

export function buildTenantPermissionCenterViewModel(
  center: TenantPermissionCenterResponse,
  options: { selectedWorkspaceId?: string } = {},
): TenantPermissionCenterViewModel {
  const selectedWorkspaceId = options.selectedWorkspaceId || center.workspaces[0]?.workspaceId || "";
  const allowedCapabilities = center.capabilities.filter((capability) => capability.effect === "allow").length;
  const blockedCapabilities = center.capabilities.filter((capability) => capability.effect === "deny").length;
  const emptyReasons = tenantPermissionCenterEmptyReasons(center);
  return {
    canManageAdministrators: center.operatorBoundary.canManageAdministrators,
    dataScopeLabels: uniqueDataScopeLabels(center.permissionPackages.flatMap((item) => item.dataScopes ?? [])),
    emptyReasons,
    metric: {
      administrators: center.administrators.length,
      allowedCapabilities,
      blockedCapabilities,
      packages: center.permissionPackages.length,
      workspaces: center.workspaces.length,
    },
    operatorBoundaryLabel: operatorBoundaryLabel(center),
    primaryActions: center.nextActions,
    selectedWorkspaceId,
    status: emptyReasons.length > 0 ? "blocked" : strongestStatus(center.permissionPackages),
    tenantName: center.tenant.name || center.tenant.id,
    tenantPath: tenantPathLabel(center.scopeTenants, center.tenant),
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

function tenantPermissionCenterEmptyReasons(center: TenantPermissionCenterResponse) {
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
