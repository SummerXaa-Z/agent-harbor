package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SummerXaa-Z/ai-nexus-go-rebirth/internal/httpapi"
	"github.com/SummerXaa-Z/ai-nexus-go-rebirth/internal/store"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
}

type agentResponse struct {
	ID          string `json:"id"`
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
	RunID    string `json:"runId"`
}

func newRouter() http.Handler {
	return httpapi.New(store.NewMemory()).Router()
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
	router := newRouter()
	caller := createAgent(t, router, map[string]any{
		"name":        "Local Codex Agent",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	target := createAgent(t, router, map[string]any{
		"name":        "PMM Tracker MCP",
		"workspaceId": "ws-1",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com/mcp",
		},
	})
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
	keys := decodeData[[]keyResponse](t, request(t, router, http.MethodGet, "/api/v1/api-keys", nil, ""))
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
		"callerAgentId": caller.ID,
		"targetAgentId": target.ID,
	}, "")
	if grantResp.Code != http.StatusCreated {
		t.Fatalf("grant create failed: %d", grantResp.Code)
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
	router := newRouter()
	caller := createAgent(t, router, map[string]any{
		"name":        "CI Caller",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	})
	target := createAgent(t, router, map[string]any{
		"name":        "Read-only Ops API",
		"workspaceId": "ws-1",
		"channelType": "openapi",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "https://api.example.com",
		},
	})
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

func createAgent(t *testing.T, router http.Handler, body map[string]any) agentResponse {
	t.Helper()
	resp := request(t, router, http.MethodPost, "/api/v1/agents", body, "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("create agent failed: status=%d body=%s", resp.Code, resp.Body.String())
	}
	return decodeData[agentResponse](t, resp)
}

func request(t *testing.T, router http.Handler, method string, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	return requestWithRunID(t, router, method, path, body, bearer, "")
}

func requestWithRunID(t *testing.T, router http.Handler, method string, path string, body any, bearer string, runID string) *httptest.ResponseRecorder {
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
