package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/SummerXaa-Z/ai-nexus-go-rebirth/internal/contracts"
	"github.com/SummerXaa-Z/ai-nexus-go-rebirth/internal/domain"
	"github.com/SummerXaa-Z/ai-nexus-go-rebirth/internal/security"
	"github.com/SummerXaa-Z/ai-nexus-go-rebirth/internal/store"
)

type callerContextKey struct{}

type Server struct {
	repo store.Repository
	now  func() time.Time
}

func New(repo store.Repository) *Server {
	return &Server{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
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
		r.Post("/agents", s.createAgent)
		r.Get("/agents", s.listAgents)
		r.Get("/agents/{id}", s.getAgent)
		r.Post("/agent-keys", s.createAgentKey)
		r.Get("/api-keys", s.listAgentKeys)
		r.Post("/api-keys", s.createAgentKey)
		r.Delete("/api-keys/{id}", s.revokeAgentKey)
		r.Post("/access-grants", s.createAccessGrant)
		r.Get("/audit/traces", s.listTraces)
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Run-Id")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
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
	created, err := s.repo.CreateAgent(r.Context(), agent)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) agentFromRequest(req domain.CreateAgentRequest) (domain.Agent, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	req.ChannelType = strings.TrimSpace(req.ChannelType)
	if req.Name == "" {
		return domain.Agent{}, domain.BadRequest("VALIDATION_FAILED", "name is required")
	}
	if req.WorkspaceID == "" {
		return domain.Agent{}, domain.BadRequest("VALIDATION_FAILED", "workspaceId is required")
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}
	if req.Status == "" {
		req.Status = domain.AgentStatusDraft
	}
	if req.Status != domain.AgentStatusDraft && req.Status != domain.AgentStatusActive && req.Status != domain.AgentStatusDisabled {
		return domain.Agent{}, domain.BadRequest("VALIDATION_FAILED", "status must be draft, active, or disabled")
	}
	if req.ChannelType == "" {
		req.ChannelType = "local"
	}
	channel, ok := contracts.Channel(req.ChannelType)
	if !ok {
		return domain.Agent{}, domain.BadRequest("VALIDATION_FAILED", "channelType is not supported")
	}
	if req.ChannelConfig == nil {
		req.ChannelConfig = map[string]any{}
	}
	if security.ContainsSecretLikeKey(req.ChannelConfig) {
		return domain.Agent{}, domain.BadRequest("VALIDATION_FAILED", "channelConfig must not contain secret-like keys")
	}
	for _, key := range []string{"endpoint", "specUrl"} {
		if raw, exists := req.ChannelConfig[key]; exists {
			value, ok := raw.(string)
			if !ok {
				return domain.Agent{}, domain.BadRequest("VALIDATION_FAILED", key+" must be a string URL")
			}
			if err := security.ValidateOutboundEndpoint(value); err != nil {
				return domain.Agent{}, domain.BadRequest("VALIDATION_FAILED", err.Error())
			}
		}
	}
	if req.Status == domain.AgentStatusActive && channel.EndpointRequiredWhenActive {
		endpoint, ok := req.ChannelConfig["endpoint"].(string)
		if !ok || strings.TrimSpace(endpoint) == "" {
			return domain.Agent{}, domain.BadRequest("VALIDATION_FAILED", "active "+req.ChannelType+" agent requires channelConfig.endpoint")
		}
	}
	now := s.now()
	return domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      req.TenantID,
		WorkspaceID:   req.WorkspaceID,
		Name:          req.Name,
		Description:   req.Description,
		OwnerID:       req.OwnerID,
		ChannelType:   req.ChannelType,
		ChannelConfig: req.ChannelConfig,
		Status:        req.Status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	rows, err := s.repo.ListAgents(r.Context(), workspaceID)
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
	if ttl <= 0 || ttl > 3600 {
		ttl = 1800
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
	created, err := s.repo.CreateAgentKey(r.Context(), key)
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
	rows, err := s.repo.ListAgentKeys(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) revokeAgentKey(w http.ResponseWriter, r *http.Request) {
	key, ok, err := s.repo.RevokeAgentKey(r.Context(), chi.URLParam(r, "id"), s.now())
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
	if req.CallerID == "" || req.TargetID == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "callerAgentId and targetAgentId are required"))
		return
	}
	if _, ok, err := s.repo.GetAgent(r.Context(), req.CallerID); err != nil {
		writeError(w, err)
		return
	} else if !ok {
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
	created, err := s.repo.CreateAccessGrant(r.Context(), grant)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
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
	s.handleDataPlane(w, r, "mcp", "tools/call")
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
	decision := domain.TraceDecisionAllowed
	reason := "access grant matched"
	status := http.StatusOK
	if !s.repo.HasGrant(r.Context(), caller.ID, targetID, routeType, routeKey, s.now()) {
		decision = domain.TraceDecisionDenied
		reason = "caller has no access grant for target route"
		status = http.StatusForbidden
	}
	trace := domain.TraceEvent{
		ID:        security.NewID("trc"),
		RunID:     r.Header.Get("X-Run-Id"),
		CallerID:  caller.ID,
		TargetID:  targetID,
		RouteType: routeType,
		RouteKey:  routeKey,
		Decision:  decision,
		Reason:    reason,
		CreatedAt: s.now(),
	}
	if _, err := s.repo.AppendTrace(r.Context(), trace); err != nil {
		writeError(w, err)
		return
	}
	if decision == domain.TraceDecisionDenied {
		writeError(w, domain.PermissionDenied(reason))
		return
	}
	writeJSON(w, status, map[string]any{
		"status":  "accepted",
		"traceId": trace.ID,
		"route":   routeType,
	})
}

func (s *Server) listTraces(w http.ResponseWriter, r *http.Request) {
	rows, err := s.repo.ListTraces(r.Context(), r.URL.Query().Get("runId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}
