package httpapi_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

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

func TestMCPRPCRequiresJSONContentTypeWithoutTrace(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Non JSON MCP Caller")
	target := createDirectAgent(t, repo, "Non JSON MCP Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/list")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":"list","method":"tools/list"}`))
	req.Header.Set("Authorization", "Bearer "+key.Key)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Run-Id", "run-mcp-content-type")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("non-JSON MCP RPC should return 415, got %d body=%s", rec.Code, rec.Body.String())
	}
	var env apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "UNSUPPORTED_MEDIA_TYPE" {
		t.Fatalf("expected content-type error, got %#v", env)
	}
	traces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-mcp-content-type", nil, ""))
	if len(traces) != 0 {
		t.Fatalf("invalid MCP content type should not record trace, got %#v", traces)
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
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("Pragma", "cache")
		w.Header().Set("Expires", "Wed, 21 Oct 2099 07:28:00 GMT")
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
	if got := resp.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected AgentHarbor no-store to override upstream cache header, got %q", got)
	}
	if got := resp.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected AgentHarbor nosniff on proxied data-plane response, got %q", got)
	}
	if got := resp.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("expected AgentHarbor no-cache pragma on proxied data-plane response, got %q", got)
	}
	if got := resp.Header().Get("Expires"); got != "0" {
		t.Fatalf("expected AgentHarbor Expires 0 on proxied data-plane response, got %q", got)
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
	if !strings.Contains(env.Message, "nonexistent.invalid") || !strings.Contains(env.Message, "no such host") {
		t.Fatalf("upstream error message should carry the underlying cause, got %q", env.Message)
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

func TestOpenAPIProxyRejectsEncodedPathTraversalBeforeUpstream(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		t.Fatalf("encoded traversal path should not reach upstream: %s", r.URL.String())
	}))
	defer upstream.Close()

	caller, key := createLocalCallerWithKey(t, router, "OpenAPI Traversal Caller")
	target := createDirectAgent(t, repo, "Traversal OpenAPI", "default", "ws-1", "openapi", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL + "/base",
	})
	grantRoute(t, router, caller.ID, target.ID, "openapi", "")

	resp := request(t, router, http.MethodGet, "/api/v1/openapi/agents/"+target.ID+"/%252e%252e/admin", nil, key.Key)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected encoded traversal rejection, got %d body=%s", resp.Code, resp.Body.String())
	}
	if upstreamCalled {
		t.Fatalf("encoded traversal path reached upstream")
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
