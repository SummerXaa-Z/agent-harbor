package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

// Per-holder views (docs/engineering/0.6.0-per-holder-views-design.md): an
// admin identity may be bound to a set of caller agents; once bound, its user
// surface is narrowed to those callers server-side. Empty bindings keep the
// existing tenant-scope semantics.

const holderScopeDeniedCode = "HOLDER_SCOPE_DENIED"

// requireHolderScope is the single holder check covering every
// permission-package journey endpoint (wired into
// requirePermissionPackageDraftScope). The binding is looked up per request so
// rebinding takes effect immediately.
func (s *Server) requireHolderScope(r *http.Request, callerInstanceID string) error {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return nil
	}
	if principal.Role == adminRolePlatformAdmin {
		return nil
	}
	binding, active, err := s.holderBindingForPrincipal(r.Context(), principal)
	if err != nil {
		return err
	}
	if !active {
		return domain.PermissionDenied("admin identity is not active")
	}
	if len(binding) == 0 {
		return nil
	}
	for _, owned := range binding {
		if owned == callerInstanceID {
			return nil
		}
	}
	return domain.AppError{
		Status:  403,
		Code:    holderScopeDeniedCode,
		Message: "caller instance is outside this identity's holder scope",
	}
}

// holderOwnedAgentIDs reports the caller agents the request's admin identity
// is bound to, and whether holder narrowing is active (an active non-platform
// identity with a non-empty binding). List endpoints use it as a post-filter.
func (s *Server) holderOwnedAgentIDs(r *http.Request) ([]string, bool, error) {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return nil, false, nil
	}
	return s.holderOwnedAgentIDsForPrincipal(r.Context(), principal)
}

func (s *Server) holderOwnedAgentIDsForPrincipal(ctx context.Context, principal adminPrincipal) ([]string, bool, error) {
	if principal.Role == adminRolePlatformAdmin {
		return nil, false, nil
	}
	binding, active, err := s.holderBindingForPrincipal(ctx, principal)
	if err != nil || !active || len(binding) == 0 {
		return nil, false, err
	}
	owned := append([]string(nil), binding...)
	sort.Strings(owned)
	return owned, true, nil
}

// holderBindingForPrincipal resolves the holder binding for an actor. Managed
// identities carry their binding in the store; bootstrap identities are
// configuration-provided, cannot be bound, and therefore keep the unbound
// fallback. A missing or disabled managed identity is reported inactive.
func (s *Server) holderBindingForPrincipal(ctx context.Context, principal adminPrincipal) ([]string, bool, error) {
	identity, ok, err := s.repo.GetAdminIdentityByActor(ctx, principal.Actor)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		for _, bootstrap := range s.bootstrapAdminIdentities() {
			if bootstrap.Actor == principal.Actor && bootstrap.Status == domain.AdminIdentityStatusActive {
				return nil, true, nil
			}
		}
		return nil, false, nil
	}
	if identity.Status != domain.AdminIdentityStatusActive {
		return nil, false, nil
	}
	return identity.OwnedAgentIDs, true, nil
}

// normalizeOwnedAgentIDs trims, drops empties, dedupes and sorts a holder
// binding so stored values are stable.
func normalizeOwnedAgentIDs(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// agentKeysOwnedBy keeps only keys whose agent is holder-owned.
func agentKeysOwnedBy(rows []domain.AgentKey, owned []string) []domain.AgentKey {
	out := make([]domain.AgentKey, 0, len(rows))
	for _, row := range rows {
		if containsString(owned, row.AgentID) {
			out = append(out, row)
		}
	}
	return out
}

// callerInstancesOwnedBy keeps only rows whose caller instance is holder-owned.
func approvalRequestsOwnedBy(rows []domain.PermissionPackageApprovalRequest, owned []string) []domain.PermissionPackageApprovalRequest {
	out := make([]domain.PermissionPackageApprovalRequest, 0, len(rows))
	for _, row := range rows {
		if containsString(owned, row.CallerInstanceID) {
			out = append(out, row)
		}
	}
	return out
}

func applicationsOwnedBy(rows []domain.PermissionPackageApplication, owned []string) []domain.PermissionPackageApplication {
	out := make([]domain.PermissionPackageApplication, 0, len(rows))
	for _, row := range rows {
		if containsString(owned, row.CallerInstanceID) {
			out = append(out, row)
		}
	}
	return out
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
