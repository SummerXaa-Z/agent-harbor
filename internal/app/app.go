package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SummerXaa-Z/agent-harbor/internal/db"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type App struct {
	server *httpapi.Server
	close  func()
}

type deploymentConfigCheck struct {
	Code     string
	Severity string
	Status   string
	Message  string
}

func New(ctx context.Context) (*App, error) {
	closeFn := func() {}
	deploymentEnv := deploymentEnvFromOS()
	deploymentMode, err := deploymentModeFromEnv(deploymentEnv)
	if err != nil {
		closeFn()
		return nil, err
	}
	deploymentChecks, err := deploymentConfigPreflightFromEnv(deploymentEnv)
	if err != nil {
		closeFn()
		return nil, err
	}
	logDeploymentConfigWarnings(deploymentChecks)

	allowPrivateUpstreams, err := privateUpstreamsAllowedFromEnv()
	if err != nil {
		closeFn()
		return nil, err
	}
	approvalReviewers, err := approvalReviewersFromEnv()
	if err != nil {
		closeFn()
		return nil, err
	}
	adminIdentities, err := adminIdentitiesFromEnv()
	if err != nil {
		closeFn()
		return nil, err
	}
	allowUnauthenticatedAdmin, err := unauthenticatedAdminAllowedFromEnv()
	if err != nil {
		closeFn()
		return nil, err
	}
	corsOrigins, err := corsOriginsFromEnv()
	if err != nil {
		closeFn()
		return nil, err
	}

	repo := store.Repository(store.NewMemory())
	if databaseURL := os.Getenv("AGENT_HARBOR_DATABASE_URL"); databaseURL != "" {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			closeFn()
			return nil, fmt.Errorf("connect postgres: %w", err)
		}
		if err := db.Migrate(ctx, pool); err != nil {
			pool.Close()
			closeFn()
			return nil, err
		}
		credentialKey, err := postgresCredentialKeyFromEnv()
		if err != nil {
			pool.Close()
			closeFn()
			return nil, fmt.Errorf("parse credential key: %w", err)
		}
		repo = store.NewPostgresWithCredentialKey(pool, credentialKey)
		closeFn = pool.Close
	}

	return &App{
		server: httpapi.New(
			repo,
			httpapi.WithAdminKey(os.Getenv("AGENT_HARBOR_ADMIN_KEY")),
			httpapi.WithAdminIdentities(adminIdentities),
			httpapi.WithSessionSecret(os.Getenv("AGENT_HARBOR_SESSION_SECRET")),
			httpapi.WithUnauthenticatedAdminAllowed(allowUnauthenticatedAdmin),
			httpapi.WithPrivateUpstreamsAllowed(allowPrivateUpstreams),
			httpapi.WithPermissionPackageApprovalReviewers(approvalReviewers),
			httpapi.WithCORSOrigins(corsOrigins),
			httpapi.WithDefaultLocalCORSOrigins(deploymentMode != "production"),
		),
		close: closeFn,
	}, nil
}

func (a *App) Router() http.Handler {
	return a.server.Router()
}

func (a *App) Close() {
	a.close()
}

func deploymentEnvFromOS() map[string]string {
	keys := []string{
		"AGENT_HARBOR_DEPLOYMENT_MODE",
		"AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN",
		"AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS",
		"AGENT_HARBOR_ADMIN_KEY",
		"AGENT_HARBOR_ADMIN_IDENTITIES",
		"AGENT_HARBOR_SESSION_SECRET",
		"AGENT_HARBOR_DATABASE_URL",
		"AGENT_HARBOR_CORS_ORIGINS",
	}
	env := make(map[string]string, len(keys))
	for _, key := range keys {
		env[key] = os.Getenv(key)
	}
	return env
}

func deploymentConfigPreflightFromEnv(env map[string]string) ([]deploymentConfigCheck, error) {
	mode, err := deploymentModeFromEnv(env)
	if err != nil {
		return nil, err
	}
	checks := validateDeploymentConfig(mode, env)
	var blockers []string
	for _, check := range checks {
		if check.Severity == "blocking" && check.Status == "failed" {
			blockers = append(blockers, check.Message)
		}
	}
	if len(blockers) > 0 {
		return checks, fmt.Errorf("production deployment config failed: %s", strings.Join(blockers, "; "))
	}
	return checks, nil
}

func deploymentModeFromEnv(env map[string]string) (string, error) {
	raw := strings.TrimSpace(env["AGENT_HARBOR_DEPLOYMENT_MODE"])
	if raw == "" {
		return "development", nil
	}
	switch raw {
	case "development", "production":
		return raw, nil
	default:
		return "", fmt.Errorf("AGENT_HARBOR_DEPLOYMENT_MODE must be development or production")
	}
}

func validateDeploymentConfig(mode string, env map[string]string) []deploymentConfigCheck {
	checks := []deploymentConfigCheck{
		{
			Code:     "deployment_mode",
			Severity: "info",
			Status:   "passed",
			Message:  "deployment mode is " + mode,
		},
	}
	if mode != "production" {
		return checks
	}

	checks = append(checks,
		productionBlockingCheck(
			"unauthenticated_admin_disabled",
			envBoolTrue(env["AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN"]),
			"AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN must not be true in production",
			"unauthenticated admin access is disabled",
		),
		productionBlockingCheck(
			"private_upstreams_disabled",
			envBoolTrue(env["AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS"]),
			"AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS must not be true in production",
			"private upstream access is disabled",
		),
		productionBlockingCheck(
			"admin_authentication_configured",
			strings.TrimSpace(env["AGENT_HARBOR_ADMIN_KEY"]) == "" && strings.TrimSpace(env["AGENT_HARBOR_ADMIN_IDENTITIES"]) == "",
			"AGENT_HARBOR_ADMIN_KEY or AGENT_HARBOR_ADMIN_IDENTITIES is required in production",
			"admin authentication is configured",
		),
		productionAdminKeyStrengthCheck(env),
		productionAdminKeyUniquenessCheck(env),
		productionCORSOriginsCheck(env),
	)

	if strings.TrimSpace(env["AGENT_HARBOR_SESSION_SECRET"]) == "" {
		checks = append(checks, deploymentConfigCheck{
			Code:     "session_secret_explicit",
			Severity: "blocking",
			Status:   "failed",
			Message:  "AGENT_HARBOR_SESSION_SECRET is required in production",
		})
	} else {
		checks = append(checks, deploymentConfigCheck{
			Code:     "session_secret_explicit",
			Severity: "blocking",
			Status:   "passed",
			Message:  "AGENT_HARBOR_SESSION_SECRET is explicitly configured",
		})
		checks = append(checks, productionSessionSecretStrengthCheck(env))
	}
	if strings.TrimSpace(env["AGENT_HARBOR_DATABASE_URL"]) == "" {
		checks = append(checks, deploymentConfigCheck{
			Code:     "persistent_storage_configured",
			Severity: "warning",
			Status:   "warning",
			Message:  "AGENT_HARBOR_DATABASE_URL should be set in production so tenants, grants, credentials, and audit records survive restart",
		})
	} else {
		checks = append(checks, deploymentConfigCheck{
			Code:     "persistent_storage_configured",
			Severity: "warning",
			Status:   "passed",
			Message:  "persistent database storage is configured",
		})
	}
	return checks
}

func logDeploymentConfigWarnings(checks []deploymentConfigCheck) {
	for _, check := range checks {
		if check.Severity == "warning" && check.Status == "warning" {
			slog.Warn("deployment configuration warning", "code", check.Code, "message", check.Message)
		}
	}
}

const minProductionAdminKeyLength = 16
const minProductionSessionSecretLength = 32

var commonWeakProductionSecretValues = map[string]struct{}{
	"admin":           {},
	"agent-harbor":    {},
	"agentharbor":     {},
	"changeme":        {},
	"change-me":       {},
	"local-admin-key": {},
	"password":        {},
	"secret":          {},
	"test":            {},
	"test-admin":      {},
}

func productionAdminKeyStrengthCheck(env map[string]string) deploymentConfigCheck {
	var failures []string
	if key := strings.TrimSpace(env["AGENT_HARBOR_ADMIN_KEY"]); key != "" && weakProductionAdminKey(key) {
		failures = append(failures, "AGENT_HARBOR_ADMIN_KEY must be at least 16 characters and not a common weak value")
	}
	for _, key := range adminIdentityKeysFromConfig(env["AGENT_HARBOR_ADMIN_IDENTITIES"]) {
		if weakProductionAdminKey(key) {
			failures = append(failures, "AGENT_HARBOR_ADMIN_IDENTITIES keys must be at least 16 characters and not common weak values")
			break
		}
	}
	check := deploymentConfigCheck{
		Code:     "admin_key_strength",
		Severity: "blocking",
		Status:   "passed",
		Message:  "bootstrap admin keys meet minimum production strength",
	}
	if len(failures) > 0 {
		check.Status = "failed"
		check.Message = strings.Join(failures, "; ")
	}
	return check
}

func productionAdminKeyUniquenessCheck(env map[string]string) deploymentConfigCheck {
	check := deploymentConfigCheck{
		Code:     "admin_key_uniqueness",
		Severity: "blocking",
		Status:   "passed",
		Message:  "shared and named bootstrap admin keys are distinct",
	}
	sharedKey := strings.TrimSpace(env["AGENT_HARBOR_ADMIN_KEY"])
	if sharedKey == "" {
		return check
	}
	for _, identityKey := range adminIdentityKeysFromConfig(env["AGENT_HARBOR_ADMIN_IDENTITIES"]) {
		if identityKey == sharedKey {
			check.Status = "failed"
			check.Message = "AGENT_HARBOR_ADMIN_KEY must not match any AGENT_HARBOR_ADMIN_IDENTITIES key"
			return check
		}
	}
	return check
}

func weakProductionAdminKey(key string) bool {
	return weakProductionSecretValue(key, minProductionAdminKeyLength)
}

func productionSessionSecretStrengthCheck(env map[string]string) deploymentConfigCheck {
	check := deploymentConfigCheck{
		Code:     "session_secret_strength",
		Severity: "blocking",
		Status:   "passed",
		Message:  "session secret meets minimum production strength",
	}
	if weakProductionSecretValue(env["AGENT_HARBOR_SESSION_SECRET"], minProductionSessionSecretLength) {
		check.Status = "failed"
		check.Message = "AGENT_HARBOR_SESSION_SECRET must be at least 32 characters and not a common weak value"
	}
	return check
}

func weakProductionSecretValue(secret string, minLength int) bool {
	secret = strings.TrimSpace(secret)
	if len(secret) < minLength {
		return true
	}
	_, weak := commonWeakProductionSecretValues[strings.ToLower(secret)]
	return weak
}

func adminIdentityKeysFromConfig(raw string) []string {
	entries := strings.FieldsFunc(strings.TrimSpace(raw), func(r rune) bool {
		return r == ',' || r == ';'
	})
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		_, config, ok := strings.Cut(strings.TrimSpace(entry), "=")
		if !ok {
			continue
		}
		key, _, _ := strings.Cut(strings.TrimSpace(config), "|")
		key = strings.TrimSpace(key)
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func envBoolTrue(raw string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && parsed
}

func productionBlockingCheck(code string, failed bool, failedMessage string, passedMessage string) deploymentConfigCheck {
	check := deploymentConfigCheck{
		Code:     code,
		Severity: "blocking",
		Status:   "passed",
		Message:  passedMessage,
	}
	if failed {
		check.Status = "failed"
		check.Message = failedMessage
	}
	return check
}

func postgresCredentialKeyFromEnv() ([]byte, error) {
	raw := os.Getenv("AGENT_HARBOR_CREDENTIAL_KEY")
	if raw == "" {
		return nil, fmt.Errorf("AGENT_HARBOR_CREDENTIAL_KEY is required when AGENT_HARBOR_DATABASE_URL is set")
	}
	return security.ParseCredentialKey(raw)
}

func privateUpstreamsAllowedFromEnv() (bool, error) {
	raw := strings.TrimSpace(os.Getenv("AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS"))
	if raw == "" {
		return false, nil
	}
	allowed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("AGENT_HARBOR_ALLOW_PRIVATE_UPSTREAMS must be a boolean")
	}
	return allowed, nil
}

func unauthenticatedAdminAllowedFromEnv() (bool, error) {
	raw := strings.TrimSpace(os.Getenv("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN"))
	if raw == "" {
		return false, nil
	}
	allowed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("AGENT_HARBOR_ALLOW_UNAUTHENTICATED_ADMIN must be a boolean")
	}
	return allowed, nil
}

func productionCORSOriginsCheck(env map[string]string) deploymentConfigCheck {
	if _, err := corsOriginsFromRaw(env["AGENT_HARBOR_CORS_ORIGINS"]); err != nil {
		return deploymentConfigCheck{
			Code:     "cors_origins_valid",
			Severity: "blocking",
			Status:   "failed",
			Message:  err.Error(),
		}
	}
	return deploymentConfigCheck{
		Code:     "cors_origins_valid",
		Severity: "blocking",
		Status:   "passed",
		Message:  "CORS origins are valid",
	}
}

func corsOriginsFromEnv() ([]string, error) {
	return corsOriginsFromRaw(os.Getenv("AGENT_HARBOR_CORS_ORIGINS"))
}

func corsOriginsFromRaw(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	entries := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';'
	})
	origins := make([]string, 0, len(entries))
	for _, entry := range entries {
		origin := strings.TrimSpace(entry)
		if origin == "" {
			continue
		}
		if err := validateCORSOrigin(origin); err != nil {
			return nil, err
		}
		origins = append(origins, origin)
	}
	return origins, nil
}

func validateCORSOrigin(origin string) error {
	if strings.Contains(origin, "*") {
		return fmt.Errorf("AGENT_HARBOR_CORS_ORIGINS must contain explicit http or https origins, got %q", origin)
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("AGENT_HARBOR_CORS_ORIGINS entries must be absolute http or https origins, got %q", origin)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("AGENT_HARBOR_CORS_ORIGINS entries must use http or https, got %q", origin)
	}
	if parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("AGENT_HARBOR_CORS_ORIGINS entries must not include user info, path, query, or fragment, got %q", origin)
	}
	return nil
}

func adminIdentitiesFromEnv() ([]httpapi.AdminIdentity, error) {
	raw := strings.TrimSpace(os.Getenv("AGENT_HARBOR_ADMIN_IDENTITIES"))
	if raw == "" {
		return nil, nil
	}
	entries := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';'
	})
	identities := make([]httpapi.AdminIdentity, 0, len(entries))
	seenActors := map[string]struct{}{}
	seenKeys := map[string]struct{}{}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		actor, config, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES entries must use actor=key")
		}
		parts := strings.Split(strings.TrimSpace(config), "|")
		identity := httpapi.AdminIdentity{
			Actor: strings.TrimSpace(actor),
			Key:   strings.TrimSpace(parts[0]),
			Role:  "platform_admin",
		}
		for _, part := range parts[1:] {
			name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
			if !ok {
				return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES scoped attributes must use name=value")
			}
			switch strings.TrimSpace(name) {
			case "role":
				identity.Role = strings.TrimSpace(value)
			case "tenant", "tenantId":
				identity.TenantID = strings.TrimSpace(value)
			case "workspace", "workspaceId":
				identity.WorkspaceID = strings.TrimSpace(value)
			default:
				return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES unknown scoped attribute %q", strings.TrimSpace(name))
			}
		}
		if identity.Actor == "" || identity.Key == "" {
			return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES entries must include actor and key")
		}
		if _, ok := seenActors[identity.Actor]; ok {
			return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES duplicate actor %q", identity.Actor)
		}
		if _, ok := seenKeys[identity.Key]; ok {
			return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES duplicate key for actor %q", identity.Actor)
		}
		if identity.Role == "" {
			identity.Role = "platform_admin"
		}
		if identity.Role != "platform_admin" && identity.TenantID == "" {
			return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES scoped admin roles must include tenant")
		}
		seenActors[identity.Actor] = struct{}{}
		seenKeys[identity.Key] = struct{}{}
		identities = append(identities, identity)
	}
	return identities, nil
}

func approvalReviewersFromEnv() ([]domain.PermissionPackageApprovalReviewer, error) {
	raw := strings.TrimSpace(os.Getenv("AGENT_HARBOR_APPROVAL_REVIEWERS"))
	if raw == "" {
		return nil, nil
	}
	entries := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';'
	})
	reviewers := make([]domain.PermissionPackageApprovalReviewer, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		reviewer, scope, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("AGENT_HARBOR_APPROVAL_REVIEWERS entries must use reviewer=tenantId/workspaceId")
		}
		tenantID, workspaceID, ok := strings.Cut(strings.TrimSpace(scope), "/")
		if !ok {
			return nil, fmt.Errorf("AGENT_HARBOR_APPROVAL_REVIEWERS entries must use reviewer=tenantId/workspaceId")
		}
		rule := domain.PermissionPackageApprovalReviewer{
			Reviewer:    strings.TrimSpace(reviewer),
			TenantID:    strings.TrimSpace(tenantID),
			WorkspaceID: strings.TrimSpace(workspaceID),
		}
		if rule.Reviewer == "" || rule.TenantID == "" || rule.WorkspaceID == "" {
			return nil, fmt.Errorf("AGENT_HARBOR_APPROVAL_REVIEWERS entries must include reviewer, tenantId, and workspaceId")
		}
		reviewers = append(reviewers, rule)
	}
	return reviewers, nil
}
