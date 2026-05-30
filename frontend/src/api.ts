import {
  evidenceRuns,
  routePolicies,
  sampleAgents,
  sampleChannels,
  sampleProviders,
  sampleTraces,
  systemMetrics,
} from './data'
import type {
  AccessGrant,
  Agent,
  ApiEnvelope,
  CatalogData,
  ChannelContract,
  ConsoleData,
  CreateAccessGrantRequest,
  CreateAgentKeyRequest,
  CreateAgentKeyResponse,
  CreateAgentRequest,
  ManagementScope,
  ProviderContract,
  TraceEvent,
  TraceFilters,
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
  method?: 'GET' | 'POST' | 'DELETE'
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
    if (error instanceof TypeError) {
      return { data: fallback, ok: false }
    }
    throw error
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

export async function createAgent(body: CreateAgentRequest, adminKey?: string): Promise<Agent> {
  return request<Agent>('/api/v1/agents', { adminKey, body })
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

export async function loadConsoleData(
  adminKey?: string,
  traceFilters: TraceFilters = {},
  scope?: ManagementScope,
): Promise<ConsoleData> {
  const [catalogResult, agentsResult, grantsResult, tracesResult] = await Promise.all([
    withFallback(() => fetchCatalog(), {
      providers: sampleProviders,
      channels: sampleChannels,
    }),
    withFallback(() => fetchAgents(scope, adminKey), sampleAgents),
    withFallback(() => fetchAccessGrants(scope, adminKey), []),
    withFallback(() => fetchTraces(traceFilters, scope, adminKey), sampleTraces),
  ])

  return {
    providers: catalogResult.data.providers,
    channels: catalogResult.data.channels,
    agents: agentsResult.data,
    accessGrants: grantsResult.data,
    traces: tracesResult.data,
    routePolicies: grantsResult.ok ? [] : routePolicies,
    evidenceRuns,
    systemMetrics,
    loadedFromApi: catalogResult.ok && agentsResult.ok && grantsResult.ok && tracesResult.ok,
    grantsLoadedFromApi: grantsResult.ok,
    apiBase,
  }
}
