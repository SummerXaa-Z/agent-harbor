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
import type {
  AccessGrant,
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
  CreateTenantEntitlementRequest,
  CreateWorkspaceAssignmentRequest,
  InstanceAssignment,
  ManagementScope,
  ProviderContract,
  RotateAgentCredentialsRequest,
  RoutePolicy,
  SystemMetric,
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
  body?: unknown
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  signal?: AbortSignal
}

class ApiRequestError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiRequestError'
  }
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (options.adminKey?.trim()) headers['X-Admin-Key'] = options.adminKey.trim()
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

export async function refreshTargetCapabilities(targetId: string, adminKey?: string): Promise<Capability[]> {
  return request<Capability[]>(`/api/v1/targets/${encodeURIComponent(targetId)}/capabilities:refresh`, {
    adminKey,
    method: 'POST',
  })
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
