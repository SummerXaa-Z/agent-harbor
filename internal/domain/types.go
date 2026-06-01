package domain

import (
	"encoding/json"
	"time"
)

type AgentStatus string

const (
	AgentStatusDraft    AgentStatus = "draft"
	AgentStatusActive   AgentStatus = "active"
	AgentStatusDisabled AgentStatus = "disabled"
)

type Agent struct {
	ID                string            `json:"id"`
	TenantID          string            `json:"tenantId"`
	WorkspaceID       string            `json:"workspaceId"`
	Name              string            `json:"name"`
	Description       string            `json:"description,omitempty"`
	OwnerID           string            `json:"ownerId,omitempty"`
	ChannelType       string            `json:"channelType"`
	ChannelConfig     map[string]any    `json:"channelConfig,omitempty"`
	Credentials       map[string]string `json:"-"`
	CredentialVersion int               `json:"credentialVersion"`
	Status            AgentStatus       `json:"status"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
}

type CreateAgentRequest struct {
	TenantID      string            `json:"tenantId"`
	WorkspaceID   string            `json:"workspaceId"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	OwnerID       string            `json:"ownerId"`
	ChannelType   string            `json:"channelType"`
	ChannelConfig map[string]any    `json:"channelConfig"`
	Credentials   map[string]string `json:"credentials"`
	Status        AgentStatus       `json:"status"`
}

type UpdateAgentRequest struct {
	Name          *string         `json:"name"`
	Description   *string         `json:"description"`
	OwnerID       *string         `json:"ownerId"`
	ChannelConfig *map[string]any `json:"channelConfig"`
	Status        *AgentStatus    `json:"status"`
}

type RotateAgentCredentialsRequest struct {
	Credentials map[string]string `json:"credentials"`
}

type AuditEvent struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenantId"`
	WorkspaceID  string         `json:"workspaceId"`
	Actor        string         `json:"actor"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId"`
	Summary      string         `json:"summary,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

type AgentKey struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agentId"`
	Name      string    `json:"name"`
	Hash      string    `json:"-"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	RevokedAt time.Time `json:"revokedAt,omitempty,omitzero"`
}

type CreateAgentKeyRequest struct {
	AgentID          string `json:"agentId"`
	Name             string `json:"name"`
	ExpiresInSeconds int64  `json:"expiresInSeconds"`
}

type CreateAgentKeyResponse struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agentId"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AccessGrant struct {
	ID        string    `json:"id"`
	CallerID  string    `json:"callerAgentId"`
	TargetID  string    `json:"targetAgentId"`
	RouteType string    `json:"routeType,omitempty"`
	RouteKey  string    `json:"routeKey,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty,omitzero"`
	RevokedAt time.Time `json:"revokedAt,omitempty,omitzero"`
}

type CreateAccessGrantRequest struct {
	CallerID  string `json:"callerAgentId"`
	TargetID  string `json:"targetAgentId"`
	RouteType string `json:"routeType"`
	RouteKey  string `json:"routeKey"`
}

type RoutePolicyEffect string

const (
	RoutePolicyEffectAllow RoutePolicyEffect = "allow"
	RoutePolicyEffectDeny  RoutePolicyEffect = "deny"
)

type RoutePolicyStatus string

const (
	RoutePolicyStatusEnabled  RoutePolicyStatus = "enabled"
	RoutePolicyStatusDisabled RoutePolicyStatus = "disabled"
)

type RoutePolicy struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenantId"`
	WorkspaceID string            `json:"workspaceId"`
	Name        string            `json:"name"`
	CallerID    string            `json:"callerAgentId"`
	TargetID    string            `json:"targetAgentId"`
	RouteType   string            `json:"routeType"`
	RouteKey    string            `json:"routeKey,omitempty"`
	Effect      RoutePolicyEffect `json:"effect"`
	Status      RoutePolicyStatus `json:"status"`
	Priority    int               `json:"priority"`
	Retry       *RoutePolicyRetry `json:"retry,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type RoutePolicyRetry struct {
	MaxAttempts int   `json:"maxAttempts"`
	BackoffMs   int   `json:"backoffMs"`
	StatusCodes []int `json:"statusCodes"`
}

type RoutePolicyRetryRequest struct {
	MaxAttempts *int  `json:"maxAttempts"`
	BackoffMs   *int  `json:"backoffMs"`
	StatusCodes []int `json:"statusCodes"`
}

type CreateRoutePolicyRequest struct {
	Name      string                   `json:"name"`
	CallerID  string                   `json:"callerAgentId"`
	TargetID  string                   `json:"targetAgentId"`
	RouteType string                   `json:"routeType"`
	RouteKey  string                   `json:"routeKey"`
	Effect    RoutePolicyEffect        `json:"effect"`
	Status    RoutePolicyStatus        `json:"status"`
	Priority  *int                     `json:"priority"`
	Retry     *RoutePolicyRetryRequest `json:"retry"`
}

type UpdateRoutePolicyRequest struct {
	Name      *string            `json:"name"`
	RouteType *string            `json:"routeType"`
	RouteKey  *string            `json:"routeKey"`
	Effect    *RoutePolicyEffect `json:"effect"`
	Status    *RoutePolicyStatus `json:"status"`
	Priority  *int               `json:"priority"`
	Retry     json.RawMessage    `json:"retry"`
}

type RouteAccessDecision struct {
	Allowed  bool              `json:"allowed"`
	Source   string            `json:"source"`
	PolicyID string            `json:"policyId,omitempty"`
	Reason   string            `json:"reason"`
	Retry    *RoutePolicyRetry `json:"retry,omitempty"`
}

type CapabilityType string

const (
	CapabilityTypeMCPTool   CapabilityType = "mcp_tool"
	CapabilityTypeMCPMethod CapabilityType = "mcp_method"
)

type CapabilityAction string

const (
	CapabilityActionRead    CapabilityAction = "read"
	CapabilityActionWrite   CapabilityAction = "write"
	CapabilityActionDelete  CapabilityAction = "delete"
	CapabilityActionExecute CapabilityAction = "execute"
	CapabilityActionExport  CapabilityAction = "export"
	CapabilityActionAdmin   CapabilityAction = "admin"
)

type CapabilitySensitivity string

const (
	CapabilitySensitivityPublic       CapabilitySensitivity = "public"
	CapabilitySensitivityInternal     CapabilitySensitivity = "internal"
	CapabilitySensitivityConfidential CapabilitySensitivity = "confidential"
	CapabilitySensitivityRestricted   CapabilitySensitivity = "restricted"
)

type CapabilityRisk string

const (
	CapabilityRiskLow      CapabilityRisk = "low"
	CapabilityRiskMedium   CapabilityRisk = "medium"
	CapabilityRiskHigh     CapabilityRisk = "high"
	CapabilityRiskCritical CapabilityRisk = "critical"
)

type CapabilityEnforcementMode string

const (
	CapabilityEnforcementGateway          CapabilityEnforcementMode = "gateway"
	CapabilityEnforcementContextForwarded CapabilityEnforcementMode = "context_forwarded"
	CapabilityEnforcementDownstreamNative CapabilityEnforcementMode = "downstream_native"
	CapabilityEnforcementAdvisory         CapabilityEnforcementMode = "advisory"
)

type CapabilityDiscoveryStatus string

const (
	CapabilityDiscoveryPendingReview CapabilityDiscoveryStatus = "pending_review"
	CapabilityDiscoveryApproved      CapabilityDiscoveryStatus = "approved"
	CapabilityDiscoveryDeprecated    CapabilityDiscoveryStatus = "deprecated"
	CapabilityDiscoveryRemoved       CapabilityDiscoveryStatus = "removed"
)

type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny  PolicyEffect = "deny"
)

type PolicyStatus string

const (
	PolicyStatusEnabled  PolicyStatus = "enabled"
	PolicyStatusDisabled PolicyStatus = "disabled"
)

type DataScope struct {
	DataDomain     string `json:"dataDomain,omitempty"`
	Dataset        string `json:"dataset,omitempty"`
	Schema         string `json:"schema,omitempty"`
	Table          string `json:"table,omitempty"`
	Field          string `json:"field,omitempty"`
	Classification string `json:"classification,omitempty"`
	Region         string `json:"region,omitempty"`
	TenantFilter   string `json:"tenantFilter,omitempty"`
	MaskingPolicy  string `json:"maskingPolicy,omitempty"`
	RowFilter      string `json:"rowFilter,omitempty"`
}

type Capability struct {
	ID              string                    `json:"id"`
	TargetID        string                    `json:"targetId"`
	Type            CapabilityType            `json:"type"`
	Key             string                    `json:"key"`
	DisplayName     string                    `json:"displayName"`
	Description     string                    `json:"description,omitempty"`
	Action          CapabilityAction          `json:"action"`
	InputSchema     map[string]any            `json:"inputSchema,omitempty"`
	OutputSchema    map[string]any            `json:"outputSchema,omitempty"`
	NativeScopes    []string                  `json:"nativeScopes,omitempty"`
	DataDomains     []string                  `json:"dataDomains,omitempty"`
	DataScopes      []DataScope               `json:"dataScopes,omitempty"`
	Sensitivity     CapabilitySensitivity     `json:"sensitivity"`
	RiskLevel       CapabilityRisk            `json:"riskLevel"`
	EnforcementMode CapabilityEnforcementMode `json:"enforcementMode"`
	DiscoveryStatus CapabilityDiscoveryStatus `json:"discoveryStatus"`
	Version         int                       `json:"version"`
	DiscoveredAt    time.Time                 `json:"discoveredAt"`
	UpdatedAt       time.Time                 `json:"updatedAt"`
}

type UpdateCapabilityRequest struct {
	DiscoveryStatus *CapabilityDiscoveryStatus `json:"discoveryStatus"`
	Sensitivity     *CapabilitySensitivity     `json:"sensitivity"`
	RiskLevel       *CapabilityRisk            `json:"riskLevel"`
	DataScopes      []DataScope                `json:"dataScopes"`
}

type TenantEntitlement struct {
	ID           string       `json:"id"`
	TenantID     string       `json:"tenantId"`
	TargetID     string       `json:"targetId"`
	CapabilityID string       `json:"capabilityId"`
	Effect       PolicyEffect `json:"effect"`
	DataScopes   []DataScope  `json:"dataScopes,omitempty"`
	Status       PolicyStatus `json:"status"`
	Priority     int          `json:"priority"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type CreateTenantEntitlementRequest struct {
	TenantID     string       `json:"tenantId"`
	TargetID     string       `json:"targetId"`
	CapabilityID string       `json:"capabilityId"`
	Effect       PolicyEffect `json:"effect"`
	DataScopes   []DataScope  `json:"dataScopes"`
	Status       PolicyStatus `json:"status"`
	Priority     *int         `json:"priority"`
}

type WorkspaceAssignment struct {
	ID                  string       `json:"id"`
	TenantEntitlementID string       `json:"tenantEntitlementId"`
	TenantID            string       `json:"tenantId"`
	WorkspaceID         string       `json:"workspaceId"`
	Effect              PolicyEffect `json:"effect"`
	DataScopes          []DataScope  `json:"dataScopes,omitempty"`
	Status              PolicyStatus `json:"status"`
	CreatedAt           time.Time    `json:"createdAt"`
	UpdatedAt           time.Time    `json:"updatedAt"`
}

type CreateWorkspaceAssignmentRequest struct {
	TenantEntitlementID string       `json:"tenantEntitlementId"`
	WorkspaceID         string       `json:"workspaceId"`
	Effect              PolicyEffect `json:"effect"`
	DataScopes          []DataScope  `json:"dataScopes"`
	Status              PolicyStatus `json:"status"`
}

type InstanceAssignment struct {
	ID                    string       `json:"id"`
	WorkspaceAssignmentID string       `json:"workspaceAssignmentId"`
	TenantID              string       `json:"tenantId"`
	WorkspaceID           string       `json:"workspaceId"`
	CallerInstanceID      string       `json:"callerInstanceId"`
	SubjectSelector       string       `json:"subjectSelector,omitempty"`
	Effect                PolicyEffect `json:"effect"`
	DataScopes            []DataScope  `json:"dataScopes,omitempty"`
	Status                PolicyStatus `json:"status"`
	CreatedAt             time.Time    `json:"createdAt"`
	UpdatedAt             time.Time    `json:"updatedAt"`
}

type CreateInstanceAssignmentRequest struct {
	WorkspaceAssignmentID string       `json:"workspaceAssignmentId"`
	CallerInstanceID      string       `json:"callerInstanceId"`
	SubjectSelector       string       `json:"subjectSelector"`
	Effect                PolicyEffect `json:"effect"`
	DataScopes            []DataScope  `json:"dataScopes"`
	Status                PolicyStatus `json:"status"`
}

type CapabilityAccessDecision struct {
	Allowed               bool        `json:"allowed"`
	Source                string      `json:"source"`
	CapabilityID          string      `json:"capabilityId,omitempty"`
	EntitlementID         string      `json:"entitlementId,omitempty"`
	WorkspaceAssignmentID string      `json:"workspaceAssignmentId,omitempty"`
	InstanceAssignmentID  string      `json:"instanceAssignmentId,omitempty"`
	Reason                string      `json:"reason"`
	DataScopes            []DataScope `json:"dataScopes,omitempty"`
}

type TraceDecision string

const (
	TraceDecisionAllowed TraceDecision = "allowed"
	TraceDecisionDenied  TraceDecision = "denied"
)

type TraceEvent struct {
	ID                    string        `json:"id"`
	RunID                 string        `json:"runId,omitempty"`
	CallerID              string        `json:"callerAgentId,omitempty"`
	TargetID              string        `json:"targetAgentId"`
	RouteType             string        `json:"routeType"`
	RouteKey              string        `json:"routeKey,omitempty"`
	TenantID              string        `json:"tenantId,omitempty"`
	WorkspaceID           string        `json:"workspaceId,omitempty"`
	CallerInstanceID      string        `json:"callerInstanceId,omitempty"`
	SubjectID             string        `json:"subjectId,omitempty"`
	CapabilityID          string        `json:"capabilityId,omitempty"`
	CapabilityVersion     int           `json:"capabilityVersion,omitempty"`
	EntitlementID         string        `json:"entitlementId,omitempty"`
	WorkspaceAssignmentID string        `json:"workspaceAssignmentId,omitempty"`
	InstanceAssignmentID  string        `json:"instanceAssignmentId,omitempty"`
	DataScopes            []DataScope   `json:"dataScopes,omitempty"`
	Decision              TraceDecision `json:"decision"`
	Reason                string        `json:"reason,omitempty"`
	DurationMs            int64         `json:"durationMs,omitempty"`
	UpstreamAttempts      int           `json:"upstreamAttempts,omitempty"`
	UpstreamStatus        int           `json:"upstreamStatus,omitempty"`
	UpstreamError         string        `json:"upstreamError,omitempty"`
	CreatedAt             time.Time     `json:"createdAt"`
}

type SystemMetric struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Value     int       `json:"value"`
	Unit      string    `json:"unit,omitempty"`
	Trend     string    `json:"trend"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type FieldContract struct {
	Key              string `json:"key"`
	Type             string `json:"type"`
	Required         bool   `json:"required"`
	OutboundURL      bool   `json:"outboundUrl,omitempty"`
	SecretDisallowed bool   `json:"secretDisallowed,omitempty"`
}

type ProviderContract struct {
	SchemaVersion        string          `json:"schemaVersion"`
	Key                  string          `json:"key"`
	Label                string          `json:"label"`
	ChannelType          string          `json:"channelType"`
	DefaultEndpoint      string          `json:"defaultEndpoint,omitempty"`
	ChannelConfigFields  []FieldContract `json:"channelConfigFields"`
	RequiredCreds        []string        `json:"requiredCreds,omitempty"`
	OptionalCreds        []string        `json:"optionalCreds,omitempty"`
	FutureMetadataPolicy string          `json:"futureMetadataPolicy"`
}

type ChannelContract struct {
	SchemaVersion              string          `json:"schemaVersion"`
	Key                        string          `json:"key"`
	Label                      string          `json:"label"`
	EndpointRequiredWhenActive bool            `json:"endpointRequiredWhenActive"`
	ChannelConfigFields        []FieldContract `json:"channelConfigFields"`
	FutureMetadataPolicy       string          `json:"futureMetadataPolicy"`
}
