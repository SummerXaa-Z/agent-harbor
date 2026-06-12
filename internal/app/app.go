package app

import (
	"context"
	"fmt"
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
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		actor, key, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES entries must use actor=key")
		}
		identity := httpapi.AdminIdentity{
			Actor: strings.TrimSpace(actor),
			Key:   strings.TrimSpace(key),
		}
		if identity.Actor == "" || identity.Key == "" {
			return nil, fmt.Errorf("AGENT_HARBOR_ADMIN_IDENTITIES entries must include actor and key")
		}
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
