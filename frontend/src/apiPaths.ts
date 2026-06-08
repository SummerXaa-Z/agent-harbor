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

export interface PermissionPackageApplicationHealthPathFilter {
  callerInstanceId?: string
  limit?: number
  targetId?: string
  templateId?: string
  tenantId?: string
  workspaceId?: string
}

export interface PermissionPackageProductionReadinessPathFilter {
  approvalRequestId?: string
  callerInstanceId?: string
  region?: string
  requestText?: string
  subjectId?: string
  subjectSelector?: string
  targetId?: string
  templateId?: string
  tenantId?: string
  traceLimit?: number
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

export interface PermissionPackageApplicationImpactPathScope {
  rehearsal?: 'grant_drift'
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

export function permissionPackageApplicationImpactPath(
  applicationId: string,
  scope: PermissionPackageApplicationImpactPathScope = {},
): string {
  const query = queryString({
    rehearsal: scope.rehearsal,
    tenantId: scope.tenantId,
    workspaceId: scope.workspaceId,
  })
  return `/api/v1/permission-packages/applications/${encodeURIComponent(applicationId.trim())}/impact${query}`
}

export function permissionPackageApplicationHealthPath(filter: PermissionPackageApplicationHealthPathFilter): string {
  const query = queryString({
    callerInstanceId: filter.callerInstanceId,
    limit: filter.limit ? String(filter.limit) : undefined,
    targetId: filter.targetId,
    templateId: filter.templateId,
    tenantId: filter.tenantId,
    workspaceId: filter.workspaceId,
  })
  return `/api/v1/permission-packages/applications/health${query}`
}

export function permissionPackageProductionReadinessPath(filter: PermissionPackageProductionReadinessPathFilter): string {
  const query = queryString({
    approvalRequestId: filter.approvalRequestId,
    callerInstanceId: filter.callerInstanceId,
    region: filter.region,
    requestText: filter.requestText,
    subjectId: filter.subjectId,
    subjectSelector: filter.subjectSelector,
    targetId: filter.targetId,
    templateId: filter.templateId,
    tenantId: filter.tenantId,
    traceLimit: filter.traceLimit ? String(filter.traceLimit) : undefined,
    workspaceId: filter.workspaceId,
  })
  return `/api/v1/permission-packages/production-readiness${query}`
}

export function permissionPackageProductionEvidenceReportPath(filter: PermissionPackageProductionReadinessPathFilter): string {
  const query = queryString({
    approvalRequestId: filter.approvalRequestId,
    callerInstanceId: filter.callerInstanceId,
    region: filter.region,
    requestText: filter.requestText,
    subjectId: filter.subjectId,
    subjectSelector: filter.subjectSelector,
    targetId: filter.targetId,
    templateId: filter.templateId,
    tenantId: filter.tenantId,
    traceLimit: filter.traceLimit ? String(filter.traceLimit) : undefined,
    workspaceId: filter.workspaceId,
  })
  return `/api/v1/permission-packages/production-readiness/report${query}`
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
