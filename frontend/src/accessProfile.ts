import type {
  AccessProfileFilters,
  AccessProfileScopeStatus,
  DataScope,
  TenantAccessProfile,
  TenantAccessProfileGrant,
} from './types'

export const DEFAULT_ACCESS_PROFILE_TRACE_LIMIT = 20

export type AccessProfileTone = 'success' | 'warning' | 'danger' | 'info' | 'neutral'

export type AccessProfileTraceLimitResult =
  | { ok: true; value: number }
  | { ok: false; message: string }

export function normalizeAccessProfileFilters(
  filters: AccessProfileFilters = {},
): Record<string, string | undefined> {
  return {
    callerInstanceId: trimmed(filters.callerInstanceId),
    capabilityId: trimmed(filters.capabilityId),
    targetId: trimmed(filters.targetId),
    traceLimit: traceLimitQueryValue(filters.traceLimit),
    workspaceId: trimmed(filters.workspaceId),
  }
}

export function parseAccessProfileTraceLimit(value: number | string | undefined): AccessProfileTraceLimitResult {
  const raw = value === undefined || value === '' ? String(DEFAULT_ACCESS_PROFILE_TRACE_LIMIT) : String(value).trim()
  const parsed = Number(raw)
  if (!Number.isInteger(parsed) || parsed < 0 || parsed > 100) {
    return { ok: false, message: 'Trace limit must be an integer between 0 and 100.' }
  }
  return { ok: true, value: parsed }
}

export function scopeStatusTone(status: AccessProfileScopeStatus): AccessProfileTone {
  return status === 'invalid' ? 'danger' : 'success'
}

export function summarizeDataScopes(
  scopes?: DataScope[],
  emptyLabel = 'no data scope',
  readableLabels: Record<string, string> = {},
): string {
  if (!scopes || scopes.length === 0) return emptyLabel
  const labels = scopes
    .map((scope) =>
      [scope.dataDomain, scope.dataset, scope.schema, scope.table, scope.field, scope.classification, scope.region]
        .filter((value): value is string => Boolean(value))
        .map((value) => readableDataScopeValue(value, readableLabels))
        .join('/'),
    )
    .filter(Boolean)
  if (labels.length === 0) return emptyLabel
  return labels.length > 2 ? `${labels.slice(0, 2).join(', ')} +${labels.length - 2}` : labels.join(', ')
}

export function countInvalidAccessProfileRows(profile?: TenantAccessProfile | null): number {
  if (!profile) return 0
  return (profile.grants ?? []).reduce((total, grant) => total + countInvalidGrantRows(grant), 0)
}

export function countInvalidGrantRows(grant: TenantAccessProfileGrant): number {
  const grantInvalid = grant.scopeStatus === 'invalid' ? 1 : 0
  return (grant.workspaceAssignments ?? []).reduce((workspaceTotal, workspace) => {
    const workspaceInvalid = workspace.scopeStatus === 'invalid' ? 1 : 0
    const instanceInvalid = (workspace.instanceAssignments ?? []).filter((instance) => instance.scopeStatus === 'invalid').length
    return workspaceTotal + workspaceInvalid + instanceInvalid
  }, grantInvalid)
}

function traceLimitQueryValue(value: AccessProfileFilters['traceLimit']): string | undefined {
  const parsed = parseAccessProfileTraceLimit(value)
  return parsed.ok ? String(parsed.value) : undefined
}

function trimmed(value?: string): string | undefined {
  const next = value?.trim()
  return next || undefined
}

function readableDataScopeValue(value: string, labels: Record<string, string>): string {
  return labels[value] ?? value
}
