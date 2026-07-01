package httpapi_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"sort"
	"strconv"
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
	Name         string   `json:"name"`
	APIVersion   string   `json:"apiVersion"`
	AuthRequired bool     `json:"authRequired"`
	Capabilities []string `json:"capabilities"`
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
	ID          string         `json:"id"`
	Actor       string         `json:"actor"`
	DisplayName string         `json:"displayName"`
	Role        string         `json:"role"`
	TenantID    string         `json:"tenantId"`
	WorkspaceID string         `json:"workspaceId"`
	Status      string         `json:"status"`
	Source      string         `json:"source"`
	KeyPrefix   string         `json:"keyPrefix"`
	CreatedBy   string         `json:"createdBy"`
	UpdatedBy   string         `json:"updatedBy"`
	Metadata    map[string]any `json:"metadata,omitempty"`
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
	NextActions []string `json:"nextActions"`
}

type permissionPackageApplyResponse struct {
	Draft                permissionPackageDraftResponse        `json:"draft"`
	TenantEntitlements   []tenantEntitlementResponse           `json:"tenantEntitlements"`
	WorkspaceAssignments []workspaceAssignmentResponse         `json:"workspaceAssignments"`
	InstanceAssignments  []instanceAssignmentResponse          `json:"instanceAssignments"`
	Application          *permissionPackageApplicationResponse `json:"application"`
}

type permissionPackageApplyPreflightResponse struct {
	Draft          permissionPackageDraftResponse                 `json:"draft"`
	Summary        permissionPackageApplyPreflightSummary         `json:"summary"`
	Checks         []permissionPackageApplyPreflightCheck         `json:"checks"`
	Planned        permissionPackageApplyPreflightPlannedChanges  `json:"planned"`
	ExistingGrants []permissionPackageApplyPreflightExistingGrant `json:"existingGrants"`
	NextActions    []string                                       `json:"nextActions"`
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
	ReportVersion        string                                      `json:"reportVersion"`
	Status               string                                      `json:"status"`
	Scope                permissionPackageProductionEvidenceScope    `json:"scope"`
	Summary              permissionPackageProductionReadinessSummary `json:"summary"`
	Checks               []permissionPackageProductionReadinessCheck `json:"checks"`
	Evidence             permissionPackageProductionEvidenceRefs     `json:"evidence"`
	NextActionCode       string                                      `json:"nextActionCode"`
	NextActions          []string                                    `json:"nextActions"`
	ReadinessGeneratedAt string                                      `json:"readinessGeneratedAt"`
}

type permissionPackageProductionEvidenceScope struct {
	TenantID         string `json:"tenantId"`
	WorkspaceID      string `json:"workspaceId"`
	TemplateID       string `json:"templateId"`
	TargetID         string `json:"targetId"`
	CallerInstanceID string `json:"callerInstanceId"`
	SubjectID        string `json:"subjectId"`
	Region           string `json:"region"`
	SubjectSelector  string `json:"subjectSelector"`
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
	Present              bool     `json:"present"`
	ID                   string   `json:"id"`
	TemplateVersion      int      `json:"templateVersion"`
	AllowedCapabilityIDs []string `json:"allowedCapabilityIds"`
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
	SubjectSelector         string                              `json:"subjectSelector"`
	RequestText             string                              `json:"requestText"`
	Region                  string                              `json:"region"`
	DataScopes              []dataScopeResponse                 `json:"dataScopes"`
	AllowedCapabilityIDs    []string                            `json:"allowedCapabilityIds"`
	AllowedCapabilityKeys   []string                            `json:"allowedCapabilityKeys"`
	PolicyGate              permissionPackagePolicyGateResponse `json:"policyGate"`
	Status                  string                              `json:"status"`
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
	Tools             []mcpToolResponse `json:"tools"`
	Content           []mcpContentItem  `json:"content"`
	StructuredContent json.RawMessage   `json:"structuredContent"`
}

type mcpToolResponse struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
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
	NextActions []string `json:"nextActions"`
}

type managementMCPExplainAccessResponse struct {
	Outcome     string                          `json:"outcome"`
	Summary     string                          `json:"summary"`
	Decision    domain.CapabilityAccessDecision `json:"decision"`
	Evidence    []managementMCPExplainEvidence  `json:"evidence"`
	DataScopes  []domain.DataScope              `json:"dataScopes"`
	NextActions []string                        `json:"nextActions"`
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

func TestHealthAndContracts(t *testing.T) {
	router := newRouter()

	resp := request(t, router, http.MethodGet, "/healthz", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("health status = %d", resp.Code)
	}

	providers := decodeData[[]map[string]any](t, request(t, router, http.MethodGet, "/api/v1/contracts/providers", nil, ""))
	if len(providers) == 0 || providers[0]["schemaVersion"] == "" {
		t.Fatalf("provider contracts missing schemaVersion: %#v", providers)
	}

	channels := decodeData[[]map[string]any](t, request(t, router, http.MethodGet, "/api/v1/contracts/channels", nil, ""))
	if len(channels) < 3 {
		t.Fatalf("expected channel catalog, got %#v", channels)
	}
}

func TestSystemInfoIncludesConsoleCompatibilityContract(t *testing.T) {
	router := httpapi.New(store.NewMemory(), httpapi.WithAdminKey("secret")).Router()

	resp := request(t, router, http.MethodGet, "/api/v1/system/info", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("system info should be public, status=%d body=%s", resp.Code, resp.Body.String())
	}
	info := decodeData[systemInfoResponse](t, resp)
	if info.Name != "AgentHarbor" {
		t.Fatalf("unexpected system name %q", info.Name)
	}
	if strings.TrimSpace(info.APIVersion) == "" {
		t.Fatalf("apiVersion should be present: %#v", info)
	}
	if !info.AuthRequired {
		t.Fatalf("system info should report auth required when admin auth is configured: %#v", info)
	}
	devInfo := decodeData[systemInfoResponse](t, request(t, newRouter(), http.MethodGet, "/api/v1/system/info", nil, ""))
	if devInfo.AuthRequired {
		t.Fatalf("dev unauthenticated router should report authRequired=false: %#v", devInfo)
	}
	capabilities := make(map[string]bool, len(info.Capabilities))
	for _, capability := range info.Capabilities {
		capabilities[capability] = true
	}
	for _, capability := range []string{
		"permission_package_approval_requests",
		"permission_package_approval_withdraw",
		"permission_package_apply_preflight",
		"permission_package_applications",
		"permission_package_application_health",
		"permission_package_application_impact",
		"permission_package_production_readiness",
		"permission_package_consumed_approval_recovery",
	} {
		if !capabilities[capability] {
			t.Fatalf("system info missing required console capability %q: %#v", capability, info.Capabilities)
		}
	}
}

func TestAdminEndpointsRequireConfiguredAuthenticationByDefault(t *testing.T) {
	router := httpapi.New(store.NewMemory()).Router()

	health := request(t, router, http.MethodGet, "/healthz", nil, "")
	if health.Code != http.StatusOK {
		t.Fatalf("health should remain public, got %d", health.Code)
	}

	agents := request(t, router, http.MethodGet, "/api/v1/agents", nil, "")
	if agents.Code != http.StatusUnauthorized || !strings.Contains(agents.Body.String(), "admin authentication is required") {
		t.Fatalf("admin endpoint should reject missing configured auth, status=%d body=%s", agents.Code, agents.Body.String())
	}
}

func TestManagedAdminIdentityAuthenticatesWithoutBootstrapConfig(t *testing.T) {
	repo := store.NewMemory()
	now := time.Now().UTC()
	_, err := repo.CreateAdminIdentityWithAudit(context.Background(), domain.AdminIdentity{
		ID:          "adm_stored",
		Actor:       "stored-admin",
		DisplayName: "Stored Admin",
		Role:        domain.AdminIdentityRolePlatformAdmin,
		KeyHash:     security.HashSecret("stored-key"),
		KeyPrefix:   "ahadm_stored",
		Status:      domain.AdminIdentityStatusActive,
		Source:      domain.AdminIdentitySourceManaged,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, func(created domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{
			ID:           "audit_stored_admin",
			Action:       "admin_identity.created",
			ResourceType: "admin_identity",
			ResourceID:   created.ID,
			Actor:        "test",
			CreatedAt:    now,
		}
	})
	if err != nil {
		t.Fatalf("seed managed admin identity: %v", err)
	}

	router := httpapi.New(repo).Router()
	agents := requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents", nil, "", "stored-key")
	if agents.Code != http.StatusOK {
		t.Fatalf("stored managed admin should authenticate without bootstrap config, got %d body=%s", agents.Code, agents.Body.String())
	}
}

func TestLocalDevCORS(t *testing.T) {
	router := newRouter()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/contracts/channels", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5174")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5174" {
		t.Fatalf("unexpected allowed origin %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-AgentHarbor-Subject-Id") {
		t.Fatalf("subject header missing from CORS allow headers %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-AgentHarbor-CSRF") {
		t.Fatalf("csrf header missing from CORS allow headers %q", got)
	}

	req = httptest.NewRequest(http.MethodOptions, "/api/v1/auth/session", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5176")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status for non-default local console port = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5176" {
		t.Fatalf("unexpected allowed origin for non-default local console port %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("credentialed console session CORS missing, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/contracts/channels", nil)
	req.Header.Set("Origin", "https://example.invalid")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected CORS header for disallowed origin %q", got)
	}

	req = httptest.NewRequest(http.MethodOptions, "/api/v1/contracts/channels", nil)
	req.Header.Set("Origin", "https://example.invalid")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code == http.StatusNoContent {
		t.Fatalf("disallowed preflight should not be short-circuited")
	}
}

func TestConfiguredCORSOriginAllowsBrowserGateMCPPreflight(t *testing.T) {
	origin := "http://127.0.0.1:15174"
	router := newRouterWithCORSOrigins([]string{origin})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/mcp/agents/browser-gate/rpc", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-AgentHarbor-Subject-Id")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("unexpected allowed origin %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-AgentHarbor-Subject-Id") {
		t.Fatalf("subject header missing from CORS allow headers %q", got)
	}
}

func TestAdminKeyProtectsManagementEndpoints(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	missing := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Blocked Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
	}, "")
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing admin key should be unauthorized, got %d", missing.Code)
	}

	wrong := requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents", nil, "", "wrong-admin")
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong admin key should be unauthorized, got %d", wrong.Code)
	}

	created := decodeData[agentResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Admin Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	}, "", "test-admin"))
	if created.ID == "" {
		t.Fatalf("expected created agent with admin key: %#v", created)
	}

	contracts := request(t, router, http.MethodGet, "/api/v1/contracts/channels", nil, "")
	if contracts.Code != http.StatusOK {
		t.Fatalf("contracts should remain public, got %d", contracts.Code)
	}
}

func TestConsoleAuthSessionProtectsManagementEndpoints(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	missing := request(t, router, http.MethodGet, "/api/v1/auth/session", nil, "")
	if missing.Code != http.StatusOK {
		t.Fatalf("session status should be public, got %d body=%s", missing.Code, missing.Body.String())
	}
	if got := missing.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("session status should not be cached, got Cache-Control=%q", got)
	}
	missingSession := decodeData[map[string]any](t, missing)
	if missingSession["authenticated"] != false || missingSession["requiresLogin"] != true {
		t.Fatalf("expected unauthenticated production session, got %#v", missingSession)
	}

	wrongLogin := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "")
	if wrongLogin.Code != http.StatusUnauthorized || len(wrongLogin.Result().Cookies()) != 0 {
		t.Fatalf("wrong login should be unauthorized without setting cookies, status=%d body=%s cookies=%#v", wrongLogin.Code, wrongLogin.Body.String(), wrongLogin.Result().Cookies())
	}

	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "test-admin"}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login should succeed, got %d body=%s", login.Code, login.Body.String())
	}
	if got := login.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("login response should not be cached, got Cache-Control=%q", got)
	}
	session := decodeData[map[string]any](t, login)
	if session["authenticated"] != true || session["actor"] != "admin-key" || session["requiresLogin"] != true {
		t.Fatalf("unexpected login session: %#v", session)
	}
	csrfToken, ok := session["csrfToken"].(string)
	if !ok || csrfToken == "" {
		t.Fatalf("login session should include csrf token, got %#v", session)
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "agent_harbor_session" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("login should set one HttpOnly SameSite=Lax session cookie, got %#v", cookies)
	}

	blocked := requestWithCookie(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Missing CSRF Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	}, cookies[0])
	if blocked.Code != http.StatusForbidden || !strings.Contains(blocked.Body.String(), "csrf") {
		t.Fatalf("session cookie mutation without csrf should be forbidden, got %d body=%s", blocked.Code, blocked.Body.String())
	}

	invalidCSRF := requestWithCookieAndCSRF(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Invalid CSRF Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	}, cookies[0], "invalid")
	if invalidCSRF.Code != http.StatusForbidden || !strings.Contains(invalidCSRF.Body.String(), "csrf") {
		t.Fatalf("session cookie mutation with invalid csrf should be forbidden, got %d body=%s", invalidCSRF.Code, invalidCSRF.Body.String())
	}

	created := decodeData[agentResponse](t, requestWithCookieAndCSRF(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Session Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	}, cookies[0], csrfToken))
	if created.ID == "" {
		t.Fatalf("expected management request with session cookie to succeed: %#v", created)
	}

	me := decodeData[map[string]any](t, requestWithCookie(t, router, http.MethodGet, "/api/v1/auth/session", nil, cookies[0]))
	if me["authenticated"] != true || me["actor"] != "admin-key" {
		t.Fatalf("expected authenticated session status, got %#v", me)
	}
	if me["csrfToken"] == "" {
		t.Fatalf("session endpoint should refresh csrf token, got %#v", me)
	}

	logout := requestWithCookieAndCSRF(t, router, http.MethodPost, "/api/v1/auth/logout", nil, cookies[0], csrfToken)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout should succeed, got %d body=%s", logout.Code, logout.Body.String())
	}
	if got := logout.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("logout response should not be cached, got Cache-Control=%q", got)
	}
	clearedCookies := logout.Result().Cookies()
	if len(clearedCookies) != 1 || clearedCookies[0].Name != "agent_harbor_session" || clearedCookies[0].MaxAge != -1 {
		t.Fatalf("logout should clear the session cookie, got %#v", clearedCookies)
	}
}

func TestConsoleAuthSessionCookieSecureFlagUsesTLSAndTrustedForwardedProto(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	directTLS := requestLoginWithTransport(t, router, "", "", true)
	if directTLS.Code != http.StatusOK {
		t.Fatalf("TLS login should succeed, got %d body=%s", directTLS.Code, directTLS.Body.String())
	}
	tlsCookies := directTLS.Result().Cookies()
	if len(tlsCookies) != 1 || !tlsCookies[0].Secure {
		t.Fatalf("TLS login should set a Secure session cookie, got %#v", tlsCookies)
	}

	proxiedHTTPS := requestLoginWithTransport(t, router, "127.0.0.1:4000", "https", false)
	if proxiedHTTPS.Code != http.StatusOK {
		t.Fatalf("trusted forwarded HTTPS login should succeed, got %d body=%s", proxiedHTTPS.Code, proxiedHTTPS.Body.String())
	}
	proxiedCookies := proxiedHTTPS.Result().Cookies()
	if len(proxiedCookies) != 1 || !proxiedCookies[0].Secure {
		t.Fatalf("trusted forwarded HTTPS login should set a Secure session cookie, got %#v", proxiedCookies)
	}

	proxiedForwarded := requestLoginWithTransportHeaders(t, router, "10.0.0.10:4000", map[string]string{
		"Forwarded": `for=203.0.113.8;proto=https;host=console.example.com`,
	}, false)
	if proxiedForwarded.Code != http.StatusOK {
		t.Fatalf("trusted Forwarded proto login should succeed, got %d body=%s", proxiedForwarded.Code, proxiedForwarded.Body.String())
	}
	forwardedCookies := proxiedForwarded.Result().Cookies()
	if len(forwardedCookies) != 1 || !forwardedCookies[0].Secure {
		t.Fatalf("trusted Forwarded proto login should set a Secure session cookie, got %#v", forwardedCookies)
	}
}

func TestConsoleAuthSessionCookieSecureFlagIgnoresUntrustedForwardedProto(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	login := requestLoginWithTransport(t, router, "198.51.100.20:4000", "https", false)
	if login.Code != http.StatusOK {
		t.Fatalf("login should succeed even when forwarded proto is untrusted, got %d body=%s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Secure {
		t.Fatalf("public clients must not force Secure session cookies by spoofing X-Forwarded-Proto, got %#v", cookies)
	}
}

func TestConsoleLoginRateLimit(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	for i := 0; i < 5; i++ {
		resp := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "")
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("failed login %d should be unauthorized before throttle, got %d body=%s", i+1, resp.Code, resp.Body.String())
		}
	}
	limited := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "")
	if limited.Code != http.StatusTooManyRequests || !strings.Contains(limited.Body.String(), "RATE_LIMITED") {
		t.Fatalf("sixth failed login should be rate limited, got %d body=%s", limited.Code, limited.Body.String())
	}
	if got := limited.Header().Get("Retry-After"); got != "300" {
		t.Fatalf("rate limited login should expose retry window, got Retry-After=%q", got)
	}

	apiKeyCreate := decodeData[agentResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "API Key Still Works",
		"workspaceId": "ws-login-limit",
		"channelType": "local",
		"status":      "active",
	}, "", "test-admin"))
	if apiKeyCreate.ID == "" {
		t.Fatalf("api key management request should bypass login throttle: %#v", apiKeyCreate)
	}
}

func TestConsoleLoginRateLimitTrustsForwardedForOnlyFromTrustedProxy(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	for i := 0; i < 5; i++ {
		resp := requestLoginWithClientHeaders(t, router, "198.51.100.20:4000", "203.0.113."+strconv.Itoa(i+1))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("public failed login %d should be unauthorized before throttle, got %d body=%s", i+1, resp.Code, resp.Body.String())
		}
	}
	limited := requestLoginWithClientHeaders(t, router, "198.51.100.20:4000", "203.0.113.99")
	if limited.Code != http.StatusTooManyRequests || !strings.Contains(limited.Body.String(), "RATE_LIMITED") {
		t.Fatalf("public client should not bypass rate limit by changing X-Forwarded-For, got %d body=%s", limited.Code, limited.Body.String())
	}

	proxied := requestLoginWithClientHeaders(t, router, "127.0.0.1:4000", "203.0.113.200")
	if proxied.Code != http.StatusUnauthorized {
		t.Fatalf("trusted local proxy should isolate a different forwarded client, got %d body=%s", proxied.Code, proxied.Body.String())
	}
}

func TestConsoleLoginRateLimitClearsAfterSuccess(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	for i := 0; i < 4; i++ {
		resp := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "")
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("failed login %d should be unauthorized before success, got %d body=%s", i+1, resp.Code, resp.Body.String())
		}
	}
	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "test-admin"}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("valid login should clear failure count, got %d body=%s", login.Code, login.Body.String())
	}
	wrongAfterSuccess := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "")
	if wrongAfterSuccess.Code != http.StatusUnauthorized {
		t.Fatalf("wrong login after success should restart failure count, got %d body=%s", wrongAfterSuccess.Code, wrongAfterSuccess.Body.String())
	}
}

func TestConsoleAuthSessionSupportsNamedAdminIdentities(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
	})

	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "security-key"}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login should accept named identity key, got %d body=%s", login.Code, login.Body.String())
	}
	session := decodeData[map[string]any](t, login)
	if session["actor"] != "security" {
		t.Fatalf("expected named actor in session, got %#v", session)
	}
}

func TestConsoleAuthSessionReportsScopedAdminIdentity(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "east-key"}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login should accept scoped admin key, got %d body=%s", login.Code, login.Body.String())
	}
	session := decodeData[map[string]any](t, login)
	if session["actor"] != "east-admin" || session["role"] != "tenant_admin" || session["tenantId"] != "tenant-east" || session["workspaceId"] != "ws-support" {
		t.Fatalf("unexpected scoped session response: %#v", session)
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %#v", cookies)
	}
	me := decodeData[map[string]any](t, requestWithCookie(t, router, http.MethodGet, "/api/v1/auth/session", nil, cookies[0]))
	if me["actor"] != "east-admin" || me["role"] != "tenant_admin" || me["tenantId"] != "tenant-east" || me["workspaceId"] != "ws-support" {
		t.Fatalf("session endpoint should return scoped principal, got %#v", me)
	}
}

func TestManagedAdminIdentityLifecycleAndScopedLogin(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	create := decodeData[createAdminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":       "east-admin",
		"displayName": "East Administrator",
		"role":        "tenant_admin",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
	}, "", "platform-key"))
	if create.Identity.Actor != "east-admin" || create.Identity.Source != "managed" || create.Identity.Status != "active" {
		t.Fatalf("unexpected created managed admin: %#v", create)
	}
	if create.Key == "" || !strings.Contains(create.Key, create.Identity.KeyPrefix) {
		t.Fatalf("expected one-time key with visible prefix, got response=%#v", create)
	}

	list := decodeData[[]adminIdentityResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/admin-identities", nil, "", "platform-key"))
	if len(list) != 2 {
		t.Fatalf("expected bootstrap plus managed identity, got %#v", list)
	}
	for _, row := range list {
		if bytes.Contains(mustJSON(t, row), []byte(create.Key)) {
			t.Fatalf("list response must not expose one-time admin key: %#v", row)
		}
	}

	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": create.Key}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("managed admin key should log in, got %d body=%s", login.Code, login.Body.String())
	}
	session := decodeData[map[string]any](t, login)
	if session["actor"] != "east-admin" || session["role"] != "tenant_admin" || session["tenantId"] != "tenant-east" || session["workspaceId"] != "ws-support" {
		t.Fatalf("unexpected managed session: %#v", session)
	}
	csrfToken, ok := session["csrfToken"].(string)
	if !ok || csrfToken == "" {
		t.Fatalf("managed login should return csrf token, got %#v", session)
	}
	managedCookies := login.Result().Cookies()
	if len(managedCookies) != 1 {
		t.Fatalf("expected managed login session cookie, got %#v", managedCookies)
	}

	rotate := decodeData[rotateAdminIdentityKeyResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+create.Identity.ID+"/key:rotate", nil, "", "platform-key"))
	if rotate.Key == "" || rotate.Key == create.Key || rotate.Identity.KeyPrefix == create.Identity.KeyPrefix {
		t.Fatalf("expected rotated key and prefix, got %#v", rotate)
	}
	oldLogin := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": create.Key}, "")
	if oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old key must be invalid after rotation, got %d body=%s", oldLogin.Code, oldLogin.Body.String())
	}
	staleSessionWrite := requestWithCookieAndCSRF(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Stale Managed Session Caller",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
		"channelType": "local",
		"status":      "active",
	}, managedCookies[0], csrfToken)
	if staleSessionWrite.Code != http.StatusUnauthorized {
		t.Fatalf("old managed session must be invalid after key rotation, got %d body=%s", staleSessionWrite.Code, staleSessionWrite.Body.String())
	}
	newLogin := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": rotate.Key}, "")
	if newLogin.Code != http.StatusOK {
		t.Fatalf("rotated key should log in, got %d body=%s", newLogin.Code, newLogin.Body.String())
	}
	rotatedSession := decodeData[map[string]any](t, newLogin)
	rotatedCSRFToken, ok := rotatedSession["csrfToken"].(string)
	if !ok || rotatedCSRFToken == "" {
		t.Fatalf("rotated managed login should return csrf token, got %#v", rotatedSession)
	}
	rotatedCookies := newLogin.Result().Cookies()
	if len(rotatedCookies) != 1 {
		t.Fatalf("expected rotated managed login session cookie, got %#v", rotatedCookies)
	}

	disabled := decodeData[adminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+create.Identity.ID+":disable", nil, "", "platform-key"))
	if disabled.Status != "disabled" {
		t.Fatalf("expected disabled managed admin, got %#v", disabled)
	}
	disabledLogin := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": rotate.Key}, "")
	if disabledLogin.Code != http.StatusUnauthorized {
		t.Fatalf("disabled admin key must be invalid, got %d body=%s", disabledLogin.Code, disabledLogin.Body.String())
	}
	disabledSessionWrite := requestWithCookieAndCSRF(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Disabled Managed Session Caller",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
		"channelType": "local",
		"status":      "active",
	}, rotatedCookies[0], rotatedCSRFToken)
	if disabledSessionWrite.Code != http.StatusUnauthorized {
		t.Fatalf("disabled managed session must be invalid, got %d body=%s", disabledSessionWrite.Code, disabledSessionWrite.Body.String())
	}
	rotateDisabled := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+create.Identity.ID+"/key:rotate", nil, "", "platform-key")
	if rotateDisabled.Code != http.StatusConflict || !strings.Contains(rotateDisabled.Body.String(), "ADMIN_IDENTITY_DISABLED") {
		t.Fatalf("disabled managed admin key rotation should be rejected, got %d body=%s", rotateDisabled.Code, rotateDisabled.Body.String())
	}
	disableAgain := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+create.Identity.ID+":disable", nil, "", "platform-key")
	if disableAgain.Code != http.StatusConflict || !strings.Contains(disableAgain.Body.String(), "ADMIN_IDENTITY_DISABLED") {
		t.Fatalf("disabled managed admin disable retry should be rejected, got %d body=%s", disableAgain.Code, disableAgain.Body.String())
	}

	events := decodeData[[]auditEventResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/audit/events?resourceType=admin_identity", nil, "", "platform-key"))
	if got := auditActions(events); !reflect.DeepEqual(got, []string{"admin_identity.created", "admin_identity.key_rotated", "admin_identity.disabled"}) {
		t.Fatalf("unexpected admin identity audit actions: %#v", got)
	}
	for _, event := range events {
		raw := mustJSON(t, event)
		if bytes.Contains(raw, []byte(create.Key)) || bytes.Contains(raw, []byte(rotate.Key)) || bytes.Contains(raw, []byte("keyHash")) {
			t.Fatalf("audit event must not expose admin key material: %s", raw)
		}
	}
}

func TestManagedAdminSessionIssuedAfterRotationInSameSecondStaysValid(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 26, 10, 0, 0, 100*int(time.Millisecond), time.UTC)
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{
			{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()

	create := decodeData[createAdminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":       "same-second-admin",
		"displayName": "Same Second Admin",
		"role":        "tenant_admin",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
	}, "", "platform-key"))

	now = time.Date(2026, 6, 26, 10, 0, 0, 200*int(time.Millisecond), time.UTC)
	rotate := decodeData[rotateAdminIdentityKeyResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+create.Identity.ID+"/key:rotate", nil, "", "platform-key"))

	now = time.Date(2026, 6, 26, 10, 0, 0, 300*int(time.Millisecond), time.UTC)
	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": rotate.Key}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("rotated key should log in inside the rotation second, got %d body=%s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected rotated login cookie, got %#v", cookies)
	}

	now = time.Date(2026, 6, 26, 10, 0, 0, 400*int(time.Millisecond), time.UTC)
	session := requestWithCookie(t, router, http.MethodGet, "/api/v1/auth/session", nil, cookies[0])
	if session.Code != http.StatusOK {
		t.Fatalf("same-second post-rotation session should stay valid, got %d body=%s", session.Code, session.Body.String())
	}
}

func TestManagedAdminSessionAcceptsLegacySecondIssuedAt(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 26, 11, 0, 0, 100*int(time.Millisecond), time.UTC)
	sessionSecret := "legacy-session-secret-0123456789"
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{
			{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		}),
		httpapi.WithClock(func() time.Time { return now }),
		httpapi.WithSessionSecret(sessionSecret),
	).Router()

	create := decodeData[createAdminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":       "legacy-session-admin",
		"displayName": "Legacy Session Admin",
		"role":        "tenant_admin",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
	}, "", "platform-key"))

	now = time.Date(2026, 6, 26, 11, 0, 0, 200*int(time.Millisecond), time.UTC)
	_ = decodeData[rotateAdminIdentityKeyResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+create.Identity.ID+"/key:rotate", nil, "", "platform-key"))

	issuedAt := time.Date(2026, 6, 26, 11, 0, 1, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	token := legacyConsoleSessionToken(t, sessionSecret, map[string]any{
		"actor":       "legacy-session-admin",
		"role":        "tenant_admin",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
		"issuedAt":    issuedAt.Unix(),
		"expiresAt":   expiresAt.Unix(),
	})
	now = time.Date(2026, 6, 26, 11, 0, 1, 100*int(time.Millisecond), time.UTC)
	session := requestWithCookie(t, router, http.MethodGet, "/api/v1/auth/session", nil, &http.Cookie{Name: "agent_harbor_session", Value: token})
	if session.Code != http.StatusOK {
		t.Fatalf("legacy second-issued session should stay valid, got %d body=%s", session.Code, session.Body.String())
	}
}

func TestScopedAdminCannotManageAdminIdentities(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{name: "list", method: http.MethodGet, path: "/api/v1/admin-identities"},
		{name: "create", method: http.MethodPost, path: "/api/v1/admin-identities", body: map[string]any{"actor": "bad", "role": "tenant_admin", "tenantId": "tenant-east"}},
		{name: "rotate", method: http.MethodPost, path: "/api/v1/admin-identities/adm_missing/key:rotate"},
		{name: "disable", method: http.MethodPost, path: "/api/v1/admin-identities/adm_missing:disable"},
	} {
		resp := requestWithAdmin(t, router, tc.method, tc.path, tc.body, "", "east-key")
		if resp.Code != http.StatusForbidden {
			t.Fatalf("%s should be forbidden for scoped admin, got %d body=%s", tc.name, resp.Code, resp.Body.String())
		}
	}
}

func TestManagementMCPAdminIdentityTools(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east"},
	})

	tools := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "tools",
		"method":  "tools/list",
	}, "", "platform-key"))
	for _, name := range []string{"list_admin_identities", "create_admin_identity", "rotate_admin_identity_key", "disable_admin_identity"} {
		if !mcpToolNamesContain(tools.Result.Tools, name) {
			t.Fatalf("admin identity tool %q missing from tools/list: %#v", name, tools.Result.Tools)
		}
	}

	create := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "create-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "create_admin_identity",
			"arguments": map[string]any{"actor": "mcp-east", "role": "tenant_admin", "tenantId": "tenant-east"},
		},
	}, "", "platform-key"))
	var created createAdminIdentityResponse
	if err := json.Unmarshal(create.Result.StructuredContent, &created); err != nil {
		t.Fatalf("decode create_admin_identity result: %v", err)
	}
	if created.Identity.Actor != "mcp-east" || !strings.HasPrefix(created.Key, "ahadm_") {
		t.Fatalf("unexpected create_admin_identity result: %#v", created)
	}

	rotate := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "rotate-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "rotate_admin_identity_key",
			"arguments": map[string]any{"id": created.Identity.ID},
		},
	}, "", "platform-key"))
	var rotated rotateAdminIdentityKeyResponse
	if err := json.Unmarshal(rotate.Result.StructuredContent, &rotated); err != nil {
		t.Fatalf("decode rotate_admin_identity_key result: %v", err)
	}
	if rotated.Key == "" || rotated.Key == created.Key || rotated.Identity.KeyPrefix == created.Identity.KeyPrefix {
		t.Fatalf("unexpected rotate_admin_identity_key result: %#v", rotated)
	}

	disable := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "disable-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "disable_admin_identity",
			"arguments": map[string]any{"id": created.Identity.ID},
		},
	}, "", "platform-key"))
	var disabled adminIdentityResponse
	if err := json.Unmarshal(disable.Result.StructuredContent, &disabled); err != nil {
		t.Fatalf("decode disable_admin_identity result: %v", err)
	}
	if disabled.Status != "disabled" {
		t.Fatalf("unexpected disable_admin_identity result: %#v", disabled)
	}

	rotateDisabled := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "rotate-disabled-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "rotate_admin_identity_key",
			"arguments": map[string]any{"id": created.Identity.ID},
		},
	}, "", "platform-key")
	if rotateDisabled.Code != http.StatusOK {
		t.Fatalf("expected MCP HTTP 200 for disabled rotation tool error, got %d body=%s", rotateDisabled.Code, rotateDisabled.Body.String())
	}
	var rotateDisabledPayload mcpEnvelopeResponse
	if err := json.Unmarshal(rotateDisabled.Body.Bytes(), &rotateDisabledPayload); err != nil {
		t.Fatalf("decode disabled rotation MCP response: %v body=%s", err, rotateDisabled.Body.String())
	}
	if rotateDisabledPayload.Error == nil || rotateDisabledPayload.Error.Data == nil ||
		rotateDisabledPayload.Error.Data.AppCode != "ADMIN_IDENTITY_DISABLED" ||
		rotateDisabledPayload.Error.Data.HTTPStatus != http.StatusConflict {
		t.Fatalf("disabled admin rotation MCP error should include app code and status, got %#v body=%s", rotateDisabledPayload, rotateDisabled.Body.String())
	}

	disableAgain := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "disable-disabled-admin",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "disable_admin_identity",
			"arguments": map[string]any{"id": created.Identity.ID},
		},
	}, "", "platform-key")
	if disableAgain.Code != http.StatusOK {
		t.Fatalf("expected MCP HTTP 200 for disabled disable tool error, got %d body=%s", disableAgain.Code, disableAgain.Body.String())
	}
	var disableAgainPayload mcpEnvelopeResponse
	if err := json.Unmarshal(disableAgain.Body.Bytes(), &disableAgainPayload); err != nil {
		t.Fatalf("decode disabled disable MCP response: %v body=%s", err, disableAgain.Body.String())
	}
	if disableAgainPayload.Error == nil || disableAgainPayload.Error.Data == nil ||
		disableAgainPayload.Error.Data.AppCode != "ADMIN_IDENTITY_DISABLED" ||
		disableAgainPayload.Error.Data.HTTPStatus != http.StatusConflict {
		t.Fatalf("disabled admin disable MCP error should include app code and status, got %#v body=%s", disableAgainPayload, disableAgain.Body.String())
	}

	forbidden := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "forbidden",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_admin_identities",
			"arguments": map[string]any{},
		},
	}, "", "east-key")
	if forbidden.Code != http.StatusOK {
		t.Fatalf("expected MCP HTTP 200 for tool error, got %d body=%s", forbidden.Code, forbidden.Body.String())
	}
	var forbiddenPayload mcpEnvelopeResponse
	if err := json.Unmarshal(forbidden.Body.Bytes(), &forbiddenPayload); err != nil {
		t.Fatalf("decode forbidden MCP response: %v body=%s", err, forbidden.Body.String())
	}
	if forbiddenPayload.Error == nil || !strings.Contains(forbiddenPayload.Error.Message, "platform administrator is required") {
		t.Fatalf("scoped admin MCP call should be rejected, got %#v body=%s", forbiddenPayload, forbidden.Body.String())
	}
}

func TestManagedAdminIdentityRejectsInvalidActorSyntax(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	for _, actor := range []string{"security reviewer", "security/reviewer", "security|reviewer", strings.Repeat("a", 81)} {
		t.Run(actor, func(t *testing.T) {
			res := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
				"actor":       actor,
				"displayName": "Invalid Security Reviewer",
				"role":        "security_reviewer",
				"tenantId":    "tenant-east",
			}, "", "platform-key")
			if res.Code != http.StatusBadRequest {
				t.Fatalf("invalid managed admin actor should be rejected, got %d body=%s", res.Code, res.Body.String())
			}
			if body := res.Body.String(); !strings.Contains(body, "actor") || !strings.Contains(body, "letters, numbers") {
				t.Fatalf("expected actor syntax message, got body=%s", body)
			}
		})
	}
}

func TestManagedAdminIdentityRejectsReservedActors(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	for _, actor := range []string{"admin-key", "local-dev"} {
		t.Run(actor, func(t *testing.T) {
			res := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
				"actor":       actor,
				"displayName": "Reserved Actor",
				"role":        "platform_admin",
			}, "", "platform-key")
			if res.Code != http.StatusBadRequest {
				t.Fatalf("reserved managed admin actor should be rejected, got %d body=%s", res.Code, res.Body.String())
			}
			if body := res.Body.String(); !strings.Contains(body, "actor") || !strings.Contains(body, "reserved") {
				t.Fatalf("expected reserved actor message, got body=%s", body)
			}
		})
	}
}

func TestManagedAdminIdentityRejectsBootstrapActorReuse(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	res := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":       "platform",
		"displayName": "Duplicate Bootstrap Actor",
		"role":        "tenant_admin",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
	}, "", "platform-key")
	if res.Code != http.StatusBadRequest {
		t.Fatalf("managed admin actor should not reuse bootstrap actor, got %d body=%s", res.Code, res.Body.String())
	}
	if body := res.Body.String(); !strings.Contains(body, "actor already exists") {
		t.Fatalf("expected duplicate actor message, got body=%s", body)
	}
}

func TestAdminIdentityLifecycleRejectsBootstrapAndSelfDisable(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	list := decodeData[[]adminIdentityResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/admin-identities", nil, "", "platform-key"))
	if len(list) != 1 || list[0].Source != "bootstrap" {
		t.Fatalf("expected one bootstrap identity, got %#v", list)
	}
	rotateBootstrap := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+list[0].ID+"/key:rotate", nil, "", "platform-key")
	if rotateBootstrap.Code != http.StatusBadRequest {
		t.Fatalf("bootstrap identity rotation should be rejected, got %d body=%s", rotateBootstrap.Code, rotateBootstrap.Body.String())
	}

	created := decodeData[createAdminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":       "managed-platform",
		"displayName": "Managed Platform",
		"role":        "platform_admin",
	}, "", "platform-key"))
	disableManagedPlatform := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+created.Identity.ID+":disable", nil, "", created.Key)
	if disableManagedPlatform.Code != http.StatusForbidden {
		t.Fatalf("self-disable should be rejected, got %d body=%s", disableManagedPlatform.Code, disableManagedPlatform.Body.String())
	}
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

func TestTenantPermissionCenterSummarizesTenantForPlatformAdmin(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	tenantID, workspaceID, _, target, capability := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	_, err := repo.CreateAdminIdentityWithAudit(context.Background(), domain.AdminIdentity{
		ID:          "adm_center_tenant",
		Actor:       "tenant-admin-center",
		DisplayName: "Tenant Center Admin",
		Role:        domain.AdminIdentityRoleTenantAdmin,
		KeyHash:     security.HashSecret("tenant-admin-key"),
		KeyPrefix:   "ahadm_center",
		Status:      domain.AdminIdentityStatusActive,
		Source:      domain.AdminIdentitySourceManaged,
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, func(identity domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: "audit_center_admin", Action: "admin_identity.created", ResourceType: "admin_identity", ResourceID: identity.ID, Actor: "platform", CreatedAt: now}
	})
	if err != nil {
		t.Fatalf("seed admin identity: %v", err)
	}

	center := decodeData[tenantPermissionCenterResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "platform-key"))
	if center.Tenant.ID != tenantID {
		t.Fatalf("unexpected tenant center tenant: %#v", center.Tenant)
	}
	if center.OperatorBoundary.Actor != "platform" || !center.OperatorBoundary.CanManageAdministrators {
		t.Fatalf("platform boundary should manage administrators, got %#v", center.OperatorBoundary)
	}
	if len(center.Administrators) != 1 || center.Administrators[0].Actor != "tenant-admin-center" {
		t.Fatalf("expected tenant administrator summary, got %#v", center.Administrators)
	}
	if len(center.Workspaces) != 1 || center.Workspaces[0].WorkspaceID != workspaceID || center.Workspaces[0].CallerCount != 1 || center.Workspaces[0].TargetCount != 1 {
		t.Fatalf("unexpected workspace summary: %#v", center.Workspaces)
	}
	if len(center.Capabilities) != 1 || center.Capabilities[0].CapabilityID != capability.ID || center.Capabilities[0].TargetID != target.ID || center.Capabilities[0].Effect != "allow" {
		t.Fatalf("unexpected capability summary: %#v", center.Capabilities)
	}
	if len(center.PermissionPacks) == 0 || center.PermissionPacks[0].Status == "" {
		t.Fatalf("expected permission package projection, got %#v", center.PermissionPacks)
	}
	if got := tenantCenterActionCodes(center.NextActions); !reflect.DeepEqual(got, []string{"manage_administrators", "open_access_profile", "start_permission_change"}) {
		t.Fatalf("unexpected next actions: %#v", got)
	}
	raw := mustJSON(t, center)
	if bytes.Contains(raw, []byte("tenant-admin-key")) || bytes.Contains(raw, []byte("keyHash")) {
		t.Fatalf("tenant center must not expose admin key material: %s", raw)
	}
}

func tenantCenterActionCodes(actions []tenantPermissionCenterNextAction) []string {
	codes := make([]string, 0, len(actions))
	for _, action := range actions {
		codes = append(codes, action.Code)
	}
	sort.Strings(codes)
	return codes
}

func TestTenantPermissionCenterHonorsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, workspaceID, _, _, _ := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-west-center", "tenant-root-center", "Finance", now)

	scopedResponse := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "east-key")
	center := decodeData[tenantPermissionCenterResponse](t, scopedResponse)
	if center.OperatorBoundary.Actor != "east-admin" || center.OperatorBoundary.CanManageAdministrators {
		t.Fatalf("tenant admin boundary should be read-only for admin management, got %#v", center.OperatorBoundary)
	}
	if center.Administrators == nil || bytes.Contains(scopedResponse.Body.Bytes(), []byte(`"administrators":null`)) {
		t.Fatalf("tenant admin boundary should return an empty administrators array, body=%s", scopedResponse.Body.String())
	}
	if center.OperatorBoundary.TenantID != tenantID || center.OperatorBoundary.WorkspaceID != workspaceID {
		t.Fatalf("unexpected scoped boundary: %#v", center.OperatorBoundary)
	}
	if got := tenantCenterActionCodes(center.NextActions); slices.Contains(got, "manage_administrators") {
		t.Fatalf("tenant admin should not get manage_administrators action: %#v", got)
	}

	widen := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/tenant-west-center/permission-center", nil, "", "east-key")
	if widen.Code != http.StatusForbidden {
		t.Fatalf("tenant admin should not fetch outside tenant center, got %d body=%s", widen.Code, widen.Body.String())
	}
}

func TestTenantPermissionCenterFiltersCapabilitiesToScopedWorkspace(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, _, _, _, supportCapability := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", tenantID, "ws-finance-center", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	financeEntitlement := createDirectTenantEntitlement(t, repo, tenantID, financeTarget.ID, financeCapability.ID, []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)
	createDirectWorkspaceAssignment(t, repo, financeEntitlement.ID, tenantID, "ws-finance-center", []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)

	center := decodeData[tenantPermissionCenterResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "support-key"))
	if len(center.Capabilities) != 1 {
		t.Fatalf("scoped workspace should see exactly one capability, got %#v", center.Capabilities)
	}
	if center.Capabilities[0].CapabilityID != supportCapability.ID {
		t.Fatalf("scoped workspace leaked another capability: %#v", center.Capabilities)
	}
	raw := mustJSON(t, center)
	if bytes.Contains(raw, []byte(financeCapability.ID)) || bytes.Contains(raw, []byte("finance")) || bytes.Contains(raw, []byte("ws-finance-center")) {
		t.Fatalf("scoped workspace should not expose finance capability or data scope: %s", raw)
	}
}

func TestTenantPermissionCenterRedactsOutOfScopeHydratedObjects(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-child-center", WorkspaceID: "ws-support-center"},
	})
	tenantID, _, _, _, _ := seedTenantPermissionCenterFixture(t, repo)
	now := time.Now().UTC()
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", tenantID, "ws-finance-center", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	financeEntitlement := createDirectTenantEntitlement(t, repo, tenantID, financeTarget.ID, financeCapability.ID, []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)
	createDirectWorkspaceAssignment(t, repo, financeEntitlement.ID, tenantID, "ws-support-center", []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)

	resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/"+tenantID+"/permission-center", nil, "", "support-key")
	if resp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read tenant permission center, got %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if strings.Contains(body, "Finance Export Service") || strings.Contains(body, "export_invoices") {
		t.Fatalf("tenant permission center leaked hydrated out-of-scope object details: %s", body)
	}
}

func TestTenantPermissionCenterRequiresRegisteredTenant(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})
	resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/unregistered/permission-center", nil, "", "platform-key")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("permission center should require registered tenant, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestScopedAdminIdentityCannotWidenManagementLists(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastTarget := createDirectAgent(t, repo, "East MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, eastTarget.ID, "east_tool", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	createDirectCapabilityWithAction(t, repo, westTarget.ID, "west_tool", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	tenants := decodeData[[]tenantResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants", nil, "", "east-key"))
	if got := tenantResponseIDs(tenants); !reflect.DeepEqual(got, []string{"tenant-east"}) {
		t.Fatalf("scoped tenant admin should only see own tenant tree, got %#v", got)
	}

	agents := decodeData[[]agentResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents", nil, "", "east-key"))
	if len(agents) != 1 || agents[0].ID != eastTarget.ID {
		t.Fatalf("scoped tenant admin should only see own agents, got %#v", agents)
	}

	capabilities := decodeData[[]capabilityResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/capabilities", nil, "", "east-key"))
	if len(capabilities) != 1 || capabilities[0].Key != "east_tool" {
		t.Fatalf("scoped tenant admin should only see own capabilities, got %#v", capabilities)
	}

	widen := requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents?tenantId=tenant-west&workspaceId=ws-finance", nil, "", "east-key")
	if widen.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not widen to another tenant, got %d body=%s", widen.Code, widen.Body.String())
	}
}

func TestScopedAdminIdentityCannotReadOutsideScopeByID(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastAgent := createDirectAgent(t, repo, "East MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	westAgent := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)

	ownAgent := decodeData[agentResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents/"+eastAgent.ID, nil, "", "east-key"))
	if ownAgent.ID != eastAgent.ID {
		t.Fatalf("scoped admin should read own agent by id, got %#v", ownAgent)
	}

	westTenant := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/tenant-west", nil, "", "east-key")
	if westTenant.Code != http.StatusForbidden || strings.Contains(westTenant.Body.String(), "West tenant") {
		t.Fatalf("scoped admin should not read outside tenant by id, got %d body=%s", westTenant.Code, westTenant.Body.String())
	}

	westAgentByID := requestWithAdmin(t, router, http.MethodGet, "/api/v1/agents/"+westAgent.ID, nil, "", "east-key")
	if westAgentByID.Code != http.StatusForbidden || strings.Contains(westAgentByID.Body.String(), "West MCP") {
		t.Fatalf("scoped admin should not read outside agent by id, got %d body=%s", westAgentByID.Code, westAgentByID.Body.String())
	}
}

func TestScopedAdminIdentityAccessGrantListHidesCrossScopeRelations(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	westCaller := createDirectAgent(t, repo, "West caller", "tenant-west", "ws-finance", "local", domain.AgentStatusActive, nil)

	supportGrant, err := repo.CreateAccessGrant(t.Context(), domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  supportCaller.ID,
		TargetID:  supportTarget.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("create support grant: %v", err)
	}
	crossGrant, err := repo.CreateAccessGrant(t.Context(), domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  westCaller.ID,
		TargetID:  supportTarget.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("create cross-scope grant: %v", err)
	}

	resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/access-grants", nil, "", "support-key")
	if resp.Code != http.StatusOK {
		t.Fatalf("list access grants failed: %d body=%s", resp.Code, resp.Body.String())
	}
	grants := decodeData[[]domain.AccessGrant](t, resp)
	if len(grants) != 1 || grants[0].ID != supportGrant.ID {
		t.Fatalf("scoped admin should only see same-scope grant %s, got %#v", supportGrant.ID, grants)
	}
	if body := resp.Body.String(); strings.Contains(body, crossGrant.ID) || strings.Contains(body, westCaller.ID) {
		t.Fatalf("scoped access grant list leaked cross-scope relation: %s", body)
	}
}

func TestScopedAdminIdentityCannotRevokeCrossScopeAccessGrant(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	crossGrant, err := repo.CreateAccessGrant(t.Context(), domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  supportCaller.ID,
		TargetID:  westTarget.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("create cross-scope grant: %v", err)
	}

	resp := requestWithAdmin(t, router, http.MethodDelete, "/api/v1/access-grants/"+crossGrant.ID, nil, "", "support-key")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("cross-scope revoke should look not found, got %d body=%s", resp.Code, resp.Body.String())
	}
	if body := resp.Body.String(); strings.Contains(body, crossGrant.ID) || strings.Contains(body, westTarget.ID) {
		t.Fatalf("cross-scope revoke leaked inaccessible grant details: %s", body)
	}
	grants, err := repo.ListAccessGrants(t.Context(), store.ManagementScope{})
	if err != nil {
		t.Fatalf("list grants: %v", err)
	}
	if len(grants) != 1 || grants[0].ID != crossGrant.ID || !grants[0].RevokedAt.IsZero() {
		t.Fatalf("cross-scope grant should remain unchanged, got %#v", grants)
	}
}

func TestScopedAdminIdentityCannotOperateDirtyRoutePolicy(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	dirtyPolicy := domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    supportCaller.TenantID,
		WorkspaceID: supportCaller.WorkspaceID,
		Name:        "Dirty cross scope allow",
		CallerID:    supportCaller.ID,
		TargetID:    westTarget.ID,
		RouteType:   "mcp",
		RouteKey:    "tools/call",
		Effect:      domain.RoutePolicyEffectAllow,
		Status:      domain.RoutePolicyStatusEnabled,
		Priority:    100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := repo.CreateRoutePolicy(t.Context(), dirtyPolicy); err != nil {
		t.Fatalf("create dirty route policy: %v", err)
	}

	listResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/route-policies?tenantId=tenant-east&workspaceId=ws-support", nil, "", "support-key")
	policies := decodeData[[]routePolicyResponse](t, listResp)
	if len(policies) != 0 {
		t.Fatalf("dirty route policy should not be visible to scoped admin, got %#v", policies)
	}
	if body := listResp.Body.String(); strings.Contains(body, dirtyPolicy.ID) || strings.Contains(body, westTarget.ID) {
		t.Fatalf("dirty route policy list leaked inaccessible details: %s", body)
	}

	patchResp := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/route-policies/"+dirtyPolicy.ID, map[string]any{
		"name": "Patched dirty route policy",
	}, "", "support-key")
	if patchResp.Code != http.StatusNotFound {
		t.Fatalf("dirty route policy patch should look not found, got %d body=%s", patchResp.Code, patchResp.Body.String())
	}
	if body := patchResp.Body.String(); strings.Contains(body, dirtyPolicy.ID) || strings.Contains(body, westTarget.ID) {
		t.Fatalf("dirty route policy patch leaked inaccessible details: %s", body)
	}

	deleteResp := requestWithAdmin(t, router, http.MethodDelete, "/api/v1/route-policies/"+dirtyPolicy.ID, nil, "", "support-key")
	if deleteResp.Code != http.StatusNotFound {
		t.Fatalf("dirty route policy delete should look not found, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	if body := deleteResp.Body.String(); strings.Contains(body, dirtyPolicy.ID) || strings.Contains(body, westTarget.ID) {
		t.Fatalf("dirty route policy delete leaked inaccessible details: %s", body)
	}
	loaded, ok, err := repo.GetRoutePolicy(t.Context(), dirtyPolicy.ID)
	if err != nil || !ok {
		t.Fatalf("dirty route policy should remain stored: ok=%v err=%v", ok, err)
	}
	if loaded.Name != dirtyPolicy.Name || loaded.Status != domain.RoutePolicyStatusEnabled {
		t.Fatalf("dirty route policy should remain unchanged, got %#v", loaded)
	}
}

func TestTenantAccessProfileHonorsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	supportCapability := createDirectCapabilityWithAction(t, repo, supportTarget.ID, "search_ticket", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	supportEntitlement := createDirectTenantEntitlement(t, repo, "tenant-east", supportTarget.ID, supportCapability.ID, []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)
	supportWorkspace := createDirectWorkspaceAssignment(t, repo, supportEntitlement.ID, "tenant-east", "ws-support", []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)
	createDirectInstanceAssignment(t, repo, supportWorkspace.ID, "tenant-east", "ws-support", supportCaller.ID, []domain.DataScope{{DataDomain: "support", Dataset: "tickets", Region: "us-east"}}, now)

	financeCaller := createDirectAgent(t, repo, "Finance caller", "tenant-east", "ws-finance", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	financeEntitlement := createDirectTenantEntitlement(t, repo, "tenant-east", financeTarget.ID, financeCapability.ID, []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)
	financeWorkspace := createDirectWorkspaceAssignment(t, repo, financeEntitlement.ID, "tenant-east", "ws-finance", []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)
	createDirectInstanceAssignment(t, repo, financeWorkspace.ID, "tenant-east", "ws-finance", financeCaller.ID, []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)

	profile := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/tenant-east/access-profile", nil, "", "support-key")
	if profile.Code != http.StatusOK {
		t.Fatalf("scoped admin should read own access profile, got %d body=%s", profile.Code, profile.Body.String())
	}
	body := profile.Body.String()
	if !strings.Contains(body, supportCapability.ID) || strings.Contains(body, financeCapability.ID) || strings.Contains(body, "export_invoices") || strings.Contains(body, "ws-finance") {
		t.Fatalf("scoped access profile should stay inside workspace boundary, body=%s", body)
	}

	widen := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/tenant-east/access-profile?workspaceId=ws-finance", nil, "", "support-key")
	if widen.Code != http.StatusForbidden {
		t.Fatalf("scoped admin should not widen access profile to another workspace, got %d body=%s", widen.Code, widen.Body.String())
	}
}

func TestTenantAccessProfileRedactsOutOfScopeHydratedObjects(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	financeTarget := createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	entitlement := createDirectTenantEntitlement(t, repo, "tenant-east", financeTarget.ID, financeCapability.ID, []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)
	createDirectWorkspaceAssignment(t, repo, entitlement.ID, "tenant-east", "ws-support", []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}, now)

	resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/tenants/tenant-east/access-profile", nil, "", "support-key")
	if resp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read own access profile, got %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if strings.Contains(body, "Finance MCP") || strings.Contains(body, "export_invoices") {
		t.Fatalf("scoped access profile leaked hydrated out-of-scope object details: %s", body)
	}
}

func TestExplainAccessDecisionHonorsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastCaller := createDirectAgent(t, repo, "East caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	westCapability := createDirectCapabilityWithAction(t, repo, westTarget.ID, "west_tool", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	path := "/api/v1/access-decisions:explain?tenantId=tenant-east&workspaceId=ws-support&callerInstanceId=" + eastCaller.ID + "&subjectId=role:support&targetId=" + westTarget.ID + "&capabilityId=" + westCapability.ID
	resp := requestWithAdmin(t, router, http.MethodGet, path, nil, "", "east-key")
	if resp.Code != http.StatusForbidden || strings.Contains(resp.Body.String(), "West MCP") || strings.Contains(resp.Body.String(), "west_tool") {
		t.Fatalf("scoped admin should not explain outside target metadata, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestScopedAdminIdentityCannotMutateOutsideScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	westCaller := createDirectAgent(t, repo, "West caller", "tenant-west", "ws-finance", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	westCapability := createDirectCapabilityWithAction(t, repo, westTarget.ID, "west_tool", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	allowedAgent := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "East caller",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
		"channelType": "local",
		"status":      "active",
	}, "", "east-key")
	if allowedAgent.Code != http.StatusCreated {
		t.Fatalf("scoped tenant admin should create inside own scope, got %d body=%s", allowedAgent.Code, allowedAgent.Body.String())
	}

	deniedAgent := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "West caller",
		"tenantId":    "tenant-west",
		"workspaceId": "ws-finance",
		"channelType": "local",
	}, "", "east-key")
	if deniedAgent.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create outside tenant, got %d body=%s", deniedAgent.Code, deniedAgent.Body.String())
	}

	deniedTenant := requestWithAdmin(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-west-child",
		"parentTenantId": "tenant-west",
		"name":           "West child",
	}, "", "east-key")
	if deniedTenant.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create tenant outside scope, got %d body=%s", deniedTenant.Code, deniedTenant.Body.String())
	}

	allowedTenant := requestWithAdmin(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-east-child",
		"parentTenantId": "tenant-east",
		"name":           "East child",
	}, "", "east-key")
	if allowedTenant.Code != http.StatusCreated {
		t.Fatalf("scoped tenant admin should create child tenant inside scope, got %d body=%s", allowedTenant.Code, allowedTenant.Body.String())
	}

	updateAgent := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/agents/"+westTarget.ID, map[string]any{
		"status": "disabled",
	}, "", "east-key")
	if updateAgent.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not update outside agent, got %d body=%s", updateAgent.Code, updateAgent.Body.String())
	}

	createKey := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": westCaller.ID,
	}, "", "east-key")
	if createKey.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create outside agent key, got %d body=%s", createKey.Code, createKey.Body.String())
	}

	createGrant := requestWithAdmin(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": westCaller.ID,
		"targetAgentId": westTarget.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, "", "east-key")
	if createGrant.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create outside access grant, got %d body=%s", createGrant.Code, createGrant.Body.String())
	}

	createRoute := requestWithAdmin(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"callerAgentId": westCaller.ID,
		"targetAgentId": westTarget.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "allow",
	}, "", "east-key")
	if createRoute.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not create outside route policy, got %d body=%s", createRoute.Code, createRoute.Body.String())
	}

	updateCapability := requestWithAdmin(t, router, http.MethodPatch, "/api/v1/capabilities/"+westCapability.ID, map[string]any{
		"discoveryStatus": "approved",
	}, "", "east-key")
	if updateCapability.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not update outside capability, got %d body=%s", updateCapability.Code, updateCapability.Body.String())
	}

	refreshCapabilities := requestWithAdmin(t, router, http.MethodPost, "/api/v1/targets/"+westTarget.ID+"/capabilities:refresh", nil, "", "east-key")
	if refreshCapabilities.Code != http.StatusForbidden {
		t.Fatalf("scoped tenant admin should not refresh outside target, got %d body=%s", refreshCapabilities.Code, refreshCapabilities.Body.String())
	}
}

func TestScopedAdminIdentityCannotOperatePermissionPackageOutsideScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	westCaller := createDirectAgent(t, repo, "West caller", "tenant-west", "ws-finance", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, westTarget.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	input := map[string]any{
		"callerInstanceId": westCaller.ID,
		"region":           "us-east",
		"requestText":      "Allow sales read access.",
		"subjectSelector":  "role:sales",
		"targetId":         westTarget.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-west",
		"workspaceId":      "ws-finance",
	}

	for _, tc := range []struct {
		name   string
		path   string
		method string
	}{
		{name: "draft", method: http.MethodPost, path: "/api/v1/permission-packages/drafts"},
		{name: "preflight", method: http.MethodPost, path: "/api/v1/permission-packages:preflight"},
		{name: "apply", method: http.MethodPost, path: "/api/v1/permission-packages:apply"},
		{name: "approval", method: http.MethodPost, path: "/api/v1/permission-packages/approval-requests"},
		{name: "workbench", method: http.MethodPost, path: "/api/v1/permission-packages/workbench:preview"},
	} {
		resp := requestWithAdmin(t, router, tc.method, tc.path, input, "", "east-key")
		if resp.Code != http.StatusForbidden {
			t.Fatalf("%s should reject permission package outside admin scope, got %d body=%s", tc.name, resp.Code, resp.Body.String())
		}
	}
}

func TestScopedAdminIdentityCannotGrantPermissionPackageToOutsideTarget(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	eastCaller := createDirectAgent(t, repo, "East caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	input := map[string]any{
		"callerInstanceId": eastCaller.ID,
		"region":           "us-east",
		"requestText":      "Allow support read access.",
		"subjectSelector":  "role:support",
		"targetId":         financeTarget.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}

	for _, tc := range []struct {
		name   string
		path   string
		method string
		body   any
	}{
		{name: "draft", method: http.MethodPost, path: "/api/v1/permission-packages/drafts", body: input},
		{name: "preflight", method: http.MethodPost, path: "/api/v1/permission-packages:preflight", body: input},
		{name: "apply", method: http.MethodPost, path: "/api/v1/permission-packages:apply", body: input},
		{name: "approval", method: http.MethodPost, path: "/api/v1/permission-packages/approval-requests", body: input},
		{name: "workbench", method: http.MethodPost, path: "/api/v1/permission-packages/workbench:preview", body: input},
		{name: "production readiness", method: http.MethodGet, path: permissionPackageProductionReadinessPath(input, "", "role:support"), body: nil},
		{name: "production report", method: http.MethodGet, path: permissionPackageProductionEvidenceReportPath(input, "", "role:support"), body: nil},
		{name: "tenant entitlement", method: http.MethodPost, path: "/api/v1/tenant-entitlements", body: map[string]any{
			"tenantId":     "tenant-east",
			"targetId":     financeTarget.ID,
			"capabilityId": financeCapability.ID,
			"effect":       "allow",
			"status":       "enabled",
			"priority":     40,
		}},
	} {
		resp := requestWithAdmin(t, router, tc.method, tc.path, tc.body, "", "east-key")
		if resp.Code != http.StatusForbidden || strings.Contains(resp.Body.String(), financeTarget.ID) ||
			strings.Contains(resp.Body.String(), financeCapability.ID) || strings.Contains(resp.Body.String(), "Finance MCP") ||
			strings.Contains(resp.Body.String(), "search_customer") {
			t.Fatalf("%s should reject outside target capability, got %d body=%s", tc.name, resp.Code, resp.Body.String())
		}
	}
}

func TestScopedAdminIdentityCannotUseOutsideAssignments(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeCaller := createDirectAgent(t, repo, "Finance caller", "tenant-east", "ws-finance", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	financeScopes := []domain.DataScope{{DataDomain: "finance", Dataset: "invoices", Region: "eu-west"}}
	financeEntitlement := createDirectTenantEntitlement(t, repo, "tenant-east", financeTarget.ID, financeCapability.ID, financeScopes, now)
	financeWorkspace := createDirectWorkspaceAssignment(t, repo, financeEntitlement.ID, "tenant-east", "ws-finance", financeScopes, now)
	createDirectInstanceAssignment(t, repo, financeWorkspace.ID, "tenant-east", "ws-finance", financeCaller.ID, financeScopes, now)

	workspaceResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": financeEntitlement.ID,
		"workspaceId":         "ws-support",
		"effect":              "allow",
		"status":              "enabled",
		"dataScopes":          financeScopes,
	}, "", "support-key")
	if workspaceResp.Code != http.StatusForbidden || strings.Contains(workspaceResp.Body.String(), "Finance") || strings.Contains(workspaceResp.Body.String(), "export_invoices") {
		t.Fatalf("scoped admin should not attach outside target entitlement, got %d body=%s", workspaceResp.Code, workspaceResp.Body.String())
	}

	instanceResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": financeWorkspace.ID,
		"callerInstanceId":      supportCaller.ID,
		"subjectSelector":       "role:support",
		"effect":                "allow",
		"status":                "enabled",
		"dataScopes":            financeScopes,
	}, "", "support-key")
	if instanceResp.Code != http.StatusForbidden || strings.Contains(instanceResp.Body.String(), "caller instance must match") ||
		strings.Contains(instanceResp.Body.String(), "Finance") || strings.Contains(instanceResp.Body.String(), "export_invoices") {
		t.Fatalf("scoped admin should not inspect outside workspace assignment, got %d body=%s", instanceResp.Code, instanceResp.Body.String())
	}
}

func TestManagementMCPInheritsScopedAdminBoundary(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastAgent := createDirectAgent(t, repo, "East caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	westCaller := createDirectAgent(t, repo, "West caller", "tenant-west", "ws-finance", "local", domain.AgentStatusActive, nil)
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	westCapability := createDirectCapabilityWithAction(t, repo, westTarget.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	listCall := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "scoped-list-agents",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_agents",
			"arguments": map[string]any{},
		},
	}, "", "east-key"))
	var agents []agentResponse
	if err := json.Unmarshal(listCall.Result.StructuredContent, &agents); err != nil {
		t.Fatalf("decode scoped MCP agents: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != eastAgent.ID {
		t.Fatalf("scoped management MCP should only list own agents, got %#v", agents)
	}

	widen := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "scoped-list-west",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_agents",
			"arguments": map[string]any{
				"tenantId":    "tenant-west",
				"workspaceId": "ws-finance",
			},
		},
	}, "", "east-key")
	var widenEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(widen.Body.Bytes(), &widenEnvelope); err != nil {
		t.Fatalf("decode scoped MCP widen envelope: %v body=%s", err, widen.Body.String())
	}
	if widenEnvelope.Error == nil || !strings.Contains(widenEnvelope.Error.Message, "outside authenticated admin scope") {
		t.Fatalf("expected scoped MCP widen rejection, got %#v body=%s", widenEnvelope, widen.Body.String())
	}

	draftOutside := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "scoped-draft-west",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "draft_permission_package",
			"arguments": map[string]any{
				"callerInstanceId": westCaller.ID,
				"region":           "us-east",
				"requestText":      "Allow sales read access.",
				"subjectSelector":  "role:sales",
				"targetId":         westTarget.ID,
				"templateId":       "sales-readonly",
				"tenantId":         "tenant-west",
				"workspaceId":      "ws-finance",
			},
		},
	}, "", "east-key")
	var draftEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(draftOutside.Body.Bytes(), &draftEnvelope); err != nil {
		t.Fatalf("decode scoped MCP draft envelope: %v body=%s", err, draftOutside.Body.String())
	}
	if draftEnvelope.Error == nil || !strings.Contains(draftEnvelope.Error.Message, "outside authenticated admin scope") {
		t.Fatalf("expected scoped MCP draft rejection, got %#v body=%s", draftEnvelope, draftOutside.Body.String())
	}

	explainOutside := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "scoped-explain-west",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "explain_access_decision",
			"arguments": map[string]any{
				"capabilityId":     westCapability.ID,
				"callerInstanceId": eastAgent.ID,
				"subjectId":        "role:support",
				"targetId":         westTarget.ID,
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-support",
			},
		},
	}, "", "east-key")
	var explainEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(explainOutside.Body.Bytes(), &explainEnvelope); err != nil {
		t.Fatalf("decode scoped MCP explain envelope: %v body=%s", err, explainOutside.Body.String())
	}
	if explainEnvelope.Error == nil || !strings.Contains(explainEnvelope.Error.Message, "outside") ||
		strings.Contains(explainOutside.Body.String(), "West MCP") || strings.Contains(explainOutside.Body.String(), "search_customer") {
		t.Fatalf("expected scoped MCP explain rejection without outside metadata, got %#v body=%s", explainEnvelope, explainOutside.Body.String())
	}
}

func TestConsoleAuthSessionReportsDevelopmentMode(t *testing.T) {
	router := newRouter()

	session := decodeData[map[string]any](t, request(t, router, http.MethodGet, "/api/v1/auth/session", nil, ""))
	if session["authenticated"] != true || session["actor"] != "local-dev" || session["requiresLogin"] != false {
		t.Fatalf("dev unauthenticated mode should not require browser login, got %#v", session)
	}
}

func TestAgentRegistryValidation(t *testing.T) {
	router := newRouter()

	created := createAgent(t, router, map[string]any{
		"name":        "Local Test Agent",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	if created.ID == "" || created.WorkspaceID != "ws-1" {
		t.Fatalf("unexpected created agent: %#v", created)
	}

	list := decodeData[[]agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents?workspaceId=ws-1", nil, ""))
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected list response: %#v", list)
	}

	read := decodeData[agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents/"+created.ID, nil, ""))
	if read.Name != "Local Test Agent" {
		t.Fatalf("unexpected get response: %#v", read)
	}

	secretResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Agent",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"apiKey":   "do-not-store-here",
		},
	}, "")
	if secretResp.Code != http.StatusBadRequest {
		t.Fatalf("secret-like channelConfig should fail, got %d", secretResp.Code)
	}

	nestedSecretResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Nested Bad Agent",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"metadata": map[string]any{
				"credentialHeaders": map[string]any{
					"Authorization": "apiToken",
				},
			},
		},
	}, "")
	if nestedSecretResp.Code != http.StatusBadRequest {
		t.Fatalf("nested credentialHeaders should not bypass secret-like channelConfig validation, got %d", nestedSecretResp.Code)
	}

	unsafeResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Unsafe Agent",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "http://127.0.0.1:8080/mcp",
		},
	}, "")
	if unsafeResp.Code != http.StatusBadRequest {
		t.Fatalf("unsafe endpoint should fail, got %d", unsafeResp.Code)
	}

	missingEndpoint := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Missing Endpoint MCP",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
	}, "")
	if missingEndpoint.Code != http.StatusBadRequest {
		t.Fatalf("active MCP without endpoint should fail, got %d", missingEndpoint.Code)
	}

	badEndpointType := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Endpoint Type",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": 123,
		},
	}, "")
	if badEndpointType.Code != http.StatusBadRequest {
		t.Fatalf("non-string endpoint should fail, got %d", badEndpointType.Code)
	}
}

func TestCreateAgentAllowsPrivateEndpointWhenExplicit(t *testing.T) {
	router := newRouterWithPrivateUpstreams()

	resp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Local MCP Target",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "http://127.0.0.1:8080/mcp",
		},
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("private endpoint should be allowed when explicit option is enabled, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestDataPlaneAllowedDeniedTraces(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"name":        "Local Test Agent",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "PMM Tracker MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId":          caller.ID,
		"name":             "unit-key",
		"expiresInSeconds": 900,
	}, ""))
	if key.Key == "" || key.Prefix == "" {
		t.Fatalf("expected one-time key plaintext and prefix: %#v", key)
	}
	targetKey := request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": target.ID,
	}, "")
	if targetKey.Code != http.StatusBadRequest {
		t.Fatalf("target agent key should fail, got %d", targetKey.Code)
	}
	longKey := request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId":          caller.ID,
		"expiresInSeconds": 3601,
	}, "")
	if longKey.Code != http.StatusBadRequest {
		t.Fatalf("agent key ttl above one hour should fail, got %d", longKey.Code)
	}
	keysResp := request(t, router, http.MethodGet, "/api/v1/api-keys", nil, "")
	if strings.Contains(keysResp.Body.String(), "0001-01-01") {
		t.Fatalf("listed keys should omit zero times: %s", keysResp.Body.String())
	}
	keys := decodeData[[]keyResponse](t, keysResp)
	if len(keys) != 1 || keys[0].Key != "" || keys[0].ID != key.ID {
		t.Fatalf("expected listed key without plaintext, got %#v", keys)
	}

	denied := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-1")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected denied without grant, got %d", denied.Code)
	}

	grantResp := request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": " " + caller.ID + " ",
		"targetAgentId": " " + target.ID + " ",
		"routeType":     " mcp ",
		"routeKey":      " tools/call ",
	}, "")
	if grantResp.Code != http.StatusCreated {
		t.Fatalf("grant create failed: %d", grantResp.Code)
	}
	grantsResp := request(t, router, http.MethodGet, "/api/v1/access-grants", nil, "")
	if strings.Contains(grantsResp.Body.String(), "0001-01-01") {
		t.Fatalf("listed grants should omit zero times: %s", grantsResp.Body.String())
	}
	grants := decodeData[[]grantResponse](t, grantsResp)
	if len(grants) != 1 || grants[0].CallerID != caller.ID || grants[0].TargetID != target.ID || grants[0].RouteType != "mcp" || grants[0].RouteKey != "tools/call" {
		t.Fatalf("unexpected grant list: %#v", grants)
	}

	allowed := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-1")
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected allowed with grant, got %d", allowed.Code)
	}

	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-1", nil, ""))
	if len(traces) != 2 {
		t.Fatalf("expected two traces, got %#v", traces)
	}
	if traces[0].Decision != "denied" || traces[1].Decision != "allowed" {
		t.Fatalf("expected denied then allowed trace, got %#v", traces)
	}
	deniedTraces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-1&decision=denied&callerAgentId="+caller.ID+"&targetAgentId="+target.ID, nil, ""))
	if len(deniedTraces) != 1 || deniedTraces[0].Decision != "denied" {
		t.Fatalf("expected one denied trace with filters, got %#v", deniedTraces)
	}
	badDecision := request(t, router, http.MethodGet, "/api/v1/audit/traces?decision=maybe", nil, "")
	if badDecision.Code != http.StatusBadRequest {
		t.Fatalf("bad decision filter should fail, got %d", badDecision.Code)
	}

	revoked := request(t, router, http.MethodDelete, "/api/v1/api-keys/"+key.ID, nil, "")
	if revoked.Code != http.StatusOK {
		t.Fatalf("revoke key failed: %d", revoked.Code)
	}
	afterRevoke := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID, map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if afterRevoke.Code != http.StatusUnauthorized {
		t.Fatalf("revoked key should be unauthorized, got %d", afterRevoke.Code)
	}
}

func TestManagementMutationsRejectDuplicateAgentAndKey(t *testing.T) {
	router := newRouter()
	first := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"tenantId":    "tenant-dup",
		"name":        "Support Agent",
		"workspaceId": "ws-dup",
		"channelType": "local",
		"status":      "active",
	}, "")
	if first.Code != http.StatusCreated {
		t.Fatalf("first agent create failed: %d body=%s", first.Code, first.Body.String())
	}
	created := decodeData[agentResponse](t, first)

	duplicate := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"tenantId":    "tenant-dup",
		"name":        " support agent ",
		"workspaceId": "ws-dup",
		"channelType": "local",
		"status":      "active",
	}, "")
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), "DUPLICATE_RESOURCE_MUTATION") {
		t.Fatalf("duplicate agent should be a conflict, got %d body=%s", duplicate.Code, duplicate.Body.String())
	}

	firstKey := request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId":          created.ID,
		"name":             "console key",
		"expiresInSeconds": 900,
	}, "")
	if firstKey.Code != http.StatusCreated {
		t.Fatalf("first key create failed: %d body=%s", firstKey.Code, firstKey.Body.String())
	}
	duplicateKey := request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId":          created.ID,
		"name":             "console key",
		"expiresInSeconds": 900,
	}, "")
	if duplicateKey.Code != http.StatusConflict || !strings.Contains(duplicateKey.Body.String(), "DUPLICATE_RESOURCE_MUTATION") {
		t.Fatalf("duplicate key should be a conflict, got %d body=%s", duplicateKey.Code, duplicateKey.Body.String())
	}
}

func TestManagementCredentialRotationDuplicateIsNoop(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	agent := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-rotate",
		"name":        "Credential Target",
		"workspaceId": "ws-rotate",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": "Bearer one",
		},
	})
	if agent.CredentialVersion != 1 {
		t.Fatalf("expected initial credential version 1, got %#v", agent)
	}

	duplicate := decodeData[agentResponse](t, request(t, router, http.MethodPost, "/api/v1/agents/"+agent.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{
			"apiToken": "Bearer one",
		},
	}, ""))
	if duplicate.CredentialVersion != agent.CredentialVersion {
		t.Fatalf("duplicate credential rotation should be a no-op, got %#v", duplicate)
	}
	events, err := repo.ListAuditEvents(context.Background(), store.AuditEventFilter{Action: "agent.credentials_rotated"})
	if err != nil {
		t.Fatalf("list rotation audit events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("duplicate credential rotation should not append audit events: %#v", events)
	}

	rotated := decodeData[agentResponse](t, request(t, router, http.MethodPost, "/api/v1/agents/"+agent.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{
			"apiToken": "Bearer two",
		},
	}, ""))
	if rotated.CredentialVersion != agent.CredentialVersion+1 {
		t.Fatalf("changed credential rotation should increment version, got %#v", rotated)
	}
}

func TestRoutePolicyCRUDAndAudit(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-policy",
		"name":        "Policy Caller",
		"workspaceId": "ws-policy",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Policy Target", "tenant-policy", "ws-policy", "mcp", domain.AgentStatusActive, nil)
	crossScopeTarget := createDirectAgent(t, repo, "Cross Scope Target", "tenant-other", "ws-other", "mcp", domain.AgentStatusActive, nil)

	crossScopeResp := request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Cross scope allow",
		"callerAgentId": caller.ID,
		"targetAgentId": crossScopeTarget.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/list",
		"effect":        "allow",
	}, "")
	if crossScopeResp.Code != http.StatusBadRequest {
		t.Fatalf("cross-scope route policy should be rejected, got %d body=%s", crossScopeResp.Code, crossScopeResp.Body.String())
	}
	badRetryAttempts := request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Bad retry attempts",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/list",
		"effect":        "allow",
		"retry": map[string]any{
			"maxAttempts": 5,
		},
	}, "")
	if badRetryAttempts.Code != http.StatusBadRequest {
		t.Fatalf("route policy retry maxAttempts should be rejected, got %d body=%s", badRetryAttempts.Code, badRetryAttempts.Body.String())
	}

	createResp := request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          " Allow tool list ",
		"callerAgentId": " " + caller.ID + " ",
		"targetAgentId": " " + target.ID + " ",
		"routeType":     " mcp ",
		"routeKey":      " tools/list ",
		"effect":        "allow",
		"priority":      25,
	}, "")
	if createResp.Code != http.StatusCreated {
		t.Fatalf("route policy create failed: status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	created := decodeData[routePolicyResponse](t, createResp)
	if created.TenantID != "tenant-policy" || created.WorkspaceID != "ws-policy" ||
		created.Name != "Allow tool list" || created.RouteType != "mcp" || created.RouteKey != "tools/list" ||
		created.Effect != "allow" || created.Status != "enabled" || created.Priority != 25 {
		t.Fatalf("unexpected created route policy: %#v", created)
	}
	duplicate := request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Duplicate allow tool list",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/list",
		"effect":        "allow",
		"priority":      25,
	}, "")
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), "DUPLICATE_RESOURCE_MUTATION") {
		t.Fatalf("duplicate route policy should be a conflict, got %d body=%s", duplicate.Code, duplicate.Body.String())
	}

	list := decodeData[[]routePolicyResponse](t, request(t, router, http.MethodGet, "/api/v1/route-policies?tenantId=tenant-policy&workspaceId=ws-policy", nil, ""))
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("expected scoped route policy list, got %#v", list)
	}

	updated := decodeData[routePolicyResponse](t, request(t, router, http.MethodPatch, "/api/v1/route-policies/"+created.ID, map[string]any{
		"effect":   "deny",
		"name":     "Deny tool list",
		"priority": 40,
		"status":   "enabled",
	}, ""))
	if updated.Effect != "deny" || updated.Name != "Deny tool list" || updated.Priority != 40 || updated.Status != "enabled" {
		t.Fatalf("unexpected updated route policy: %#v", updated)
	}
	badRetryStatus := request(t, router, http.MethodPatch, "/api/v1/route-policies/"+created.ID, map[string]any{
		"retry": map[string]any{
			"statusCodes": []any{429},
		},
	}, "")
	if badRetryStatus.Code != http.StatusBadRequest {
		t.Fatalf("route policy retry statusCodes should be rejected, got %d body=%s", badRetryStatus.Code, badRetryStatus.Body.String())
	}

	disabled := decodeData[routePolicyResponse](t, request(t, router, http.MethodDelete, "/api/v1/route-policies/"+created.ID, nil, ""))
	if disabled.Status != "disabled" {
		t.Fatalf("delete should disable route policy, got %#v", disabled)
	}

	events := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?resourceId="+created.ID, nil, ""))
	if got := auditActions(events); strings.Join(got, ",") != "route_policy.created,route_policy.updated,route_policy.disabled" {
		t.Fatalf("unexpected route policy audit actions: %#v events=%#v", got, events)
	}
}

func TestDirectCrossScopeRoutePolicyIsIgnoredByDataPlane(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-a",
		"name":        "Cross Scope Caller",
		"workspaceId": "ws-a",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Cross Scope Target", "tenant-b", "ws-b", "mcp", domain.AgentStatusActive, nil)
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": caller.ID,
	}, ""))

	now := time.Now().UTC()
	if _, err := repo.CreateRoutePolicy(t.Context(), domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    caller.TenantID,
		WorkspaceID: caller.WorkspaceID,
		Name:        "Direct cross scope allow",
		CallerID:    caller.ID,
		TargetID:    target.ID,
		RouteType:   "mcp",
		RouteKey:    "tools/call",
		Effect:      domain.RoutePolicyEffectAllow,
		Status:      domain.RoutePolicyStatusEnabled,
		Priority:    100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("create direct cross-scope route policy: %v", err)
	}

	policies := decodeData[[]routePolicyResponse](t, request(t, router, http.MethodGet, "/api/v1/route-policies?tenantId=tenant-a&workspaceId=ws-a", nil, ""))
	if len(policies) != 0 {
		t.Fatalf("direct cross-scope route policy should not be visible in scoped management list, got %#v", policies)
	}

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("direct cross-scope route policy should not allow data-plane call, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRoutePolicyCreatePreservesExplicitZeroPriority(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Zero Priority Caller")
	target := createDirectAgent(t, repo, "Zero Priority Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)

	deny := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Medium deny",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "deny",
		"priority":      50,
	}, ""))
	allow := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Lowest allow",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "allow",
		"priority":      0,
	}, ""))
	if deny.Priority != 50 || allow.Priority != 0 {
		t.Fatalf("expected explicit priorities to be preserved, deny=%#v allow=%#v", deny, allow)
	}

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("priority 50 deny should beat explicit priority 0 allow, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRoutePoliciesDriveDataPlaneBeforeLegacyGrants(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Route Policy Caller")
	target := createDirectAgent(t, repo, "Route Policy Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	legacyAllowed := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-route-policy")
	if legacyAllowed.Code != http.StatusOK {
		t.Fatalf("legacy grant should allow before policies, got %d body=%s", legacyAllowed.Code, legacyAllowed.Body.String())
	}

	denyPolicy := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Deny calls",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "deny",
		"priority":      100,
	}, ""))
	denied := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-route-policy")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("route policy deny should override legacy grant, got %d body=%s", denied.Code, denied.Body.String())
	}

	allowPolicy := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Allow calls",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "allow",
		"priority":      200,
	}, ""))
	allowed := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-route-policy")
	if allowed.Code != http.StatusOK {
		t.Fatalf("higher priority allow policy should win, got %d body=%s", allowed.Code, allowed.Body.String())
	}

	request(t, router, http.MethodDelete, "/api/v1/route-policies/"+allowPolicy.ID, nil, "")
	deniedAgain := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-route-policy")
	if deniedAgain.Code != http.StatusForbidden {
		t.Fatalf("disabled allow should reveal deny policy %s, got %d", denyPolicy.ID, deniedAgain.Code)
	}

	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-route-policy", nil, ""))
	reasons := make([]string, 0, len(traces))
	for _, trace := range traces {
		reasons = append(reasons, trace.Reason)
	}
	if !strings.Contains(strings.Join(reasons, ","), "route policy denied") || !strings.Contains(strings.Join(reasons, ","), "route policy allowed") {
		t.Fatalf("expected route policy trace reasons, got %#v", traces)
	}
}

func TestOpenAPIOperationUsesGrant(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"name":        "CI Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Read-only Ops API", "default", "ws-1", "openapi", domain.AgentStatusActive, nil)
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": caller.ID,
	}, ""))
	request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "openapi",
		"routeKey":      "getProjects",
	}, "")

	allowed := request(t, router, http.MethodPost, "/api/v1/openapi/agents/"+target.ID+"/operations/getProjects", map[string]any{}, key.Key)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected allowed operation, got %d", allowed.Code)
	}

	denied := request(t, router, http.MethodPost, "/api/v1/openapi/agents/"+target.ID+"/operations/deleteProject", map[string]any{}, key.Key)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected denied different operation, got %d", denied.Code)
	}

	traversal := request(t, router, http.MethodGet, "/api/v1/openapi/agents/"+target.ID+"/../admin", nil, key.Key)
	if traversal.Code != http.StatusBadRequest {
		t.Fatalf("expected traversal rejection, got %d", traversal.Code)
	}
}

func TestManagementScopeFiltersLists(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)

	inScope := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-a",
		"name":        "In Scope Caller",
		"workspaceId": "ws-a",
		"channelType": "local",
		"status":      "active",
	})
	sameTenantOtherWorkspace := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-a",
		"name":        "Other Workspace",
		"workspaceId": "ws-b",
		"channelType": "local",
		"status":      "active",
	})
	otherTenantSameWorkspace := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-b",
		"name":        "Other Tenant",
		"workspaceId": "ws-a",
		"channelType": "local",
		"status":      "active",
	})

	for _, agent := range []agentResponse{inScope, sameTenantOtherWorkspace, otherTenantSameWorkspace} {
		request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
			"agentId": agent.ID,
			"name":    "key-" + agent.ID,
		}, "")
	}
	includedGrant := decodeData[grantResponse](t, request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": inScope.ID,
		"targetAgentId": inScope.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, ""))
	request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": inScope.ID,
		"targetAgentId": sameTenantOtherWorkspace.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, "")
	request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": sameTenantOtherWorkspace.ID,
		"targetAgentId": otherTenantSameWorkspace.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, "")

	now := time.Now().UTC()
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:        security.NewID("trc"),
		RunID:     "scope-run",
		CallerID:  inScope.ID,
		TargetID:  inScope.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		Decision:  domain.TraceDecisionAllowed,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("append included trace: %v", err)
	}
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:        security.NewID("trc"),
		RunID:     "scope-run",
		CallerID:  inScope.ID,
		TargetID:  sameTenantOtherWorkspace.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		Decision:  domain.TraceDecisionAllowed,
		CreatedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("append caller-only trace: %v", err)
	}
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:        security.NewID("trc"),
		RunID:     "scope-run",
		CallerID:  sameTenantOtherWorkspace.ID,
		TargetID:  otherTenantSameWorkspace.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		Decision:  domain.TraceDecisionAllowed,
		CreatedAt: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("append excluded trace: %v", err)
	}
	crossWorkspaceCapability := createDirectCapabilityWithAction(t, repo, sameTenantOtherWorkspace.ID, "export_finance", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:                security.NewID("trc"),
		RunID:             "scope-run",
		CallerID:          inScope.ID,
		TargetID:          sameTenantOtherWorkspace.ID,
		RouteType:         "mcp",
		RouteKey:          "tools/call",
		TenantID:          inScope.TenantID,
		WorkspaceID:       inScope.WorkspaceID,
		CallerInstanceID:  inScope.ID,
		CapabilityID:      crossWorkspaceCapability.ID,
		CapabilityVersion: crossWorkspaceCapability.Version,
		Decision:          domain.TraceDecisionAllowed,
		CreatedAt:         now.Add(3 * time.Second),
	}); err != nil {
		t.Fatalf("append cross-workspace capability trace: %v", err)
	}

	scopeQuery := "?tenantId=tenant-a&workspaceId=ws-a"
	agents := decodeData[[]agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents"+scopeQuery, nil, ""))
	if len(agents) != 1 || agents[0].ID != inScope.ID {
		t.Fatalf("expected scoped agents to contain only in-scope agent, got %#v", agents)
	}
	allAgents := decodeData[[]agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents", nil, ""))
	if len(allAgents) != 3 {
		t.Fatalf("unscoped agents should preserve old list behavior, got %#v", allAgents)
	}

	keys := decodeData[[]keyResponse](t, request(t, router, http.MethodGet, "/api/v1/api-keys"+scopeQuery, nil, ""))
	if len(keys) != 1 || keys[0].AgentID != inScope.ID {
		t.Fatalf("expected scoped keys to contain only in-scope caller key, got %#v", keys)
	}
	grants := decodeData[[]grantResponse](t, request(t, router, http.MethodGet, "/api/v1/access-grants"+scopeQuery, nil, ""))
	if len(grants) != 1 || grants[0].ID != includedGrant.ID {
		t.Fatalf("expected scoped grants to require caller and target in scope, got %#v", grants)
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces"+scopeQuery+"&runId=scope-run", nil, ""))
	if len(traces) != 1 || traces[0].CallerID != inScope.ID {
		t.Fatalf("expected scoped traces to require caller and target in scope, got %#v", traces)
	}
}

func TestTenantHierarchyAPIsAndScopedManagementLists(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)

	root := decodeData[tenantResponse](t, request(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":     "tenant-root",
		"name":   "Root Tenant",
		"status": "active",
	}, ""))
	child := decodeData[tenantResponse](t, request(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-child",
		"parentTenantId": root.ID,
		"name":           "Child Tenant",
		"status":         "active",
	}, ""))
	grandchild := decodeData[tenantResponse](t, request(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-grandchild",
		"parentTenantId": child.ID,
		"name":           "Grandchild Tenant",
		"status":         "active",
	}, ""))
	if root.Level != 1 || child.Level != 2 || grandchild.Level != 3 {
		t.Fatalf("unexpected tenant levels: root=%#v child=%#v grandchild=%#v", root, child, grandchild)
	}
	level4 := request(t, router, http.MethodPost, "/api/v1/tenants", map[string]any{
		"id":             "tenant-level-4",
		"parentTenantId": grandchild.ID,
		"name":           "Too Deep",
		"status":         "active",
	}, "")
	if level4.Code != http.StatusBadRequest {
		t.Fatalf("fourth-level tenant should fail, got %d body=%s", level4.Code, level4.Body.String())
	}

	createAgent(t, router, map[string]any{"tenantId": root.ID, "workspaceId": "ws-a", "name": "Root Agent", "channelType": "local", "status": "active"})
	createAgent(t, router, map[string]any{"tenantId": child.ID, "workspaceId": "ws-a", "name": "Child Agent", "channelType": "local", "status": "active"})
	createAgent(t, router, map[string]any{"tenantId": grandchild.ID, "workspaceId": "ws-a", "name": "Grandchild Agent", "channelType": "local", "status": "active"})
	createAgent(t, router, map[string]any{"tenantId": "tenant-unrelated", "workspaceId": "ws-a", "name": "Unrelated Agent", "channelType": "local", "status": "active"})

	tenants := decodeData[[]tenantResponse](t, request(t, router, http.MethodGet, "/api/v1/tenants?tenantId="+root.ID, nil, ""))
	if got := tenantResponseIDs(tenants); !reflect.DeepEqual(got, []string{root.ID, child.ID, grandchild.ID}) {
		t.Fatalf("tenant subtree = %#v", got)
	}
	children := decodeData[[]tenantResponse](t, request(t, router, http.MethodGet, "/api/v1/tenants?parentTenantId="+root.ID, nil, ""))
	if got := tenantResponseIDs(children); !reflect.DeepEqual(got, []string{child.ID}) {
		t.Fatalf("direct children = %#v", got)
	}
	agents := decodeData[[]agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents?tenantId="+root.ID+"&workspaceId=ws-a", nil, ""))
	if got := agentResponseTenantIDs(agents); !reflect.DeepEqual(got, []string{root.ID, child.ID, grandchild.ID}) {
		t.Fatalf("scoped agent tenants = %#v", got)
	}
}

func TestTenantHierarchyAllowsParentTargetEntitlementToDescendant(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()

	for _, body := range []map[string]any{
		{"id": "tenant-root", "name": "Root", "status": "active"},
		{"id": "tenant-child", "parentTenantId": "tenant-root", "name": "Child", "status": "active"},
		{"id": "tenant-unrelated", "name": "Unrelated", "status": "active"},
	} {
		resp := request(t, router, http.MethodPost, "/api/v1/tenants", body, "")
		if resp.Code != http.StatusCreated {
			t.Fatalf("create tenant failed: status=%d body=%s", resp.Code, resp.Body.String())
		}
	}
	target := createDirectAgent(t, repo, "Root MCP", "tenant-root", "ws-root", "mcp", domain.AgentStatusActive, nil)
	capability := domain.Capability{
		ID:              security.NewID("cap"),
		TargetID:        target.ID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             "search_customer",
		DisplayName:     "search_customer",
		Action:          domain.CapabilityActionRead,
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       domain.CapabilityRiskLow,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(t.Context(), capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}

	entitlement := request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-child",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, "")
	if entitlement.Code != http.StatusCreated {
		t.Fatalf("descendant entitlement should be allowed, got %d body=%s", entitlement.Code, entitlement.Body.String())
	}
	unrelated := request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-unrelated",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, "")
	if unrelated.Code != http.StatusBadRequest {
		t.Fatalf("unrelated entitlement should be rejected, got %d body=%s", unrelated.Code, unrelated.Body.String())
	}
}

func TestRuntimeMetricsSummarizeDataPlaneTraces(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Runtime Metrics Caller")
	target := createDirectAgent(t, repo, "Runtime Metrics Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)

	denied := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected denied call before grant, got %d body=%s", denied.Code, denied.Body.String())
	}
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")
	allowed := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected allowed call after grant, got %d body=%s", allowed.Code, allowed.Body.String())
	}

	metricsResp := request(t, router, http.MethodGet, "/api/v1/metrics/runtime?workspaceId=ws-1", nil, "")
	if metricsResp.Code != http.StatusOK {
		t.Fatalf("expected runtime metrics endpoint, got %d body=%s", metricsResp.Code, metricsResp.Body.String())
	}
	metrics := decodeData[[]metricResponse](t, metricsResp)
	if got := metricByID(t, metrics, "gateway_calls_total").Value; got != 2 {
		t.Fatalf("gateway_calls_total = %d, want 2; metrics=%#v", got, metrics)
	}
	if got := metricByID(t, metrics, "allowed_rate").Value; got != 50 {
		t.Fatalf("allowed_rate = %d, want 50; metrics=%#v", got, metrics)
	}
	if got := metricByID(t, metrics, "upstream_error_rate").Value; got != 0 {
		t.Fatalf("upstream_error_rate = %d, want 0; metrics=%#v", got, metrics)
	}
	if got := metricByID(t, metrics, "avg_latency_ms").Value; got != 0 {
		t.Fatalf("avg_latency_ms = %d, want 0 for local stub calls; metrics=%#v", got, metrics)
	}
}

func TestDisableAgentBlocksExistingKey(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"name":        "Disposable Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Stub MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": caller.ID,
	}, ""))
	request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, "")

	beforeDisable := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if beforeDisable.Code != http.StatusOK {
		t.Fatalf("expected existing key to work before disable, got %d", beforeDisable.Code)
	}

	disabled := decodeData[agentResponse](t, request(t, router, http.MethodDelete, "/api/v1/agents/"+caller.ID, nil, ""))
	if disabled.Status != "disabled" {
		t.Fatalf("expected disabled agent response, got %#v", disabled)
	}
	afterDisable := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if afterDisable.Code != http.StatusUnauthorized {
		t.Fatalf("disabled caller key should be unauthorized, got %d", afterDisable.Code)
	}
}

func TestDisabledTargetDeniesLaterCalls(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Target Disable Caller")
	target := createDirectAgent(t, repo, "Disable Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	disabled := decodeData[agentResponse](t, request(t, router, http.MethodDelete, "/api/v1/agents/"+target.ID, nil, ""))
	if disabled.Status != "disabled" {
		t.Fatalf("expected disabled target response, got %#v", disabled)
	}
	resp := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-disabled-target")
	if resp.Code != http.StatusForbidden {
		t.Fatalf("disabled target should deny data-plane call, got %d body=%s", resp.Code, resp.Body.String())
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-disabled-target", nil, ""))
	if len(traces) != 1 || traces[0].Decision != "denied" || traces[0].Reason != "target agent is not active" {
		t.Fatalf("expected denied disabled-target trace, got %#v", traces)
	}
}

func TestRevokeAccessGrantDeniesLaterCalls(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"name":        "Revocable Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	target := createDirectAgent(t, repo, "Revocable Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": caller.ID,
	}, ""))
	grant := decodeData[grantResponse](t, request(t, router, http.MethodPost, "/api/v1/access-grants", map[string]any{
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, ""))

	allowed := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected allowed before grant revoke, got %d", allowed.Code)
	}

	revoked := request(t, router, http.MethodDelete, "/api/v1/access-grants/"+grant.ID, nil, "")
	if revoked.Code != http.StatusOK {
		t.Fatalf("expected revoke grant success, got %d body=%s", revoked.Code, revoked.Body.String())
	}
	denied := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("revoked grant should deny later call, got %d", denied.Code)
	}
}

func TestMCPMethodRouteKeyUsesJSONRPCMethod(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "MCP Method Caller")
	target := createDirectAgent(t, repo, "Method MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/list")

	allowed := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "list",
		"method":  "tools/list",
	}, key.Key, "run-method-policy")
	if allowed.Code != http.StatusOK {
		t.Fatalf("tools/list grant should allow tools/list call, got %d body=%s", allowed.Code, allowed.Body.String())
	}
	denied := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "call",
		"method":  "tools/call",
	}, key.Key, "run-method-policy")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("tools/list grant should not allow tools/call call, got %d body=%s", denied.Code, denied.Body.String())
	}
	upperCaseDenied := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "list-upper",
		"method":  "TOOLS/LIST",
	}, key.Key, "run-method-policy")
	if upperCaseDenied.Code != http.StatusForbidden {
		t.Fatalf("MCP route keys should be case-sensitive, got %d body=%s", upperCaseDenied.Code, upperCaseDenied.Body.String())
	}

	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-method-policy", nil, ""))
	if len(traces) != 3 || traces[0].RouteKey != "tools/list" || traces[1].RouteKey != "tools/call" || traces[2].RouteKey != "TOOLS/LIST" {
		t.Fatalf("expected traces with actual MCP methods, got %#v", traces)
	}
}

func TestMCPInvalidMethodReturnsValidationErrorWithoutTrace(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Bad MCP Caller")
	target := createDirectAgent(t, repo, "Bad MCP Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	grantRoute(t, router, caller.ID, target.ID, "mcp", "")

	resp := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "bad",
		"method":  "",
	}, key.Key, "run-bad-method")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("missing MCP method should be validation error, got %d body=%s", resp.Code, resp.Body.String())
	}
	var env apiEnvelope
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "VALIDATION_FAILED" {
		t.Fatalf("expected validation error, got %#v", env)
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-bad-method", nil, ""))
	if len(traces) != 0 {
		t.Fatalf("invalid MCP body should not record trace, got %#v", traces)
	}
}

func TestMCPProxyRelaysAllowedUpstreamResponse(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/mcp" {
			t.Fatalf("unexpected upstream request %s %s", r.Method, r.URL.String())
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upstream body: %v", err)
		}
		if !strings.Contains(string(body), `"method":"tools/call"`) {
			t.Fatalf("upstream body did not receive original request: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"upstream":true}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Proxy Caller")
	target := createDirectAgent(t, repo, "Proxy MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected upstream status, got %d body=%s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected upstream content-type, got %q", got)
	}
	if strings.TrimSpace(resp.Body.String()) != `{"upstream":true}` {
		t.Fatalf("expected raw upstream body, got %s", resp.Body.String())
	}
}

func TestUpstreamProxyForwardsConfiguredHeaders(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-AgentHarbor-Tenant"); got != "default" {
			t.Fatalf("expected configured tenant header, got %q", got)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"headers":true}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Header Proxy Caller")
	target := createDirectAgent(t, repo, "Header MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
		"headers": map[string]any{
			"X-AgentHarbor-Tenant": "default",
		},
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected upstream status, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestUpstreamProxyInjectsCredentialHeaders(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer credential-redaction-secret" {
			t.Fatalf("expected credential Authorization header, got %q", got)
		}
		if got := r.Header.Get("X-AgentHarbor-Tenant"); got != "default" {
			t.Fatalf("expected configured non-secret header, got %q", got)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"credentials":true}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Credential Header Caller")
	target := createDirectAgentWithCredentials(t, repo, "Credential Header MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
		"headers": map[string]any{
			"X-AgentHarbor-Tenant": "default",
		},
		"credentialHeaders": map[string]any{
			"Authorization": "apiToken",
		},
	}, map[string]string{
		"apiToken": "Bearer credential-redaction-secret",
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected upstream status, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestMCPToolCallInjectsReservedAgentHarborContext(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	var capturedContext string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := r.Header.Values("X-AgentHarbor-Context")
		if len(values) != 1 {
			t.Fatalf("expected one Agent Harbor context header, got %#v", values)
		}
		capturedContext = values[0]
		if capturedContext == "caller-spoof" || capturedContext == "configured-spoof" || capturedContext == "credential-spoof" {
			t.Fatalf("reserved context header was not generated by Agent Harbor: %q", capturedContext)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"context":true}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Context Header Caller")
	target := createDirectAgentWithCredentials(t, repo, "Context Header MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL,
		"headers": map[string]any{
			"X-AgentHarbor-Context": "configured-spoof",
		},
		"credentialHeaders": map[string]any{
			"X-AgentHarbor-Context": "reservedContext",
		},
	}, map[string]string{
		"reservedContext": "credential-spoof",
	})
	now := time.Now().UTC()
	capability := domain.Capability{
		ID:          security.NewID("cap"),
		TargetID:    target.ID,
		Type:        domain.CapabilityTypeMCPTool,
		Key:         "search_customer",
		DisplayName: "search_customer",
		Action:      domain.CapabilityActionRead,
		DataScopes: []domain.DataScope{{
			DataDomain:   "crm",
			TenantFilter: "tenant_id = 'default'",
		}},
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       domain.CapabilityRiskLow,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(t.Context(), capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}
	entitlement, err := repo.CreateTenantEntitlement(t.Context(), domain.TenantEntitlement{
		ID:           security.NewID("ent"),
		TenantID:     caller.TenantID,
		TargetID:     target.ID,
		CapabilityID: capability.ID,
		Effect:       domain.PolicyEffectAllow,
		Status:       domain.PolicyStatusEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	workspaceAssignment, err := repo.CreateWorkspaceAssignment(t.Context(), domain.WorkspaceAssignment{
		ID:                  security.NewID("wsa"),
		TenantEntitlementID: entitlement.ID,
		TenantID:            caller.TenantID,
		WorkspaceID:         caller.WorkspaceID,
		Effect:              domain.PolicyEffectAllow,
		DataScopes:          []domain.DataScope{{Region: "us-east"}},
		Status:              domain.PolicyStatusEnabled,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		t.Fatalf("create workspace assignment: %v", err)
	}
	instanceAssignment, err := repo.CreateInstanceAssignment(t.Context(), domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:*",
		Effect:                domain.PolicyEffectAllow,
		DataScopes:            []domain.DataScope{{Table: "accounts"}},
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err != nil {
		t.Fatalf("create instance assignment: %v", err)
	}

	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      "call-context",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "search_customer",
			"arguments": map[string]any{"query": "Acme"},
		},
	}); err != nil {
		t.Fatalf("encode request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key.Key)
	req.Header.Set("X-AgentHarbor-Context", "caller-spoof")
	req.Header.Set("X-AgentHarbor-Subject-Id", "user:ops")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected upstream status, got %d body=%s", rec.Code, rec.Body.String())
	}

	decoded, err := base64.RawURLEncoding.DecodeString(capturedContext)
	if err != nil {
		t.Fatalf("decode context header: %v", err)
	}
	var payload struct {
		SchemaVersion    string             `json:"schemaVersion"`
		PlatformID       string             `json:"platformId"`
		TenantID         string             `json:"tenantId"`
		WorkspaceID      string             `json:"workspaceId"`
		TargetID         string             `json:"targetId"`
		CallerInstanceID string             `json:"callerInstanceId"`
		CallerSubject    string             `json:"callerSubject"`
		CapabilityID     string             `json:"capabilityId"`
		CapabilityKey    string             `json:"capabilityKey"`
		ToolName         string             `json:"toolName"`
		DataScopes       []domain.DataScope `json:"dataScopes"`
	}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		t.Fatalf("unmarshal context header: %v", err)
	}
	wantScopes := []domain.DataScope{{
		DataDomain:   "crm",
		Table:        "accounts",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'default'",
	}}
	if payload.SchemaVersion != "2026-06-01" || payload.PlatformID != "default" || payload.TenantID != caller.TenantID || payload.WorkspaceID != caller.WorkspaceID || payload.TargetID != target.ID {
		t.Fatalf("unexpected context identity: %#v", payload)
	}
	if payload.CallerInstanceID != caller.ID || payload.CallerSubject != "user:ops" || payload.CapabilityID != capability.ID || payload.CapabilityKey != capability.Key || payload.ToolName != "search_customer" {
		t.Fatalf("unexpected context capability fields: %#v", payload)
	}
	if !reflect.DeepEqual(payload.DataScopes, wantScopes) {
		t.Fatalf("context data scopes = %#v, want %#v", payload.DataScopes, wantScopes)
	}
	if instanceAssignment.ID == "" {
		t.Fatalf("instance assignment was not created")
	}
}

func TestUpstreamProxyRetriesRetryableStatusThenReturnsSuccess(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	attempts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upstream body: %v", err)
		}
		if !strings.Contains(string(body), `"method":"tools/call"`) {
			t.Fatalf("upstream body did not receive original request on attempt %d: %s", attempts, string(body))
		}
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"temporary":true}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Retry Caller")
	target := createDirectAgent(t, repo, "Retry MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
		"retry": map[string]any{
			"maxAttempts": 2,
			"backoffMs":   0,
			"statusCodes": []any{503},
		},
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected successful retried upstream status, got %d body=%s", resp.Code, resp.Body.String())
	}
	if attempts != 2 {
		t.Fatalf("expected two upstream attempts, got %d", attempts)
	}
	if got := resp.Header().Get("X-AgentHarbor-Upstream-Attempts"); got != "2" {
		t.Fatalf("expected attempts header 2, got %q", got)
	}
	if strings.TrimSpace(resp.Body.String()) != `{"ok":true}` {
		t.Fatalf("expected final upstream body, got %s", resp.Body.String())
	}
}

func TestRoutePolicyRetryOverridesTargetRetry(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	attempts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"temporary":true}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"policyRetry":true}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Policy Retry Caller")
	target := createDirectAgent(t, repo, "Policy Retry MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
		"retry": map[string]any{
			"maxAttempts": 1,
		},
	})
	policy := decodeData[routePolicyResponse](t, request(t, router, http.MethodPost, "/api/v1/route-policies", map[string]any{
		"name":          "Allow calls with retry",
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
		"effect":        "allow",
		"priority":      100,
		"retry": map[string]any{
			"maxAttempts": 2,
			"backoffMs":   0,
			"statusCodes": []any{503},
		},
	}, ""))
	if policy.Retry == nil || policy.Retry.MaxAttempts != 2 || policy.Retry.BackoffMs != 0 || len(policy.Retry.StatusCodes) != 1 || policy.Retry.StatusCodes[0] != 503 {
		t.Fatalf("expected normalized retry on route policy response, got %#v", policy.Retry)
	}

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("policy retry should recover upstream status, got %d body=%s", resp.Code, resp.Body.String())
	}
	if attempts != 2 {
		t.Fatalf("expected two upstream attempts from policy retry, got %d", attempts)
	}
	if got := resp.Header().Get("X-AgentHarbor-Upstream-Attempts"); got != "2" {
		t.Fatalf("expected attempts header 2, got %q", got)
	}
	cleared := decodeData[routePolicyResponse](t, request(t, router, http.MethodPatch, "/api/v1/route-policies/"+policy.ID, map[string]any{
		"retry": nil,
	}, ""))
	if cleared.Retry != nil {
		t.Fatalf("expected retry override to clear, got %#v", cleared.Retry)
	}
}

func TestProxyTraceMetricsRecordAttemptsStatusAndDuration(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	attempts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"temporary":true}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Trace Metrics Caller")
	target := createDirectAgent(t, repo, "Trace Metrics MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
		"retry": map[string]any{
			"maxAttempts": 2,
			"backoffMs":   0,
			"statusCodes": []any{503},
		},
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "metrics-proxy")
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected upstream success after retry, got %d body=%s", resp.Code, resp.Body.String())
	}

	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=metrics-proxy", nil, ""))
	if len(traces) != 1 {
		t.Fatalf("expected one allowed trace, got %#v", traces)
	}
	trace := traces[0]
	if trace.UpstreamAttempts != 2 {
		t.Fatalf("upstreamAttempts = %d, want 2; trace=%#v", trace.UpstreamAttempts, trace)
	}
	if trace.UpstreamStatus != http.StatusAccepted {
		t.Fatalf("upstreamStatus = %d, want 202; trace=%#v", trace.UpstreamStatus, trace)
	}
	if trace.UpstreamError != "" {
		t.Fatalf("upstreamError = %q, want empty; trace=%#v", trace.UpstreamError, trace)
	}
	if trace.DurationMs <= 0 {
		t.Fatalf("durationMs = %d, want positive; trace=%#v", trace.DurationMs, trace)
	}
}

func TestProxySuccessDoesNotFailWhenTraceAppendFails(t *testing.T) {
	memory := store.NewMemory()
	repo := &failingAllowedTraceRepository{Repository: memory}
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Trace Failure Caller")
	target := createDirectAgent(t, repo, "Trace Failure MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "trace-fail-after-upstream")
	if resp.Code != http.StatusAccepted {
		t.Fatalf("upstream success should not be converted to trace failure, got %d body=%s", resp.Code, resp.Body.String())
	}
	if strings.TrimSpace(resp.Body.String()) != `{"accepted":true}` {
		t.Fatalf("expected upstream body despite trace failure, got %s", resp.Body.String())
	}
}

func TestUpstreamProxyReturnsLastRetryableStatusAfterAttemptsExhausted(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	attempts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"attempt":` + strconv.Itoa(attempts) + `}`))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Retry Exhaust Caller")
	target := createDirectAgent(t, repo, "Retry Exhaust MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
		"retry": map[string]any{
			"maxAttempts": 3,
			"backoffMs":   0,
			"statusCodes": []any{503},
		},
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected final retryable status, got %d body=%s", resp.Code, resp.Body.String())
	}
	if attempts != 3 {
		t.Fatalf("expected three upstream attempts, got %d", attempts)
	}
	if got := resp.Header().Get("X-AgentHarbor-Upstream-Attempts"); got != "3" {
		t.Fatalf("expected attempts header 3, got %q", got)
	}
	if strings.TrimSpace(resp.Body.String()) != `{"attempt":3}` {
		t.Fatalf("expected final upstream body, got %s", resp.Body.String())
	}
}

func TestProxyRejectsOversizedBufferedBody(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("oversized request should not reach upstream")
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Oversized Body Caller")
	target := createDirectAgent(t, repo, "Oversized Body MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	payload := `{"jsonrpc":"2.0","method":"tools/call","params":{"blob":"` + strings.Repeat("x", 4<<20) + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key.Key)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected oversized body to return 413, got %d body=%s", rec.Code, rec.Body.String())
	}
	var env apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("expected PAYLOAD_TOO_LARGE, got %#v", env)
	}
}

func TestProxyUpstreamDNSFailureReturnsDNSError(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	originalTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, &net.DNSError{Err: "no such host", Name: "nonexistent.invalid", IsNotFound: true}
	})
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	caller, key := createLocalCallerWithKey(t, router, "DNS Proxy Caller")
	target := createDirectAgent(t, repo, "DNS Proxy MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": "http://nonexistent.invalid/mcp",
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected upstream DNS failure to return 502, got %d body=%s", resp.Code, resp.Body.String())
	}
	var env apiEnvelope
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "UPSTREAM_DNS_ERROR" {
		t.Fatalf("expected UPSTREAM_DNS_ERROR, got %#v", env)
	}
	if got := resp.Header().Get("X-AgentHarbor-Upstream-Attempts"); got != "1" {
		t.Fatalf("expected attempts header 1, got %q", got)
	}
}

func TestProxyDoesNotRetryCanceledContext(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	attempts := 0
	originalTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return nil, context.Canceled
	})
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	caller, key := createLocalCallerWithKey(t, router, "Canceled Proxy Caller")
	target := createDirectAgent(t, repo, "Canceled Proxy MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": "https://api.example.com/mcp",
		"retry": map[string]any{
			"maxAttempts": 3,
			"backoffMs":   0,
			"statusCodes": []any{502, 503, 504},
		},
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected canceled upstream request to return 502, got %d body=%s", resp.Code, resp.Body.String())
	}
	if attempts != 1 {
		t.Fatalf("expected canceled request not to retry, got %d attempts", attempts)
	}
	if got := resp.Header().Get("X-AgentHarbor-Upstream-Attempts"); got != "1" {
		t.Fatalf("expected attempts header 1, got %q", got)
	}
	var env apiEnvelope
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "UPSTREAM_ERROR" {
		t.Fatalf("expected UPSTREAM_ERROR, got %#v", env)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestAgentRejectsSecretLikeHeaders(t *testing.T) {
	router := newRouter()
	for _, headerName := range []string{"Authorization", "Cookie", "X-Api-Key"} {
		resp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
			"name":        "Secret Header MCP",
			"workspaceId": "ws-1",
			"channelType": "mcp",
			"status":      "active",
			"channelConfig": map[string]any{
				"endpoint": "https://api.example.com/mcp",
				"headers": map[string]any{
					headerName: "should-not-live-here",
				},
			},
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("secret-like configured header %s should fail, got %d body=%s", headerName, resp.Code, resp.Body.String())
		}
	}
}

func TestAgentCredentialsAreAcceptedAndRedacted(t *testing.T) {
	router := newRouter()
	secret := "Bearer credential-redaction-secret"

	resp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Credentialed MCP",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": secret,
		},
	}, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("credentialed agent create failed: status=%d body=%s", resp.Code, resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), secret) || strings.Contains(resp.Body.String(), "credentials") {
		t.Fatalf("create response leaked credentials: %s", resp.Body.String())
	}
	created := decodeData[agentResponse](t, resp)

	read := request(t, router, http.MethodGet, "/api/v1/agents/"+created.ID, nil, "")
	if read.Code != http.StatusOK {
		t.Fatalf("get credentialed agent failed: status=%d body=%s", read.Code, read.Body.String())
	}
	if strings.Contains(read.Body.String(), secret) || strings.Contains(read.Body.String(), "credentials") {
		t.Fatalf("get response leaked credentials: %s", read.Body.String())
	}

	list := request(t, router, http.MethodGet, "/api/v1/agents?workspaceId=ws-1", nil, "")
	if list.Code != http.StatusOK {
		t.Fatalf("list credentialed agent failed: status=%d body=%s", list.Code, list.Body.String())
	}
	if strings.Contains(list.Body.String(), secret) || strings.Contains(list.Body.String(), "credentials") {
		t.Fatalf("list response leaked credentials: %s", list.Body.String())
	}
}

func TestPatchAgentUpdatesMutableFieldsAndValidatesConfig(t *testing.T) {
	router := newRouter()
	secret := "Bearer patch-secret"
	createdResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Patchable MCP",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "draft",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": secret,
		},
	}, "")
	if createdResp.Code != http.StatusCreated {
		t.Fatalf("create patchable agent failed: status=%d body=%s", createdResp.Code, createdResp.Body.String())
	}
	created := decodeData[agentResponse](t, createdResp)

	updatedResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"name":        "Patched MCP",
		"description": "updated through partial patch",
		"ownerId":     "platform-team",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api-updated.example.com/mcp",
			"headers": map[string]any{
				"X-AgentHarbor-Tenant": "default",
			},
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
	}, "")
	if updatedResp.Code != http.StatusOK {
		t.Fatalf("patch agent should succeed, got %d body=%s", updatedResp.Code, updatedResp.Body.String())
	}
	if strings.Contains(updatedResp.Body.String(), secret) || strings.Contains(updatedResp.Body.String(), "credentials") {
		t.Fatalf("patch response leaked credentials: %s", updatedResp.Body.String())
	}
	updated := decodeData[agentResponse](t, updatedResp)
	if updated.Name != "Patched MCP" || updated.Description != "updated through partial patch" || updated.OwnerID != "platform-team" || updated.Status != "active" {
		t.Fatalf("unexpected patched agent metadata: %#v", updated)
	}
	if updated.ChannelConfig["endpoint"] != "https://api-updated.example.com/mcp" {
		t.Fatalf("unexpected patched channel config: %#v", updated.ChannelConfig)
	}

	secretConfig := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"channelConfig": map[string]any{
			"endpoint":      "https://api.example.com/mcp",
			"Authorization": "do-not-store-here",
		},
	}, "")
	if secretConfig.Code != http.StatusBadRequest {
		t.Fatalf("secret-like patch channelConfig should fail, got %d body=%s", secretConfig.Code, secretConfig.Body.String())
	}

	arraySecretConfig := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"metadata": []any{
				map[string]any{"authorization": "Bearer should-not-echo"},
			},
		},
	}, "")
	if arraySecretConfig.Code != http.StatusBadRequest {
		t.Fatalf("secret-like patch channelConfig inside arrays should fail, got %d body=%s", arraySecretConfig.Code, arraySecretConfig.Body.String())
	}

	unsafeEndpoint := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"channelConfig": map[string]any{
			"endpoint": "http://127.0.0.1:8080/mcp",
		},
	}, "")
	if unsafeEndpoint.Code != http.StatusBadRequest {
		t.Fatalf("unsafe patch endpoint should fail, got %d body=%s", unsafeEndpoint.Code, unsafeEndpoint.Body.String())
	}
}

func TestRotateAgentCredentialsTakesEffectOnNextProxyCall(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	seenAuthorizations := []string{}
	originalTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seenAuthorizations = append(seenAuthorizations, r.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	})
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	caller, key := createLocalCallerWithKey(t, router, "Rotate Caller")
	target := createDirectAgentWithCredentials(t, repo, "Rotate MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": "https://api.example.com/mcp",
		"credentialHeaders": map[string]any{
			"Authorization": "apiToken",
		},
	}, map[string]string{
		"apiToken": "Bearer old-token",
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	beforeRotate := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if beforeRotate.Code != http.StatusAccepted {
		t.Fatalf("expected proxy call before rotate, got %d body=%s", beforeRotate.Code, beforeRotate.Body.String())
	}

	rotateResp := request(t, router, http.MethodPost, "/api/v1/agents/"+target.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{
			"apiToken": "Bearer new-token",
		},
	}, "")
	if rotateResp.Code != http.StatusOK {
		t.Fatalf("rotate credentials should succeed, got %d body=%s", rotateResp.Code, rotateResp.Body.String())
	}
	if strings.Contains(rotateResp.Body.String(), "Bearer new-token") || strings.Contains(rotateResp.Body.String(), "credentials") {
		t.Fatalf("rotate response leaked credentials: %s", rotateResp.Body.String())
	}

	afterRotate := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if afterRotate.Code != http.StatusAccepted {
		t.Fatalf("expected proxy call after rotate, got %d body=%s", afterRotate.Code, afterRotate.Body.String())
	}
	if len(seenAuthorizations) != 2 || seenAuthorizations[0] != "Bearer old-token" || seenAuthorizations[1] != "Bearer new-token" {
		t.Fatalf("unexpected Authorization headers after rotation: %#v", seenAuthorizations)
	}
}

func TestManagementAuditEventsRecordAgentLifecycleWithoutSecrets(t *testing.T) {
	router := newRouter()
	oldSecret := "Bearer audit-old-secret"
	newSecret := "Bearer audit-new-secret"

	createResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Audited MCP",
		"tenantId":    "tenant-audit",
		"workspaceId": "ws-audit",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": oldSecret,
		},
	}, "")
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create audited agent failed: status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	if strings.Contains(createResp.Body.String(), oldSecret) || strings.Contains(createResp.Body.String(), "credentials") {
		t.Fatalf("create response leaked credentials: %s", createResp.Body.String())
	}
	created := decodeData[agentResponse](t, createResp)
	if created.CredentialVersion != 1 {
		t.Fatalf("credentialed agent should start at version 1, got %#v", created)
	}

	updateResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+created.ID, map[string]any{
		"name":   "Audited MCP Updated",
		"status": "draft",
	}, "")
	if updateResp.Code != http.StatusOK {
		t.Fatalf("patch audited agent failed: status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	updated := decodeData[agentResponse](t, updateResp)
	if updated.CredentialVersion != 1 {
		t.Fatalf("metadata update should not change credential version: %#v", updated)
	}

	emptyRotateResp := request(t, router, http.MethodPost, "/api/v1/agents/"+created.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{},
	}, "")
	if emptyRotateResp.Code != http.StatusBadRequest {
		t.Fatalf("empty credential rotation should fail, got %d body=%s", emptyRotateResp.Code, emptyRotateResp.Body.String())
	}

	rotateResp := request(t, router, http.MethodPost, "/api/v1/agents/"+created.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{
			"apiToken": newSecret,
		},
	}, "")
	if rotateResp.Code != http.StatusOK {
		t.Fatalf("rotate audited credentials failed: status=%d body=%s", rotateResp.Code, rotateResp.Body.String())
	}
	if strings.Contains(rotateResp.Body.String(), oldSecret) || strings.Contains(rotateResp.Body.String(), newSecret) || strings.Contains(rotateResp.Body.String(), "credentials") {
		t.Fatalf("rotate response leaked credentials: %s", rotateResp.Body.String())
	}
	rotated := decodeData[agentResponse](t, rotateResp)
	if rotated.CredentialVersion != 2 {
		t.Fatalf("credential rotation should increment version to 2, got %#v", rotated)
	}

	eventsResp := request(t, router, http.MethodGet, "/api/v1/audit/events?resourceId="+created.ID, nil, "")
	if eventsResp.Code != http.StatusOK {
		t.Fatalf("list audit events failed: status=%d body=%s", eventsResp.Code, eventsResp.Body.String())
	}
	if strings.Contains(eventsResp.Body.String(), oldSecret) || strings.Contains(eventsResp.Body.String(), newSecret) {
		t.Fatalf("audit events leaked credential values: %s", eventsResp.Body.String())
	}
	events := decodeData[[]auditEventResponse](t, eventsResp)
	if got := auditActions(events); strings.Join(got, ",") != "agent.created,agent.updated,agent.credentials_rotated" {
		t.Fatalf("unexpected audit actions: %#v events=%#v", got, events)
	}
	rotation := events[2]
	if rotation.Metadata["credentialVersion"] != float64(2) {
		t.Fatalf("rotation audit event should include new credential version, got %#v", rotation.Metadata)
	}
	keys, ok := rotation.Metadata["credentialKeys"].([]any)
	if !ok || len(keys) != 1 || keys[0] != "apiToken" {
		t.Fatalf("rotation audit event should include credential key names only, got %#v", rotation.Metadata)
	}
}

func TestManagementAuditFailureBlocksAgentCreateAndUpdate(t *testing.T) {
	base := store.NewMemory()
	router := newRouterWithRepo(&failingAuditedAgentRepository{Repository: base})

	createResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Audit Required Agent",
		"tenantId":    "tenant-audit-failure",
		"workspaceId": "ws-audit-failure",
		"channelType": "local",
		"status":      "active",
	}, "")
	if createResp.Code != http.StatusInternalServerError {
		t.Fatalf("create should fail when audit persistence fails: status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	agents, err := base.ListAgents(t.Context(), store.AgentFilter{ManagementScope: store.ManagementScope{
		TenantID:    "tenant-audit-failure",
		WorkspaceID: "ws-audit-failure",
	}})
	if err != nil {
		t.Fatalf("list agents after failed create: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("failed audited create should not persist agent: %#v", agents)
	}

	existing := domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      "tenant-audit-failure",
		WorkspaceID:   "ws-audit-failure",
		Name:          "Original Agent",
		ChannelType:   "local",
		ChannelConfig: map[string]any{},
		Status:        domain.AgentStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if _, err := base.CreateAgent(t.Context(), existing); err != nil {
		t.Fatalf("seed existing agent: %v", err)
	}

	updateResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+existing.ID, map[string]any{
		"name": "Updated Agent",
	}, "")
	if updateResp.Code != http.StatusInternalServerError {
		t.Fatalf("update should fail when audit persistence fails: status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	after, ok, err := base.GetAgent(t.Context(), existing.ID)
	if err != nil {
		t.Fatalf("get agent after failed update: %v", err)
	}
	if !ok {
		t.Fatalf("seeded agent disappeared after failed update")
	}
	if after.Name != "Original Agent" {
		t.Fatalf("failed audited update should keep previous name, got %q", after.Name)
	}
}

func TestRotateAgentCredentialsRejectsEmptyCredentialBag(t *testing.T) {
	router := newRouter()
	agent := createAgent(t, router, map[string]any{
		"name":        "No Credential Local",
		"workspaceId": "ws-audit",
		"channelType": "local",
		"status":      "active",
	})

	resp := request(t, router, http.MethodPost, "/api/v1/agents/"+agent.ID+"/credentials:rotate", map[string]any{
		"credentials": map[string]any{},
	}, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("empty credential rotation should fail, got %d body=%s", resp.Code, resp.Body.String())
	}
	read := decodeData[agentResponse](t, request(t, router, http.MethodGet, "/api/v1/agents/"+agent.ID, nil, ""))
	if read.CredentialVersion != 0 {
		t.Fatalf("rejected empty rotation should not change credential version, got %#v", read)
	}
}

func TestManagementAuditEventsFilterByScopeActionAndResource(t *testing.T) {
	router := newRouter()
	first := createAgent(t, router, map[string]any{
		"name":        "First Audited Agent",
		"tenantId":    "tenant-filter",
		"workspaceId": "ws-filter-a",
		"channelType": "local",
		"status":      "active",
	})
	second := createAgent(t, router, map[string]any{
		"name":        "Second Audited Agent",
		"tenantId":    "tenant-filter",
		"workspaceId": "ws-filter-b",
		"channelType": "local",
		"status":      "active",
	})
	patchResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+second.ID, map[string]any{
		"description": "only second agent was updated",
	}, "")
	if patchResp.Code != http.StatusOK {
		t.Fatalf("patch second audited agent failed: status=%d body=%s", patchResp.Code, patchResp.Body.String())
	}

	workspaceEvents := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?tenantId=tenant-filter&workspaceId=ws-filter-a", nil, ""))
	if len(workspaceEvents) != 1 || workspaceEvents[0].ResourceID != first.ID || workspaceEvents[0].Action != "agent.created" {
		t.Fatalf("workspace filter should return only first create event, got %#v", workspaceEvents)
	}

	updateEvents := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?action=agent.updated&resourceType=agent", nil, ""))
	if len(updateEvents) != 1 || updateEvents[0].ResourceID != second.ID {
		t.Fatalf("action/resourceType filter should return second update event, got %#v", updateEvents)
	}

	secondEvents := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?resourceId="+second.ID, nil, ""))
	if got := auditActions(secondEvents); strings.Join(got, ",") != "agent.created,agent.updated" {
		t.Fatalf("resourceId filter should return second lifecycle events, got %#v events=%#v", got, secondEvents)
	}

	limitedEvents := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?tenantId=tenant-filter&limit=1", nil, ""))
	if len(limitedEvents) != 1 {
		t.Fatalf("limit=1 should return one audit event, got %#v", limitedEvents)
	}
}

func TestAgentRejectsInvalidProxyConfig(t *testing.T) {
	router := newRouter()
	badHeaderValue := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Header Value",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"headers": map[string]any{
				"X-AgentHarbor-Tenant": 123,
			},
		},
	}, "")
	if badHeaderValue.Code != http.StatusBadRequest {
		t.Fatalf("non-string configured header should fail, got %d body=%s", badHeaderValue.Code, badHeaderValue.Body.String())
	}

	badHeaderName := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Header Name",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"headers": map[string]any{
				"X-Bad\nHeader": "value",
			},
		},
	}, "")
	if badHeaderName.Code != http.StatusBadRequest {
		t.Fatalf("invalid configured header name should fail, got %d body=%s", badHeaderName.Code, badHeaderName.Body.String())
	}

	badCredentialValue := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Credential Value",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		"credentials": map[string]any{
			"apiToken": "Bearer good\nX-Bad: injected",
		},
	}, "")
	if badCredentialValue.Code != http.StatusBadRequest {
		t.Fatalf("credential header value with newline should fail, got %d body=%s", badCredentialValue.Code, badCredentialValue.Body.String())
	}

	badCredentialKey := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Credential Key",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "Bearer copied-secret",
			},
		},
		"credentials": map[string]any{
			"Bearer copied-secret": "Bearer credential-redaction-secret",
		},
	}, "")
	if badCredentialKey.Code != http.StatusBadRequest {
		t.Fatalf("credential key that looks like secret material should fail, got %d body=%s", badCredentialKey.Code, badCredentialKey.Body.String())
	}

	badTimeout := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Timeout",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"timeoutMs": 30001,
		},
	}, "")
	if badTimeout.Code != http.StatusBadRequest {
		t.Fatalf("out-of-range timeout should fail, got %d body=%s", badTimeout.Code, badTimeout.Body.String())
	}

	missingCredential := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Missing Credential",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
	}, "")
	if missingCredential.Code != http.StatusBadRequest {
		t.Fatalf("missing credential header reference should fail, got %d body=%s", missingCredential.Code, missingCredential.Body.String())
	}

	badRetryAttempts := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Retry Attempts",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"retry": map[string]any{
				"maxAttempts": 5,
			},
		},
	}, "")
	if badRetryAttempts.Code != http.StatusBadRequest {
		t.Fatalf("out-of-range retry maxAttempts should fail, got %d body=%s", badRetryAttempts.Code, badRetryAttempts.Body.String())
	}

	badRetryStatus := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Bad Retry Status",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
		"channelConfig": map[string]any{
			"retry": map[string]any{
				"statusCodes": []any{429},
			},
		},
	}, "")
	if badRetryStatus.Code != http.StatusBadRequest {
		t.Fatalf("non-5xx retry status should fail, got %d body=%s", badRetryStatus.Code, badRetryStatus.Body.String())
	}
}

func TestUpstreamTimeoutReturnsGatewayTimeout(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Timeout Caller")
	target := createDirectAgent(t, repo, "Slow MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint":  upstream.URL + "/mcp",
		"timeoutMs": 1,
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-timeout")
	if resp.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected upstream timeout to return 504, got %d body=%s", resp.Code, resp.Body.String())
	}
	var env apiEnvelope
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "UPSTREAM_TIMEOUT" {
		t.Fatalf("expected UPSTREAM_TIMEOUT, got %#v", env)
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-timeout", nil, ""))
	if len(traces) != 1 || traces[0].Decision != "allowed" {
		t.Fatalf("expected allowed trace before timeout, got %#v", traces)
	}
}

func TestOpenAPIProxyRelaysRelativePath(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/base/projects/42" || r.URL.RawQuery != "include=stats" {
			t.Fatalf("unexpected upstream request %s %s", r.Method, r.URL.String())
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upstream body: %v", err)
		}
		if !strings.Contains(string(body), `"name":"nexus"`) {
			t.Fatalf("upstream body did not receive original request: %s", string(body))
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("updated"))
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "OpenAPI Proxy Caller")
	target := createDirectAgent(t, repo, "Proxy OpenAPI", "default", "ws-1", "openapi", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/base",
	})
	grantRoute(t, router, caller.ID, target.ID, "openapi", "projects/42")

	resp := request(t, router, http.MethodPut, "/api/v1/openapi/agents/"+target.ID+"/projects/42?include=stats", map[string]any{
		"name": "nexus",
	}, key.Key)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected upstream status, got %d body=%s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("expected upstream content-type, got %q", got)
	}
	if strings.TrimSpace(resp.Body.String()) != "updated" {
		t.Fatalf("expected raw upstream body, got %s", resp.Body.String())
	}
}

func TestProxyUpstreamConnectFailureRecordsTraceAndReturnsConnectError(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("closed upstream should not receive request")
	}))
	endpoint := upstream.URL + "/mcp"
	upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "Failing Proxy Caller")
	target := createDirectAgent(t, repo, "Failing Proxy MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": endpoint,
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key, "run-upstream-fail")
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected upstream failure to return 502, got %d body=%s", resp.Code, resp.Body.String())
	}
	var env apiEnvelope
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "UPSTREAM_CONNECT_ERROR" {
		t.Fatalf("expected UPSTREAM_CONNECT_ERROR, got %#v", env)
	}
	if got := resp.Header().Get("X-AgentHarbor-Upstream-Attempts"); got != "1" {
		t.Fatalf("expected attempts header 1, got %q", got)
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-upstream-fail", nil, ""))
	if len(traces) != 1 || traces[0].Decision != "allowed" || traces[0].TargetID != target.ID {
		t.Fatalf("expected allowed trace recorded before proxy failure, got %#v", traces)
	}
	if traces[0].UpstreamAttempts != 1 || traces[0].UpstreamError != "UPSTREAM_CONNECT_ERROR" || traces[0].DurationMs <= 0 {
		t.Fatalf("expected proxy failure metrics on trace, got %#v", traces[0])
	}
}

func TestProxyUpstreamTLSFailureReturnsTLSError(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("untrusted TLS upstream should not receive request")
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "TLS Proxy Caller")
	target := createDirectAgent(t, repo, "TLS Proxy MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/mcp",
	})
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")

	resp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected upstream TLS failure to return 502, got %d body=%s", resp.Code, resp.Body.String())
	}
	var env apiEnvelope
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "UPSTREAM_TLS_ERROR" {
		t.Fatalf("expected UPSTREAM_TLS_ERROR, got %#v", env)
	}
	if got := resp.Header().Get("X-AgentHarbor-Upstream-Attempts"); got != "1" {
		t.Fatalf("expected attempts header 1, got %q", got)
	}
}

func TestMCPCapabilityDiscoveryAndAssignmentManagement(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"capability-discovery","result":{"tools":[{"name":"search_customer","description":"Search customers","inputSchema":{"type":"object"}},{"name":"export_contracts","description":"Export contracts","inputSchema":{"type":"object"}}]}}`))
	}))
	defer upstream.Close()

	now := time.Now().UTC()
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "Capability Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Capability MCP", "tenant-a", "ws-sales", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL,
	})

	capabilities := decodeData[[]capabilityResponse](t, request(t, router, http.MethodPost, "/api/v1/targets/"+target.ID+"/capabilities:refresh", nil, ""))
	if len(capabilities) != 2 {
		t.Fatalf("expected two discovered capabilities, got %#v", capabilities)
	}
	search := capabilityByKey(t, capabilities, "search_customer")
	if search.DiscoveryStatus != "pending_review" || search.Action != "read" {
		t.Fatalf("unexpected search capability defaults: %#v", search)
	}
	export := capabilityByKey(t, capabilities, "export_contracts")
	if export.DiscoveryStatus != "pending_review" || export.Action != "export" || export.RiskLevel != "high" {
		t.Fatalf("unexpected export capability defaults: %#v", export)
	}

	approved := decodeData[capabilityResponse](t, request(t, router, http.MethodPatch, "/api/v1/capabilities/"+search.ID, map[string]any{
		"discoveryStatus": "approved",
	}, ""))
	if approved.DiscoveryStatus != "approved" {
		t.Fatalf("capability should be approved, got %#v", approved)
	}
	entitlement := decodeData[tenantEntitlementResponse](t, request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-a",
		"targetId":     target.ID,
		"capabilityId": search.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, ""))
	if entitlement.TenantID != "tenant-a" || entitlement.CapabilityID != search.ID {
		t.Fatalf("unexpected entitlement: %#v", entitlement)
	}
	workspaceAssignment := decodeData[workspaceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": entitlement.ID,
		"workspaceId":         "ws-sales",
		"effect":              "allow",
		"status":              "enabled",
	}, ""))
	if workspaceAssignment.WorkspaceID != "ws-sales" {
		t.Fatalf("unexpected workspace assignment: %#v", workspaceAssignment)
	}
	for _, subjectSelector := range []string{"", " ", "*"} {
		resp := request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
			"workspaceAssignmentId": workspaceAssignment.ID,
			"callerInstanceId":      caller.ID,
			"subjectSelector":       subjectSelector,
			"effect":                "allow",
			"status":                "enabled",
		}, "")
		if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "subjectSelector") {
			t.Fatalf("subjectSelector %q should be rejected, status=%d body=%s", subjectSelector, resp.Code, resp.Body.String())
		}
	}
	instanceAssignment := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": workspaceAssignment.ID,
		"callerInstanceId":      caller.ID,
		"subjectSelector":       "user:sales-*",
		"effect":                "allow",
		"status":                "enabled",
	}, ""))
	if instanceAssignment.CallerInstanceID != caller.ID {
		t.Fatalf("unexpected instance assignment: %#v", instanceAssignment)
	}
}

func TestPermissionPackageDraftAndApplyManagement(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-biz", "tenant-root", "Business tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-biz", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	export := createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	templates := decodeData[[]permissionPackageTemplateResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/templates", nil, ""))
	if len(templates) == 0 || templates[0].ID != "sales-readonly" || templates[0].Version != 1 {
		t.Fatalf("expected sales-readonly template first, got %#v", templates)
	}
	accessSubjects := decodeData[[]domain.PermissionPackageAccessSubject](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/access-subjects", nil, ""))
	if len(accessSubjects) == 0 || accessSubjects[0].ID != "role:support-agent" || accessSubjects[0].SubjectSelector != "user:support-*" {
		t.Fatalf("expected support-agent access subject first, got %#v", accessSubjects)
	}
	var registeredMember domain.PermissionPackageAccessSubject
	for _, subject := range accessSubjects {
		if subject.ID == "member:support-002" {
			registeredMember = subject
			break
		}
	}
	if registeredMember.SubjectSelector != "user:support-002" || registeredMember.WorkspaceID != "ws-permission-package-approval" {
		t.Fatalf("expected registered support member with workspace scope, got %#v", registeredMember)
	}

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读，禁止导出合同。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	draft := decodeData[permissionPackageDraftResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, ""))
	if !draft.Readiness.CanApply || draft.Template.ID != "sales-readonly" || draft.Template.Version != 1 {
		t.Fatalf("expected applicable sales-readonly draft, got %#v", draft)
	}
	if draft.PolicyGate.Decision != "allow" || !draft.PolicyGate.CanApplyDirectly || draft.PolicyGate.PolicyVersion != 1 {
		t.Fatalf("expected direct-apply policy gate, got %#v", draft.PolicyGate)
	}
	directApprovalResp := request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "")
	if directApprovalResp.Code != http.StatusBadRequest {
		t.Fatalf("direct-apply package should not create approval request, status=%d body=%s", directApprovalResp.Code, directApprovalResp.Body.String())
	}
	if len(draft.AllowedCapabilities) != 1 || draft.AllowedCapabilities[0].ID != search.ID {
		t.Fatalf("expected search_customer allowed, got %#v", draft.AllowedCapabilities)
	}
	if len(draft.BlockedCapabilities) != 1 || draft.BlockedCapabilities[0].ID != export.ID {
		t.Fatalf("expected export_contracts blocked, got %#v", draft.BlockedCapabilities)
	}
	if len(draft.DataScopes) != 1 || draft.DataScopes[0].Region != "华东" || draft.DataScopes[0].TenantFilter != "tenant_id = 'tenant-east'" {
		t.Fatalf("unexpected draft data scopes: %#v", draft.DataScopes)
	}
	if len(draft.SimulationRows) != 4 || draft.SimulationRows[0].ExpectedDecision != "allow" || draft.SimulationRows[1].ExpectedDecision != "deny" {
		t.Fatalf("unexpected simulation rows: %#v", draft.SimulationRows)
	}

	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, ""))
	if len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != search.ID {
		t.Fatalf("expected one applied tenant entitlement for search, got %#v", applied.TenantEntitlements)
	}
	if len(applied.WorkspaceAssignments) != 1 || applied.WorkspaceAssignments[0].WorkspaceID != "ws-sales" {
		t.Fatalf("expected one workspace assignment, got %#v", applied.WorkspaceAssignments)
	}
	if len(applied.InstanceAssignments) != 1 || applied.InstanceAssignments[0].CallerInstanceID != caller.ID {
		t.Fatalf("expected one caller instance assignment, got %#v", applied.InstanceAssignments)
	}
	if applied.Application == nil || applied.Application.DraftID != applied.Draft.ID ||
		applied.Application.TemplateID != "sales-readonly" || applied.Application.TemplateVersion != 1 ||
		applied.Application.TenantID != "tenant-east" || applied.Application.WorkspaceID != "ws-sales" ||
		applied.Application.TargetID != target.ID || applied.Application.CallerInstanceID != caller.ID ||
		len(applied.Application.AllowedCapabilityIDs) != 1 || applied.Application.AllowedCapabilityIDs[0] != search.ID ||
		len(applied.Application.TenantEntitlementIDs) != 1 || applied.Application.TenantEntitlementIDs[0] != applied.TenantEntitlements[0].ID ||
		len(applied.Application.WorkspaceAssignmentIDs) != 1 || applied.Application.WorkspaceAssignmentIDs[0] != applied.WorkspaceAssignments[0].ID ||
		len(applied.Application.InstanceAssignmentIDs) != 1 || applied.Application.InstanceAssignmentIDs[0] != applied.InstanceAssignments[0].ID ||
		len(applied.Application.DataScopes) != 1 || applied.Application.DataScopes[0].Region != "华东" {
		t.Fatalf("unexpected permission package application: %#v", applied.Application)
	}
	applications := decodeData[[]permissionPackageApplicationResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?tenantId=tenant-root&workspaceId=ws-sales&templateId=sales-readonly&targetId="+target.ID+"&callerInstanceId="+caller.ID+"&limit=1", nil, ""))
	if len(applications) != 1 || applications[0].ID != applied.Application.ID || applications[0].DraftID != applied.Draft.ID {
		t.Fatalf("expected listed application record, got %#v", applications)
	}
	health := decodeData[permissionPackageApplicationHealthResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-root&workspaceId=ws-sales&templateId=sales-readonly&targetId="+target.ID+"&callerInstanceId="+caller.ID+"&limit=10", nil, ""))
	if health.Summary.Total != 1 || health.Summary.Ready != 1 || health.Summary.Drifted != 0 || health.Summary.NeedsReview != 0 {
		t.Fatalf("expected ready application health summary, got %#v", health.Summary)
	}
	if len(health.Applications) != 1 {
		t.Fatalf("expected one application health row, got %#v", health.Applications)
	}
	healthRow := health.Applications[0]
	if healthRow.Application.ID != applied.Application.ID || healthRow.Application.DraftID != applied.Draft.ID ||
		healthRow.Status != "ready" || healthRow.CreatedObjectCount != 3 || healthRow.ActiveObjectCount != 3 ||
		healthRow.MissingObjectCount != 0 || !healthRow.RollbackReady {
		t.Fatalf("unexpected ready application health row: %#v", healthRow)
	}
	if healthRow.BlockerCodes == nil || len(healthRow.BlockerCodes) != 0 {
		t.Fatalf("expected ready health blocker codes to encode as an empty array, got %#v", healthRow.BlockerCodes)
	}
	impact := decodeData[permissionPackageApplicationImpactResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+applied.Application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales", nil, ""))
	if impact.Application.ID != applied.Application.ID || impact.Application.DraftID != applied.Draft.ID {
		t.Fatalf("expected impact for applied application, got %#v", impact.Application)
	}
	if impact.Rehearsal != nil {
		t.Fatalf("real impact should not include rehearsal metadata, got %#v", impact.Rehearsal)
	}
	if impact.Summary.CreatedObjectCount != 3 || impact.Summary.ActiveObjectCount != 3 ||
		impact.Summary.MissingObjectCount != 0 || !impact.Summary.RollbackReady {
		t.Fatalf("unexpected application impact summary: %#v", impact.Summary)
	}
	if !impactObjectsContain(impact.CreatedObjects, "tenant_entitlement", applied.TenantEntitlements[0].ID, "enabled", "disable") ||
		!impactObjectsContain(impact.CreatedObjects, "workspace_assignment", applied.WorkspaceAssignments[0].ID, "enabled", "disable") ||
		!impactObjectsContain(impact.CreatedObjects, "instance_assignment", applied.InstanceAssignments[0].ID, "enabled", "disable") {
		t.Fatalf("expected created grant objects in impact review, got %#v", impact.CreatedObjects)
	}
	if len(impact.CapabilityReviews) != 1 || impact.CapabilityReviews[0].ID != search.ID ||
		impact.CapabilityReviews[0].CurrentStatus != string(domain.CapabilityDiscoveryApproved) ||
		impact.CapabilityReviews[0].RollbackAction != "manual_review" {
		t.Fatalf("expected capability manual review row, got %#v", impact.CapabilityReviews)
	}
	if !impact.RollbackReview.Ready || len(impact.RollbackReview.Blockers) != 0 || len(impact.RollbackReview.Steps) == 0 {
		t.Fatalf("expected ready rollback review steps, got %#v", impact.RollbackReview)
	}
	if impact.RollbackReview.Blockers == nil {
		t.Fatalf("expected rollback blockers to encode as an empty array, got nil")
	}
	if impact.RollbackReview.BlockerCodes == nil || len(impact.RollbackReview.BlockerCodes) != 0 {
		t.Fatalf("expected rollback blocker codes to encode as an empty array, got %#v", impact.RollbackReview.BlockerCodes)
	}
	if impact.RemediationPlan.ExecutionMode != "read_only" || !impact.RemediationPlan.Ready {
		t.Fatalf("expected ready read-only remediation plan, got %#v", impact.RemediationPlan)
	}
	if impact.RemediationPlan.Blockers == nil || len(impact.RemediationPlan.Blockers) != 0 {
		t.Fatalf("expected remediation blockers to encode as an empty array, got %#v", impact.RemediationPlan.Blockers)
	}
	if impact.RemediationPlan.BlockerCodes == nil || len(impact.RemediationPlan.BlockerCodes) != 0 {
		t.Fatalf("expected remediation blocker codes to encode as an empty array, got %#v", impact.RemediationPlan.BlockerCodes)
	}
	if len(impact.RemediationPlan.Actions) == 0 {
		t.Fatalf("expected remediation actions, got %#v", impact.RemediationPlan)
	}
	for _, action := range impact.RemediationPlan.Actions {
		if !action.ReadOnly {
			t.Fatalf("expected all remediation actions to be read-only, got %#v in %#v", action, impact.RemediationPlan.Actions)
		}
	}
	if !remediationActionsContain(impact.RemediationPlan.Actions, "capability", search.ID, "manual_review") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "instance_assignment", applied.InstanceAssignments[0].ID, "disable") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "workspace_assignment", applied.WorkspaceAssignments[0].ID, "disable") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "tenant_entitlement", applied.TenantEntitlements[0].ID, "disable") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "access_decision", applied.Application.ID, "verify") {
		t.Fatalf("expected complete remediation action sequence, got %#v", impact.RemediationPlan.Actions)
	}
	instanceOrder := remediationActionOrder(impact.RemediationPlan.Actions, "instance_assignment", applied.InstanceAssignments[0].ID, "disable")
	workspaceOrder := remediationActionOrder(impact.RemediationPlan.Actions, "workspace_assignment", applied.WorkspaceAssignments[0].ID, "disable")
	tenantOrder := remediationActionOrder(impact.RemediationPlan.Actions, "tenant_entitlement", applied.TenantEntitlements[0].ID, "disable")
	if instanceOrder == 0 || workspaceOrder == 0 || tenantOrder == 0 || !(instanceOrder < workspaceOrder && workspaceOrder < tenantOrder) {
		t.Fatalf("expected instance before workspace before tenant remediation order, got actions %#v", impact.RemediationPlan.Actions)
	}
	driftRehearsal := decodeData[permissionPackageApplicationImpactResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+applied.Application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales&rehearsal=grant_drift", nil, ""))
	if driftRehearsal.Rehearsal == nil || !driftRehearsal.Rehearsal.Enabled || driftRehearsal.Rehearsal.Scenario != "grant_drift" {
		t.Fatalf("expected grant_drift rehearsal metadata, got %#v", driftRehearsal.Rehearsal)
	}
	if driftRehearsal.Summary.CreatedObjectCount != 3 || driftRehearsal.Summary.ActiveObjectCount >= driftRehearsal.Summary.CreatedObjectCount ||
		driftRehearsal.Summary.MissingObjectCount != 1 || driftRehearsal.Summary.RollbackReady {
		t.Fatalf("unexpected rehearsal impact summary: %#v", driftRehearsal.Summary)
	}
	for _, code := range []string{"missing_created_objects", "inactive_created_objects"} {
		if !containsString(driftRehearsal.RollbackReview.BlockerCodes, code) {
			t.Fatalf("expected rehearsal rollback blocker code %q, got %#v", code, driftRehearsal.RollbackReview.BlockerCodes)
		}
		if !containsString(driftRehearsal.RemediationPlan.BlockerCodes, code) {
			t.Fatalf("expected rehearsal remediation blocker code %q, got %#v", code, driftRehearsal.RemediationPlan.BlockerCodes)
		}
	}
	if driftRehearsal.RollbackReview.Ready || driftRehearsal.RemediationPlan.Ready {
		t.Fatalf("rehearsal drift should not be ready: rollback=%#v remediation=%#v", driftRehearsal.RollbackReview, driftRehearsal.RemediationPlan)
	}
	if !remediationActionsContain(driftRehearsal.RemediationPlan.Actions, "workspace_assignment", applied.WorkspaceAssignments[0].ID, "investigate") ||
		!remediationActionsContain(driftRehearsal.RemediationPlan.Actions, "instance_assignment", applied.InstanceAssignments[0].ID, "investigate") ||
		!remediationActionsContain(driftRehearsal.RemediationPlan.Actions, "access_decision", applied.Application.ID, "verify") {
		t.Fatalf("expected rehearsal investigate and verify actions, got %#v", driftRehearsal.RemediationPlan.Actions)
	}
	for _, action := range driftRehearsal.RemediationPlan.Actions {
		if !action.ReadOnly {
			t.Fatalf("expected rehearsal actions to be read-only, got %#v in %#v", action, driftRehearsal.RemediationPlan.Actions)
		}
	}
	realImpactAfterRehearsal := decodeData[permissionPackageApplicationImpactResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+applied.Application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales", nil, ""))
	if realImpactAfterRehearsal.Rehearsal != nil {
		t.Fatalf("real impact after rehearsal should not include rehearsal metadata, got %#v", realImpactAfterRehearsal.Rehearsal)
	}
	if realImpactAfterRehearsal.Summary.CreatedObjectCount != 3 || realImpactAfterRehearsal.Summary.ActiveObjectCount != 3 ||
		realImpactAfterRehearsal.Summary.MissingObjectCount != 0 || !realImpactAfterRehearsal.Summary.RollbackReady {
		t.Fatalf("rehearsal should not persist grant drift, got real summary %#v", realImpactAfterRehearsal.Summary)
	}
	badRehearsal := request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+applied.Application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales&rehearsal=unknown", nil, "")
	if badRehearsal.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid rehearsal to fail, status=%d body=%s", badRehearsal.Code, badRehearsal.Body.String())
	}
	badHealthLimit := request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?limit=0", nil, "")
	if badHealthLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid application health limit to fail, status=%d body=%s", badHealthLimit.Code, badHealthLimit.Body.String())
	}
	badLimit := request(t, router, http.MethodGet, "/api/v1/permission-packages/applications?limit=0", nil, "")
	if badLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid application limit to fail, status=%d body=%s", badLimit.Code, badLimit.Body.String())
	}
	updated, ok, err := repo.GetCapability(t.Context(), search.ID)
	if err != nil || !ok {
		t.Fatalf("get updated capability: ok=%v err=%v", ok, err)
	}
	if updated.DiscoveryStatus != domain.CapabilityDiscoveryApproved || len(updated.DataScopes) != 1 || updated.DataScopes[0].Region != "华东" {
		t.Fatalf("expected package apply to approve and scope capability, got %#v", updated)
	}
	events := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?action=permission_package.applied", nil, ""))
	if len(events) != 1 || events[0].TenantID != "tenant-east" || events[0].ResourceID != applied.Application.ID ||
		events[0].Metadata["applicationId"] != applied.Application.ID || events[0].Metadata["draftId"] != applied.Draft.ID ||
		events[0].Metadata["templateVersion"] != float64(1) {
		t.Fatalf("expected permission_package.applied audit event, got %#v", events)
	}
}

func TestPermissionPackagePreflightDirectApplyIsReadOnly(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读，禁止导出合同。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	preflight := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if !preflight.Summary.CanApply || preflight.Summary.BlockingCount != 0 || preflight.Summary.WarningCount != 0 ||
		preflight.Summary.PlannedCapabilityCount != 1 || preflight.Summary.PlannedTenantEntitlementCount != 1 ||
		preflight.Summary.PlannedWorkspaceAssignmentCount != 1 || preflight.Summary.PlannedInstanceAssignmentCount != 1 ||
		preflight.Summary.RequiresApproval || preflight.Summary.ApprovalReady {
		t.Fatalf("expected ready direct preflight summary, got %#v", preflight.Summary)
	}
	if !permissionPackagePreflightHasCheck(preflight.Checks, "draft_ready", "passed") ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "policy_gate", "passed") ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "data_scope_fit", "passed") ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "planned_changes", "info") {
		t.Fatalf("expected passed preflight checks, got %#v", preflight.Checks)
	}
	if len(preflight.Planned.Capabilities) != 1 || preflight.Planned.Capabilities[0].ID != search.ID ||
		len(preflight.Planned.TenantEntitlements) != 1 || preflight.Planned.TenantEntitlements[0].CapabilityID != search.ID ||
		len(preflight.Planned.WorkspaceAssignments) != 1 || preflight.Planned.WorkspaceAssignments[0].WorkspaceID != "ws-sales" ||
		len(preflight.Planned.InstanceAssignments) != 1 || preflight.Planned.InstanceAssignments[0].CallerInstanceID != caller.ID {
		t.Fatalf("unexpected planned changes: %#v", preflight.Planned)
	}
	entitlements, err := repo.ListTenantEntitlements(t.Context(), store.EntitlementFilter{})
	if err != nil {
		t.Fatalf("list entitlements: %v", err)
	}
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.applied"})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(entitlements) != 0 || len(applications) != 0 || len(events) != 0 {
		t.Fatalf("preflight must not write records: entitlements=%#v applications=%#v events=%#v", entitlements, applications, events)
	}
}

func TestPermissionPackagePreflightApprovalRequiredStates(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	missingApproval := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if missingApproval.Summary.CanApply || missingApproval.Summary.BlockingCount == 0 || !missingApproval.Summary.RequiresApproval ||
		missingApproval.Summary.ApprovalReady || !permissionPackagePreflightHasCheck(missingApproval.Checks, "approval_request_missing", "blocking") {
		t.Fatalf("expected missing approval to block preflight, got %#v", missingApproval)
	}

	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	if approved.Status != "approved" {
		t.Fatalf("expected approved request, got %#v", approved)
	}
	approvedInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	approvedPreflight := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", approvedInput, ""))
	if !approvedPreflight.Summary.CanApply || approvedPreflight.Summary.BlockingCount != 0 ||
		!approvedPreflight.Summary.RequiresApproval || !approvedPreflight.Summary.ApprovalReady ||
		!permissionPackagePreflightHasCheck(approvedPreflight.Checks, "approval_request_ready", "passed") ||
		len(approvedPreflight.Planned.Capabilities) != 1 || approvedPreflight.Planned.Capabilities[0].ID != updateTicket.ID {
		t.Fatalf("expected approved preflight to be ready, got %#v", approvedPreflight)
	}
	loadedApproval, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approved.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after preflight: ok=%v err=%v", ok, err)
	}
	if !loadedApproval.ConsumedAt.IsZero() || loadedApproval.ConsumedByApplicationID != "" {
		t.Fatalf("preflight must not consume approval, got %#v", loadedApproval)
	}
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if len(applications) != 0 {
		t.Fatalf("preflight must not create applications, got %#v", applications)
	}

	mismatchedInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            "eu-west",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	mismatched := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", mismatchedInput, ""))
	if mismatched.Summary.CanApply || !permissionPackagePreflightHasCheck(mismatched.Checks, "approval_request_invalid", "blocking") {
		t.Fatalf("expected mismatched approval to block preflight, got %#v", mismatched)
	}
}

func TestPermissionPackageWorkbenchPreviewSummarizesPrimaryJourney(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	searchCustomer := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	exportContracts := createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))

	beforeApply := decodeData[permissionPackageWorkbenchPreviewResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/workbench:preview", input, ""))
	if beforeApply.ApprovalRequest == nil || beforeApply.ApprovalRequest.ID != approved.ID {
		t.Fatalf("expected workbench preview to select the approved request, got %#v", beforeApply.ApprovalRequest)
	}
	if beforeApply.Summary.Status != "ready_to_apply" || beforeApply.Summary.PrimaryActionCode != "apply_permission_package" ||
		!beforeApply.Summary.ApprovalRequired || !beforeApply.Summary.CanApply || beforeApply.Summary.Applied ||
		beforeApply.Summary.AllowedCapabilityCount != 2 || beforeApply.Summary.BlockedCapabilityCount != 1 ||
		beforeApply.Summary.PlannedObjectCount != 6 || beforeApply.ProductionReadiness == nil ||
		beforeApply.ProductionReadiness.NextActionCode != "apply_permission_package" {
		t.Fatalf("expected ready-to-apply workbench summary, got %#v", beforeApply.Summary)
	}
	if !permissionPackageWorkbenchHasStep(beforeApply.Summary.Steps, "approval", "complete", "approval_approved") ||
		!permissionPackageWorkbenchHasStep(beforeApply.Summary.Steps, "apply", "current", "apply_ready") ||
		!permissionPackageWorkbenchHasStep(beforeApply.Summary.Steps, "validation", "waiting", "validation_waiting") {
		t.Fatalf("expected approval complete and apply current steps, got %#v", beforeApply.Summary.Steps)
	}
	if requestStep, ok := permissionPackageWorkbenchStepByKey(beforeApply.Summary.Steps, "request"); !ok || requestStep.Count != 0 || requestStep.Total != 0 {
		t.Fatalf("expected request step to avoid capability count noise, got step=%#v ok=%v", requestStep, ok)
	}

	applyInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            input["region"],
		"requestText":       input["requestText"],
		"subjectSelector":   input["subjectSelector"],
		"targetId":          target.ID,
		"templateId":        input["templateId"],
		"tenantId":          input["tenantId"],
		"workspaceId":       input["workspaceId"],
	}
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, ""))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, exportContracts, "export_contracts", "user:support-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, searchCustomer, "search_customer", "user:support-001", now.Add(2*time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, updateTicket, "update_ticket", "user:support-001", now.Add(3*time.Minute))

	afterEvidence := decodeData[permissionPackageWorkbenchPreviewResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/workbench:preview", input, ""))
	if afterEvidence.LatestApplication == nil || afterEvidence.LatestApplication.ID != applied.Application.ID ||
		afterEvidence.Summary.Status != "production_ready" || afterEvidence.Summary.PrimaryActionCode != "export_production_evidence" ||
		!afterEvidence.Summary.Applied || !afterEvidence.Summary.RuntimeEvidenceReady || !afterEvidence.Summary.ProductionReady {
		t.Fatalf("expected production-ready workbench summary after evidence, got summary=%#v app=%#v", afterEvidence.Summary, afterEvidence.LatestApplication)
	}
	if !permissionPackageWorkbenchHasStep(afterEvidence.Summary.Steps, "approval", "complete", "approval_approved") ||
		!permissionPackageWorkbenchHasStep(afterEvidence.Summary.Steps, "acceptance", "complete", "acceptance_ready") {
		t.Fatalf("expected completed approval and acceptance after evidence, got %#v", afterEvidence.Summary.Steps)
	}

	secondApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	afterNewRequest := decodeData[permissionPackageWorkbenchPreviewResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/workbench:preview", input, ""))
	if afterNewRequest.ApprovalRequest == nil || afterNewRequest.ApprovalRequest.ID != secondApproval.ID {
		t.Fatalf("expected workbench preview to select the new pending request, got %#v", afterNewRequest.ApprovalRequest)
	}
	if afterNewRequest.LatestApplication != nil || afterNewRequest.ProductionReadiness != nil {
		t.Fatalf("expected pending request to suppress historical production evidence, got app=%#v readiness=%#v", afterNewRequest.LatestApplication, afterNewRequest.ProductionReadiness)
	}
	if afterNewRequest.Summary.Status != "awaiting_approval" || afterNewRequest.Summary.PrimaryActionCode != "review_approval_request" ||
		afterNewRequest.Summary.Applied || afterNewRequest.Summary.ProductionReady || afterNewRequest.Summary.CanApply {
		t.Fatalf("expected pending request to restart the approval step, got %#v", afterNewRequest.Summary)
	}
	if !permissionPackageWorkbenchHasStep(afterNewRequest.Summary.Steps, "approval", "current", "approval_pending") ||
		!permissionPackageWorkbenchHasStep(afterNewRequest.Summary.Steps, "apply", "waiting", "apply_waiting") {
		t.Fatalf("expected pending approval and waiting apply steps, got %#v", afterNewRequest.Summary.Steps)
	}
}

func TestPermissionPackageProductionReadinessBlocksBeforeApplyAndReadyAfterEvidence(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	searchCustomer := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	exportContracts := createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	if approved.Status != "approved" {
		t.Fatalf("expected approved request, got %#v", approved)
	}

	before := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, permissionPackageProductionReadinessPath(input, approved.ID, "user:support-001"), nil, ""))
	if before.Status != "blocked" || before.Summary.BlockingCount == 0 || before.Summary.HasApplication {
		t.Fatalf("expected production readiness to block before apply, got %#v", before)
	}
	if before.NextActionCode != "apply_permission_package" {
		t.Fatalf("expected apply next action before application evidence, got %q", before.NextActionCode)
	}
	if !permissionPackageProductionReadinessHasCheck(before.Checks, "preflight_ready", "passed") ||
		!permissionPackageProductionReadinessHasCheck(before.Checks, "application_present", "blocking") {
		t.Fatalf("expected ready preflight and missing application blocker, got %#v", before.Checks)
	}
	beforeReport := decodeData[permissionPackageProductionEvidenceReportResponse](t, request(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(input, approved.ID, "user:support-001"), nil, ""))
	if beforeReport.ReportVersion != "production-readiness-report/v1" || beforeReport.Status != "blocked" ||
		beforeReport.Scope.TenantID != "tenant-east" || beforeReport.Scope.SubjectID != "user:support-001" ||
		beforeReport.Evidence.Application.Present || beforeReport.Summary.BlockingCount == 0 ||
		beforeReport.NextActionCode != "apply_permission_package" ||
		!permissionPackageProductionReadinessHasCheck(beforeReport.Checks, "application_present", "blocking") {
		t.Fatalf("expected blocked production evidence report before apply, got %#v", beforeReport)
	}

	applyInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            input["region"],
		"requestText":       input["requestText"],
		"subjectSelector":   input["subjectSelector"],
		"targetId":          target.ID,
		"templateId":        input["templateId"],
		"tenantId":          input["tenantId"],
		"workspaceId":       input["workspaceId"],
	}
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, ""))
	if applied.Application == nil || applied.Application.TemplateID != "support-ticket-triage" {
		t.Fatalf("expected applied support package application, got %#v", applied)
	}
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, exportContracts, "export_contracts", "user:support-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, updateTicket, "update_ticket", "user:support-001", now.Add(2*time.Minute))

	after := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, permissionPackageProductionReadinessPath(input, "", "user:support-001"), nil, ""))
	if after.Status != "ready" || after.Summary.BlockingCount != 0 || !after.Summary.HasApplication ||
		!after.Summary.HasAllowedTrace || !after.Summary.HasDeniedTrace || !after.Summary.HasAppliedAudit ||
		!after.Summary.AccessProfileReady || after.LatestApplication == nil || after.LatestApplication.ID != applied.Application.ID {
		t.Fatalf("expected production readiness after evidence, got %#v", after)
	}
	if after.NextActionCode != "export_production_evidence" {
		t.Fatalf("expected evidence export next action after readiness, got %q", after.NextActionCode)
	}
	if after.RuntimeEvidence.AllowedTrace == nil || after.RuntimeEvidence.AllowedTrace.CapabilityID != updateTicket.ID ||
		after.RuntimeEvidence.DeniedTrace == nil || after.RuntimeEvidence.DeniedTrace.CapabilityID != exportContracts.ID ||
		after.AuditEvidence.AppliedEvent == nil || after.AuditEvidence.AppliedEvent.ResourceID != applied.Application.ID {
		t.Fatalf("expected runtime and audit evidence, got runtime=%#v audit=%#v", after.RuntimeEvidence, after.AuditEvidence)
	}
	afterReport := decodeData[permissionPackageProductionEvidenceReportResponse](t, request(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(input, "", "user:support-001"), nil, ""))
	if afterReport.ReportVersion != "production-readiness-report/v1" || afterReport.Status != "ready" ||
		afterReport.Evidence.Application.ID != applied.Application.ID ||
		afterReport.Evidence.Application.TemplateVersion != 1 ||
		len(afterReport.Evidence.Application.AllowedCapabilityIDs) != len(applied.Application.AllowedCapabilityIDs) ||
		afterReport.Evidence.Runtime.AllowedTraceID == "" || afterReport.Evidence.Runtime.DeniedTraceID == "" ||
		afterReport.Evidence.Audit.AppliedEventID == "" ||
		afterReport.NextActionCode != "export_production_evidence" ||
		afterReport.Evidence.AccessProfile.Present != true ||
		afterReport.ReadinessGeneratedAt == "" {
		t.Fatalf("expected ready production evidence report after evidence, got %#v", afterReport)
	}
	for _, check := range []string{
		"application_present",
		"application_health_ready",
		"impact_ready",
		"access_profile_chain_present",
		"runtime_allowed_trace_present",
		"runtime_denied_trace_present",
		"applied_audit_event_present",
	} {
		if !permissionPackageProductionReadinessHasCheck(after.Checks, check, "passed") {
			t.Fatalf("expected readiness check %q to pass, got %#v", check, after.Checks)
		}
	}
	limitedTracePath := permissionPackageProductionReadinessPath(input, "", "user:support-001") + "&traceLimit=1"
	limitedTrace := decodeData[permissionPackageProductionReadinessResponse](t, request(t, router, http.MethodGet, limitedTracePath, nil, ""))
	if limitedTrace.Status != "blocked" || limitedTrace.RuntimeEvidence.AllowedTrace == nil || limitedTrace.RuntimeEvidence.DeniedTrace != nil ||
		limitedTrace.NextActionCode != "run_denied_runtime_call" ||
		!permissionPackageProductionReadinessHasCheck(limitedTrace.Checks, "runtime_denied_trace_present", "blocking") {
		t.Fatalf("expected traceLimit=1 to inspect only latest trace and block on missing denied evidence, got %#v", limitedTrace)
	}
	allowedIDs := map[string]struct{}{}
	for _, capability := range applied.Draft.AllowedCapabilities {
		allowedIDs[capability.ID] = struct{}{}
	}
	if len(allowedIDs) != 2 {
		t.Fatalf("expected support package to include two allowed capabilities, got %#v", applied.Draft.AllowedCapabilities)
	}
	if _, ok := allowedIDs[searchCustomer.ID]; !ok {
		t.Fatalf("expected support package to include search capability, got %#v", applied.Draft.AllowedCapabilities)
	}
	if _, ok := allowedIDs[updateTicket.ID]; !ok {
		t.Fatalf("expected support package to include search and update capabilities, got %#v", applied.Draft.AllowedCapabilities)
	}
}

func TestPermissionPackagePreflightDetectsDataScopeConflict(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithActionAndScopes(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, []domain.DataScope{{
		DataDomain:   "crm",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-east'",
	}}, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "eu-west",
		"requestText":      "给销售助手开通客户只读。",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	preflight := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if preflight.Summary.CanApply || preflight.Summary.BlockingCount == 0 ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "data_scope_fit", "blocking") {
		t.Fatalf("expected data-scope conflict to block preflight, got %#v", preflight)
	}
}

func TestPermissionPackagePreflightWarnsAboutExistingGrantChain(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, ""))
	if len(applied.TenantEntitlements) != 1 || applied.Application == nil {
		t.Fatalf("expected seed apply to create a grant chain, got %#v", applied)
	}
	preflight := decodeData[permissionPackageApplyPreflightResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:preflight", input, ""))
	if !preflight.Summary.CanApply || preflight.Summary.WarningCount == 0 || preflight.Summary.ExistingGrantCount != 1 ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "existing_grant_chain", "warning") ||
		len(preflight.ExistingGrants) != 1 || preflight.ExistingGrants[0].CapabilityID != search.ID ||
		preflight.ExistingGrants[0].TenantEntitlementID != applied.TenantEntitlements[0].ID ||
		preflight.ExistingGrants[0].WorkspaceAssignmentID != applied.WorkspaceAssignments[0].ID ||
		preflight.ExistingGrants[0].InstanceAssignmentID != applied.InstanceAssignments[0].ID {
		t.Fatalf("expected existing grant warning, got %#v", preflight)
	}
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if len(applications) != 1 {
		t.Fatalf("preflight should not create another application, got %#v", applications)
	}
}

func TestPermissionPackageApplicationImpactReportsDriftBlockers(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	target := createDirectAgent(t, repo, "Drift MCP Target", "tenant-east", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	caller := createDirectAgent(t, repo, "Drift Caller", "tenant-east", "ws-sales", "local", domain.AgentStatusActive, nil)
	entitlement, err := repo.CreateTenantEntitlement(t.Context(), domain.TenantEntitlement{
		ID:           "ent-disabled-drift",
		TenantID:     "tenant-east",
		TargetID:     target.ID,
		CapabilityID: "cap-drift",
		Effect:       domain.PolicyEffectAllow,
		Status:       domain.PolicyStatusDisabled,
		DataScopes:   []domain.DataScope{{DataDomain: "crm", Region: "华东"}},
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create disabled entitlement: %v", err)
	}
	workspaceAssignment, err := repo.CreateWorkspaceAssignment(t.Context(), domain.WorkspaceAssignment{
		ID:                  "wsa-disabled-drift",
		TenantEntitlementID: entitlement.ID,
		TenantID:            "tenant-east",
		WorkspaceID:         "ws-sales",
		Effect:              domain.PolicyEffectAllow,
		Status:              domain.PolicyStatusDisabled,
		DataScopes:          []domain.DataScope{{DataDomain: "crm", Region: "华东"}},
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		t.Fatalf("create disabled workspace assignment: %v", err)
	}
	instanceAssignment, err := repo.CreateInstanceAssignment(t.Context(), domain.InstanceAssignment{
		ID:                    "ina-disabled-drift",
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-sales",
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:sales-*",
		Effect:                domain.PolicyEffectAllow,
		Status:                domain.PolicyStatusDisabled,
		DataScopes:            []domain.DataScope{{DataDomain: "crm", Region: "华东"}},
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err != nil {
		t.Fatalf("create disabled instance assignment: %v", err)
	}
	application, err := repo.CreatePermissionPackageApplication(t.Context(), domain.PermissionPackageApplication{
		ID:                     "ppa-drift",
		DraftID:                "draft-drift",
		TemplateID:             "sales-readonly",
		TemplateVersion:        1,
		TenantID:               "tenant-east",
		WorkspaceID:            "ws-sales",
		TargetID:               target.ID,
		CallerInstanceID:       caller.ID,
		SubjectSelector:        "user:sales-*",
		RequestText:            "drift review",
		Region:                 "华东",
		DataScopes:             []domain.DataScope{{DataDomain: "crm", Region: "华东"}},
		AllowedCapabilityIDs:   []string{},
		AllowedCapabilityKeys:  []string{},
		TenantEntitlementIDs:   []string{entitlement.ID},
		WorkspaceAssignmentIDs: []string{"wsa-missing-drift", workspaceAssignment.ID},
		InstanceAssignmentIDs:  []string{instanceAssignment.ID},
		AppliedAt:              now,
	})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	impact := decodeData[permissionPackageApplicationImpactResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+application.ID+"/impact?tenantId=tenant-root&workspaceId=ws-sales", nil, ""))
	if impact.Summary.CreatedObjectCount != 4 || impact.Summary.ActiveObjectCount != 0 ||
		impact.Summary.MissingObjectCount != 1 || impact.Summary.RollbackReady {
		t.Fatalf("unexpected drift impact summary: %#v", impact.Summary)
	}
	health := decodeData[permissionPackageApplicationHealthResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-root&workspaceId=ws-sales&limit=10", nil, ""))
	if health.Summary.Total != 1 || health.Summary.Ready != 0 || health.Summary.Drifted != 1 || health.Summary.NeedsReview != 0 {
		t.Fatalf("expected drifted application health summary, got %#v", health.Summary)
	}
	if len(health.Applications) != 1 {
		t.Fatalf("expected one drifted application health row, got %#v", health.Applications)
	}
	healthRow := health.Applications[0]
	if healthRow.Application.ID != application.ID || healthRow.Status != "drifted" || healthRow.CreatedObjectCount != 4 ||
		healthRow.ActiveObjectCount != 0 || healthRow.MissingObjectCount != 1 || healthRow.RollbackReady {
		t.Fatalf("unexpected drifted application health row: %#v", healthRow)
	}
	for _, code := range []string{"missing_created_objects", "inactive_created_objects"} {
		if !containsString(healthRow.BlockerCodes, code) {
			t.Fatalf("expected drift health blocker code %q, got %#v", code, healthRow.BlockerCodes)
		}
	}
	for _, code := range []string{"missing_created_objects", "inactive_created_objects", "no_allowed_capabilities"} {
		if !containsString(impact.RollbackReview.BlockerCodes, code) {
			t.Fatalf("expected rollback blocker code %q, got %#v", code, impact.RollbackReview.BlockerCodes)
		}
		if !containsString(impact.RemediationPlan.BlockerCodes, code) {
			t.Fatalf("expected remediation blocker code %q, got %#v", code, impact.RemediationPlan.BlockerCodes)
		}
	}
	if impact.RollbackReview.Ready || impact.RemediationPlan.Ready {
		t.Fatalf("drift impact should not be ready: rollback=%#v remediation=%#v", impact.RollbackReview, impact.RemediationPlan)
	}
	if impact.RollbackReview.BlockerCodes == nil || impact.RemediationPlan.BlockerCodes == nil {
		t.Fatalf("expected blocker codes to encode as arrays: rollback=%#v remediation=%#v", impact.RollbackReview.BlockerCodes, impact.RemediationPlan.BlockerCodes)
	}
	if !remediationActionsContain(impact.RemediationPlan.Actions, "tenant_entitlement", entitlement.ID, "investigate") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "workspace_assignment", "wsa-missing-drift", "investigate") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "workspace_assignment", workspaceAssignment.ID, "investigate") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "instance_assignment", instanceAssignment.ID, "investigate") ||
		!remediationActionsContain(impact.RemediationPlan.Actions, "access_decision", application.ID, "verify") {
		t.Fatalf("expected drift remediation investigate and verify actions, got %#v", impact.RemediationPlan.Actions)
	}
	for _, action := range impact.RemediationPlan.Actions {
		if !action.ReadOnly {
			t.Fatalf("expected drift remediation actions to remain read-only, got %#v", action)
		}
	}
}

func TestPermissionPackageApplicationImpactRedactsOutOfScopeCapabilityHydration(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance MCP", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	application, err := repo.CreatePermissionPackageApplication(t.Context(), domain.PermissionPackageApplication{
		ID:                    "ppa-cross-capability",
		DraftID:               "draft-cross-capability",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              supportTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		SubjectSelector:       "role:support",
		RequestText:           "dirty application review",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{"export_invoices"},
		AppliedAt:             now,
	})
	if err != nil {
		t.Fatalf("create dirty application: %v", err)
	}

	impact := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+application.ID+"/impact?tenantId=tenant-east&workspaceId=ws-support", nil, "", "support-key")
	if impact.Code != http.StatusOK {
		t.Fatalf("scoped admin should read own application impact, got %d body=%s", impact.Code, impact.Body.String())
	}
	health := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-east&workspaceId=ws-support&limit=10", nil, "", "support-key")
	if health.Code != http.StatusOK {
		t.Fatalf("scoped admin should read own application health, got %d body=%s", health.Code, health.Body.String())
	}
	for _, resp := range []*httptest.ResponseRecorder{impact, health} {
		body := resp.Body.String()
		if strings.Contains(body, "export_invoices") || strings.Contains(body, "Finance MCP") {
			t.Fatalf("scoped application response leaked out-of-scope hydrated capability details: %s", body)
		}
	}
}

func TestPermissionPackageApplicationReadsRedactDirtyApplicationCapabilityScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	dirtyCapabilityApplication, err := repo.CreatePermissionPackageApplication(t.Context(), domain.PermissionPackageApplication{
		ID:                    "ppa-dirty-capability",
		DraftID:               "draft-dirty-capability",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              supportTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		SubjectSelector:       "role:support",
		RequestText:           "dirty capability application",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{financeCapability.Key},
		AppliedAt:             now,
	})
	if err != nil {
		t.Fatalf("create dirty capability application: %v", err)
	}
	dirtyTargetApplication, err := repo.CreatePermissionPackageApplication(t.Context(), domain.PermissionPackageApplication{
		ID:                    "ppa-dirty-target",
		DraftID:               "draft-dirty-target",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              financeTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		SubjectSelector:       "role:support",
		RequestText:           "dirty target application",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "finance", Region: "eu-west"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{financeCapability.Key},
		AppliedAt:             now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("create dirty target application: %v", err)
	}
	if _, err := repo.AppendAuditEvent(t.Context(), domain.AuditEvent{
		ID:           "aud-dirty-application",
		TenantID:     "tenant-east",
		WorkspaceID:  "ws-support",
		Actor:        "platform",
		Action:       "permission_package.applied",
		ResourceType: "permission_package",
		ResourceID:   dirtyCapabilityApplication.ID,
		Summary:      "Permission package applied",
		Metadata: map[string]any{
			"applicationId":          dirtyCapabilityApplication.ID,
			"draftId":                dirtyCapabilityApplication.DraftID,
			"templateId":             dirtyCapabilityApplication.TemplateID,
			"templateVersion":        dirtyCapabilityApplication.TemplateVersion,
			"targetId":               financeTarget.ID,
			"callerInstanceId":       supportCaller.ID,
			"subjectSelector":        dirtyCapabilityApplication.SubjectSelector,
			"allowedCapabilityIds":   []string{financeCapability.ID},
			"allowedCapabilityKeys":  []string{financeCapability.Key},
			"tenantEntitlementIds":   []string{"tte-cross-scope"},
			"workspaceAssignmentIds": []string{"wsa-cross-scope"},
			"instanceAssignmentIds":  []string{"ias-cross-scope"},
		},
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("append dirty applied audit event: %v", err)
	}

	leaks := []string{financeTarget.ID, financeTarget.Name, financeCapability.ID, financeCapability.Key, "ws-finance"}
	listPath := "/api/v1/permission-packages/applications?tenantId=tenant-east&workspaceId=ws-support&templateId=support-ticket-triage&targetId=" + supportTarget.ID + "&callerInstanceId=" + supportCaller.ID + "&limit=10"
	listResp := requestWithAdmin(t, router, http.MethodGet, listPath, nil, "", "support-key")
	if listResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read in-scope application list, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	assertResponseDoesNotContain(t, listResp.Body.String(), leaks...)
	applications := decodeData[[]permissionPackageApplicationResponse](t, listResp)
	if len(applications) != 1 || applications[0].ID != dirtyCapabilityApplication.ID {
		t.Fatalf("expected scoped application list to keep the in-scope application only, got %#v", applications)
	}
	if len(applications[0].AllowedCapabilityIDs) != 0 || len(applications[0].AllowedCapabilityKeys) != 0 {
		t.Fatalf("expected dirty out-of-scope capabilities to be redacted from application list, got %#v", applications[0])
	}

	mcpList := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "dirty-application-list",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_applications",
			"arguments": map[string]any{
				"callerInstanceId": supportCaller.ID,
				"limit":            10,
				"targetId":         supportTarget.ID,
				"templateId":       "support-ticket-triage",
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-support",
			},
		},
	}, "", "support-key"))
	assertResponseDoesNotContain(t, string(mcpList.Result.StructuredContent), leaks...)
	var mcpApplications []permissionPackageApplicationResponse
	if err := json.Unmarshal(mcpList.Result.StructuredContent, &mcpApplications); err != nil {
		t.Fatalf("decode management MCP application list: %v", err)
	}
	if len(mcpApplications) != 1 || mcpApplications[0].ID != dirtyCapabilityApplication.ID ||
		len(mcpApplications[0].AllowedCapabilityIDs) != 0 || len(mcpApplications[0].AllowedCapabilityKeys) != 0 {
		t.Fatalf("expected MCP application list to redact dirty out-of-scope capabilities, got %#v", mcpApplications)
	}

	healthResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/health?tenantId=tenant-east&workspaceId=ws-support&limit=10", nil, "", "support-key")
	if healthResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read application health, got %d body=%s", healthResp.Code, healthResp.Body.String())
	}
	assertResponseDoesNotContain(t, healthResp.Body.String(), append(leaks, dirtyTargetApplication.ID)...)
	health := decodeData[permissionPackageApplicationHealthResponse](t, healthResp)
	if health.Summary.Total != 1 || len(health.Applications) != 1 || health.Applications[0].Application.ID != dirtyCapabilityApplication.ID {
		t.Fatalf("expected application health to hide dirty target applications, got %#v", health)
	}

	input := map[string]any{
		"callerInstanceId": supportCaller.ID,
		"region":           "us-east",
		"requestText":      "validate support package status",
		"subjectSelector":  "role:support",
		"targetId":         supportTarget.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	readinessResp := requestWithAdmin(t, router, http.MethodGet, permissionPackageProductionReadinessPath(input, "", "user:support-001"), nil, "", "support-key")
	if readinessResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read production readiness, got %d body=%s", readinessResp.Code, readinessResp.Body.String())
	}
	assertResponseDoesNotContain(t, readinessResp.Body.String(), leaks...)
	readiness := decodeData[permissionPackageProductionReadinessResponse](t, readinessResp)
	if readiness.LatestApplication == nil || readiness.LatestApplication.ID != dirtyCapabilityApplication.ID {
		t.Fatalf("expected readiness to keep the in-scope latest application, got %#v", readiness.LatestApplication)
	}
	if len(readiness.LatestApplication.AllowedCapabilityIDs) != 0 || len(readiness.LatestApplication.AllowedCapabilityKeys) != 0 {
		t.Fatalf("expected readiness latest application to redact out-of-scope capabilities, got %#v", readiness.LatestApplication)
	}

	reportResp := requestWithAdmin(t, router, http.MethodGet, permissionPackageProductionEvidenceReportPath(input, "", "user:support-001"), nil, "", "support-key")
	if reportResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read production report, got %d body=%s", reportResp.Code, reportResp.Body.String())
	}
	assertResponseDoesNotContain(t, reportResp.Body.String(), leaks...)
	report := decodeData[permissionPackageProductionEvidenceReportResponse](t, reportResp)
	if report.Evidence.Application.ID != dirtyCapabilityApplication.ID || len(report.Evidence.Application.AllowedCapabilityIDs) != 0 {
		t.Fatalf("expected production report application evidence to redact out-of-scope capabilities, got %#v", report.Evidence.Application)
	}

	auditResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/audit/events?tenantId=tenant-east&workspaceId=ws-support&action=permission_package.applied&resourceId="+dirtyCapabilityApplication.ID, nil, "", "support-key")
	if auditResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read applied audit event, got %d body=%s", auditResp.Code, auditResp.Body.String())
	}
	assertResponseDoesNotContain(t, auditResp.Body.String(), leaks...)
	auditEvents := decodeData[[]auditEventResponse](t, auditResp)
	if len(auditEvents) != 1 {
		t.Fatalf("expected one scoped applied audit event, got %#v", auditEvents)
	}
	if auditEvents[0].Metadata["targetId"] != supportTarget.ID || auditEvents[0].Metadata["callerInstanceId"] != supportCaller.ID {
		t.Fatalf("expected applied audit metadata to be rebuilt from visible application scope, got %#v", auditEvents[0].Metadata)
	}
	if raw, ok := auditEvents[0].Metadata["allowedCapabilityIds"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("expected applied audit metadata to redact out-of-scope capability ids, got %#v", auditEvents[0].Metadata)
	}
	if raw, ok := auditEvents[0].Metadata["allowedCapabilityKeys"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("expected applied audit metadata to redact out-of-scope capability keys, got %#v", auditEvents[0].Metadata)
	}
	if readiness.AuditEvidence.AppliedEvent == nil {
		t.Fatalf("expected readiness to include scoped applied audit event")
	}
	if readiness.AuditEvidence.AppliedEvent.Metadata["targetId"] != supportTarget.ID {
		t.Fatalf("expected readiness applied audit event to use visible target, got %#v", readiness.AuditEvidence.AppliedEvent.Metadata)
	}
	if raw, ok := readiness.AuditEvidence.AppliedEvent.Metadata["allowedCapabilityIds"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("expected readiness applied audit event to redact out-of-scope capability ids, got %#v", readiness.AuditEvidence.AppliedEvent.Metadata)
	}
	mcpReadiness := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "dirty-production-readiness",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "check_permission_package_production_readiness",
			"arguments": input,
		},
	}, "", "support-key"))
	assertResponseDoesNotContain(t, string(mcpReadiness.Result.StructuredContent), leaks...)

	targetImpactResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/applications/"+dirtyTargetApplication.ID+"/impact?tenantId=tenant-east&workspaceId=ws-support", nil, "", "support-key")
	if targetImpactResp.Code != http.StatusNotFound || strings.Contains(targetImpactResp.Body.String(), financeTarget.ID) ||
		strings.Contains(targetImpactResp.Body.String(), financeCapability.ID) || strings.Contains(targetImpactResp.Body.String(), financeCapability.Key) {
		t.Fatalf("dirty target application impact should be hidden without leaking finance details, got %d body=%s", targetImpactResp.Code, targetImpactResp.Body.String())
	}
}

func TestPermissionPackageApprovalAuditMetadataRedactsDirtyScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	approval, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), domain.PermissionPackageApprovalRequest{
		ID:                    "ppar-dirty-audit",
		DraftID:               "draft-dirty-approval",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              supportTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		SubjectSelector:       "role:support",
		RequestText:           "approve support ticket updates",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{financeCapability.Key},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
			Reasons: []domain.PermissionPackagePolicyReason{{
				ID:            "policy:dirty-audit",
				CapabilityID:  financeCapability.ID,
				CapabilityKey: financeCapability.Key,
				Severity:      "high",
				Message:       "Approval is required.",
				ReasonKey:     "permissionPolicy.actionApprovalRequired",
			}},
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "support-admin",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create dirty approval request: %v", err)
	}
	if _, err := repo.AppendAuditEvent(t.Context(), domain.AuditEvent{
		ID:           "aud-dirty-approval-request",
		TenantID:     "tenant-east",
		WorkspaceID:  "ws-support",
		Actor:        "support-admin",
		Action:       "permission_package.approval_requested",
		ResourceType: "permission_package_approval_request",
		ResourceID:   approval.ID,
		Summary:      "Permission package approval requested",
		Metadata: map[string]any{
			"approvalRequestId":    approval.ID,
			"draftId":              approval.DraftID,
			"templateId":           approval.TemplateID,
			"templateVersion":      approval.TemplateVersion,
			"policyVersion":        approval.PolicyVersion,
			"targetId":             financeTarget.ID,
			"callerInstanceId":     supportCaller.ID,
			"status":               approval.Status,
			"requestedBy":          approval.RequestedBy,
			"reviewedBy":           approval.ReviewedBy,
			"reasonCount":          1,
			"allowedCapabilityIds": []string{financeCapability.ID},
			"expiresAt":            approval.ExpiresAt,
		},
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("append dirty approval audit event: %v", err)
	}

	leaks := []string{financeTarget.ID, financeTarget.Name, financeCapability.ID, financeCapability.Key, "ws-finance"}
	auditResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/audit/events?tenantId=tenant-east&workspaceId=ws-support&action=permission_package.approval_requested&resourceId="+approval.ID, nil, "", "support-key")
	if auditResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read approval audit event, got %d body=%s", auditResp.Code, auditResp.Body.String())
	}
	assertResponseDoesNotContain(t, auditResp.Body.String(), leaks...)
	auditEvents := decodeData[[]auditEventResponse](t, auditResp)
	if len(auditEvents) != 1 {
		t.Fatalf("expected one scoped approval audit event, got %#v", auditEvents)
	}
	if auditEvents[0].Metadata["targetId"] != supportTarget.ID || auditEvents[0].Metadata["callerInstanceId"] != supportCaller.ID {
		t.Fatalf("expected approval audit metadata to be rebuilt from visible approval request scope, got %#v", auditEvents[0].Metadata)
	}
	if raw, ok := auditEvents[0].Metadata["allowedCapabilityIds"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("expected approval audit metadata to redact out-of-scope capability ids, got %#v", auditEvents[0].Metadata)
	}
}

func TestPermissionPackageApprovalListRedactsDirtyCapabilityScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportTarget := createDirectAgent(t, repo, "Support MCP", "tenant-east", "ws-support", "mcp", domain.AgentStatusActive, nil)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	financeCapability := createDirectCapabilityWithAction(t, repo, financeTarget.ID, "export_invoices", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)
	approval, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), domain.PermissionPackageApprovalRequest{
		ID:                    "ppar-dirty-list",
		DraftID:               "draft-dirty-list",
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              "tenant-east",
		WorkspaceID:           "ws-support",
		TargetID:              supportTarget.ID,
		CallerInstanceID:      supportCaller.ID,
		SubjectSelector:       "role:support",
		RequestText:           "approve support ticket updates",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{financeCapability.ID},
		AllowedCapabilityKeys: []string{financeCapability.Key},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
			Reasons: []domain.PermissionPackagePolicyReason{{
				ID:            "policy:dirty-list",
				CapabilityID:  financeCapability.ID,
				CapabilityKey: financeCapability.Key,
				Severity:      "high",
				Message:       "Approval is required.",
				ReasonKey:     "permissionPolicy.actionApprovalRequired",
			}},
			NextActions: []string{"Review approval scope."},
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "support-admin",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create dirty approval request: %v", err)
	}

	listResp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-east&workspaceId=ws-support&status=pending", nil, "", "support-key")
	if listResp.Code != http.StatusOK {
		t.Fatalf("scoped admin should read approval request list, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	assertResponseDoesNotContain(t, listResp.Body.String(), financeTarget.ID, financeTarget.Name, financeCapability.ID, financeCapability.Key, "ws-finance")
	approvals := decodeData[[]permissionPackageApprovalRequestResponse](t, listResp)
	if len(approvals) != 1 || approvals[0].ID != approval.ID {
		t.Fatalf("expected one scoped approval request, got %#v", approvals)
	}
	if len(approvals[0].AllowedCapabilityIDs) != 0 || len(approvals[0].AllowedCapabilityKeys) != 0 {
		t.Fatalf("expected dirty approval capabilities to be redacted, got ids=%#v keys=%#v", approvals[0].AllowedCapabilityIDs, approvals[0].AllowedCapabilityKeys)
	}
	if len(approvals[0].PolicyGate.Reasons) != 0 {
		t.Fatalf("expected dirty approval policy reasons to be redacted, got %#v", approvals[0].PolicyGate.Reasons)
	}
}

func TestScopedAdminCannotResolveDirtyPermissionPackageApprovalRequest(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	supportCaller := createDirectAgent(t, repo, "Support caller", "tenant-east", "ws-support", "local", domain.AgentStatusActive, nil)
	financeTarget := createDirectAgent(t, repo, "Finance Export Service", "tenant-east", "ws-finance", "mcp", domain.AgentStatusActive, nil)
	approval, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), domain.PermissionPackageApprovalRequest{
		ID:               "ppar-dirty-resolution",
		DraftID:          "draft-dirty-resolution",
		TemplateID:       "support-ticket-triage",
		TemplateVersion:  1,
		PolicyVersion:    1,
		TenantID:         "tenant-east",
		WorkspaceID:      "ws-support",
		TargetID:         financeTarget.ID,
		CallerInstanceID: supportCaller.ID,
		SubjectSelector:  "role:support",
		RequestText:      "approve support ticket updates",
		Region:           "us-east",
		DataScopes:       []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "requester",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create dirty approval request: %v", err)
	}

	approveResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "", "support-key")
	if approveResp.Code != http.StatusForbidden {
		t.Fatalf("dirty approval request resolution should be forbidden, got %d body=%s", approveResp.Code, approveResp.Body.String())
	}
	withdrawResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/withdraw", map[string]any{
		"comment": "withdraw dirty request",
	}, "", "support-key")
	if withdrawResp.Code != http.StatusForbidden {
		t.Fatalf("dirty approval request withdraw should be forbidden, got %d body=%s", withdrawResp.Code, withdrawResp.Body.String())
	}
	mcpApprove := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "dirty-approval-approve",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": map[string]any{
				"id": approval.ID,
			},
		},
	}, "", "support-key")
	var envelope mcpEnvelopeResponse
	if err := json.Unmarshal(mcpApprove.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode MCP dirty approval envelope: %v body=%s", err, mcpApprove.Body.String())
	}
	if envelope.Error == nil || !strings.Contains(envelope.Error.Message, "outside authenticated admin scope") {
		t.Fatalf("dirty MCP approval request resolution should be rejected, got %#v body=%s", envelope, mcpApprove.Body.String())
	}
	stored, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get dirty approval request after rejected operations: ok=%v err=%v", ok, err)
	}
	if stored.Status != domain.PermissionPackageApprovalStatusPending || stored.ReviewedBy != "" || !stored.ResolvedAt.IsZero() {
		t.Fatalf("dirty approval request must remain pending after rejected operations, got %#v", stored)
	}
}

func TestApprovalResolutionRequiresExistingApprovalResources(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	approval, err := repo.CreatePermissionPackageApprovalRequest(t.Context(), domain.PermissionPackageApprovalRequest{
		ID:               "ppar-missing-resolution",
		DraftID:          "draft-missing-resolution",
		TemplateID:       "support-ticket-triage",
		TemplateVersion:  1,
		PolicyVersion:    1,
		TenantID:         "tenant-east",
		WorkspaceID:      "ws-support",
		TargetID:         "agt_missing_target",
		CallerInstanceID: "agt_missing_caller",
		SubjectSelector:  "role:support",
		RequestText:      "approve support ticket updates",
		Region:           "us-east",
		DataScopes:       []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "requester",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create missing-resource approval request: %v", err)
	}

	approveResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "", "support-key")
	if approveResp.Code != http.StatusNotFound || !strings.Contains(approveResp.Body.String(), "caller not found") {
		t.Fatalf("missing-resource approval request should be rejected, got %d body=%s", approveResp.Code, approveResp.Body.String())
	}
	mcpApprove := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "missing-approval-approve",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": map[string]any{
				"id": approval.ID,
			},
		},
	}, "", "support-key")
	var envelope mcpEnvelopeResponse
	if err := json.Unmarshal(mcpApprove.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode MCP missing-resource approval envelope: %v body=%s", err, mcpApprove.Body.String())
	}
	if envelope.Error == nil || !strings.Contains(envelope.Error.Message, "caller not found") {
		t.Fatalf("missing-resource MCP approval request should be rejected, got %#v body=%s", envelope, mcpApprove.Body.String())
	}
	stored, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get missing-resource approval request after rejected operations: ok=%v err=%v", ok, err)
	}
	if stored.Status != domain.PermissionPackageApprovalStatusPending || stored.ReviewedBy != "" || !stored.ResolvedAt.IsZero() {
		t.Fatalf("missing-resource approval request must remain pending after rejected operations, got %#v", stored)
	}
}

func TestApprovalResolutionRejectsExpiredApprovalRequest(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "security", Key: "security-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	approval := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-expired-resolution", "tenant-east", "ws-support", now)
	approval.ExpiresAt = now.Add(-time.Minute)
	approval.UpdatedAt = approval.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), approval); err != nil || !ok {
		t.Fatalf("expire approval request: ok=%v err=%v", ok, err)
	}

	approveResp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "", "security-key")
	if approveResp.Code != http.StatusBadRequest || !strings.Contains(approveResp.Body.String(), "expired") {
		t.Fatalf("expired approval request approve should be rejected, got %d body=%s", approveResp.Code, approveResp.Body.String())
	}
	mcpReject := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "expired-approval-reject",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "reject_permission_package_approval_request",
			"arguments": map[string]any{
				"id":      approval.ID,
				"comment": "expired",
			},
		},
	}, "", "security-key")
	var envelope mcpEnvelopeResponse
	if err := json.Unmarshal(mcpReject.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode MCP expired approval envelope: %v body=%s", err, mcpReject.Body.String())
	}
	if envelope.Error == nil || !strings.Contains(envelope.Error.Message, "expired") {
		t.Fatalf("expired MCP approval request resolution should be rejected, got %#v body=%s", envelope, mcpReject.Body.String())
	}
	stored, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get expired approval request after rejected operations: ok=%v err=%v", ok, err)
	}
	if stored.Status != domain.PermissionPackageApprovalStatusPending || stored.ReviewedBy != "" || !stored.ResolvedAt.IsZero() {
		t.Fatalf("expired approval request must remain pending after rejected operations, got %#v", stored)
	}
}

func TestPermissionPackageApplyRequiresApprovalForPolicyGatedDraft(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	draft := decodeData[permissionPackageDraftResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, ""))
	if !draft.Readiness.CanApply {
		t.Fatalf("expected ready draft before policy approval check, got %#v", draft.Readiness)
	}
	if draft.PolicyGate.Decision != "approval_required" || draft.PolicyGate.CanApplyDirectly || len(draft.PolicyGate.Reasons) == 0 {
		t.Fatalf("expected approval-required policy gate, got %#v", draft.PolicyGate)
	}

	applyResp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, "")
	if applyResp.Code != http.StatusBadRequest {
		t.Fatalf("expected approval-required apply to fail, status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
	if !strings.Contains(applyResp.Body.String(), "requires approval") {
		t.Fatalf("expected approval error message, body=%s", applyResp.Body.String())
	}
	entitlements, err := repo.ListTenantEntitlements(t.Context(), store.EntitlementFilter{})
	if err != nil {
		t.Fatalf("list entitlements: %v", err)
	}
	workspaceAssignments, err := repo.ListWorkspaceAssignments(t.Context(), store.AssignmentFilter{})
	if err != nil {
		t.Fatalf("list workspace assignments: %v", err)
	}
	instanceAssignments, err := repo.ListInstanceAssignments(t.Context(), store.InstanceAssignmentFilter{})
	if err != nil {
		t.Fatalf("list instance assignments: %v", err)
	}
	applications, err := repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.applied"})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	updated, ok, err := repo.GetCapability(t.Context(), updateTicket.ID)
	if err != nil || !ok {
		t.Fatalf("get capability: ok=%v err=%v", ok, err)
	}
	if len(entitlements) != 0 || len(workspaceAssignments) != 0 || len(instanceAssignments) != 0 || len(applications) != 0 || len(events) != 0 {
		t.Fatalf("approval-required package should not write records: entitlements=%#v workspace=%#v instances=%#v applications=%#v events=%#v", entitlements, workspaceAssignments, instanceAssignments, applications, events)
	}
	if updated.DiscoveryStatus != domain.CapabilityDiscoveryPendingReview {
		t.Fatalf("approval-required package should not update capability, got %#v", updated)
	}

	firstApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	if firstApproval.Status != "pending" || firstApproval.TemplateID != "support-ticket-triage" ||
		firstApproval.TemplateVersion != 1 || firstApproval.PolicyVersion != 1 ||
		firstApproval.TargetID != target.ID || firstApproval.CallerInstanceID != caller.ID ||
		len(firstApproval.AllowedCapabilityIDs) != 1 || firstApproval.AllowedCapabilityIDs[0] != updateTicket.ID ||
		len(firstApproval.PolicyGate.Reasons) == 0 {
		t.Fatalf("unexpected created approval request: %#v", firstApproval)
	}
	if firstApproval.ExpiresAt.IsZero() || !firstApproval.ExpiresAt.After(firstApproval.CreatedAt) {
		t.Fatalf("approval request should expose a future expiry: %#v", firstApproval)
	}
	listedApprovals := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-root&workspaceId=ws-support&templateId=support-ticket-triage&targetId="+target.ID+"&callerInstanceId="+caller.ID+"&status=pending&limit=1", nil, ""))
	if len(listedApprovals) != 1 || listedApprovals[0].ID != firstApproval.ID {
		t.Fatalf("expected listed pending approval request, got %#v", listedApprovals)
	}

	pendingApplyInput := map[string]any{
		"approvalRequestId": firstApproval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	pendingApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", pendingApplyInput, "")
	if pendingApply.Code != http.StatusBadRequest || !strings.Contains(pendingApply.Body.String(), "approved") {
		t.Fatalf("pending approval request should not authorize apply, status=%d body=%s", pendingApply.Code, pendingApply.Body.String())
	}
	rejectedApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+firstApproval.ID+"/reject", map[string]any{
		"reviewer": "security",
		"comment":  "too broad",
	}, ""))
	if rejectedApproval.Status != "rejected" || rejectedApproval.ReviewedBy != "security" || rejectedApproval.ReviewComment != "too broad" {
		t.Fatalf("unexpected rejected approval request: %#v", rejectedApproval)
	}
	rejectedApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", pendingApplyInput, "")
	if rejectedApply.Code != http.StatusBadRequest || !strings.Contains(rejectedApply.Body.String(), "approved") {
		t.Fatalf("rejected approval request should not authorize apply, status=%d body=%s", rejectedApply.Code, rejectedApply.Body.String())
	}

	secondApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approvedApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+secondApproval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	if approvedApproval.Status != "approved" || approvedApproval.ReviewedBy != "security" {
		t.Fatalf("unexpected approved approval request: %#v", approvedApproval)
	}
	mismatchedApplyInput := map[string]any{
		"approvalRequestId": secondApproval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "eu-west",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	mismatchedApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", mismatchedApplyInput, "")
	if mismatchedApply.Code != http.StatusBadRequest || !strings.Contains(mismatchedApply.Body.String(), "does not match") {
		t.Fatalf("mismatched approval request should not authorize apply, status=%d body=%s", mismatchedApply.Code, mismatchedApply.Body.String())
	}

	expiredApproval, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), secondApproval.ID)
	if err != nil || !ok {
		t.Fatalf("get approved approval for expiry test: ok=%v err=%v", ok, err)
	}
	expiredApproval.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	expiredApproval.UpdatedAt = expiredApproval.ExpiresAt
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), expiredApproval); err != nil || !ok {
		t.Fatalf("expire approval request: ok=%v err=%v", ok, err)
	}
	approvedApplyInput := map[string]any{
		"approvalRequestId": secondApproval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	expiredApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", approvedApplyInput, "")
	if expiredApply.Code != http.StatusBadRequest || !strings.Contains(expiredApply.Body.String(), "expired") {
		t.Fatalf("expired approval request should not authorize apply, status=%d body=%s", expiredApply.Code, expiredApply.Body.String())
	}

	thirdApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	thirdApproval = decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+thirdApproval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	approvedApplyInput["approvalRequestId"] = thirdApproval.ID
	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", approvedApplyInput, ""))
	if len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != updateTicket.ID ||
		len(applied.WorkspaceAssignments) != 1 || len(applied.InstanceAssignments) != 1 ||
		applied.Application == nil || applied.Application.TemplateID != "support-ticket-triage" {
		t.Fatalf("expected approved package apply to write records, got %#v", applied)
	}
	consumedApproval, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), thirdApproval.ID)
	if err != nil || !ok {
		t.Fatalf("get consumed approval request: ok=%v err=%v", ok, err)
	}
	if consumedApproval.ConsumedAt.IsZero() || consumedApproval.ConsumedByApplicationID != applied.Application.ID {
		t.Fatalf("approval request should be consumed by application %s, got %#v", applied.Application.ID, consumedApproval)
	}
	reusedApply := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", approvedApplyInput, "")
	if reusedApply.Code != http.StatusBadRequest ||
		!strings.Contains(reusedApply.Body.String(), "PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED") ||
		!strings.Contains(reusedApply.Body.String(), "already consumed") {
		t.Fatalf("consumed approval request should not authorize apply, status=%d body=%s", reusedApply.Code, reusedApply.Body.String())
	}
	updated, ok, err = repo.GetCapability(t.Context(), updateTicket.ID)
	if err != nil || !ok {
		t.Fatalf("get applied capability: ok=%v err=%v", ok, err)
	}
	if updated.DiscoveryStatus != domain.CapabilityDiscoveryApproved {
		t.Fatalf("approved package should update capability, got %#v", updated)
	}
	events, err = repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.applied"})
	if err != nil {
		t.Fatalf("list applied audit events: %v", err)
	}
	if len(events) != 1 || events[0].Metadata["approvalRequestId"] != thirdApproval.ID ||
		events[0].Metadata["approvalConsumedAt"] == nil || events[0].Metadata["approvalExpiresAt"] == nil {
		t.Fatalf("expected applied audit event with approval request id, got %#v", events)
	}
	applications, err = repo.ListPermissionPackageApplications(t.Context(), store.PermissionPackageApplicationFilter{})
	if err != nil {
		t.Fatalf("list applications after consumed retry: %v", err)
	}
	if len(applications) != 1 {
		t.Fatalf("consumed approval retry should not write duplicate applications: %#v", applications)
	}
}

func TestPermissionPackageApprovalRejectsCapabilityDriftAfterApproval(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	updateTicket := createDirectCapabilityWithActionAndScopes(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, []domain.DataScope{{
		DataDomain:   "crm",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-east'",
	}}, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	if approved.Status != "approved" {
		t.Fatalf("expected approved request, got %#v", approved)
	}

	changed := decodeData[capabilityResponse](t, request(t, router, http.MethodPatch, "/api/v1/capabilities/"+updateTicket.ID, map[string]any{
		"dataScopes": []map[string]any{{"dataDomain": "crm"}},
	}, ""))
	if len(changed.DataScopes) != 1 || changed.DataScopes[0].Region != "" {
		t.Fatalf("expected capability boundary to be widened, got %#v", changed.DataScopes)
	}

	applyInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            input["region"],
		"requestText":       input["requestText"],
		"subjectSelector":   input["subjectSelector"],
		"targetId":          target.ID,
		"templateId":        input["templateId"],
		"tenantId":          input["tenantId"],
		"workspaceId":       input["workspaceId"],
	}
	applyResp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, "")
	if applyResp.Code != http.StatusBadRequest || !strings.Contains(applyResp.Body.String(), "does not match") {
		t.Fatalf("capability drift should invalidate approval request, status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
}

func TestPermissionPackageApprovalRejectsTemplateAndPolicyVersionDrift(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "security",
	}, ""))
	stored, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approved.ID)
	if err != nil || !ok {
		t.Fatalf("get approval request for version drift: ok=%v err=%v", ok, err)
	}
	stored.TemplateVersion++
	stored.PolicyVersion++
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(t.Context(), stored); err != nil || !ok {
		t.Fatalf("update approval request version drift: ok=%v err=%v", ok, err)
	}

	applyInput := map[string]any{
		"approvalRequestId": approved.ID,
		"callerInstanceId":  caller.ID,
		"region":            input["region"],
		"requestText":       input["requestText"],
		"subjectSelector":   input["subjectSelector"],
		"targetId":          target.ID,
		"templateId":        input["templateId"],
		"tenantId":          input["tenantId"],
		"workspaceId":       input["workspaceId"],
	}
	applyResp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", applyInput, "")
	if applyResp.Code != http.StatusBadRequest || !strings.Contains(applyResp.Body.String(), "does not match") {
		t.Fatalf("template or policy version drift should invalidate approval request, status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
}

func TestPermissionPackageApprovalRejectsSelfApproval(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, ""))
	resp := request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "")
	if resp.Code != http.StatusForbidden || !strings.Contains(resp.Body.String(), "own permission package approval request") {
		t.Fatalf("self approval should be rejected, status=%d body=%s", resp.Code, resp.Body.String())
	}
	loaded, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after rejected self approval: ok=%v err=%v", ok, err)
	}
	if loaded.Status != domain.PermissionPackageApprovalStatusPending || loaded.ReviewedBy != "" {
		t.Fatalf("self approval rejection must leave request pending, got %#v", loaded)
	}
}

func TestPermissionPackageApprovalRequestWithdraw(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "", "requester-key"))
	impersonation := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/withdraw", map[string]any{
		"comment": "not my request",
	}, "", "security-key")
	if impersonation.Code != http.StatusForbidden || !strings.Contains(impersonation.Body.String(), "requester can withdraw") {
		t.Fatalf("non-requester withdraw should be rejected, status=%d body=%s", impersonation.Code, impersonation.Body.String())
	}

	withdrawn := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/withdraw", map[string]any{
		"comment": "wrong scope",
	}, "", "requester-key"))
	if withdrawn.Status != "withdrawn" || withdrawn.ReviewedBy != "requester" || withdrawn.ReviewComment != "wrong scope" || withdrawn.ResolvedAt.IsZero() {
		t.Fatalf("unexpected withdrawn approval request: %#v", withdrawn)
	}
	pending := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-root&workspaceId=ws-support&status=pending&limit=10", nil, "", "requester-key"))
	if len(pending) != 0 {
		t.Fatalf("withdrawn approval should leave pending queue, got %#v", pending)
	}
	withdrawnRows := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-root&workspaceId=ws-support&status=withdrawn&limit=10", nil, "", "requester-key"))
	if len(withdrawnRows) != 1 || withdrawnRows[0].ID != approval.ID {
		t.Fatalf("expected withdrawn approval in withdrawn list, got %#v", withdrawnRows)
	}
	withdrawnApplyInput := map[string]any{
		"approvalRequestId": approval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	withdrawnApply := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages:apply", withdrawnApplyInput, "", "requester-key")
	if withdrawnApply.Code != http.StatusBadRequest || !strings.Contains(withdrawnApply.Body.String(), "approved") {
		t.Fatalf("withdrawn approval request should not authorize apply, status=%d body=%s", withdrawnApply.Code, withdrawnApply.Body.String())
	}
	events, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{Action: "permission_package.approval_withdrawn"})
	if err != nil {
		t.Fatalf("list withdraw audit events: %v", err)
	}
	if len(events) != 1 || events[0].Actor != "requester" || events[0].ResourceID != approval.ID || events[0].Metadata["status"] != domain.PermissionPackageApprovalStatusWithdrawn {
		t.Fatalf("expected withdrawn audit event, got %#v", events)
	}

	approvedApproval := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "", "requester-key"))
	approvedApproval = decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approvedApproval.ID+"/approve", nil, "", "security-key"))
	withdrawApproved := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approvedApproval.ID+"/withdraw", map[string]any{
		"comment": "too late",
	}, "", "requester-key")
	if withdrawApproved.Code != http.StatusBadRequest || !strings.Contains(withdrawApproved.Body.String(), "already resolved") {
		t.Fatalf("approved approval should not be withdrawable, status=%d body=%s", withdrawApproved.Code, withdrawApproved.Body.String())
	}
	loaded, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approvedApproval.ID)
	if err != nil || !ok {
		t.Fatalf("get approved approval after withdraw attempt: ok=%v err=%v", ok, err)
	}
	if loaded.Status != domain.PermissionPackageApprovalStatusApproved {
		t.Fatalf("approved approval should remain approved, got %#v", loaded)
	}
}

func TestPermissionPackageApprovalReviewerUsesAuthenticatedAdminIdentity(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
	})
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	approval := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests", input, "", "requester-key"))
	if approval.RequestedBy != "requester" {
		t.Fatalf("approval request should be attributed to authenticated requester, got %#v", approval)
	}
	impersonation := requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", map[string]any{
		"reviewer": "requester",
	}, "", "security-key")
	if impersonation.Code != http.StatusForbidden || !strings.Contains(impersonation.Body.String(), "authenticated admin identity") {
		t.Fatalf("reviewer impersonation should be rejected, status=%d body=%s", impersonation.Code, impersonation.Body.String())
	}
	approved := decodeData[permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+approval.ID+"/approve", nil, "", "security-key"))
	if approved.Status != "approved" || approved.ReviewedBy != "security" {
		t.Fatalf("approval should use authenticated reviewer identity, got %#v", approved)
	}
}

func TestPermissionPackageApprovalReviewerQueueUsesAuthenticatedAdminIdentity(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 30, 0, 0, time.UTC)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-auth-east-support", "tenant-east", "ws-support", now)
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	impersonatedQueue := requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security&limit=10", nil, "", "requester-key")
	if impersonatedQueue.Code != http.StatusForbidden || !strings.Contains(impersonatedQueue.Body.String(), "authenticated admin identity") {
		t.Fatalf("reviewer queue impersonation should be rejected, status=%d body=%s", impersonatedQueue.Code, impersonatedQueue.Body.String())
	}

	securityQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security&limit=10", nil, "", "security-key"))
	if len(securityQueue) != 1 || securityQueue[0].ID != eastSupport.ID {
		t.Fatalf("authenticated reviewer should see routed queue, got %#v", securityQueue)
	}
}

func TestPermissionPackageApprovalReviewerQueueDefaultsToAuthenticatedReviewer(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 40, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-default-east-support", "tenant-east", "ws-support", now)
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-default-east-finance", "tenant-east", "ws-finance", now.Add(time.Minute))
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "security", Key: "security-key", Role: "security_reviewer", TenantID: "tenant-east"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	securityQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&limit=10", nil, "", "security-key"))
	if len(securityQueue) != 1 || securityQueue[0].ID != eastSupport.ID {
		t.Fatalf("authenticated reviewer default queue should be routed, got %#v", securityQueue)
	}

	platformQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&limit=10", nil, "", "platform-key"))
	if len(platformQueue) != 2 {
		t.Fatalf("platform admin should keep all-queue visibility when reviewer is omitted, got %#v", platformQueue)
	}
}

func TestManagementMCPApprovalReviewerUsesAuthenticatedAdminIdentity(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 45, 0, 0, time.UTC)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-auth-east-support", "tenant-east", "ws-support", now)
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "requester", Key: "requester-key"},
		{Actor: "security", Key: "security-key"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	impersonatedList := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-impersonation",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"reviewer": "security",
				"status":   "pending",
				"limit":    10,
			},
		},
	}, "", "requester-key")
	var impersonatedListEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(impersonatedList.Body.Bytes(), &impersonatedListEnvelope); err != nil {
		t.Fatalf("decode impersonated MCP list envelope: %v body=%s", err, impersonatedList.Body.String())
	}
	if impersonatedListEnvelope.Error == nil || !strings.Contains(impersonatedListEnvelope.Error.Message, "authenticated admin identity") {
		t.Fatalf("expected MCP reviewer list impersonation rejection, got %#v body=%s", impersonatedListEnvelope, impersonatedList.Body.String())
	}

	impersonatedApprove := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve-impersonation",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": map[string]any{
				"id":       eastSupport.ID,
				"reviewer": "security",
			},
		},
	}, "", "requester-key")
	var impersonatedApproveEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(impersonatedApprove.Body.Bytes(), &impersonatedApproveEnvelope); err != nil {
		t.Fatalf("decode impersonated MCP approve envelope: %v body=%s", err, impersonatedApprove.Body.String())
	}
	if impersonatedApproveEnvelope.Error == nil || !strings.Contains(impersonatedApproveEnvelope.Error.Message, "authenticated admin identity") {
		t.Fatalf("expected MCP reviewer approve impersonation rejection, got %#v body=%s", impersonatedApproveEnvelope, impersonatedApprove.Body.String())
	}
	stillPending, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), eastSupport.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after MCP impersonation attempts: ok=%v err=%v", ok, err)
	}
	if stillPending.Status != domain.PermissionPackageApprovalStatusPending || stillPending.ReviewedBy != "" {
		t.Fatalf("MCP reviewer impersonation should not mutate approval request: %#v", stillPending)
	}

	approveCall := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve-authenticated",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": map[string]any{
				"reviewer": "security",
				"id":       eastSupport.ID,
			},
		},
	}, "", "security-key"))
	var approved permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(approveCall.Result.StructuredContent, &approved); err != nil {
		t.Fatalf("decode authenticated MCP approval: %v", err)
	}
	if approved.Status != "approved" || approved.ReviewedBy != "security" {
		t.Fatalf("authenticated MCP reviewer should approve as themselves, got %#v", approved)
	}
}

func TestManagementMCPApprovalReviewerQueueDefaultsToAuthenticatedReviewer(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 50, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-default-east-support", "tenant-east", "ws-support", now)
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-default-east-finance", "tenant-east", "ws-finance", now.Add(time.Minute))
	router := newRouterWithRepoAdminIdentitiesAndApprovalReviewers(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "security", Key: "security-key", Role: "security_reviewer", TenantID: "tenant-east"},
	}, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	securityCall := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-default-reviewer",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"status": "pending",
				"limit":  10,
			},
		},
	}, "", "security-key"))
	var securityQueue []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(securityCall.Result.StructuredContent, &securityQueue); err != nil {
		t.Fatalf("decode authenticated MCP default reviewer queue: %v", err)
	}
	if len(securityQueue) != 1 || securityQueue[0].ID != eastSupport.ID {
		t.Fatalf("authenticated MCP reviewer default queue should be routed, got %#v", securityQueue)
	}

	platformCall := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-default-platform",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"status": "pending",
				"limit":  10,
			},
		},
	}, "", "platform-key"))
	var platformQueue []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(platformCall.Result.StructuredContent, &platformQueue); err != nil {
		t.Fatalf("decode platform MCP default approval queue: %v", err)
	}
	if len(platformQueue) != 2 {
		t.Fatalf("platform admin should keep all-queue MCP visibility when reviewer is omitted, got %#v", platformQueue)
	}
}

func TestPermissionPackageApprovalReviewerRouting(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-east-support", "tenant-east", "ws-support", now)
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-east-sales", "tenant-east", "ws-sales", now.Add(time.Minute))
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-west-support", "tenant-west", "ws-support", now.Add(2*time.Minute))
	router := newRouterWithRepoAndApprovalReviewers(repo, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security-east", TenantID: "tenant-east", WorkspaceID: "ws-support"},
		{Reviewer: "security-root", TenantID: "tenant-root", WorkspaceID: "*"},
		{Reviewer: "security-root", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	eastQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security-east&limit=10", nil, ""))
	if len(eastQueue) != 1 || eastQueue[0].ID != eastSupport.ID {
		t.Fatalf("expected security-east to see only east support approvals, got %#v", eastQueue)
	}
	eastSalesQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security-east&workspaceId=ws-sales&limit=10", nil, ""))
	if len(eastSalesQueue) != 0 {
		t.Fatalf("expected security-east workspace route to exclude east sales approvals, got %#v", eastSalesQueue)
	}
	rootQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security-root&limit=10", nil, ""))
	if len(rootQueue) != 3 {
		t.Fatalf("expected root reviewer to see deduplicated tenant subtree approvals, got %#v", rootQueue)
	}
	rootEastQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?tenantId=tenant-east&status=pending&reviewer=security-root&limit=10", nil, ""))
	if len(rootEastQueue) != 2 {
		t.Fatalf("expected root reviewer tenant query to narrow to east approvals, got %#v", rootEastQueue)
	}
	limitedRootQueue := decodeData[[]permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodGet, "/api/v1/permission-packages/approval-requests?status=pending&reviewer=security-root&limit=2", nil, ""))
	if len(limitedRootQueue) != 2 || limitedRootQueue[0].ID != "ppar-west-support" || limitedRootQueue[1].ID != "ppar-east-sales" {
		t.Fatalf("expected root reviewer queue to be sorted and limited after dedupe, got %#v", limitedRootQueue)
	}

	unauthorized := request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+eastSupport.ID+"/approve", map[string]any{
		"reviewer": "security-root-ws-sales",
		"comment":  "wrong workspace",
	}, "")
	if unauthorized.Code != http.StatusForbidden || !strings.Contains(unauthorized.Body.String(), "not allowed") {
		t.Fatalf("expected unauthorized reviewer rejection, status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	stillPending, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), eastSupport.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after unauthorized review: ok=%v err=%v", ok, err)
	}
	if stillPending.Status != domain.PermissionPackageApprovalStatusPending || stillPending.ReviewedBy != "" {
		t.Fatalf("unauthorized review should not mutate approval request: %#v", stillPending)
	}

	approved := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+eastSupport.ID+"/approve", map[string]any{
		"reviewer": "security-east",
		"comment":  "approved by routed reviewer",
	}, ""))
	if approved.Status != "approved" || approved.ReviewedBy != "security-east" || approved.ReviewComment != "approved by routed reviewer" {
		t.Fatalf("unexpected routed approval result: %#v", approved)
	}
}

func TestManagementMCPPermissionPackageApprovalReviewerRouting(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 6, 6, 11, 0, 0, 0, time.UTC)
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	createDirectTenant(t, repo, "tenant-west", "tenant-root", "West tenant", now)
	eastSupport := createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-east-support", "tenant-east", "ws-support", now)
	createDirectPermissionPackageApprovalRequest(t, repo, "ppar-mcp-west-support", "tenant-west", "ws-support", now.Add(time.Minute))
	router := newRouterWithRepoAndApprovalReviewers(repo, []domain.PermissionPackageApprovalReviewer{
		{Reviewer: "security-east", TenantID: "tenant-east", WorkspaceID: "ws-support"},
		{Reviewer: "security-west", TenantID: "tenant-west", WorkspaceID: "ws-support"},
	})

	listCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list-routed",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"reviewer": "security-east",
				"status":   "pending",
				"limit":    10,
			},
		},
	}, ""))
	var approvals []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(listCall.Result.StructuredContent, &approvals); err != nil {
		t.Fatalf("decode routed approvals structured content: %v", err)
	}
	if len(approvals) != 1 || approvals[0].ID != eastSupport.ID {
		t.Fatalf("expected routed MCP approval queue, got %#v", approvals)
	}

	unauthorized := request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve-denied",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": map[string]any{
				"id":       eastSupport.ID,
				"reviewer": "security-west",
			},
		},
	}, "")
	var unauthorizedEnvelope mcpEnvelopeResponse
	if err := json.Unmarshal(unauthorized.Body.Bytes(), &unauthorizedEnvelope); err != nil {
		t.Fatalf("decode unauthorized MCP envelope: %v body=%s", err, unauthorized.Body.String())
	}
	if unauthorizedEnvelope.Error == nil || !strings.Contains(unauthorizedEnvelope.Error.Message, "not allowed") {
		t.Fatalf("expected routed MCP approval rejection, got %#v body=%s", unauthorizedEnvelope, unauthorized.Body.String())
	}

	approveCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve-routed",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": map[string]any{
				"id":       eastSupport.ID,
				"reviewer": "security-east",
				"comment":  "approved through routed MCP queue",
			},
		},
	}, ""))
	var approved permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(approveCall.Result.StructuredContent, &approved); err != nil {
		t.Fatalf("decode routed MCP approval result: %v", err)
	}
	if approved.Status != "approved" || approved.ReviewedBy != "security-east" {
		t.Fatalf("unexpected routed MCP approval result: %#v", approved)
	}
}

func TestPermissionPackageDraftDetectsDataScopeConflicts(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithActionAndScopes(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, []domain.DataScope{{
		DataDomain:   "crm",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-east'",
	}}, now)

	input := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "eu-west",
		"requestText":      "给销售助手开通客户只读。",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	draft := decodeData[permissionPackageDraftResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/drafts", input, ""))
	if draft.Readiness.CanApply || len(draft.Readiness.Warnings) == 0 {
		t.Fatalf("expected data-scope conflict warning, got %#v", draft.Readiness)
	}
	applyResp := request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", input, "")
	if applyResp.Code != http.StatusBadRequest {
		t.Fatalf("expected apply to reject conflicting data scopes, status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
}

func TestManagementMCPToolsListAndPermissionPackageCalls(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	exportContracts := createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	tools := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "tools-list",
		"method":  "tools/list",
	}, ""))
	if !mcpToolNamesContain(tools.Result.Tools, "draft_permission_package") ||
		!mcpToolNamesContain(tools.Result.Tools, "preflight_permission_package") ||
		!mcpToolNamesContain(tools.Result.Tools, "apply_permission_package") ||
		!mcpToolNamesContain(tools.Result.Tools, "create_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "list_permission_package_approval_requests") ||
		!mcpToolNamesContain(tools.Result.Tools, "approve_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "reject_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "withdraw_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "list_permission_package_applications") ||
		!mcpToolNamesContain(tools.Result.Tools, "check_permission_package_production_readiness") ||
		!mcpToolNamesContain(tools.Result.Tools, "export_permission_package_production_evidence") {
		t.Fatalf("management MCP tools missing permission package tools: %#v", tools.Result.Tools)
	}

	args := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	draftCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "draft",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "draft_permission_package",
			"arguments": args,
		},
	}, ""))
	var draft permissionPackageDraftResponse
	if err := json.Unmarshal(draftCall.Result.StructuredContent, &draft); err != nil {
		t.Fatalf("decode draft structured content: %v", err)
	}
	if !draft.Readiness.CanApply || len(draft.AllowedCapabilities) != 1 || draft.AllowedCapabilities[0].ID != search.ID {
		t.Fatalf("unexpected management MCP draft: %#v", draft)
	}

	preflightCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "preflight",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "preflight_permission_package",
			"arguments": args,
		},
	}, ""))
	var preflight permissionPackageApplyPreflightResponse
	if err := json.Unmarshal(preflightCall.Result.StructuredContent, &preflight); err != nil {
		t.Fatalf("decode preflight structured content: %v", err)
	}
	if !preflight.Summary.CanApply || preflight.Summary.BlockingCount != 0 ||
		!permissionPackagePreflightHasCheck(preflight.Checks, "planned_changes", "info") {
		t.Fatalf("unexpected management MCP preflight: %#v", preflight)
	}
	entitlements, err := repo.ListTenantEntitlements(t.Context(), store.EntitlementFilter{})
	if err != nil {
		t.Fatalf("list entitlements after MCP preflight: %v", err)
	}
	if len(entitlements) != 0 {
		t.Fatalf("management MCP preflight must not create entitlements: %#v", entitlements)
	}

	appliedCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "apply",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "apply_permission_package",
			"arguments": args,
		},
	}, ""))
	var applied permissionPackageApplyResponse
	if err := json.Unmarshal(appliedCall.Result.StructuredContent, &applied); err != nil {
		t.Fatalf("decode apply structured content: %v", err)
	}
	if len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != search.ID {
		t.Fatalf("expected one applied entitlement, got %#v", applied.TenantEntitlements)
	}
	applicationsCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "applications",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_applications",
			"arguments": map[string]any{
				"tenantId":         "tenant-root",
				"workspaceId":      "ws-sales",
				"templateId":       "sales-readonly",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"limit":            1,
			},
		},
	}, ""))
	var applications []permissionPackageApplicationResponse
	if err := json.Unmarshal(applicationsCall.Result.StructuredContent, &applications); err != nil {
		t.Fatalf("decode applications structured content: %v", err)
	}
	if applied.Application == nil || len(applications) != 1 || applications[0].ID != applied.Application.ID || applications[0].TemplateVersion != 1 {
		t.Fatalf("unexpected management MCP applications: applied=%#v rows=%#v", applied.Application, applications)
	}
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionDenied, caller, target, exportContracts, "export_contracts", "user:sales-001", now.Add(time.Minute))
	appendPermissionPackageReadinessTrace(t, repo, domain.TraceDecisionAllowed, caller, target, search, "search_customer", "user:sales-001", now.Add(2*time.Minute))
	readinessCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "production-readiness",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "check_permission_package_production_readiness",
			"arguments": map[string]any{
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-sales",
				"templateId":       "sales-readonly",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"subjectId":        "user:sales-001",
			},
		},
	}, ""))
	var readiness permissionPackageProductionReadinessResponse
	if err := json.Unmarshal(readinessCall.Result.StructuredContent, &readiness); err != nil {
		t.Fatalf("decode production readiness structured content: %v", err)
	}
	if readiness.Status != "ready" || readiness.LatestApplication == nil || readiness.LatestApplication.ID != applied.Application.ID ||
		!permissionPackageProductionReadinessHasCheck(readiness.Checks, "runtime_allowed_trace_present", "passed") ||
		!permissionPackageProductionReadinessHasCheck(readiness.Checks, "applied_audit_event_present", "passed") {
		t.Fatalf("unexpected production readiness MCP result: %#v", readiness)
	}
	reportCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "production-evidence",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "export_permission_package_production_evidence",
			"arguments": map[string]any{
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-sales",
				"templateId":       "sales-readonly",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"subjectId":        "user:sales-001",
			},
		},
	}, ""))
	var report permissionPackageProductionEvidenceReportResponse
	if err := json.Unmarshal(reportCall.Result.StructuredContent, &report); err != nil {
		t.Fatalf("decode production evidence report structured content: %v", err)
	}
	if report.ReportVersion != "production-readiness-report/v1" || report.Status != "ready" ||
		report.Evidence.Application.ID != applied.Application.ID ||
		report.Evidence.Runtime.AllowedTraceID == "" ||
		report.Evidence.Audit.AppliedEventID == "" {
		t.Fatalf("unexpected production evidence MCP report: %#v", report)
	}
	events := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?action=permission_package.applied", nil, ""))
	if len(events) != 1 || events[0].ResourceType != "permission_package" {
		t.Fatalf("expected permission package audit event, got %#v", events)
	}
}

func TestManagementMCPPermissionPackageApprovalRequestFlow(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	updateTicket := createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	args := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "us-east",
		"requestText":      "Allow support triage updates for this tenant.",
		"subjectSelector":  "user:support-*",
		"targetId":         target.ID,
		"templateId":       "support-ticket-triage",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-support",
	}
	draftCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "draft",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "draft_permission_package",
			"arguments": args,
		},
	}, ""))
	var draft permissionPackageDraftResponse
	if err := json.Unmarshal(draftCall.Result.StructuredContent, &draft); err != nil {
		t.Fatalf("decode draft structured content: %v", err)
	}
	if !draft.Readiness.CanApply || draft.PolicyGate.Decision != "approval_required" || draft.PolicyGate.CanApplyDirectly {
		t.Fatalf("expected approval-required management MCP draft, got %#v", draft)
	}

	missingApprovalPreflightCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "preflight-missing-approval",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "preflight_permission_package",
			"arguments": args,
		},
	}, ""))
	var missingApprovalPreflight permissionPackageApplyPreflightResponse
	if err := json.Unmarshal(missingApprovalPreflightCall.Result.StructuredContent, &missingApprovalPreflight); err != nil {
		t.Fatalf("decode missing approval preflight structured content: %v", err)
	}
	if missingApprovalPreflight.Summary.CanApply || !permissionPackagePreflightHasCheck(missingApprovalPreflight.Checks, "approval_request_missing", "blocking") {
		t.Fatalf("expected missing approval MCP preflight to block, got %#v", missingApprovalPreflight)
	}

	createCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-create",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "create_permission_package_approval_request",
			"arguments": args,
		},
	}, ""))
	var approval permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(createCall.Result.StructuredContent, &approval); err != nil {
		t.Fatalf("decode approval structured content: %v", err)
	}
	if approval.Status != "pending" || approval.TemplateID != "support-ticket-triage" ||
		len(approval.AllowedCapabilityIDs) != 1 || approval.AllowedCapabilityIDs[0] != updateTicket.ID {
		t.Fatalf("unexpected management MCP approval request: %#v", approval)
	}

	listCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-list",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_permission_package_approval_requests",
			"arguments": map[string]any{
				"tenantId":         "tenant-root",
				"workspaceId":      "ws-support",
				"templateId":       "support-ticket-triage",
				"targetId":         target.ID,
				"callerInstanceId": caller.ID,
				"status":           "pending",
				"limit":            1,
			},
		},
	}, ""))
	var approvals []permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(listCall.Result.StructuredContent, &approvals); err != nil {
		t.Fatalf("decode approvals structured content: %v", err)
	}
	if len(approvals) != 1 || approvals[0].ID != approval.ID {
		t.Fatalf("unexpected management MCP approval request list: %#v", approvals)
	}

	withdrawCreateCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-create-withdraw",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "create_permission_package_approval_request",
			"arguments": args,
		},
	}, ""))
	var withdrawApproval permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(withdrawCreateCall.Result.StructuredContent, &withdrawApproval); err != nil {
		t.Fatalf("decode withdraw approval structured content: %v", err)
	}
	withdrawCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-withdraw",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "withdraw_permission_package_approval_request",
			"arguments": map[string]any{
				"id":      withdrawApproval.ID,
				"comment": "replaced by narrower request",
			},
		},
	}, ""))
	var withdrawn permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(withdrawCall.Result.StructuredContent, &withdrawn); err != nil {
		t.Fatalf("decode withdrawn structured content: %v", err)
	}
	if withdrawn.Status != "withdrawn" || withdrawn.ReviewedBy != "local-dev" || withdrawn.ReviewComment != "replaced by narrower request" {
		t.Fatalf("unexpected management MCP withdrawn request: %#v", withdrawn)
	}

	approveCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "approval-approve",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "approve_permission_package_approval_request",
			"arguments": map[string]any{
				"id":       approval.ID,
				"reviewer": "security",
				"comment":  "approved via MCP",
			},
		},
	}, ""))
	var approved permissionPackageApprovalRequestResponse
	if err := json.Unmarshal(approveCall.Result.StructuredContent, &approved); err != nil {
		t.Fatalf("decode approved structured content: %v", err)
	}
	if approved.Status != "approved" || approved.ReviewedBy != "security" {
		t.Fatalf("unexpected management MCP approved request: %#v", approved)
	}

	applyArgs := map[string]any{
		"approvalRequestId": approval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "us-east",
		"requestText":       "Allow support triage updates for this tenant.",
		"subjectSelector":   "user:support-*",
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
	}
	approvedPreflightCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "preflight-approved",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "preflight_permission_package",
			"arguments": applyArgs,
		},
	}, ""))
	var approvedPreflight permissionPackageApplyPreflightResponse
	if err := json.Unmarshal(approvedPreflightCall.Result.StructuredContent, &approvedPreflight); err != nil {
		t.Fatalf("decode approved preflight structured content: %v", err)
	}
	if !approvedPreflight.Summary.CanApply || !approvedPreflight.Summary.ApprovalReady ||
		!permissionPackagePreflightHasCheck(approvedPreflight.Checks, "approval_request_ready", "passed") {
		t.Fatalf("expected approved MCP preflight to pass, got %#v", approvedPreflight)
	}
	loadedApproval, ok, err := repo.GetPermissionPackageApprovalRequest(t.Context(), approval.ID)
	if err != nil || !ok {
		t.Fatalf("get approval after MCP preflight: ok=%v err=%v", ok, err)
	}
	if !loadedApproval.ConsumedAt.IsZero() {
		t.Fatalf("MCP preflight must not consume approval, got %#v", loadedApproval)
	}

	appliedCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "apply",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "apply_permission_package",
			"arguments": applyArgs,
		},
	}, ""))
	var applied permissionPackageApplyResponse
	if err := json.Unmarshal(appliedCall.Result.StructuredContent, &applied); err != nil {
		t.Fatalf("decode apply structured content: %v", err)
	}
	if len(applied.TenantEntitlements) != 1 || applied.TenantEntitlements[0].CapabilityID != updateTicket.ID ||
		applied.Application == nil || applied.Application.TemplateID != "support-ticket-triage" {
		t.Fatalf("expected approved management MCP apply, got %#v", applied)
	}
	events := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?action=permission_package.applied", nil, ""))
	if len(events) != 1 || events[0].Metadata["approvalRequestId"] != approval.ID {
		t.Fatalf("expected management MCP applied audit with approval request id, got %#v", events)
	}
}

func TestManagementMCPReadTools(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	capability := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)

	agentsCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "agents",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_agents",
			"arguments": map[string]any{
				"tenantId": "tenant-east",
			},
		},
	}, ""))
	var agents []agentResponse
	if err := json.Unmarshal(agentsCall.Result.StructuredContent, &agents); err != nil {
		t.Fatalf("decode agents structured content: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != caller.ID {
		t.Fatalf("unexpected agent list: %#v", agents)
	}

	capabilitiesCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "capabilities",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "list_capabilities",
			"arguments": map[string]any{
				"targetId": target.ID,
			},
		},
	}, ""))
	var capabilities []capabilityResponse
	if err := json.Unmarshal(capabilitiesCall.Result.StructuredContent, &capabilities); err != nil {
		t.Fatalf("decode capabilities structured content: %v", err)
	}
	if len(capabilities) != 1 || capabilities[0].ID != capability.ID {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}

	profileCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "profile",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_tenant_access_profile",
			"arguments": map[string]any{
				"tenantId":    "tenant-east",
				"workspaceId": "ws-sales",
			},
		},
	}, ""))
	var profile struct {
		Summary struct {
			TenantCount int `json:"tenantCount"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(profileCall.Result.StructuredContent, &profile); err != nil {
		t.Fatalf("decode profile structured content: %v", err)
	}
	if profile.Summary.TenantCount != 1 {
		t.Fatalf("unexpected profile summary: %#v", profile.Summary)
	}
}

func TestManagementMCPExplainPermissionPackageDraft(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithActionAndScopes(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, []domain.DataScope{{
		DataDomain:   "crm",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-east'",
	}}, now)
	createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	call := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "explain-draft",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "explain_permission_package_draft",
			"arguments": map[string]any{
				"callerInstanceId": caller.ID,
				"region":           "eu-west",
				"requestText":      "给销售助手开通客户只读。",
				"subjectSelector":  "user:sales-*",
				"targetId":         target.ID,
				"templateId":       "sales-readonly",
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-sales",
			},
		},
	}, ""))
	var explanation managementMCPExplainPermissionPackageResponse
	if err := json.Unmarshal(call.Result.StructuredContent, &explanation); err != nil {
		t.Fatalf("decode draft explanation: %v", err)
	}
	if explanation.Outcome != "blocked" || explanation.Readiness.CanApply {
		t.Fatalf("expected blocked draft explanation, got %#v", explanation)
	}
	if !containsString(explanation.Readiness.Warnings, "Permission package data scopes exceed capability boundary for search_customer.") {
		t.Fatalf("expected data-scope warning, got %#v", explanation.Readiness.Warnings)
	}
	if len(explanation.BlockedSimulationRows) == 0 || len(explanation.NextActions) == 0 || explanation.Summary == "" {
		t.Fatalf("expected blocked rows, next actions, and summary, got %#v", explanation)
	}
}

func TestManagementMCPExplainPermissionPackageDraftRequiresApproval(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-support", Name: "Support Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Support MCP", "tenant-root", "ws-support", "mcp", domain.AgentStatusActive, nil)
	createDirectCapabilityWithAction(t, repo, target.ID, "update_ticket", domain.CapabilityActionWrite, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	call := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "explain-approval",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "explain_permission_package_draft",
			"arguments": map[string]any{
				"callerInstanceId": caller.ID,
				"region":           "us-east",
				"requestText":      "Allow support triage updates for this tenant.",
				"subjectSelector":  "user:support-*",
				"targetId":         target.ID,
				"templateId":       "support-ticket-triage",
				"tenantId":         "tenant-east",
				"workspaceId":      "ws-support",
			},
		},
	}, ""))
	var explanation managementMCPExplainPermissionPackageResponse
	if err := json.Unmarshal(call.Result.StructuredContent, &explanation); err != nil {
		t.Fatalf("decode approval draft explanation: %v", err)
	}
	if explanation.Outcome != "approval_required" || !explanation.Readiness.CanApply {
		t.Fatalf("expected approval-required ready draft explanation, got %#v", explanation)
	}
	if explanation.PolicyGate.Decision != "approval_required" || explanation.PolicyGate.CanApplyDirectly || len(explanation.PolicyGate.Reasons) == 0 {
		t.Fatalf("expected policy gate in MCP explanation, got %#v", explanation.PolicyGate)
	}
	if !strings.Contains(explanation.Summary, "requires approval") {
		t.Fatalf("expected approval summary, got %q", explanation.Summary)
	}
	if !strings.Contains(strings.Join(explanation.NextActions, " "), "approval") {
		t.Fatalf("expected approval next action, got %#v", explanation.NextActions)
	}
	if strings.Contains(strings.Join(explanation.NextActions, " "), "Apply the permission package") {
		t.Fatalf("approval-required draft should not suggest direct apply, got %#v", explanation.NextActions)
	}
}

func TestManagementMCPExplainAccessDecision(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	search.DiscoveryStatus = domain.CapabilityDiscoveryApproved
	search.UpdatedAt = now
	if _, ok, err := repo.UpdateCapability(t.Context(), search); err != nil || !ok {
		t.Fatalf("approve capability: ok=%v err=%v", ok, err)
	}

	explainArgs := map[string]any{
		"callerInstanceId": caller.ID,
		"capabilityId":     search.ID,
		"subjectId":        "user:sales-001",
		"targetId":         target.ID,
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	deniedCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "explain-denied",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "explain_access_decision",
			"arguments": explainArgs,
		},
	}, ""))
	var denied managementMCPExplainAccessResponse
	if err := json.Unmarshal(deniedCall.Result.StructuredContent, &denied); err != nil {
		t.Fatalf("decode denied explanation: %v", err)
	}
	if denied.Outcome != "denied" || denied.Decision.Allowed || denied.Decision.Reason != "tenant has no entitlement for capability" {
		t.Fatalf("unexpected denied explanation: %#v", denied)
	}
	if !strings.Contains(strings.Join(denied.NextActions, " "), "permission package") {
		t.Fatalf("expected permission package next action, got %#v", denied.NextActions)
	}

	packageArgs := map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}
	_ = decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "apply",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "apply_permission_package",
			"arguments": packageArgs,
		},
	}, ""))

	allowedCall := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "explain-allowed",
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "explain_access_decision",
			"arguments": explainArgs,
		},
	}, ""))
	var allowed managementMCPExplainAccessResponse
	if err := json.Unmarshal(allowedCall.Result.StructuredContent, &allowed); err != nil {
		t.Fatalf("decode allowed explanation: %v", err)
	}
	if allowed.Outcome != "allowed" || !allowed.Decision.Allowed || len(allowed.DataScopes) == 0 {
		t.Fatalf("unexpected allowed explanation: %#v", allowed)
	}
	if !explainEvidenceContains(allowed.Evidence, "tenant_entitlement") ||
		!explainEvidenceContains(allowed.Evidence, "workspace_assignment") ||
		!explainEvidenceContains(allowed.Evidence, "instance_assignment") {
		t.Fatalf("expected entitlement/workspace/instance evidence, got %#v", allowed.Evidence)
	}
}

func TestAccessDecisionExplain(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	createDirectTenant(t, repo, "tenant-root", "", "Root tenant", now)
	createDirectTenant(t, repo, "tenant-east", "tenant-root", "East tenant", now)
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-east", WorkspaceID: "ws-sales", Name: "Sales Assistant", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "CRM MCP", "tenant-root", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	search := createDirectCapabilityWithAction(t, repo, target.ID, "search_customer", domain.CapabilityActionRead, domain.CapabilityRiskLow, domain.CapabilitySensitivityInternal, now)
	search.DiscoveryStatus = domain.CapabilityDiscoveryApproved
	search.UpdatedAt = now
	if _, ok, err := repo.UpdateCapability(t.Context(), search); err != nil || !ok {
		t.Fatalf("approve capability: ok=%v err=%v", ok, err)
	}

	explainPath := "/api/v1/access-decisions:explain?tenantId=tenant-east&workspaceId=ws-sales&callerInstanceId=" + caller.ID + "&targetId=" + target.ID + "&capabilityId=" + search.ID + "&subjectId=user:sales-001"
	denied := decodeData[managementMCPExplainAccessResponse](t, request(t, router, http.MethodGet, explainPath, nil, ""))
	if denied.Outcome != "denied" || denied.Decision.Allowed || denied.Decision.Reason != "tenant has no entitlement for capability" {
		t.Fatalf("unexpected denied explanation: %#v", denied)
	}
	if !strings.Contains(strings.Join(denied.NextActions, " "), "permission package") {
		t.Fatalf("expected permission package next action, got %#v", denied.NextActions)
	}

	applied := decodeData[permissionPackageApplyResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages:apply", map[string]any{
		"callerInstanceId": caller.ID,
		"region":           "华东",
		"requestText":      "给销售助手开通客户只读。",
		"subjectSelector":  "user:sales-*",
		"targetId":         target.ID,
		"templateId":       "sales-readonly",
		"tenantId":         "tenant-east",
		"workspaceId":      "ws-sales",
	}, ""))
	if applied.Application == nil {
		t.Fatalf("expected permission package application evidence")
	}

	allowed := decodeData[managementMCPExplainAccessResponse](t, request(t, router, http.MethodGet, explainPath, nil, ""))
	if allowed.Outcome != "allowed" || !allowed.Decision.Allowed || len(allowed.DataScopes) == 0 {
		t.Fatalf("unexpected allowed explanation: %#v", allowed)
	}
	if !explainEvidenceContains(allowed.Evidence, "tenant_entitlement") ||
		!explainEvidenceContains(allowed.Evidence, "workspace_assignment") ||
		!explainEvidenceContains(allowed.Evidence, "instance_assignment") {
		t.Fatalf("expected entitlement/workspace/instance evidence, got %#v", allowed.Evidence)
	}
}

func TestCapabilityAssignmentDataScopesMustNarrowHierarchy(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	now := time.Now().UTC()
	caller := domain.Agent{ID: security.NewID("agt"), TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "Scoped Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(t.Context(), caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	target := createDirectAgent(t, repo, "Scoped MCP", "tenant-a", "ws-sales", "mcp", domain.AgentStatusActive, nil)
	capability := domain.Capability{
		ID:          security.NewID("cap"),
		TargetID:    target.ID,
		Type:        domain.CapabilityTypeMCPTool,
		Key:         "search_customer",
		DisplayName: "search_customer",
		Action:      domain.CapabilityActionRead,
		DataScopes: []domain.DataScope{{
			DataDomain:   "crm",
			Region:       "us-east",
			TenantFilter: "tenant_id = 'tenant-a'",
		}},
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       domain.CapabilityRiskLow,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(t.Context(), capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}

	badEntitlement := request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-a",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
		"dataScopes": []map[string]any{{
			"dataDomain": "crm",
			"region":     "eu-west",
		}},
	}, "")
	if badEntitlement.Code != http.StatusBadRequest {
		t.Fatalf("tenant entitlement scope expansion should fail, got %d body=%s", badEntitlement.Code, badEntitlement.Body.String())
	}

	entitlement := decodeData[tenantEntitlementResponse](t, request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-a",
		"targetId":     target.ID,
		"capabilityId": capability.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, ""))
	badWorkspace := request(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": entitlement.ID,
		"workspaceId":         "ws-sales",
		"effect":              "allow",
		"status":              "enabled",
		"dataScopes": []map[string]any{{
			"dataDomain": "crm",
			"region":     "eu-west",
		}},
	}, "")
	if badWorkspace.Code != http.StatusBadRequest {
		t.Fatalf("workspace assignment scope expansion should fail, got %d body=%s", badWorkspace.Code, badWorkspace.Body.String())
	}

	workspaceAssignment := decodeData[workspaceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": entitlement.ID,
		"workspaceId":         "ws-sales",
		"effect":              "allow",
		"status":              "enabled",
		"dataScopes": []map[string]any{{
			"table": "accounts",
		}},
	}, ""))
	instanceAssignment := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": workspaceAssignment.ID,
		"callerInstanceId":      caller.ID,
		"subjectSelector":       "user:sales-*",
		"effect":                "allow",
		"status":                "enabled",
	}, ""))
	decision, err := repo.EvaluateCapabilityAccess(t.Context(), store.CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		SubjectID:        "user:sales-001",
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate capability access: %v", err)
	}
	wantScopes := []domain.DataScope{{
		DataDomain:   "crm",
		Table:        "accounts",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-a'",
	}}
	if !decision.Allowed || decision.InstanceAssignmentID != instanceAssignment.ID || !reflect.DeepEqual(decision.DataScopes, wantScopes) {
		t.Fatalf("unexpected decision: %#v want scopes %#v", decision, wantScopes)
	}
}

func TestMCPCapabilityGovernanceFiltersToolsListDeniesUnassignedToolAndTracesEvidence(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "application/json") || !strings.Contains(accept, "text/event-stream") {
			t.Fatalf("MCP upstream request should accept JSON and event-stream responses, got %q", accept)
		}
		var payload struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
			Params struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch payload.Method {
		case "tools/list":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"list-1","result":{"tools":[{"name":"search_customer","description":"Search customers","inputSchema":{"type":"object"}},{"name":"export_contracts","description":"Export contracts","inputSchema":{"type":"object"}}]}}`))
		case "tools/call":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"call-1","result":{"ok":true,"tool":"` + payload.Params.Name + `"}}`))
		default:
			t.Fatalf("unexpected upstream method %q", payload.Method)
		}
	}))
	defer upstream.Close()

	caller := createAgent(t, router, map[string]any{
		"tenantId":    "tenant-a",
		"workspaceId": "ws-sales",
		"name":        "Capability Runtime Caller",
		"channelType": "local",
		"status":      "active",
	})
	key := decodeData[keyResponse](t, request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{"agentId": caller.ID}, ""))
	target := createDirectAgent(t, repo, "Capability Runtime MCP", "tenant-a", "ws-sales", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL,
	})
	capabilities := decodeData[[]capabilityResponse](t, request(t, router, http.MethodPost, "/api/v1/targets/"+target.ID+"/capabilities:refresh", nil, ""))
	search := capabilityByKey(t, capabilities, "search_customer")
	_ = decodeData[capabilityResponse](t, request(t, router, http.MethodPatch, "/api/v1/capabilities/"+search.ID, map[string]any{
		"discoveryStatus": "approved",
	}, ""))
	entitlement := decodeData[tenantEntitlementResponse](t, request(t, router, http.MethodPost, "/api/v1/tenant-entitlements", map[string]any{
		"tenantId":     "tenant-a",
		"targetId":     target.ID,
		"capabilityId": search.ID,
		"effect":       "allow",
		"status":       "enabled",
	}, ""))
	workspaceAssignment := decodeData[workspaceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/workspace-assignments", map[string]any{
		"tenantEntitlementId": entitlement.ID,
		"workspaceId":         "ws-sales",
		"effect":              "allow",
		"status":              "enabled",
	}, ""))
	instanceAssignment := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": workspaceAssignment.ID,
		"callerInstanceId":      caller.ID,
		"subjectSelector":       "user:sales-*",
		"effect":                "allow",
		"status":                "enabled",
	}, ""))

	listResp := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "list-1",
		"method":  "tools/list",
	}, key.Key, "cap-list-run", "user:sales-001")
	if listResp.Code != http.StatusOK {
		t.Fatalf("tools/list failed: status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	if !strings.Contains(listResp.Body.String(), "search_customer") || strings.Contains(listResp.Body.String(), "export_contracts") {
		t.Fatalf("tools/list should include only assigned tool, got %s", listResp.Body.String())
	}

	denied := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "call-denied",
		"method":  "tools/call",
		"params":  map[string]any{"name": "export_contracts", "arguments": map[string]any{}},
	}, key.Key, "cap-denied-run", "user:sales-001")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("unassigned tool should be denied, got %d body=%s", denied.Code, denied.Body.String())
	}

	allowed := requestWithRunIDAndSubject(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "call-allowed",
		"method":  "tools/call",
		"params":  map[string]any{"name": "search_customer", "arguments": map[string]any{"query": "Acme"}},
	}, key.Key, "cap-allowed-run", "user:sales-001")
	if allowed.Code != http.StatusAccepted {
		t.Fatalf("assigned tool should be allowed, got %d body=%s", allowed.Code, allowed.Body.String())
	}

	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=cap-allowed-run", nil, ""))
	if len(traces) != 1 {
		t.Fatalf("expected one allowed trace, got %#v", traces)
	}
	trace := traces[0]
	if trace.TenantID != "tenant-a" || trace.WorkspaceID != "ws-sales" || trace.CallerInstanceID != caller.ID {
		t.Fatalf("trace missing runtime identity: %#v", trace)
	}
	if trace.CapabilityID != search.ID || trace.CapabilityVersion != 1 || trace.EntitlementID != entitlement.ID || trace.WorkspaceAssignmentID != workspaceAssignment.ID || trace.InstanceAssignmentID != instanceAssignment.ID {
		t.Fatalf("trace missing capability evidence: %#v", trace)
	}
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
	capability := domain.Capability{
		ID:              security.NewID("cap"),
		TargetID:        targetID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             key,
		DisplayName:     key,
		Description:     key,
		Action:          action,
		DataDomains:     []string{"crm"},
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
	for _, key := range []string{"tenantId", "workspaceId", "templateId", "targetId", "callerInstanceId", "region", "requestText", "subjectSelector"} {
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

func (r *failingAuditedAgentRepository) CreateAgentWithAudit(ctx context.Context, agent domain.Agent, build store.AgentAuditBuilder) (domain.Agent, error) {
	return domain.Agent{}, errors.New("audit insert failed")
}

func (r *failingAuditedAgentRepository) UpdateAgentWithAudit(ctx context.Context, agent domain.Agent, build store.AgentAuditBuilder) (domain.Agent, bool, error) {
	return domain.Agent{}, false, errors.New("audit insert failed")
}
