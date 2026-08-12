package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type AgentStatus string

const (
	AgentStatusDraft    AgentStatus = "draft"
	AgentStatusActive   AgentStatus = "active"
	AgentStatusDisabled AgentStatus = "disabled"
)

type TenantStatus string

const (
	TenantStatusActive   TenantStatus = "active"
	TenantStatusDisabled TenantStatus = "disabled"
)

type Tenant struct {
	ID             string       `json:"id"`
	ParentTenantID string       `json:"parentTenantId,omitempty"`
	Level          int          `json:"level"`
	Name           string       `json:"name"`
	Status         TenantStatus `json:"status"`
	CreatedAt      time.Time    `json:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

type CreateTenantRequest struct {
	ID             string       `json:"id"`
	ParentTenantID string       `json:"parentTenantId"`
	Name           string       `json:"name"`
	Status         TenantStatus `json:"status"`
}

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

type AdminIdentityRole string

const (
	AdminIdentityRolePlatformAdmin    AdminIdentityRole = "platform_admin"
	AdminIdentityRoleTenantAdmin      AdminIdentityRole = "tenant_admin"
	AdminIdentityRoleSecurityReviewer AdminIdentityRole = "security_reviewer"
)

type AdminIdentityStatus string

const (
	AdminIdentityStatusActive   AdminIdentityStatus = "active"
	AdminIdentityStatusDisabled AdminIdentityStatus = "disabled"
)

type AdminIdentitySource string

const (
	AdminIdentitySourceBootstrap AdminIdentitySource = "bootstrap"
	AdminIdentitySourceManaged   AdminIdentitySource = "managed"
)

type AdminIdentity struct {
	ID          string              `json:"id"`
	Actor       string              `json:"actor"`
	DisplayName string              `json:"displayName"`
	Role        AdminIdentityRole   `json:"role"`
	TenantID    string              `json:"tenantId,omitempty"`
	WorkspaceID string              `json:"workspaceId,omitempty"`
	Status      AdminIdentityStatus `json:"status"`
	Source      AdminIdentitySource `json:"source"`
	KeyHash     string              `json:"-"`
	KeyPrefix   string              `json:"keyPrefix,omitempty"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	LastUsedAt  time.Time           `json:"lastUsedAt,omitempty,omitzero"`
	RotatedAt   time.Time           `json:"rotatedAt,omitempty,omitzero"`
	DisabledAt  time.Time           `json:"disabledAt,omitempty,omitzero"`
	CreatedBy   string              `json:"createdBy,omitempty"`
	UpdatedBy   string              `json:"updatedBy,omitempty"`
	DisabledBy  string              `json:"disabledBy,omitempty"`
}

type CreateAdminIdentityRequest struct {
	Actor       string            `json:"actor"`
	DisplayName string            `json:"displayName"`
	Role        AdminIdentityRole `json:"role"`
	TenantID    string            `json:"tenantId"`
	WorkspaceID string            `json:"workspaceId"`
}

type CreateAdminIdentityResponse struct {
	Identity AdminIdentity `json:"identity"`
	Key      string        `json:"key"`
}

type RotateAdminIdentityKeyResponse struct {
	Identity AdminIdentity `json:"identity"`
	Key      string        `json:"key"`
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

func IsUnboundedSubjectSelector(subjectSelector string) bool {
	selector := strings.TrimSpace(subjectSelector)
	return selector == "" || selector == "*"
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

// CapabilityFingerprint is a canonical, server-computed witness for the
// capability properties that affect a later authorization decision. It is
// deliberately independent of Capability.Version: older mutation paths may
// not have treated Version as a security revision.
func CapabilityFingerprint(capability Capability) string {
	payload := struct {
		ID              string                    `json:"id"`
		TargetID        string                    `json:"targetId"`
		Type            CapabilityType            `json:"type"`
		Key             string                    `json:"key"`
		Action          CapabilityAction          `json:"action"`
		NativeScopes    []string                  `json:"nativeScopes,omitempty"`
		DataDomains     []string                  `json:"dataDomains,omitempty"`
		DataScopes      []DataScope               `json:"dataScopes,omitempty"`
		Sensitivity     CapabilitySensitivity     `json:"sensitivity"`
		RiskLevel       CapabilityRisk            `json:"riskLevel"`
		EnforcementMode CapabilityEnforcementMode `json:"enforcementMode"`
		DiscoveryStatus CapabilityDiscoveryStatus `json:"discoveryStatus"`
		Version         int                       `json:"version"`
	}{
		ID:              capability.ID,
		TargetID:        capability.TargetID,
		Type:            capability.Type,
		Key:             capability.Key,
		Action:          capability.Action,
		NativeScopes:    sortedStringCopy(capability.NativeScopes),
		DataDomains:     sortedStringCopy(capability.DataDomains),
		DataScopes:      sortedDataScopes(capability.DataScopes),
		Sensitivity:     capability.Sensitivity,
		RiskLevel:       capability.RiskLevel,
		EnforcementMode: capability.EnforcementMode,
		DiscoveryStatus: capability.DiscoveryStatus,
		Version:         capability.Version,
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return capability.ID + ":" + hex.EncodeToString(sum[:])
}

func sortedStringCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func sortedDataScopes(values []DataScope) []DataScope {
	out := append([]DataScope(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		left, _ := json.Marshal(out[i])
		right, _ := json.Marshal(out[j])
		return string(left) < string(right)
	})
	return out
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

type PermissionPackageDecision string

const (
	PermissionPackageDecisionAllow PermissionPackageDecision = "allow"
	PermissionPackageDecisionDeny  PermissionPackageDecision = "deny"
)

type PermissionPackagePolicyDecision string

const (
	PermissionPackagePolicyDecisionAllow            PermissionPackagePolicyDecision = "allow"
	PermissionPackagePolicyDecisionApprovalRequired PermissionPackagePolicyDecision = "approval_required"
)

type PermissionPackageApprovalStatus string

const (
	PermissionPackageApprovalStatusPending   PermissionPackageApprovalStatus = "pending"
	PermissionPackageApprovalStatusApproved  PermissionPackageApprovalStatus = "approved"
	PermissionPackageApprovalStatusRejected  PermissionPackageApprovalStatus = "rejected"
	PermissionPackageApprovalStatusWithdrawn PermissionPackageApprovalStatus = "withdrawn"
)

type PermissionPackageTemplate struct {
	ID                   string                           `json:"id"`
	Version              int                              `json:"version"`
	Name                 string                           `json:"name"`
	Summary              string                           `json:"summary"`
	AllowedActions       []CapabilityAction               `json:"allowedActions"`
	BlockedActions       []CapabilityAction               `json:"blockedActions"`
	BlockedRisks         []CapabilityRisk                 `json:"blockedRisks"`
	BlockedSensitivities []CapabilitySensitivity          `json:"blockedSensitivities"`
	DefaultDataDomain    string                           `json:"defaultDataDomain"`
	Guardrails           []PermissionPackageSimulationRow `json:"guardrails"`
}

type PermissionPackageAccessSubject struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	LabelKey        string `json:"labelKey"`
	DetailKey       string `json:"detailKey"`
	SubjectSelector string `json:"subjectSelector"`
	TenantID        string `json:"tenantId,omitempty"`
	WorkspaceID     string `json:"workspaceId,omitempty"`
	Email           string `json:"email,omitempty"`
	Status          string `json:"status,omitempty"`
}

type PermissionPackageDraftRequest struct {
	CallerInstanceID string `json:"callerInstanceId"`
	Region           string `json:"region"`
	RequestText      string `json:"requestText"`
	SubjectSelector  string `json:"subjectSelector"`
	TargetID         string `json:"targetId"`
	TemplateID       string `json:"templateId"`
	TenantID         string `json:"tenantId"`
	WorkspaceID      string `json:"workspaceId"`
}

type PermissionPackageApplyRequest struct {
	PermissionPackageDraftRequest
	ApprovalRequestID string `json:"approvalRequestId,omitempty"`
}

type PermissionPackageDraft struct {
	ID                  string                           `json:"id"`
	Input               PermissionPackageDraftRequest    `json:"input"`
	Template            PermissionPackageTemplate        `json:"template"`
	AllowedCapabilities []Capability                     `json:"allowedCapabilities"`
	BlockedCapabilities []Capability                     `json:"blockedCapabilities"`
	DataScopes          []DataScope                      `json:"dataScopes"`
	Readiness           PermissionPackageReadiness       `json:"readiness"`
	PolicyGate          PermissionPackagePolicyGate      `json:"policyGate"`
	SimulationRows      []PermissionPackageSimulationRow `json:"simulationRows"`
}

type PermissionPackageReadiness struct {
	CanApply      bool     `json:"canApply"`
	MissingFields []string `json:"missingFields"`
	Warnings      []string `json:"warnings"`
}

type PermissionPackagePolicyGate struct {
	Decision         PermissionPackagePolicyDecision         `json:"decision"`
	CanApplyDirectly bool                                    `json:"canApplyDirectly"`
	PolicyVersion    int                                     `json:"policyVersion"`
	Reasons          []PermissionPackagePolicyReason         `json:"reasons"`
	NextActionCodes  []PermissionPackagePolicyNextActionCode `json:"nextActionCodes"`
	NextActions      []string                                `json:"nextActions"`
}

type PermissionPackagePolicyNextActionCode string

const (
	PermissionPackagePolicyNextCreateApproval PermissionPackagePolicyNextActionCode = "create_approval_request"
)

type PermissionPackagePolicyReason struct {
	ID            string            `json:"id"`
	CapabilityID  string            `json:"capabilityId,omitempty"`
	CapabilityKey string            `json:"capabilityKey,omitempty"`
	Severity      string            `json:"severity"`
	Message       string            `json:"message"`
	ReasonKey     string            `json:"reasonKey,omitempty"`
	ReasonValues  map[string]string `json:"reasonValues,omitempty"`
}

type PermissionPackageSimulationRow struct {
	ID               string                    `json:"id"`
	CapabilityID     string                    `json:"capabilityId,omitempty"`
	CapabilityKey    string                    `json:"capabilityKey"`
	ExpectedDecision PermissionPackageDecision `json:"expectedDecision"`
	Reason           string                    `json:"reason"`
	ReasonKey        string                    `json:"reasonKey,omitempty"`
	ReasonValues     map[string]string         `json:"reasonValues,omitempty"`
}

type PermissionPackageApplyResponse struct {
	Draft                PermissionPackageDraft        `json:"draft"`
	TenantEntitlements   []TenantEntitlement           `json:"tenantEntitlements"`
	WorkspaceAssignments []WorkspaceAssignment         `json:"workspaceAssignments"`
	InstanceAssignments  []InstanceAssignment          `json:"instanceAssignments"`
	Application          *PermissionPackageApplication `json:"application,omitempty"`
}

type PermissionPackagePreflightSeverity string

const (
	PermissionPackagePreflightPassed   PermissionPackagePreflightSeverity = "passed"
	PermissionPackagePreflightInfo     PermissionPackagePreflightSeverity = "info"
	PermissionPackagePreflightWarning  PermissionPackagePreflightSeverity = "warning"
	PermissionPackagePreflightBlocking PermissionPackagePreflightSeverity = "blocking"
)

type PermissionPackageApplyPreflightResponse struct {
	Draft           PermissionPackageDraft                         `json:"draft"`
	Summary         PermissionPackageApplyPreflightSummary         `json:"summary"`
	Checks          []PermissionPackageApplyPreflightCheck         `json:"checks"`
	Planned         PermissionPackageApplyPreflightPlannedChanges  `json:"planned"`
	ExistingGrants  []PermissionPackageApplyPreflightExistingGrant `json:"existingGrants"`
	NextActionCodes []PermissionPackagePreflightNextActionCode     `json:"nextActionCodes"`
	NextActions     []string                                       `json:"nextActions"`
}

type PermissionPackagePreflightNextActionCode string

const (
	PermissionPackagePreflightNextFixDraftReadiness        PermissionPackagePreflightNextActionCode = "fix_draft_readiness"
	PermissionPackagePreflightNextCreateApproval           PermissionPackagePreflightNextActionCode = "create_approval_request"
	PermissionPackagePreflightNextUseApprovedRequest       PermissionPackagePreflightNextActionCode = "use_approved_request"
	PermissionPackagePreflightNextRefreshApproval          PermissionPackagePreflightNextActionCode = "refresh_approval_request"
	PermissionPackagePreflightNextNarrowDataScope          PermissionPackagePreflightNextActionCode = "narrow_data_scope"
	PermissionPackagePreflightNextReviewExistingGrants     PermissionPackagePreflightNextActionCode = "review_existing_grants"
	PermissionPackagePreflightNextReviewCurrentApplication PermissionPackagePreflightNextActionCode = "review_current_application"
	PermissionPackagePreflightNextApplyPermissionPackage   PermissionPackagePreflightNextActionCode = "apply_permission_package"
)

type PermissionPackageApplyPreflightSummary struct {
	CanApply                        bool `json:"canApply"`
	BlockingCount                   int  `json:"blockingCount"`
	WarningCount                    int  `json:"warningCount"`
	PlannedCapabilityCount          int  `json:"plannedCapabilityCount"`
	PlannedTenantEntitlementCount   int  `json:"plannedTenantEntitlementCount"`
	PlannedWorkspaceAssignmentCount int  `json:"plannedWorkspaceAssignmentCount"`
	PlannedInstanceAssignmentCount  int  `json:"plannedInstanceAssignmentCount"`
	ExistingGrantCount              int  `json:"existingGrantCount"`
	RequiresApproval                bool `json:"requiresApproval"`
	ApprovalReady                   bool `json:"approvalReady"`
}

type PermissionPackageApplyPreflightCheck struct {
	Code          string                             `json:"code"`
	Severity      PermissionPackagePreflightSeverity `json:"severity"`
	Message       string                             `json:"message"`
	CapabilityID  string                             `json:"capabilityId,omitempty"`
	CapabilityKey string                             `json:"capabilityKey,omitempty"`
}

type PermissionPackageApplyPreflightPlannedChanges struct {
	Capabilities         []Capability          `json:"capabilities"`
	TenantEntitlements   []TenantEntitlement   `json:"tenantEntitlements"`
	WorkspaceAssignments []WorkspaceAssignment `json:"workspaceAssignments"`
	InstanceAssignments  []InstanceAssignment  `json:"instanceAssignments"`
}

type PermissionPackageApplyPreflightExistingGrant struct {
	CapabilityID          string `json:"capabilityId"`
	CapabilityKey         string `json:"capabilityKey"`
	TenantEntitlementID   string `json:"tenantEntitlementId"`
	WorkspaceAssignmentID string `json:"workspaceAssignmentId"`
	InstanceAssignmentID  string `json:"instanceAssignmentId"`
}

type PermissionPackageApprovalRequest struct {
	ID                            string                          `json:"id"`
	DraftID                       string                          `json:"draftId"`
	TemplateID                    string                          `json:"templateId"`
	TemplateVersion               int                             `json:"templateVersion"`
	PolicyVersion                 int                             `json:"policyVersion"`
	TenantID                      string                          `json:"tenantId"`
	WorkspaceID                   string                          `json:"workspaceId"`
	TargetID                      string                          `json:"targetId"`
	CallerInstanceID              string                          `json:"callerInstanceId"`
	SubjectSelector               string                          `json:"subjectSelector,omitempty"`
	RequestText                   string                          `json:"requestText,omitempty"`
	Region                        string                          `json:"region,omitempty"`
	DataScopes                    []DataScope                     `json:"dataScopes,omitempty"`
	AllowedCapabilityIDs          []string                        `json:"allowedCapabilityIds"`
	AllowedCapabilityKeys         []string                        `json:"allowedCapabilityKeys"`
	AllowedCapabilityFingerprints []string                        `json:"allowedCapabilityFingerprints"`
	PolicyGate                    PermissionPackagePolicyGate     `json:"policyGate"`
	Status                        PermissionPackageApprovalStatus `json:"status"`
	RequestedBy                   string                          `json:"requestedBy,omitempty"`
	ReviewedBy                    string                          `json:"reviewedBy,omitempty"`
	ReviewComment                 string                          `json:"reviewComment,omitempty"`
	CreatedAt                     time.Time                       `json:"createdAt"`
	UpdatedAt                     time.Time                       `json:"updatedAt"`
	ResolvedAt                    time.Time                       `json:"resolvedAt,omitempty,omitzero"`
	ExpiresAt                     time.Time                       `json:"expiresAt"`
	ConsumedAt                    time.Time                       `json:"consumedAt,omitempty,omitzero"`
	ConsumedByApplicationID       string                          `json:"consumedByApplicationId,omitempty"`
}

type PermissionPackageApprovalReviewer struct {
	Reviewer    string `json:"reviewer"`
	TenantID    string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
}

type PermissionPackageApprovalResolutionRequest struct {
	Reviewer string `json:"reviewer"`
	Comment  string `json:"comment"`
}

type PermissionPackageApplication struct {
	ID                     string      `json:"id"`
	DraftID                string      `json:"draftId"`
	TemplateID             string      `json:"templateId"`
	TemplateVersion        int         `json:"templateVersion"`
	TenantID               string      `json:"tenantId"`
	WorkspaceID            string      `json:"workspaceId"`
	TargetID               string      `json:"targetId"`
	CallerInstanceID       string      `json:"callerInstanceId"`
	SubjectSelector        string      `json:"subjectSelector,omitempty"`
	RequestText            string      `json:"requestText,omitempty"`
	Region                 string      `json:"region,omitempty"`
	DataScopes             []DataScope `json:"dataScopes,omitempty"`
	AllowedCapabilityIDs   []string    `json:"allowedCapabilityIds"`
	AllowedCapabilityKeys  []string    `json:"allowedCapabilityKeys"`
	TenantEntitlementIDs   []string    `json:"tenantEntitlementIds"`
	WorkspaceAssignmentIDs []string    `json:"workspaceAssignmentIds"`
	InstanceAssignmentIDs  []string    `json:"instanceAssignmentIds"`
	AppliedAt              time.Time   `json:"appliedAt"`
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
	CapabilityFingerprint string        `json:"capabilityFingerprint,omitempty"`
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

// TaskResultSummaryVerification records the explicit human attestation that a
// task result summary was reviewed and redacted before persistence. It is kept
// deliberately narrow: this first memory slice does not accept preferences,
// policies, or permission conclusions.
type TaskResultSummaryVerification string

const (
	TaskResultSummaryVerificationHumanReviewedRedacted TaskResultSummaryVerification = "human_reviewed_redacted"
)

type VerifiedTaskResultSummary struct {
	ID               string                        `json:"id"`
	TenantID         string                        `json:"tenantId"`
	WorkspaceID      string                        `json:"workspaceId"`
	CallerInstanceID string                        `json:"callerInstanceId"`
	SubjectID        string                        `json:"subjectId"`
	TargetID         string                        `json:"targetId"`
	CapabilityID     string                        `json:"capabilityId"`
	SourceTraceID    string                        `json:"sourceTraceId"`
	DataScopes       []DataScope                   `json:"dataScopes"`
	Summary          string                        `json:"summary"`
	PayloadDigest    string                        `json:"payloadDigest"`
	Verification     TaskResultSummaryVerification `json:"verification"`
	VerifiedBy       string                        `json:"verifiedBy"`
	VerifiedAt       time.Time                     `json:"verifiedAt"`
	CreatedAt        time.Time                     `json:"createdAt"`
	ExpiresAt        time.Time                     `json:"expiresAt"`
}

// CreateVerifiedTaskResultSummaryRequest intentionally does not accept a
// free-text summary, scope, target, capability, or provenance fields other
// than sourceTraceId. The stored summary and all scope fields are generated
// server-side from the immutable runtime trace.
type CreateVerifiedTaskResultSummaryRequest struct {
	MemoryKind    string                        `json:"memoryKind,omitempty"`
	SourceTraceID string                        `json:"sourceTraceId"`
	Verification  TaskResultSummaryVerification `json:"verification"`
	ExpiresAt     time.Time                     `json:"expiresAt"`
}

type TaskResultSummaryGateDecision string

const (
	TaskResultSummaryGateAllowed          TaskResultSummaryGateDecision = "allowed"
	TaskResultSummaryGateDenied           TaskResultSummaryGateDecision = "denied"
	TaskResultSummaryGateApprovalRequired TaskResultSummaryGateDecision = "approval_required"
)

// TaskResultSummaryGateResult is deliberately safe to return when a request
// is denied: it has no memory, source trace, digest, or submitted summary.
type TaskResultSummaryGateResult struct {
	Decision       TaskResultSummaryGateDecision `json:"decision"`
	ReasonCode     string                        `json:"reasonCode"`
	NextActionCode string                        `json:"nextActionCode"`
}

type TaskResultSummaryReadResponse struct {
	TaskResultSummaryGateResult
	Memory *VerifiedTaskResultSummary `json:"memory,omitempty"`
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
