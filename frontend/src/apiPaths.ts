import type { PermissionPackageApprovalStatus } from './permissionPackages'

function queryString(params: Record<string, string | undefined>): string {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value?.trim()) query.set(key, value.trim())
  })
  const value = query.toString()
  return value ? `?${value}` : ''
}

export interface PermissionPackageApprovalRequestPathFilter {
  callerInstanceId?: string
  limit?: number
  reviewer?: string
  status?: PermissionPackageApprovalStatus
  targetId?: string
  templateId?: string
  tenantId?: string
  workspaceId?: string
}

export function permissionPackageApprovalRequestsPath(filter: PermissionPackageApprovalRequestPathFilter): string {
  const query = queryString({
    callerInstanceId: filter.callerInstanceId,
    limit: filter.limit ? String(filter.limit) : undefined,
    reviewer: filter.reviewer,
    status: filter.status,
    targetId: filter.targetId,
    templateId: filter.templateId,
    tenantId: filter.tenantId,
    workspaceId: filter.workspaceId,
  })
  return `/api/v1/permission-packages/approval-requests${query}`
}
