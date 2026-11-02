package httpapi_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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
		"capability_data_domain_classification_v1",
		"permission_package_requested_capability_v1",
		"permission_package_approval_requests",
		"permission_package_approval_withdraw",
		"permission_package_apply_preflight",
		"permission_package_applications",
		"permission_package_application_health",
		"permission_package_application_impact",
		"permission_package_production_readiness",
		"permission_package_access_handoff_v1",
		"permission_package_access_handoff_tokens_v1",
		"permission_package_consumed_approval_recovery",
		"management_mcp_tools_metadata_v4",
	} {
		if !capabilities[capability] {
			t.Fatalf("system info missing required console capability %q: %#v", capability, info.Capabilities)
		}
	}
	for _, capability := range []string{"metrics_daily_v1", "target_probe_v1"} {
		if !capabilities[capability] {
			t.Fatalf("system info missing redesigned console capability %q: %#v", capability, info.Capabilities)
		}
	}
	if info.ManagementMcpToolCatalog.MetadataVersion != 4 {
		t.Fatalf("management MCP catalog metadata version = %d, want 4", info.ManagementMcpToolCatalog.MetadataVersion)
	}
	if info.ManagementMcpToolCatalog.ToolCount < 20 {
		t.Fatalf("management MCP catalog toolCount should summarize the current tool catalog: %#v", info.ManagementMcpToolCatalog)
	}
	if info.ManagementMcpToolCatalog.ConfirmationRequiredTools == 0 {
		t.Fatalf("management MCP catalog should report confirmation-required write tools: %#v", info.ManagementMcpToolCatalog)
	}
	if info.ManagementMcpToolCatalog.ToolsWithConfirmationSchema != info.ManagementMcpToolCatalog.ConfirmationRequiredTools {
		t.Fatalf("management MCP catalog confirmation schema summary mismatch: %#v", info.ManagementMcpToolCatalog)
	}
	if len(info.ManagementMcpToolCatalog.CatalogDigest) != 64 {
		t.Fatalf("management MCP catalog digest should be a sha256 hex digest: %#v", info.ManagementMcpToolCatalog)
	}
	requiredMetadata := make(map[string]bool, len(info.ManagementMcpToolCatalog.RequiredMetadata))
	for _, field := range info.ManagementMcpToolCatalog.RequiredMetadata {
		requiredMetadata[field] = true
	}
	for _, field := range []string{"safety", "access", "lifecycle", "execution"} {
		if !requiredMetadata[field] {
			t.Fatalf("management MCP catalog required metadata missing %q: %#v", field, info.ManagementMcpToolCatalog.RequiredMetadata)
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

func assertBrowserSecurityHeaders(t *testing.T, rec *httptest.ResponseRecorder, label string) {
	t.Helper()
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("%s should set Referrer-Policy no-referrer, got %q", label, got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("%s should set X-Frame-Options DENY, got %q", label, got)
	}
	if got := rec.Header().Get("Permissions-Policy"); got != "camera=(), microphone=(), geolocation=()" {
		t.Fatalf("%s should set restrictive Permissions-Policy, got %q", label, got)
	}
}

func TestBrowserSecurityHeaders(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	health := request(t, router, http.MethodGet, "/healthz", nil, "")
	if health.Code != http.StatusOK {
		t.Fatalf("expected public health to succeed, got %d", health.Code)
	}
	assertBrowserSecurityHeaders(t, health, "public health responses")

	unauthorized := request(t, router, http.MethodGet, "/api/v1/agents", nil, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized management read, got %d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	assertBrowserSecurityHeaders(t, unauthorized, "management error responses")

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/agents", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5174")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected CORS preflight to be short-circuited, got %d", rec.Code)
	}
	assertBrowserSecurityHeaders(t, rec, "CORS preflight responses")
}

func TestBrowserSecurityHeadersSetHSTSOnlyForHTTPS(t *testing.T) {
	router := newRouter()
	const expectedHSTS = "max-age=31536000; includeSubDomains"

	plainHTTP := request(t, router, http.MethodGet, "/healthz", nil, "")
	if got := plainHTTP.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("plain HTTP responses should not set HSTS, got %q", got)
	}

	tlsRec, tlsReq := buildRequest(t, http.MethodGet, "/healthz", nil, "", "", "")
	tlsReq.TLS = &tls.ConnectionState{}
	router.ServeHTTP(tlsRec, tlsReq)
	if got := tlsRec.Header().Get("Strict-Transport-Security"); got != expectedHSTS {
		t.Fatalf("TLS responses should set HSTS, got %q", got)
	}

	trustedProxyRec, trustedProxyReq := buildRequest(t, http.MethodGet, "/healthz", nil, "", "", "")
	trustedProxyReq.RemoteAddr = "127.0.0.1:4000"
	trustedProxyReq.Header.Set("X-Forwarded-Proto", "https")
	router.ServeHTTP(trustedProxyRec, trustedProxyReq)
	if got := trustedProxyRec.Header().Get("Strict-Transport-Security"); got != expectedHSTS {
		t.Fatalf("trusted forwarded HTTPS responses should set HSTS, got %q", got)
	}

	untrustedProxyRec, untrustedProxyReq := buildRequest(t, http.MethodGet, "/healthz", nil, "", "", "")
	untrustedProxyReq.RemoteAddr = "198.51.100.20:4000"
	untrustedProxyReq.Header.Set("X-Forwarded-Proto", "https")
	router.ServeHTTP(untrustedProxyRec, untrustedProxyReq)
	if got := untrustedProxyRec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("untrusted forwarded HTTPS should not force HSTS, got %q", got)
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

func TestManagementJSONRejectsOversizedBody(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	payload := `{"name":"` + strings.Repeat("x", 1<<20) + `","workspaceId":"ws-1","channelType":"local","status":"active"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", "test-admin")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected oversized management JSON body to return 413, got %d", rec.Code)
	}
	var env apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("expected PAYLOAD_TOO_LARGE, got %#v", env)
	}
}

func TestManagementJSONRejectsTrailingTokens(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	payload := `{"name":"Trailing Agent","workspaceId":"ws-1","channelType":"local","status":"active"} {}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", "test-admin")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected trailing management JSON body to return 400, got %d", rec.Code)
	}
	var env apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "INVALID_JSON" {
		t.Fatalf("expected INVALID_JSON, got %#v", env)
	}
}

func TestManagementJSONRejectsNonJSONContentType(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	payload := `{"name":"Text Agent","workspaceId":"ws-1","channelType":"local","status":"active"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Admin-Key", "test-admin")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected non-JSON management request to return 415, got %d", rec.Code)
	}
	var env apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error != "UNSUPPORTED_MEDIA_TYPE" {
		t.Fatalf("expected UNSUPPORTED_MEDIA_TYPE, got %#v", env)
	}
}

func TestManagementJSONAcceptsJSONContentTypeWithCharset(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	payload := `{"name":"Charset Agent","workspaceId":"ws-1","channelType":"local","status":"active"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-Admin-Key", "test-admin")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected JSON content type with charset to be accepted, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestJSONResponsesSetNoSniffHeader(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	ok := request(t, router, http.MethodGet, "/healthz", nil, "")
	if got := ok.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("successful JSON responses should set nosniff, got %q", got)
	}

	unauthorized := request(t, router, http.MethodGet, "/api/v1/agents", nil, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized management read, got %d", unauthorized.Code)
	}
	if got := unauthorized.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("error JSON responses should set nosniff, got %q", got)
	}
}

func TestManagementMCPResponsesSetNoSniffHeader(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	resp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id":      "tools",
		"method":  "tools/list",
	}, "", "test-admin")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected management MCP tools/list to succeed, got %d body=%s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("management MCP JSON responses should set nosniff, got %q", got)
	}
}

func TestManagementMCPRequiresJSONContentType(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/management/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":"tools","method":"tools/list"}`))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Admin-Key", "test-admin")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("management MCP parse errors should keep JSON-RPC HTTP 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload mcpEnvelopeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode management MCP error: %v body=%s", err, rec.Body.String())
	}
	if payload.Error == nil || payload.Error.Code != -32700 || !strings.Contains(payload.Error.Message, "application/json") {
		t.Fatalf("expected JSON content-type parse error, got %#v", payload.Error)
	}
}

func TestManagementMCPRejectsBodiesLargerThanJSONLimit(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	oversized := `{"jsonrpc":"2.0","id":"tools","method":"tools/list","padding":"` + strings.Repeat("x", 1<<20) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/management/mcp", strings.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", "test-admin")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("management MCP parse errors should keep JSON-RPC HTTP 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload mcpEnvelopeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode management MCP error: %v body=%s", err, rec.Body.String())
	}
	if payload.Error == nil || payload.Error.Code != -32700 || !strings.Contains(payload.Error.Message, "1MiB") {
		t.Fatalf("expected payload size parse error, got %#v", payload.Error)
	}
}

func assertSensitiveNoCacheHeaders(t *testing.T, rec *httptest.ResponseRecorder, label string) {
	t.Helper()
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("%s should set Cache-Control no-store, got %q", label, got)
	}
	if got := rec.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("%s should set Pragma no-cache, got %q", label, got)
	}
	if got := rec.Header().Get("Expires"); got != "0" {
		t.Fatalf("%s should set Expires 0, got %q", label, got)
	}
}

func TestManagementResponsesSetNoStore(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	publicHealth := request(t, router, http.MethodGet, "/healthz", nil, "")
	if got := publicHealth.Header().Get("Cache-Control"); got == "no-store" {
		t.Fatalf("public health should not be forced no-store, got %q", got)
	}

	unauthorized := request(t, router, http.MethodGet, "/api/v1/agents", nil, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized management read, got %d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	assertSensitiveNoCacheHeaders(t, unauthorized, "unauthorized management responses")

	created := requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "No Store Agent",
		"workspaceId": "ws-1",
		"channelType": "local",
		"status":      "active",
	}, "", "test-admin")
	if created.Code != http.StatusCreated {
		t.Fatalf("expected management write to succeed, got %d body=%s", created.Code, created.Body.String())
	}
	assertSensitiveNoCacheHeaders(t, created, "management responses")
}

func TestDataPlaneResponsesSetNoStore(t *testing.T) {
	router := newRouter()

	mcp := request(t, router, http.MethodPost, "/api/v1/mcp/agents/target-1/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/list",
	}, "")
	if mcp.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized data-plane MCP call, got %d body=%s", mcp.Code, mcp.Body.String())
	}
	assertSensitiveNoCacheHeaders(t, mcp, "data-plane MCP responses")

	openapi := request(t, router, http.MethodPost, "/api/v1/openapi/agents/target-1/operations/listCustomers", map[string]any{}, "")
	if openapi.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized data-plane OpenAPI call, got %d body=%s", openapi.Code, openapi.Body.String())
	}
	assertSensitiveNoCacheHeaders(t, openapi, "data-plane OpenAPI responses")
}

func TestConsoleAuthSessionProtectsManagementEndpoints(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	missing := request(t, router, http.MethodGet, "/api/v1/auth/session", nil, "")
	if missing.Code != http.StatusOK {
		t.Fatalf("session status should be public, got %d body=%s", missing.Code, missing.Body.String())
	}
	assertSensitiveNoCacheHeaders(t, missing, "session status")
	missingSession := decodeData[map[string]any](t, missing)
	if missingSession["authenticated"] != false || missingSession["requiresLogin"] != true {
		t.Fatalf("expected unauthenticated production session, got %#v", missingSession)
	}

	wrongLogin := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "")
	if wrongLogin.Code != http.StatusUnauthorized || len(wrongLogin.Result().Cookies()) != 0 {
		t.Fatalf("wrong login should be unauthorized without setting cookies, status=%d body=%s cookies=%#v", wrongLogin.Code, wrongLogin.Body.String(), wrongLogin.Result().Cookies())
	}
	assertSensitiveNoCacheHeaders(t, wrongLogin, "failed login response")

	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "test-admin"}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login should succeed, got %d body=%s", login.Code, login.Body.String())
	}
	assertSensitiveNoCacheHeaders(t, login, "login response")
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
	assertSensitiveNoCacheHeaders(t, logout, "logout response")
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

func TestConsoleAuthSessionReportsDevelopmentMode(t *testing.T) {
	router := newRouter()

	session := decodeData[map[string]any](t, request(t, router, http.MethodGet, "/api/v1/auth/session", nil, ""))
	if session["authenticated"] != true || session["actor"] != "local-dev" || session["requiresLogin"] != false {
		t.Fatalf("dev unauthenticated mode should not require browser login, got %#v", session)
	}
}
