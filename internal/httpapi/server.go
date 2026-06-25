package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/SummerXaa-Z/agent-harbor/internal/contracts"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/permissionpack"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type callerContextKey struct{}
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
	defaultConsoleSessionTTL            = 12 * time.Hour
	consoleSessionCookieName            = "agent_harbor_session"
	developmentAdminActor               = "local-dev"
	systemAPIVersion                    = "2026-06-15"
)

var systemCapabilities = []string{
	"permission_package_approval_requests",
	"permission_package_approval_withdraw",
	"permission_package_apply_preflight",
	"permission_package_applications",
	"permission_package_application_health",
	"permission_package_application_impact",
	"permission_package_production_readiness",
	"permission_package_consumed_approval_recovery",
}

type proxyRetryPolicy struct {
	maxAttempts      int
	backoff          time.Duration
	retryStatusCodes map[int]struct{}
}

type proxyTraceResult struct {
	durationMs       int64
	upstreamAttempts int
	upstreamStatus   int
	upstreamError    string
}

type upstreamRequestMutator func(*http.Request) error

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
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
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
			r.Use(s.requireAdmin)
			r.Get("/admin-identities", s.listAdminIdentities)
			r.Post("/admin-identities", s.createAdminIdentity)
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
			r.Get("/capabilities", s.listCapabilities)
			r.Patch("/capabilities/{id}", s.updateCapability)
			r.Get("/permission-packages/templates", s.listPermissionPackageTemplates)
			r.Get("/permission-packages/access-subjects", s.listPermissionPackageAccessSubjects)
			r.Post("/permission-packages/workbench:preview", s.previewPermissionPackageWorkbench)
			r.Get("/permission-packages/production-readiness/report", s.getPermissionPackageProductionEvidenceReport)
			r.Get("/permission-packages/production-readiness", s.getPermissionPackageProductionReadiness)
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
			r.Post("/workspace-assignments", s.createWorkspaceAssignment)
			r.Get("/workspace-assignments", s.listWorkspaceAssignments)
			r.Post("/instance-assignments", s.createInstanceAssignment)
			r.Get("/instance-assignments", s.listInstanceAssignments)
			r.Get("/audit/events", s.listAuditEvents)
			r.Get("/audit/traces", s.listTraces)
			r.Get("/metrics/runtime", s.runtimeMetrics)
		})
		r.Group(func(r chi.Router) {
			r.Use(s.requireAgentKey)
			r.Post("/mcp/agents/{targetId}", s.mcpRPC)
			r.Post("/mcp/agents/{targetId}/rpc", s.mcpRPC)
			r.Post("/openapi/agents/{targetId}/operations/{operationId}", s.openapiOperation)
			r.HandleFunc("/openapi/agents/{targetId}/*", s.openapiRelativePath)
		})
	})
	return r
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

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type systemInfoResponse struct {
	Name         string   `json:"name"`
	APIVersion   string   `json:"apiVersion"`
	AuthRequired bool     `json:"authRequired"`
	Capabilities []string `json:"capabilities"`
}

func (s *Server) systemInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, systemInfoResponse{
		Name:         "AgentHarbor",
		APIVersion:   systemAPIVersion,
		AuthRequired: !s.developmentAdminBypassActive(),
		Capabilities: append([]string(nil), systemCapabilities...),
	})
}

func (s *Server) listProviderContracts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Providers())
}

func (s *Server) listChannelContracts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Channels())
}

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTenantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	tenant, err := s.tenantFromRequest(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	tenantScopeID := tenant.ID
	if tenant.ParentTenantID != "" {
		tenantScopeID = tenant.ParentTenantID
	}
	if err := s.requireTenantManagementScope(r, tenantScopeID); err != nil {
		writeError(w, err)
		return
	}
	created, err := s.repo.CreateTenantWithAudit(r.Context(), tenant, func(created domain.Tenant) domain.AuditEvent {
		return s.managementAuditEvent(r, created.ID, "", "tenant.created", "tenant", created.ID, "Tenant created", map[string]any{
			"parentTenantId": created.ParentTenantID,
			"level":          created.Level,
			"status":         created.Status,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) tenantFromRequest(ctx context.Context, req domain.CreateTenantRequest) (domain.Tenant, error) {
	req.ID = strings.TrimSpace(req.ID)
	req.ParentTenantID = strings.TrimSpace(req.ParentTenantID)
	req.Name = strings.TrimSpace(req.Name)
	if req.ID == "" {
		req.ID = security.NewID("ten")
	}
	if !validTenantID(req.ID) {
		return domain.Tenant{}, domain.BadRequest("VALIDATION_FAILED", "tenant id is invalid")
	}
	if req.Name == "" {
		return domain.Tenant{}, domain.BadRequest("VALIDATION_FAILED", "name is required")
	}
	status, err := normalizeTenantStatus(req.Status, domain.TenantStatusActive)
	if err != nil {
		return domain.Tenant{}, err
	}
	level := 1
	if req.ParentTenantID != "" {
		parent, ok, err := s.repo.GetTenant(ctx, req.ParentTenantID)
		if err != nil {
			return domain.Tenant{}, err
		}
		if !ok {
			return domain.Tenant{}, domain.NotFound("parent tenant not found")
		}
		if parent.Status != domain.TenantStatusActive {
			return domain.Tenant{}, domain.BadRequest("VALIDATION_FAILED", "parent tenant must be active")
		}
		if parent.Level >= 3 {
			return domain.Tenant{}, domain.BadRequest("VALIDATION_FAILED", "tenant hierarchy supports at most three levels")
		}
		level = parent.Level + 1
	}
	now := s.now()
	return domain.Tenant{
		ID:             req.ID,
		ParentTenantID: req.ParentTenantID,
		Level:          level,
		Name:           req.Name,
		Status:         status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListTenants(r.Context(), store.TenantFilter{
		TenantID:       scope.TenantID,
		ParentTenantID: strings.TrimSpace(r.URL.Query().Get("parentTenantId")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) getTenant(w http.ResponseWriter, r *http.Request) {
	tenant, ok, err := s.repo.GetTenant(r.Context(), strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("tenant not found"))
		return
	}
	writeJSON(w, http.StatusOK, tenant)
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	agent, err := s.agentFromRequest(req)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.requireAgentManagementScope(r, agent); err != nil {
		writeError(w, err)
		return
	}
	if err := s.rejectDuplicateAgentCreate(r.Context(), agent); err != nil {
		writeError(w, err)
		return
	}
	created, err := s.repo.CreateAgentWithAudit(r.Context(), agent, func(created domain.Agent) domain.AuditEvent {
		return s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "agent.created", "agent", created.ID, "Agent created", map[string]any{
			"channelType":        created.ChannelType,
			"status":             string(created.Status),
			"credentialVersion":  created.CredentialVersion,
			"hasCredentials":     len(created.Credentials) > 0,
			"credentialKeyCount": len(created.Credentials),
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) agentFromRequest(req domain.CreateAgentRequest) (domain.Agent, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	req.ChannelType = strings.TrimSpace(req.ChannelType)
	if req.TenantID == "" {
		req.TenantID = "default"
	}
	if req.Status == "" {
		req.Status = domain.AgentStatusDraft
	}
	if req.ChannelType == "" {
		req.ChannelType = "local"
	}
	if req.ChannelConfig == nil {
		req.ChannelConfig = map[string]any{}
	}
	credentials, err := normalizeCredentials(req.Credentials)
	if err != nil {
		return domain.Agent{}, err
	}
	now := s.now()
	agent := domain.Agent{
		ID:                security.NewID("agt"),
		TenantID:          req.TenantID,
		WorkspaceID:       req.WorkspaceID,
		Name:              req.Name,
		Description:       req.Description,
		OwnerID:           req.OwnerID,
		ChannelType:       req.ChannelType,
		ChannelConfig:     req.ChannelConfig,
		Credentials:       credentials,
		CredentialVersion: initialCredentialVersion(credentials),
		Status:            req.Status,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := validateAgentForSave(agent, s.allowPrivateUpstreams); err != nil {
		return domain.Agent{}, err
	}
	return agent, nil
}

func validateAgentForSave(agent domain.Agent, allowPrivateUpstreams bool) error {
	if strings.TrimSpace(agent.Name) == "" {
		return domain.BadRequest("VALIDATION_FAILED", "name is required")
	}
	if strings.TrimSpace(agent.WorkspaceID) == "" {
		return domain.BadRequest("VALIDATION_FAILED", "workspaceId is required")
	}
	if agent.Status != domain.AgentStatusDraft && agent.Status != domain.AgentStatusActive && agent.Status != domain.AgentStatusDisabled {
		return domain.BadRequest("VALIDATION_FAILED", "status must be draft, active, or disabled")
	}
	channel, ok := contracts.Channel(agent.ChannelType)
	if !ok {
		return domain.BadRequest("VALIDATION_FAILED", "channelType is not supported")
	}
	if agent.ChannelConfig == nil {
		agent.ChannelConfig = map[string]any{}
	}
	if channelConfigContainsSecretLikeKey(agent.ChannelConfig) {
		return domain.BadRequest("VALIDATION_FAILED", "channelConfig must not contain secret-like keys")
	}
	if err := validateConfiguredHeaders(agent.ChannelConfig); err != nil {
		return err
	}
	if err := validateCredentialHeaders(agent.ChannelConfig, agent.Credentials); err != nil {
		return err
	}
	if _, err := proxyTimeoutFromConfig(agent.ChannelConfig); err != nil {
		return err
	}
	if _, err := proxyRetryPolicyFromConfig(agent.ChannelConfig); err != nil {
		return err
	}
	for _, key := range []string{"endpoint", "specUrl"} {
		if raw, exists := agent.ChannelConfig[key]; exists {
			value, ok := raw.(string)
			if !ok {
				return domain.BadRequest("VALIDATION_FAILED", key+" must be a string URL")
			}
			if err := security.ValidateOutboundEndpoint(value, security.EndpointValidationOptions{AllowPrivateHosts: allowPrivateUpstreams}); err != nil {
				return domain.BadRequest("VALIDATION_FAILED", err.Error())
			}
		}
	}
	if agent.Status == domain.AgentStatusActive && channel.EndpointRequiredWhenActive {
		endpoint, ok := agent.ChannelConfig["endpoint"].(string)
		if !ok || strings.TrimSpace(endpoint) == "" {
			return domain.BadRequest("VALIDATION_FAILED", "active "+agent.ChannelType+" agent requires channelConfig.endpoint")
		}
	}
	return nil
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListAgents(r.Context(), store.AgentFilter{ManagementScope: scope})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	agent, ok, err := s.repo.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent not found"))
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	existing, ok, err := s.repo.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	updated := existing
	if req.Name != nil {
		updated.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updated.Description = strings.TrimSpace(*req.Description)
	}
	if req.OwnerID != nil {
		updated.OwnerID = strings.TrimSpace(*req.OwnerID)
	}
	if req.Status != nil {
		updated.Status = *req.Status
	}
	if req.ChannelConfig != nil {
		updated.ChannelConfig = *req.ChannelConfig
		if updated.ChannelConfig == nil {
			updated.ChannelConfig = map[string]any{}
		}
	}
	updated.UpdatedAt = s.now()
	if err := validateAgentForSave(updated, s.allowPrivateUpstreams); err != nil {
		writeError(w, err)
		return
	}
	fields := agentPatchFields(req)
	saved, ok, err := s.repo.UpdateAgentWithAudit(r.Context(), updated, func(saved domain.Agent) domain.AuditEvent {
		return s.managementAuditEvent(r, saved.TenantID, saved.WorkspaceID, "agent.updated", "agent", saved.ID, "Agent updated", map[string]any{
			"fields":            fields,
			"status":            string(saved.Status),
			"credentialVersion": saved.CredentialVersion,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent not found"))
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) rotateAgentCredentials(w http.ResponseWriter, r *http.Request) {
	var req domain.RotateAgentCredentialsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	credentials, err := normalizeCredentials(req.Credentials)
	if err != nil {
		writeError(w, err)
		return
	}
	if len(credentials) == 0 {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "credentials must include at least one credential"))
		return
	}
	agent, ok, err := s.repo.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, agent); err != nil {
		writeError(w, err)
		return
	}
	effective := agent
	effective.Credentials = credentials
	if err := validateAgentForSave(effective, s.allowPrivateUpstreams); err != nil {
		writeError(w, err)
		return
	}
	if sameCredentials(agent.Credentials, credentials) {
		writeJSON(w, http.StatusOK, agent)
		return
	}
	now := s.now()
	updated, ok, err := s.repo.RotateAgentCredentialsWithAudit(r.Context(), agent.ID, credentials, now, func(updated domain.Agent) domain.AuditEvent {
		return s.managementAuditEvent(r, updated.TenantID, updated.WorkspaceID, "agent.credentials_rotated", "agent", updated.ID, "Agent credentials rotated", map[string]any{
			"credentialKeys":    credentialKeyNames(updated.Credentials),
			"credentialVersion": updated.CredentialVersion,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent not found"))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) disableAgent(w http.ResponseWriter, r *http.Request) {
	agent, ok, err := s.repo.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, agent); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	disabled, ok, err := s.repo.DisableAgentWithAudit(r.Context(), agent.ID, now, func(disabled domain.Agent) domain.AuditEvent {
		return s.managementAuditEvent(r, disabled.TenantID, disabled.WorkspaceID, "agent.disabled", "agent", disabled.ID, "Agent disabled", map[string]any{
			"status":            string(disabled.Status),
			"credentialVersion": disabled.CredentialVersion,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent not found"))
		return
	}
	writeJSON(w, http.StatusOK, disabled)
}

func (s *Server) createAgentKey(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateAgentKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.AgentID = strings.TrimSpace(req.AgentID)
	req.Name = strings.TrimSpace(req.Name)
	if strings.TrimSpace(req.AgentID) == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "agentId is required"))
		return
	}
	if req.Name == "" {
		req.Name = "dev-agent-key"
	}
	agent, ok, err := s.repo.GetAgent(r.Context(), req.AgentID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, agent); err != nil {
		writeError(w, err)
		return
	}
	if agent.Status != domain.AgentStatusActive {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "agent key requires active caller agent"))
		return
	}
	if agent.ChannelType != "local" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "agent key can only be issued for local caller agents"))
		return
	}
	ttl := req.ExpiresInSeconds
	if ttl == 0 {
		ttl = 1800
	} else if ttl < 0 || ttl > 3600 {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "expiresInSeconds must be between 1 and 3600"))
		return
	}
	now := s.now()
	if err := s.rejectRecentDuplicateAgentKey(r.Context(), agent, req.Name, now); err != nil {
		writeError(w, err)
		return
	}
	plaintext, prefix := security.NewAgentKey()
	key := domain.AgentKey{
		ID:        security.NewID("key"),
		AgentID:   req.AgentID,
		Name:      req.Name,
		Hash:      security.HashSecret(plaintext),
		Prefix:    prefix,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(ttl) * time.Second),
	}
	created, err := s.repo.CreateAgentKeyWithAudit(r.Context(), key, func(created domain.AgentKey) domain.AuditEvent {
		return s.managementAuditEvent(r, agent.TenantID, agent.WorkspaceID, "agent_key.created", "agent_key", created.ID, "Agent key created", map[string]any{
			"agentId":   created.AgentID,
			"name":      created.Name,
			"expiresAt": created.ExpiresAt,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, domain.CreateAgentKeyResponse{
		ID:        created.ID,
		AgentID:   created.AgentID,
		Name:      created.Name,
		Key:       plaintext,
		Prefix:    created.Prefix,
		CreatedAt: created.CreatedAt,
		ExpiresAt: created.ExpiresAt,
	})
}

func (s *Server) listAgentKeys(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListAgentKeys(r.Context(), scope)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) revokeAgentKey(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "id")
	tenantID, workspaceID := "", ""
	foundForAudit := false
	keys, err := s.repo.ListAgentKeys(r.Context(), store.ManagementScope{})
	if err != nil {
		writeError(w, err)
		return
	}
	for _, existing := range keys {
		if existing.ID != keyID {
			continue
		}
		foundForAudit = true
		if agent, ok, err := s.repo.GetAgent(r.Context(), existing.AgentID); err != nil {
			writeError(w, err)
			return
		} else if ok {
			tenantID = agent.TenantID
			workspaceID = agent.WorkspaceID
		}
		break
	}
	if !foundForAudit {
		writeError(w, domain.NotFound("agent key not found"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: tenantID, WorkspaceID: workspaceID}); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	key, ok, err := s.repo.RevokeAgentKeyWithAudit(r.Context(), keyID, now, func(revoked domain.AgentKey) domain.AuditEvent {
		return s.managementAuditEvent(r, tenantID, workspaceID, "agent_key.revoked", "agent_key", revoked.ID, "Agent key revoked", map[string]any{
			"agentId": revoked.AgentID,
			"name":    revoked.Name,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("agent key not found"))
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (s *Server) createAccessGrant(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateAccessGrantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.CallerID = strings.TrimSpace(req.CallerID)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.RouteType = strings.TrimSpace(req.RouteType)
	req.RouteKey = strings.TrimSpace(req.RouteKey)
	if req.CallerID == "" || req.TargetID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "callerAgentId and targetAgentId are required"))
		return
	}
	caller, ok, err := s.repo.GetAgent(r.Context(), req.CallerID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("caller agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, caller); err != nil {
		writeError(w, err)
		return
	}
	target, ok, err := s.repo.GetAgent(r.Context(), req.TargetID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, target); err != nil {
		writeError(w, err)
		return
	}
	grant := domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  req.CallerID,
		TargetID:  req.TargetID,
		RouteType: req.RouteType,
		RouteKey:  req.RouteKey,
		CreatedAt: s.now(),
	}
	created, err := s.repo.CreateAccessGrantWithAudit(r.Context(), grant, func(created domain.AccessGrant) domain.AuditEvent {
		return s.managementAuditEvent(r, caller.TenantID, caller.WorkspaceID, "access_grant.created", "access_grant", created.ID, "Access grant created", map[string]any{
			"callerAgentId": created.CallerID,
			"targetAgentId": created.TargetID,
			"routeType":     created.RouteType,
			"routeKey":      created.RouteKey,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listAccessGrants(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListAccessGrants(r.Context(), scope)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) revokeAccessGrant(w http.ResponseWriter, r *http.Request) {
	grantID := chi.URLParam(r, "id")
	tenantID, workspaceID := "", ""
	foundForAudit := false
	grants, err := s.repo.ListAccessGrants(r.Context(), store.ManagementScope{})
	if err != nil {
		writeError(w, err)
		return
	}
	for _, existing := range grants {
		if existing.ID != grantID {
			continue
		}
		foundForAudit = true
		if caller, ok, err := s.repo.GetAgent(r.Context(), existing.CallerID); err != nil {
			writeError(w, err)
			return
		} else if ok {
			tenantID = caller.TenantID
			workspaceID = caller.WorkspaceID
		}
		break
	}
	if !foundForAudit {
		writeError(w, domain.NotFound("access grant not found"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: tenantID, WorkspaceID: workspaceID}); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	grant, ok, err := s.repo.RevokeAccessGrantWithAudit(r.Context(), grantID, now, func(revoked domain.AccessGrant) domain.AuditEvent {
		return s.managementAuditEvent(r, tenantID, workspaceID, "access_grant.revoked", "access_grant", revoked.ID, "Access grant revoked", map[string]any{
			"callerAgentId": revoked.CallerID,
			"targetAgentId": revoked.TargetID,
			"routeType":     revoked.RouteType,
			"routeKey":      revoked.RouteKey,
		})
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("access grant not found"))
		return
	}
	writeJSON(w, http.StatusOK, grant)
}

func (s *Server) createRoutePolicy(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateRoutePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CallerID = strings.TrimSpace(req.CallerID)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.RouteType = strings.TrimSpace(req.RouteType)
	req.RouteKey = strings.TrimSpace(req.RouteKey)
	if req.CallerID == "" || req.TargetID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "callerAgentId and targetAgentId are required"))
		return
	}
	if req.RouteType == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "routeType is required"))
		return
	}
	effect, err := normalizeRoutePolicyEffect(req.Effect, domain.RoutePolicyEffectAllow)
	if err != nil {
		writeError(w, err)
		return
	}
	status, err := normalizeRoutePolicyStatus(req.Status, domain.RoutePolicyStatusEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	priority, err := normalizeRoutePolicyPriority(req.Priority)
	if err != nil {
		writeError(w, err)
		return
	}
	retry, err := normalizeRoutePolicyRetry(req.Retry, "retry")
	if err != nil {
		writeError(w, err)
		return
	}
	caller, ok, err := s.repo.GetAgent(r.Context(), req.CallerID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("caller agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, caller); err != nil {
		writeError(w, err)
		return
	}
	target, ok, err := s.repo.GetAgent(r.Context(), req.TargetID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return
	}
	if caller.TenantID != target.TenantID || caller.WorkspaceID != target.WorkspaceID {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "caller and target agents must be in the same tenant and workspace for route policies"))
		return
	}
	if req.Name == "" {
		req.Name = defaultRoutePolicyName(req.RouteType, req.RouteKey, effect)
	}
	now := s.now()
	policy := domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    caller.TenantID,
		WorkspaceID: caller.WorkspaceID,
		Name:        req.Name,
		CallerID:    caller.ID,
		TargetID:    target.ID,
		RouteType:   req.RouteType,
		RouteKey:    req.RouteKey,
		Effect:      effect,
		Status:      status,
		Priority:    priority,
		Retry:       retry,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.rejectDuplicateRoutePolicy(r.Context(), policy); err != nil {
		writeError(w, err)
		return
	}
	created, err := s.repo.CreateRoutePolicyWithAudit(r.Context(), policy, func(created domain.RoutePolicy) domain.AuditEvent {
		return s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "route_policy.created", "route_policy", created.ID, "Route policy created", routePolicyAuditMetadata(created))
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listRoutePolicies(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListRoutePolicies(r.Context(), scope)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) updateRoutePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "id")
	existing, ok, err := s.repo.GetRoutePolicy(r.Context(), policyID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("route policy not found"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: existing.TenantID, WorkspaceID: existing.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	var req domain.UpdateRoutePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	policy := existing
	if req.Name != nil {
		policy.Name = strings.TrimSpace(*req.Name)
	}
	if req.RouteType != nil {
		policy.RouteType = strings.TrimSpace(*req.RouteType)
	}
	if req.RouteKey != nil {
		policy.RouteKey = strings.TrimSpace(*req.RouteKey)
	}
	if policy.RouteType == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "routeType is required"))
		return
	}
	if req.Effect != nil {
		effect, err := normalizeRoutePolicyEffect(*req.Effect, existing.Effect)
		if err != nil {
			writeError(w, err)
			return
		}
		policy.Effect = effect
	}
	if req.Status != nil {
		status, err := normalizeRoutePolicyStatus(*req.Status, existing.Status)
		if err != nil {
			writeError(w, err)
			return
		}
		policy.Status = status
	}
	if req.Priority != nil {
		if *req.Priority < 0 {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "priority must be zero or greater"))
			return
		}
		policy.Priority = *req.Priority
	}
	if req.Retry != nil {
		retry, err := routePolicyRetryFromPatch(req.Retry)
		if err != nil {
			writeError(w, err)
			return
		}
		policy.Retry = retry
	}
	if policy.Name == "" {
		policy.Name = defaultRoutePolicyName(policy.RouteType, policy.RouteKey, policy.Effect)
	}
	policy.UpdatedAt = s.now()
	updated, ok, err := s.repo.UpdateRoutePolicyWithAudit(r.Context(), policy, func(updated domain.RoutePolicy) domain.AuditEvent {
		return s.managementAuditEvent(r, updated.TenantID, updated.WorkspaceID, "route_policy.updated", "route_policy", updated.ID, "Route policy updated", routePolicyAuditMetadata(updated))
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("route policy not found"))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) disableRoutePolicy(w http.ResponseWriter, r *http.Request) {
	existing, ok, err := s.repo.GetRoutePolicy(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("route policy not found"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: existing.TenantID, WorkspaceID: existing.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	policy, ok, err := s.repo.DisableRoutePolicyWithAudit(r.Context(), existing.ID, now, func(disabled domain.RoutePolicy) domain.AuditEvent {
		return s.managementAuditEvent(r, disabled.TenantID, disabled.WorkspaceID, "route_policy.disabled", "route_policy", disabled.ID, "Route policy disabled", routePolicyAuditMetadata(disabled))
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("route policy not found"))
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (s *Server) refreshTargetCapabilities(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "targetId"))
	target, ok, err := s.repo.GetAgent(r.Context(), targetID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return
	}
	if err := s.requireAgentManagementScope(r, target); err != nil {
		writeError(w, err)
		return
	}
	if target.ChannelType != "mcp" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "capability refresh currently supports mcp targets only"))
		return
	}
	discovered, err := s.discoverMCPCapabilities(r.Context(), target)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, target.TenantID, target.WorkspaceID, "capabilities.refreshed", "agent", target.ID, "Capabilities refreshed", map[string]any{
		"targetId":        target.ID,
		"capabilityCount": len(discovered),
	})); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, discovered)
}

func (s *Server) listCapabilities(w http.ResponseWriter, r *http.Request) {
	status := domain.CapabilityDiscoveryStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validCapabilityDiscoveryStatus(status) {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "status must be pending_review, approved, deprecated, or removed"))
		return
	}
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListCapabilities(r.Context(), store.CapabilityFilter{
		ManagementScope: scope,
		TargetID:        strings.TrimSpace(r.URL.Query().Get("targetId")),
		Status:          status,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) updateCapability(w http.ResponseWriter, r *http.Request) {
	existing, ok, err := s.repo.GetCapability(r.Context(), strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("capability not found"))
		return
	}
	if err := s.requireCapabilityManagementScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	var req domain.UpdateCapabilityRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	updated := existing
	if req.DiscoveryStatus != nil {
		if !validCapabilityDiscoveryStatus(*req.DiscoveryStatus) {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "discoveryStatus must be pending_review, approved, deprecated, or removed"))
			return
		}
		updated.DiscoveryStatus = *req.DiscoveryStatus
	}
	if req.Sensitivity != nil {
		if !validCapabilitySensitivity(*req.Sensitivity) {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "sensitivity must be public, internal, confidential, or restricted"))
			return
		}
		updated.Sensitivity = *req.Sensitivity
	}
	if req.RiskLevel != nil {
		if !validCapabilityRisk(*req.RiskLevel) {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", "riskLevel must be low, medium, high, or critical"))
			return
		}
		updated.RiskLevel = *req.RiskLevel
	}
	if req.DataScopes != nil {
		updated.DataScopes = req.DataScopes
	}
	updated.UpdatedAt = s.now()
	saved, ok, err := s.repo.UpdateCapability(r.Context(), updated)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("capability not found"))
		return
	}
	if target, ok, err := s.repo.GetAgent(r.Context(), saved.TargetID); err != nil {
		writeError(w, err)
		return
	} else if ok {
		if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, target.TenantID, target.WorkspaceID, "capability.updated", "capability", saved.ID, "Capability updated", map[string]any{
			"targetId":        saved.TargetID,
			"capabilityKey":   saved.Key,
			"discoveryStatus": saved.DiscoveryStatus,
			"sensitivity":     saved.Sensitivity,
			"riskLevel":       saved.RiskLevel,
		})); err != nil {
			writeError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listPermissionPackageTemplates(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, permissionpack.Templates())
}

func (s *Server) listPermissionPackageAccessSubjects(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, permissionpack.AccessSubjects())
}

func (s *Server) listPermissionPackageApplications(w http.ResponseWriter, r *http.Request) {
	limit, err := auditLimitFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListPermissionPackageApplications(r.Context(), store.PermissionPackageApplicationFilter{
		ManagementScope:  scope,
		TemplateID:       strings.TrimSpace(r.URL.Query().Get("templateId")),
		TargetID:         strings.TrimSpace(r.URL.Query().Get("targetId")),
		CallerInstanceID: strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		Limit:            limit,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

type permissionPackageApplicationHealthResponse struct {
	Summary      permissionPackageApplicationHealthSummary `json:"summary"`
	Applications []permissionPackageApplicationHealthRow   `json:"applications"`
}

type permissionPackageApplicationHealthSummary struct {
	Total       int `json:"total"`
	Ready       int `json:"ready"`
	Drifted     int `json:"drifted"`
	NeedsReview int `json:"needsReview"`
}

type permissionPackageApplicationHealthRow struct {
	Application        domain.PermissionPackageApplication `json:"application"`
	Status             string                              `json:"status"`
	BlockerCodes       []string                            `json:"blockerCodes"`
	CreatedObjectCount int                                 `json:"createdObjectCount"`
	ActiveObjectCount  int                                 `json:"activeObjectCount"`
	MissingObjectCount int                                 `json:"missingObjectCount"`
	RollbackReady      bool                                `json:"rollbackReady"`
}

func (s *Server) listPermissionPackageApplicationHealth(w http.ResponseWriter, r *http.Request) {
	limit, err := auditLimitFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	applications, err := s.repo.ListPermissionPackageApplications(r.Context(), store.PermissionPackageApplicationFilter{
		ManagementScope:  scope,
		TemplateID:       strings.TrimSpace(r.URL.Query().Get("templateId")),
		TargetID:         strings.TrimSpace(r.URL.Query().Get("targetId")),
		CallerInstanceID: strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		Limit:            limit,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	response := permissionPackageApplicationHealthResponse{
		Applications: []permissionPackageApplicationHealthRow{},
	}
	for _, application := range applications {
		impact, err := s.permissionPackageApplicationImpact(r.Context(), application)
		if err != nil {
			writeError(w, err)
			return
		}
		status := permissionPackageApplicationHealthStatus(impact)
		switch status {
		case "ready":
			response.Summary.Ready++
		case "drifted":
			response.Summary.Drifted++
		default:
			response.Summary.NeedsReview++
		}
		response.Applications = append(response.Applications, permissionPackageApplicationHealthRow{
			Application:        application,
			Status:             status,
			BlockerCodes:       append([]string{}, impact.RollbackReview.BlockerCodes...),
			CreatedObjectCount: impact.Summary.CreatedObjectCount,
			ActiveObjectCount:  impact.Summary.ActiveObjectCount,
			MissingObjectCount: impact.Summary.MissingObjectCount,
			RollbackReady:      impact.Summary.RollbackReady,
		})
	}
	response.Summary.Total = len(response.Applications)
	writeJSON(w, http.StatusOK, response)
}

func permissionPackageApplicationHealthStatus(impact permissionPackageApplicationImpactResponse) string {
	if impact.RollbackReview.Ready {
		return "ready"
	}
	if permissionPackageApplicationBlockerCodesContain(impact.RollbackReview.BlockerCodes, "missing_created_objects") ||
		permissionPackageApplicationBlockerCodesContain(impact.RollbackReview.BlockerCodes, "inactive_created_objects") {
		return "drifted"
	}
	return "needs_review"
}

func permissionPackageApplicationBlockerCodesContain(blockerCodes []string, want string) bool {
	for _, blockerCode := range blockerCodes {
		if blockerCode == want {
			return true
		}
	}
	return false
}

type permissionPackageProductionReadinessQuery struct {
	TenantID          string
	WorkspaceID       string
	TemplateID        string
	TargetID          string
	CallerInstanceID  string
	SubjectID         string
	Region            string
	RequestText       string
	SubjectSelector   string
	ApprovalRequestID string
	TraceLimit        int
}

type permissionPackageWorkbenchPreviewResponse struct {
	Draft               domain.PermissionPackageDraft                 `json:"draft"`
	ApprovalRequest     *domain.PermissionPackageApprovalRequest      `json:"approvalRequest,omitempty"`
	LatestApplication   *domain.PermissionPackageApplication          `json:"latestApplication,omitempty"`
	ProductionReadiness *permissionPackageProductionReadinessResponse `json:"productionReadiness,omitempty"`
	Summary             permissionPackageWorkbenchSummary             `json:"summary"`
	GeneratedAt         time.Time                                     `json:"generatedAt"`
}

type permissionPackageWorkbenchSummary struct {
	Status                 string                           `json:"status"`
	PrimaryActionCode      string                           `json:"primaryActionCode"`
	NextActionCode         string                           `json:"nextActionCode,omitempty"`
	ApprovalRequired       bool                             `json:"approvalRequired"`
	CanApply               bool                             `json:"canApply"`
	Applied                bool                             `json:"applied"`
	RuntimeEvidenceReady   bool                             `json:"runtimeEvidenceReady"`
	ProductionReady        bool                             `json:"productionReady"`
	AllowedCapabilityCount int                              `json:"allowedCapabilityCount"`
	BlockedCapabilityCount int                              `json:"blockedCapabilityCount"`
	PlannedObjectCount     int                              `json:"plannedObjectCount"`
	ReadinessReadyCount    int                              `json:"readinessReadyCount"`
	ReadinessTotalCount    int                              `json:"readinessTotalCount"`
	BlockingCount          int                              `json:"blockingCount"`
	WarningCount           int                              `json:"warningCount"`
	Steps                  []permissionPackageWorkbenchStep `json:"steps"`
}

type permissionPackageWorkbenchStep struct {
	Key        string `json:"key"`
	Status     string `json:"status"`
	DetailCode string `json:"detailCode"`
	Count      int    `json:"count,omitempty"`
	Total      int    `json:"total,omitempty"`
}

type permissionPackageProductionReadinessResponse struct {
	Status            string                                          `json:"status"`
	Summary           permissionPackageProductionReadinessSummary     `json:"summary"`
	Checks            []permissionPackageProductionReadinessCheck     `json:"checks"`
	LatestApplication *domain.PermissionPackageApplication            `json:"latestApplication,omitempty"`
	Preflight         *domain.PermissionPackageApplyPreflightResponse `json:"preflight,omitempty"`
	ApplicationHealth *permissionPackageApplicationHealthRow          `json:"applicationHealth,omitempty"`
	ApplicationImpact *permissionPackageApplicationImpactResponse     `json:"applicationImpact,omitempty"`
	AccessProfile     *tenantAccessProfileResponse                    `json:"accessProfile,omitempty"`
	RuntimeEvidence   permissionPackageRuntimeEvidence                `json:"runtimeEvidence"`
	AuditEvidence     permissionPackageAuditEvidence                  `json:"auditEvidence"`
	NextActionCode    string                                          `json:"nextActionCode"`
	NextActions       []string                                        `json:"nextActions"`
	GeneratedAt       time.Time                                       `json:"generatedAt"`
}

type permissionPackageProductionReadinessSummary struct {
	ReadyCount         int  `json:"readyCount"`
	WarningCount       int  `json:"warningCount"`
	BlockingCount      int  `json:"blockingCount"`
	HasApplication     bool `json:"hasApplication"`
	HasAllowedTrace    bool `json:"hasAllowedTrace"`
	HasDeniedTrace     bool `json:"hasDeniedTrace"`
	HasAppliedAudit    bool `json:"hasAppliedAudit"`
	AccessProfileReady bool `json:"accessProfileReady"`
}

type permissionPackageProductionReadinessCheck struct {
	Code       string                                    `json:"code"`
	Severity   domain.PermissionPackagePreflightSeverity `json:"severity"`
	Message    string                                    `json:"message"`
	EvidenceID string                                    `json:"evidenceId,omitempty"`
}

type permissionPackageRuntimeEvidence struct {
	AllowedTrace *domain.TraceEvent `json:"allowedTrace,omitempty"`
	DeniedTrace  *domain.TraceEvent `json:"deniedTrace,omitempty"`
}

type permissionPackageAuditEvidence struct {
	AppliedEvent *domain.AuditEvent `json:"appliedEvent,omitempty"`
}

const permissionPackageProductionEvidenceReportVersion = "production-readiness-report/v1"

type permissionPackageProductionEvidenceReportResponse struct {
	ReportVersion        string                                      `json:"reportVersion"`
	GeneratedAt          time.Time                                   `json:"generatedAt"`
	Scope                permissionPackageProductionEvidenceScope    `json:"scope"`
	Status               string                                      `json:"status"`
	Summary              permissionPackageProductionReadinessSummary `json:"summary"`
	Checks               []permissionPackageProductionReadinessCheck `json:"checks"`
	Evidence             permissionPackageProductionEvidenceRefs     `json:"evidence"`
	NextActionCode       string                                      `json:"nextActionCode"`
	NextActions          []string                                    `json:"nextActions"`
	ReadinessGeneratedAt time.Time                                   `json:"readinessGeneratedAt"`
}

type permissionPackageProductionEvidenceScope struct {
	TenantID         string `json:"tenantId"`
	WorkspaceID      string `json:"workspaceId"`
	TemplateID       string `json:"templateId"`
	TargetID         string `json:"targetId"`
	CallerInstanceID string `json:"callerInstanceId"`
	SubjectID        string `json:"subjectId,omitempty"`
	Region           string `json:"region,omitempty"`
	SubjectSelector  string `json:"subjectSelector,omitempty"`
}

type permissionPackageProductionEvidenceRefs struct {
	Application       permissionPackageProductionApplicationEvidence `json:"application"`
	Runtime           permissionPackageProductionRuntimeEvidence     `json:"runtime"`
	Audit             permissionPackageProductionAuditEvidence       `json:"audit"`
	AccessProfile     permissionPackageProductionEvidenceState       `json:"accessProfile"`
	ApplicationHealth permissionPackageProductionEvidenceState       `json:"applicationHealth"`
	ApplicationImpact permissionPackageProductionEvidenceState       `json:"applicationImpact"`
}

type permissionPackageProductionApplicationEvidence struct {
	Present               bool               `json:"present"`
	ID                    string             `json:"id,omitempty"`
	DraftID               string             `json:"draftId,omitempty"`
	TemplateVersion       int                `json:"templateVersion,omitempty"`
	AppliedAt             *time.Time         `json:"appliedAt,omitempty"`
	AllowedCapabilityIDs  []string           `json:"allowedCapabilityIds,omitempty"`
	AllowedCapabilityKeys []string           `json:"allowedCapabilityKeys,omitempty"`
	DataScopes            []domain.DataScope `json:"dataScopes,omitempty"`
}

type permissionPackageProductionRuntimeEvidence struct {
	AllowedTraceID string `json:"allowedTraceId,omitempty"`
	DeniedTraceID  string `json:"deniedTraceId,omitempty"`
}

type permissionPackageProductionAuditEvidence struct {
	AppliedEventID string `json:"appliedEventId,omitempty"`
}

type permissionPackageProductionEvidenceState struct {
	Present bool   `json:"present"`
	Status  string `json:"status,omitempty"`
}

func (s *Server) previewPermissionPackageWorkbench(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageDraftScope(r, req.PermissionPackageDraftRequest); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.permissionPackageWorkbenchPreview(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getPermissionPackageProductionReadiness(w http.ResponseWriter, r *http.Request) {
	query, err := permissionPackageProductionReadinessQueryFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageQueryScope(r, query); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.permissionPackageProductionReadiness(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getPermissionPackageProductionEvidenceReport(w http.ResponseWriter, r *http.Request) {
	query, err := permissionPackageProductionReadinessQueryFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageQueryScope(r, query); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.permissionPackageProductionEvidenceReport(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func permissionPackageProductionReadinessQueryFromRequest(r *http.Request) (permissionPackageProductionReadinessQuery, error) {
	values := r.URL.Query()
	query := permissionPackageProductionReadinessQuery{
		TenantID:          strings.TrimSpace(values.Get("tenantId")),
		WorkspaceID:       strings.TrimSpace(values.Get("workspaceId")),
		TemplateID:        strings.TrimSpace(values.Get("templateId")),
		TargetID:          strings.TrimSpace(values.Get("targetId")),
		CallerInstanceID:  strings.TrimSpace(values.Get("callerInstanceId")),
		SubjectID:         strings.TrimSpace(values.Get("subjectId")),
		Region:            strings.TrimSpace(values.Get("region")),
		RequestText:       strings.TrimSpace(values.Get("requestText")),
		SubjectSelector:   strings.TrimSpace(values.Get("subjectSelector")),
		ApprovalRequestID: strings.TrimSpace(values.Get("approvalRequestId")),
		TraceLimit:        defaultAccessProfileTraceLimit,
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{name: "tenantId", value: query.TenantID},
		{name: "workspaceId", value: query.WorkspaceID},
		{name: "templateId", value: query.TemplateID},
		{name: "targetId", value: query.TargetID},
		{name: "callerInstanceId", value: query.CallerInstanceID},
	} {
		if required.value == "" {
			return permissionPackageProductionReadinessQuery{}, domain.BadRequest("VALIDATION_FAILED", required.name+" is required")
		}
	}
	if raw := strings.TrimSpace(values.Get("traceLimit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 || limit > maxAccessProfileTraceLimit {
			return permissionPackageProductionReadinessQuery{}, domain.BadRequest("VALIDATION_FAILED", "traceLimit must be between 0 and 100")
		}
		query.TraceLimit = limit
	}
	return query, nil
}

func (s *Server) permissionPackageWorkbenchPreview(ctx context.Context, req domain.PermissionPackageApplyRequest) (permissionPackageWorkbenchPreviewResponse, error) {
	req.ApprovalRequestID = strings.TrimSpace(req.ApprovalRequestID)
	draft, err := s.buildPermissionPackageDraft(ctx, req.PermissionPackageDraftRequest)
	if err != nil {
		return permissionPackageWorkbenchPreviewResponse{}, err
	}
	approval, err := s.permissionPackageWorkbenchApprovalRequest(ctx, req.ApprovalRequestID, draft)
	if err != nil {
		return permissionPackageWorkbenchPreviewResponse{}, err
	}
	if approval != nil && approval.Status == domain.PermissionPackageApprovalStatusApproved {
		req.ApprovalRequestID = approval.ID
	}

	var readiness *permissionPackageProductionReadinessResponse
	if permissionPackageWorkbenchShouldCheckReadiness(draft, approval) {
		next, err := s.permissionPackageProductionReadiness(ctx, permissionPackageProductionReadinessQueryFromApplyRequest(req))
		if err != nil {
			return permissionPackageWorkbenchPreviewResponse{}, err
		}
		readiness = &next
	}

	return permissionPackageWorkbenchPreviewResponse{
		Draft:               draft,
		ApprovalRequest:     approval,
		LatestApplication:   permissionPackageWorkbenchLatestApplication(readiness),
		ProductionReadiness: readiness,
		Summary:             permissionPackageWorkbenchSummaryFor(draft, approval, readiness),
		GeneratedAt:         s.now(),
	}, nil
}

func permissionPackageWorkbenchShouldCheckReadiness(draft domain.PermissionPackageDraft, approval *domain.PermissionPackageApprovalRequest) bool {
	if !permissionPackageWorkbenchCanCheckReadiness(draft) {
		return false
	}
	if approval == nil {
		return true
	}
	return approval.Status == domain.PermissionPackageApprovalStatusApproved
}

func permissionPackageProductionReadinessQueryFromApplyRequest(req domain.PermissionPackageApplyRequest) permissionPackageProductionReadinessQuery {
	input := trimPermissionPackageDraftRequest(req.PermissionPackageDraftRequest)
	return permissionPackageProductionReadinessQuery{
		TenantID:          input.TenantID,
		WorkspaceID:       input.WorkspaceID,
		TemplateID:        input.TemplateID,
		TargetID:          input.TargetID,
		CallerInstanceID:  input.CallerInstanceID,
		Region:            input.Region,
		RequestText:       input.RequestText,
		SubjectSelector:   input.SubjectSelector,
		ApprovalRequestID: strings.TrimSpace(req.ApprovalRequestID),
		TraceLimit:        defaultAccessProfileTraceLimit,
	}
}

func permissionPackageWorkbenchCanCheckReadiness(draft domain.PermissionPackageDraft) bool {
	input := draft.Input
	return input.TenantID != "" &&
		input.WorkspaceID != "" &&
		input.TemplateID != "" &&
		input.TargetID != "" &&
		input.CallerInstanceID != ""
}

func (s *Server) permissionPackageWorkbenchApprovalRequest(ctx context.Context, approvalRequestID string, draft domain.PermissionPackageDraft) (*domain.PermissionPackageApprovalRequest, error) {
	if draft.PolicyGate.CanApplyDirectly || !draft.Readiness.CanApply {
		return nil, nil
	}
	if approvalRequestID != "" {
		approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(ctx, approvalRequestID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.NotFound("approval request not found")
		}
		if !permissionPackageApprovalRequestMatchesDraftSnapshot(approval, draft) {
			return nil, domain.BadRequest("VALIDATION_FAILED", "approval request does not match current permission request")
		}
		return &approval, nil
	}
	rows, err := s.repo.ListPermissionPackageApprovalRequests(ctx, store.PermissionPackageApprovalRequestFilter{
		ManagementScope:  store.ManagementScope{TenantID: draft.Input.TenantID, WorkspaceID: draft.Input.WorkspaceID},
		TemplateID:       draft.Template.ID,
		TargetID:         draft.Input.TargetID,
		CallerInstanceID: draft.Input.CallerInstanceID,
		Limit:            20,
	})
	if err != nil {
		return nil, err
	}
	sortPermissionPackageApprovalRequests(rows)
	var selected *domain.PermissionPackageApprovalRequest
	for _, row := range rows {
		if !permissionPackageApprovalRequestMatchesDraftSnapshot(row, draft) {
			continue
		}
		if row.Status == domain.PermissionPackageApprovalStatusApproved &&
			validatePermissionPackageApprovalForDraft(row, draft, s.now()) != nil {
			continue
		}
		candidate := row
		if selected == nil || permissionPackageApprovalPreviewRank(candidate) > permissionPackageApprovalPreviewRank(*selected) {
			selected = &candidate
		}
	}
	return selected, nil
}

func permissionPackageApprovalPreviewRank(request domain.PermissionPackageApprovalRequest) int {
	switch request.Status {
	case domain.PermissionPackageApprovalStatusApproved:
		return 3
	case domain.PermissionPackageApprovalStatusPending:
		return 2
	case domain.PermissionPackageApprovalStatusRejected:
		return 1
	case domain.PermissionPackageApprovalStatusWithdrawn:
		return 1
	default:
		return 0
	}
}

func permissionPackageApprovalRequestMatchesDraftSnapshot(approval domain.PermissionPackageApprovalRequest, draft domain.PermissionPackageDraft) bool {
	allowedCapabilityIDs, allowedCapabilityKeys := permissionPackageCapabilityIDsAndKeys(draft.AllowedCapabilities)
	allowedCapabilityFingerprints := permissionPackageCapabilityFingerprints(draft.AllowedCapabilities)
	return approval.DraftID == draft.ID &&
		approval.TemplateID == draft.Template.ID &&
		approval.TemplateVersion == draft.Template.Version &&
		approval.PolicyVersion == draft.PolicyGate.PolicyVersion &&
		approval.TenantID == draft.Input.TenantID &&
		approval.WorkspaceID == draft.Input.WorkspaceID &&
		approval.TargetID == draft.Input.TargetID &&
		approval.CallerInstanceID == draft.Input.CallerInstanceID &&
		approval.SubjectSelector == draft.Input.SubjectSelector &&
		approval.RequestText == draft.Input.RequestText &&
		approval.Region == draft.Input.Region &&
		samePermissionPackageDataScopes(approval.DataScopes, draft.DataScopes) &&
		sameStringSet(approval.AllowedCapabilityIDs, allowedCapabilityIDs) &&
		sameStringSet(approval.AllowedCapabilityKeys, allowedCapabilityKeys) &&
		sameStringSet(approval.AllowedCapabilityFingerprints, allowedCapabilityFingerprints)
}

func permissionPackageWorkbenchLatestApplication(readiness *permissionPackageProductionReadinessResponse) *domain.PermissionPackageApplication {
	if readiness == nil {
		return nil
	}
	return readiness.LatestApplication
}

func permissionPackageWorkbenchSummaryFor(draft domain.PermissionPackageDraft, approval *domain.PermissionPackageApprovalRequest, readiness *permissionPackageProductionReadinessResponse) permissionPackageWorkbenchSummary {
	approvalRequired := !draft.PolicyGate.CanApplyDirectly
	approvalResolvedWithoutApproval := approvalRequired && approval != nil &&
		(approval.Status == domain.PermissionPackageApprovalStatusRejected || approval.Status == domain.PermissionPackageApprovalStatusWithdrawn)
	applied := readiness != nil && readiness.LatestApplication != nil
	approvalApproved := !approvalRequired || applied || approval != nil && approval.Status == domain.PermissionPackageApprovalStatusApproved
	runtimeEvidenceReady := readiness != nil && readiness.Summary.HasAllowedTrace && readiness.Summary.HasDeniedTrace
	productionReady := readiness != nil && readiness.Status == "ready"
	canApply := draft.Readiness.CanApply && approvalApproved && !approvalResolvedWithoutApproval
	summary := permissionPackageWorkbenchSummary{
		ApprovalRequired:       approvalRequired,
		CanApply:               canApply,
		Applied:                applied,
		RuntimeEvidenceReady:   runtimeEvidenceReady,
		ProductionReady:        productionReady,
		AllowedCapabilityCount: len(draft.AllowedCapabilities),
		BlockedCapabilityCount: len(draft.BlockedCapabilities),
	}
	if readiness != nil {
		summary.NextActionCode = readiness.NextActionCode
		summary.ReadinessReadyCount = readiness.Summary.ReadyCount
		summary.ReadinessTotalCount = len(readiness.Checks)
		summary.BlockingCount = readiness.Summary.BlockingCount
		summary.WarningCount = readiness.Summary.WarningCount
		if readiness.Preflight != nil {
			summary.PlannedObjectCount = readiness.Preflight.Summary.PlannedTenantEntitlementCount +
				readiness.Preflight.Summary.PlannedWorkspaceAssignmentCount +
				readiness.Preflight.Summary.PlannedInstanceAssignmentCount
		}
	}
	if summary.PlannedObjectCount == 0 {
		summary.PlannedObjectCount = len(draft.AllowedCapabilities) * 3
	}
	summary.Status, summary.PrimaryActionCode = permissionPackageWorkbenchStatusAndAction(draft, approval, applied, productionReady, canApply)
	summary.Steps = permissionPackageWorkbenchSteps(draft, approval, applied, runtimeEvidenceReady, productionReady, canApply)
	return summary
}

func permissionPackageWorkbenchStatusAndAction(draft domain.PermissionPackageDraft, approval *domain.PermissionPackageApprovalRequest, applied bool, productionReady bool, canApply bool) (string, string) {
	approvalRequired := !draft.PolicyGate.CanApplyDirectly
	if !draft.Readiness.CanApply {
		return "needs_input", "complete_request"
	}
	if productionReady {
		return "production_ready", "export_production_evidence"
	}
	if applied {
		return "validating", "run_runtime_validation"
	}
	if approvalRequired {
		if approval == nil ||
			approval.Status == domain.PermissionPackageApprovalStatusRejected ||
			approval.Status == domain.PermissionPackageApprovalStatusWithdrawn {
			return "awaiting_approval", "create_approval_request"
		}
		if approval.Status == domain.PermissionPackageApprovalStatusPending {
			return "awaiting_approval", "review_approval_request"
		}
	}
	if !applied && canApply {
		return "ready_to_apply", "apply_permission_package"
	}
	return "blocked", "complete_request"
}

func permissionPackageWorkbenchSteps(draft domain.PermissionPackageDraft, approval *domain.PermissionPackageApprovalRequest, applied bool, runtimeEvidenceReady bool, productionReady bool, canApply bool) []permissionPackageWorkbenchStep {
	approvalRequired := !draft.PolicyGate.CanApplyDirectly
	approvalComplete := !approvalRequired || applied || approval != nil && approval.Status == domain.PermissionPackageApprovalStatusApproved
	requestStatus := "complete"
	requestDetail := "request_ready"
	if !draft.Readiness.CanApply {
		requestStatus = "current"
		requestDetail = "request_needs_input"
	}
	approvalStatus := "waiting"
	approvalDetail := "approval_waiting"
	if approvalComplete {
		approvalStatus = "complete"
		if approvalRequired {
			approvalDetail = "approval_approved"
		} else {
			approvalDetail = "approval_not_required"
		}
	} else if draft.Readiness.CanApply {
		approvalStatus = "current"
		approvalDetail = "approval_required"
		if approval != nil && approval.Status == domain.PermissionPackageApprovalStatusPending {
			approvalDetail = "approval_pending"
		}
		if approval != nil && approval.Status == domain.PermissionPackageApprovalStatusRejected {
			approvalStatus = "blocked"
			approvalDetail = "approval_rejected"
		}
		if approval != nil && approval.Status == domain.PermissionPackageApprovalStatusWithdrawn {
			approvalDetail = "approval_withdrawn"
		}
	}
	applyStatus := "waiting"
	applyDetail := "apply_waiting"
	if applied {
		applyStatus = "complete"
		applyDetail = "apply_done"
	} else if canApply {
		applyStatus = "current"
		applyDetail = "apply_ready"
	}
	validationStatus := "waiting"
	validationDetail := "validation_waiting"
	if runtimeEvidenceReady {
		validationStatus = "complete"
		validationDetail = "validation_ready"
	} else if applied {
		validationStatus = "current"
		validationDetail = "validation_needed"
	}
	acceptanceStatus := "waiting"
	acceptanceDetail := "acceptance_waiting"
	if productionReady {
		acceptanceStatus = "complete"
		acceptanceDetail = "acceptance_ready"
	} else if applied {
		acceptanceStatus = "current"
		acceptanceDetail = "acceptance_needed"
	}
	return []permissionPackageWorkbenchStep{
		{Key: "request", Status: requestStatus, DetailCode: requestDetail},
		{Key: "approval", Status: approvalStatus, DetailCode: approvalDetail},
		{Key: "apply", Status: applyStatus, DetailCode: applyDetail},
		{Key: "validation", Status: validationStatus, DetailCode: validationDetail},
		{Key: "acceptance", Status: acceptanceStatus, DetailCode: acceptanceDetail},
	}
}

func (s *Server) permissionPackageProductionReadiness(ctx context.Context, query permissionPackageProductionReadinessQuery) (permissionPackageProductionReadinessResponse, error) {
	result := permissionPackageProductionReadinessResponse{
		Checks:      []permissionPackageProductionReadinessCheck{},
		NextActions: []string{},
		GeneratedAt: s.now(),
	}
	applications, err := s.repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
		ManagementScope:  store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID},
		TemplateID:       query.TemplateID,
		TargetID:         query.TargetID,
		CallerInstanceID: query.CallerInstanceID,
		Limit:            1,
	})
	if err != nil {
		return permissionPackageProductionReadinessResponse{}, err
	}
	if len(applications) > 0 {
		latest := applications[0]
		result.LatestApplication = &latest
		query = permissionPackageProductionReadinessQueryWithApplicationDefaults(query, latest)
	}

	preflight, err := s.preflightPermissionPackageRequest(ctx, domain.PermissionPackageApplyRequest{
		PermissionPackageDraftRequest: domain.PermissionPackageDraftRequest{
			CallerInstanceID: query.CallerInstanceID,
			Region:           query.Region,
			RequestText:      query.RequestText,
			SubjectSelector:  query.SubjectSelector,
			TargetID:         query.TargetID,
			TemplateID:       query.TemplateID,
			TenantID:         query.TenantID,
			WorkspaceID:      query.WorkspaceID,
		},
		ApprovalRequestID: query.ApprovalRequestID,
	})
	if err != nil {
		return permissionPackageProductionReadinessResponse{}, err
	}
	result.Preflight = &preflight
	if permissionPackageProductionPreflightReady(preflight, result.LatestApplication) {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("preflight_ready", domain.PermissionPackagePreflightPassed, "Permission package draft and safety preflight are acceptable for production readiness.", ""))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("preflight_ready", domain.PermissionPackagePreflightBlocking, "Permission package preflight still has blocking checks.", ""))
		permissionPackageProductionAddNextAction(&result, "resolve_preflight_blockers", "Resolve apply preflight blockers before claiming production readiness.")
	}

	if result.LatestApplication == nil {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_present", domain.PermissionPackagePreflightBlocking, "No permission package application exists for this tenant, workspace, template, target, and caller.", ""))
		permissionPackageProductionAddNextAction(&result, "apply_permission_package", "Apply the approved permission package before production readiness.")
	} else {
		result.Summary.HasApplication = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_present", domain.PermissionPackagePreflightPassed, "Permission package application record is present.", result.LatestApplication.ID))
		if permissionPackageProductionApplicationScopeMatches(query, *result.LatestApplication) {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_scope_match", domain.PermissionPackagePreflightPassed, "Latest application matches the requested production scope.", result.LatestApplication.ID))
		} else {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_scope_match", domain.PermissionPackagePreflightBlocking, "Latest application does not match the requested production scope.", result.LatestApplication.ID))
			permissionPackageProductionAddNextAction(&result, "review_application_scope", "Inspect the latest permission package application scope before go-live.")
		}
		impact, err := s.permissionPackageApplicationImpact(ctx, *result.LatestApplication)
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		result.ApplicationImpact = &impact
		healthStatus := permissionPackageApplicationHealthStatus(impact)
		health := permissionPackageApplicationHealthRow{
			Application:        *result.LatestApplication,
			Status:             healthStatus,
			BlockerCodes:       append([]string{}, impact.RollbackReview.BlockerCodes...),
			CreatedObjectCount: impact.Summary.CreatedObjectCount,
			ActiveObjectCount:  impact.Summary.ActiveObjectCount,
			MissingObjectCount: impact.Summary.MissingObjectCount,
			RollbackReady:      impact.Summary.RollbackReady,
		}
		result.ApplicationHealth = &health
		if healthStatus == "ready" {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_health_ready", domain.PermissionPackagePreflightPassed, "Latest permission package application is healthy.", result.LatestApplication.ID))
		} else {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("application_health_ready", domain.PermissionPackagePreflightBlocking, "Latest permission package application is not healthy.", result.LatestApplication.ID))
			permissionPackageProductionAddNextAction(&result, "review_application_health", "Review application health and drift blockers before production readiness.")
		}
		if impact.RollbackReview.Ready && impact.Summary.MissingObjectCount == 0 {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("impact_ready", domain.PermissionPackagePreflightPassed, "Application impact review shows active created grant objects.", result.LatestApplication.ID))
		} else {
			result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("impact_ready", domain.PermissionPackagePreflightBlocking, "Application impact review has missing or inactive created objects.", result.LatestApplication.ID))
			permissionPackageProductionAddNextAction(&result, "resolve_impact_blockers", "Resolve impact review blockers before production readiness.")
		}
	}

	profile, err := s.buildTenantAccessProfile(ctx, query.TenantID, accessProfileQuery{
		WorkspaceID:      query.WorkspaceID,
		TargetID:         query.TargetID,
		CallerInstanceID: query.CallerInstanceID,
		TraceLimit:       query.TraceLimit,
	})
	if err != nil {
		return permissionPackageProductionReadinessResponse{}, err
	}
	result.AccessProfile = &profile
	accessEvidenceID := permissionPackageProductionAccessProfileEvidenceID(profile, query, result.LatestApplication)
	if accessEvidenceID != "" {
		result.Summary.AccessProfileReady = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("access_profile_chain_present", domain.PermissionPackagePreflightPassed, "Tenant access profile contains an effective target, workspace, and caller grant chain.", accessEvidenceID))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("access_profile_chain_present", domain.PermissionPackagePreflightBlocking, "Tenant access profile does not contain an effective grant chain for this caller and target.", ""))
		permissionPackageProductionAddNextAction(&result, "verify_access_profile", "Verify tenant entitlement, workspace assignment, and caller assignment records.")
	}

	traces := []domain.TraceEvent{}
	if query.TraceLimit > 0 {
		traces, err = s.repo.ListTraces(ctx, store.TraceFilter{
			ManagementScope: store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID},
			CallerID:        query.CallerInstanceID,
			TargetID:        query.TargetID,
			Limit:           query.TraceLimit,
		})
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
	}
	result.RuntimeEvidence.AllowedTrace = permissionPackageProductionLatestTrace(traces, domain.TraceDecisionAllowed, query.SubjectID)
	result.RuntimeEvidence.DeniedTrace = permissionPackageProductionLatestTrace(traces, domain.TraceDecisionDenied, query.SubjectID)
	if result.RuntimeEvidence.AllowedTrace != nil {
		result.Summary.HasAllowedTrace = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_allowed_trace_present", domain.PermissionPackagePreflightPassed, "Runtime allowed record is present for this caller and target.", result.RuntimeEvidence.AllowedTrace.ID))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_allowed_trace_present", domain.PermissionPackagePreflightBlocking, "Runtime allowed record is missing for this caller and target.", ""))
		permissionPackageProductionAddNextAction(&result, "run_allowed_runtime_call", "Run an allowed MCP call with the production subject before go-live.")
	}
	if result.RuntimeEvidence.DeniedTrace != nil {
		result.Summary.HasDeniedTrace = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_denied_trace_present", domain.PermissionPackagePreflightPassed, "Runtime denied record is present for this caller and target.", result.RuntimeEvidence.DeniedTrace.ID))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("runtime_denied_trace_present", domain.PermissionPackagePreflightBlocking, "Runtime denied record is missing for this caller and target.", ""))
		permissionPackageProductionAddNextAction(&result, "run_denied_runtime_call", "Run a denied MCP call that proves blocked tools stay blocked.")
	}

	if result.LatestApplication != nil {
		events, err := s.repo.ListAuditEvents(ctx, store.AuditEventFilter{
			ManagementScope: store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID},
			Action:          "permission_package.applied",
			ResourceID:      result.LatestApplication.ID,
			Limit:           1,
		})
		if err != nil {
			return permissionPackageProductionReadinessResponse{}, err
		}
		if len(events) > 0 {
			appliedEvent := events[0]
			result.AuditEvidence.AppliedEvent = &appliedEvent
		}
	}
	if result.AuditEvidence.AppliedEvent != nil {
		result.Summary.HasAppliedAudit = true
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("applied_audit_event_present", domain.PermissionPackagePreflightPassed, "Applied audit event is present for this permission package application.", result.AuditEvidence.AppliedEvent.ID))
	} else {
		result.Checks = append(result.Checks, permissionPackageProductionReadinessCheckFor("applied_audit_event_present", domain.PermissionPackagePreflightBlocking, "Applied audit event is missing for this permission package application.", ""))
		permissionPackageProductionAddNextAction(&result, "verify_applied_audit", "Verify the permission package applied audit record before production readiness.")
	}

	result.Summary = permissionPackageProductionReadinessSummaryFor(result)
	result.Status = permissionPackageProductionReadinessStatus(result.Summary)
	if result.Status == "ready" {
		permissionPackageProductionAddNextAction(&result, "export_production_evidence", "Production readiness is complete.")
	}
	return result, nil
}

func (s *Server) permissionPackageProductionEvidenceReport(ctx context.Context, query permissionPackageProductionReadinessQuery) (permissionPackageProductionEvidenceReportResponse, error) {
	readiness, err := s.permissionPackageProductionReadiness(ctx, query)
	if err != nil {
		return permissionPackageProductionEvidenceReportResponse{}, err
	}
	return permissionPackageProductionEvidenceReportFromReadiness(query, readiness), nil
}

func permissionPackageProductionEvidenceReportFromReadiness(query permissionPackageProductionReadinessQuery, readiness permissionPackageProductionReadinessResponse) permissionPackageProductionEvidenceReportResponse {
	scope := permissionPackageProductionEvidenceScope{
		TenantID:         query.TenantID,
		WorkspaceID:      query.WorkspaceID,
		TemplateID:       query.TemplateID,
		TargetID:         query.TargetID,
		CallerInstanceID: query.CallerInstanceID,
		SubjectID:        query.SubjectID,
		Region:           query.Region,
		SubjectSelector:  query.SubjectSelector,
	}
	evidence := permissionPackageProductionEvidenceRefs{
		AccessProfile:     permissionPackageProductionEvidenceState{Present: readiness.Summary.AccessProfileReady},
		ApplicationHealth: permissionPackageProductionEvidenceState{Present: readiness.ApplicationHealth != nil},
		ApplicationImpact: permissionPackageProductionEvidenceState{Present: readiness.ApplicationImpact != nil},
	}
	if readiness.ApplicationHealth != nil {
		evidence.ApplicationHealth.Status = readiness.ApplicationHealth.Status
	}
	if readiness.ApplicationImpact != nil && readiness.ApplicationImpact.Summary.RollbackReady {
		evidence.ApplicationImpact.Status = "ready"
	} else if readiness.ApplicationImpact != nil {
		evidence.ApplicationImpact.Status = "blocked"
	}
	if readiness.LatestApplication != nil {
		application := readiness.LatestApplication
		scope.Region = stringOrDefault(scope.Region, application.Region)
		scope.SubjectSelector = stringOrDefault(scope.SubjectSelector, application.SubjectSelector)
		appliedAt := application.AppliedAt
		evidence.Application = permissionPackageProductionApplicationEvidence{
			Present:               true,
			ID:                    application.ID,
			DraftID:               application.DraftID,
			TemplateVersion:       application.TemplateVersion,
			AppliedAt:             &appliedAt,
			AllowedCapabilityIDs:  append([]string(nil), application.AllowedCapabilityIDs...),
			AllowedCapabilityKeys: append([]string(nil), application.AllowedCapabilityKeys...),
			DataScopes:            domain.CloneDataScopes(application.DataScopes),
		}
	}
	if readiness.RuntimeEvidence.AllowedTrace != nil {
		evidence.Runtime.AllowedTraceID = readiness.RuntimeEvidence.AllowedTrace.ID
	}
	if readiness.RuntimeEvidence.DeniedTrace != nil {
		evidence.Runtime.DeniedTraceID = readiness.RuntimeEvidence.DeniedTrace.ID
	}
	if readiness.AuditEvidence.AppliedEvent != nil {
		evidence.Audit.AppliedEventID = readiness.AuditEvidence.AppliedEvent.ID
	}
	return permissionPackageProductionEvidenceReportResponse{
		ReportVersion:        permissionPackageProductionEvidenceReportVersion,
		GeneratedAt:          readiness.GeneratedAt,
		Scope:                scope,
		Status:               readiness.Status,
		Summary:              readiness.Summary,
		Checks:               append([]permissionPackageProductionReadinessCheck(nil), readiness.Checks...),
		Evidence:             evidence,
		NextActionCode:       readiness.NextActionCode,
		NextActions:          append([]string(nil), readiness.NextActions...),
		ReadinessGeneratedAt: readiness.GeneratedAt,
	}
}

func stringOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func permissionPackageProductionReadinessQueryWithApplicationDefaults(query permissionPackageProductionReadinessQuery, application domain.PermissionPackageApplication) permissionPackageProductionReadinessQuery {
	if query.Region == "" {
		query.Region = application.Region
	}
	if query.RequestText == "" {
		query.RequestText = application.RequestText
	}
	if query.SubjectSelector == "" {
		query.SubjectSelector = application.SubjectSelector
	}
	return query
}

func permissionPackageProductionPreflightReady(preflight domain.PermissionPackageApplyPreflightResponse, latestApplication *domain.PermissionPackageApplication) bool {
	if preflight.Summary.CanApply {
		return true
	}
	if latestApplication == nil {
		return false
	}
	for _, check := range preflight.Checks {
		if check.Severity != domain.PermissionPackagePreflightBlocking {
			continue
		}
		if check.Code == "approval_request_missing" || check.Code == "approval_request_invalid" {
			continue
		}
		return false
	}
	return true
}

func permissionPackageProductionApplicationScopeMatches(query permissionPackageProductionReadinessQuery, application domain.PermissionPackageApplication) bool {
	return application.TenantID == query.TenantID &&
		application.WorkspaceID == query.WorkspaceID &&
		application.TemplateID == query.TemplateID &&
		application.TargetID == query.TargetID &&
		application.CallerInstanceID == query.CallerInstanceID
}

func permissionPackageProductionAccessProfileEvidenceID(profile tenantAccessProfileResponse, query permissionPackageProductionReadinessQuery, application *domain.PermissionPackageApplication) string {
	allowedCapabilityIDs := map[string]struct{}{}
	if application != nil {
		for _, capabilityID := range application.AllowedCapabilityIDs {
			allowedCapabilityIDs[capabilityID] = struct{}{}
		}
	}
	for _, grant := range profile.Grants {
		if grant.ScopeStatus != accessProfileScopeValid || grant.Target == nil || grant.Target.ID != query.TargetID || grant.Capability == nil {
			continue
		}
		if len(allowedCapabilityIDs) > 0 {
			if _, ok := allowedCapabilityIDs[grant.Capability.ID]; !ok {
				continue
			}
		}
		for _, workspace := range grant.WorkspaceAssignments {
			if workspace.ScopeStatus != accessProfileScopeValid || workspace.WorkspaceAssignment.WorkspaceID != query.WorkspaceID {
				continue
			}
			for _, instance := range workspace.InstanceAssignments {
				if instance.ScopeStatus == accessProfileScopeValid && instance.InstanceAssignment.CallerInstanceID == query.CallerInstanceID {
					return instance.InstanceAssignment.ID
				}
			}
		}
	}
	return ""
}

func permissionPackageProductionLatestTrace(traces []domain.TraceEvent, decision domain.TraceDecision, subjectID string) *domain.TraceEvent {
	for index := len(traces) - 1; index >= 0; index-- {
		trace := traces[index]
		if trace.Decision != decision {
			continue
		}
		if subjectID != "" && trace.SubjectID != subjectID {
			continue
		}
		return &trace
	}
	return nil
}

func permissionPackageProductionReadinessCheckFor(code string, severity domain.PermissionPackagePreflightSeverity, message string, evidenceID string) permissionPackageProductionReadinessCheck {
	return permissionPackageProductionReadinessCheck{
		Code:       code,
		Severity:   severity,
		Message:    message,
		EvidenceID: evidenceID,
	}
}

func permissionPackageProductionAddNextAction(result *permissionPackageProductionReadinessResponse, code string, message string) {
	if result.NextActionCode == "" {
		result.NextActionCode = code
	}
	result.NextActions = appendUniqueString(result.NextActions, message)
}

func permissionPackageProductionReadinessSummaryFor(result permissionPackageProductionReadinessResponse) permissionPackageProductionReadinessSummary {
	summary := result.Summary
	for _, check := range result.Checks {
		switch check.Severity {
		case domain.PermissionPackagePreflightPassed:
			summary.ReadyCount++
		case domain.PermissionPackagePreflightWarning:
			summary.WarningCount++
		case domain.PermissionPackagePreflightBlocking:
			summary.BlockingCount++
		}
	}
	return summary
}

func permissionPackageProductionReadinessStatus(summary permissionPackageProductionReadinessSummary) string {
	if summary.BlockingCount > 0 {
		return "blocked"
	}
	if summary.WarningCount > 0 {
		return "needs_review"
	}
	return "ready"
}

type permissionPackageApplicationImpactResponse struct {
	Application       domain.PermissionPackageApplication            `json:"application"`
	Summary           permissionPackageApplicationImpactSummary      `json:"summary"`
	CreatedObjects    []permissionPackageApplicationImpactObject     `json:"createdObjects"`
	CapabilityReviews []permissionPackageApplicationImpactCapability `json:"capabilityReviews"`
	RollbackReview    permissionPackageApplicationRollbackReview     `json:"rollbackReview"`
	RemediationPlan   permissionPackageApplicationRemediationPlan    `json:"remediationPlan"`
	Rehearsal         *permissionPackageApplicationImpactRehearsal   `json:"rehearsal,omitempty"`
}

type permissionPackageApplicationImpactRehearsal struct {
	Enabled  bool   `json:"enabled"`
	Scenario string `json:"scenario"`
}

type permissionPackageApplicationImpactSummary struct {
	CreatedObjectCount int  `json:"createdObjectCount"`
	ActiveObjectCount  int  `json:"activeObjectCount"`
	MissingObjectCount int  `json:"missingObjectCount"`
	RollbackReady      bool `json:"rollbackReady"`
}

type permissionPackageApplicationImpactObject struct {
	ID             string             `json:"id"`
	Type           string             `json:"type"`
	CurrentStatus  string             `json:"currentStatus"`
	RollbackAction string             `json:"rollbackAction"`
	DataScopes     []domain.DataScope `json:"dataScopes,omitempty"`
}

type permissionPackageApplicationImpactCapability struct {
	ID             string `json:"id"`
	Key            string `json:"key,omitempty"`
	CurrentStatus  string `json:"currentStatus"`
	RollbackAction string `json:"rollbackAction"`
}

type permissionPackageApplicationRollbackReview struct {
	Ready        bool     `json:"ready"`
	Blockers     []string `json:"blockers"`
	BlockerCodes []string `json:"blockerCodes"`
	Steps        []string `json:"steps"`
}

type permissionPackageApplicationRemediationPlan struct {
	ExecutionMode string                                          `json:"executionMode"`
	Ready         bool                                            `json:"ready"`
	Blockers      []string                                        `json:"blockers"`
	BlockerCodes  []string                                        `json:"blockerCodes"`
	Actions       []permissionPackageApplicationRemediationAction `json:"actions"`
}

type permissionPackageApplicationRemediationAction struct {
	ID            string `json:"id"`
	Order         int    `json:"order"`
	TargetType    string `json:"targetType"`
	TargetID      string `json:"targetId"`
	Action        string `json:"action"`
	CurrentStatus string `json:"currentStatus,omitempty"`
	Reason        string `json:"reason"`
	ReadOnly      bool   `json:"readOnly"`
}

func (s *Server) getPermissionPackageApplicationImpact(w http.ResponseWriter, r *http.Request) {
	applicationID := strings.TrimSpace(chi.URLParam(r, "id"))
	if applicationID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "permission package application id is required"))
		return
	}
	rehearsal := strings.TrimSpace(r.URL.Query().Get("rehearsal"))
	if rehearsal != "" && rehearsal != "grant_drift" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "rehearsal must be grant_drift when provided"))
		return
	}
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListPermissionPackageApplications(r.Context(), store.PermissionPackageApplicationFilter{
		ID:              applicationID,
		ManagementScope: scope,
		Limit:           1,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if len(rows) == 0 {
		writeError(w, domain.NotFound("permission package application not found"))
		return
	}
	impact, err := s.permissionPackageApplicationImpact(r.Context(), rows[0])
	if err != nil {
		writeError(w, err)
		return
	}
	if rehearsal == "grant_drift" {
		impact = permissionPackageApplicationImpactGrantDriftRehearsal(impact)
	}
	writeJSON(w, http.StatusOK, impact)
}

func (s *Server) permissionPackageApplicationImpact(ctx context.Context, application domain.PermissionPackageApplication) (permissionPackageApplicationImpactResponse, error) {
	createdObjects, err := s.permissionPackageApplicationImpactObjects(ctx, application)
	if err != nil {
		return permissionPackageApplicationImpactResponse{}, err
	}
	capabilityReviews, err := s.permissionPackageApplicationImpactCapabilities(ctx, application)
	if err != nil {
		return permissionPackageApplicationImpactResponse{}, err
	}
	summary := permissionPackageApplicationImpactSummaryFor(createdObjects)
	rollbackReview := permissionPackageApplicationRollbackReviewFor(application, summary)
	return permissionPackageApplicationImpactResponse{
		Application:       application,
		Summary:           summary,
		CreatedObjects:    createdObjects,
		CapabilityReviews: capabilityReviews,
		RollbackReview:    rollbackReview,
		RemediationPlan:   permissionPackageApplicationRemediationPlanFor(application, createdObjects, capabilityReviews, rollbackReview),
	}, nil
}

func permissionPackageApplicationImpactSummaryFor(createdObjects []permissionPackageApplicationImpactObject) permissionPackageApplicationImpactSummary {
	summary := permissionPackageApplicationImpactSummary{CreatedObjectCount: len(createdObjects)}
	for _, row := range createdObjects {
		switch row.CurrentStatus {
		case string(domain.PolicyStatusEnabled):
			summary.ActiveObjectCount++
		case "missing":
			summary.MissingObjectCount++
		}
	}
	summary.RollbackReady = summary.CreatedObjectCount > 0 &&
		summary.ActiveObjectCount == summary.CreatedObjectCount &&
		summary.MissingObjectCount == 0
	return summary
}

func permissionPackageApplicationImpactGrantDriftRehearsal(impact permissionPackageApplicationImpactResponse) permissionPackageApplicationImpactResponse {
	next := impact
	next.CreatedObjects = append([]permissionPackageApplicationImpactObject(nil), impact.CreatedObjects...)
	for index := range next.CreatedObjects {
		if next.CreatedObjects[index].Type != "workspace_assignment" {
			continue
		}
		next.CreatedObjects[index].CurrentStatus = "missing"
		next.CreatedObjects[index].RollbackAction = "investigate"
		next.CreatedObjects[index].DataScopes = nil
		break
	}
	instanceMarked := false
	for index := range next.CreatedObjects {
		if next.CreatedObjects[index].Type != "instance_assignment" {
			continue
		}
		next.CreatedObjects[index].CurrentStatus = string(domain.PolicyStatusDisabled)
		next.CreatedObjects[index].RollbackAction = "investigate"
		instanceMarked = true
		break
	}
	if !instanceMarked {
		for index := range next.CreatedObjects {
			if next.CreatedObjects[index].Type != "tenant_entitlement" {
				continue
			}
			next.CreatedObjects[index].CurrentStatus = string(domain.PolicyStatusDisabled)
			next.CreatedObjects[index].RollbackAction = "investigate"
			break
		}
	}
	next.Summary = permissionPackageApplicationImpactSummaryFor(next.CreatedObjects)
	next.RollbackReview = permissionPackageApplicationRollbackReviewFor(next.Application, next.Summary)
	next.RemediationPlan = permissionPackageApplicationRemediationPlanFor(next.Application, next.CreatedObjects, next.CapabilityReviews, next.RollbackReview)
	next.Rehearsal = &permissionPackageApplicationImpactRehearsal{
		Enabled:  true,
		Scenario: "grant_drift",
	}
	return next
}

func (s *Server) permissionPackageApplicationImpactObjects(ctx context.Context, application domain.PermissionPackageApplication) ([]permissionPackageApplicationImpactObject, error) {
	objects := make([]permissionPackageApplicationImpactObject, 0,
		len(application.TenantEntitlementIDs)+len(application.WorkspaceAssignmentIDs)+len(application.InstanceAssignmentIDs))

	entitlements, err := s.repo.ListTenantEntitlements(ctx, store.EntitlementFilter{
		ManagementScope: store.ManagementScope{TenantID: application.TenantID},
		TargetID:        application.TargetID,
	})
	if err != nil {
		return nil, err
	}
	entitlementByID := map[string]domain.TenantEntitlement{}
	for _, entitlement := range entitlements {
		entitlementByID[entitlement.ID] = entitlement
	}
	for _, id := range application.TenantEntitlementIDs {
		if entitlement, ok := entitlementByID[id]; ok {
			objects = append(objects, permissionPackageImpactObjectFromGrant("tenant_entitlement", entitlement.ID, string(entitlement.Status), entitlement.DataScopes))
		} else {
			objects = append(objects, missingPermissionPackageImpactObject("tenant_entitlement", id))
		}
	}

	workspaceAssignments, err := s.repo.ListWorkspaceAssignments(ctx, store.AssignmentFilter{
		ManagementScope: store.ManagementScope{TenantID: application.TenantID, WorkspaceID: application.WorkspaceID},
	})
	if err != nil {
		return nil, err
	}
	workspaceAssignmentByID := map[string]domain.WorkspaceAssignment{}
	for _, assignment := range workspaceAssignments {
		workspaceAssignmentByID[assignment.ID] = assignment
	}
	for _, id := range application.WorkspaceAssignmentIDs {
		if assignment, ok := workspaceAssignmentByID[id]; ok {
			objects = append(objects, permissionPackageImpactObjectFromGrant("workspace_assignment", assignment.ID, string(assignment.Status), assignment.DataScopes))
		} else {
			objects = append(objects, missingPermissionPackageImpactObject("workspace_assignment", id))
		}
	}

	instanceAssignments, err := s.repo.ListInstanceAssignments(ctx, store.InstanceAssignmentFilter{
		ManagementScope:  store.ManagementScope{TenantID: application.TenantID, WorkspaceID: application.WorkspaceID},
		CallerInstanceID: application.CallerInstanceID,
	})
	if err != nil {
		return nil, err
	}
	instanceAssignmentByID := map[string]domain.InstanceAssignment{}
	for _, assignment := range instanceAssignments {
		instanceAssignmentByID[assignment.ID] = assignment
	}
	for _, id := range application.InstanceAssignmentIDs {
		if assignment, ok := instanceAssignmentByID[id]; ok {
			objects = append(objects, permissionPackageImpactObjectFromGrant("instance_assignment", assignment.ID, string(assignment.Status), assignment.DataScopes))
		} else {
			objects = append(objects, missingPermissionPackageImpactObject("instance_assignment", id))
		}
	}
	return objects, nil
}

func (s *Server) permissionPackageApplicationImpactCapabilities(ctx context.Context, application domain.PermissionPackageApplication) ([]permissionPackageApplicationImpactCapability, error) {
	rows := make([]permissionPackageApplicationImpactCapability, 0, len(application.AllowedCapabilityIDs))
	for index, id := range application.AllowedCapabilityIDs {
		capability, ok, err := s.repo.GetCapability(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			rows = append(rows, permissionPackageApplicationImpactCapability{
				ID:             capability.ID,
				Key:            capability.Key,
				CurrentStatus:  string(capability.DiscoveryStatus),
				RollbackAction: "manual_review",
			})
			continue
		}
		key := ""
		if index < len(application.AllowedCapabilityKeys) {
			key = application.AllowedCapabilityKeys[index]
		}
		rows = append(rows, permissionPackageApplicationImpactCapability{
			ID:             id,
			Key:            key,
			CurrentStatus:  "missing",
			RollbackAction: "investigate",
		})
	}
	return rows, nil
}

func permissionPackageImpactObjectFromGrant(objectType string, id string, currentStatus string, dataScopes []domain.DataScope) permissionPackageApplicationImpactObject {
	rollbackAction := "investigate"
	if currentStatus == string(domain.PolicyStatusEnabled) {
		rollbackAction = "disable"
	}
	return permissionPackageApplicationImpactObject{
		ID:             id,
		Type:           objectType,
		CurrentStatus:  currentStatus,
		RollbackAction: rollbackAction,
		DataScopes:     append([]domain.DataScope(nil), dataScopes...),
	}
}

func missingPermissionPackageImpactObject(objectType string, id string) permissionPackageApplicationImpactObject {
	return permissionPackageApplicationImpactObject{
		ID:             id,
		Type:           objectType,
		CurrentStatus:  "missing",
		RollbackAction: "investigate",
	}
}

func permissionPackageApplicationRollbackReviewFor(application domain.PermissionPackageApplication, summary permissionPackageApplicationImpactSummary) permissionPackageApplicationRollbackReview {
	review := permissionPackageApplicationRollbackReview{
		Ready:        summary.RollbackReady,
		Blockers:     []string{},
		BlockerCodes: []string{},
		Steps: []string{
			"Review capability discovery status manually; shared capabilities are not automatically downgraded by rollback.",
			"Disable recorded instance assignments before workspace assignments.",
			"Disable recorded workspace assignments before tenant entitlements.",
			"Disable recorded tenant entitlements and then verify effective access decisions.",
		},
	}
	if summary.MissingObjectCount > 0 {
		review.Blockers = append(review.Blockers, "Some recorded grant objects are missing; investigate drift before rollback.")
		review.BlockerCodes = append(review.BlockerCodes, "missing_created_objects")
	}
	if summary.ActiveObjectCount != summary.CreatedObjectCount {
		review.Blockers = append(review.Blockers, "Some recorded grant objects are not enabled; review partial rollback or manual changes.")
		review.BlockerCodes = append(review.BlockerCodes, "inactive_created_objects")
	}
	if len(application.AllowedCapabilityIDs) == 0 {
		review.Blockers = append(review.Blockers, "Application has no recorded allowed capabilities.")
		review.BlockerCodes = append(review.BlockerCodes, "no_allowed_capabilities")
	}
	if len(review.Blockers) > 0 {
		review.Ready = false
	}
	return review
}

func permissionPackageApplicationRemediationPlanFor(application domain.PermissionPackageApplication, createdObjects []permissionPackageApplicationImpactObject, capabilityReviews []permissionPackageApplicationImpactCapability, rollbackReview permissionPackageApplicationRollbackReview) permissionPackageApplicationRemediationPlan {
	plan := permissionPackageApplicationRemediationPlan{
		ExecutionMode: "read_only",
		Ready:         rollbackReview.Ready,
		Blockers:      append([]string{}, rollbackReview.Blockers...),
		BlockerCodes:  append([]string{}, rollbackReview.BlockerCodes...),
		Actions:       []permissionPackageApplicationRemediationAction{},
	}
	for _, capability := range capabilityReviews {
		action := capability.RollbackAction
		reason := "shared_capability_manual_review"
		if action == "investigate" {
			reason = "capability_drift_investigation"
		}
		plan.addAction("capability", capability.ID, action, capability.CurrentStatus, reason)
	}
	for _, objectType := range []string{"instance_assignment", "workspace_assignment", "tenant_entitlement"} {
		for _, object := range createdObjects {
			if object.Type != objectType || object.RollbackAction == "disable" {
				continue
			}
			plan.addAction(object.Type, object.ID, "investigate", object.CurrentStatus, "grant_drift_investigation")
		}
	}
	for _, objectType := range []string{"instance_assignment", "workspace_assignment", "tenant_entitlement"} {
		for _, object := range createdObjects {
			if object.Type != objectType || object.RollbackAction != "disable" {
				continue
			}
			plan.addAction(object.Type, object.ID, "disable", object.CurrentStatus, permissionPackageDisableRemediationReason(object.Type))
		}
	}
	plan.addAction("access_decision", application.ID, "verify", "", "verify_effective_access")
	if len(plan.Blockers) > 0 {
		plan.Ready = false
	}
	return plan
}

func (plan *permissionPackageApplicationRemediationPlan) addAction(targetType string, targetID string, action string, currentStatus string, reason string) {
	order := len(plan.Actions) + 1
	plan.Actions = append(plan.Actions, permissionPackageApplicationRemediationAction{
		ID:            fmt.Sprintf("remediation:%02d:%s:%s:%s", order, targetType, targetID, action),
		Order:         order,
		TargetType:    targetType,
		TargetID:      targetID,
		Action:        action,
		CurrentStatus: currentStatus,
		Reason:        reason,
		ReadOnly:      true,
	})
}

func permissionPackageDisableRemediationReason(objectType string) string {
	switch objectType {
	case "instance_assignment":
		return "disable_instance_assignment"
	case "workspace_assignment":
		return "disable_workspace_assignment"
	case "tenant_entitlement":
		return "disable_tenant_entitlement"
	default:
		return "grant_drift_investigation"
	}
}

func (s *Server) listPermissionPackageApprovalRequests(w http.ResponseWriter, r *http.Request) {
	limit, err := auditLimitFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	status := domain.PermissionPackageApprovalStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validPermissionPackageApprovalStatus(status) {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "status must be pending, approved, rejected, or withdrawn"))
		return
	}
	reviewer := strings.TrimSpace(r.URL.Query().Get("reviewer"))
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter := store.PermissionPackageApprovalRequestFilter{
		ManagementScope:  scope,
		TemplateID:       strings.TrimSpace(r.URL.Query().Get("templateId")),
		TargetID:         strings.TrimSpace(r.URL.Query().Get("targetId")),
		CallerInstanceID: strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		Status:           status,
		Limit:            limit,
	}
	rows, err := s.listPermissionPackageApprovalRequestsForRequest(r.Context(), r, filter, reviewer, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) createPermissionPackageDraft(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageDraftRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageDraftScope(r, req); err != nil {
		writeError(w, err)
		return
	}
	draft, err := s.buildPermissionPackageDraft(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (s *Server) preflightPermissionPackage(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageDraftScope(r, req.PermissionPackageDraftRequest); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.preflightPermissionPackageRequest(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) preflightPermissionPackageRequest(ctx context.Context, req domain.PermissionPackageApplyRequest) (domain.PermissionPackageApplyPreflightResponse, error) {
	req.ApprovalRequestID = strings.TrimSpace(req.ApprovalRequestID)
	draft, err := s.buildPermissionPackageDraft(ctx, req.PermissionPackageDraftRequest)
	if err != nil {
		return domain.PermissionPackageApplyPreflightResponse{}, err
	}

	result := domain.PermissionPackageApplyPreflightResponse{
		Draft:  draft,
		Checks: []domain.PermissionPackageApplyPreflightCheck{},
		Planned: domain.PermissionPackageApplyPreflightPlannedChanges{
			Capabilities:         []domain.Capability{},
			TenantEntitlements:   []domain.TenantEntitlement{},
			WorkspaceAssignments: []domain.WorkspaceAssignment{},
			InstanceAssignments:  []domain.InstanceAssignment{},
		},
		ExistingGrants: []domain.PermissionPackageApplyPreflightExistingGrant{},
		NextActions:    []string{},
	}

	if draft.Readiness.CanApply {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("draft_ready", domain.PermissionPackagePreflightPassed, "Permission package draft is ready to evaluate.", "", ""))
	} else {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("draft_not_ready", domain.PermissionPackagePreflightBlocking, "Permission package draft is not ready to apply.", "", ""))
		result.NextActions = appendUniqueString(result.NextActions, "Fix draft readiness blockers before applying this permission request.")
	}

	requiresApproval := !draft.PolicyGate.CanApplyDirectly
	approvalReady := false
	if requiresApproval {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("policy_gate", domain.PermissionPackagePreflightInfo, "Policy gate requires an approved permission package approval request.", "", ""))
		if req.ApprovalRequestID == "" {
			result.Checks = append(result.Checks, permissionPackagePreflightCheck("approval_request_missing", domain.PermissionPackagePreflightBlocking, "Permission package requires approval before apply.", "", ""))
			result.NextActions = appendUniqueString(result.NextActions, "Create and approve an approval request for this permission request, then preflight again with approvalRequestId.")
		} else {
			approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(ctx, req.ApprovalRequestID)
			if err != nil {
				return domain.PermissionPackageApplyPreflightResponse{}, err
			}
			if !ok {
				result.Checks = append(result.Checks, permissionPackagePreflightCheck("approval_request_invalid", domain.PermissionPackagePreflightBlocking, "Approval request was not found.", "", ""))
				result.NextActions = appendUniqueString(result.NextActions, "Use an approved approvalRequestId that matches the current draft.")
			} else if err := validatePermissionPackageApprovalForDraft(approval, draft, s.now()); err != nil {
				result.Checks = append(result.Checks, permissionPackagePreflightCheck("approval_request_invalid", domain.PermissionPackagePreflightBlocking, err.Error(), "", ""))
				result.NextActions = appendUniqueString(result.NextActions, "Refresh approval or create a new approval request for the current draft.")
			} else {
				approvalReady = true
				result.Checks = append(result.Checks, permissionPackagePreflightCheck("approval_request_ready", domain.PermissionPackagePreflightPassed, "Approval request is approved and matches the current draft.", "", ""))
			}
		}
	} else {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("policy_gate", domain.PermissionPackagePreflightPassed, "Policy gate allows direct apply.", "", ""))
	}

	dataScopeConflictCount := 0
	for _, capability := range draft.AllowedCapabilities {
		effectiveScopes, ok := domain.EffectiveDataScopes(capability.DataScopes, draft.DataScopes)
		if !ok {
			dataScopeConflictCount++
			result.Checks = append(result.Checks, permissionPackagePreflightCheck("data_scope_fit", domain.PermissionPackagePreflightBlocking, "Permission package dataScopes exceed capability dataScopes.", capability.ID, capability.Key))
			result.NextActions = appendUniqueString(result.NextActions, "Narrow region or data scopes so the package stays inside every capability boundary.")
			continue
		}
		result.Planned.Capabilities = append(result.Planned.Capabilities, permissionPackagePreflightPlannedCapability(capability, effectiveScopes))
		entitlement, workspaceAssignment, instanceAssignment := permissionPackagePreflightPlannedGrantChain(draft, capability, effectiveScopes)
		result.Planned.TenantEntitlements = append(result.Planned.TenantEntitlements, entitlement)
		result.Planned.WorkspaceAssignments = append(result.Planned.WorkspaceAssignments, workspaceAssignment)
		result.Planned.InstanceAssignments = append(result.Planned.InstanceAssignments, instanceAssignment)

		existingGrants, err := s.permissionPackagePreflightExistingGrants(ctx, draft, capability)
		if err != nil {
			return domain.PermissionPackageApplyPreflightResponse{}, err
		}
		for _, existing := range existingGrants {
			result.ExistingGrants = append(result.ExistingGrants, existing)
			result.Checks = append(result.Checks, permissionPackagePreflightCheck("existing_grant_chain", domain.PermissionPackagePreflightWarning, "An enabled grant chain already exists for this tenant, workspace, caller, and capability.", capability.ID, capability.Key))
			result.NextActions = appendUniqueString(result.NextActions, "Review existing grant chains before applying another permission request for the same caller and capability.")
		}
	}
	if dataScopeConflictCount == 0 {
		result.Checks = append(result.Checks, permissionPackagePreflightCheck("data_scope_fit", domain.PermissionPackagePreflightPassed, "Permission package dataScopes fit all allowed capability boundaries.", "", ""))
	}
	result.Checks = append(result.Checks, permissionPackagePreflightCheck("planned_changes", domain.PermissionPackagePreflightInfo, "Preflight planned grant objects without writing them.", "", ""))

	result.Summary = permissionPackagePreflightSummary(result, requiresApproval, approvalReady)
	if result.Summary.CanApply {
		result.NextActions = appendUniqueString(result.NextActions, "Apply this permission request when the reviewer is ready.")
	}
	return result, nil
}

func (s *Server) createPermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageDraftRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.requirePermissionPackageDraftScope(r, req); err != nil {
		writeError(w, err)
		return
	}
	created, err := s.createPermissionPackageApprovalRequestRecord(r.Context(), req, managementActor(r), s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "permission_package.approval_requested", "permission_package_approval_request", created.ID, "Permission package approval requested", permissionPackageApprovalAuditMetadata(created))); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) approvePermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request) {
	s.resolvePermissionPackageApprovalRequest(w, r, domain.PermissionPackageApprovalStatusApproved)
}

func (s *Server) rejectPermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request) {
	s.resolvePermissionPackageApprovalRequest(w, r, domain.PermissionPackageApprovalStatusRejected)
}

func (s *Server) withdrawPermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, domain.NotFound("approval request not found"))
		return
	}
	var req domain.PermissionPackageApprovalResolutionRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
	}
	existing, ok, err := s.repo.GetPermissionPackageApprovalRequest(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("approval request not found"))
		return
	}
	if err := s.requirePermissionPackageApprovalRequestScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	requester := managementActor(r)
	saved, err := s.withdrawPermissionPackageApprovalRequestRecord(r.Context(), existing, requester, req.Comment, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, saved.TenantID, saved.WorkspaceID, "permission_package.approval_withdrawn", "permission_package_approval_request", saved.ID, "Permission package approval withdrawn", permissionPackageApprovalAuditMetadata(saved))); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) resolvePermissionPackageApprovalRequest(w http.ResponseWriter, r *http.Request, status domain.PermissionPackageApprovalStatus) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, domain.NotFound("approval request not found"))
		return
	}
	var req domain.PermissionPackageApprovalResolutionRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
	}
	existing, ok, err := s.repo.GetPermissionPackageApprovalRequest(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("approval request not found"))
		return
	}
	if err := s.requirePermissionPackageApprovalRequestScope(r, existing); err != nil {
		writeError(w, err)
		return
	}
	reviewer, err := reviewerFromRequest(req.Reviewer, r)
	if err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	if err := s.validatePermissionPackageApprovalReviewer(r.Context(), reviewer, existing); err != nil {
		writeError(w, err)
		return
	}
	saved, err := s.resolvePermissionPackageApprovalRequestRecord(r.Context(), existing, status, reviewer, req.Comment, now)
	if err != nil {
		writeError(w, err)
		return
	}
	action := "permission_package.approval_approved"
	summary := "Permission package approval approved"
	if status == domain.PermissionPackageApprovalStatusRejected {
		action = "permission_package.approval_rejected"
		summary = "Permission package approval rejected"
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, saved.TenantID, saved.WorkspaceID, action, "permission_package_approval_request", saved.ID, summary, permissionPackageApprovalAuditMetadata(saved))); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) applyPermissionPackage(w http.ResponseWriter, r *http.Request) {
	var req domain.PermissionPackageApplyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.applyPermissionPackageRequest(r, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) applyPermissionPackageRequest(r *http.Request, req domain.PermissionPackageApplyRequest) (domain.PermissionPackageApplyResponse, error) {
	req.ApprovalRequestID = strings.TrimSpace(req.ApprovalRequestID)
	if err := s.requirePermissionPackageDraftScope(r, req.PermissionPackageDraftRequest); err != nil {
		return domain.PermissionPackageApplyResponse{}, err
	}
	draft, err := s.buildPermissionPackageDraft(r.Context(), req.PermissionPackageDraftRequest)
	if err != nil {
		return domain.PermissionPackageApplyResponse{}, err
	}
	if !draft.Readiness.CanApply {
		return domain.PermissionPackageApplyResponse{}, domain.BadRequest("VALIDATION_FAILED", "permission package draft is not ready to apply")
	}
	approvalRequestID := ""
	var approvalForApply *domain.PermissionPackageApprovalRequest
	if !draft.PolicyGate.CanApplyDirectly {
		if req.ApprovalRequestID == "" {
			return domain.PermissionPackageApplyResponse{}, domain.BadRequest("VALIDATION_FAILED", "permission package requires approval before apply")
		}
		approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(r.Context(), req.ApprovalRequestID)
		if err != nil {
			return domain.PermissionPackageApplyResponse{}, err
		}
		if !ok {
			return domain.PermissionPackageApplyResponse{}, domain.NotFound("approval request not found")
		}
		if err := validatePermissionPackageApprovalForDraft(approval, draft, s.now()); err != nil {
			return domain.PermissionPackageApplyResponse{}, err
		}
		approvalRequestID = approval.ID
		approvalForApply = &approval
	}

	now := s.now()
	result := domain.PermissionPackageApplyResponse{
		Draft:                draft,
		TenantEntitlements:   []domain.TenantEntitlement{},
		WorkspaceAssignments: []domain.WorkspaceAssignment{},
		InstanceAssignments:  []domain.InstanceAssignment{},
	}
	appliedCapabilityIDs := make([]string, 0, len(draft.AllowedCapabilities))
	appliedCapabilityKeys := make([]string, 0, len(draft.AllowedCapabilities))
	tenantEntitlementIDs := make([]string, 0, len(draft.AllowedCapabilities))
	workspaceAssignmentIDs := make([]string, 0, len(draft.AllowedCapabilities))
	instanceAssignmentIDs := make([]string, 0, len(draft.AllowedCapabilities))
	capabilityMutations := make([]domain.Capability, 0, len(draft.AllowedCapabilities))
	tenantEntitlements := make([]domain.TenantEntitlement, 0, len(draft.AllowedCapabilities))
	workspaceAssignments := make([]domain.WorkspaceAssignment, 0, len(draft.AllowedCapabilities))
	instanceAssignments := make([]domain.InstanceAssignment, 0, len(draft.AllowedCapabilities))

	for _, capability := range draft.AllowedCapabilities {
		effectiveScopes, ok := domain.EffectiveDataScopes(capability.DataScopes, draft.DataScopes)
		if !ok {
			return domain.PermissionPackageApplyResponse{}, domain.BadRequest("VALIDATION_FAILED", "permission package dataScopes exceed capability dataScopes")
		}
		updatedCapability := capability
		updatedCapability.DiscoveryStatus = domain.CapabilityDiscoveryApproved
		updatedCapability.DataScopes = effectiveScopes
		updatedCapability.UpdatedAt = now
		capabilityMutations = append(capabilityMutations, updatedCapability)

		entitlement := domain.TenantEntitlement{
			ID:           security.NewID("ent"),
			TenantID:     draft.Input.TenantID,
			TargetID:     draft.Input.TargetID,
			CapabilityID: updatedCapability.ID,
			Effect:       domain.PolicyEffectAllow,
			DataScopes:   effectiveScopes,
			Status:       domain.PolicyStatusEnabled,
			Priority:     40,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		workspaceAssignment := domain.WorkspaceAssignment{
			ID:                  security.NewID("wsa"),
			TenantEntitlementID: entitlement.ID,
			TenantID:            draft.Input.TenantID,
			WorkspaceID:         draft.Input.WorkspaceID,
			Effect:              domain.PolicyEffectAllow,
			DataScopes:          effectiveScopes,
			Status:              domain.PolicyStatusEnabled,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		instanceAssignment := domain.InstanceAssignment{
			ID:                    security.NewID("ina"),
			WorkspaceAssignmentID: workspaceAssignment.ID,
			TenantID:              draft.Input.TenantID,
			WorkspaceID:           draft.Input.WorkspaceID,
			CallerInstanceID:      draft.Input.CallerInstanceID,
			SubjectSelector:       draft.Input.SubjectSelector,
			Effect:                domain.PolicyEffectAllow,
			DataScopes:            effectiveScopes,
			Status:                domain.PolicyStatusEnabled,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		tenantEntitlements = append(tenantEntitlements, entitlement)
		workspaceAssignments = append(workspaceAssignments, workspaceAssignment)
		instanceAssignments = append(instanceAssignments, instanceAssignment)
		appliedCapabilityIDs = append(appliedCapabilityIDs, updatedCapability.ID)
		appliedCapabilityKeys = append(appliedCapabilityKeys, updatedCapability.Key)
		tenantEntitlementIDs = append(tenantEntitlementIDs, entitlement.ID)
		workspaceAssignmentIDs = append(workspaceAssignmentIDs, workspaceAssignment.ID)
		instanceAssignmentIDs = append(instanceAssignmentIDs, instanceAssignment.ID)
	}

	application := domain.PermissionPackageApplication{
		ID:                     security.NewID("ppa"),
		DraftID:                draft.ID,
		TemplateID:             draft.Template.ID,
		TemplateVersion:        draft.Template.Version,
		TenantID:               draft.Input.TenantID,
		WorkspaceID:            draft.Input.WorkspaceID,
		TargetID:               draft.Input.TargetID,
		CallerInstanceID:       draft.Input.CallerInstanceID,
		SubjectSelector:        draft.Input.SubjectSelector,
		RequestText:            draft.Input.RequestText,
		Region:                 draft.Input.Region,
		DataScopes:             draft.DataScopes,
		AllowedCapabilityIDs:   appliedCapabilityIDs,
		AllowedCapabilityKeys:  appliedCapabilityKeys,
		TenantEntitlementIDs:   tenantEntitlementIDs,
		WorkspaceAssignmentIDs: workspaceAssignmentIDs,
		InstanceAssignmentIDs:  instanceAssignmentIDs,
		AppliedAt:              now,
	}
	var consumedApproval *domain.PermissionPackageApprovalRequest
	if approvalForApply != nil {
		approval := *approvalForApply
		approval.ConsumedAt = now
		approval.ConsumedByApplicationID = application.ID
		approval.UpdatedAt = now
		consumedApproval = &approval
	}

	auditMetadata := map[string]any{
		"applicationId":          application.ID,
		"draftId":                draft.ID,
		"templateId":             draft.Template.ID,
		"templateVersion":        draft.Template.Version,
		"targetId":               draft.Input.TargetID,
		"callerInstanceId":       draft.Input.CallerInstanceID,
		"subjectSelector":        draft.Input.SubjectSelector,
		"allowedCapabilityIds":   appliedCapabilityIDs,
		"allowedCapabilityKeys":  appliedCapabilityKeys,
		"tenantEntitlementIds":   tenantEntitlementIDs,
		"workspaceAssignmentIds": workspaceAssignmentIDs,
		"instanceAssignmentIds":  instanceAssignmentIDs,
	}
	if consumedApproval != nil {
		auditMetadata["approvalRequestId"] = consumedApproval.ID
		auditMetadata["approvalExpiresAt"] = consumedApproval.ExpiresAt
		auditMetadata["approvalConsumedAt"] = consumedApproval.ConsumedAt
		auditMetadata["approvalConsumedByApplicationId"] = consumedApproval.ConsumedByApplicationID
	} else if approvalRequestID != "" {
		auditMetadata["approvalRequestId"] = approvalRequestID
	}
	applyResult, err := s.repo.ApplyPermissionPackage(r.Context(), store.PermissionPackageApplyMutation{
		Capabilities:         capabilityMutations,
		TenantEntitlements:   tenantEntitlements,
		WorkspaceAssignments: workspaceAssignments,
		InstanceAssignments:  instanceAssignments,
		Application:          application,
		ApprovalRequest:      consumedApproval,
		AuditEvent:           s.managementAuditEvent(r, draft.Input.TenantID, draft.Input.WorkspaceID, "permission_package.applied", "permission_package", application.ID, "Permission package applied", auditMetadata),
	})
	if err != nil {
		if errors.Is(err, store.ErrPermissionPackageApprovalNotConsumable) {
			return domain.PermissionPackageApplyResponse{}, s.permissionPackageApprovalNotConsumableError(r.Context(), approvalRequestID, draft, now)
		}
		return domain.PermissionPackageApplyResponse{}, err
	}
	result.TenantEntitlements = applyResult.TenantEntitlements
	result.WorkspaceAssignments = applyResult.WorkspaceAssignments
	result.InstanceAssignments = applyResult.InstanceAssignments
	result.Application = &applyResult.Application
	return result, nil
}

func permissionPackagePreflightCheck(code string, severity domain.PermissionPackagePreflightSeverity, message string, capabilityID string, capabilityKey string) domain.PermissionPackageApplyPreflightCheck {
	return domain.PermissionPackageApplyPreflightCheck{
		Code:          code,
		Severity:      severity,
		Message:       message,
		CapabilityID:  capabilityID,
		CapabilityKey: capabilityKey,
	}
}

func permissionPackagePreflightSummary(result domain.PermissionPackageApplyPreflightResponse, requiresApproval bool, approvalReady bool) domain.PermissionPackageApplyPreflightSummary {
	summary := domain.PermissionPackageApplyPreflightSummary{
		CanApply:                        true,
		PlannedCapabilityCount:          len(result.Planned.Capabilities),
		PlannedTenantEntitlementCount:   len(result.Planned.TenantEntitlements),
		PlannedWorkspaceAssignmentCount: len(result.Planned.WorkspaceAssignments),
		PlannedInstanceAssignmentCount:  len(result.Planned.InstanceAssignments),
		ExistingGrantCount:              len(result.ExistingGrants),
		RequiresApproval:                requiresApproval,
		ApprovalReady:                   approvalReady,
	}
	for _, check := range result.Checks {
		switch check.Severity {
		case domain.PermissionPackagePreflightBlocking:
			summary.BlockingCount++
		case domain.PermissionPackagePreflightWarning:
			summary.WarningCount++
		}
	}
	summary.CanApply = summary.BlockingCount == 0
	return summary
}

func permissionPackagePreflightPlannedCapability(capability domain.Capability, dataScopes []domain.DataScope) domain.Capability {
	planned := capability
	planned.DiscoveryStatus = domain.CapabilityDiscoveryApproved
	planned.DataScopes = append([]domain.DataScope(nil), dataScopes...)
	return planned
}

func permissionPackagePreflightPlannedGrantChain(draft domain.PermissionPackageDraft, capability domain.Capability, dataScopes []domain.DataScope) (domain.TenantEntitlement, domain.WorkspaceAssignment, domain.InstanceAssignment) {
	entitlementID := "planned:ent:" + capability.ID
	workspaceAssignmentID := "planned:wsa:" + capability.ID
	return domain.TenantEntitlement{
			ID:           entitlementID,
			TenantID:     draft.Input.TenantID,
			TargetID:     draft.Input.TargetID,
			CapabilityID: capability.ID,
			Effect:       domain.PolicyEffectAllow,
			DataScopes:   append([]domain.DataScope(nil), dataScopes...),
			Status:       domain.PolicyStatusEnabled,
			Priority:     40,
		}, domain.WorkspaceAssignment{
			ID:                  workspaceAssignmentID,
			TenantEntitlementID: entitlementID,
			TenantID:            draft.Input.TenantID,
			WorkspaceID:         draft.Input.WorkspaceID,
			Effect:              domain.PolicyEffectAllow,
			DataScopes:          append([]domain.DataScope(nil), dataScopes...),
			Status:              domain.PolicyStatusEnabled,
		}, domain.InstanceAssignment{
			ID:                    "planned:ina:" + capability.ID,
			WorkspaceAssignmentID: workspaceAssignmentID,
			TenantID:              draft.Input.TenantID,
			WorkspaceID:           draft.Input.WorkspaceID,
			CallerInstanceID:      draft.Input.CallerInstanceID,
			SubjectSelector:       draft.Input.SubjectSelector,
			Effect:                domain.PolicyEffectAllow,
			DataScopes:            append([]domain.DataScope(nil), dataScopes...),
			Status:                domain.PolicyStatusEnabled,
		}
}

func (s *Server) permissionPackagePreflightExistingGrants(ctx context.Context, draft domain.PermissionPackageDraft, capability domain.Capability) ([]domain.PermissionPackageApplyPreflightExistingGrant, error) {
	entitlements, err := s.repo.ListTenantEntitlements(ctx, store.EntitlementFilter{
		ManagementScope: store.ManagementScope{TenantID: draft.Input.TenantID},
		TargetID:        draft.Input.TargetID,
		CapabilityID:    capability.ID,
	})
	if err != nil {
		return nil, err
	}
	grants := []domain.PermissionPackageApplyPreflightExistingGrant{}
	for _, entitlement := range entitlements {
		if entitlement.TenantID != draft.Input.TenantID ||
			entitlement.TargetID != draft.Input.TargetID ||
			entitlement.CapabilityID != capability.ID ||
			entitlement.Effect != domain.PolicyEffectAllow ||
			entitlement.Status != domain.PolicyStatusEnabled {
			continue
		}
		workspaceAssignments, err := s.repo.ListWorkspaceAssignments(ctx, store.AssignmentFilter{
			ManagementScope: store.ManagementScope{
				TenantID:    draft.Input.TenantID,
				WorkspaceID: draft.Input.WorkspaceID,
			},
			EntitlementID: entitlement.ID,
		})
		if err != nil {
			return nil, err
		}
		for _, workspaceAssignment := range workspaceAssignments {
			if workspaceAssignment.TenantID != draft.Input.TenantID ||
				workspaceAssignment.WorkspaceID != draft.Input.WorkspaceID ||
				workspaceAssignment.TenantEntitlementID != entitlement.ID ||
				workspaceAssignment.Effect != domain.PolicyEffectAllow ||
				workspaceAssignment.Status != domain.PolicyStatusEnabled {
				continue
			}
			instanceAssignments, err := s.repo.ListInstanceAssignments(ctx, store.InstanceAssignmentFilter{
				ManagementScope: store.ManagementScope{
					TenantID:    draft.Input.TenantID,
					WorkspaceID: draft.Input.WorkspaceID,
				},
				CallerInstanceID: draft.Input.CallerInstanceID,
				CapabilityID:     capability.ID,
			})
			if err != nil {
				return nil, err
			}
			for _, instanceAssignment := range instanceAssignments {
				if instanceAssignment.TenantID != draft.Input.TenantID ||
					instanceAssignment.WorkspaceID != draft.Input.WorkspaceID ||
					instanceAssignment.WorkspaceAssignmentID != workspaceAssignment.ID ||
					instanceAssignment.CallerInstanceID != draft.Input.CallerInstanceID ||
					instanceAssignment.Effect != domain.PolicyEffectAllow ||
					instanceAssignment.Status != domain.PolicyStatusEnabled ||
					!permissionPackageSubjectSelectorsOverlap(instanceAssignment.SubjectSelector, draft.Input.SubjectSelector) {
					continue
				}
				grants = append(grants, domain.PermissionPackageApplyPreflightExistingGrant{
					CapabilityID:          capability.ID,
					CapabilityKey:         capability.Key,
					TenantEntitlementID:   entitlement.ID,
					WorkspaceAssignmentID: workspaceAssignment.ID,
					InstanceAssignmentID:  instanceAssignment.ID,
				})
			}
		}
	}
	return grants, nil
}

func permissionPackageSubjectSelectorsOverlap(existing string, requested string) bool {
	existing = strings.TrimSpace(existing)
	requested = strings.TrimSpace(requested)
	return existing == "" || requested == "" || existing == requested
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func (s *Server) createTenantEntitlement(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTenantEntitlementRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.CapabilityID = strings.TrimSpace(req.CapabilityID)
	if req.TenantID == "" || req.TargetID == "" || req.CapabilityID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "tenantId, targetId, and capabilityId are required"))
		return
	}
	effect, err := normalizePolicyEffect(req.Effect, domain.PolicyEffectAllow)
	if err != nil {
		writeError(w, err)
		return
	}
	status, err := normalizePolicyStatus(req.Status, domain.PolicyStatusEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	priority, err := normalizePolicyPriority(req.Priority)
	if err != nil {
		writeError(w, err)
		return
	}
	target, ok, err := s.repo.GetAgent(r.Context(), req.TargetID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return
	}
	allowedTenant, err := s.tenantCanReceiveTargetEntitlement(r.Context(), target.TenantID, req.TenantID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !allowedTenant {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "tenantId must match target tenantId or be a descendant tenant"))
		return
	}
	capability, ok, err := s.repo.GetCapability(r.Context(), req.CapabilityID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok || capability.TargetID != target.ID {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "capabilityId must belong to targetId"))
		return
	}
	if err := s.requireTenantManagementScope(r, req.TenantID); err != nil {
		writeError(w, err)
		return
	}
	if _, ok := domain.EffectiveDataScopes(capability.DataScopes, req.DataScopes); !ok {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "dataScopes must be equal to or narrower than capability dataScopes"))
		return
	}
	now := s.now()
	entitlement := domain.TenantEntitlement{
		ID:           security.NewID("ent"),
		TenantID:     req.TenantID,
		TargetID:     req.TargetID,
		CapabilityID: req.CapabilityID,
		Effect:       effect,
		DataScopes:   req.DataScopes,
		Status:       status,
		Priority:     priority,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	created, err := s.repo.CreateTenantEntitlement(r.Context(), entitlement)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, target.TenantID, target.WorkspaceID, "tenant_entitlement.created", "tenant_entitlement", created.ID, "Tenant entitlement created", map[string]any{
		"targetId":      created.TargetID,
		"capabilityId":  created.CapabilityID,
		"capabilityKey": capability.Key,
		"effect":        created.Effect,
		"status":        created.Status,
		"priority":      created.Priority,
	})); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listTenantEntitlements(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{
		ManagementScope: scope,
		TargetID:        strings.TrimSpace(r.URL.Query().Get("targetId")),
		CapabilityID:    strings.TrimSpace(r.URL.Query().Get("capabilityId")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) createWorkspaceAssignment(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateWorkspaceAssignmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.TenantEntitlementID = strings.TrimSpace(req.TenantEntitlementID)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	if req.TenantEntitlementID == "" || req.WorkspaceID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "tenantEntitlementId and workspaceId are required"))
		return
	}
	effect, err := normalizePolicyEffect(req.Effect, domain.PolicyEffectAllow)
	if err != nil {
		writeError(w, err)
		return
	}
	status, err := normalizePolicyStatus(req.Status, domain.PolicyStatusEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	entitlements, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	entitlement, ok := findTenantEntitlement(entitlements, req.TenantEntitlementID)
	if !ok {
		writeError(w, domain.NotFound("tenant entitlement not found"))
		return
	}
	target, ok, err := s.repo.GetAgent(r.Context(), entitlement.TargetID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return
	}
	entitlementScopes, err := s.effectiveTenantEntitlementDataScopes(r.Context(), entitlement)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, ok := domain.EffectiveDataScopes(entitlementScopes, req.DataScopes); !ok {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "dataScopes must be equal to or narrower than tenant entitlement dataScopes"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: entitlement.TenantID, WorkspaceID: req.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	assignment := domain.WorkspaceAssignment{
		ID:                  security.NewID("wsa"),
		TenantEntitlementID: entitlement.ID,
		TenantID:            entitlement.TenantID,
		WorkspaceID:         req.WorkspaceID,
		Effect:              effect,
		DataScopes:          req.DataScopes,
		Status:              status,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	created, err := s.repo.CreateWorkspaceAssignment(r.Context(), assignment)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "workspace_assignment.created", "workspace_assignment", created.ID, "Workspace assignment created", map[string]any{
		"tenantEntitlementId": created.TenantEntitlementID,
		"targetId":            target.ID,
		"capabilityId":        entitlement.CapabilityID,
		"effect":              created.Effect,
		"status":              created.Status,
	})); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listWorkspaceAssignments(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListWorkspaceAssignments(r.Context(), store.AssignmentFilter{
		ManagementScope: scope,
		EntitlementID:   strings.TrimSpace(r.URL.Query().Get("entitlementId")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) createInstanceAssignment(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateInstanceAssignmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.WorkspaceAssignmentID = strings.TrimSpace(req.WorkspaceAssignmentID)
	req.CallerInstanceID = strings.TrimSpace(req.CallerInstanceID)
	req.SubjectSelector = strings.TrimSpace(req.SubjectSelector)
	if req.WorkspaceAssignmentID == "" || req.CallerInstanceID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "workspaceAssignmentId and callerInstanceId are required"))
		return
	}
	if domain.IsUnboundedSubjectSelector(req.SubjectSelector) {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "subjectSelector is required and cannot be *"))
		return
	}
	effect, err := normalizePolicyEffect(req.Effect, domain.PolicyEffectAllow)
	if err != nil {
		writeError(w, err)
		return
	}
	status, err := normalizePolicyStatus(req.Status, domain.PolicyStatusEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	workspaceAssignments, err := s.repo.ListWorkspaceAssignments(r.Context(), store.AssignmentFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	workspaceAssignment, ok := findWorkspaceAssignment(workspaceAssignments, req.WorkspaceAssignmentID)
	if !ok {
		writeError(w, domain.NotFound("workspace assignment not found"))
		return
	}
	entitlements, err := s.repo.ListTenantEntitlements(r.Context(), store.EntitlementFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	entitlement, ok := findTenantEntitlement(entitlements, workspaceAssignment.TenantEntitlementID)
	if !ok {
		writeError(w, domain.NotFound("tenant entitlement not found"))
		return
	}
	workspaceScopes, err := s.effectiveWorkspaceAssignmentDataScopes(r.Context(), entitlement, workspaceAssignment)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, ok := domain.EffectiveDataScopes(workspaceScopes, req.DataScopes); !ok {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "dataScopes must be equal to or narrower than workspace assignment dataScopes"))
		return
	}
	caller, ok, err := s.repo.GetAgent(r.Context(), req.CallerInstanceID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("caller instance not found"))
		return
	}
	if caller.TenantID != workspaceAssignment.TenantID || caller.WorkspaceID != workspaceAssignment.WorkspaceID {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "caller instance must match workspace assignment tenant and workspace"))
		return
	}
	if err := s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: workspaceAssignment.TenantID, WorkspaceID: workspaceAssignment.WorkspaceID}); err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	assignment := domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              workspaceAssignment.TenantID,
		WorkspaceID:           workspaceAssignment.WorkspaceID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       req.SubjectSelector,
		Effect:                effect,
		DataScopes:            req.DataScopes,
		Status:                status,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	created, err := s.repo.CreateInstanceAssignment(r.Context(), assignment)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.repo.AppendAuditEvent(r.Context(), s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "instance_assignment.created", "instance_assignment", created.ID, "Instance assignment created", map[string]any{
		"workspaceAssignmentId": created.WorkspaceAssignmentID,
		"callerInstanceId":      created.CallerInstanceID,
		"effect":                created.Effect,
		"status":                created.Status,
	})); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listInstanceAssignments(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListInstanceAssignments(r.Context(), store.InstanceAssignmentFilter{
		ManagementScope:  scope,
		CallerInstanceID: strings.TrimSpace(r.URL.Query().Get("callerInstanceId")),
		CapabilityID:     strings.TrimSpace(r.URL.Query().Get("capabilityId")),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) requireAgentKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := security.BearerToken(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, domain.Unauthorized("missing bearer token"))
			return
		}
		caller, ok, err := s.repo.FindAgentByKeyHash(r.Context(), security.HashSecret(token), s.now())
		if err != nil {
			writeError(w, err)
			return
		}
		if !ok {
			writeError(w, domain.Unauthorized("invalid or expired bearer token"))
			return
		}
		ctx := context.WithValue(r.Context(), callerContextKey{}, caller)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func callerFromContext(ctx context.Context) domain.Agent {
	caller, _ := ctx.Value(callerContextKey{}).(domain.Agent)
	return caller
}

func (s *Server) mcpRPC(w http.ResponseWriter, r *http.Request) {
	info, err := mcpRequestInfoFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(info.Body))
	if info.Method == "tools/list" {
		if s.handleMCPToolsList(w, r, info) {
			return
		}
	}
	if info.Method == "tools/call" && info.ToolName != "" {
		if s.handleMCPToolCall(w, r, info) {
			return
		}
	}
	s.handleDataPlane(w, r, "mcp", info.Method)
}

func (s *Server) openapiOperation(w http.ResponseWriter, r *http.Request) {
	s.handleDataPlane(w, r, "openapi", chi.URLParam(r, "operationId"))
}

func (s *Server) openapiRelativePath(w http.ResponseWriter, r *http.Request) {
	relativePath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if relativePath == "" || strings.Contains(relativePath, "..") || strings.Contains(relativePath, "://") {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "openapi relative path is invalid"))
		return
	}
	s.handleDataPlane(w, r, "openapi", relativePath)
}

func (s *Server) handleDataPlane(w http.ResponseWriter, r *http.Request, routeType string, routeKey string) {
	caller := callerFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	decision, err := s.repo.EvaluateRouteAccess(r.Context(), caller.ID, targetID, routeType, routeKey, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	if !decision.Allowed {
		reason := decision.Reason
		if reason == "" {
			reason = "caller has no route policy or access grant for target route"
		}
		if _, err := s.recordDataPlaneTrace(r, caller.ID, targetID, routeType, routeKey, domain.TraceDecisionDenied, reason, proxyTraceResult{}); err != nil {
			writeError(w, err)
			return
		}
		writeError(w, domain.PermissionDenied(reason))
		return
	}
	target, ok, err := s.repo.GetAgent(r.Context(), targetID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return
	}
	if target.Status != domain.AgentStatusActive {
		reason := "target agent is not active"
		if _, err := s.recordDataPlaneTrace(r, caller.ID, targetID, routeType, routeKey, domain.TraceDecisionDenied, reason, proxyTraceResult{}); err != nil {
			writeError(w, err)
			return
		}
		writeError(w, domain.PermissionDenied(reason))
		return
	}
	allowedReason := decision.Reason
	if allowedReason == "" {
		allowedReason = "access grant matched"
	}
	recordAllowedTrace := func(result proxyTraceResult) (domain.TraceEvent, error) {
		return s.recordDataPlaneTrace(r, caller.ID, targetID, routeType, routeKey, domain.TraceDecisionAllowed, allowedReason, result)
	}
	if s.proxyUpstreamIfConfigured(w, r, target, routeType, routeKey, decision.Retry, recordAllowedTrace, nil) {
		return
	}
	trace, err := recordAllowedTrace(proxyTraceResult{})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "accepted",
		"traceId": trace.ID,
		"route":   routeType,
	})
}

type runtimeIdentity struct {
	PlatformID       string
	TenantID         string
	WorkspaceID      string
	CallerInstanceID string
	SubjectID        string
}

type traceRecordInput struct {
	Identity           runtimeIdentity
	CallerID           string
	TargetID           string
	RouteType          string
	RouteKey           string
	Decision           domain.TraceDecision
	Reason             string
	Capability         domain.Capability
	CapabilityDecision domain.CapabilityAccessDecision
	ProxyResult        proxyTraceResult
}

func identityFromRequest(r *http.Request, caller domain.Agent) runtimeIdentity {
	return runtimeIdentity{
		PlatformID:       "default",
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		SubjectID:        strings.TrimSpace(r.Header.Get("X-AgentHarbor-Subject-Id")),
	}
}

func (s *Server) handleMCPToolCall(w http.ResponseWriter, r *http.Request, info mcpRequestInfo) bool {
	caller := callerFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	capability, found, hasCatalog, err := s.mcpToolCapability(r.Context(), targetID, info.ToolName)
	if err != nil {
		writeError(w, err)
		return true
	}
	if !hasCatalog {
		return false
	}
	identity := identityFromRequest(r, caller)
	if !found {
		reason := "capability is not registered for target"
		if _, err := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:  identity,
			CallerID:  caller.ID,
			TargetID:  targetID,
			RouteType: "mcp",
			RouteKey:  info.Method,
			Decision:  domain.TraceDecisionDenied,
			Reason:    reason,
		}); err != nil {
			writeError(w, err)
			return true
		}
		writeError(w, domain.PermissionDenied(reason))
		return true
	}
	decision, err := s.repo.EvaluateCapabilityAccess(r.Context(), store.CapabilityAccessRequest{
		TenantID:         identity.TenantID,
		WorkspaceID:      identity.WorkspaceID,
		CallerInstanceID: identity.CallerInstanceID,
		SubjectID:        identity.SubjectID,
		TargetID:         targetID,
		CapabilityID:     capability.ID,
		Now:              s.now(),
	})
	if err != nil {
		writeError(w, err)
		return true
	}
	if !decision.Allowed {
		if _, err := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:           identity,
			CallerID:           caller.ID,
			TargetID:           targetID,
			RouteType:          "mcp",
			RouteKey:           info.Method,
			Decision:           domain.TraceDecisionDenied,
			Reason:             decision.Reason,
			Capability:         capability,
			CapabilityDecision: decision,
		}); err != nil {
			writeError(w, err)
			return true
		}
		writeError(w, domain.PermissionDenied(decision.Reason))
		return true
	}
	target, ok, err := s.repo.GetAgent(r.Context(), targetID)
	if err != nil {
		writeError(w, err)
		return true
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return true
	}
	if target.Status != domain.AgentStatusActive {
		reason := "target agent is not active"
		if _, err := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:           identity,
			CallerID:           caller.ID,
			TargetID:           targetID,
			RouteType:          "mcp",
			RouteKey:           info.Method,
			Decision:           domain.TraceDecisionDenied,
			Reason:             reason,
			Capability:         capability,
			CapabilityDecision: decision,
		}); err != nil {
			writeError(w, err)
			return true
		}
		writeError(w, domain.PermissionDenied(reason))
		return true
	}
	recordAllowedTrace := func(result proxyTraceResult) (domain.TraceEvent, error) {
		return s.recordCapabilityTrace(r, traceRecordInput{
			Identity:           identity,
			CallerID:           caller.ID,
			TargetID:           target.ID,
			RouteType:          "mcp",
			RouteKey:           info.Method,
			Decision:           domain.TraceDecisionAllowed,
			Reason:             decision.Reason,
			Capability:         capability,
			CapabilityDecision: decision,
			ProxyResult:        result,
		})
	}
	contextHeader, err := agentHarborContextHeaderValue(identity, target.ID, capability, decision, info.ToolName)
	if err != nil {
		writeError(w, err)
		return true
	}
	r.Body = io.NopCloser(bytes.NewReader(info.Body))
	if s.proxyUpstreamIfConfigured(w, r, target, "mcp", info.Method, nil, recordAllowedTrace, func(req *http.Request) error {
		req.Header.Set(agentHarborContextHeader, contextHeader)
		return nil
	}) {
		return true
	}
	trace, err := recordAllowedTrace(proxyTraceResult{})
	if err != nil {
		writeError(w, err)
		return true
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "accepted",
		"traceId":      trace.ID,
		"route":        "mcp",
		"capabilityId": capability.ID,
	})
	return true
}

func (s *Server) handleMCPToolsList(w http.ResponseWriter, r *http.Request, info mcpRequestInfo) bool {
	caller := callerFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	capabilities, err := s.repo.ListCapabilities(r.Context(), store.CapabilityFilter{TargetID: targetID})
	if err != nil {
		writeError(w, err)
		return true
	}
	if len(capabilities) == 0 {
		return false
	}
	target, ok, err := s.repo.GetAgent(r.Context(), targetID)
	if err != nil {
		writeError(w, err)
		return true
	}
	if !ok {
		writeError(w, domain.NotFound("target agent not found"))
		return true
	}
	if target.Status != domain.AgentStatusActive {
		reason := "target agent is not active"
		if _, err := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:  identityFromRequest(r, caller),
			CallerID:  caller.ID,
			TargetID:  targetID,
			RouteType: "mcp",
			RouteKey:  info.Method,
			Decision:  domain.TraceDecisionDenied,
			Reason:    reason,
		}); err != nil {
			writeError(w, err)
			return true
		}
		writeError(w, domain.PermissionDenied(reason))
		return true
	}
	identity := identityFromRequest(r, caller)
	allowedTools := map[string]domain.Capability{}
	for _, capability := range capabilities {
		if capability.Type != domain.CapabilityTypeMCPTool {
			continue
		}
		decision, err := s.repo.EvaluateCapabilityAccess(r.Context(), store.CapabilityAccessRequest{
			TenantID:         identity.TenantID,
			WorkspaceID:      identity.WorkspaceID,
			CallerInstanceID: identity.CallerInstanceID,
			SubjectID:        identity.SubjectID,
			TargetID:         targetID,
			CapabilityID:     capability.ID,
			Now:              s.now(),
		})
		if err != nil {
			writeError(w, err)
			return true
		}
		if decision.Allowed {
			allowedTools[capability.Key] = capability
		}
	}
	if endpoint, _ := target.ChannelConfig["endpoint"].(string); strings.TrimSpace(endpoint) != "" {
		body, statusCode, contentType, result, err := s.callMCPUpstream(r, target, info.Body)
		if err != nil {
			if _, recordErr := s.recordCapabilityTrace(r, traceRecordInput{
				Identity:    identity,
				CallerID:    caller.ID,
				TargetID:    targetID,
				RouteType:   "mcp",
				RouteKey:    info.Method,
				Decision:    domain.TraceDecisionAllowed,
				Reason:      "filtered tools/list by capability assignments",
				ProxyResult: result,
			}); recordErr != nil {
				writeError(w, recordErr)
				return true
			}
			writeError(w, err)
			return true
		}
		filtered, err := filterMCPToolsListBody(body, allowedTools)
		if err != nil {
			writeError(w, err)
			return true
		}
		if _, err := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:    identity,
			CallerID:    caller.ID,
			TargetID:    targetID,
			RouteType:   "mcp",
			RouteKey:    info.Method,
			Decision:    domain.TraceDecisionAllowed,
			Reason:      "filtered tools/list by capability assignments",
			ProxyResult: result,
		}); err != nil {
			writeError(w, err)
			return true
		}
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(statusCode)
		_, _ = w.Write(filtered)
		return true
	}
	if _, err := s.recordCapabilityTrace(r, traceRecordInput{
		Identity:  identity,
		CallerID:  caller.ID,
		TargetID:  targetID,
		RouteType: "mcp",
		RouteKey:  info.Method,
		Decision:  domain.TraceDecisionAllowed,
		Reason:    "filtered tools/list by capability assignments",
	}); err != nil {
		writeError(w, err)
		return true
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"jsonrpc": "2.0",
		"id":      requestJSONRPCID(info.Body),
		"result": map[string]any{
			"tools": capabilitiesForToolsList(allowedTools),
		},
	})
	return true
}

func (s *Server) recordDataPlaneTrace(r *http.Request, callerID string, targetID string, routeType string, routeKey string, decision domain.TraceDecision, reason string, result proxyTraceResult) (domain.TraceEvent, error) {
	trace := domain.TraceEvent{
		ID:               security.NewID("trc"),
		RunID:            r.Header.Get("X-Run-Id"),
		CallerID:         callerID,
		TargetID:         targetID,
		RouteType:        routeType,
		RouteKey:         routeKey,
		Decision:         decision,
		Reason:           reason,
		DurationMs:       result.durationMs,
		UpstreamAttempts: result.upstreamAttempts,
		UpstreamStatus:   result.upstreamStatus,
		UpstreamError:    result.upstreamError,
		CreatedAt:        s.now(),
	}
	return s.repo.AppendTrace(r.Context(), trace)
}

func (s *Server) recordCapabilityTrace(r *http.Request, input traceRecordInput) (domain.TraceEvent, error) {
	trace := domain.TraceEvent{
		ID:                    security.NewID("trc"),
		RunID:                 r.Header.Get("X-Run-Id"),
		CallerID:              input.CallerID,
		TargetID:              input.TargetID,
		RouteType:             input.RouteType,
		RouteKey:              input.RouteKey,
		TenantID:              input.Identity.TenantID,
		WorkspaceID:           input.Identity.WorkspaceID,
		CallerInstanceID:      input.Identity.CallerInstanceID,
		SubjectID:             input.Identity.SubjectID,
		CapabilityID:          input.Capability.ID,
		CapabilityVersion:     input.Capability.Version,
		EntitlementID:         input.CapabilityDecision.EntitlementID,
		WorkspaceAssignmentID: input.CapabilityDecision.WorkspaceAssignmentID,
		InstanceAssignmentID:  input.CapabilityDecision.InstanceAssignmentID,
		DataScopes:            input.CapabilityDecision.DataScopes,
		Decision:              input.Decision,
		Reason:                input.Reason,
		DurationMs:            input.ProxyResult.durationMs,
		UpstreamAttempts:      input.ProxyResult.upstreamAttempts,
		UpstreamStatus:        input.ProxyResult.upstreamStatus,
		UpstreamError:         input.ProxyResult.upstreamError,
		CreatedAt:             s.now(),
	}
	return s.repo.AppendTrace(r.Context(), trace)
}

func (s *Server) proxyUpstreamIfConfigured(w http.ResponseWriter, r *http.Request, target domain.Agent, routeType string, routeKey string, retryOverride *domain.RoutePolicyRetry, recordAllowedTrace func(proxyTraceResult) (domain.TraceEvent, error), mutateRequest upstreamRequestMutator) bool {
	endpoint, ok := target.ChannelConfig["endpoint"].(string)
	endpoint = strings.TrimSpace(endpoint)
	if !ok || endpoint == "" {
		return false
	}
	startedAt := time.Now()
	upstreamURL := endpoint
	method := http.MethodPost
	if routeType == "openapi" {
		var err error
		upstreamURL, err = openAPIUpstreamURL(endpoint, routeKey, r.URL.RawQuery)
		if err != nil {
			writeProxyError(w, recordAllowedTrace, startedAt, 0, err)
			return true
		}
		method = r.Method
	}
	timeout, err := proxyTimeoutFromConfig(target.ChannelConfig)
	if err != nil {
		writeProxyError(w, recordAllowedTrace, startedAt, 0, err)
		return true
	}
	retryPolicy := proxyRetryPolicyFromRoutePolicyRetry(retryOverride)
	if retryPolicy == nil {
		parsedRetryPolicy, err := proxyRetryPolicyFromConfig(target.ChannelConfig)
		if err != nil {
			writeProxyError(w, recordAllowedTrace, startedAt, 0, err)
			return true
		}
		retryPolicy = &parsedRetryPolicy
	}
	body, err := readProxyBody(r.Body)
	if err != nil {
		writeProxyError(w, recordAllowedTrace, startedAt, 0, err)
		return true
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	for attempt := 1; attempt <= retryPolicy.maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, upstreamURL, bytes.NewReader(body))
		if err != nil {
			writeProxyError(w, recordAllowedTrace, startedAt, 0, domain.UpstreamError("upstream request could not be prepared"))
			return true
		}
		copyUpstreamRequestHeaders(req.Header, r.Header)
		if err := copyConfiguredHeaders(req.Header, target.ChannelConfig); err != nil {
			writeProxyError(w, recordAllowedTrace, startedAt, 0, err)
			return true
		}
		if err := copyCredentialHeaders(req.Header, target.ChannelConfig, target.Credentials); err != nil {
			writeProxyError(w, recordAllowedTrace, startedAt, 0, err)
			return true
		}
		if routeType == "mcp" {
			setMCPUpstreamHeaders(req.Header)
		}
		if mutateRequest != nil {
			if err := mutateRequest(req); err != nil {
				writeProxyError(w, recordAllowedTrace, startedAt, 0, err)
				return true
			}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if shouldRetryUpstreamError(ctx, err) && attempt < retryPolicy.maxAttempts {
				if !sleepBeforeRetry(ctx, retryPolicy.backoff) {
					w.Header().Set("X-AgentHarbor-Upstream-Attempts", strconv.Itoa(attempt))
					writeProxyError(w, recordAllowedTrace, startedAt, attempt, domain.UpstreamTimeout("upstream request timed out"))
					return true
				}
				continue
			}
			w.Header().Set("X-AgentHarbor-Upstream-Attempts", strconv.Itoa(attempt))
			writeProxyError(w, recordAllowedTrace, startedAt, attempt, classifyUpstreamError(ctx, err))
			return true
		}
		if retryPolicy.shouldRetryStatus(resp.StatusCode) && attempt < retryPolicy.maxAttempts {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if !sleepBeforeRetry(ctx, retryPolicy.backoff) {
				w.Header().Set("X-AgentHarbor-Upstream-Attempts", strconv.Itoa(attempt))
				writeProxyError(w, recordAllowedTrace, startedAt, attempt, domain.UpstreamTimeout("upstream request timed out"))
				return true
			}
			continue
		}
		if _, err := recordAllowedTrace(proxyTraceResult{
			durationMs:       elapsedProxyDurationMs(startedAt),
			upstreamAttempts: attempt,
			upstreamStatus:   resp.StatusCode,
		}); err != nil {
			// Upstream has already completed. Preserve its response to avoid
			// encouraging callers to retry non-idempotent operations.
		}
		defer resp.Body.Close()
		if contentType := resp.Header.Get("Content-Type"); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.Header().Set("X-AgentHarbor-Upstream-Attempts", strconv.Itoa(attempt))
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		return true
	}
	writeProxyError(w, recordAllowedTrace, startedAt, retryPolicy.maxAttempts, domain.UpstreamError("upstream retry policy exhausted unexpectedly"))
	return true
}

func writeProxyError(w http.ResponseWriter, recordAllowedTrace func(proxyTraceResult) (domain.TraceEvent, error), startedAt time.Time, attempts int, err error) {
	result := proxyTraceResult{
		durationMs:       elapsedProxyDurationMs(startedAt),
		upstreamAttempts: attempts,
	}
	var appErr domain.AppError
	if errors.As(err, &appErr) && strings.HasPrefix(appErr.Code, "UPSTREAM_") {
		result.upstreamError = appErr.Code
	}
	if _, recordErr := recordAllowedTrace(result); recordErr != nil {
		writeError(w, recordErr)
		return
	}
	writeError(w, err)
}

func elapsedProxyDurationMs(startedAt time.Time) int64 {
	elapsed := time.Since(startedAt).Milliseconds()
	if elapsed <= 0 {
		return 1
	}
	return elapsed
}

func openAPIUpstreamURL(endpoint string, relativePath string, rawQuery string) (string, error) {
	if relativePath == "" || strings.Contains(relativePath, "..") || strings.Contains(relativePath, "://") {
		return "", domain.BadRequest("VALIDATION_FAILED", "openapi relative path is invalid")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", domain.UpstreamError("target endpoint is invalid")
	}
	parsed.Path = path.Join(parsed.Path, relativePath)
	if !strings.HasPrefix(parsed.Path, "/") {
		parsed.Path = "/" + parsed.Path
	}
	parsed.RawQuery = rawQuery
	return parsed.String(), nil
}

func copyUpstreamRequestHeaders(dst http.Header, src http.Header) {
	for _, key := range []string{"Content-Type", "Accept"} {
		if isReservedAgentHarborHeader(key) {
			continue
		}
		if value := src.Get(key); value != "" {
			dst.Set(key, value)
		}
	}
}

const mcpStreamableHTTPAccept = "application/json, text/event-stream"

func setMCPUpstreamHeaders(header http.Header) {
	header.Set("Accept", mcpStreamableHTTPAccept)
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
}

func readProxyBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, maxProxyBodyBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, domain.BadRequest("VALIDATION_FAILED", "proxy request body could not be read")
	}
	if len(payload) > maxProxyBodyBytes {
		return nil, domain.PayloadTooLarge("proxy request body exceeds 4MiB")
	}
	return payload, nil
}

type mcpRequestInfo struct {
	Method   string
	ToolName string
	Body     []byte
}

func mcpRequestInfoFromRequest(r *http.Request) (mcpRequestInfo, error) {
	body, err := readProxyBody(r.Body)
	if err != nil {
		return mcpRequestInfo{}, err
	}
	var payload struct {
		Method *string `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return mcpRequestInfo{}, domain.BadRequest("VALIDATION_FAILED", "mcp request body must be valid JSON")
	}
	if payload.Method == nil || strings.TrimSpace(*payload.Method) == "" {
		return mcpRequestInfo{}, domain.BadRequest("VALIDATION_FAILED", "mcp request method is required")
	}
	return mcpRequestInfo{
		Method:   strings.TrimSpace(*payload.Method),
		ToolName: strings.TrimSpace(payload.Params.Name),
		Body:     body,
	}, nil
}

func (s *Server) mcpToolCapability(ctx context.Context, targetID string, toolName string) (domain.Capability, bool, bool, error) {
	capabilities, err := s.repo.ListCapabilities(ctx, store.CapabilityFilter{TargetID: targetID})
	if err != nil {
		return domain.Capability{}, false, false, err
	}
	hasCatalog := false
	for _, capability := range capabilities {
		if capability.Type != domain.CapabilityTypeMCPTool {
			continue
		}
		hasCatalog = true
		if capability.Key == toolName {
			return capability, true, true, nil
		}
	}
	return domain.Capability{}, false, hasCatalog, nil
}

func (s *Server) callMCPUpstream(r *http.Request, target domain.Agent, body []byte) ([]byte, int, string, proxyTraceResult, error) {
	startedAt := time.Now()
	endpoint, ok := target.ChannelConfig["endpoint"].(string)
	endpoint = strings.TrimSpace(endpoint)
	if !ok || endpoint == "" {
		return nil, 0, "", proxyTraceResult{}, domain.UpstreamError("target endpoint is missing")
	}
	timeout, err := proxyTimeoutFromConfig(target.ChannelConfig)
	if err != nil {
		return nil, 0, "", proxyTraceResult{}, err
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, "", proxyTraceResult{}, domain.UpstreamError("upstream request could not be prepared")
	}
	copyUpstreamRequestHeaders(req.Header, r.Header)
	if err := copyConfiguredHeaders(req.Header, target.ChannelConfig); err != nil {
		return nil, 0, "", proxyTraceResult{}, err
	}
	if err := copyCredentialHeaders(req.Header, target.ChannelConfig, target.Credentials); err != nil {
		return nil, 0, "", proxyTraceResult{}, err
	}
	setMCPUpstreamHeaders(req.Header)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		classified := classifyUpstreamError(ctx, err)
		result := proxyTraceResult{
			durationMs:       elapsedProxyDurationMs(startedAt),
			upstreamAttempts: 1,
		}
		var appErr domain.AppError
		if errors.As(classified, &appErr) {
			result.upstreamError = appErr.Code
		}
		return nil, 0, "", result, classified
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxProxyBodyBytes+1))
	if err != nil {
		return nil, 0, "", proxyTraceResult{}, domain.UpstreamError("upstream response could not be read")
	}
	if len(payload) > maxProxyBodyBytes {
		return nil, 0, "", proxyTraceResult{}, domain.PayloadTooLarge("upstream response exceeds 4MiB")
	}
	return payload, resp.StatusCode, resp.Header.Get("Content-Type"), proxyTraceResult{
		durationMs:       elapsedProxyDurationMs(startedAt),
		upstreamAttempts: 1,
		upstreamStatus:   resp.StatusCode,
	}, nil
}

func filterMCPToolsListBody(body []byte, allowed map[string]domain.Capability) ([]byte, error) {
	var payload struct {
		JSONRPC string          `json:"jsonrpc,omitempty"`
		ID      any             `json:"id,omitempty"`
		Result  json.RawMessage `json:"result"`
		Error   any             `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, domain.BadRequest("VALIDATION_FAILED", "mcp tools/list response must be valid JSON")
	}
	if payload.Error != nil {
		return body, nil
	}
	var result struct {
		Tools []map[string]any `json:"tools"`
	}
	if err := json.Unmarshal(payload.Result, &result); err != nil {
		return nil, domain.BadRequest("VALIDATION_FAILED", "mcp tools/list result must include tools")
	}
	filtered := make([]map[string]any, 0, len(result.Tools))
	for _, tool := range result.Tools {
		name, _ := tool["name"].(string)
		if _, ok := allowed[name]; ok {
			filtered = append(filtered, tool)
		}
	}
	result.Tools = filtered
	payload.Result = nil
	out := map[string]any{
		"jsonrpc": payload.JSONRPC,
		"id":      payload.ID,
		"result":  result,
	}
	if out["jsonrpc"] == "" {
		delete(out, "jsonrpc")
	}
	if out["id"] == nil {
		delete(out, "id")
	}
	return json.Marshal(out)
}

func requestJSONRPCID(body []byte) any {
	var payload struct {
		ID any `json:"id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return payload.ID
}

func capabilitiesForToolsList(capabilities map[string]domain.Capability) []map[string]any {
	keys := make([]string, 0, len(capabilities))
	for key := range capabilities {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	tools := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		capability := capabilities[key]
		tool := map[string]any{
			"name":        capability.Key,
			"description": capability.Description,
			"inputSchema": capability.InputSchema,
		}
		if capability.DisplayName != "" {
			tool["title"] = capability.DisplayName
		}
		tools = append(tools, tool)
	}
	return tools
}

func validateConfiguredHeaders(config map[string]any) error {
	raw, exists := config["headers"]
	if !exists {
		return nil
	}
	headers, ok := raw.(map[string]any)
	if !ok {
		return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers must be an object")
	}
	for name, value := range headers {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers names must be non-empty")
		}
		if !validHeaderName(trimmedName) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers names must be valid HTTP header names")
		}
		if security.IsSecretLikeKey(trimmedName) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers must not contain secret-like names")
		}
		headerValue, ok := value.(string)
		if !ok {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers values must be strings")
		}
		if containsHeaderNewline(headerValue) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers values must not contain CR or LF")
		}
	}
	return nil
}

func normalizeCredentials(credentials map[string]string) (map[string]string, error) {
	if len(credentials) == 0 {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(credentials))
	for key, value := range credentials {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, domain.BadRequest("VALIDATION_FAILED", "credentials keys must be non-empty")
		}
		if !validCredentialKey(trimmedKey) {
			return nil, domain.BadRequest("VALIDATION_FAILED", "credentials keys must be 1-64 character identifiers")
		}
		if strings.TrimSpace(value) == "" {
			return nil, domain.BadRequest("VALIDATION_FAILED", "credentials values must be non-empty strings")
		}
		if containsHeaderNewline(value) {
			return nil, domain.BadRequest("VALIDATION_FAILED", "credentials values must not contain CR or LF")
		}
		if _, exists := out[trimmedKey]; exists {
			return nil, domain.BadRequest("VALIDATION_FAILED", "credentials keys must be unique after trimming")
		}
		out[trimmedKey] = value
	}
	return out, nil
}

func validateCredentialHeaders(config map[string]any, credentials map[string]string) error {
	raw, exists := config["credentialHeaders"]
	if !exists {
		return nil
	}
	headers, ok := raw.(map[string]any)
	if !ok {
		return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders must be an object")
	}
	for headerName, credentialKey := range headers {
		trimmedHeaderName := strings.TrimSpace(headerName)
		if trimmedHeaderName == "" {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders names must be non-empty")
		}
		if !validHeaderName(trimmedHeaderName) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders names must be valid HTTP header names")
		}
		key, ok := credentialKey.(string)
		if !ok {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders values must be credential key strings")
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders values must be credential key strings")
		}
		if !validCredentialKey(key) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders values must be credential key identifiers")
		}
		if _, exists := credentials[key]; !exists {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders references missing credentials")
		}
	}
	return nil
}

func channelConfigContainsSecretLikeKey(config map[string]any) bool {
	return channelConfigContainsSecretLikeKeyAt(config, true)
}

func channelConfigContainsSecretLikeKeyAt(config map[string]any, allowCredentialHeaders bool) bool {
	for key, nested := range config {
		if allowCredentialHeaders && key == "credentialHeaders" {
			continue
		}
		if security.IsSecretLikeKey(key) {
			return true
		}
		if channelConfigValueContainsSecretLikeKey(nested, false) {
			return true
		}
	}
	return false
}

func channelConfigValueContainsSecretLikeKey(value any, allowCredentialHeaders bool) bool {
	switch typed := value.(type) {
	case map[string]any:
		return channelConfigContainsSecretLikeKeyAt(typed, allowCredentialHeaders)
	case []any:
		for _, item := range typed {
			if channelConfigValueContainsSecretLikeKey(item, allowCredentialHeaders) {
				return true
			}
		}
	}
	return false
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r < 33 || r > 126 {
			return false
		}
		switch r {
		case '(', ')', '<', '>', '@', ',', ';', '\\', '"', '/', '[', ']', '?', '=', '{', '}', ':':
			return false
		}
	}
	return true
}

func containsHeaderNewline(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}

func validCredentialKey(key string) bool {
	if key == "" || len(key) > 64 {
		return false
	}
	for i, r := range key {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			continue
		}
		if i > 0 && (r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			continue
		}
		return false
	}
	return true
}

func copyConfiguredHeaders(dst http.Header, config map[string]any) error {
	raw, exists := config["headers"]
	if !exists {
		return nil
	}
	headers, ok := raw.(map[string]any)
	if !ok {
		return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers must be an object")
	}
	for name, value := range headers {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" || !validHeaderName(trimmedName) || security.IsSecretLikeKey(trimmedName) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers contains invalid header name")
		}
		if isReservedAgentHarborHeader(trimmedName) {
			continue
		}
		headerValue, ok := value.(string)
		if !ok {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers values must be strings")
		}
		if containsHeaderNewline(headerValue) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers values must not contain CR or LF")
		}
		dst.Set(trimmedName, headerValue)
	}
	return nil
}

func copyCredentialHeaders(dst http.Header, config map[string]any, credentials map[string]string) error {
	raw, exists := config["credentialHeaders"]
	if !exists {
		return nil
	}
	headers, ok := raw.(map[string]any)
	if !ok {
		return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders must be an object")
	}
	for name, credentialKey := range headers {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" || !validHeaderName(trimmedName) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders references missing credentials")
		}
		if isReservedAgentHarborHeader(trimmedName) {
			continue
		}
		key, ok := credentialKey.(string)
		if !ok {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders values must be credential key strings")
		}
		key = strings.TrimSpace(key)
		value, exists := credentials[key]
		if key == "" || !validCredentialKey(key) || !exists {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders references missing credentials")
		}
		if containsHeaderNewline(value) {
			return domain.BadRequest("VALIDATION_FAILED", "credentials values must not contain CR or LF")
		}
		dst.Set(trimmedName, value)
	}
	return nil
}

func proxyTimeoutFromConfig(config map[string]any) (time.Duration, error) {
	raw, exists := config["timeoutMs"]
	if !exists {
		return defaultProxyTimeout, nil
	}
	var timeoutMs int64
	switch value := raw.(type) {
	case int:
		timeoutMs = int64(value)
	case int64:
		timeoutMs = value
	case float64:
		timeoutMs = int64(value)
		if value != float64(timeoutMs) {
			return 0, domain.BadRequest("VALIDATION_FAILED", "channelConfig.timeoutMs must be an integer")
		}
	default:
		return 0, domain.BadRequest("VALIDATION_FAILED", "channelConfig.timeoutMs must be an integer")
	}
	if timeoutMs < 1 || timeoutMs > int64(maxProxyTimeout/time.Millisecond) {
		return 0, domain.BadRequest("VALIDATION_FAILED", "channelConfig.timeoutMs must be between 1 and 30000")
	}
	return time.Duration(timeoutMs) * time.Millisecond, nil
}

func proxyRetryPolicyFromConfig(config map[string]any) (proxyRetryPolicy, error) {
	policy := proxyRetryPolicy{
		maxAttempts:      1,
		retryStatusCodes: defaultRetryStatusCodes(),
	}
	raw, exists := config["retry"]
	if !exists {
		return policy, nil
	}
	retry, ok := raw.(map[string]any)
	if !ok {
		return proxyRetryPolicy{}, domain.BadRequest("VALIDATION_FAILED", "channelConfig.retry must be an object")
	}
	if rawMaxAttempts, exists := retry["maxAttempts"]; exists {
		maxAttempts, err := configInteger(rawMaxAttempts, "channelConfig.retry.maxAttempts")
		if err != nil {
			return proxyRetryPolicy{}, err
		}
		if maxAttempts < 1 || maxAttempts > maxRetryAttempts {
			return proxyRetryPolicy{}, domain.BadRequest("VALIDATION_FAILED", "channelConfig.retry.maxAttempts must be between 1 and 4")
		}
		policy.maxAttempts = int(maxAttempts)
	}
	if rawBackoff, exists := retry["backoffMs"]; exists {
		backoffMs, err := configInteger(rawBackoff, "channelConfig.retry.backoffMs")
		if err != nil {
			return proxyRetryPolicy{}, err
		}
		if backoffMs < 0 || backoffMs > int64(maxRetryBackoff/time.Millisecond) {
			return proxyRetryPolicy{}, domain.BadRequest("VALIDATION_FAILED", "channelConfig.retry.backoffMs must be between 0 and 1000")
		}
		policy.backoff = time.Duration(backoffMs) * time.Millisecond
	}
	if rawStatusCodes, exists := retry["statusCodes"]; exists {
		values, ok := rawStatusCodes.([]any)
		if !ok {
			return proxyRetryPolicy{}, domain.BadRequest("VALIDATION_FAILED", "channelConfig.retry.statusCodes must be an array")
		}
		statusCodes := make(map[int]struct{}, len(values))
		for _, rawStatusCode := range values {
			statusCode, err := configInteger(rawStatusCode, "channelConfig.retry.statusCodes")
			if err != nil {
				return proxyRetryPolicy{}, err
			}
			if statusCode < 500 || statusCode > 599 {
				return proxyRetryPolicy{}, domain.BadRequest("VALIDATION_FAILED", "channelConfig.retry.statusCodes must contain 5xx status codes")
			}
			statusCodes[int(statusCode)] = struct{}{}
		}
		policy.retryStatusCodes = statusCodes
	}
	return policy, nil
}

func proxyRetryPolicyFromRoutePolicyRetry(retry *domain.RoutePolicyRetry) *proxyRetryPolicy {
	if retry == nil {
		return nil
	}
	statusCodes := make(map[int]struct{}, len(retry.StatusCodes))
	for _, statusCode := range retry.StatusCodes {
		statusCodes[statusCode] = struct{}{}
	}
	return &proxyRetryPolicy{
		maxAttempts:      retry.MaxAttempts,
		backoff:          time.Duration(retry.BackoffMs) * time.Millisecond,
		retryStatusCodes: statusCodes,
	}
}

func defaultRetryStatusCodes() map[int]struct{} {
	return map[int]struct{}{
		http.StatusBadGateway:         {},
		http.StatusServiceUnavailable: {},
		http.StatusGatewayTimeout:     {},
	}
}

func configInteger(raw any, field string) (int64, error) {
	switch value := raw.(type) {
	case int:
		return int64(value), nil
	case int64:
		return value, nil
	case float64:
		integer := int64(value)
		if value != float64(integer) {
			return 0, domain.BadRequest("VALIDATION_FAILED", field+" must be an integer")
		}
		return integer, nil
	default:
		return 0, domain.BadRequest("VALIDATION_FAILED", field+" must be an integer")
	}
}

func (p proxyRetryPolicy) shouldRetryStatus(statusCode int) bool {
	_, ok := p.retryStatusCodes[statusCode]
	return ok
}

func shouldRetryUpstreamError(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	return true
}

func classifyUpstreamError(ctx context.Context, err error) domain.AppError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return domain.UpstreamTimeout("upstream request timed out")
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return domain.UpstreamError("upstream request canceled")
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return domain.UpstreamDNSError("upstream DNS lookup failed")
	}
	var unknownAuthority x509.UnknownAuthorityError
	var certificateInvalid x509.CertificateInvalidError
	var hostnameInvalid x509.HostnameError
	var tlsRecord tls.RecordHeaderError
	if errors.As(err, &unknownAuthority) ||
		errors.As(err, &certificateInvalid) ||
		errors.As(err, &hostnameInvalid) ||
		errors.As(err, &tlsRecord) ||
		strings.Contains(strings.ToLower(err.Error()), "tls:") {
		return domain.UpstreamTLSError("upstream TLS handshake failed")
	}
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE) ||
		strings.Contains(strings.ToLower(err.Error()), "connection refused") ||
		strings.Contains(strings.ToLower(err.Error()), "connection reset") {
		return domain.UpstreamConnectError("upstream connection failed")
	}
	return domain.UpstreamError("upstream request failed")
}

func sleepBeforeRetry(ctx context.Context, backoff time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	if backoff <= 0 {
		return true
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *Server) listTraces(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter := store.TraceFilter{
		ManagementScope: scope,
		RunID:           r.URL.Query().Get("runId"),
		CallerID:        r.URL.Query().Get("callerAgentId"),
		TargetID:        r.URL.Query().Get("targetAgentId"),
	}
	switch decision := domain.TraceDecision(r.URL.Query().Get("decision")); decision {
	case "", domain.TraceDecisionAllowed, domain.TraceDecisionDenied:
		filter.Decision = decision
	default:
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "decision must be allowed or denied"))
		return
	}
	rows, err := s.repo.ListTraces(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter := store.AuditEventFilter{
		ManagementScope: scope,
		Action:          strings.TrimSpace(r.URL.Query().Get("action")),
		ResourceType:    strings.TrimSpace(r.URL.Query().Get("resourceType")),
		ResourceID:      strings.TrimSpace(r.URL.Query().Get("resourceId")),
	}
	limit, err := auditLimitFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter.Limit = limit
	rows, err := s.repo.ListAuditEvents(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) runtimeMetrics(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListTraces(r.Context(), store.TraceFilter{ManagementScope: scope})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summarizeRuntimeMetrics(rows, s.now()))
}

func summarizeRuntimeMetrics(traces []domain.TraceEvent, updatedAt time.Time) []domain.SystemMetric {
	total := len(traces)
	allowed := 0
	upstreamCalls := 0
	upstreamErrors := 0
	latencyCount := 0
	var latencyTotal int64

	for _, trace := range traces {
		if trace.Decision == domain.TraceDecisionAllowed {
			allowed++
		}
		if trace.Decision != domain.TraceDecisionAllowed || !hasUpstreamResult(trace) {
			continue
		}
		upstreamCalls++
		if trace.UpstreamError != "" || trace.UpstreamStatus >= 500 {
			upstreamErrors++
		}
		if trace.DurationMs > 0 {
			latencyCount++
			latencyTotal += trace.DurationMs
		}
	}

	allowedRate := percentage(allowed, total)
	upstreamErrorRate := percentage(upstreamErrors, upstreamCalls)
	avgLatency := averageInt64(latencyTotal, latencyCount)

	return []domain.SystemMetric{
		{
			ID:        "gateway_calls_total",
			Label:     "Gateway calls",
			Value:     total,
			Trend:     "flat",
			Status:    gatewayCallStatus(total),
			UpdatedAt: updatedAt,
		},
		{
			ID:        "allowed_rate",
			Label:     "Allowed rate",
			Value:     allowedRate,
			Unit:      "%",
			Trend:     "flat",
			Status:    allowedRateStatus(allowedRate),
			UpdatedAt: updatedAt,
		},
		{
			ID:        "upstream_error_rate",
			Label:     "Upstream errors",
			Value:     upstreamErrorRate,
			Unit:      "%",
			Trend:     "flat",
			Status:    upstreamErrorRateStatus(upstreamErrorRate),
			UpdatedAt: updatedAt,
		},
		{
			ID:        "avg_latency_ms",
			Label:     "Avg latency",
			Value:     avgLatency,
			Unit:      "ms",
			Trend:     "flat",
			Status:    latencyStatus(avgLatency),
			UpdatedAt: updatedAt,
		},
	}
}

func hasUpstreamResult(trace domain.TraceEvent) bool {
	return trace.UpstreamAttempts > 0 || trace.UpstreamStatus > 0 || trace.UpstreamError != "" || trace.DurationMs > 0
}

func percentage(numerator int, denominator int) int {
	if denominator <= 0 {
		return 0
	}
	return (numerator*100 + denominator/2) / denominator
}

func averageInt64(total int64, count int) int {
	if count <= 0 {
		return 0
	}
	return int((total + int64(count)/2) / int64(count))
}

func gatewayCallStatus(total int) string {
	if total == 0 {
		return "warning"
	}
	return "healthy"
}

func allowedRateStatus(rate int) string {
	if rate >= 95 {
		return "healthy"
	}
	if rate >= 80 {
		return "warning"
	}
	return "critical"
}

func upstreamErrorRateStatus(rate int) string {
	if rate <= 1 {
		return "healthy"
	}
	if rate <= 5 {
		return "warning"
	}
	return "critical"
}

func latencyStatus(avgLatencyMs int) string {
	if avgLatencyMs <= 300 {
		return "healthy"
	}
	if avgLatencyMs <= 1000 {
		return "warning"
	}
	return "critical"
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

func (s *Server) requirePermissionPackageDraftScope(r *http.Request, req domain.PermissionPackageDraftRequest) error {
	req = trimPermissionPackageDraftRequest(req)
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: req.TenantID, WorkspaceID: req.WorkspaceID})
}

func (s *Server) requirePermissionPackageQueryScope(r *http.Request, query permissionPackageProductionReadinessQuery) error {
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: query.TenantID, WorkspaceID: query.WorkspaceID})
}

func (s *Server) requirePermissionPackageApprovalRequestScope(r *http.Request, approval domain.PermissionPackageApprovalRequest) error {
	return s.requireRequestedScopeAllowed(r, store.ManagementScope{TenantID: approval.TenantID, WorkspaceID: approval.WorkspaceID})
}

func auditLimitFromRequest(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return defaultAuditLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > maxAuditLimit {
		return 0, domain.BadRequest("VALIDATION_FAILED", "limit must be between 1 and 500")
	}
	return limit, nil
}

func (s *Server) buildPermissionPackageDraft(ctx context.Context, req domain.PermissionPackageDraftRequest) (domain.PermissionPackageDraft, error) {
	req = trimPermissionPackageDraftRequest(req)
	if req.TargetID != "" {
		target, ok, err := s.repo.GetAgent(ctx, req.TargetID)
		if err != nil {
			return domain.PermissionPackageDraft{}, err
		}
		if !ok {
			return domain.PermissionPackageDraft{}, domain.NotFound("target agent not found")
		}
		if req.TenantID != "" {
			allowedTenant, err := s.tenantCanReceiveTargetEntitlement(ctx, target.TenantID, req.TenantID)
			if err != nil {
				return domain.PermissionPackageDraft{}, err
			}
			if !allowedTenant {
				return domain.PermissionPackageDraft{}, domain.BadRequest("VALIDATION_FAILED", "tenantId must match target tenantId or be a descendant tenant")
			}
		}
	}
	if req.CallerInstanceID != "" {
		caller, ok, err := s.repo.GetAgent(ctx, req.CallerInstanceID)
		if err != nil {
			return domain.PermissionPackageDraft{}, err
		}
		if !ok {
			return domain.PermissionPackageDraft{}, domain.NotFound("caller instance not found")
		}
		if req.TenantID != "" && caller.TenantID != req.TenantID {
			return domain.PermissionPackageDraft{}, domain.BadRequest("VALIDATION_FAILED", "caller instance must match permission package tenantId")
		}
		if req.WorkspaceID != "" && caller.WorkspaceID != req.WorkspaceID {
			return domain.PermissionPackageDraft{}, domain.BadRequest("VALIDATION_FAILED", "caller instance must match permission package workspaceId")
		}
	}
	capabilities := []domain.Capability{}
	if req.TargetID != "" {
		rows, err := s.repo.ListCapabilities(ctx, store.CapabilityFilter{TargetID: req.TargetID})
		if err != nil {
			return domain.PermissionPackageDraft{}, err
		}
		capabilities = rows
	}
	return permissionpack.BuildDraft(req, capabilities)
}

func trimPermissionPackageDraftRequest(req domain.PermissionPackageDraftRequest) domain.PermissionPackageDraftRequest {
	req.CallerInstanceID = strings.TrimSpace(req.CallerInstanceID)
	req.Region = strings.TrimSpace(req.Region)
	req.RequestText = strings.TrimSpace(req.RequestText)
	req.SubjectSelector = strings.TrimSpace(req.SubjectSelector)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.TemplateID = strings.TrimSpace(req.TemplateID)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	return req
}

func (s *Server) validatePermissionPackageApprovalReviewer(ctx context.Context, reviewer string, approval domain.PermissionPackageApprovalRequest) error {
	allowed, err := s.permissionPackageApprovalReviewerCanReview(ctx, reviewer, approval)
	if err != nil {
		return err
	}
	if !allowed {
		return domain.PermissionDenied("reviewer is not allowed to review this approval request")
	}
	return nil
}

func (s *Server) listPermissionPackageApprovalRequestsForReviewer(ctx context.Context, filter store.PermissionPackageApprovalRequestFilter, reviewer string, limit int) ([]domain.PermissionPackageApprovalRequest, error) {
	reviewer = strings.TrimSpace(reviewer)
	if len(s.approvalReviewers) == 0 {
		filter.Limit = limit
		return s.repo.ListPermissionPackageApprovalRequests(ctx, filter)
	}
	if reviewer == "" {
		return nil, nil
	}
	seen := map[string]struct{}{}
	rows := []domain.PermissionPackageApprovalRequest{}
	for _, rule := range s.approvalReviewers {
		if strings.TrimSpace(rule.Reviewer) != reviewer {
			continue
		}
		ruleFilter, ok, err := s.permissionPackageApprovalReviewerListFilter(ctx, filter, rule, limit)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		ruleRows, err := s.repo.ListPermissionPackageApprovalRequests(ctx, ruleFilter)
		if err != nil {
			return nil, err
		}
		for _, row := range ruleRows {
			if _, exists := seen[row.ID]; exists {
				continue
			}
			allowed, err := s.permissionPackageApprovalReviewerCanReview(ctx, reviewer, row)
			if err != nil {
				return nil, err
			}
			if !allowed {
				continue
			}
			seen[row.ID] = struct{}{}
			rows = append(rows, row)
		}
	}
	sortPermissionPackageApprovalRequests(rows)
	return limitPermissionPackageApprovalRequests(rows, limit), nil
}

func (s *Server) listPermissionPackageApprovalRequestsForRequest(ctx context.Context, r *http.Request, filter store.PermissionPackageApprovalRequestFilter, reviewer string, limit int) ([]domain.PermissionPackageApprovalRequest, error) {
	reviewer, scoped, err := s.permissionPackageApprovalListReviewer(r, reviewer)
	if err != nil {
		return nil, err
	}
	if scoped {
		return s.listPermissionPackageApprovalRequestsForReviewer(ctx, filter, reviewer, limit)
	}
	filter.Limit = limit
	return s.repo.ListPermissionPackageApprovalRequests(ctx, filter)
}

func (s *Server) permissionPackageApprovalListReviewer(r *http.Request, reviewer string) (string, bool, error) {
	reviewer = strings.TrimSpace(reviewer)
	if reviewer != "" {
		resolved, err := reviewerFromRequest(reviewer, r)
		return resolved, true, err
	}
	if len(s.approvalReviewers) == 0 {
		return "", false, nil
	}
	principal, ok := requestAdminPrincipal(r)
	if !ok || principal.Role == adminRolePlatformAdmin {
		return "", false, nil
	}
	resolved, err := reviewerFromRequest("", r)
	return resolved, true, err
}

func (s *Server) permissionPackageApprovalReviewerListFilter(ctx context.Context, filter store.PermissionPackageApprovalRequestFilter, rule domain.PermissionPackageApprovalReviewer, limit int) (store.PermissionPackageApprovalRequestFilter, bool, error) {
	tenantID, ok, err := s.intersectApprovalReviewerTenantScope(ctx, filter.TenantID, rule.TenantID)
	if err != nil || !ok {
		return store.PermissionPackageApprovalRequestFilter{}, ok, err
	}
	workspaceID, ok := intersectApprovalReviewerWorkspaceScope(filter.WorkspaceID, rule.WorkspaceID)
	if !ok {
		return store.PermissionPackageApprovalRequestFilter{}, false, nil
	}
	filter.TenantID = tenantID
	filter.WorkspaceID = workspaceID
	filter.Limit = limit
	return filter, true, nil
}

func (s *Server) intersectApprovalReviewerTenantScope(ctx context.Context, requestTenantID string, ruleTenantID string) (string, bool, error) {
	requestTenantID = strings.TrimSpace(requestTenantID)
	ruleTenantID = strings.TrimSpace(ruleTenantID)
	if ruleTenantID == "" || ruleTenantID == "*" {
		return requestTenantID, true, nil
	}
	if requestTenantID == "" {
		return ruleTenantID, true, nil
	}
	if requestTenantID == ruleTenantID {
		return requestTenantID, true, nil
	}
	requestWithinRule, err := s.approvalReviewerTenantMatches(ctx, ruleTenantID, requestTenantID)
	if err != nil || requestWithinRule {
		return requestTenantID, requestWithinRule, err
	}
	ruleWithinRequest, err := s.approvalReviewerTenantMatches(ctx, requestTenantID, ruleTenantID)
	if err != nil || ruleWithinRequest {
		return ruleTenantID, ruleWithinRequest, err
	}
	return "", false, nil
}

func intersectApprovalReviewerWorkspaceScope(requestWorkspaceID string, ruleWorkspaceID string) (string, bool) {
	requestWorkspaceID = strings.TrimSpace(requestWorkspaceID)
	ruleWorkspaceID = strings.TrimSpace(ruleWorkspaceID)
	if ruleWorkspaceID == "" || ruleWorkspaceID == "*" {
		return requestWorkspaceID, true
	}
	if requestWorkspaceID == "" || requestWorkspaceID == ruleWorkspaceID {
		return ruleWorkspaceID, true
	}
	return "", false
}

func (s *Server) permissionPackageApprovalReviewerCanReview(ctx context.Context, reviewer string, approval domain.PermissionPackageApprovalRequest) (bool, error) {
	reviewer = strings.TrimSpace(reviewer)
	if len(s.approvalReviewers) == 0 {
		return reviewer != "", nil
	}
	if reviewer == "" {
		return false, nil
	}
	for _, rule := range s.approvalReviewers {
		if strings.TrimSpace(rule.Reviewer) != reviewer {
			continue
		}
		if !approvalReviewerWorkspaceMatches(rule.WorkspaceID, approval.WorkspaceID) {
			continue
		}
		matches, err := s.approvalReviewerTenantMatches(ctx, rule.TenantID, approval.TenantID)
		if err != nil {
			return false, err
		}
		if matches {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) approvalReviewerTenantMatches(ctx context.Context, ruleTenantID string, approvalTenantID string) (bool, error) {
	ruleTenantID = strings.TrimSpace(ruleTenantID)
	approvalTenantID = strings.TrimSpace(approvalTenantID)
	if ruleTenantID == "" || ruleTenantID == "*" {
		return approvalTenantID != "", nil
	}
	if ruleTenantID == approvalTenantID {
		return true, nil
	}
	tenants, err := s.repo.ListTenants(ctx, store.TenantFilter{TenantID: ruleTenantID})
	if err != nil {
		return false, err
	}
	for _, tenant := range tenants {
		if tenant.ID == approvalTenantID {
			return true, nil
		}
	}
	return false, nil
}

func approvalReviewerWorkspaceMatches(ruleWorkspaceID string, approvalWorkspaceID string) bool {
	ruleWorkspaceID = strings.TrimSpace(ruleWorkspaceID)
	if ruleWorkspaceID == "" || ruleWorkspaceID == "*" {
		return true
	}
	return ruleWorkspaceID == strings.TrimSpace(approvalWorkspaceID)
}

func sortPermissionPackageApprovalRequests(rows []domain.PermissionPackageApprovalRequest) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
}

func limitPermissionPackageApprovalRequests(rows []domain.PermissionPackageApprovalRequest, limit int) []domain.PermissionPackageApprovalRequest {
	if limit > 0 && len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func (s *Server) createPermissionPackageApprovalRequestRecord(ctx context.Context, req domain.PermissionPackageDraftRequest, requestedBy string, now time.Time) (domain.PermissionPackageApprovalRequest, error) {
	draft, err := s.buildPermissionPackageDraft(ctx, req)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if !draft.Readiness.CanApply {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "permission package draft is not ready to request approval")
	}
	if draft.PolicyGate.CanApplyDirectly {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "permission package does not require approval")
	}
	approval := permissionPackageApprovalRequestFromDraft(draft, requestedBy, now)
	return s.repo.CreatePermissionPackageApprovalRequest(ctx, approval)
}

func (s *Server) resolvePermissionPackageApprovalRequestRecord(ctx context.Context, existing domain.PermissionPackageApprovalRequest, status domain.PermissionPackageApprovalStatus, reviewer string, comment string, now time.Time) (domain.PermissionPackageApprovalRequest, error) {
	if existing.Status != domain.PermissionPackageApprovalStatusPending {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request is already resolved")
	}
	if strings.TrimSpace(existing.RequestedBy) != "" && strings.TrimSpace(reviewer) == strings.TrimSpace(existing.RequestedBy) {
		return domain.PermissionPackageApprovalRequest{}, domain.PermissionDenied("reviewer cannot resolve their own permission package approval request")
	}
	updated := existing
	updated.Status = status
	updated.ReviewedBy = strings.TrimSpace(reviewer)
	updated.ReviewComment = strings.TrimSpace(comment)
	updated.UpdatedAt = now
	updated.ResolvedAt = now
	saved, ok, err := s.repo.UpdatePermissionPackageApprovalRequest(ctx, updated)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if !ok {
		return domain.PermissionPackageApprovalRequest{}, domain.NotFound("approval request not found")
	}
	return saved, nil
}

func (s *Server) withdrawPermissionPackageApprovalRequestRecord(ctx context.Context, existing domain.PermissionPackageApprovalRequest, requester string, comment string, now time.Time) (domain.PermissionPackageApprovalRequest, error) {
	if existing.Status != domain.PermissionPackageApprovalStatusPending {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request is already resolved")
	}
	if !existing.ConsumedAt.IsZero() || strings.TrimSpace(existing.ConsumedByApplicationID) != "" {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request is already consumed")
	}
	if !existing.ExpiresAt.IsZero() && !now.Before(existing.ExpiresAt) {
		return domain.PermissionPackageApprovalRequest{}, domain.BadRequest("VALIDATION_FAILED", "approval request has expired")
	}
	requester = strings.TrimSpace(requester)
	if strings.TrimSpace(existing.RequestedBy) != "" && requester != strings.TrimSpace(existing.RequestedBy) {
		return domain.PermissionPackageApprovalRequest{}, domain.PermissionDenied("only the original requester can withdraw this approval request")
	}
	updated := existing
	updated.Status = domain.PermissionPackageApprovalStatusWithdrawn
	updated.ReviewedBy = requester
	updated.ReviewComment = strings.TrimSpace(comment)
	updated.UpdatedAt = now
	updated.ResolvedAt = now
	saved, ok, err := s.repo.UpdatePermissionPackageApprovalRequest(ctx, updated)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if !ok {
		return domain.PermissionPackageApprovalRequest{}, domain.NotFound("approval request not found")
	}
	return saved, nil
}

func permissionPackageApprovalRequestFromDraft(draft domain.PermissionPackageDraft, requestedBy string, now time.Time) domain.PermissionPackageApprovalRequest {
	allowedCapabilityIDs, allowedCapabilityKeys := permissionPackageCapabilityIDsAndKeys(draft.AllowedCapabilities)
	allowedCapabilityFingerprints := permissionPackageCapabilityFingerprints(draft.AllowedCapabilities)
	return domain.PermissionPackageApprovalRequest{
		ID:                            security.NewID("ppar"),
		DraftID:                       draft.ID,
		TemplateID:                    draft.Template.ID,
		TemplateVersion:               draft.Template.Version,
		PolicyVersion:                 draft.PolicyGate.PolicyVersion,
		TenantID:                      draft.Input.TenantID,
		WorkspaceID:                   draft.Input.WorkspaceID,
		TargetID:                      draft.Input.TargetID,
		CallerInstanceID:              draft.Input.CallerInstanceID,
		SubjectSelector:               draft.Input.SubjectSelector,
		RequestText:                   draft.Input.RequestText,
		Region:                        draft.Input.Region,
		DataScopes:                    append([]domain.DataScope(nil), draft.DataScopes...),
		AllowedCapabilityIDs:          allowedCapabilityIDs,
		AllowedCapabilityKeys:         allowedCapabilityKeys,
		AllowedCapabilityFingerprints: allowedCapabilityFingerprints,
		PolicyGate:                    draft.PolicyGate,
		Status:                        domain.PermissionPackageApprovalStatusPending,
		RequestedBy:                   requestedBy,
		CreatedAt:                     now,
		UpdatedAt:                     now,
		ExpiresAt:                     now.Add(defaultPermissionPackageApprovalTTL),
	}
}

func validatePermissionPackageApprovalForDraft(approval domain.PermissionPackageApprovalRequest, draft domain.PermissionPackageDraft, now time.Time) error {
	if approval.Status != domain.PermissionPackageApprovalStatusApproved {
		return domain.BadRequest("VALIDATION_FAILED", "permission package approval request must be approved before apply")
	}
	if !approval.ConsumedAt.IsZero() {
		return permissionPackageApprovalAlreadyConsumedError()
	}
	if !approval.ExpiresAt.IsZero() && !now.Before(approval.ExpiresAt) {
		return domain.BadRequest("VALIDATION_FAILED", "permission package approval request has expired")
	}
	allowedCapabilityIDs, allowedCapabilityKeys := permissionPackageCapabilityIDsAndKeys(draft.AllowedCapabilities)
	allowedCapabilityFingerprints := permissionPackageCapabilityFingerprints(draft.AllowedCapabilities)
	if approval.DraftID != draft.ID ||
		approval.TemplateID != draft.Template.ID ||
		approval.TemplateVersion != draft.Template.Version ||
		approval.PolicyVersion != draft.PolicyGate.PolicyVersion ||
		approval.TenantID != draft.Input.TenantID ||
		approval.WorkspaceID != draft.Input.WorkspaceID ||
		approval.TargetID != draft.Input.TargetID ||
		approval.CallerInstanceID != draft.Input.CallerInstanceID ||
		approval.SubjectSelector != draft.Input.SubjectSelector ||
		approval.RequestText != draft.Input.RequestText ||
		approval.Region != draft.Input.Region ||
		!samePermissionPackageDataScopes(approval.DataScopes, draft.DataScopes) ||
		!sameStringSet(approval.AllowedCapabilityIDs, allowedCapabilityIDs) ||
		!sameStringSet(approval.AllowedCapabilityKeys, allowedCapabilityKeys) ||
		!sameStringSet(approval.AllowedCapabilityFingerprints, allowedCapabilityFingerprints) {
		return domain.BadRequest("VALIDATION_FAILED", "approved permission package approval request does not match current draft")
	}
	return nil
}

func (s *Server) permissionPackageApprovalNotConsumableError(ctx context.Context, approvalRequestID string, draft domain.PermissionPackageDraft, now time.Time) error {
	approval, ok, err := s.repo.GetPermissionPackageApprovalRequest(ctx, approvalRequestID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.NotFound("approval request not found")
	}
	if !approval.ConsumedAt.IsZero() {
		return permissionPackageApprovalAlreadyConsumedError()
	}
	if err := validatePermissionPackageApprovalForDraft(approval, draft, now); err != nil {
		return err
	}
	return domain.BadRequest("VALIDATION_FAILED", "permission package approval request is no longer available")
}

func permissionPackageApprovalAlreadyConsumedError() domain.AppError {
	return domain.BadRequest("PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED", "permission package approval request is already consumed")
}

func permissionPackageCapabilityIDsAndKeys(capabilities []domain.Capability) ([]string, []string) {
	ids := make([]string, 0, len(capabilities))
	keys := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		ids = append(ids, capability.ID)
		keys = append(keys, capability.Key)
	}
	return ids, keys
}

type permissionPackageCapabilityFingerprintPayload struct {
	ID              string                           `json:"id"`
	TargetID        string                           `json:"targetId"`
	Type            domain.CapabilityType            `json:"type"`
	Key             string                           `json:"key"`
	Action          domain.CapabilityAction          `json:"action"`
	NativeScopes    []string                         `json:"nativeScopes,omitempty"`
	DataDomains     []string                         `json:"dataDomains,omitempty"`
	DataScopes      []domain.DataScope               `json:"dataScopes,omitempty"`
	Sensitivity     domain.CapabilitySensitivity     `json:"sensitivity"`
	RiskLevel       domain.CapabilityRisk            `json:"riskLevel"`
	EnforcementMode domain.CapabilityEnforcementMode `json:"enforcementMode"`
	DiscoveryStatus domain.CapabilityDiscoveryStatus `json:"discoveryStatus"`
	Version         int                              `json:"version"`
}

func permissionPackageCapabilityFingerprints(capabilities []domain.Capability) []string {
	fingerprints := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		fingerprints = append(fingerprints, permissionPackageCapabilityFingerprint(capability))
	}
	sort.Strings(fingerprints)
	return fingerprints
}

func permissionPackageCapabilityFingerprint(capability domain.Capability) string {
	payload := permissionPackageCapabilityFingerprintPayload{
		ID:              capability.ID,
		TargetID:        capability.TargetID,
		Type:            capability.Type,
		Key:             capability.Key,
		Action:          capability.Action,
		NativeScopes:    sortedStringCopy(capability.NativeScopes),
		DataDomains:     sortedStringCopy(capability.DataDomains),
		DataScopes:      sortedDataScopes(capability.DataScopes),
		Sensitivity:     capability.Sensitivity,
		RiskLevel:       capability.RiskLevel,
		EnforcementMode: capability.EnforcementMode,
		DiscoveryStatus: capability.DiscoveryStatus,
		Version:         capability.Version,
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return capability.ID + ":" + hex.EncodeToString(sum[:])
}

func sortedStringCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func sortedDataScopes(values []domain.DataScope) []domain.DataScope {
	out := append([]domain.DataScope(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		return dataScopeSortKey(out[i]) < dataScopeSortKey(out[j])
	})
	return out
}

func dataScopeSortKey(scope domain.DataScope) string {
	data, _ := json.Marshal(scope)
	return string(data)
}

func samePermissionPackageDataScopes(left []domain.DataScope, right []domain.DataScope) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func permissionPackageApprovalAuditMetadata(request domain.PermissionPackageApprovalRequest) map[string]any {
	return map[string]any{
		"approvalRequestId":       request.ID,
		"draftId":                 request.DraftID,
		"templateId":              request.TemplateID,
		"templateVersion":         request.TemplateVersion,
		"policyVersion":           request.PolicyVersion,
		"targetId":                request.TargetID,
		"callerInstanceId":        request.CallerInstanceID,
		"status":                  request.Status,
		"requestedBy":             request.RequestedBy,
		"reviewedBy":              request.ReviewedBy,
		"reasonCount":             len(request.PolicyGate.Reasons),
		"allowedCapabilityIds":    request.AllowedCapabilityIDs,
		"expiresAt":               request.ExpiresAt,
		"consumedAt":              request.ConsumedAt,
		"consumedByApplicationId": request.ConsumedByApplicationID,
	}
}

func (s *Server) managementAuditEvent(r *http.Request, tenantID string, workspaceID string, action string, resourceType string, resourceID string, summary string, metadata map[string]any) domain.AuditEvent {
	if metadata == nil {
		metadata = map[string]any{}
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

type mcpToolsListResponse struct {
	Result struct {
		Tools []mcpToolDescription `json:"tools"`
	} `json:"result"`
}

type mcpToolDescription struct {
	Name         string         `json:"name"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema"`
}

func (s *Server) discoverMCPCapabilities(ctx context.Context, target domain.Agent) ([]domain.Capability, error) {
	endpoint, ok := target.ChannelConfig["endpoint"].(string)
	endpoint = strings.TrimSpace(endpoint)
	if !ok || endpoint == "" {
		return nil, domain.BadRequest("VALIDATION_FAILED", "mcp target requires channelConfig.endpoint for capability discovery")
	}
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "capability-discovery",
		"method":  "tools/list",
		"params":  map[string]any{},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, domain.UpstreamError("capability discovery request could not be prepared")
	}
	timeout, err := proxyTimeoutFromConfig(target.ChannelConfig)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, domain.UpstreamError("capability discovery request could not be prepared")
	}
	req.Header.Set("Content-Type", "application/json")
	if err := copyConfiguredHeaders(req.Header, target.ChannelConfig); err != nil {
		return nil, err
	}
	if err := copyCredentialHeaders(req.Header, target.ChannelConfig, target.Credentials); err != nil {
		return nil, err
	}
	setMCPUpstreamHeaders(req.Header)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, classifyUpstreamError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, domain.UpstreamError("capability discovery upstream returned non-2xx status")
	}
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxProxyBodyBytes+1))
	if err != nil {
		return nil, domain.UpstreamError("capability discovery response could not be read")
	}
	if len(responseBody) > maxProxyBodyBytes {
		return nil, domain.PayloadTooLarge("capability discovery response exceeds 4MiB")
	}
	var result mcpToolsListResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, domain.BadRequest("VALIDATION_FAILED", "capability discovery response must be valid MCP tools/list JSON")
	}
	existing, err := s.repo.ListCapabilities(ctx, store.CapabilityFilter{TargetID: target.ID})
	if err != nil {
		return nil, err
	}
	existingByKey := map[string]domain.Capability{}
	for _, capability := range existing {
		if capability.Type == domain.CapabilityTypeMCPTool {
			existingByKey[capability.Key] = capability
		}
	}
	now := s.now()
	seen := map[string]struct{}{}
	out := make([]domain.Capability, 0, len(result.Result.Tools))
	for _, tool := range result.Result.Tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
		capability := capabilityFromMCPTool(target.ID, tool, now)
		if current, ok := existingByKey[name]; ok {
			capability.ID = current.ID
			capability.DiscoveredAt = current.DiscoveredAt
			capability.Version = current.Version
			capability.DiscoveryStatus = current.DiscoveryStatus
			if mcpCapabilityChanged(current, capability) {
				capability.Version++
				capability.DiscoveryStatus = domain.CapabilityDiscoveryPendingReview
			}
		}
		saved, err := s.repo.UpsertCapability(ctx, capability)
		if err != nil {
			return nil, err
		}
		out = append(out, saved)
	}
	for key, current := range existingByKey {
		if _, ok := seen[key]; ok || current.DiscoveryStatus == domain.CapabilityDiscoveryRemoved {
			continue
		}
		current.DiscoveryStatus = domain.CapabilityDiscoveryRemoved
		current.UpdatedAt = now
		saved, ok, err := s.repo.UpdateCapability(ctx, current)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, saved)
		}
	}
	return out, nil
}

func capabilityFromMCPTool(targetID string, tool mcpToolDescription, now time.Time) domain.Capability {
	name := strings.TrimSpace(tool.Name)
	displayName := strings.TrimSpace(tool.Title)
	if displayName == "" {
		displayName = name
	}
	action := inferCapabilityAction(name)
	return domain.Capability{
		ID:              security.NewID("cap"),
		TargetID:        targetID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             name,
		DisplayName:     displayName,
		Description:     strings.TrimSpace(tool.Description),
		Action:          action,
		InputSchema:     nonNilMap(tool.InputSchema),
		OutputSchema:    nonNilMap(tool.OutputSchema),
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       riskForCapabilityAction(action),
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryPendingReview,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
}

func mcpCapabilityChanged(left domain.Capability, right domain.Capability) bool {
	return left.DisplayName != right.DisplayName ||
		left.Description != right.Description ||
		left.Action != right.Action ||
		jsonStable(left.InputSchema) != jsonStable(right.InputSchema) ||
		jsonStable(left.OutputSchema) != jsonStable(right.OutputSchema)
}

func inferCapabilityAction(name string) domain.CapabilityAction {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"search", "list", "get", "query", "read"} {
		if strings.HasPrefix(lower, prefix) {
			return domain.CapabilityActionRead
		}
	}
	if strings.HasPrefix(lower, "export") || strings.Contains(lower, "export") {
		return domain.CapabilityActionExport
	}
	for _, prefix := range []string{"delete", "remove"} {
		if strings.HasPrefix(lower, prefix) {
			return domain.CapabilityActionDelete
		}
	}
	for _, prefix := range []string{"update", "create", "write", "patch"} {
		if strings.HasPrefix(lower, prefix) {
			return domain.CapabilityActionWrite
		}
	}
	return domain.CapabilityActionExecute
}

func riskForCapabilityAction(action domain.CapabilityAction) domain.CapabilityRisk {
	switch action {
	case domain.CapabilityActionRead:
		return domain.CapabilityRiskLow
	case domain.CapabilityActionExport, domain.CapabilityActionDelete, domain.CapabilityActionAdmin:
		return domain.CapabilityRiskHigh
	default:
		return domain.CapabilityRiskMedium
	}
}

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func jsonStable(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func validCapabilityDiscoveryStatus(status domain.CapabilityDiscoveryStatus) bool {
	switch status {
	case domain.CapabilityDiscoveryPendingReview, domain.CapabilityDiscoveryApproved, domain.CapabilityDiscoveryDeprecated, domain.CapabilityDiscoveryRemoved:
		return true
	default:
		return false
	}
}

func validCapabilitySensitivity(sensitivity domain.CapabilitySensitivity) bool {
	switch sensitivity {
	case domain.CapabilitySensitivityPublic, domain.CapabilitySensitivityInternal, domain.CapabilitySensitivityConfidential, domain.CapabilitySensitivityRestricted:
		return true
	default:
		return false
	}
}

func validCapabilityRisk(risk domain.CapabilityRisk) bool {
	switch risk {
	case domain.CapabilityRiskLow, domain.CapabilityRiskMedium, domain.CapabilityRiskHigh, domain.CapabilityRiskCritical:
		return true
	default:
		return false
	}
}

func validPermissionPackageApprovalStatus(status domain.PermissionPackageApprovalStatus) bool {
	switch status {
	case domain.PermissionPackageApprovalStatusPending,
		domain.PermissionPackageApprovalStatusApproved,
		domain.PermissionPackageApprovalStatusRejected,
		domain.PermissionPackageApprovalStatusWithdrawn:
		return true
	default:
		return false
	}
}

func normalizePolicyEffect(value domain.PolicyEffect, fallback domain.PolicyEffect) (domain.PolicyEffect, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.PolicyEffectAllow && value != domain.PolicyEffectDeny {
		return "", domain.BadRequest("VALIDATION_FAILED", "effect must be allow or deny")
	}
	return value, nil
}

func normalizePolicyStatus(value domain.PolicyStatus, fallback domain.PolicyStatus) (domain.PolicyStatus, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.PolicyStatusEnabled && value != domain.PolicyStatusDisabled {
		return "", domain.BadRequest("VALIDATION_FAILED", "status must be enabled or disabled")
	}
	return value, nil
}

func normalizeTenantStatus(value domain.TenantStatus, fallback domain.TenantStatus) (domain.TenantStatus, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.TenantStatusActive && value != domain.TenantStatusDisabled {
		return "", domain.BadRequest("VALIDATION_FAILED", "status must be active or disabled")
	}
	return value, nil
}

func validTenantID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	return !strings.ContainsAny(value, " \t\r\n/")
}

func normalizePolicyPriority(value *int) (int, error) {
	if value == nil {
		return 100, nil
	}
	if *value < 0 {
		return 0, domain.BadRequest("VALIDATION_FAILED", "priority must be zero or greater")
	}
	return *value, nil
}

func findTenantEntitlement(rows []domain.TenantEntitlement, id string) (domain.TenantEntitlement, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return domain.TenantEntitlement{}, false
}

func findWorkspaceAssignment(rows []domain.WorkspaceAssignment, id string) (domain.WorkspaceAssignment, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return domain.WorkspaceAssignment{}, false
}

func (s *Server) effectiveTenantEntitlementDataScopes(ctx context.Context, entitlement domain.TenantEntitlement) ([]domain.DataScope, error) {
	capability, ok, err := s.repo.GetCapability(ctx, entitlement.CapabilityID)
	if err != nil {
		return nil, err
	}
	if !ok || capability.TargetID != entitlement.TargetID {
		return nil, domain.BadRequest("VALIDATION_FAILED", "tenant entitlement capability is not registered for target")
	}
	scopes, ok := domain.EffectiveDataScopes(capability.DataScopes, entitlement.DataScopes)
	if !ok {
		return nil, domain.BadRequest("VALIDATION_FAILED", "tenant entitlement dataScopes exceed capability dataScopes")
	}
	return scopes, nil
}

func (s *Server) effectiveWorkspaceAssignmentDataScopes(ctx context.Context, entitlement domain.TenantEntitlement, assignment domain.WorkspaceAssignment) ([]domain.DataScope, error) {
	entitlementScopes, err := s.effectiveTenantEntitlementDataScopes(ctx, entitlement)
	if err != nil {
		return nil, err
	}
	scopes, ok := domain.EffectiveDataScopes(entitlementScopes, assignment.DataScopes)
	if !ok {
		return nil, domain.BadRequest("VALIDATION_FAILED", "workspace assignment dataScopes exceed tenant entitlement dataScopes")
	}
	return scopes, nil
}

func (s *Server) tenantCanReceiveTargetEntitlement(ctx context.Context, targetTenantID string, granteeTenantID string) (bool, error) {
	targetTenantID = strings.TrimSpace(targetTenantID)
	granteeTenantID = strings.TrimSpace(granteeTenantID)
	if targetTenantID == "" || granteeTenantID == "" {
		return false, nil
	}
	if targetTenantID == granteeTenantID {
		return true, nil
	}
	current, ok, err := s.repo.GetTenant(ctx, granteeTenantID)
	if err != nil || !ok {
		return false, err
	}
	for current.ParentTenantID != "" {
		if current.ParentTenantID == targetTenantID {
			return true, nil
		}
		parent, ok, err := s.repo.GetTenant(ctx, current.ParentTenantID)
		if err != nil || !ok {
			return false, err
		}
		current = parent
	}
	return false, nil
}

func normalizeRoutePolicyEffect(value domain.RoutePolicyEffect, fallback domain.RoutePolicyEffect) (domain.RoutePolicyEffect, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.RoutePolicyEffectAllow && value != domain.RoutePolicyEffectDeny {
		return "", domain.BadRequest("VALIDATION_FAILED", "effect must be allow or deny")
	}
	return value, nil
}

func normalizeRoutePolicyStatus(value domain.RoutePolicyStatus, fallback domain.RoutePolicyStatus) (domain.RoutePolicyStatus, error) {
	if value == "" {
		return fallback, nil
	}
	if value != domain.RoutePolicyStatusEnabled && value != domain.RoutePolicyStatusDisabled {
		return "", domain.BadRequest("VALIDATION_FAILED", "status must be enabled or disabled")
	}
	return value, nil
}

func normalizeRoutePolicyPriority(value *int) (int, error) {
	if value == nil {
		return 100, nil
	}
	if *value < 0 {
		return 0, domain.BadRequest("VALIDATION_FAILED", "priority must be zero or greater")
	}
	return *value, nil
}

func normalizeRoutePolicyRetry(req *domain.RoutePolicyRetryRequest, fieldPrefix string) (*domain.RoutePolicyRetry, error) {
	if req == nil {
		return nil, nil
	}
	maxAttempts := 1
	if req.MaxAttempts != nil {
		if *req.MaxAttempts < 1 || *req.MaxAttempts > maxRetryAttempts {
			return nil, domain.BadRequest("VALIDATION_FAILED", fieldPrefix+".maxAttempts must be between 1 and 4")
		}
		maxAttempts = *req.MaxAttempts
	}
	backoffMs := 0
	if req.BackoffMs != nil {
		if *req.BackoffMs < 0 || *req.BackoffMs > int(maxRetryBackoff/time.Millisecond) {
			return nil, domain.BadRequest("VALIDATION_FAILED", fieldPrefix+".backoffMs must be between 0 and 1000")
		}
		backoffMs = *req.BackoffMs
	}
	statusCodes := []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout}
	if req.StatusCodes != nil {
		statusCodes = append([]int(nil), req.StatusCodes...)
	}
	for _, statusCode := range statusCodes {
		if statusCode < 500 || statusCode > 599 {
			return nil, domain.BadRequest("VALIDATION_FAILED", fieldPrefix+".statusCodes must contain 5xx status codes")
		}
	}
	return &domain.RoutePolicyRetry{
		MaxAttempts: maxAttempts,
		BackoffMs:   backoffMs,
		StatusCodes: statusCodes,
	}, nil
}

func routePolicyRetryFromPatch(raw json.RawMessage) (*domain.RoutePolicyRetry, error) {
	if string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	var req domain.RoutePolicyRetryRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, domain.BadRequest("INVALID_JSON", "retry must be an object or null")
	}
	return normalizeRoutePolicyRetry(&req, "retry")
}

func defaultRoutePolicyName(routeType string, routeKey string, effect domain.RoutePolicyEffect) string {
	key := routeKey
	if key == "" {
		key = "*"
	}
	return string(effect) + " " + routeType + ":" + key
}

func routePolicyAuditMetadata(policy domain.RoutePolicy) map[string]any {
	metadata := map[string]any{
		"callerAgentId": policy.CallerID,
		"targetAgentId": policy.TargetID,
		"routeType":     policy.RouteType,
		"routeKey":      policy.RouteKey,
		"effect":        policy.Effect,
		"status":        policy.Status,
		"priority":      policy.Priority,
	}
	if policy.Retry != nil {
		metadata["retry"] = map[string]any{
			"maxAttempts": policy.Retry.MaxAttempts,
			"backoffMs":   policy.Retry.BackoffMs,
			"statusCodes": policy.Retry.StatusCodes,
		}
	}
	return metadata
}

func initialCredentialVersion(credentials map[string]string) int {
	if len(credentials) == 0 {
		return 0
	}
	return 1
}

func credentialKeyNames(credentials map[string]string) []string {
	keys := make([]string, 0, len(credentials))
	for key := range credentials {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func agentPatchFields(req domain.UpdateAgentRequest) []string {
	fields := []string{}
	if req.Name != nil {
		fields = append(fields, "name")
	}
	if req.Description != nil {
		fields = append(fields, "description")
	}
	if req.OwnerID != nil {
		fields = append(fields, "ownerId")
	}
	if req.ChannelConfig != nil {
		fields = append(fields, "channelConfig")
	}
	if req.Status != nil {
		fields = append(fields, "status")
	}
	return fields
}
