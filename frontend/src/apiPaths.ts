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

export interface AccessDecisionExplainPathRequest {
  callerInstanceId?: string
  capabilityId?: string
  subjectId?: string
  targetId?: string
  tenantId?: string
  workspaceId?: string
}

export function accessDecisionExplainPath(request: AccessDecisionExplainPathRequest): string {
  const query = queryString({
    callerInstanceId: request.callerInstanceId,
    capabilityId: request.capabilityId,
    subjectId: request.subjectId,
    targetId: request.targetId,
    tenantId: request.tenantId,
    workspaceId: request.workspaceId,
  })
  return `/api/v1/access-decisions:explain${query}`
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
