package httpapi

import (
	"bytes"
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/SummerXaa-Z/agent-harbor/internal/contracts"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type callerContextKey struct{}

type Server struct {
	repo     store.Repository
	now      func() time.Time
	adminKey string
}

const (
	defaultProxyTimeout = 10 * time.Second
	maxProxyTimeout     = 30 * time.Second
	maxRetryAttempts    = 4
	maxRetryBackoff     = time.Second
	maxProxyBodyBytes   = 4 << 20
	defaultAuditLimit   = 100
	maxAuditLimit       = 500
)

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

type Option func(*Server)

func WithAdminKey(key string) Option {
	return func(s *Server) {
		s.adminKey = strings.TrimSpace(key)
	}
}

func New(repo store.Repository, options ...Option) *Server {
	server := &Server{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
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
	r.Use(localDevCORS)

	r.Get("/healthz", s.health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/contracts/providers", s.listProviderContracts)
		r.Get("/contracts/channels", s.listChannelContracts)
		r.Group(func(r chi.Router) {
			r.Use(s.requireAdmin)
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
			r.Post("/route-policies", s.createRoutePolicy)
			r.Get("/route-policies", s.listRoutePolicies)
			r.Patch("/route-policies/{id}", s.updateRoutePolicy)
			r.Delete("/route-policies/{id}", s.disableRoutePolicy)
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

func localDevCORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]struct{}{
		"http://localhost:4174": {},
		"http://localhost:5174": {},
		"http://127.0.0.1:4174": {},
		"http://127.0.0.1:5174": {},
		"http://[::1]:4174":     {},
		"http://[::1]:5174":     {},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Key, X-Run-Id")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.adminKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		provided := r.Header.Get("X-Admin-Key")
		if len(provided) != len(s.adminKey) || subtle.ConstantTimeCompare([]byte(provided), []byte(s.adminKey)) != 1 {
			writeError(w, domain.Unauthorized("missing or invalid admin key"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listProviderContracts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Providers())
}

func (s *Server) listChannelContracts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, contracts.Channels())
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
	if err := validateAgentForSave(agent); err != nil {
		return domain.Agent{}, err
	}
	return agent, nil
}

func validateAgentForSave(agent domain.Agent) error {
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
			if err := security.ValidateOutboundEndpoint(value); err != nil {
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
	rows, err := s.repo.ListAgents(r.Context(), store.AgentFilter{ManagementScope: managementScopeFromRequest(r)})
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
	if err := validateAgentForSave(updated); err != nil {
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
	effective := agent
	effective.Credentials = credentials
	if err := validateAgentForSave(effective); err != nil {
		writeError(w, err)
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
	plaintext, prefix := security.NewAgentKey()
	now := s.now()
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
	rows, err := s.repo.ListAgentKeys(r.Context(), managementScopeFromRequest(r))
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
	if _, ok, err := s.repo.GetAgent(r.Context(), req.TargetID); err != nil {
		writeError(w, err)
		return
	} else if !ok {
		writeError(w, domain.NotFound("target agent not found"))
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
	rows, err := s.repo.ListAccessGrants(r.Context(), managementScopeFromRequest(r))
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
	rows, err := s.repo.ListRoutePolicies(r.Context(), managementScopeFromRequest(r))
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
	routeKey, body, err := mcpRouteKeyFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	s.handleDataPlane(w, r, "mcp", routeKey)
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
	if s.proxyUpstreamIfConfigured(w, r, target, routeType, routeKey, decision.Retry, recordAllowedTrace) {
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

func (s *Server) proxyUpstreamIfConfigured(w http.ResponseWriter, r *http.Request, target domain.Agent, routeType string, routeKey string, retryOverride *domain.RoutePolicyRetry, recordAllowedTrace func(proxyTraceResult) (domain.TraceEvent, error)) bool {
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
		if value := src.Get(key); value != "" {
			dst.Set(key, value)
		}
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

func mcpRouteKeyFromRequest(r *http.Request) (string, []byte, error) {
	body, err := readProxyBody(r.Body)
	if err != nil {
		return "", nil, err
	}
	var payload struct {
		Method *string `json:"method"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, domain.BadRequest("VALIDATION_FAILED", "mcp request body must be valid JSON")
	}
	if payload.Method == nil || strings.TrimSpace(*payload.Method) == "" {
		return "", nil, domain.BadRequest("VALIDATION_FAILED", "mcp request method is required")
	}
	return strings.TrimSpace(*payload.Method), body, nil
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
		headerValue, ok := value.(string)
		if !ok {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers values must be strings")
		}
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" || !validHeaderName(trimmedName) || security.IsSecretLikeKey(trimmedName) {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.headers contains invalid header name")
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
		key, ok := credentialKey.(string)
		if !ok {
			return domain.BadRequest("VALIDATION_FAILED", "channelConfig.credentialHeaders values must be credential key strings")
		}
		trimmedName := strings.TrimSpace(name)
		key = strings.TrimSpace(key)
		value, exists := credentials[key]
		if trimmedName == "" || !validHeaderName(trimmedName) || key == "" || !validCredentialKey(key) || !exists {
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
	filter := store.TraceFilter{
		ManagementScope: managementScopeFromRequest(r),
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
	filter := store.AuditEventFilter{
		ManagementScope: managementScopeFromRequest(r),
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
	rows, err := s.repo.ListTraces(r.Context(), store.TraceFilter{ManagementScope: managementScopeFromRequest(r)})
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
	if strings.TrimSpace(r.Header.Get("X-Admin-Key")) != "" {
		return "admin-key"
	}
	return "local-dev"
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
