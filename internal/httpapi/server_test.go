package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	ID          string `json:"id"`
	TenantID    string `json:"tenantId"`
	Name        string `json:"name"`
	WorkspaceID string `json:"workspaceId"`
	ChannelType string `json:"channelType"`
	Status      string `json:"status"`
}

type keyResponse struct {
	ID      string `json:"id"`
	AgentID string `json:"agentId"`
	Key     string `json:"key"`
	Prefix  string `json:"prefix"`
}

type traceResponse struct {
	ID       string `json:"id"`
	CallerID string `json:"callerAgentId"`
	TargetID string `json:"targetAgentId"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
	RunID    string `json:"runId"`
}

type grantResponse struct {
	ID        string `json:"id"`
	CallerID  string `json:"callerAgentId"`
	TargetID  string `json:"targetAgentId"`
	RouteType string `json:"routeType"`
	RouteKey  string `json:"routeKey"`
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
		"name":        "Local Codex Agent",
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
	if read.Name != "Local Codex Agent" {
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

func TestDataPlaneAllowedDeniedTraces(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller := createAgent(t, router, map[string]any{
		"name":        "Local Codex Agent",
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

func TestProxyUpstreamFailureRecordsTraceAndReturnsBadGateway(t *testing.T) {
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
	if env.Error != "UPSTREAM_ERROR" {
		t.Fatalf("expected UPSTREAM_ERROR, got %#v", env)
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-upstream-fail", nil, ""))
	if len(traces) != 1 || traces[0].Decision != "allowed" || traces[0].TargetID != target.ID {
		t.Fatalf("expected allowed trace recorded before proxy failure, got %#v", traces)
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
	if channelConfig == nil {
		channelConfig = map[string]any{}
	}
	now := time.Now().UTC()
	agent := domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      tenantID,
		WorkspaceID:   workspaceID,
		Name:          name,
		ChannelType:   channelType,
		ChannelConfig: channelConfig,
		Status:        status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	created, err := repo.CreateAgent(t.Context(), agent)
	if err != nil {
		t.Fatalf("create direct agent: %v", err)
	}
	return created
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
