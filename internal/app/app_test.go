package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const strongProductionAdminKey = "test-production-admin-key-32"
const strongProductionSessionSecret = "test-production-session-secret-32-bytes"

func TestPostgresCredentialKeyFromEnvRequiresKey(t *testing.T) {
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", "")

	_, err := postgresCredentialKeyFromEnv()
	if err == nil {
		t.Fatalf("expected PostgreSQL credential key config error")
	}
}

func TestPostgresCredentialKeyFromEnvParsesRawKey(t *testing.T) {
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", "AgentHarborCredentialKey-2026!!!")

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

func TestNewProductionModeRequiresExplicitCORSOrigins(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", strongProductionAdminKey)
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/mcp/agents/browser-gate/rpc", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5174")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-AgentHarbor-Subject-Id")
	rec := httptest.NewRecorder()

	app.Router().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("production mode should not allow default local dev CORS origin, got %q", got)
	}
}

func TestNewProductionModeAllowsExplicitCORSOrigins(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", strongProductionAdminKey)
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_CORS_ORIGINS", "https://console.example.com")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/mcp/agents/browser-gate/rpc", nil)
	req.Header.Set("Origin", "https://console.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-AgentHarbor-Subject-Id")
	rec := httptest.NewRecorder()

	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("configured production CORS origin should allow preflight, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("unexpected allowed production origin %q", got)
	}
}

func TestNewRejectsInvalidConfiguredCORSOrigin(t *testing.T) {
	for _, raw := range []string{
		"*",
		"console.example.com",
		"https://console.example.com:bad",
		"https://console.example.com/path",
		"ftp://console.example.com",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("AGENT_HARBOR_CORS_ORIGINS", raw)

			_, err := New(context.Background())
			if err == nil {
				t.Fatalf("expected invalid configured CORS origin %q to fail", raw)
			}
			if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_CORS_ORIGINS") {
				t.Fatalf("expected CORS origin error to name env var, got %q", got)
			}
		})
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

func TestAdminIdentitiesFromEnvParsesScopedRules(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", "platform=platform-key|role=platform_admin,tenant-east=east-key|role=tenant_admin|tenant=tenant-east|workspace=ws-support,legacy=legacy-key")

	identities, err := adminIdentitiesFromEnv()
	if err != nil {
		t.Fatalf("parse admin identities: %v", err)
	}
	if len(identities) != 3 {
		t.Fatalf("expected three identities, got %#v", identities)
	}
	if identities[0].Actor != "platform" || identities[0].Key != "platform-key" || identities[0].Role != "platform_admin" || identities[0].TenantID != "" || identities[0].WorkspaceID != "" {
		t.Fatalf("unexpected platform identity: %#v", identities[0])
	}
	if identities[1].Actor != "tenant-east" || identities[1].Key != "east-key" || identities[1].Role != "tenant_admin" || identities[1].TenantID != "tenant-east" || identities[1].WorkspaceID != "ws-support" {
		t.Fatalf("unexpected scoped identity: %#v", identities[1])
	}
	if identities[2].Actor != "legacy" || identities[2].Key != "legacy-key" || identities[2].Role != "platform_admin" || identities[2].TenantID != "" || identities[2].WorkspaceID != "" {
		t.Fatalf("legacy identity should default to platform admin: %#v", identities[2])
	}
}

func TestAdminIdentitiesFromEnvRejectsDuplicateActors(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", "platform=platform-key,platform=another-key|role=tenant_admin|tenant=tenant-east")

	_, err := adminIdentitiesFromEnv()
	if err == nil {
		t.Fatalf("expected duplicate admin identity actor to fail")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate actor") {
		t.Fatalf("expected duplicate actor error, got %q", got)
	}
}

func TestAdminIdentitiesFromEnvRejectsDuplicateKeys(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", "platform=shared-key,tenant-east=shared-key|role=tenant_admin|tenant=tenant-east")

	_, err := adminIdentitiesFromEnv()
	if err == nil {
		t.Fatalf("expected duplicate admin identity key to fail")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate key") {
		t.Fatalf("expected duplicate key error, got %q", got)
	}
}

func TestApprovalReviewersFromEnvRejectsMalformedRules(t *testing.T) {
	t.Setenv("AGENT_HARBOR_APPROVAL_REVIEWERS", "security-root=tenant-root")

	_, err := approvalReviewersFromEnv()
	if err == nil {
		t.Fatalf("expected malformed approval reviewer env value to fail")
	}
}

func TestDeploymentConfigPreflightBlocksUnsafeProductionFlags(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", strongProductionAdminKey)
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN", "true")
	t.Setenv("AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS", "true")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected unsafe production config to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN") || !strings.Contains(got, "AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS") {
		t.Fatalf("expected production config error to name unsafe flags, got %q", got)
	}
}

func TestDeploymentConfigPreflightBlocksTruthyProductionFlags(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", strongProductionAdminKey)
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN", "1")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected truthy unsafe production config to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN") {
		t.Fatalf("expected production config error to name truthy unsafe flag, got %q", got)
	}
}

func TestDeploymentConfigPreflightRequiresProductionAdminAuthentication(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected missing production admin authentication to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_KEY") || !strings.Contains(got, "AGENT_HARBOR_ADMIN_IDENTITIES") {
		t.Fatalf("expected production config error to name admin authentication, got %q", got)
	}
}

func TestDeploymentConfigPreflightBlocksShortProductionAdminKey(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       "short-key",
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
	})
	if err == nil {
		t.Fatalf("expected short production admin key to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_KEY") {
		t.Fatalf("expected production config error to name short admin key, got %q", got)
	}
	if !hasDeploymentCheck(checks, "admin_key_strength", "blocking", "failed") {
		t.Fatalf("expected admin key strength failure, got %#v", checks)
	}
}

func TestNewRunsProductionPreflightBeforeStorageInitialization(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", "short-key")
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_DATABASE_URL", "://invalid-postgres-url")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected production preflight to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_KEY") {
		t.Fatalf("expected production preflight error before storage initialization, got %q", got)
	}
}

func TestDeploymentConfigPreflightBlocksCommonProductionAdminKey(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       "changeme",
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
	})
	if err == nil {
		t.Fatalf("expected common production admin key to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_KEY") {
		t.Fatalf("expected production config error to name common admin key, got %q", got)
	}
	if !hasDeploymentCheck(checks, "admin_key_strength", "blocking", "failed") {
		t.Fatalf("expected admin key strength failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksShortProductionAdminIdentityKey(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":  "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES": "platform=short-key|role=platform_admin",
		"AGENT_HARBOR_SESSION_SECRET":   strongProductionSessionSecret,
	})
	if err == nil {
		t.Fatalf("expected short production admin identity key to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_IDENTITIES") {
		t.Fatalf("expected production config error to name short admin identity key, got %q", got)
	}
	if !hasDeploymentCheck(checks, "admin_key_strength", "blocking", "failed") {
		t.Fatalf("expected admin key strength failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksSharedAdminKeyMatchingNamedIdentity(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":  "production",
		"AGENT_HARBOR_ADMIN_KEY":        strongProductionAdminKey,
		"AGENT_HARBOR_ADMIN_IDENTITIES": "platform=" + strongProductionAdminKey + "|role=platform_admin",
		"AGENT_HARBOR_SESSION_SECRET":   strongProductionSessionSecret,
	})
	if err == nil {
		t.Fatalf("expected shared admin key matching a named identity key to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_KEY") || !strings.Contains(got, "AGENT_HARBOR_ADMIN_IDENTITIES") {
		t.Fatalf("expected production config error to name conflicting admin key settings, got %q", got)
	}
	if !hasDeploymentCheck(checks, "admin_key_uniqueness", "blocking", "failed") {
		t.Fatalf("expected admin key uniqueness failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightAllowsSafeProductionConfig(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", strongProductionAdminKey)
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("safe production config should start: %v", err)
	}
	defer app.Close()
}

func TestDeploymentConfigPreflightBlocksInvalidProductionCORSOrigins(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
		"AGENT_HARBOR_CORS_ORIGINS":    "*",
	})
	if err == nil {
		t.Fatalf("expected invalid production CORS origins to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_CORS_ORIGINS") {
		t.Fatalf("expected production config error to name CORS origins, got %q", got)
	}
	if !hasDeploymentCheck(checks, "cors_origins_valid", "blocking", "failed") {
		t.Fatalf("expected CORS origin validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightRequiresProductionSessionSecret(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
	})
	if err == nil {
		t.Fatalf("expected missing production session secret to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_SESSION_SECRET") {
		t.Fatalf("expected production config error to name missing session secret, got %q", got)
	}
	if !hasDeploymentCheck(checks, "session_secret_explicit", "blocking", "failed") {
		t.Fatalf("expected session secret explicit failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksShortProductionSessionSecret(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  "short-secret",
	})
	if err == nil {
		t.Fatalf("expected short production session secret to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_SESSION_SECRET") {
		t.Fatalf("expected production config error to name short session secret, got %q", got)
	}
	if !hasDeploymentCheck(checks, "session_secret_strength", "blocking", "failed") {
		t.Fatalf("expected session secret strength failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksCommonProductionSessionSecret(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  "secret",
	})
	if err == nil {
		t.Fatalf("expected common production session secret to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_SESSION_SECRET") {
		t.Fatalf("expected production config error to name common session secret, got %q", got)
	}
	if !hasDeploymentCheck(checks, "session_secret_strength", "blocking", "failed") {
		t.Fatalf("expected session secret strength failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightWarnsForInMemoryProductionStorage(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
	})
	if err != nil {
		t.Fatalf("preflight should warn, not fail: %v", err)
	}
	if !hasDeploymentCheck(checks, "persistent_storage_configured", "warning", "warning") {
		t.Fatalf("expected persistent storage warning, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightAcceptsProductionDatabaseStorage(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":    "postgres://agent-harbor.example.invalid/agent_harbor",
	})
	if err != nil {
		t.Fatalf("preflight should accept database storage: %v", err)
	}
	if !hasDeploymentCheck(checks, "persistent_storage_configured", "warning", "passed") {
		t.Fatalf("expected persistent storage pass, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightKeepsDevelopmentModeCompatible(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN", "true")

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("development mode should preserve explicit unauthenticated admin: %v", err)
	}
	defer app.Close()
}

func TestDeploymentModeRejectsInvalidValue(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "prod")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected invalid deployment mode to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_DEPLOYMENT_MODE") {
		t.Fatalf("expected deployment mode error, got %q", got)
	}
}

func hasDeploymentCheck(checks []deploymentConfigCheck, code, severity, status string) bool {
	for _, check := range checks {
		if check.Code == code && check.Severity == severity && check.Status == status {
			return true
		}
	}
	return false
}
