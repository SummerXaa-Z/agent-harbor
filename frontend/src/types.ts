export type JsonPrimitive = string | number | boolean | null
export type JsonValue = JsonPrimitive | JsonValue[] | { [key: string]: JsonValue }
export type JsonObject = { [key: string]: JsonValue }

export type AgentStatus = 'draft' | 'active' | 'disabled'
export type TraceDecision = 'allowed' | 'denied'

export interface Agent {
  id: string
  tenantId: string
  workspaceId: string
  name: string
  description?: string
  ownerId?: string
  channelType: string
  channelConfig?: JsonObject
  credentialVersion: number
  status: AgentStatus
  createdAt: string
  updatedAt: string
}

export interface CreateAgentRequest {
  tenantId?: string
  workspaceId: string
  name: string
  description?: string
  ownerId?: string
  channelType?: string
  channelConfig?: JsonObject
  credentials?: Record<string, string>
  status?: AgentStatus
}

export interface UpdateAgentRequest {
  name?: string
  description?: string
  ownerId?: string
  channelConfig?: JsonObject
  status?: AgentStatus
}

export interface RotateAgentCredentialsRequest {
  credentials: Record<string, string>
}

export interface ManagementScope {
  tenantId: string
  workspaceId: string
}

export interface AgentKey {
  id: string
  agentId: string
  name: string
  prefix: string
  createdAt: string
  expiresAt: string
  revokedAt?: string
}

export interface CreateAgentKeyRequest {
  agentId: string
  name?: string
  expiresInSeconds?: number
}

export interface CreateAgentKeyResponse extends AgentKey {
  key: string
}

export interface AccessGrant {
  id: string
  callerAgentId: string
  targetAgentId: string
  routeType?: string
  routeKey?: string
  createdAt: string
  expiresAt?: string
  revokedAt?: string
}

export interface CreateAccessGrantRequest {
  callerAgentId: string
  targetAgentId: string
  routeType?: string
  routeKey?: string
}

export interface TraceEvent {
  id: string
  runId?: string
  callerAgentId?: string
  targetAgentId: string
  routeType: string
  routeKey?: string
  decision: TraceDecision
  reason?: string
  durationMs?: number
  upstreamAttempts?: number
  upstreamStatus?: number
  upstreamError?: string
  createdAt: string
}

export interface AuditEvent {
  id: string
  tenantId: string
  workspaceId: string
  actor: string
  action: string
  resourceType: string
  resourceId: string
  summary?: string
  metadata?: Record<string, JsonValue>
  createdAt: string
}

export interface FieldContract {
  key: string
  type: string
  required: boolean
  outboundUrl?: boolean
  secretDisallowed?: boolean
}

export interface ProviderContract {
  schemaVersion: string
  key: string
  label: string
  channelType: string
  defaultEndpoint?: string
  channelConfigFields: FieldContract[]
  requiredCreds?: string[]
  optionalCreds?: string[]
  futureMetadataPolicy: string
}

export interface ChannelContract {
  schemaVersion: string
  key: string
  label: string
  endpointRequiredWhenActive: boolean
  channelConfigFields: FieldContract[]
  futureMetadataPolicy: string
}

export interface ApiEnvelope<T> {
  code: number
  data?: T
  error?: string
  message?: string
}

export interface RoutePolicy {
  id: string
  name: string
  callerAgentId: string
  targetAgentId: string
  routeType: string
  routeKey?: string
  effect: 'allow' | 'deny'
  status: 'enabled' | 'disabled'
  priority: number
  lastMatchedAt?: string
  createdAt: string
}

export interface EvidenceRun {
  id: string
  runId: string
  title: string
  agentId: string
  decision: TraceDecision
  status: 'passed' | 'warning' | 'failed'
  checks: number
  startedAt: string
  completedAt?: string
  summary: string
}

export interface SystemMetric {
  id: string
  label: string
  value: number
  unit?: string
  trend: 'up' | 'down' | 'flat'
  status: 'healthy' | 'warning' | 'critical'
  updatedAt: string
}

export interface CatalogData {
  providers: ProviderContract[]
  channels: ChannelContract[]
}

export interface ConsoleData {
  providers: ProviderContract[]
  channels: ChannelContract[]
  agents: Agent[]
  accessGrants: AccessGrant[]
  traces: TraceEvent[]
  auditEvents: AuditEvent[]
  routePolicies: RoutePolicy[]
  evidenceRuns: EvidenceRun[]
  systemMetrics: SystemMetric[]
  loadedFromApi: boolean
  grantsLoadedFromApi: boolean
  apiBase: string
}

export interface TraceFilters {
  runId?: string
  decision?: TraceDecision | ''
  callerAgentId?: string
  targetAgentId?: string
}
