import {
  evidenceRuns,
  routePolicies,
  sampleAgents,
  sampleAuditEvents,
  sampleCapabilities,
  sampleChannels,
  sampleInstanceAssignments,
  sampleProviders,
  sampleTenantAccessProfile,
  sampleTenantEntitlements,
  sampleTraces,
  sampleWorkspaceAssignments,
  systemMetrics,
} from './data'
import { normalizeAccessProfileFilters } from './accessProfile'
import {
  accessDecisionExplainPath,
  permissionPackageApplicationImpactPath,
  permissionPackageApprovalRequestsPath,
  type PermissionPackageApplicationImpactPathScope,
  type PermissionPackageApprovalRequestPathFilter,
} from './apiPaths'
import type {
  PermissionPackageApprovalRequest,
  PermissionPackageApprovalStatus,
  PermissionPackageApplyInput,
  PermissionPackageApplyResult,
  PermissionPackageApplicationImpact,
  PermissionPackageDraft,
  PermissionPackageDraftInput,
  PermissionPackageTemplate,
} from './permissionPackages'
import type {
  AccessGrant,
  AccessDecisionExplainRequest,
  AccessDecisionExplainResult,
  AccessProfileFilters,
  Agent,
  ApiEnvelope,
  AuditEvent,
  Capability,
  CapabilityDiscoveryStatus,
  CatalogData,
  ChannelContract,
  ConsoleData,
  CreateAccessGrantRequest,
  CreateAgentKeyRequest,
  CreateAgentKeyResponse,
  CreateAgentRequest,
  CreateInstanceAssignmentRequest,
  CreateRoutePolicyRequest,
  CreateTenantRequest,
  CreateTenantEntitlementRequest,
  CreateWorkspaceAssignmentRequest,
  InstanceAssignment,
  ManagementScope,
  McpRpcCallResult,
  ProviderContract,
  RotateAgentCredentialsRequest,
  RoutePolicy,
  SystemMetric,
  Tenant,
  TenantAccessProfile,
  TenantAccessProfileData,
  TenantEntitlement,
  TraceEvent,
  TraceFilters,
  UpdateAgentRequest,
  UpdateCapabilityRequest,
  WorkspaceAssignment,
} from './types'

const DEFAULT_API_BASE = 'http://127.0.0.1:9090'

export type HealthCheckStatus = 'ok' | 'error'

export interface HealthCheckResult {
  status: HealthCheckStatus
  message: string
}

export const defaultMockMcpHealthUrl = 'http://127.0.0.1:8787/healthz'

type ViteImportMeta = ImportMeta & {
  env?: {
    VITE_API_BASE?: string
  }
}

export const apiBase =
  (import.meta as ViteImportMeta).env?.VITE_API_BASE?.replace(/\/+$/, '') || DEFAULT_API_BASE

function endpoint(path: string): string {
  return `${apiBase}${path.startsWith('/') ? path : `/${path}`}`
}

function queryString(params: Record<string, string | undefined>): string {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value?.trim()) query.set(key, value.trim())
  })
  const value = query.toString()
  return value ? `?${value}` : ''
}

function isEnvelope<T>(value: unknown): value is ApiEnvelope<T> {
  return Boolean(value && typeof value === 'object' && 'code' in value)
}

interface RequestOptions {
  adminKey?: string
  bearerToken?: string
  body?: unknown
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  runId?: string
  signal?: AbortSignal
}

export class ApiRequestError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiRequestError'
    this.status = status
  }
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (options.adminKey?.trim()) headers['X-Admin-Key'] = options.adminKey.trim()
  if (options.bearerToken?.trim()) headers.Authorization = `Bearer ${options.bearerToken.trim()}`
  if (options.runId?.trim()) headers['X-Run-Id'] = options.runId.trim()
  if (options.body !== undefined) headers['Content-Type'] = 'application/json'

  const response = await fetch(endpoint(path), {
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    headers,
    method: options.method ?? (options.body === undefined ? 'GET' : 'POST'),
    signal: options.signal,
  })

  let payload: unknown
  try {
    payload = (await response.json()) as unknown
  } catch {
    payload = undefined
  }
  if (!response.ok) {
    const message = isEnvelope<T>(payload) ? payload.message || payload.error : response.statusText
    throw new ApiRequestError(response.status, message || `Request failed with status ${response.status}`)
  }

  if (isEnvelope<T>(payload)) {
    if (payload.code !== 0) {
      throw new Error(payload.message || payload.error || 'API returned an error')
    }
    if (payload.data === undefined) {
      throw new Error('API response did not include data')
    }
    return payload.data
  }

  return payload as T
}

async function withFallback<T>(loader: () => Promise<T>, fallback: T): Promise<{ data: T; ok: boolean }> {
  try {
    return { data: await loader(), ok: true }
  } catch (error) {
    if (isFetchNetworkError(error)) {
      return { data: fallback, ok: false }
    }
    throw error
  }
}

function isFetchNetworkError(error: unknown): boolean {
  if (!(error instanceof TypeError)) return false
  const message = error.message.toLowerCase()
  return (
    message.includes('failed to fetch') ||
    message.includes('load failed') ||
    message.includes('networkerror') ||
    message.includes('network request failed')
  )
}

export function isApiCompatibilityFallbackError(error: unknown): boolean {
  return isFetchNetworkError(error) || (error instanceof ApiRequestError && error.status === 404)
}

export async function checkApiHealth(signal?: AbortSignal): Promise<HealthCheckResult> {
  return checkJsonHealth(endpoint('/healthz'), signal)
}

export async function checkMockMcpHealth(
  url: string = defaultMockMcpHealthUrl,
  signal?: AbortSignal,
): Promise<HealthCheckResult> {
  return checkJsonHealth(url, signal)
}

export async function checkSubjectHeaderCors(signal?: AbortSignal): Promise<HealthCheckResult> {
  try {
    const response = await fetch(endpoint('/healthz'), {
      headers: {
        Accept: 'application/json',
        'X-AgentHarbor-Subject-Id': 'preflight-probe',
      },
      signal,
    })
    if (!response.ok) {
      return { status: 'error', message: `HTTP ${response.status}` }
    }
    return { status: 'ok', message: 'ok' }
  } catch (error) {
    return {
      status: 'error',
      message: error instanceof Error ? error.message : 'subject header check failed',
    }
  }
}

async function checkJsonHealth(url: string, signal?: AbortSignal): Promise<HealthCheckResult> {
  try {
    const response = await fetch(url, {
      headers: { Accept: 'application/json' },
      signal,
    })
    if (!response.ok) {
      return { status: 'error', message: `HTTP ${response.status}` }
    }
    return { status: 'ok', message: 'ok' }
  } catch (error) {
    return {
      status: 'error',
      message: error instanceof Error ? error.message : 'health check failed',
    }
  }
}

export async function fetchProviders(signal?: AbortSignal): Promise<ProviderContract[]> {
  return request<ProviderContract[]>('/api/v1/contracts/providers', { signal })
}

export async function fetchChannels(signal?: AbortSignal): Promise<ChannelContract[]> {
  return request<ChannelContract[]>('/api/v1/contracts/channels', { signal })
}

export async function fetchCatalog(signal?: AbortSignal): Promise<CatalogData> {
  const [providers, channels] = await Promise.all([fetchProviders(signal), fetchChannels(signal)])
  return { providers, channels }
}

export async function fetchAgents(
  scope?: ManagementScope,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<Agent[]> {
  const query = queryString({
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<Agent[]>(`/api/v1/agents${query}`, { adminKey, signal })
}

export async function fetchAccessGrants(
  scope?: ManagementScope,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<AccessGrant[]> {
  const query = queryString({
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<AccessGrant[]>(`/api/v1/access-grants${query}`, { adminKey, signal })
}

export async function fetchRoutePolicies(
  scope?: ManagementScope,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<RoutePolicy[]> {
  const query = queryString({
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<RoutePolicy[]>(`/api/v1/route-policies${query}`, { adminKey, signal })
}

export async function fetchTraces(
  filters: TraceFilters = {},
  scope?: ManagementScope,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<TraceEvent[]> {
  const query = queryString({
    callerAgentId: filters.callerAgentId,
    decision: filters.decision || undefined,
    runId: filters.runId,
    targetAgentId: filters.targetAgentId,
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<TraceEvent[]>(`/api/v1/audit/traces${query}`, { adminKey, signal })
}

type AuditEventScope = Partial<ManagementScope> & {
  action?: string
  resourceType?: string
  resourceId?: string
}

type CapabilityFilter = {
  targetId?: string
  status?: CapabilityDiscoveryStatus | ''
}

type EntitlementFilter = {
  targetId?: string
  capabilityId?: string
}

type WorkspaceAssignmentFilter = {
  entitlementId?: string
}

type InstanceAssignmentFilter = {
  callerInstanceId?: string
  capabilityId?: string
}

export async function fetchAuditEvents(
  scope?: AuditEventScope,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<AuditEvent[]> {
  const query = queryString({
    action: scope?.action,
    resourceId: scope?.resourceId,
    resourceType: scope?.resourceType,
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<AuditEvent[]>(`/api/v1/audit/events${query}`, { adminKey, signal })
}

export async function fetchRuntimeMetrics(
  scope?: ManagementScope,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<SystemMetric[]> {
  const query = queryString({
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<SystemMetric[]>(`/api/v1/metrics/runtime${query}`, { adminKey, signal })
}

export async function fetchTenantAccessProfile(
  tenantId: string,
  adminKey?: string,
  filters: AccessProfileFilters = {},
  signal?: AbortSignal,
): Promise<TenantAccessProfile> {
  const query = queryString(normalizeAccessProfileFilters(filters))
  return request<TenantAccessProfile>(
    `/api/v1/tenants/${encodeURIComponent(tenantId.trim())}/access-profile${query}`,
    { adminKey, signal },
  )
}

export async function loadTenantAccessProfile(
  tenantId: string,
  adminKey?: string,
  filters: AccessProfileFilters = {},
  signal?: AbortSignal,
): Promise<TenantAccessProfileData> {
  const result = await withFallback(
    () => fetchTenantAccessProfile(tenantId, adminKey, filters, signal),
    sampleTenantAccessProfile,
  )
  return {
    ...result.data,
    loadedFromApi: result.ok,
    apiBase,
  }
}

export async function fetchAccessDecisionExplanation(
  requestBody: AccessDecisionExplainRequest,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<AccessDecisionExplainResult> {
  return request<AccessDecisionExplainResult>(accessDecisionExplainPath(requestBody), { adminKey, signal })
}

export async function refreshTargetCapabilities(targetId: string, adminKey?: string): Promise<Capability[]> {
  return request<Capability[]>(`/api/v1/targets/${encodeURIComponent(targetId)}/capabilities:refresh`, {
    adminKey,
    method: 'POST',
  })
}

export async function createTenant(body: CreateTenantRequest, adminKey?: string): Promise<Tenant> {
  return request<Tenant>('/api/v1/tenants', { adminKey, body })
}

export async function callMcpRpc(
  targetId: string,
  body: unknown,
  agentKey: string,
  runId: string,
  adminKey?: string,
  subjectId?: string,
): Promise<McpRpcCallResult> {
  const headers: Record<string, string> = {
    Accept: 'application/json',
    Authorization: `Bearer ${agentKey.trim()}`,
    'Content-Type': 'application/json',
  }
  if (adminKey?.trim()) headers['X-Admin-Key'] = adminKey.trim()
  if (runId.trim()) headers['X-Run-Id'] = runId.trim()
  if (subjectId?.trim()) headers['X-AgentHarbor-Subject-Id'] = subjectId.trim()

  const response = await fetch(endpoint(`/api/v1/mcp/agents/${encodeURIComponent(targetId)}/rpc`), {
    body: JSON.stringify(body),
    headers,
    method: 'POST',
  })

  let payload: unknown
  try {
    payload = (await response.json()) as unknown
  } catch {
    payload = undefined
  }

  if (!response.ok && response.status !== 403) {
    const message = isEnvelope<unknown>(payload) ? payload.message || payload.error : response.statusText
    throw new ApiRequestError(response.status, message || `Request failed with status ${response.status}`)
  }

  return {
    ok: response.ok,
    payload,
    status: response.status,
  }
}

export async function fetchCapabilities(
  scope?: ManagementScope,
  adminKey?: string,
  filter: CapabilityFilter = {},
  signal?: AbortSignal,
): Promise<Capability[]> {
  const query = queryString({
    status: filter.status || undefined,
    targetId: filter.targetId,
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<Capability[]>(`/api/v1/capabilities${query}`, { adminKey, signal })
}

export async function fetchPermissionPackageTemplates(
  adminKey?: string,
  signal?: AbortSignal,
): Promise<PermissionPackageTemplate[]> {
  return request<PermissionPackageTemplate[]>('/api/v1/permission-packages/templates', { adminKey, signal })
}

export async function createPermissionPackageDraftFromApi(
  body: PermissionPackageDraftInput,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<PermissionPackageDraft> {
  return request<PermissionPackageDraft>('/api/v1/permission-packages/drafts', { adminKey, body, signal })
}

export async function applyPermissionPackage(
  body: PermissionPackageApplyInput,
  adminKey?: string,
): Promise<PermissionPackageApplyResult> {
  return request<PermissionPackageApplyResult>('/api/v1/permission-packages:apply', { adminKey, body })
}

export async function fetchPermissionPackageApprovalRequests(
  filter: PermissionPackageApprovalRequestPathFilter,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<PermissionPackageApprovalRequest[]> {
  return request<PermissionPackageApprovalRequest[]>(permissionPackageApprovalRequestsPath(filter), { adminKey, signal })
}

export async function fetchPermissionPackageApplicationImpact(
  applicationId: string,
  scope?: PermissionPackageApplicationImpactPathScope,
  adminKey?: string,
  signal?: AbortSignal,
): Promise<PermissionPackageApplicationImpact> {
  return request<PermissionPackageApplicationImpact>(
    permissionPackageApplicationImpactPath(applicationId, scope),
    { adminKey, signal },
  )
}

export async function createPermissionPackageApprovalRequest(
  body: PermissionPackageDraftInput,
  adminKey?: string,
): Promise<PermissionPackageApprovalRequest> {
  return request<PermissionPackageApprovalRequest>('/api/v1/permission-packages/approval-requests', { adminKey, body })
}

export async function approvePermissionPackageApprovalRequest(
  id: string,
  body: { comment?: string; reviewer?: string } = {},
  adminKey?: string,
): Promise<PermissionPackageApprovalRequest> {
  return request<PermissionPackageApprovalRequest>(`/api/v1/permission-packages/approval-requests/${encodeURIComponent(id)}/approve`, { adminKey, body })
}

export async function rejectPermissionPackageApprovalRequest(
  id: string,
  body: { comment?: string; reviewer?: string } = {},
  adminKey?: string,
): Promise<PermissionPackageApprovalRequest> {
  return request<PermissionPackageApprovalRequest>(`/api/v1/permission-packages/approval-requests/${encodeURIComponent(id)}/reject`, { adminKey, body })
}

export async function fetchTenantEntitlements(
  scope?: ManagementScope,
  adminKey?: string,
  filter: EntitlementFilter = {},
  signal?: AbortSignal,
): Promise<TenantEntitlement[]> {
  const query = queryString({
    capabilityId: filter.capabilityId,
    targetId: filter.targetId,
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<TenantEntitlement[]>(`/api/v1/tenant-entitlements${query}`, { adminKey, signal })
}

export async function fetchWorkspaceAssignments(
  scope?: ManagementScope,
  adminKey?: string,
  filter: WorkspaceAssignmentFilter = {},
  signal?: AbortSignal,
): Promise<WorkspaceAssignment[]> {
  const query = queryString({
    entitlementId: filter.entitlementId,
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<WorkspaceAssignment[]>(`/api/v1/workspace-assignments${query}`, { adminKey, signal })
}

export async function fetchInstanceAssignments(
  scope?: ManagementScope,
  adminKey?: string,
  filter: InstanceAssignmentFilter = {},
  signal?: AbortSignal,
): Promise<InstanceAssignment[]> {
  const query = queryString({
    callerInstanceId: filter.callerInstanceId,
    capabilityId: filter.capabilityId,
    tenantId: scope?.tenantId,
    workspaceId: scope?.workspaceId,
  })
  return request<InstanceAssignment[]>(`/api/v1/instance-assignments${query}`, { adminKey, signal })
}

export async function createAgent(body: CreateAgentRequest, adminKey?: string): Promise<Agent> {
  return request<Agent>('/api/v1/agents', { adminKey, body })
}

export async function updateAgent(id: string, body: UpdateAgentRequest, adminKey?: string): Promise<Agent> {
  return request<Agent>(`/api/v1/agents/${encodeURIComponent(id)}`, { adminKey, body, method: 'PATCH' })
}

export async function rotateAgentCredentials(
  id: string,
  body: RotateAgentCredentialsRequest,
  adminKey?: string,
): Promise<Agent> {
  return request<Agent>(`/api/v1/agents/${encodeURIComponent(id)}/credentials:rotate`, { adminKey, body })
}

export async function disableAgent(id: string, adminKey?: string): Promise<Agent> {
  return request<Agent>(`/api/v1/agents/${encodeURIComponent(id)}`, { adminKey, method: 'DELETE' })
}

export async function createAgentKey(
  body: CreateAgentKeyRequest,
  adminKey?: string,
): Promise<CreateAgentKeyResponse> {
  return request<CreateAgentKeyResponse>('/api/v1/agent-keys', { adminKey, body })
}

export async function createAccessGrant(
  body: CreateAccessGrantRequest,
  adminKey?: string,
): Promise<AccessGrant> {
  return request<AccessGrant>('/api/v1/access-grants', { adminKey, body })
}

export async function revokeAccessGrant(id: string, adminKey?: string): Promise<AccessGrant> {
  return request<AccessGrant>(`/api/v1/access-grants/${encodeURIComponent(id)}`, {
    adminKey,
    method: 'DELETE',
  })
}

export async function createRoutePolicy(
  body: CreateRoutePolicyRequest,
  adminKey?: string,
): Promise<RoutePolicy> {
  return request<RoutePolicy>('/api/v1/route-policies', { adminKey, body })
}

export async function updateCapability(
  id: string,
  body: UpdateCapabilityRequest,
  adminKey?: string,
): Promise<Capability> {
  return request<Capability>(`/api/v1/capabilities/${encodeURIComponent(id)}`, {
    adminKey,
    body,
    method: 'PATCH',
  })
}

export async function createTenantEntitlement(
  body: CreateTenantEntitlementRequest,
  adminKey?: string,
): Promise<TenantEntitlement> {
  return request<TenantEntitlement>('/api/v1/tenant-entitlements', { adminKey, body })
}

export async function createWorkspaceAssignment(
  body: CreateWorkspaceAssignmentRequest,
  adminKey?: string,
): Promise<WorkspaceAssignment> {
  return request<WorkspaceAssignment>('/api/v1/workspace-assignments', { adminKey, body })
}

export async function createInstanceAssignment(
  body: CreateInstanceAssignmentRequest,
  adminKey?: string,
): Promise<InstanceAssignment> {
  return request<InstanceAssignment>('/api/v1/instance-assignments', { adminKey, body })
}

export async function disableRoutePolicy(id: string, adminKey?: string): Promise<RoutePolicy> {
  return request<RoutePolicy>(`/api/v1/route-policies/${encodeURIComponent(id)}`, {
    adminKey,
    method: 'DELETE',
  })
}

export async function loadConsoleData(
  adminKey?: string,
  traceFilters: TraceFilters = {},
  scope?: ManagementScope,
): Promise<ConsoleData> {
  const [
    catalogResult,
    agentsResult,
    grantsResult,
    capabilitiesResult,
    entitlementResult,
    workspaceAssignmentResult,
    instanceAssignmentResult,
    policiesResult,
    tracesResult,
    auditEventsResult,
    metricsResult,
  ] = await Promise.all([
    withFallback(() => fetchCatalog(), {
      providers: sampleProviders,
      channels: sampleChannels,
    }),
    withFallback(() => fetchAgents(scope, adminKey), sampleAgents),
    withFallback(() => fetchAccessGrants(scope, adminKey), []),
    withFallback(() => fetchCapabilities(scope, adminKey), sampleCapabilities),
    withFallback(() => fetchTenantEntitlements(scope, adminKey), sampleTenantEntitlements),
    withFallback(() => fetchWorkspaceAssignments(scope, adminKey), sampleWorkspaceAssignments),
    withFallback(() => fetchInstanceAssignments(scope, adminKey), sampleInstanceAssignments),
    withFallback(() => fetchRoutePolicies(scope, adminKey), routePolicies),
    withFallback(() => fetchTraces(traceFilters, scope, adminKey), sampleTraces),
    withFallback(() => fetchAuditEvents(scope, adminKey), sampleAuditEvents),
    withFallback(() => fetchRuntimeMetrics(scope, adminKey), systemMetrics),
  ])

  const loadedFromApi =
    catalogResult.ok &&
    agentsResult.ok &&
    grantsResult.ok &&
    capabilitiesResult.ok &&
    entitlementResult.ok &&
    workspaceAssignmentResult.ok &&
    instanceAssignmentResult.ok &&
    policiesResult.ok &&
    tracesResult.ok &&
    auditEventsResult.ok &&
    metricsResult.ok

  return {
    providers: catalogResult.data.providers,
    channels: catalogResult.data.channels,
    agents: agentsResult.data,
    accessGrants: grantsResult.data,
    capabilities: capabilitiesResult.data,
    tenantEntitlements: entitlementResult.data,
    workspaceAssignments: workspaceAssignmentResult.data,
    instanceAssignments: instanceAssignmentResult.data,
    traces: tracesResult.data,
    auditEvents: auditEventsResult.data,
    routePolicies: policiesResult.data,
    evidenceRuns: loadedFromApi ? [] : evidenceRuns,
    systemMetrics: metricsResult.data,
    loadedFromApi,
    grantsLoadedFromApi: grantsResult.ok,
    capabilitiesLoadedFromApi: capabilitiesResult.ok,
    capabilityAssignmentsLoadedFromApi:
      entitlementResult.ok && workspaceAssignmentResult.ok && instanceAssignmentResult.ok,
    routePoliciesLoadedFromApi: policiesResult.ok,
    apiBase,
  }
}
