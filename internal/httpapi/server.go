package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type callerContextKey struct{}

type agentKeyContextKey struct{}

type adminActorContextKey struct{}

type AdminIdentity struct {
	Actor       string
	Key         string
	Role        string
	TenantID    string
	WorkspaceID string
}

type Server struct {
	repo                      store.Repository
	now                       func() time.Time
	adminKey                  string
	adminIdentities           []AdminIdentity
	allowUnauthenticatedAdmin bool
	allowPrivateUpstreams     bool
	approvalReviewers         []domain.PermissionPackageApprovalReviewer
	corsOrigins               []string
	defaultLocalCORSOrigins   bool
	loginFailureMu            sync.Mutex
	loginFailures             map[string]consoleLoginFailure
	sessionSecret             []byte
}

const (
	defaultProxyTimeout                 = 10 * time.Second
	maxProxyTimeout                     = 30 * time.Second
	maxRetryAttempts                    = 4
	maxRetryBackoff                     = time.Second
	maxProxyBodyBytes                   = 4 << 20
	defaultAuditLimit                   = 100
	maxAuditLimit                       = 500
	defaultPermissionPackageApprovalTTL = 24 * time.Hour
	defaultAgentKeyTTLSeconds           = int64(1800)
	maxAgentKeyTTLSeconds               = int64(3600)
	defaultConsoleSessionTTL            = 12 * time.Hour
	consoleSessionCookieName            = "agent_harbor_session"
	developmentAdminActor               = "local-dev"
	systemAPIVersion                    = "2026-06-15"
)

type Option func(*Server)

func WithAdminKey(key string) Option {
	return func(s *Server) {
		s.adminKey = strings.TrimSpace(key)
	}
}

func WithAdminIdentities(identities []AdminIdentity) Option {
	return func(s *Server) {
		s.adminIdentities = make([]AdminIdentity, 0, len(identities))
		for _, identity := range identities {
			normalized, ok := normalizeAdminIdentity(identity)
			if !ok {
				continue
			}
			s.adminIdentities = append(s.adminIdentities, normalized)
		}
	}
}

func WithSessionSecret(secret string) Option {
	return func(s *Server) {
		secret = strings.TrimSpace(secret)
		if secret != "" {
			s.sessionSecret = []byte(secret)
		}
	}
}

func WithClock(now func() time.Time) Option {
	return func(s *Server) {
		if now != nil {
			s.now = func() time.Time { return now().UTC() }
		}
	}
}

func WithUnauthenticatedAdminAllowed(allowed bool) Option {
	return func(s *Server) {
		s.allowUnauthenticatedAdmin = allowed
	}
}

func WithPrivateUpstreamsAllowed(allowed bool) Option {
	return func(s *Server) {
		s.allowPrivateUpstreams = allowed
	}
}

func WithPermissionPackageApprovalReviewers(reviewers []domain.PermissionPackageApprovalReviewer) Option {
	return func(s *Server) {
		s.approvalReviewers = make([]domain.PermissionPackageApprovalReviewer, 0, len(reviewers))
		for _, reviewer := range reviewers {
			normalized := domain.PermissionPackageApprovalReviewer{
				Reviewer:    strings.TrimSpace(reviewer.Reviewer),
				TenantID:    strings.TrimSpace(reviewer.TenantID),
				WorkspaceID: strings.TrimSpace(reviewer.WorkspaceID),
			}
			if normalized.Reviewer == "" {
				continue
			}
			s.approvalReviewers = append(s.approvalReviewers, normalized)
		}
	}
}

func WithCORSOrigins(origins []string) Option {
	return func(s *Server) {
		s.corsOrigins = make([]string, 0, len(origins))
		for _, origin := range origins {
			normalized := strings.TrimSpace(origin)
			if normalized == "" {
				continue
			}
			s.corsOrigins = append(s.corsOrigins, normalized)
		}
	}
}

func WithDefaultLocalCORSOrigins(enabled bool) Option {
	return func(s *Server) {
		s.defaultLocalCORSOrigins = enabled
	}
}

func New(repo store.Repository, options ...Option) *Server {
	server := &Server{
		repo:                    repo,
		now:                     func() time.Time { return time.Now().UTC() },
		defaultLocalCORSOrigins: true,
	}
	for _, option := range options {
		option(server)
	}
	return server
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(jsonPanicRecovery)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(browserSecurityHeaders)
	r.Use(localDevCORS(s.corsOrigins, s.defaultLocalCORSOrigins))

	r.Get("/healthz", s.health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/system/info", s.systemInfo)
		r.Get("/contracts/providers", s.listProviderContracts)
		r.Get("/contracts/channels", s.listChannelContracts)
		r.Get("/auth/session", s.getAuthSession)
		r.Post("/auth/login", s.login)
		r.Post("/auth/logout", s.logout)
		r.Group(func(r chi.Router) {
			r.Use(sensitiveResponseHeaders)
			r.Use(s.requireAdmin)
			r.Get("/admin-identities", s.listAdminIdentities)
			r.Post("/admin-identities", s.createAdminIdentity)
			r.Patch("/admin-identities/{id}", s.updateAdminIdentityOwnedAgents)
			r.Post("/admin-identities/{id}/key:rotate", s.rotateAdminIdentityKey)
			r.Post("/admin-identities/{id}:disable", s.disableAdminIdentity)
			r.Post("/tenants", s.createTenant)
			r.Get("/tenants", s.listTenants)
			r.Get("/tenants/{id}/permission-center", s.getTenantPermissionCenter)
			r.Get("/tenants/{id}/access-profile", s.getTenantAccessProfile)
			r.Get("/tenants/{id}", s.getTenant)
			r.Post("/agents", s.createAgent)
			r.Get("/agents", s.listAgents)
			r.Get("/agents/{id}", s.getAgent)
			r.Patch("/agents/{id}", s.updateAgent)
			r.Delete("/agents/{id}", s.disableAgent)
			r.Post("/agents/{id}/credentials:rotate", s.rotateAgentCredentials)
			r.Post("/agent-keys", s.createAgentKey)
			r.Get("/api-keys", s.listAgentKeys)
			r.Post("/api-keys", s.createAgentKey)
			r.Delete("/api-keys/{id}", s.revokeAgentKey)
			r.Post("/access-grants", s.createAccessGrant)
			r.Get("/access-grants", s.listAccessGrants)
			r.Delete("/access-grants/{id}", s.revokeAccessGrant)
			r.Get("/access-decisions:explain", s.explainAccessDecision)
			r.Post("/route-policies", s.createRoutePolicy)
			r.Get("/route-policies", s.listRoutePolicies)
			r.Patch("/route-policies/{id}", s.updateRoutePolicy)
			r.Delete("/route-policies/{id}", s.disableRoutePolicy)
			r.Post("/targets/{targetId}/capabilities:refresh", s.refreshTargetCapabilities)
			r.Post("/targets/{targetId}:probe", s.probeTarget)
			r.Get("/capabilities", s.listCapabilities)
			r.Patch("/capabilities/{id}", s.updateCapability)
			r.Get("/permission-packages/templates", s.listPermissionPackageTemplates)
			r.Get("/permission-packages/access-subjects", s.listPermissionPackageAccessSubjects)
			r.Post("/permission-packages/workbench:preview", s.previewPermissionPackageWorkbench)
			r.Get("/permission-packages/production-readiness/report", s.getPermissionPackageProductionEvidenceReport)
			r.Get("/permission-packages/production-readiness", s.getPermissionPackageProductionReadiness)
			r.Get("/permission-packages/access-handoff", s.getPermissionPackageAccessHandoff)
			r.Post("/permission-packages/access-handoff/tokens", s.createAccessHandoffToken)
			r.Post("/permission-packages/access-handoff/events", s.reportAccessHandoffEvent)
			r.Delete("/permission-packages/access-handoff/tokens/{id}", s.revokeAccessHandoffToken)
			r.Get("/permission-packages/applications", s.listPermissionPackageApplications)
			r.Get("/permission-packages/applications/health", s.listPermissionPackageApplicationHealth)
			r.Get("/permission-packages/applications/{id}/impact", s.getPermissionPackageApplicationImpact)
			r.Get("/permission-packages/approval-requests", s.listPermissionPackageApprovalRequests)
			r.Post("/permission-packages/approval-requests", s.createPermissionPackageApprovalRequest)
			r.Post("/permission-packages/approval-requests/{id}/approve", s.approvePermissionPackageApprovalRequest)
			r.Post("/permission-packages/approval-requests/{id}/reject", s.rejectPermissionPackageApprovalRequest)
			r.Post("/permission-packages/approval-requests/{id}/withdraw", s.withdrawPermissionPackageApprovalRequest)
			r.Post("/permission-packages/drafts", s.createPermissionPackageDraft)
			r.Post("/permission-packages:preflight", s.preflightPermissionPackage)
			r.Post("/permission-packages:apply", s.applyPermissionPackage)
			r.Post("/management/mcp", s.managementMCP)
			r.Post("/management/mcp/rpc", s.managementMCP)
			r.Post("/tenant-entitlements", s.createTenantEntitlement)
			r.Get("/tenant-entitlements", s.listTenantEntitlements)
			r.Delete("/tenant-entitlements/{id}", s.deleteTenantEntitlement)
			r.Post("/workspace-assignments", s.createWorkspaceAssignment)
			r.Get("/workspace-assignments", s.listWorkspaceAssignments)
			r.Delete("/workspace-assignments/{id}", s.deleteWorkspaceAssignment)
			r.Post("/instance-assignments", s.createInstanceAssignment)
			r.Get("/instance-assignments", s.listInstanceAssignments)
			r.Delete("/instance-assignments/{id}", s.deleteInstanceAssignment)
			r.Get("/audit/events", s.listAuditEvents)
			r.Get("/audit/traces", s.listTraces)
			r.Get("/metrics/runtime", s.runtimeMetrics)
			r.Get("/metrics/daily", s.dailyMetrics)
		})
		r.Group(func(r chi.Router) {
			r.Use(sensitiveResponseHeaders)
			r.Use(s.requireAgentKey)
			r.Get("/self/access-profile", s.getSelfAccessProfile)
			r.Post("/mcp/agents/{targetId}", s.mcpRPC)
			r.Post("/mcp/agents/{targetId}/rpc", s.mcpRPC)
			r.Post("/openapi/agents/{targetId}/operations/{operationId}", s.openapiOperation)
			r.HandleFunc("/openapi/agents/{targetId}/*", s.openapiRelativePath)
		})
	})
	return r
}

var recoveredPanicLogger = func(r *http.Request, value any, stack []byte) {
	log.Printf("%s\n%s", recoveredPanicLogMessage(r, value), string(stack))
}

func recoveredPanicLogMessage(r *http.Request, value any) string {
	return fmt.Sprintf(
		"agent-harbor panic recovered requestId=%s method=%s path=%s panicType=%T",
		middleware.GetReqID(r.Context()),
		r.Method,
		r.URL.Path,
		value,
	)
}

func jsonPanicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}
				recoveredPanicLogger(r, recovered, debug.Stack())
				if ww.Status() == 0 {
					writeError(ww, errors.New("panic recovered"))
				}
			}
		}()
		next.ServeHTTP(ww, r)
	})
}

func sensitiveResponseHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSensitiveNoCacheHeaders(w)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func setSensitiveNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func browserSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		if isHTTPSRequest(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func localDevCORS(extraOrigins []string, includeDefaultLocalOrigins bool) func(http.Handler) http.Handler {
	allowedOrigins := map[string]struct{}{}
	if includeDefaultLocalOrigins {
		allowedOrigins = map[string]struct{}{
			"http://localhost:4173": {},
			"http://localhost:4174": {},
			"http://localhost:4175": {},
			"http://localhost:4176": {},
			"http://localhost:5173": {},
			"http://localhost:5174": {},
			"http://localhost:5175": {},
			"http://localhost:5176": {},
			"http://127.0.0.1:4173": {},
			"http://127.0.0.1:4174": {},
			"http://127.0.0.1:4175": {},
			"http://127.0.0.1:4176": {},
			"http://127.0.0.1:5173": {},
			"http://127.0.0.1:5174": {},
			"http://127.0.0.1:5175": {},
			"http://127.0.0.1:5176": {},
			"http://[::1]:4173":     {},
			"http://[::1]:4174":     {},
			"http://[::1]:4175":     {},
			"http://[::1]:4176":     {},
			"http://[::1]:5173":     {},
			"http://[::1]:5174":     {},
			"http://[::1]:5175":     {},
			"http://[::1]:5176":     {},
		}
	}
	for _, origin := range extraOrigins {
		allowedOrigins[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowedOrigins[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Key, X-Run-Id, X-AgentHarbor-Subject-Id, X-AgentHarbor-CSRF")
				w.Header().Set("Vary", "Origin")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if principal, _, ok := s.developmentSession(); ok {
			ctx := context.WithValue(r.Context(), adminActorContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		provided := r.Header.Get("X-Admin-Key")
		if principal, ok := s.adminPrincipalForKey(r.Context(), provided); ok {
			ctx := context.WithValue(r.Context(), adminActorContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if sessionToken, ok := consoleSessionTokenFromRequest(r); ok {
			principal, _, sessionOK := s.verifyConsoleSession(r.Context(), sessionToken)
			if !sessionOK {
				if !s.hasConfiguredAdminAuthentication(r.Context()) {
					writeError(w, domain.Unauthorized("admin authentication is required"))
					return
				}
				writeError(w, domain.Unauthorized("missing or invalid admin key"))
				return
			}
			if err := s.validateConsoleSessionCSRF(r, sessionToken); err != nil {
				writeError(w, err)
				return
			}
			ctx := context.WithValue(r.Context(), adminActorContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if !s.hasConfiguredAdminAuthentication(r.Context()) {
			writeError(w, domain.Unauthorized("admin authentication is required"))
			return
		}
		writeError(w, domain.Unauthorized("missing or invalid admin key"))
	})
}

func (s *Server) hasConfiguredAdminAuthentication(ctx context.Context) bool {
	if s.adminKey != "" || len(s.adminIdentities) > 0 {
		return true
	}
	rows, err := s.repo.ListAdminIdentities(ctx)
	if err != nil {
		return true
	}
	for _, row := range rows {
		if row.Status == domain.AdminIdentityStatusActive {
			return true
		}
	}
	return false
}

func (s *Server) adminPrincipalForKey(ctx context.Context, provided string) (adminPrincipal, bool) {
	provided = strings.TrimSpace(provided)
	if provided != "" {
		managed, ok, err := s.repo.FindAdminIdentityByKeyHash(ctx, security.HashSecret(provided))
		if err == nil && ok && managed.Status == domain.AdminIdentityStatusActive {
			_ = s.repo.TouchAdminIdentityLastUsed(ctx, managed.ID, s.now())
			return adminPrincipalFromManagedIdentity(managed), true
		}
	}
	for _, identity := range s.adminIdentities {
		if len(provided) == len(identity.Key) && subtle.ConstantTimeCompare([]byte(provided), []byte(identity.Key)) == 1 {
			return identity.principal(), true
		}
	}
	if s.adminKey != "" {
		if len(provided) != len(s.adminKey) || subtle.ConstantTimeCompare([]byte(provided), []byte(s.adminKey)) != 1 {
			return adminPrincipal{}, false
		}
		return platformAdminPrincipal("admin-key"), true
	}
	return adminPrincipal{}, false
}

func (s *Server) adminPrincipalForActor(ctx context.Context, actor string) (adminPrincipal, bool) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return adminPrincipal{}, false
	}
	managed, ok, err := s.repo.GetAdminIdentityByActor(ctx, actor)
	if err == nil && ok && managed.Status == domain.AdminIdentityStatusActive {
		return adminPrincipalFromManagedIdentity(managed), true
	}
	for _, identity := range s.adminIdentities {
		if identity.Actor == actor {
			return identity.principal(), true
		}
	}
	if actor == "admin-key" && s.adminKey != "" {
		return platformAdminPrincipal("admin-key"), true
	}
	if actor == developmentAdminActor && s.developmentAdminBypassActive() {
		return platformAdminPrincipal(developmentAdminActor), true
	}
	return adminPrincipal{}, false
}

func requestAuthenticatedAdminActor(r *http.Request) (string, bool) {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return "", false
	}
	return principal.Actor, true
}

func reviewerFromRequest(reviewer string, r *http.Request) (string, error) {
	reviewer = strings.TrimSpace(reviewer)
	if actor, ok := requestAuthenticatedAdminActor(r); ok {
		if actor == developmentAdminActor {
			if reviewer != "" {
				return reviewer, nil
			}
			return actor, nil
		}
		if reviewer == "" {
			return actor, nil
		}
		if reviewer != actor {
			return "", domain.PermissionDenied("reviewer must match authenticated admin identity")
		}
		return reviewer, nil
	}
	if reviewer != "" {
		return reviewer, nil
	}
	return managementActor(r), nil
}

func managementScopeFromRequest(r *http.Request) store.ManagementScope {
	return store.ManagementScope{
		TenantID:    strings.TrimSpace(r.URL.Query().Get("tenantId")),
		WorkspaceID: strings.TrimSpace(r.URL.Query().Get("workspaceId")),
	}
}

func (s *Server) effectiveManagementScopeFromRequest(r *http.Request) (store.ManagementScope, error) {
	return s.effectiveManagementScopeForRequest(r, managementScopeFromRequest(r))
}

func (s *Server) effectiveManagementScopeForRequest(r *http.Request, requested store.ManagementScope) (store.ManagementScope, error) {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return requested, nil
	}
	return s.effectiveManagementScope(r.Context(), requested, principal)
}

func (s *Server) managementAuditEvent(r *http.Request, tenantID string, workspaceID string, action string, resourceType string, resourceID string, summary string, metadata map[string]any) domain.AuditEvent {
	if metadata == nil {
		metadata = map[string]any{}
	}
	if confirmation, ok := managementMCPWriteConfirmationFromContext(r.Context()); ok {
		metadata["managementMcpTool"] = confirmation.ToolName
		metadata["managementMcpConfirmationReason"] = confirmation.Reason
	}
	return domain.AuditEvent{
		ID:           security.NewID("aud"),
		TenantID:     tenantID,
		WorkspaceID:  workspaceID,
		Actor:        managementActor(r),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Summary:      summary,
		Metadata:     metadata,
		CreatedAt:    s.now(),
	}
}

func managementActor(r *http.Request) string {
	if actor, ok := requestAuthenticatedAdminActor(r); ok {
		return actor
	}
	if strings.TrimSpace(r.Header.Get("X-Admin-Key")) != "" {
		return "admin-key"
	}
	return developmentAdminActor
}
