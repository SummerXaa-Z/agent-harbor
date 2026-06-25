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
const strongProductionCredentialKey = "AgentHarborCredentialKey-2026!!!"

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

func TestProductionModeDisablesDefaultLocalCORSOrigins(t *testing.T) {
	if defaultLocalCORSOriginsAllowed("production") {
		t.Fatalf("production mode should disable default local CORS origins")
	}
	if !defaultLocalCORSOriginsAllowed("development") {
		t.Fatalf("development mode should keep default local CORS origins")
	}
}

func TestDeploymentConfigPreflightAllowsExplicitProductionCORSOrigins(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":    "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":  strongProductionCredentialKey,
		"AGENT_HARBOR_CORS_ORIGINS":    "https://console.example.com",
	})
	if err != nil {
		t.Fatalf("configured production CORS origin should pass preflight: %v", err)
	}
	if !hasDeploymentCheck(checks, "cors_origins_valid", "blocking", "passed") {
		t.Fatalf("expected production CORS origin validation pass, got %#v", checks)
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

func TestAdminIdentitiesFromEnvRejectsReservedActors(t *testing.T) {
	for _, actor := range []string{"admin-key", "local-dev"} {
		t.Run(actor, func(t *testing.T) {
			t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", actor+"=reserved-actor-key|role=platform_admin")

			_, err := adminIdentitiesFromEnv()
			if err == nil {
				t.Fatalf("expected reserved admin identity actor %q to fail", actor)
			}
			if got := err.Error(); !strings.Contains(got, "reserved actor") || !strings.Contains(got, actor) {
				t.Fatalf("expected reserved actor error, got %q", got)
			}
		})
	}
}

func TestAdminIdentitiesFromEnvRejectsUnknownRole(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", "platform=platform-key|role=owner|tenant=tenant-east")

	_, err := adminIdentitiesFromEnv()
	if err == nil {
		t.Fatalf("expected unknown admin identity role to fail")
	}
	if got := err.Error(); !strings.Contains(got, "role must be platform_admin, tenant_admin, or security_reviewer") {
		t.Fatalf("expected role validation error, got %q", got)
	}
}

func TestAdminIdentitiesFromEnvRejectsScopedPlatformAdmin(t *testing.T) {
	t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", "platform=platform-key|role=platform_admin|tenant=tenant-east|workspace=ws-support")

	_, err := adminIdentitiesFromEnv()
	if err == nil {
		t.Fatalf("expected scoped platform admin identity to fail")
	}
	if got := err.Error(); !strings.Contains(got, "platform_admin") || !strings.Contains(got, "must not include tenant or workspace") {
		t.Fatalf("expected platform admin scope validation error, got %q", got)
	}
}

func TestApprovalReviewersFromEnvRejectsMalformedRules(t *testing.T) {
	t.Setenv("AGENT_HARBOR_APPROVAL_REVIEWERS", "security-root=tenant-root")

	_, err := approvalReviewersFromEnv()
	if err == nil {
		t.Fatalf("expected malformed approval reviewer env value to fail")
	}
}

func TestDeploymentConfigPreflightBlocksMalformedProductionApprovalReviewers(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":    "production",
		"AGENT_HARBOR_ADMIN_KEY":          strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":     strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":       "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":     strongProductionCredentialKey,
		"AGENT_HARBOR_APPROVAL_REVIEWERS": "security-root=tenant-root",
	})
	if err == nil {
		t.Fatalf("expected malformed production approval reviewers to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_APPROVAL_REVIEWERS") {
		t.Fatalf("expected production config error to name approval reviewers, got %q", got)
	}
	if !hasDeploymentCheck(checks, "approval_reviewers_valid", "blocking", "failed") {
		t.Fatalf("expected approval reviewer validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightAcceptsProductionApprovalReviewers(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":    "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES":   "platform=" + strongProductionAdminKey + "|role=platform_admin;security-root=security-root-production-key-32|role=security_reviewer|tenant=tenant-root;security-east=security-east-production-key-32|role=security_reviewer|tenant=tenant-east|workspace=ws-support",
		"AGENT_HARBOR_SESSION_SECRET":     strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":       "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":     strongProductionCredentialKey,
		"AGENT_HARBOR_APPROVAL_REVIEWERS": "security-root=tenant-root/*;security-east=tenant-east/ws-support",
	})
	if err != nil {
		t.Fatalf("valid production approval reviewers should pass preflight: %v", err)
	}
	if !hasDeploymentCheck(checks, "approval_reviewers_valid", "blocking", "passed") {
		t.Fatalf("expected approval reviewer validation pass, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksApprovalReviewerWithoutAdminIdentity(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":    "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES":   "platform=" + strongProductionAdminKey + "|role=platform_admin",
		"AGENT_HARBOR_SESSION_SECRET":     strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":       "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":     strongProductionCredentialKey,
		"AGENT_HARBOR_APPROVAL_REVIEWERS": "security-east=tenant-east/ws-support",
	})
	if err == nil {
		t.Fatalf("expected approval reviewer without admin identity to fail")
	}
	if got := err.Error(); !strings.Contains(got, "security-east") || !strings.Contains(got, "AGENT_HARBOR_ADMIN_IDENTITIES") {
		t.Fatalf("expected production config error to explain missing reviewer identity, got %q", got)
	}
	if !hasDeploymentCheck(checks, "approval_reviewers_valid", "blocking", "failed") {
		t.Fatalf("expected approval reviewer validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksApprovalReviewerWithoutReviewerRole(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":    "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES":   "platform=" + strongProductionAdminKey + "|role=platform_admin;east=tenant-admin-production-key-32|role=tenant_admin|tenant=tenant-east|workspace=ws-support",
		"AGENT_HARBOR_SESSION_SECRET":     strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":       "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":     strongProductionCredentialKey,
		"AGENT_HARBOR_APPROVAL_REVIEWERS": "east=tenant-east/ws-support",
	})
	if err == nil {
		t.Fatalf("expected approval reviewer with tenant_admin role to fail")
	}
	if got := err.Error(); !strings.Contains(got, "east") || !strings.Contains(got, "security_reviewer") {
		t.Fatalf("expected production config error to explain reviewer role requirement, got %q", got)
	}
	if !hasDeploymentCheck(checks, "approval_reviewers_valid", "blocking", "failed") {
		t.Fatalf("expected approval reviewer validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksApprovalReviewerOutsideAdminTenantScope(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":    "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES":   "platform=" + strongProductionAdminKey + "|role=platform_admin;security-east=security-east-production-key-32|role=security_reviewer|tenant=tenant-west|workspace=ws-support",
		"AGENT_HARBOR_SESSION_SECRET":     strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":       "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":     strongProductionCredentialKey,
		"AGENT_HARBOR_APPROVAL_REVIEWERS": "security-east=tenant-east/ws-support",
	})
	if err == nil {
		t.Fatalf("expected approval reviewer route outside admin tenant scope to fail")
	}
	if got := err.Error(); !strings.Contains(got, "security-east") || !strings.Contains(got, "tenant/workspace scope") {
		t.Fatalf("expected production config error to explain reviewer tenant scope mismatch, got %q", got)
	}
	if !hasDeploymentCheck(checks, "approval_reviewers_valid", "blocking", "failed") {
		t.Fatalf("expected approval reviewer validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksApprovalReviewerWiderThanAdminWorkspaceScope(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":    "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES":   "platform=" + strongProductionAdminKey + "|role=platform_admin;security-east=security-east-production-key-32|role=security_reviewer|tenant=tenant-east|workspace=ws-support",
		"AGENT_HARBOR_SESSION_SECRET":     strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":       "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":     strongProductionCredentialKey,
		"AGENT_HARBOR_APPROVAL_REVIEWERS": "security-east=tenant-east/*",
	})
	if err == nil {
		t.Fatalf("expected approval reviewer route wider than admin workspace scope to fail")
	}
	if got := err.Error(); !strings.Contains(got, "security-east") || !strings.Contains(got, "tenant/workspace scope") {
		t.Fatalf("expected production config error to explain reviewer workspace scope mismatch, got %q", got)
	}
	if !hasDeploymentCheck(checks, "approval_reviewers_valid", "blocking", "failed") {
		t.Fatalf("expected approval reviewer validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksMalformedProductionAdminIdentities(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":  "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES": "platform=" + strongProductionAdminKey + "|role=owner|tenant=tenant-east",
		"AGENT_HARBOR_SESSION_SECRET":   strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":     "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":   strongProductionCredentialKey,
	})
	if err == nil {
		t.Fatalf("expected malformed production admin identities to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_IDENTITIES") {
		t.Fatalf("expected production config error to name admin identities, got %q", got)
	}
	if !hasDeploymentCheck(checks, "admin_identities_valid", "blocking", "failed") {
		t.Fatalf("expected admin identity validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksReservedProductionAdminIdentityActor(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":  "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES": "admin-key=" + strongProductionAdminKey + "|role=platform_admin",
		"AGENT_HARBOR_SESSION_SECRET":   strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":     "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":   strongProductionCredentialKey,
	})
	if err == nil {
		t.Fatalf("expected reserved production admin identity actor to fail")
	}
	if got := err.Error(); !strings.Contains(got, "reserved actor") || !strings.Contains(got, "admin-key") {
		t.Fatalf("expected production config error to explain reserved actor, got %q", got)
	}
	if !hasDeploymentCheck(checks, "admin_identities_valid", "blocking", "failed") {
		t.Fatalf("expected admin identity validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksScopedProductionPlatformAdminIdentity(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":  "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES": "platform=" + strongProductionAdminKey + "|role=platform_admin|tenant=tenant-east|workspace=ws-support",
		"AGENT_HARBOR_SESSION_SECRET":   strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":     "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":   strongProductionCredentialKey,
	})
	if err == nil {
		t.Fatalf("expected scoped production platform admin identity to fail")
	}
	if got := err.Error(); !strings.Contains(got, "platform_admin") || !strings.Contains(got, "must not include tenant or workspace") {
		t.Fatalf("expected production config error to explain platform admin scope mismatch, got %q", got)
	}
	if !hasDeploymentCheck(checks, "admin_identities_valid", "blocking", "failed") {
		t.Fatalf("expected admin identity validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightAcceptsProductionAdminIdentities(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":  "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES": "platform=" + strongProductionAdminKey + "|role=platform_admin;east=tenant-admin-production-key-32|role=tenant_admin|tenant=tenant-east|workspace=ws-support",
		"AGENT_HARBOR_SESSION_SECRET":   strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":     "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":   strongProductionCredentialKey,
	})
	if err != nil {
		t.Fatalf("valid production admin identities should pass preflight: %v", err)
	}
	if !hasDeploymentCheck(checks, "admin_identities_valid", "blocking", "passed") {
		t.Fatalf("expected admin identity validation pass, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightRequiresProductionBootstrapPlatformAdmin(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":  "production",
		"AGENT_HARBOR_ADMIN_IDENTITIES": "east=tenant-admin-production-key-32|role=tenant_admin|tenant=tenant-east|workspace=ws-support",
		"AGENT_HARBOR_SESSION_SECRET":   strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":     "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":   strongProductionCredentialKey,
	})
	if err == nil {
		t.Fatalf("expected production config without bootstrap platform admin to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_KEY") || !strings.Contains(got, "platform_admin") {
		t.Fatalf("expected production config error to name platform admin recovery config, got %q", got)
	}
	if !hasDeploymentCheck(checks, "platform_admin_configured", "blocking", "failed") {
		t.Fatalf("expected platform admin validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightAcceptsProductionScopedAdminsWithSharedAdminKey(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE":  "production",
		"AGENT_HARBOR_ADMIN_KEY":        strongProductionAdminKey,
		"AGENT_HARBOR_ADMIN_IDENTITIES": "east=tenant-admin-production-key-32|role=tenant_admin|tenant=tenant-east|workspace=ws-support",
		"AGENT_HARBOR_SESSION_SECRET":   strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":     "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":   strongProductionCredentialKey,
	})
	if err != nil {
		t.Fatalf("shared admin key should satisfy production platform admin preflight: %v", err)
	}
	if !hasDeploymentCheck(checks, "platform_admin_configured", "blocking", "passed") {
		t.Fatalf("expected platform admin validation pass, got %#v", checks)
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
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", strongProductionCredentialKey)

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected production preflight to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_KEY") {
		t.Fatalf("expected production preflight error before storage initialization, got %q", got)
	}
}

func TestNewRunsCredentialKeyPreflightBeforeStorageInitialization(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", strongProductionAdminKey)
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_DATABASE_URL", "://invalid-postgres-url")
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", "")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected production credential key preflight to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_CREDENTIAL_KEY") {
		t.Fatalf("expected credential key preflight error before storage initialization, got %q", got)
	}
	if got := err.Error(); strings.Contains(got, "connect postgres") {
		t.Fatalf("credential key preflight should fail before PostgreSQL connection, got %q", got)
	}
}

func TestNewRunsDatabaseURLPreflightBeforeStorageInitialization(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", strongProductionAdminKey)
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_DATABASE_URL", "postgres://agent_harbor:super-secret@127.0.0.1:bad/agent_harbor")
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", strongProductionCredentialKey)

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected production database URL preflight to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_DATABASE_URL") {
		t.Fatalf("expected database URL preflight error before storage initialization, got %q", got)
	}
	if got := err.Error(); strings.Contains(got, "connect postgres") || strings.Contains(got, "super-secret") {
		t.Fatalf("database URL preflight should fail before PostgreSQL connection without leaking credentials, got %q", got)
	}
}

func TestNewRunsApprovalReviewerPreflightBeforeStorageInitialization(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_KEY", strongProductionAdminKey)
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_DATABASE_URL", "postgres://agent_harbor:super-secret@127.0.0.1:bad/agent_harbor")
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", strongProductionCredentialKey)
	t.Setenv("AGENT_HARBOR_APPROVAL_REVIEWERS", "security-root=tenant-root")

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected production approval reviewer preflight to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_APPROVAL_REVIEWERS") {
		t.Fatalf("expected approval reviewer preflight error before storage initialization, got %q", got)
	}
	if got := err.Error(); strings.Contains(got, "connect postgres") || strings.Contains(got, "super-secret") {
		t.Fatalf("approval reviewer preflight should fail before PostgreSQL connection without leaking credentials, got %q", got)
	}
}

func TestNewRunsAdminIdentitiesPreflightBeforeStorageInitialization(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", "platform="+strongProductionAdminKey+"|role=owner|tenant=tenant-east")
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_DATABASE_URL", "postgres://agent_harbor:super-secret@127.0.0.1:bad/agent_harbor")
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", strongProductionCredentialKey)

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected production admin identity preflight to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_ADMIN_IDENTITIES") {
		t.Fatalf("expected admin identity preflight error before storage initialization, got %q", got)
	}
	if got := err.Error(); strings.Contains(got, "connect postgres") || strings.Contains(got, "super-secret") {
		t.Fatalf("admin identity preflight should fail before PostgreSQL connection without leaking credentials, got %q", got)
	}
}

func TestNewRunsPlatformAdminPreflightBeforeStorageInitialization(t *testing.T) {
	t.Setenv("AGENT_HARBOR_DEPLOYMENT_MODE", "production")
	t.Setenv("AGENT_HARBOR_ADMIN_IDENTITIES", "east=tenant-admin-production-key-32|role=tenant_admin|tenant=tenant-east|workspace=ws-support")
	t.Setenv("AGENT_HARBOR_SESSION_SECRET", strongProductionSessionSecret)
	t.Setenv("AGENT_HARBOR_DATABASE_URL", "postgres://agent_harbor:super-secret@127.0.0.1:bad/agent_harbor")
	t.Setenv("AGENT_HARBOR_CREDENTIAL_KEY", strongProductionCredentialKey)

	_, err := New(context.Background())
	if err == nil {
		t.Fatalf("expected production platform admin preflight to fail")
	}
	if got := err.Error(); !strings.Contains(got, "platform_admin") {
		t.Fatalf("expected platform admin preflight error before storage initialization, got %q", got)
	}
	if got := err.Error(); strings.Contains(got, "connect postgres") || strings.Contains(got, "super-secret") {
		t.Fatalf("platform admin preflight should fail before PostgreSQL connection without leaking credentials, got %q", got)
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
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":    "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":  strongProductionCredentialKey,
	})
	if err != nil {
		t.Fatalf("safe production config should pass preflight: %v", err)
	}
	if !hasDeploymentCheck(checks, "persistent_storage_configured", "blocking", "passed") {
		t.Fatalf("expected persistent storage pass, got %#v", checks)
	}
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

func TestDeploymentConfigPreflightRequiresProductionDatabaseStorage(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
	})
	if err == nil {
		t.Fatalf("expected missing production database URL to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_DATABASE_URL") {
		t.Fatalf("expected production config error to name missing database URL, got %q", got)
	}
	if !hasDeploymentCheck(checks, "persistent_storage_configured", "blocking", "failed") {
		t.Fatalf("expected persistent storage failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightAcceptsProductionDatabaseStorage(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":    "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":  strongProductionCredentialKey,
	})
	if err != nil {
		t.Fatalf("preflight should accept database storage: %v", err)
	}
	if !hasDeploymentCheck(checks, "persistent_storage_configured", "blocking", "passed") {
		t.Fatalf("expected persistent storage pass, got %#v", checks)
	}
	if !hasDeploymentCheck(checks, "database_url_valid", "blocking", "passed") {
		t.Fatalf("expected database URL validation pass, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksInvalidProductionDatabaseURL(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":    "postgres://agent_harbor:super-secret@127.0.0.1:bad/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":  strongProductionCredentialKey,
	})
	if err == nil {
		t.Fatalf("expected invalid production database URL to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_DATABASE_URL") {
		t.Fatalf("expected production config error to name database URL, got %q", got)
	}
	if got := err.Error(); strings.Contains(got, "super-secret") {
		t.Fatalf("database URL preflight should not leak credentials, got %q", got)
	}
	if !hasDeploymentCheck(checks, "database_url_valid", "blocking", "failed") {
		t.Fatalf("expected database URL validation failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightRequiresProductionCredentialKey(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":    "postgres://agent-harbor.example.invalid/agent_harbor",
	})
	if err == nil {
		t.Fatalf("expected missing production credential key to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_CREDENTIAL_KEY") {
		t.Fatalf("expected production config error to name missing credential key, got %q", got)
	}
	if !hasDeploymentCheck(checks, "credential_key_configured", "blocking", "failed") {
		t.Fatalf("expected credential key failure, got %#v", checks)
	}
}

func TestDeploymentConfigPreflightBlocksWeakProductionCredentialKey(t *testing.T) {
	checks, err := deploymentConfigPreflightFromEnv(map[string]string{
		"AGENT_HARBOR_DEPLOYMENT_MODE": "production",
		"AGENT_HARBOR_ADMIN_KEY":       strongProductionAdminKey,
		"AGENT_HARBOR_SESSION_SECRET":  strongProductionSessionSecret,
		"AGENT_HARBOR_DATABASE_URL":    "postgres://agent-harbor.example.invalid/agent_harbor",
		"AGENT_HARBOR_CREDENTIAL_KEY":  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err == nil {
		t.Fatalf("expected weak production credential key to fail")
	}
	if got := err.Error(); !strings.Contains(got, "AGENT_HARBOR_CREDENTIAL_KEY") {
		t.Fatalf("expected production config error to name weak credential key, got %q", got)
	}
	if !hasDeploymentCheck(checks, "credential_key_configured", "blocking", "failed") {
		t.Fatalf("expected credential key failure, got %#v", checks)
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
