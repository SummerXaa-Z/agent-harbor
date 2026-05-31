package domain

import "time"

type AgentStatus string

const (
	AgentStatusDraft    AgentStatus = "draft"
	AgentStatusActive   AgentStatus = "active"
	AgentStatusDisabled AgentStatus = "disabled"
)

type Agent struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenantId"`
	WorkspaceID   string            `json:"workspaceId"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	OwnerID       string            `json:"ownerId,omitempty"`
	ChannelType   string            `json:"channelType"`
	ChannelConfig map[string]any    `json:"channelConfig,omitempty"`
	Credentials   map[string]string `json:"-"`
	Status        AgentStatus       `json:"status"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
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

type TraceDecision string

const (
	TraceDecisionAllowed TraceDecision = "allowed"
	TraceDecisionDenied  TraceDecision = "denied"
)

type TraceEvent struct {
	ID        string        `json:"id"`
	RunID     string        `json:"runId,omitempty"`
	CallerID  string        `json:"callerAgentId,omitempty"`
	TargetID  string        `json:"targetAgentId"`
	RouteType string        `json:"routeType"`
	RouteKey  string        `json:"routeKey,omitempty"`
	Decision  TraceDecision `json:"decision"`
	Reason    string        `json:"reason,omitempty"`
	CreatedAt time.Time     `json:"createdAt"`
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
