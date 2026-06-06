package httpapi_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	ID              string `json:"id"`
	TargetID        string `json:"targetId"`
	Type            string `json:"type"`
	Key             string `json:"key"`
	DisplayName     string `json:"displayName"`
	Description     string `json:"description"`
	Action          string `json:"action"`
	Sensitivity     string `json:"sensitivity"`
	RiskLevel       string `json:"riskLevel"`
	EnforcementMode string `json:"enforcementMode"`
	DiscoveryStatus string `json:"discoveryStatus"`
	Version         int    `json:"version"`
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
	return httpapi.New(store.NewMemory()).Router()
}

func newRouterWithAdmin(adminKey string) http.Handler {
	return httpapi.New(store.NewMemory(), httpapi.WithAdminKey(adminKey)).Router()
}

func newRouterWithRepo(repo store.Repository) http.Handler {
	return httpapi.New(repo).Router()
}

func newRouterWithRepoAndApprovalReviewers(repo store.Repository, reviewers []domain.PermissionPackageApprovalReviewer) http.Handler {
	return httpapi.New(repo, httpapi.WithPermissionPackageApprovalReviewers(reviewers)).Router()
}

func newRouterWithPrivateUpstreams() http.Handler {
	return httpapi.New(store.NewMemory(), httpapi.WithPrivateUpstreamsAllowed(true)).Router()
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
		"targetAgentId": sameTenantOtherWorkspace.ID,
		"routeType":     "mcp",
		"routeKey":      "tools/call",
	}, ""))
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
		TargetID:  sameTenantOtherWorkspace.ID,
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
		CallerID:  sameTenantOtherWorkspace.ID,
		TargetID:  otherTenantSameWorkspace.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		Decision:  domain.TraceDecisionAllowed,
		CreatedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("append excluded trace: %v", err)
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
		t.Fatalf("expected scoped grants to match caller or target in scope, got %#v", grants)
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces"+scopeQuery+"&runId=scope-run", nil, ""))
	if len(traces) != 1 || traces[0].CallerID != inScope.ID {
		t.Fatalf("expected scoped traces to match caller or target in scope, got %#v", traces)
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
	instanceAssignment := decodeData[instanceAssignmentResponse](t, request(t, router, http.MethodPost, "/api/v1/instance-assignments", map[string]any{
		"workspaceAssignmentId": workspaceAssignment.ID,
		"callerInstanceId":      caller.ID,
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
	approvedApproval := decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+secondApproval.ID+"/approve", nil, ""))
	if approvedApproval.Status != "approved" || approvedApproval.ReviewedBy != "local-dev" {
		t.Fatalf("unexpected approved approval request: %#v", approvedApproval)
	}
	mismatchedApplyInput := map[string]any{
		"approvalRequestId": secondApproval.ID,
		"callerInstanceId":  caller.ID,
		"region":            "eu-west",
		"requestText":       "Allow support triage updates for this tenant.",
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
	thirdApproval = decodeData[permissionPackageApprovalRequestResponse](t, request(t, router, http.MethodPost, "/api/v1/permission-packages/approval-requests/"+thirdApproval.ID+"/approve", nil, ""))
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
	if reusedApply.Code != http.StatusBadRequest || !strings.Contains(reusedApply.Body.String(), "already consumed") {
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
	createDirectCapabilityWithAction(t, repo, target.ID, "export_contracts", domain.CapabilityActionExport, domain.CapabilityRiskHigh, domain.CapabilitySensitivityConfidential, now)

	tools := decodeMCPResult(t, request(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "tools-list",
		"method":  "tools/list",
	}, ""))
	if !mcpToolNamesContain(tools.Result.Tools, "draft_permission_package") ||
		!mcpToolNamesContain(tools.Result.Tools, "apply_permission_package") ||
		!mcpToolNamesContain(tools.Result.Tools, "create_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "list_permission_package_approval_requests") ||
		!mcpToolNamesContain(tools.Result.Tools, "approve_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "reject_permission_package_approval_request") ||
		!mcpToolNamesContain(tools.Result.Tools, "list_permission_package_applications") {
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
		"targetId":          target.ID,
		"templateId":        "support-ticket-triage",
		"tenantId":          "tenant-east",
		"workspaceId":       "ws-support",
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
		"effect":                "allow",
		"status":                "enabled",
	}, ""))
	decision, err := repo.EvaluateCapabilityAccess(t.Context(), store.CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
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
		"effect":                "allow",
		"status":                "enabled",
	}, ""))

	listResp := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "list-1",
		"method":  "tools/list",
	}, key.Key, "cap-list-run")
	if listResp.Code != http.StatusOK {
		t.Fatalf("tools/list failed: status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	if !strings.Contains(listResp.Body.String(), "search_customer") || strings.Contains(listResp.Body.String(), "export_contracts") {
		t.Fatalf("tools/list should include only assigned tool, got %s", listResp.Body.String())
	}

	denied := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "call-denied",
		"method":  "tools/call",
		"params":  map[string]any{"name": "export_contracts", "arguments": map[string]any{}},
	}, key.Key, "cap-denied-run")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("unassigned tool should be denied, got %d body=%s", denied.Code, denied.Body.String())
	}

	allowed := requestWithRunID(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"id":      "call-allowed",
		"method":  "tools/call",
		"params":  map[string]any{"name": "search_customer", "arguments": map[string]any{"query": "Acme"}},
	}, key.Key, "cap-allowed-run")
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
	request := domain.PermissionPackageApprovalRequest{
		ID:                    id,
		DraftID:               "ppd_" + id,
		TemplateID:            "support-ticket-triage",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              tenantID,
		WorkspaceID:           workspaceID,
		TargetID:              "agt_target_" + id,
		CallerInstanceID:      "agt_caller_" + id,
		SubjectSelector:       "user:support-*",
		RequestText:           "Allow support triage updates.",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Region: "us-east"}},
		AllowedCapabilityIDs:  []string{"cap_" + id},
		AllowedCapabilityKeys: []string{"update_ticket"},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
			Reasons: []domain.PermissionPackagePolicyReason{{
				ID:            "policy:" + id,
				CapabilityID:  "cap_" + id,
				CapabilityKey: "update_ticket",
				Severity:      "high",
				Message:       "Approval is required.",
				ReasonKey:     "permissionPolicy.actionApprovalRequired",
				ReasonValues:  map[string]string{"action": "write"},
			}},
			NextActions: []string{"Request approval before applying this permission package."},
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "admin-key",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
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

func requestWithAdmin(t *testing.T, router http.Handler, method string, path string, body any, bearer string, adminKey string) *httptest.ResponseRecorder {
	t.Helper()
	return requestWithRunIDAndAdmin(t, router, method, path, body, bearer, "", adminKey)
}

func requestWithRunIDAndAdmin(t *testing.T, router http.Handler, method string, path string, body any, bearer string, runID string, adminKey string) *httptest.ResponseRecorder {
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
	router.ServeHTTP(rec, req)
	return rec
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
