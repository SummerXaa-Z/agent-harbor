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
  Agent,
  ApiEnvelope,
  CatalogData,
  ChannelContract,
  ConsoleData,
  ProviderContract,
  TraceEvent,
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

function isEnvelope<T>(value: unknown): value is ApiEnvelope<T> {
  return Boolean(value && typeof value === 'object' && 'code' in value)
}

async function request<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(endpoint(path), {
    headers: { Accept: 'application/json' },
    signal,
  })

  const payload = (await response.json()) as unknown
  if (!response.ok) {
    const message = isEnvelope<T>(payload) ? payload.message || payload.error : response.statusText
    throw new Error(message || `Request failed with status ${response.status}`)
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
  } catch {
    return { data: fallback, ok: false }
  }
}

export async function fetchProviders(signal?: AbortSignal): Promise<ProviderContract[]> {
  return request<ProviderContract[]>('/api/v1/contracts/providers', signal)
}

export async function fetchChannels(signal?: AbortSignal): Promise<ChannelContract[]> {
  return request<ChannelContract[]>('/api/v1/contracts/channels', signal)
}

export async function fetchCatalog(signal?: AbortSignal): Promise<CatalogData> {
  const [providers, channels] = await Promise.all([fetchProviders(signal), fetchChannels(signal)])
  return { providers, channels }
}

export async function fetchAgents(workspaceId?: string, signal?: AbortSignal): Promise<Agent[]> {
  const query = workspaceId ? `?workspaceId=${encodeURIComponent(workspaceId)}` : ''
  return request<Agent[]>(`/api/v1/agents${query}`, signal)
}

export async function fetchTraces(runId?: string, signal?: AbortSignal): Promise<TraceEvent[]> {
  const query = runId ? `?runId=${encodeURIComponent(runId)}` : ''
  return request<TraceEvent[]>(`/api/v1/audit/traces${query}`, signal)
}

export async function loadConsoleData(): Promise<ConsoleData> {
  const [catalogResult, agentsResult, tracesResult] = await Promise.all([
    withFallback(() => fetchCatalog(), {
      providers: sampleProviders,
      channels: sampleChannels,
    }),
    withFallback(() => fetchAgents(), sampleAgents),
    withFallback(() => fetchTraces(), sampleTraces),
  ])

  return {
    providers: catalogResult.data.providers,
    channels: catalogResult.data.channels,
    agents: agentsResult.data,
    traces: tracesResult.data,
    routePolicies,
    evidenceRuns,
    systemMetrics,
    loadedFromApi: catalogResult.ok && agentsResult.ok && tracesResult.ok,
    apiBase,
  }
}
