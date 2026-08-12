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

export type TenantStatus = 'active' | 'disabled'

export interface Tenant {
  id: string
  parentTenantId?: string
  level: number
  name: string
  status: TenantStatus
  createdAt: string
  updatedAt: string
}

export interface CreateTenantRequest {
  id: string
  parentTenantId?: string
  name: string
  status?: TenantStatus
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

export type RoutePolicyEffect = 'allow' | 'deny'
export type RoutePolicyStatus = 'enabled' | 'disabled'
export type CapabilityType = 'mcp_tool' | 'mcp_method'
export type CapabilityAction = 'read' | 'write' | 'delete' | 'execute' | 'export' | 'admin'
export type CapabilitySensitivity = 'public' | 'internal' | 'confidential' | 'restricted'
export type CapabilityRisk = 'low' | 'medium' | 'high' | 'critical'
export type CapabilityEnforcementMode = 'gateway' | 'context_forwarded' | 'downstream_native' | 'advisory'
export type CapabilityDiscoveryStatus = 'pending_review' | 'approved' | 'deprecated' | 'removed'

export interface CreateRoutePolicyRequest {
  name?: string
  callerAgentId: string
  targetAgentId: string
  routeType: string
  routeKey?: string
  effect?: RoutePolicyEffect
  status?: RoutePolicyStatus
  priority?: number
  retry?: RoutePolicyRetry
}

export interface RoutePolicyRetry {
  maxAttempts: number
  backoffMs: number
  statusCodes: number[]
}

export interface TraceEvent {
  id: string
  runId?: string
  callerAgentId?: string
  targetAgentId: string
  routeType: string
  routeKey?: string
  tenantId?: string
  workspaceId?: string
  callerInstanceId?: string
  subjectId?: string
  capabilityId?: string
  capabilityVersion?: number
  entitlementId?: string
  workspaceAssignmentId?: string
  instanceAssignmentId?: string
  dataScopes?: DataScope[]
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

export interface ConsoleSession {
  actor?: string
  authenticated: boolean
  csrfToken?: string
  expiresAt?: string
  role?: string
  requiresLogin: boolean
  tenantId?: string
  workspaceId?: string
}

export interface RoutePolicy {
  id: string
  tenantId: string
  workspaceId: string
  name: string
  callerAgentId: string
  targetAgentId: string
  routeType: string
  routeKey?: string
  effect: RoutePolicyEffect
  status: RoutePolicyStatus
  priority: number
  retry?: RoutePolicyRetry
  lastMatchedAt?: string
  createdAt: string
  updatedAt: string
}

export interface DataScope {
  dataDomain?: string
  dataset?: string
  schema?: string
  table?: string
  field?: string
  classification?: string
  region?: string
  tenantFilter?: string
  maskingPolicy?: string
  rowFilter?: string
}

export interface Capability {
  id: string
  targetId: string
  type: CapabilityType
  key: string
  displayName: string
  description?: string
  action: CapabilityAction
  inputSchema?: JsonObject
  outputSchema?: JsonObject
  nativeScopes?: string[]
  dataDomains?: string[]
  dataScopes?: DataScope[]
  sensitivity: CapabilitySensitivity
  riskLevel: CapabilityRisk
  enforcementMode: CapabilityEnforcementMode
  discoveryStatus: CapabilityDiscoveryStatus
  version: number
  discoveredAt: string
  updatedAt: string
}

export interface UpdateCapabilityRequest {
  discoveryStatus?: CapabilityDiscoveryStatus
  sensitivity?: CapabilitySensitivity
  riskLevel?: CapabilityRisk
  dataDomains?: string[]
  dataScopes?: DataScope[]
}

export interface TenantEntitlement {
  id: string
  tenantId: string
  targetId: string
  capabilityId: string
  effect: RoutePolicyEffect
  dataScopes?: DataScope[]
  status: RoutePolicyStatus
  priority: number
  createdAt: string
  updatedAt: string
}

export interface CreateTenantEntitlementRequest {
  tenantId: string
  targetId: string
  capabilityId: string
  effect?: RoutePolicyEffect
  dataScopes?: DataScope[]
  status?: RoutePolicyStatus
  priority?: number
}

export interface WorkspaceAssignment {
  id: string
  tenantEntitlementId: string
  tenantId: string
  workspaceId: string
  effect: RoutePolicyEffect
  dataScopes?: DataScope[]
  status: RoutePolicyStatus
  createdAt: string
  updatedAt: string
}

export interface CreateWorkspaceAssignmentRequest {
  tenantEntitlementId: string
  workspaceId: string
  effect?: RoutePolicyEffect
  dataScopes?: DataScope[]
  status?: RoutePolicyStatus
}

export interface InstanceAssignment {
  id: string
  workspaceAssignmentId: string
  tenantId: string
  workspaceId: string
  callerInstanceId: string
  subjectSelector?: string
  effect: RoutePolicyEffect
  dataScopes?: DataScope[]
  status: RoutePolicyStatus
  createdAt: string
  updatedAt: string
}

export interface CreateInstanceAssignmentRequest {
  workspaceAssignmentId: string
  callerInstanceId: string
  subjectSelector: string
  effect?: RoutePolicyEffect
  dataScopes?: DataScope[]
  status?: RoutePolicyStatus
}

export interface AcceptanceRun {
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
  tenants: Tenant[]
  providers: ProviderContract[]
  channels: ChannelContract[]
  agents: Agent[]
  accessGrants: AccessGrant[]
  capabilities: Capability[]
  tenantEntitlements: TenantEntitlement[]
  workspaceAssignments: WorkspaceAssignment[]
  instanceAssignments: InstanceAssignment[]
  traces: TraceEvent[]
  auditEvents: AuditEvent[]
  routePolicies: RoutePolicy[]
  evidenceRuns: AcceptanceRun[]
  systemMetrics: SystemMetric[]
  loadedFromApi: boolean
  setupLoadedFromApi: boolean
  grantsLoadedFromApi: boolean
  capabilitiesLoadedFromApi: boolean
  capabilityAssignmentsLoadedFromApi: boolean
  routePoliciesLoadedFromApi: boolean
  apiBase: string
}

export type AccessProfileScopeStatus = 'valid' | 'invalid'

export interface AccessProfileFilters {
  workspaceId?: string
  targetId?: string
  capabilityId?: string
  callerInstanceId?: string
  subjectId?: string
  traceLimit?: number | string
}

export interface AccessProfileHandoffContext {
  tenantId: string
  tenantName: string
  tenantPath?: string
  workspaceId: string
  workspaceName: string
  callerInstanceId?: string
  callerName?: string
  targetId?: string
  targetName?: string
  capabilityId?: string
  capabilityName?: string
}

export interface PermissionChangeHandoffContext {
  tenantId: string
  tenantName?: string
  workspaceId: string
  workspaceName?: string
  callerInstanceId?: string
  callerName?: string
  targetId?: string
  targetName?: string
  capabilityId?: string
  capabilityName?: string
  brokenLayer?: string
  decisionReason?: string
  decisionSource?: string
  subjectId?: string
  templateId?: string
  intentText?: string
  sourceView: 'ask' | 'tenants' | 'registry'
}

export interface CapabilityGovernanceHandoffContext {
  tenantId: string
  tenantName?: string
  workspaceId: string
  workspaceName?: string
  targetId: string
  targetName?: string
  capabilityId?: string
  sourceView: 'registry' | 'ask'
}

export interface AskHandoffContext {
  tenantId?: string
  workspaceId?: string
  callerInstanceId?: string
  targetId?: string
  capabilityId?: string
  subjectId?: string
  sourceView: 'registry' | 'capabilities' | 'access' | 'ai-admin'
}

export interface AccessProfileSummary {
  tenantCount: number
  grantCount: number
  targetCount: number
  capabilityCount: number
  workspaceAssignmentCount: number
  instanceAssignmentCount: number
  recentAllowedTraceCount: number
  recentDeniedTraceCount: number
}

export interface TenantAccessProfileInstance {
  instanceAssignment: InstanceAssignment
  callerInstance?: Agent
  effectiveInstanceDataScopes?: DataScope[]
  scopeStatus: AccessProfileScopeStatus
  scopeReason?: string
}

export interface TenantAccessProfileWorkspace {
  workspaceAssignment: WorkspaceAssignment
  effectiveWorkspaceDataScopes?: DataScope[]
  scopeStatus: AccessProfileScopeStatus
  scopeReason?: string
  instanceAssignments: TenantAccessProfileInstance[]
}

export interface TenantAccessProfileGrant {
  tenantEntitlement: TenantEntitlement
  target?: Agent
  capability?: Capability
  effectiveTenantDataScopes?: DataScope[]
  scopeStatus: AccessProfileScopeStatus
  scopeReason?: string
  workspaceAssignments: TenantAccessProfileWorkspace[]
}

export interface TenantAccessProfile {
  tenant: Tenant
  scopeTenants: Tenant[]
  summary: AccessProfileSummary
  grants: TenantAccessProfileGrant[]
  recentTraces: TraceEvent[]
  generatedAt: string
}

export interface TenantAccessProfileData extends TenantAccessProfile {
  loadedFromApi: boolean
  apiBase: string
}

export interface AccessDecisionExplainRequest {
  tenantId: string
  workspaceId: string
  callerInstanceId: string
  targetId: string
  capabilityId: string
  subjectId?: string
}

export interface CapabilityAccessDecision {
  allowed: boolean
  source: string
  capabilityId?: string
  entitlementId?: string
  workspaceAssignmentId?: string
  instanceAssignmentId?: string
  reason: string
  dataScopes?: DataScope[]
}

export interface AccessDecisionExplainEvidence {
  layer: string
  status: string
  id?: string
  message: string
}

export interface AccessDecisionExplainResult {
  outcome: 'allowed' | 'denied'
  summary: string
  request: AccessDecisionExplainRequest
  decision: CapabilityAccessDecision
  evidence: AccessDecisionExplainEvidence[]
  dataScopes?: DataScope[]
  nextActionCodes?: string[]
  nextActions: string[]
}

export interface TraceFilters {
  runId?: string
  decision?: TraceDecision | ''
  callerAgentId?: string
  targetAgentId?: string
}

export interface McpRpcCallResult {
  ok: boolean
  payload: unknown
  status: number
}

export type AdminIdentityRole = 'platform_admin' | 'tenant_admin' | 'security_reviewer'
export type AdminIdentityStatus = 'active' | 'disabled'
export type AdminIdentitySource = 'bootstrap' | 'managed'

export interface AdminIdentity {
  id: string
  actor: string
  displayName: string
  role: AdminIdentityRole
  tenantId?: string
  workspaceId?: string
  status: AdminIdentityStatus
  source: AdminIdentitySource
  keyPrefix?: string
  createdAt: string
  updatedAt: string
  lastUsedAt?: string
  rotatedAt?: string
  disabledAt?: string
  createdBy?: string
  updatedBy?: string
  disabledBy?: string
}

export interface CreateAdminIdentityRequest {
  actor: string
  displayName?: string
  role: AdminIdentityRole
  tenantId?: string
  workspaceId?: string
}

export interface CreateAdminIdentityResponse {
  identity: AdminIdentity
  key: string
}

export interface RotateAdminIdentityKeyResponse {
  identity: AdminIdentity
  key: string
}

export type TenantPermissionCenterStatus = 'ready' | 'needs_review' | 'blocked'
export type TenantPermissionCenterActionTarget = 'ai-admin' | 'access' | 'admin-access' | 'getting-started'

export interface TenantPermissionCenterOperatorBoundary {
  actor: string
  role: AdminIdentityRole
  tenantId?: string
  workspaceId?: string
  canManageAdministrators: boolean
}

export interface TenantPermissionCenterAdministrator {
  id: string
  actor: string
  displayName: string
  role: AdminIdentityRole
  tenantId?: string
  workspaceId?: string
  status: AdminIdentityStatus
  source: AdminIdentitySource
}

export interface TenantPermissionCenterWorkspace {
  workspaceId: string
  callerCount: number
  targetCount: number
  assignmentCount: number
}

export interface TenantPermissionCenterPackage {
  templateId: string
  templateName: string
  status: TenantPermissionCenterStatus
  allowedCapabilityCount: number
  blockedCapabilityCount: number
  dataScopes?: DataScope[]
  latestApplicationId?: string
}

export interface TenantPermissionCenterCapability {
  targetId: string
  targetName: string
  capabilityId: string
  capabilityName: string
  effect: RoutePolicyEffect
  dataScopes?: DataScope[]
  workspaceIds: string[]
}

export interface TenantPermissionCenterNextAction {
  code: string
  targetView: TenantPermissionCenterActionTarget
}

export interface TenantPermissionCenterResponse {
  tenant: Tenant
  scopeTenants: Tenant[]
  operatorBoundary: TenantPermissionCenterOperatorBoundary
  administrators: TenantPermissionCenterAdministrator[]
  workspaces: TenantPermissionCenterWorkspace[]
  permissionPackages: TenantPermissionCenterPackage[]
  capabilities: TenantPermissionCenterCapability[]
  nextActions: TenantPermissionCenterNextAction[]
  generatedAt: string
}
