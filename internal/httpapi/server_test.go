package httpapi_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
}

type systemInfoResponse struct {
	Name                     string   `json:"name"`
	APIVersion               string   `json:"apiVersion"`
	AuthRequired             bool     `json:"authRequired"`
	Capabilities             []string `json:"capabilities"`
	ManagementMcpToolCatalog struct {
		MetadataVersion             int      `json:"metadataVersion"`
		RequiredMetadata            []string `json:"requiredMetadata"`
		CatalogDigest               string   `json:"catalogDigest"`
		ToolCount                   int      `json:"toolCount"`
		ConfirmationRequiredTools   int      `json:"confirmationRequiredTools"`
		ToolsWithConfirmationSchema int      `json:"toolsWithConfirmationSchema"`
	} `json:"managementMcpToolCatalog"`
}

type agentResponse struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenantId"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	OwnerID           string         `json:"ownerId"`
	WorkspaceID       string         `json:"workspaceId"`
	ChannelType       string         `json:"channelType"`
	ChannelConfig     map[string]any `json:"channelConfig"`
	CredentialVersion int            `json:"credentialVersion"`
	Status            string         `json:"status"`
}

type keyResponse struct {
	ID      string `json:"id"`
	AgentID string `json:"agentId"`
	Key     string `json:"key"`
	Prefix  string `json:"prefix"`
}

type traceResponse struct {
	ID                    string `json:"id"`
	CallerID              string `json:"callerAgentId"`
	TargetID              string `json:"targetAgentId"`
	RouteKey              string `json:"routeKey"`
	TenantID              string `json:"tenantId"`
	WorkspaceID           string `json:"workspaceId"`
	CallerInstanceID      string `json:"callerInstanceId"`
	CapabilityID          string `json:"capabilityId"`
	CapabilityVersion     int    `json:"capabilityVersion"`
	EntitlementID         string `json:"entitlementId"`
	WorkspaceAssignmentID string `json:"workspaceAssignmentId"`
	InstanceAssignmentID  string `json:"instanceAssignmentId"`
	Decision              string `json:"decision"`
	Reason                string `json:"reason"`
	RunID                 string `json:"runId"`
	DurationMs            int64  `json:"durationMs"`
	UpstreamAttempts      int    `json:"upstreamAttempts"`
	UpstreamStatus        int    `json:"upstreamStatus"`
	UpstreamError         string `json:"upstreamError"`
}

type auditEventResponse struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenantId"`
	WorkspaceID  string         `json:"workspaceId"`
	Actor        string         `json:"actor"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId"`
	Summary      string         `json:"summary"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    string         `json:"createdAt"`
}

type adminIdentityResponse struct {
	ID            string         `json:"id"`
	Actor         string         `json:"actor"`
	DisplayName   string         `json:"displayName"`
	Role          string         `json:"role"`
	TenantID      string         `json:"tenantId"`
	WorkspaceID   string         `json:"workspaceId"`
	Status        string         `json:"status"`
	Source        string         `json:"source"`
	KeyPrefix     string         `json:"keyPrefix"`
	CreatedBy     string         `json:"createdBy"`
	UpdatedBy     string         `json:"updatedBy"`
	OwnedAgentIDs []string       `json:"ownedAgentIds,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type createAdminIdentityResponse struct {
	Identity adminIdentityResponse `json:"identity"`
	Key      string                `json:"key"`
}

type rotateAdminIdentityKeyResponse struct {
	Identity adminIdentityResponse `json:"identity"`
	Key      string                `json:"key"`
}

type grantResponse struct {
	ID        string `json:"id"`
	CallerID  string `json:"callerAgentId"`
	TargetID  string `json:"targetAgentId"`
	RouteType string `json:"routeType"`
	RouteKey  string `json:"routeKey"`
}

type routePolicyResponse struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	CallerID    string `json:"callerAgentId"`
	TargetID    string `json:"targetAgentId"`
	RouteType   string `json:"routeType"`
	RouteKey    string `json:"routeKey"`
	Effect      string `json:"effect"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	Retry       *struct {
		MaxAttempts int   `json:"maxAttempts"`
		BackoffMs   int   `json:"backoffMs"`
		StatusCodes []int `json:"statusCodes"`
	} `json:"retry"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type metricResponse struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Value     int    `json:"value"`
	Unit      string `json:"unit"`
	Trend     string `json:"trend"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
}

type capabilityResponse struct {
	ID              string              `json:"id"`
	TargetID        string              `json:"targetId"`
	Type            string              `json:"type"`
	Key             string              `json:"key"`
	DisplayName     string              `json:"displayName"`
	Description     string              `json:"description"`
	Action          string              `json:"action"`
	DataDomains     []string            `json:"dataDomains"`
	DataScopes      []dataScopeResponse `json:"dataScopes"`
	Sensitivity     string              `json:"sensitivity"`
	RiskLevel       string              `json:"riskLevel"`
	EnforcementMode string              `json:"enforcementMode"`
	DiscoveryStatus string              `json:"discoveryStatus"`
	Version         int                 `json:"version"`
}

type dataScopeResponse struct {
	DataDomain   string `json:"dataDomain"`
	Region       string `json:"region"`
	TenantFilter string `json:"tenantFilter"`
}

type permissionPackageTemplateResponse struct {
	ID             string   `json:"id"`
	Version        int      `json:"version"`
	Name           string   `json:"name"`
	AllowedActions []string `json:"allowedActions"`
	BlockedActions []string `json:"blockedActions"`
}

type permissionPackageDraftResponse struct {
	ID                  string                            `json:"id"`
	Template            permissionPackageTemplateResponse `json:"template"`
	AllowedCapabilities []capabilityResponse              `json:"allowedCapabilities"`
	BlockedCapabilities []capabilityResponse              `json:"blockedCapabilities"`
	DataScopes          []dataScopeResponse               `json:"dataScopes"`
	Readiness           struct {
		CanApply bool     `json:"canApply"`
		Warnings []string `json:"warnings"`
	} `json:"readiness"`
	PolicyGate     permissionPackagePolicyGateResponse `json:"policyGate"`
	SimulationRows []struct {
		CapabilityKey    string `json:"capabilityKey"`
		ExpectedDecision string `json:"expectedDecision"`
		ReasonKey        string `json:"reasonKey"`
	} `json:"simulationRows"`
}

type permissionPackagePolicyGateResponse struct {
	Decision         string `json:"decision"`
	CanApplyDirectly bool   `json:"canApplyDirectly"`
	PolicyVersion    int    `json:"policyVersion"`
	Reasons          []struct {
		ID            string            `json:"id"`
		CapabilityID  string            `json:"capabilityId"`
		CapabilityKey string            `json:"capabilityKey"`
		Severity      string            `json:"severity"`
		Message       string            `json:"message"`
		ReasonKey     string            `json:"reasonKey"`
		ReasonValues  map[string]string `json:"reasonValues"`
	} `json:"reasons"`
	NextActionCodes []string `json:"nextActionCodes"`
	NextActions     []string `json:"nextActions"`
}

type permissionPackageApplyResponse struct {
	Draft                permissionPackageDraftResponse        `json:"draft"`
	TenantEntitlements   []tenantEntitlementResponse           `json:"tenantEntitlements"`
	WorkspaceAssignments []workspaceAssignmentResponse         `json:"workspaceAssignments"`
	InstanceAssignments  []instanceAssignmentResponse          `json:"instanceAssignments"`
	Application          *permissionPackageApplicationResponse `json:"application"`
}

type permissionPackageApplyPreflightResponse struct {
	Draft           permissionPackageDraftResponse                 `json:"draft"`
	Summary         permissionPackageApplyPreflightSummary         `json:"summary"`
	Checks          []permissionPackageApplyPreflightCheck         `json:"checks"`
	Planned         permissionPackageApplyPreflightPlannedChanges  `json:"planned"`
	ExistingGrants  []permissionPackageApplyPreflightExistingGrant `json:"existingGrants"`
	NextActionCodes []string                                       `json:"nextActionCodes"`
	NextActions     []string                                       `json:"nextActions"`
}

type permissionPackageApplyPreflightSummary struct {
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

type permissionPackageApplyPreflightCheck struct {
	Code          string `json:"code"`
	Severity      string `json:"severity"`
	Message       string `json:"message"`
	CapabilityID  string `json:"capabilityId"`
	CapabilityKey string `json:"capabilityKey"`
}

type permissionPackageApplyPreflightPlannedChanges struct {
	Capabilities         []capabilityResponse          `json:"capabilities"`
	TenantEntitlements   []tenantEntitlementResponse   `json:"tenantEntitlements"`
	WorkspaceAssignments []workspaceAssignmentResponse `json:"workspaceAssignments"`
	InstanceAssignments  []instanceAssignmentResponse  `json:"instanceAssignments"`
}

type permissionPackageApplyPreflightExistingGrant struct {
	CapabilityID          string `json:"capabilityId"`
	CapabilityKey         string `json:"capabilityKey"`
	TenantEntitlementID   string `json:"tenantEntitlementId"`
	WorkspaceAssignmentID string `json:"workspaceAssignmentId"`
	InstanceAssignmentID  string `json:"instanceAssignmentId"`
}

type permissionPackageApplicationResponse struct {
	ID                     string              `json:"id"`
	DraftID                string              `json:"draftId"`
	TemplateID             string              `json:"templateId"`
	TemplateVersion        int                 `json:"templateVersion"`
	TenantID               string              `json:"tenantId"`
	WorkspaceID            string              `json:"workspaceId"`
	TargetID               string              `json:"targetId"`
	CallerInstanceID       string              `json:"callerInstanceId"`
	RequestedCapabilityID  string              `json:"requestedCapabilityId"`
	SubjectSelector        string              `json:"subjectSelector"`
	RequestText            string              `json:"requestText"`
	Region                 string              `json:"region"`
	DataScopes             []dataScopeResponse `json:"dataScopes"`
	AllowedCapabilityIDs   []string            `json:"allowedCapabilityIds"`
	AllowedCapabilityKeys  []string            `json:"allowedCapabilityKeys"`
	TenantEntitlementIDs   []string            `json:"tenantEntitlementIds"`
	WorkspaceAssignmentIDs []string            `json:"workspaceAssignmentIds"`
	InstanceAssignmentIDs  []string            `json:"instanceAssignmentIds"`
}

type permissionPackageApplicationImpactResponse struct {
	Application       permissionPackageApplicationResponse `json:"application"`
	Summary           permissionPackageImpactSummary       `json:"summary"`
	CreatedObjects    []permissionPackageImpactObject      `json:"createdObjects"`
	CapabilityReviews []permissionPackageImpactCapability  `json:"capabilityReviews"`
	RollbackReview    permissionPackageRollbackReview      `json:"rollbackReview"`
	RemediationPlan   permissionPackageRemediationPlan     `json:"remediationPlan"`
	Rehearsal         *permissionPackageImpactRehearsal    `json:"rehearsal"`
}

type permissionPackageImpactRehearsal struct {
	Enabled  bool   `json:"enabled"`
	Scenario string `json:"scenario"`
}

type permissionPackageApplicationHealthResponse struct {
	Summary      permissionPackageApplicationHealthSummary `json:"summary"`
	Applications []permissionPackageApplicationHealthRow   `json:"applications"`
}

type permissionPackageApplicationHealthSummary struct {
	Total       int `json:"total"`
	Ready       int `json:"ready"`
	Drifted     int `json:"drifted"`
	NeedsReview int `json:"needsReview"`
}

type permissionPackageApplicationHealthRow struct {
	Application        permissionPackageApplicationResponse `json:"application"`
	Status             string                               `json:"status"`
	BlockerCodes       []string                             `json:"blockerCodes"`
	CreatedObjectCount int                                  `json:"createdObjectCount"`
	ActiveObjectCount  int                                  `json:"activeObjectCount"`
	MissingObjectCount int                                  `json:"missingObjectCount"`
	RollbackReady      bool                                 `json:"rollbackReady"`
}

type permissionPackageProductionReadinessResponse struct {
	Status            string                                      `json:"status"`
	Summary           permissionPackageProductionReadinessSummary `json:"summary"`
	Checks            []permissionPackageProductionReadinessCheck `json:"checks"`
	LatestApplication *permissionPackageApplicationResponse       `json:"latestApplication"`
	Preflight         *permissionPackageApplyPreflightResponse    `json:"preflight"`
	RuntimeEvidence   permissionPackageRuntimeEvidence            `json:"runtimeEvidence"`
	AuditEvidence     permissionPackageAuditEvidence              `json:"auditEvidence"`
	NextActionCode    string                                      `json:"nextActionCode"`
	NextActions       []string                                    `json:"nextActions"`
}

type permissionPackageWorkbenchPreviewResponse struct {
	Draft               permissionPackageDraftResponse                `json:"draft"`
	ApprovalRequest     *permissionPackageApprovalRequestResponse     `json:"approvalRequest"`
	LatestApplication   *permissionPackageApplicationResponse         `json:"latestApplication"`
	ProductionReadiness *permissionPackageProductionReadinessResponse `json:"productionReadiness"`
	Summary             permissionPackageWorkbenchSummary             `json:"summary"`
}

type permissionPackageWorkbenchSummary struct {
	Status                 string                           `json:"status"`
	PrimaryActionCode      string                           `json:"primaryActionCode"`
	NextActionCode         string                           `json:"nextActionCode"`
	ApprovalRequired       bool                             `json:"approvalRequired"`
	CanApply               bool                             `json:"canApply"`
	Applied                bool                             `json:"applied"`
	RuntimeEvidenceReady   bool                             `json:"runtimeEvidenceReady"`
	ProductionReady        bool                             `json:"productionReady"`
	AllowedCapabilityCount int                              `json:"allowedCapabilityCount"`
	BlockedCapabilityCount int                              `json:"blockedCapabilityCount"`
	PlannedObjectCount     int                              `json:"plannedObjectCount"`
	ReadinessReadyCount    int                              `json:"readinessReadyCount"`
	ReadinessTotalCount    int                              `json:"readinessTotalCount"`
	BlockingCount          int                              `json:"blockingCount"`
	WarningCount           int                              `json:"warningCount"`
	Steps                  []permissionPackageWorkbenchStep `json:"steps"`
}

type permissionPackageWorkbenchStep struct {
	Key        string `json:"key"`
	Status     string `json:"status"`
	DetailCode string `json:"detailCode"`
	Count      int    `json:"count"`
	Total      int    `json:"total"`
}

type permissionPackageProductionReadinessSummary struct {
	ReadyCount         int  `json:"readyCount"`
	WarningCount       int  `json:"warningCount"`
	BlockingCount      int  `json:"blockingCount"`
	HasApplication     bool `json:"hasApplication"`
	HasAllowedTrace    bool `json:"hasAllowedTrace"`
	HasDeniedTrace     bool `json:"hasDeniedTrace"`
	HasAppliedAudit    bool `json:"hasAppliedAudit"`
	AccessProfileReady bool `json:"accessProfileReady"`
}

type permissionPackageProductionReadinessCheck struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	EvidenceID string `json:"evidenceId"`
}

type permissionPackageRuntimeEvidence struct {
	AllowedTrace *traceResponse `json:"allowedTrace"`
	DeniedTrace  *traceResponse `json:"deniedTrace"`
}

type permissionPackageAuditEvidence struct {
	AppliedEvent *auditEventResponse `json:"appliedEvent"`
}

type permissionPackageProductionEvidenceReportResponse struct {
	ReportVersion         string                                      `json:"reportVersion"`
	ReportDigest          string                                      `json:"reportDigest"`
	ReportDigestAlgorithm string                                      `json:"reportDigestAlgorithm"`
	GeneratedBy           string                                      `json:"generatedBy"`
	Status                string                                      `json:"status"`
	PlatformContract      permissionPackageProductionPlatformContract `json:"platformContract"`
	Scope                 permissionPackageProductionEvidenceScope    `json:"scope"`
	Summary               permissionPackageProductionReadinessSummary `json:"summary"`
	Checks                []permissionPackageProductionReadinessCheck `json:"checks"`
	Evidence              permissionPackageProductionEvidenceRefs     `json:"evidence"`
	NextActionCode        string                                      `json:"nextActionCode"`
	NextActions           []string                                    `json:"nextActions"`
	ReadinessGeneratedAt  string                                      `json:"readinessGeneratedAt"`
}

type permissionPackageProductionPlatformContract struct {
	APIVersion               string `json:"apiVersion"`
	ManagementMcpToolCatalog struct {
		MetadataVersion int    `json:"metadataVersion"`
		CatalogDigest   string `json:"catalogDigest"`
	} `json:"managementMcpToolCatalog"`
}

type permissionPackageProductionEvidenceScope struct {
	TenantID              string `json:"tenantId"`
	WorkspaceID           string `json:"workspaceId"`
	TemplateID            string `json:"templateId"`
	TargetID              string `json:"targetId"`
	CallerInstanceID      string `json:"callerInstanceId"`
	RequestedCapabilityID string `json:"requestedCapabilityId"`
	SubjectID             string `json:"subjectId"`
	Region                string `json:"region"`
	SubjectSelector       string `json:"subjectSelector"`
}

type permissionPackageProductionEvidenceRefs struct {
	Application       permissionPackageProductionApplicationEvidence `json:"application"`
	Runtime           permissionPackageProductionRuntimeEvidence     `json:"runtime"`
	Audit             permissionPackageProductionAuditEvidence       `json:"audit"`
	AccessProfile     permissionPackageProductionEvidenceState       `json:"accessProfile"`
	ApplicationHealth permissionPackageProductionEvidenceState       `json:"applicationHealth"`
	ApplicationImpact permissionPackageProductionEvidenceState       `json:"applicationImpact"`
}

type permissionPackageProductionApplicationEvidence struct {
	Present               bool     `json:"present"`
	ID                    string   `json:"id"`
	RequestedCapabilityID string   `json:"requestedCapabilityId"`
	TemplateVersion       int      `json:"templateVersion"`
	AllowedCapabilityIDs  []string `json:"allowedCapabilityIds"`
}

type permissionPackageProductionRuntimeEvidence struct {
	AllowedTraceID string `json:"allowedTraceId"`
	DeniedTraceID  string `json:"deniedTraceId"`
}

type permissionPackageProductionAuditEvidence struct {
	AppliedEventID string `json:"appliedEventId"`
}

type permissionPackageProductionEvidenceState struct {
	Present bool   `json:"present"`
	Status  string `json:"status"`
}

type permissionPackageImpactSummary struct {
	CreatedObjectCount int  `json:"createdObjectCount"`
	ActiveObjectCount  int  `json:"activeObjectCount"`
	MissingObjectCount int  `json:"missingObjectCount"`
	RollbackReady      bool `json:"rollbackReady"`
}

type permissionPackageImpactObject struct {
	ID             string              `json:"id"`
	Type           string              `json:"type"`
	CurrentStatus  string              `json:"currentStatus"`
	RollbackAction string              `json:"rollbackAction"`
	DataScopes     []dataScopeResponse `json:"dataScopes"`
}

type permissionPackageImpactCapability struct {
	ID             string `json:"id"`
	Key            string `json:"key"`
	CurrentStatus  string `json:"currentStatus"`
	RollbackAction string `json:"rollbackAction"`
}

type permissionPackageRollbackReview struct {
	Ready        bool     `json:"ready"`
	Blockers     []string `json:"blockers"`
	BlockerCodes []string `json:"blockerCodes"`
	Steps        []string `json:"steps"`
}

type permissionPackageRemediationPlan struct {
	ExecutionMode string                               `json:"executionMode"`
	Ready         bool                                 `json:"ready"`
	Blockers      []string                             `json:"blockers"`
	BlockerCodes  []string                             `json:"blockerCodes"`
	Actions       []permissionPackageRemediationAction `json:"actions"`
}

type permissionPackageRemediationAction struct {
	ID            string `json:"id"`
	Order         int    `json:"order"`
	TargetType    string `json:"targetType"`
	TargetID      string `json:"targetId"`
	Action        string `json:"action"`
	CurrentStatus string `json:"currentStatus"`
	Reason        string `json:"reason"`
	ReadOnly      bool   `json:"readOnly"`
}

type permissionPackageApprovalRequestResponse struct {
	ID                      string                              `json:"id"`
	DraftID                 string                              `json:"draftId"`
	TemplateID              string                              `json:"templateId"`
	TemplateVersion         int                                 `json:"templateVersion"`
	PolicyVersion           int                                 `json:"policyVersion"`
	TenantID                string                              `json:"tenantId"`
	WorkspaceID             string                              `json:"workspaceId"`
	TargetID                string                              `json:"targetId"`
	CallerInstanceID        string                              `json:"callerInstanceId"`
	RequestedCapabilityID   string                              `json:"requestedCapabilityId"`
	SubjectSelector         string                              `json:"subjectSelector"`
	RequestText             string                              `json:"requestText"`
	Region                  string                              `json:"region"`
	DataScopes              []dataScopeResponse                 `json:"dataScopes"`
	AllowedCapabilityIDs    []string                            `json:"allowedCapabilityIds"`
	AllowedCapabilityKeys   []string                            `json:"allowedCapabilityKeys"`
	PolicyGate              permissionPackagePolicyGateResponse `json:"policyGate"`
	Status                  string                              `json:"status"`
	EffectiveStatus         string                              `json:"effectiveStatus"`
	IsExpired               bool                                `json:"isExpired"`
	RequestedBy             string                              `json:"requestedBy"`
	ReviewedBy              string                              `json:"reviewedBy"`
	ReviewComment           string                              `json:"reviewComment"`
	CreatedAt               time.Time                           `json:"createdAt"`
	UpdatedAt               time.Time                           `json:"updatedAt"`
	ResolvedAt              time.Time                           `json:"resolvedAt"`
	ExpiresAt               time.Time                           `json:"expiresAt"`
	ConsumedAt              time.Time                           `json:"consumedAt"`
	ConsumedByApplicationID string                              `json:"consumedByApplicationId"`
}

type mcpEnvelopeResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      any              `json:"id"`
	Result  mcpResultPayload `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    *struct {
			AppCode    string `json:"appCode"`
			HTTPStatus int    `json:"httpStatus"`
		} `json:"data"`
	} `json:"error"`
}

type mcpResultPayload struct {
	MetadataVersion   int               `json:"metadataVersion"`
	CatalogDigest     string            `json:"catalogDigest"`
	Tools             []mcpToolResponse `json:"tools"`
	Content           []mcpContentItem  `json:"content"`
	StructuredContent json.RawMessage   `json:"structuredContent"`
}

type mcpToolResponse struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	InputSchema map[string]any   `json:"inputSchema"`
	Safety      mcpToolSafety    `json:"safety"`
	Access      mcpToolAccess    `json:"access"`
	Lifecycle   mcpToolLifecycle `json:"lifecycle"`
	Execution   mcpToolExecution `json:"execution"`
}

type mcpToolSafety struct {
	OperationType string `json:"operationType"`
	ReadOnly      bool   `json:"readOnly"`
	MutatesState  bool   `json:"mutatesState"`
	ApprovalMode  string `json:"approvalMode"`
}

type mcpToolAccess struct {
	RequiredRole  string `json:"requiredRole"`
	ScopeBoundary string `json:"scopeBoundary"`
	ReviewerBound bool   `json:"reviewerBound"`
}

type mcpToolLifecycle struct {
	Status        string `json:"status"`
	PreferredName string `json:"preferredName,omitempty"`
}

type mcpToolExecution struct {
	Idempotency          string `json:"idempotency"`
	ConfirmationRequired bool   `json:"confirmationRequired"`
	PreflightTool        string `json:"preflightTool,omitempty"`
	AuditResourceType    string `json:"auditResourceType,omitempty"`
	ReturnsSecret        bool   `json:"returnsSecret,omitempty"`
}

type mcpContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type managementMCPExplainPermissionPackageResponse struct {
	Outcome   string `json:"outcome"`
	Summary   string `json:"summary"`
	Readiness struct {
		CanApply bool     `json:"canApply"`
		Warnings []string `json:"warnings"`
	} `json:"readiness"`
	PolicyGate            permissionPackagePolicyGateResponse `json:"policyGate"`
	BlockedSimulationRows []struct {
		CapabilityKey    string `json:"capabilityKey"`
		ExpectedDecision string `json:"expectedDecision"`
	} `json:"blockedSimulationRows"`
	NextActionCodes []string `json:"nextActionCodes"`
	NextActions     []string `json:"nextActions"`
}

type managementMCPExplainAccessResponse struct {
	Outcome         string                          `json:"outcome"`
	Summary         string                          `json:"summary"`
	Decision        domain.CapabilityAccessDecision `json:"decision"`
	Evidence        []managementMCPExplainEvidence  `json:"evidence"`
	DataScopes      []domain.DataScope              `json:"dataScopes"`
	NextActionCodes []string                        `json:"nextActionCodes"`
	NextActions     []string                        `json:"nextActions"`
}

type managementMCPExplainEvidence struct {
	Layer   string `json:"layer"`
	Status  string `json:"status"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

type tenantResponse struct {
	ID             string `json:"id"`
	ParentTenantID string `json:"parentTenantId"`
	Level          int    `json:"level"`
	Name           string `json:"name"`
	Status         string `json:"status"`
}

type tenantPermissionCenterResponse struct {
	Tenant           tenantResponse                            `json:"tenant"`
	ScopeTenants     []tenantResponse                          `json:"scopeTenants"`
	OperatorBoundary tenantPermissionCenterOperatorBoundary    `json:"operatorBoundary"`
	Administrators   []tenantPermissionCenterAdministrator     `json:"administrators"`
	Workspaces       []tenantPermissionCenterWorkspace         `json:"workspaces"`
	PermissionPacks  []tenantPermissionCenterPermissionPackage `json:"permissionPackages"`
	Capabilities     []tenantPermissionCenterCapability        `json:"capabilities"`
	NextActions      []tenantPermissionCenterNextAction        `json:"nextActions"`
	GeneratedAt      string                                    `json:"generatedAt"`
}

type tenantPermissionCenterOperatorBoundary struct {
	Actor                   string `json:"actor"`
	Role                    string `json:"role"`
	TenantID                string `json:"tenantId,omitempty"`
	WorkspaceID             string `json:"workspaceId,omitempty"`
	CanManageAdministrators bool   `json:"canManageAdministrators"`
}

type tenantPermissionCenterAdministrator struct {
	ID          string `json:"id"`
	Actor       string `json:"actor"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	TenantID    string `json:"tenantId,omitempty"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	Status      string `json:"status"`
	Source      string `json:"source"`
}

type tenantPermissionCenterWorkspace struct {
	WorkspaceID     string `json:"workspaceId"`
	CallerCount     int    `json:"callerCount"`
	TargetCount     int    `json:"targetCount"`
	AssignmentCount int    `json:"assignmentCount"`
}

type tenantPermissionCenterPermissionPackage struct {
	TemplateID             string             `json:"templateId"`
	TemplateName           string             `json:"templateName"`
	Status                 string             `json:"status"`
	AllowedCapabilityCount int                `json:"allowedCapabilityCount"`
	BlockedCapabilityCount int                `json:"blockedCapabilityCount"`
	DataScopes             []domain.DataScope `json:"dataScopes,omitempty"`
	LatestApplicationID    string             `json:"latestApplicationId,omitempty"`
}

type tenantPermissionCenterCapability struct {
	TargetID       string             `json:"targetId"`
	TargetName     string             `json:"targetName"`
	CapabilityID   string             `json:"capabilityId"`
	CapabilityName string             `json:"capabilityName"`
	Effect         string             `json:"effect"`
	DataScopes     []domain.DataScope `json:"dataScopes,omitempty"`
	WorkspaceIDs   []string           `json:"workspaceIds"`
}

type tenantPermissionCenterNextAction struct {
	Code       string `json:"code"`
	TargetView string `json:"targetView"`
}

type tenantEntitlementResponse struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenantId"`
	TargetID     string `json:"targetId"`
	CapabilityID string `json:"capabilityId"`
	Effect       string `json:"effect"`
	Status       string `json:"status"`
	Priority     int    `json:"priority"`
}

type workspaceAssignmentResponse struct {
	ID                  string `json:"id"`
	TenantEntitlementID string `json:"tenantEntitlementId"`
	TenantID            string `json:"tenantId"`
	WorkspaceID         string `json:"workspaceId"`
	Effect              string `json:"effect"`
	Status              string `json:"status"`
}

type instanceAssignmentResponse struct {
	ID                    string `json:"id"`
	WorkspaceAssignmentID string `json:"workspaceAssignmentId"`
	TenantID              string `json:"tenantId"`
	WorkspaceID           string `json:"workspaceId"`
	CallerInstanceID      string `json:"callerInstanceId"`
	Effect                string `json:"effect"`
	Status                string `json:"status"`
}

func newRouter() http.Handler {
	return httpapi.New(store.NewMemory(), httpapi.WithUnauthenticatedAdminAllowed(true)).Router()
}

func newRouterWithAdmin(adminKey string) http.Handler {
	return httpapi.New(store.NewMemory(), httpapi.WithAdminKey(adminKey)).Router()
}

func newRouterWithRepo(repo store.Repository) http.Handler {
	return httpapi.New(repo, httpapi.WithUnauthenticatedAdminAllowed(true)).Router()
}

func newRouterWithRepoAndApprovalReviewers(repo store.Repository, reviewers []domain.PermissionPackageApprovalReviewer) http.Handler {
	return httpapi.New(
		repo,
		httpapi.WithUnauthenticatedAdminAllowed(true),
		httpapi.WithPermissionPackageApprovalReviewers(reviewers),
	).Router()
}

func newRouterWithRepoAndAdminIdentities(repo store.Repository, identities []httpapi.AdminIdentity) http.Handler {
	return httpapi.New(repo, httpapi.WithAdminIdentities(identities)).Router()
}

func newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo store.Repository, identities []httpapi.AdminIdentity, reviewers []domain.PermissionPackageApprovalReviewer) http.Handler {
	return httpapi.New(
		repo,
		httpapi.WithAdminIdentities(identities),
		httpapi.WithPermissionPackageApprovalReviewers(reviewers),
	).Router()
}

func newRouterWithPrivateUpstreams() http.Handler {
	return httpapi.New(
		store.NewMemory(),
		httpapi.WithUnauthenticatedAdminAllowed(true),
		httpapi.WithPrivateUpstreamsAllowed(true),
	).Router()
}

func newRouterWithCORSOrigins(origins []string) http.Handler {
	return httpapi.New(
		store.NewMemory(),
		httpapi.WithUnauthenticatedAdminAllowed(true),
		httpapi.WithCORSOrigins(origins),
	).Router()
}

func seedTenantPermissionCenterFixture(t *testing.T, repo store.Repository) (tenantID string, workspaceID string, caller domain.Agent, target domain.Agent, capability domain.Capability) {
	t.Helper()
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root-center", "", "Platform Operations", now)
	createDirectTenant(t, repo, "tenant-child-center", "tenant-root-center", "Customer Service", now)
	caller = createDirectAgent(t, repo, "Support Assistant", "tenant-child-center", "ws-support-center", "local", domain.AgentStatusActive, nil)
	target = createDirectAgent(t, repo, "Ticket Tool Service", "tenant-child-center", "ws-support-center", "mcp", domain.AgentStatusActive, nil)
	capability = createDirectCapabilityWithAction(t, repo, target.ID, "search_ticket", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	entitlement := createDirectTenantEntitlement(t, repo, "tenant-child-center", target.ID, capability.ID, []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)
	workspaceAssignment := createDirectWorkspaceAssignment(t, repo, entitlement.ID, "tenant-child-center", "ws-support-center", []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)
	createDirectInstanceAssignment(t, repo, workspaceAssignment.ID, "tenant-child-center", "ws-support-center", caller.ID, []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)
	return "tenant-child-center", "ws-support-center", caller, target, capability
}

func tenantCenterActionCodes(actions []tenantPermissionCenterNextAction) []string {
	codes := make([]string, 0, len(actions))
	for _, action := range actions {
		codes = append(codes, action.Code)
	}
	sort.Strings(codes)
	return codes
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func createAgent(t *testing.T, router http.Handler, body map[string]any) agentResponse {
	t.Helper()
	resp := request(t, router, http.MethodPost, "/api/v1/agents", body, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create agent failed: status=%d body=%s", resp.Code, resp.Body.String())
	}
	return decodeData[agentResponse](t, resp)
}

func createDirectAgent(t *testing.T, repo store.Repository, name string, tenantID string, workspaceID string, channelType string, status domain.AgentStatus, channelConfig map[string]any) domain.Agent {
	t.Helper()
	return createDirectAgentWithCredentials(t, repo, name, tenantID, workspaceID, channelType, status, channelConfig, nil)
}

func createDirectAgentWithCredentials(t *testing.T, repo store.Repository, name string, tenantID string, workspaceID string, channelType string, status domain.AgentStatus, channelConfig map[string]any, credentials map[string]string) domain.Agent {
	t.Helper()
	if channelConfig == nil {
		channelConfig = map[string]any{}
	}
	if credentials == nil {
		credentials = map[string]string{}
	}
	now := time.Now().UTC()
	agent := domain.Agent{
		ID:                security.NewID("agt"),
		TenantID:          tenantID,
		WorkspaceID:       workspaceID,
		Name:              name,
		ChannelType:       channelType,
		ChannelConfig:     channelConfig,
		Credentials:       credentials,
		CredentialVersion: credentialVersionForTest(credentials),
		Status:            status,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	created, err := repo.CreateAgent(t.Context(), agent)
	if err != nil {
		t.Fatalf("create direct agent: %v", err)
	}
	return created
}

func credentialVersionForTest(credentials map[string]string) int {
	if len(credentials) == 0 {
		return 0
	}
	return 1
}

func createLocalCallerWithKey(t *testing.T, router http.Handler, name string) (agentResponse, keyResponse) {
	t.Helper()
	caller := createAgent(t, router, map[string]any{
		"name":        name,
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": caller.ID,
	}, ""))
	return caller, key
}

func grantRoute(t *testing.T, router http.Handler, callerID string, targetID string, routeType string, routeKey string) grantResponse {
	t.Helper()
	return decodeData[grantResponse](t, request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": callerID,
		"targetAgentId": targetID,
		"routeType":     routeType,
		"routeKey":      routeKey,
	}, ""))
}

func capabilityByKey(t *testing.T, capabilities []capabilityResponse, key string) capabilityResponse {
	t.Helper()
	for _, capability := range capabilities {
		if capability.Key == key {
			return capability
		}
	}
	t.Fatalf("capability %q not found in %#v", key, capabilities)
	return capabilityResponse{}
}

func createDirectTenant(t *testing.T, repo store.Repository, id string, parentID string, name string, now time.Time) domain.Tenant {
	t.Helper()
	tenant := domain.Tenant{
		ID:             id,
		ParentTenantID: parentID,
		Name:           name,
		Status:         domain.TenantStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	created, err := repo.CreateTenant(t.Context(), tenant)
	if err != nil {
		t.Fatalf("create direct tenant: %v", err)
	}
	return created
}

func createDirectCapabilityWithAction(t *testing.T, repo store.Repository, targetID string, key string, action domain.CapabilityAction, risk domain.CapabilityRisk, sensitivity domain.CapabilitySensitivity, now time.Time) domain.Capability {
	t.Helper()
	return createDirectCapabilityWithActionAndScopes(t, repo, targetID, key, action, risk, sensitivity, nil, now)
}

func createDirectCapabilityWithActionAndScopes(t *testing.T, repo store.Repository, targetID string, key string, action domain.CapabilityAction, risk domain.CapabilityRisk, sensitivity domain.CapabilitySensitivity, dataScopes []domain.DataScope, now time.Time) domain.Capability {
	t.Helper()
	dataDomain := "crm"
	if strings.Contains(key, "ticket") {
		dataDomain = "support"
	}
	capability := domain.Capability{
		ID:              security.NewID("cap"),
		TargetID:        targetID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             key,
		DisplayName:     key,
		Description:     key,
		Action:          action,
		DataDomains:     []string{dataDomain},
		DataScopes:      dataScopes,
		Sensitivity:     sensitivity,
		RiskLevel:       risk,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryPendingReview,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	created, err := repo.UpsertCapability(t.Context(), capability)
	if err != nil {
		t.Fatalf("create direct capability: %v", err)
	}
	return created
}

func createDirectPermissionPackageApprovalRequest(t *testing.T, repo store.Repository, id string, tenantID string, workspaceID string, now time.Time) domain.PermissionPackageApprovalRequest {
	t.Helper()
	targetID := "agt_target_" + id
	callerID := "agt_caller_" + id
	capabilityID := "cap_" + id
	expiresAt := now.Add(24 * time.Hour)
	if !expiresAt.After(time.Now().UTC()) {
		expiresAt = time.Now().UTC().Add(24 * time.Hour)
	}
	if _, err := repo.CreateAgent(t.Context(), domain.Agent{
		ID:          callerID,
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		Name:        "Caller " + id,
		ChannelType: "local",
		Status:      domain.AgentStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create approval request caller %s: %v", id, err)
	}
	if _, err := repo.CreateAgent(t.Context(), domain.Agent{
		ID:          targetID,
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		Name:        "Target " + id,
		ChannelType: "mcp",
		Status:      domain.AgentStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create approval request target %s: %v", id, err)
	}
	if _, err := repo.UpsertCapability(t.Context(), domain.Capability{
		ID:              capabilityID,
		TargetID:        targetID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             "update_ticket",
		DisplayName:     "update_ticket",
		Description:     "update_ticket",
		Action:          domain.CapabilityActionWrite,
		DataDomains:     []string{"support"},
		Sensitivity:     domain.CapabilitySensitivityConfidential,
		RiskLevel:       domain.CapabilityRiskHigh,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create approval request capability %s: %v", id, err)
	}
	request := domain.PermissionPackageApprovalRequest{
		ID:                    id,
		DraftID:               "ppd_" + id,
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              tenantID,
		WorkspaceID:           workspaceID,
		TargetID:              targetID,
		CallerInstanceID:      callerID,
		SubjectSelector:       "user:support-*",
		RequestText:           "Allow support triage updates.",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{capabilityID},
		AllowedCapabilityKeys: []string{"update_ticket"},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
			Reasons: []domain.PermissionPackagePolicyReason{{
				ID:            "policy:" + id,
				CapabilityID:  capabilityID,
				CapabilityKey: "update_ticket",
				Severity:      "high",
				Message:       "Approval is required.",
				ReasonKey:     "permissionPolicy.actionApprovalRequired",
				ReasonValues:  map[string]string{"action": "write"},
			}},
			NextActions: []string{"Request approval before applying this permission request."},
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "admin-key",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   expiresAt,
	}
	created, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), request)
	if err != nil {
		t.Fatalf("create approval request %s: %v", id, err)
	}
	return created
}

func tenantResponseIDs(rows []tenantResponse) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func agentResponseTenantIDs(rows []agentResponse) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TenantID)
	}
	return ids
}

func auditActions(events []auditEventResponse) []string {
	actions := make([]string, 0, len(events))
	for _, event := range events {
		actions = append(actions, event.Action)
	}
	return actions
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test value: %v", err)
	}
	return raw
}

func decodeMCPResult(t *testing.T, resp *httptest.ResponseRecorder) mcpEnvelopeResponse {
	t.Helper()
	if resp.Code != http.StatusOK {
		t.Fatalf("expected MCP HTTP 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload mcpEnvelopeResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode MCP response: %v body=%s", err, resp.Body.String())
	}
	if payload.Error != nil {
		t.Fatalf("unexpected MCP error: %#v", payload.Error)
	}
	if payload.JSONRPC != "2.0" {
		t.Fatalf("unexpected MCP jsonrpc: %#v", payload)
	}
	return payload
}

func mcpToolNamesContain(tools []mcpToolResponse, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func withManagementMCPWriteConfirmation(args map[string]any) map[string]any {
	confirmed := make(map[string]any, len(args)+1)
	for key, value := range args {
		confirmed[key] = value
	}
	confirmed["confirmation"] = map[string]any{
		"confirmed": true,
		"reason":    "Confirmed by automated test for a state-changing Management MCP call.",
	}
	return confirmed
}

func assertMCPToolSafety(t *testing.T, tools []mcpToolResponse, name string, want mcpToolSafety) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		if tool.Safety != want {
			t.Fatalf("tool %q safety=%#v want %#v", name, tool.Safety, want)
		}
		return
	}
	t.Fatalf("tool %q missing from tools/list", name)
}

func assertMCPToolAccess(t *testing.T, tools []mcpToolResponse, name string, want mcpToolAccess) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		if tool.Access != want {
			t.Fatalf("tool %q access=%#v want %#v", name, tool.Access, want)
		}
		return
	}
	t.Fatalf("tool %q missing from tools/list", name)
}

func assertMCPToolLifecycle(t *testing.T, tools []mcpToolResponse, name string, want mcpToolLifecycle) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		if tool.Lifecycle != want {
			t.Fatalf("tool %q lifecycle=%#v want %#v", name, tool.Lifecycle, want)
		}
		return
	}
	t.Fatalf("tool %q missing from tools/list", name)
}

func assertMCPToolExecution(t *testing.T, tools []mcpToolResponse, name string, want mcpToolExecution) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		if tool.Execution != want {
			t.Fatalf("tool %q execution=%#v want %#v", name, tool.Execution, want)
		}
		return
	}
	t.Fatalf("tool %q missing from tools/list", name)
}

func assertMCPToolInputSchemaHasProperty(t *testing.T, tools []mcpToolResponse, name string, property string) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		properties, ok := tool.InputSchema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("tool %q input schema has no object properties: %#v", name, tool.InputSchema)
		}
		if _, ok := properties[property]; !ok {
			t.Fatalf("tool %q input schema missing property %q: %#v", name, property, tool.InputSchema)
		}
		return
	}
	t.Fatalf("tool %q missing from tools/list", name)
}

func assertMCPToolInputSchemaRequiresConfirmation(t *testing.T, tool mcpToolResponse) {
	t.Helper()
	properties, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("tool %q input schema has no object properties: %#v", tool.Name, tool.InputSchema)
	}
	confirmation, ok := properties["confirmation"].(map[string]any)
	if !ok || confirmation["type"] != "object" {
		t.Fatalf("tool %q input schema should expose confirmation object: %#v", tool.Name, tool.InputSchema)
	}
	if !schemaRequiredContains(tool.InputSchema, "confirmation") {
		t.Fatalf("tool %q input schema should require confirmation: %#v", tool.Name, tool.InputSchema)
	}
	confirmationProperties, ok := confirmation["properties"].(map[string]any)
	if !ok {
		t.Fatalf("tool %q confirmation schema has no properties: %#v", tool.Name, confirmation)
	}
	confirmed, ok := confirmationProperties["confirmed"].(map[string]any)
	if !ok || confirmed["type"] != "boolean" {
		t.Fatalf("tool %q confirmation schema should expose confirmed boolean: %#v", tool.Name, confirmation)
	}
	reason, ok := confirmationProperties["reason"].(map[string]any)
	if !ok || reason["type"] != "string" || reason["maxLength"] != float64(500) {
		t.Fatalf("tool %q confirmation schema should expose reason string maxLength 500: %#v", tool.Name, confirmation)
	}
	if !schemaRequiredContains(confirmation, "confirmed") || !schemaRequiredContains(confirmation, "reason") {
		t.Fatalf("tool %q confirmation schema should require confirmed and reason: %#v", tool.Name, confirmation)
	}
	if confirmation["additionalProperties"] != false {
		t.Fatalf("tool %q confirmation schema should reject extra fields: %#v", tool.Name, confirmation)
	}
}

func assertMCPToolInputSchemaDoesNotExposeConfirmation(t *testing.T, tool mcpToolResponse) {
	t.Helper()
	properties, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		return
	}
	if _, ok := properties["confirmation"]; ok {
		t.Fatalf("tool %q should not expose confirmation when it is not required: %#v", tool.Name, tool.InputSchema)
	}
}

func schemaRequiredContains(schema map[string]any, field string) bool {
	raw, ok := schema["required"].([]any)
	if !ok {
		return false
	}
	for _, value := range raw {
		if value == field {
			return true
		}
	}
	return false
}

func assertMCPToolDescriptionDoesNotContain(t *testing.T, tools []mcpToolResponse, name string, forbidden ...string) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		for _, value := range forbidden {
			if strings.Contains(strings.ToLower(tool.Description), strings.ToLower(value)) {
				t.Fatalf("tool %q description contains %q: %q", name, value, tool.Description)
			}
		}
		return
	}
	t.Fatalf("tool %q missing from tools/list", name)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertResponseDoesNotContain(t *testing.T, body string, forbidden ...string) {
	t.Helper()
	for _, value := range forbidden {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if strings.Contains(body, value) {
			t.Fatalf("response leaked forbidden value %q: %s", value, body)
		}
	}
}

func permissionPackagePreflightHasCheck(checks []permissionPackageApplyPreflightCheck, code string, severity string) bool {
	for _, check := range checks {
		if check.Code == code && check.Severity == severity {
			return true
		}
	}
	return false
}

func permissionPackageProductionReadinessHasCheck(checks []permissionPackageProductionReadinessCheck, code string, severity string) bool {
	for _, check := range checks {
		if check.Code == code && check.Severity == severity {
			return true
		}
	}
	return false
}

func permissionPackageWorkbenchHasStep(steps []permissionPackageWorkbenchStep, key string, status string, detailCode string) bool {
	for _, step := range steps {
		if step.Key == key && step.Status == status && step.DetailCode == detailCode {
			return true
		}
	}
	return false
}

func permissionPackageWorkbenchStepByKey(steps []permissionPackageWorkbenchStep, key string) (permissionPackageWorkbenchStep, bool) {
	for _, step := range steps {
		if step.Key == key {
			return step, true
		}
	}
	return permissionPackageWorkbenchStep{}, false
}

func permissionPackageProductionReadinessPath(input map[string]any, approvalRequestID string, subjectID string) string {
	values := url.Values{}
	for _, key := range []string{"tenantId", "workspaceId", "templateId", "targetId", "callerInstanceId", "requestedCapabilityId", "region", "requestText", "subjectSelector"} {
		if value, ok := input[key].(string); ok && value != "" {
			values.Set(key, value)
		}
	}
	if approvalRequestID != "" {
		values.Set("approvalRequestId", approvalRequestID)
	}
	if subjectID != "" {
		values.Set("subjectId", subjectID)
	}
	return "/api/v1/permission-packages/production-readiness?" + values.Encode()
}

func permissionPackageProductionEvidenceReportPath(input map[string]any, approvalRequestID string, subjectID string) string {
	return strings.Replace(permissionPackageProductionReadinessPath(input, approvalRequestID, subjectID), "/production-readiness?", "/production-readiness/report?", 1)
}

func appendPermissionPackageReadinessTrace(
	t *testing.T,
	repo store.Repository,
	decision domain.TraceDecision,
	caller domain.Agent,
	target domain.Agent,
	capability domain.Capability,
	routeKey string,
	subjectID string,
	createdAt time.Time,
) {
	t.Helper()
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:                security.NewID("trc"),
		RunID:             "production-readiness-" + string(decision),
		CallerID:          caller.ID,
		TargetID:          target.ID,
		RouteType:         "mcp",
		RouteKey:          routeKey,
		TenantID:          caller.TenantID,
		WorkspaceID:       caller.WorkspaceID,
		CallerInstanceID:  caller.ID,
		SubjectID:         subjectID,
		CapabilityID:      capability.ID,
		CapabilityVersion: capability.Version,
		Decision:          decision,
		Reason:            "production readiness fixture",
		CreatedAt:         createdAt,
	}); err != nil {
		t.Fatalf("append readiness trace: %v", err)
	}
}

func impactObjectsContain(rows []permissionPackageImpactObject, objectType string, id string, currentStatus string, rollbackAction string) bool {
	for _, row := range rows {
		if row.Type == objectType && row.ID == id && row.CurrentStatus == currentStatus && row.RollbackAction == rollbackAction {
			return true
		}
	}
	return false
}

func remediationActionsContain(rows []permissionPackageRemediationAction, targetType string, targetID string, action string) bool {
	return remediationActionOrder(rows, targetType, targetID, action) > 0
}

func remediationActionOrder(rows []permissionPackageRemediationAction, targetType string, targetID string, action string) int {
	for _, row := range rows {
		if row.TargetType == targetType && row.TargetID == targetID && row.Action == action {
			return row.Order
		}
	}
	return 0
}

func explainEvidenceContains(rows []managementMCPExplainEvidence, layer string) bool {
	for _, row := range rows {
		if row.Layer == layer && row.ID != "" {
			return true
		}
	}
	return false
}

func request(t *testing.T, router http.Handler, method string, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	return requestWithRunID(t, router, method, path, body, bearer, "")
}

func requestWithRunID(t *testing.T, router http.Handler, method string, path string, body any, bearer string, runID string) *httptest.ResponseRecorder {
	t.Helper()
	return requestWithRunIDAndAdmin(t, router, method, path, body, bearer, runID, "")
}

func requestWithRunIDAndSubject(t *testing.T, router http.Handler, method string, path string, body any, bearer string, runID string, subjectID string) *httptest.ResponseRecorder {
	t.Helper()
	rec, req := buildRequest(t, method, path, body, bearer, runID, "")
	if subjectID != "" {
		req.Header.Set("X-AgentHarbor-Subject-Id", subjectID)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func requestWithAdmin(t *testing.T, router http.Handler, method string, path string, body any, bearer string, adminKey string) *httptest.ResponseRecorder {
	t.Helper()
	return requestWithRunIDAndAdmin(t, router, method, path, body, bearer, "", adminKey)
}

func requestWithCookie(t *testing.T, router http.Handler, method string, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	rec, req := buildRequest(t, method, path, body, "", "", "")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func requestWithCookieAndCSRF(t *testing.T, router http.Handler, method string, path string, body any, cookie *http.Cookie, csrfToken string) *httptest.ResponseRecorder {
	t.Helper()
	rec, req := buildRequest(t, method, path, body, "", "", "")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrfToken != "" {
		req.Header.Set("X-AgentHarbor-CSRF", csrfToken)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func legacyConsoleSessionToken(t *testing.T, secret string, payload map[string]any) string {
	t.Helper()
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal legacy session payload: %v", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(rawPayload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("v1.%s.%s", encodedPayload, signature)
}

func requestWithRunIDAndAdmin(t *testing.T, router http.Handler, method string, path string, body any, bearer string, runID string, adminKey string) *httptest.ResponseRecorder {
	t.Helper()
	rec, req := buildRequest(t, method, path, body, bearer, runID, adminKey)
	router.ServeHTTP(rec, req)
	return rec
}

func requestLoginWithClientHeaders(t *testing.T, router http.Handler, remoteAddr string, forwardedFor string) *httptest.ResponseRecorder {
	t.Helper()
	rec, req := buildRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "", "", "")
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func requestLoginWithTransport(t *testing.T, router http.Handler, remoteAddr string, forwardedProto string, tlsRequest bool) *httptest.ResponseRecorder {
	t.Helper()
	headers := map[string]string{}
	if forwardedProto != "" {
		headers["X-Forwarded-Proto"] = forwardedProto
	}
	return requestLoginWithTransportHeaders(t, router, remoteAddr, headers, tlsRequest)
}

func requestLoginWithTransportHeaders(t *testing.T, router http.Handler, remoteAddr string, headers map[string]string, tlsRequest bool) *httptest.ResponseRecorder {
	t.Helper()
	rec, req := buildRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "test-admin"}, "", "", "")
	req.RemoteAddr = remoteAddr
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if tlsRequest {
		req.TLS = &tls.ConnectionState{}
	}
	router.ServeHTTP(rec, req)
	return rec
}

func buildRequest(t *testing.T, method string, path string, body any, bearer string, runID string, adminKey string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if runID != "" {
		req.Header.Set("X-Run-Id", runID)
	}
	if adminKey != "" {
		req.Header.Set("X-Admin-Key", adminKey)
	}
	rec := httptest.NewRecorder()
	return rec, req
}

func decodeData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var env apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v body=%s", err, rec.Body.String())
	}
	if env.Error != "" {
		t.Fatalf("unexpected error response: status=%d error=%s message=%s", rec.Code, env.Error, env.Message)
	}
	var out T
	if err := json.Unmarshal(env.Data, &out); err != nil {
		t.Fatalf("decode data: %v raw=%s", err, string(env.Data))
	}
	return out
}

func assertProductionReportDigest(t *testing.T, rec *httptest.ResponseRecorder, reportDigest string) {
	t.Helper()
	var env apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope for production report digest: %v body=%s", err, rec.Body.String())
	}
	assertProductionReportDigestFromRaw(t, env.Data, reportDigest)
}

func assertProductionReportDigestFromRaw(t *testing.T, raw json.RawMessage, reportDigest string) {
	t.Helper()
	if !isSHA256Hex(reportDigest) {
		t.Fatalf("production report digest should be sha256 hex, got %q", reportDigest)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode production report digest payload: %v raw=%s", err, string(raw))
	}
	if payload["reportDigestAlgorithm"] != "sha256-canonical-json-v1" {
		t.Fatalf("production report digest algorithm=%#v", payload["reportDigestAlgorithm"])
	}
	delete(payload, "reportDigest")
	canonical, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal production report digest payload: %v", err)
	}
	sum := sha256.Sum256(canonical)
	if expected := hex.EncodeToString(sum[:]); reportDigest != expected {
		t.Fatalf("production report digest=%q want %q payload=%s", reportDigest, expected, string(canonical))
	}
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') {
			continue
		}
		return false
	}
	return true
}

func metricByID(t *testing.T, metrics []metricResponse, id string) metricResponse {
	t.Helper()
	for _, metric := range metrics {
		if metric.ID == id {
			return metric
		}
	}
	t.Fatalf("metric %q not found in %#v", id, metrics)
	return metricResponse{}
}

type failingAllowedTraceRepository struct {
	store.Repository
}

func (r *failingAllowedTraceRepository) AppendTrace(ctx context.Context, event domain.TraceEvent) (domain.TraceEvent, error) {
	if event.Decision == domain.TraceDecisionAllowed {
		return domain.TraceEvent{}, errors.New("trace append failed")
	}
	return r.Repository.AppendTrace(ctx, event)
}

type failingAuditedAgentRepository struct {
	store.Repository
}

type capabilityDriftBeforeApplyRepository struct {
	store.Repository
	capabilityID string
	drifted      bool
}

func (r *capabilityDriftBeforeApplyRepository) ApplyPermissionPackage(ctx context.Context, mutation store.PermissionPackageApplyMutation) (store.PermissionPackageApplyMutationResult, error) {
	if !r.drifted {
		r.drifted = true
		capability, ok, err := r.Repository.GetCapability(ctx, r.capabilityID)
		if err != nil {
			return store.PermissionPackageApplyMutationResult{}, err
		}
		if ok {
			capability.RiskLevel = domain.CapabilityRiskCritical
			capability.Version++
			capability.UpdatedAt = capability.UpdatedAt.Add(time.Second)
			if _, _, err := r.Repository.UpdateCapability(ctx, capability); err != nil {
				return store.PermissionPackageApplyMutationResult{}, err
			}
		}
	}
	return r.Repository.ApplyPermissionPackage(ctx, mutation)
}

func (r *failingAuditedAgentRepository) CreateAgentWithAudit(ctx context.Context, agent domain.Agent, build store.AgentAuditBuilder) (domain.Agent, error) {
	return domain.Agent{}, errors.New("audit insert failed")
}

func (r *failingAuditedAgentRepository) UpdateAgentWithAudit(ctx context.Context, agent domain.Agent, build store.AgentAuditBuilder) (domain.Agent, bool, error) {
	return domain.Agent{}, false, errors.New("audit insert failed")
}
