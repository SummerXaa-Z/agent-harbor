package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostgresCredentialKeyFromEnvRequiresKey(t *testing.T) {
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", "")

	_, err := postgresCredentialKeyFromEnv()
	if err == nil {
		t.Fatalf("expected PostgreSQL credential key config error")
	}
}

func TestPostgresCredentialKeyFromEnvParsesRawKey(t *testing.T) {
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", "0123456789abcdef0123456789abcdef")

	key, err := postgresCredentialKeyFromEnv()
	if err != nil {
		t.Fatalf("parse credential key: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestNewAllowsPrivateUpstreamsFromEnv(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS", "true")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", "test-admin")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	body := map[string]any{
		"name":        "Local MCP Target",
		"workspaceId": "ws-local",
		"channelType": "mcp",
		"status":      "active",
		"channelConfig": map[string]any{
			"endpoint": "http://127.0.0.1:8099/mcp",
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", "test-admin")
	rec := httptest.NewRecorder()

	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("private upstream env flag should allow local endpoint, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNewRejectsAdminEndpointsWithoutConfiguredAuthentication(t *testing.T) {
	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()

	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin endpoint without configured auth should be unauthorized, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNewAllowsExplicitUnauthenticatedAdminForDevelopment(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN", "true")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()

	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("explicit unauthenticated admin dev flag should allow admin route, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNewExposesConsoleLoginWithConfiguredAdminKey(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", "test-admin")
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", "session-secret")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	payload, err := json.Marshal(map[string]string{"adminKey": "test-admin"})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("configured admin key should login, got %d body=%s", rec.Code, rec.Body.String())
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != "agent_harbor_session" || !cookies[0].HttpOnly {
		t.Fatalf("login should set HttpOnly console session cookie, got %#v", cookies)
	}
}

func TestNewAllowsConfiguredCORSOriginsFromEnv(t *testing.T) {
	t.Setenv("AGENT_HARBOR_CORS_ORIGINS", "http://127.0.0.1:15174")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/mcp/agents/browser-gate/rpc", nil)
	req.Header.Set("Origin", "http://127.0.0.1:15174")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-AgentHarbor-Subject-Id")
	rec := httptest.NewRecorder()

	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("configured CORS origin should allow preflight, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:15174" {
		t.Fatalf("unexpected allowed origin %q", got)
	}
}

func TestPrivateUpstreamsEnvRejectsInvalidBoolean(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS", "sometimes")

	_, err := privateUpstreamsAllowedFromEnv()
	if err == nil {
		t.Fatalf("expected invalid private upstream env value to fail")
	}
}

func TestUnauthenticatedAdminEnvRejectsInvalidBoolean(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN", "sometimes")

	_, err := unauthenticatedAdminAllowedFromEnv()
	if err == nil {
		t.Fatalf("expected invalid unauthenticated admin env value to fail")
	}
}

func TestApprovalReviewersFromEnvParsesScopedRules(t *testing.T) {
	t.Setenv("AGENT_HARBOR_APPROVAL_REVIEWERS", "security-root=tenant-root/*;security-east=tenant-east/ws-support")

	reviewers, err := approvalReviewersFromEnv()
	if err != nil {
		t.Fatalf("parse approval reviewers: %v", err)
	}
	if len(reviewers) != 2 {
		t.Fatalf("expected two approval reviewer rules, got %#v", reviewers)
	}
	if reviewers[0].Reviewer != "security-root" || reviewers[0].TenantID != "tenant-root" || reviewers[0].WorkspaceID != "*" {
		t.Fatalf("unexpected first reviewer rule: %#v", reviewers[0])
	}
	if reviewers[1].Reviewer != "security-east" || reviewers[1].TenantID != "tenant-east" || reviewers[1].WorkspaceID != "ws-support" {
		t.Fatalf("unexpected second reviewer rule: %#v", reviewers[1])
	}
}

func TestApprovalReviewersFromEnvRejectsMalformedRules(t *testing.T) {
	t.Setenv("AGENT_HARBOR_APPROVAL_REVIEWERS", "security-root=tenant-root")

	_, err := approvalReviewersFromEnv()
	if err == nil {
		t.Fatalf("expected malformed approval reviewer env value to fail")
	}
}
