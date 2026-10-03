package httpapi

import (
	"context"
	"github.com/SummerXaa-Z/agent-harbor/internal/contracts"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/go-chi/chi/v5"
	"net/http"
	"sort"
	"strings"
	"time"
)

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
	if err := s.requireTenantManagementScope(r, tenant.ID); err != nil {
		writeError(w, err)
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
	if err := s.requireAgentManagementScope(r, agent); err != nil {
		writeError(w, err)
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
		ttl = defaultAgentKeyTTLSeconds
	} else if ttl < 0 || ttl > maxAgentKeyTTLSeconds {
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
	if owned, active, err := s.holderOwnedAgentIDs(r); err != nil {
		writeError(w, err)
		return
	} else if active {
		rows = agentKeysOwnedBy(rows, owned)
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
