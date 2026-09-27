package httpapi_test

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type targetProbeResult struct {
	TargetID   string    `json:"targetId"`
	Endpoint   string    `json:"endpoint"`
	Status     string    `json:"status"`
	ErrorCode  string    `json:"errorCode"`
	Message    string    `json:"message"`
	HTTPStatus int       `json:"httpStatus"`
	ToolCount  int       `json:"toolCount"`
	DurationMs int64     `json:"durationMs"`
	CheckedAt  time.Time `json:"checkedAt"`
}

func TestTargetProbeSendsRefreshRequestWithoutWritingState(t *testing.T) {
	repo := store.NewMemory()
	checkedAt := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	router := httpapi.New(repo, httpapi.WithUnauthenticatedAdminAllowed(true), httpapi.WithClock(func() time.Time { return checkedAt })).Router()
	var received struct {
		method      string
		contentType string
		accept      string
		configured  string
		credential  string
		body        map[string]any
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.method = r.Method
		received.contentType = r.Header.Get("Content-Type")
		received.accept = r.Header.Get("Accept")
		received.configured = r.Header.Get("X-Probe-Tenant")
		received.credential = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&received.body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"target-probe","result":{"tools":[{"name":"search_customer","inputSchema":{"type":"object"}},{"name":"export_contracts","inputSchema":{"type":"object"}}]}}`))
	}))
	defer upstream.Close()
	target := createDirectAgentWithCredentials(t, repo, "Probe MCP", "tenant-probe", "ws-probe", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint":          upstream.URL + "/mcp",
		"headers":           map[string]any{"X-Probe-Tenant": "tenant-probe"},
		"credentialHeaders": map[string]any{"Authorization": "apiToken"},
	}, map[string]string{"apiToken": "Bearer probe-secret"})
	auditBefore, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{})
	if err != nil {
		t.Fatalf("list audit events before probe: %v", err)
	}

	resp := request(t, router, http.MethodPost, "/api/v1/targets/"+target.ID+":probe", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("probe should succeed, got %d body=%s", resp.Code, resp.Body.String())
	}
	result := decodeData[targetProbeResult](t, resp)
	if result.Status != "ok" || result.ErrorCode != "" || result.Message != "" || result.HTTPStatus != http.StatusOK || result.ToolCount != 2 {
		t.Fatalf("unexpected probe result: %#v", result)
	}
	if result.TargetID != target.ID || result.Endpoint != upstream.URL+"/mcp" || !result.CheckedAt.Equal(checkedAt) || result.DurationMs < 0 {
		t.Fatalf("probe result should identify the target and timing, got %#v", result)
	}
	if received.method != http.MethodPost || received.contentType != "application/json" || received.accept != "application/json, text/event-stream" {
		t.Fatalf("probe should send the refresh tools/list request, got %#v", received)
	}
	if received.configured != "tenant-probe" || received.credential != "Bearer probe-secret" {
		t.Fatalf("probe should send configured and credential headers like a refresh, got %#v", received)
	}
	if received.body["method"] != "tools/list" || received.body["id"] != "target-probe" {
		t.Fatalf("probe should call tools/list, got %#v", received.body)
	}
	if strings.Contains(resp.Body.String(), "probe-secret") {
		t.Fatalf("probe result leaked a credential: %s", resp.Body.String())
	}

	capabilities, err := repo.ListCapabilities(t.Context(), store.CapabilityFilter{TargetID: target.ID})
	if err != nil {
		t.Fatalf("list capabilities after probe: %v", err)
	}
	if len(capabilities) != 0 {
		t.Fatalf("probe must not write capabilities, got %#v", capabilities)
	}
	auditAfter, err := repo.ListAuditEvents(t.Context(), store.AuditEventFilter{})
	if err != nil {
		t.Fatalf("list audit events after probe: %v", err)
	}
	if len(auditAfter) != len(auditBefore) {
		t.Fatalf("probe must not write audit events, before=%d after=%#v", len(auditBefore), auditAfter)
	}
}

func TestTargetProbeReportsClassifiedFailuresAsResults(t *testing.T) {
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedEndpoint := closed.URL + "/mcp"
	closed.Close()
	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unavailable.Close()
	notMCP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>login</html>"))
	}))
	defer notMCP.Close()
	stalled := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// Draining the body lets the server notice the client hanging up,
		// so Close does not wait out the fallback timer.
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer stalled.Close()

	for _, tc := range []struct {
		name       string
		config     map[string]any
		errorCode  string
		message    string
		httpStatus int
	}{
		{name: "connection refused", config: map[string]any{"endpoint": closedEndpoint}, errorCode: "UPSTREAM_CONNECT_ERROR", message: "connection refused"},
		{name: "non-2xx", config: map[string]any{"endpoint": unavailable.URL}, errorCode: "UPSTREAM_ERROR", message: "non-2xx", httpStatus: http.StatusServiceUnavailable},
		{name: "not tools/list JSON", config: map[string]any{"endpoint": notMCP.URL}, errorCode: "UPSTREAM_ERROR", message: "valid MCP tools/list JSON", httpStatus: http.StatusOK},
		{name: "timeout", config: map[string]any{"endpoint": stalled.URL, "timeoutMs": 50}, errorCode: "UPSTREAM_TIMEOUT", message: "timed out"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := store.NewMemory()
			router := newRouterWithRepo(repo)
			target := createDirectAgent(t, repo, "Failing Probe MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, tc.config)
			resp := request(t, router, http.MethodPost, "/api/v1/targets/"+target.ID+":probe", nil, "")
			if resp.Code != http.StatusOK {
				t.Fatalf("probe failures should be reported as results, got %d body=%s", resp.Code, resp.Body.String())
			}
			result := decodeData[targetProbeResult](t, resp)
			if result.Status != "error" || result.ErrorCode != tc.errorCode || result.HTTPStatus != tc.httpStatus || result.ToolCount != 0 {
				t.Fatalf("unexpected probe result: %#v", result)
			}
			if !strings.Contains(result.Message, tc.message) {
				t.Fatalf("probe message should carry the cause %q, got %q", tc.message, result.Message)
			}
			if result.CheckedAt.IsZero() {
				t.Fatalf("failed probe should still report checkedAt: %#v", result)
			}
		})
	}
}

func TestTargetProbeKeepsEndpointUserinfoOutOfResults(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	originalTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, &net.DNSError{Err: "no such host", Name: "nonexistent.invalid", IsNotFound: true}
	})
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})
	target := createDirectAgent(t, repo, "Userinfo Probe MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": "http://probe-user:probe-pass@nonexistent.invalid/mcp",
	})

	resp := request(t, router, http.MethodPost, "/api/v1/targets/"+target.ID+":probe", nil, "")
	result := decodeData[targetProbeResult](t, resp)
	if result.Status != "error" || result.ErrorCode != "UPSTREAM_DNS_ERROR" || result.Endpoint != "http://nonexistent.invalid/mcp" {
		t.Fatalf("unexpected DNS probe result: %#v", result)
	}
	body := resp.Body.String()
	if strings.Contains(body, "probe-user") || strings.Contains(body, "probe-pass") {
		t.Fatalf("probe result leaked endpoint userinfo: %s", body)
	}
	if !strings.Contains(body, `"httpStatus":0`) {
		t.Fatalf("probe result should report httpStatus 0 when no response arrived: %s", body)
	}
}

func TestTargetProbeRejectsUnprobeableTargets(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"result":{"tools":[]}}`))
	}))
	defer upstream.Close()
	openapi := createDirectAgent(t, repo, "OpenAPI Target", "default", "ws-1", "openapi", domain.AgentStatusActive, map[string]any{"endpoint": upstream.URL})
	noEndpoint := createDirectAgent(t, repo, "Endpointless MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)
	badHeaders := createDirectAgent(t, repo, "Bad Header MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint": upstream.URL,
		"headers":  "X-Tenant: default",
	})
	missingCredential := createDirectAgent(t, repo, "Missing Credential MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint":          upstream.URL,
		"credentialHeaders": map[string]any{"Authorization": "apiToken"},
	})
	badTimeout := createDirectAgent(t, repo, "Bad Timeout MCP", "default", "ws-1", "mcp", domain.AgentStatusActive, map[string]any{
		"endpoint":  upstream.URL,
		"timeoutMs": 0,
	})

	for _, tc := range []struct {
		name     string
		targetID string
		status   int
		code     string
	}{
		{name: "unknown target", targetID: "agt_missing", status: http.StatusNotFound, code: "NOT_FOUND"},
		{name: "non-mcp target", targetID: openapi.ID, status: http.StatusBadRequest, code: "VALIDATION_FAILED"},
		{name: "missing endpoint", targetID: noEndpoint.ID, status: http.StatusBadRequest, code: "VALIDATION_FAILED"},
		// Configuration errors are for the agent record, not probe results.
		{name: "invalid headers", targetID: badHeaders.ID, status: http.StatusBadRequest, code: "VALIDATION_FAILED"},
		{name: "missing credential", targetID: missingCredential.ID, status: http.StatusBadRequest, code: "VALIDATION_FAILED"},
		{name: "invalid timeout", targetID: badTimeout.ID, status: http.StatusBadRequest, code: "VALIDATION_FAILED"},
	} {
		resp := request(t, router, http.MethodPost, "/api/v1/targets/"+tc.targetID+":probe", nil, "")
		if resp.Code != tc.status {
			t.Fatalf("%s: expected %d, got %d body=%s", tc.name, tc.status, resp.Code, resp.Body.String())
		}
		var env apiEnvelope
		if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
			t.Fatalf("%s: decode error envelope: %v", tc.name, err)
		}
		if env.Error != tc.code {
			t.Fatalf("%s: expected %s, got %#v", tc.name, tc.code, env)
		}
	}
	if upstreamCalls != 0 {
		t.Fatalf("rejected probes must not reach the upstream, calls=%d", upstreamCalls)
	}
}

func TestTargetProbeRequiresTargetManagementScope(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"result":{"tools":[]}}`))
	}))
	defer upstream.Close()
	westTarget := createDirectAgent(t, repo, "West MCP", "tenant-west", "ws-finance", "mcp", domain.AgentStatusActive, map[string]any{"endpoint": upstream.URL})

	resp := requestWithAdmin(t, router, http.MethodPost, "/api/v1/targets/"+westTarget.ID+":probe", nil, "", "support-key")
	if resp.Code != http.StatusForbidden {
		t.Fatalf("cross-scope probe should be denied, got %d body=%s", resp.Code, resp.Body.String())
	}
	if upstreamCalls != 0 {
		t.Fatalf("cross-scope probe must not reach the upstream, calls=%d", upstreamCalls)
	}
}
