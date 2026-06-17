package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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
	repo := store.Repository(store.NewMemory())
	closeFn := func() {}
	if databaseURL := os.Getenv("AGENT_HARBOR_DATABASE_URL"); databaseURL != "" {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return nil, fmt.Errorf("connect postgres: %w", err)
		}
		if err := db.Migrate(ctx, pool); err != nil {
			pool.Close()
			return nil, err
		}
		credentialKey, err := postgresCredentialKeyFromEnv()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("parse credential key: %w", err)
		}
		repo = store.NewPostgresWithCredentialKey(pool, credentialKey)
		closeFn = pool.Close
	}
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
	return &App{
		server: httpapi.New(
			repo,
			httpapi.WithAdminKey(os.Getenv("AGENT_HARBOR_ADMIN_KEY")),
			httpapi.WithAdminIdentities(adminIdentities),
			httpapi.WithSessionSecret(os.Getenv("AGENT_HARBOR_SESSION_SECRET")),
			httpapi.WithUnauthenticatedAdminAllowed(allowUnauthenticatedAdmin),
			httpapi.WithPrivateUpstreamsAllowed(allowPrivateUpstreams),
			httpapi.WithPermissionPackageApprovalReviewers(approvalReviewers),
			httpapi.WithCORSOrigins(corsOriginsFromEnv()),
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
	)

	if strings.TrimSpace(env["AGENT_HARBOR_SESSION_SECRET"]) == "" {
		checks = append(checks, deploymentConfigCheck{
			Code:     "session_secret_explicit",
			Severity: "warning",
			Status:   "warning",
			Message:  "AGENT_HARBOR_SESSION_SECRET should be set to a stable high-entropy value in production",
		})
	} else {
		checks = append(checks, deploymentConfigCheck{
			Code:     "session_secret_explicit",
			Severity: "warning",
			Status:   "passed",
			Message:  "AGENT_HARBOR_SESSION_SECRET is explicitly configured",
		})
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

func corsOriginsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("AGENT_HARBOR_CORS_ORIGINS"))
	if raw == "" {
		return nil
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
		origins = append(origins, origin)
	}
	return origins
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
