package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
)

func (s *Server) requirePlatformAdmin(r *http.Request) (adminPrincipal, error) {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return adminPrincipal{}, domain.Unauthorized("admin authentication is required")
	}
	principal = normalizeAdminPrincipal(principal)
	if principal.Role != adminRolePlatformAdmin {
		return adminPrincipal{}, domain.PermissionDenied("platform administrator is required")
	}
	return principal, nil
}

func (s *Server) listAdminIdentities(w http.ResponseWriter, r *http.Request) {
	rows, err := s.adminIdentityRowsForPlatform(r)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) adminIdentityRowsForPlatform(r *http.Request) ([]domain.AdminIdentity, error) {
	if _, err := s.requirePlatformAdmin(r); err != nil {
		return nil, err
	}
	managed, err := s.repo.ListAdminIdentities(r.Context())
	if err != nil {
		return nil, err
	}
	rows := append(s.bootstrapAdminIdentities(), managed...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Source != rows[j].Source {
			return rows[i].Source < rows[j].Source
		}
		return rows[i].Actor < rows[j].Actor
	})
	return rows, nil
}

func (s *Server) createAdminIdentity(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateAdminIdentityRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	created, err := s.createManagedAdminIdentity(r, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) rotateAdminIdentityKey(w http.ResponseWriter, r *http.Request) {
	rotated, err := s.rotateManagedAdminIdentityKey(r, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rotated)
}

func (s *Server) disableAdminIdentity(w http.ResponseWriter, r *http.Request) {
	disabled, err := s.disableManagedAdminIdentity(r, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, disabled)
}

func (s *Server) createManagedAdminIdentity(r *http.Request, req domain.CreateAdminIdentityRequest) (domain.CreateAdminIdentityResponse, error) {
	principal, err := s.requirePlatformAdmin(r)
	if err != nil {
		return domain.CreateAdminIdentityResponse{}, err
	}
	req, err = normalizeManagedAdminIdentityRequest(req)
	if err != nil {
		return domain.CreateAdminIdentityResponse{}, err
	}
	exists, err := s.adminActorExists(r.Context(), req.Actor)
	if err != nil {
		return domain.CreateAdminIdentityResponse{}, err
	}
	if exists {
		return domain.CreateAdminIdentityResponse{}, domain.BadRequest("VALIDATION_FAILED", "admin identity actor already exists")
	}
	plaintext, prefix := security.NewAdminKey()
	now := s.now()
	identity := domain.AdminIdentity{
		ID:          security.NewID("adm"),
		Actor:       req.Actor,
		DisplayName: req.DisplayName,
		Role:        req.Role,
		TenantID:    req.TenantID,
		WorkspaceID: req.WorkspaceID,
		Status:      domain.AdminIdentityStatusActive,
		Source:      domain.AdminIdentitySourceManaged,
		KeyHash:     security.HashSecret(plaintext),
		KeyPrefix:   prefix,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   principal.Actor,
		UpdatedBy:   principal.Actor,
	}
	created, err := s.repo.CreateAdminIdentityWithAudit(r.Context(), identity, func(created domain.AdminIdentity) domain.AuditEvent {
		return s.managementAuditEvent(r, created.TenantID, created.WorkspaceID, "admin_identity.created", "admin_identity", created.ID, "Admin identity created", adminIdentityAuditMetadata(created))
	})
	if err != nil {
		return domain.CreateAdminIdentityResponse{}, err
	}
	return domain.CreateAdminIdentityResponse{Identity: created, Key: plaintext}, nil
}

func (s *Server) rotateManagedAdminIdentityKey(r *http.Request, id string) (domain.RotateAdminIdentityKeyResponse, error) {
	principal, err := s.requirePlatformAdmin(r)
	if err != nil {
		return domain.RotateAdminIdentityKeyResponse{}, err
	}
	identity, err := s.loadMutableManagedAdminIdentity(r.Context(), id)
	if err != nil {
		return domain.RotateAdminIdentityKeyResponse{}, err
	}
	if identity.Role == domain.AdminIdentityRolePlatformAdmin && identity.Status == domain.AdminIdentityStatusActive {
		if err := s.ensureAnotherPlatformAdmin(r.Context(), identity); err != nil {
			return domain.RotateAdminIdentityKeyResponse{}, err
		}
	}
	plaintext, prefix := security.NewAdminKey()
	rotated, ok, err := s.repo.RotateAdminIdentityKeyWithAudit(r.Context(), identity.ID, security.HashSecret(plaintext), prefix, s.now(), principal.Actor, func(rotated domain.AdminIdentity) domain.AuditEvent {
		return s.managementAuditEvent(r, rotated.TenantID, rotated.WorkspaceID, "admin_identity.key_rotated", "admin_identity", rotated.ID, "Admin identity key rotated", adminIdentityAuditMetadata(rotated))
	})
	if err != nil {
		return domain.RotateAdminIdentityKeyResponse{}, err
	}
	if !ok {
		return domain.RotateAdminIdentityKeyResponse{}, domain.NotFound("admin identity not found")
	}
	return domain.RotateAdminIdentityKeyResponse{Identity: rotated, Key: plaintext}, nil
}

func (s *Server) disableManagedAdminIdentity(r *http.Request, id string) (domain.AdminIdentity, error) {
	principal, err := s.requirePlatformAdmin(r)
	if err != nil {
		return domain.AdminIdentity{}, err
	}
	identity, err := s.loadMutableManagedAdminIdentity(r.Context(), id)
	if err != nil {
		return domain.AdminIdentity{}, err
	}
	if identity.Actor == principal.Actor {
		return domain.AdminIdentity{}, domain.PermissionDenied("cannot disable the current administrator identity")
	}
	if identity.Role == domain.AdminIdentityRolePlatformAdmin && identity.Status == domain.AdminIdentityStatusActive {
		if err := s.ensureAnotherPlatformAdmin(r.Context(), identity); err != nil {
			return domain.AdminIdentity{}, err
		}
	}
	disabled, ok, err := s.repo.DisableAdminIdentityWithAudit(r.Context(), identity.ID, s.now(), principal.Actor, func(disabled domain.AdminIdentity) domain.AuditEvent {
		return s.managementAuditEvent(r, disabled.TenantID, disabled.WorkspaceID, "admin_identity.disabled", "admin_identity", disabled.ID, "Admin identity disabled", adminIdentityAuditMetadata(disabled))
	})
	if err != nil {
		return domain.AdminIdentity{}, err
	}
	if !ok {
		return domain.AdminIdentity{}, domain.NotFound("admin identity not found")
	}
	return disabled, nil
}

func (s *Server) loadMutableManagedAdminIdentity(ctx context.Context, id string) (domain.AdminIdentity, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.AdminIdentity{}, domain.BadRequest("VALIDATION_FAILED", "admin identity id is required")
	}
	if strings.HasPrefix(id, "bootstrap:") {
		return domain.AdminIdentity{}, domain.BadRequest("VALIDATION_FAILED", "bootstrap administrator identities are read-only")
	}
	identity, ok, err := s.repo.GetAdminIdentity(ctx, id)
	if err != nil {
		return domain.AdminIdentity{}, err
	}
	if !ok {
		return domain.AdminIdentity{}, domain.NotFound("admin identity not found")
	}
	return identity, nil
}

func (s *Server) adminActorExists(ctx context.Context, actor string) (bool, error) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return false, nil
	}
	if s.bootstrapAdminActorExists(actor) {
		return true, nil
	}
	_, ok, err := s.repo.GetAdminIdentityByActor(ctx, actor)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *Server) bootstrapAdminActorExists(actor string) bool {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return false
	}
	for _, identity := range s.adminIdentities {
		if identity.Actor == actor {
			return true
		}
	}
	if actor == "admin-key" && s.adminKey != "" {
		return true
	}
	if actor == developmentAdminActor && s.developmentAdminBypassActive() {
		return true
	}
	return false
}

func (s *Server) bootstrapAdminIdentities() []domain.AdminIdentity {
	rows := make([]domain.AdminIdentity, 0, len(s.adminIdentities)+1)
	now := s.now()
	for _, identity := range s.adminIdentities {
		principal := identity.principal()
		rows = append(rows, domain.AdminIdentity{
			ID:          "bootstrap:" + principal.Actor,
			Actor:       principal.Actor,
			DisplayName: principal.Actor,
			Role:        domain.AdminIdentityRole(principal.Role),
			TenantID:    principal.TenantID,
			WorkspaceID: principal.WorkspaceID,
			Status:      domain.AdminIdentityStatusActive,
			Source:      domain.AdminIdentitySourceBootstrap,
			KeyPrefix:   "bootstrap",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	if s.adminKey != "" {
		rows = append(rows, domain.AdminIdentity{
			ID:          "bootstrap:admin-key",
			Actor:       "admin-key",
			DisplayName: "Bootstrap administrator",
			Role:        domain.AdminIdentityRolePlatformAdmin,
			Status:      domain.AdminIdentityStatusActive,
			Source:      domain.AdminIdentitySourceBootstrap,
			KeyPrefix:   "bootstrap",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return rows
}

func normalizeManagedAdminIdentityRequest(req domain.CreateAdminIdentityRequest) (domain.CreateAdminIdentityRequest, error) {
	req.Actor = strings.TrimSpace(req.Actor)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	if req.Actor == "" {
		return req, domain.BadRequest("VALIDATION_FAILED", "actor is required")
	}
	if domain.ReservedAdminIdentityActor(req.Actor) {
		return req, domain.BadRequest("VALIDATION_FAILED", "actor is reserved and cannot be used for a managed administrator")
	}
	if !domain.ValidAdminIdentityActor(req.Actor) {
		return req, domain.BadRequest("VALIDATION_FAILED", domain.AdminIdentityActorSyntaxMessage)
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Actor
	}
	switch req.Role {
	case "", domain.AdminIdentityRolePlatformAdmin:
		req.Role = domain.AdminIdentityRolePlatformAdmin
		req.TenantID = ""
		req.WorkspaceID = ""
	case domain.AdminIdentityRoleTenantAdmin, domain.AdminIdentityRoleSecurityReviewer:
		if req.TenantID == "" {
			return req, domain.BadRequest("VALIDATION_FAILED", "tenantId is required for scoped administrator roles")
		}
	default:
		return req, domain.BadRequest("VALIDATION_FAILED", "role must be platform_admin, tenant_admin, or security_reviewer")
	}
	return req, nil
}

func (s *Server) activePlatformAdminCount(ctx context.Context) (int, error) {
	count := 0
	for _, identity := range s.bootstrapAdminIdentities() {
		if identity.Role == domain.AdminIdentityRolePlatformAdmin && identity.Status == domain.AdminIdentityStatusActive {
			count++
		}
	}
	managed, err := s.repo.ListAdminIdentities(ctx)
	if err != nil {
		return 0, err
	}
	for _, identity := range managed {
		if identity.Role == domain.AdminIdentityRolePlatformAdmin && identity.Status == domain.AdminIdentityStatusActive {
			count++
		}
	}
	return count, nil
}

func (s *Server) ensureAnotherPlatformAdmin(ctx context.Context, identity domain.AdminIdentity) error {
	if identity.Role != domain.AdminIdentityRolePlatformAdmin || identity.Status != domain.AdminIdentityStatusActive {
		return nil
	}
	count, err := s.activePlatformAdminCount(ctx)
	if err != nil {
		return err
	}
	if count <= 1 {
		return domain.BadRequest("VALIDATION_FAILED", "at least one active platform administrator is required")
	}
	return nil
}

func adminPrincipalFromManagedIdentity(identity domain.AdminIdentity) adminPrincipal {
	return adminPrincipal{
		Actor:       strings.TrimSpace(identity.Actor),
		Role:        normalizeAdminRole(string(identity.Role)),
		TenantID:    strings.TrimSpace(identity.TenantID),
		WorkspaceID: strings.TrimSpace(identity.WorkspaceID),
	}
}

func adminIdentityAuditMetadata(identity domain.AdminIdentity) map[string]any {
	return map[string]any{
		"adminIdentityId": identity.ID,
		"actor":           identity.Actor,
		"displayName":     identity.DisplayName,
		"role":            identity.Role,
		"tenantId":        identity.TenantID,
		"workspaceId":     identity.WorkspaceID,
		"status":          identity.Status,
		"source":          identity.Source,
		"keyPrefix":       identity.KeyPrefix,
		"createdBy":       identity.CreatedBy,
		"updatedBy":       identity.UpdatedBy,
		"disabledBy":      identity.DisabledBy,
	}
}
