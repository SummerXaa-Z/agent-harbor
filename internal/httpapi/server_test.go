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
