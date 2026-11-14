package httpapi

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/wrap"
)

// ingestWrapTraces imports operator-reported evidence, not gateway decisions.
// Scope is derived from registered identities, never uploaded tenant fields.
func (s *Server) ingestWrapTraces(w http.ResponseWriter, r *http.Request) {
	principal, ok := requestAdminPrincipal(r)
	if !ok || (principal.Role != adminRolePlatformAdmin && principal.Role != adminRoleTenantAdmin) {
		writeError(w, domain.PermissionDenied("audit import requires an operator administrator"))
		return
	}
	var request struct {
		Events []wrap.Event `json:"events"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	if len(request.Events) == 0 || len(request.Events) > 100 {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "events must contain between 1 and 100 records"))
		return
	}
	events := make([]domain.TraceEvent, 0, len(request.Events))
	for i, event := range request.Events {
		if err := validateWrapEvent(event); err != nil {
			writeError(w, domain.BadRequest("VALIDATION_FAILED", fmt.Sprintf("event %d: %s", i, err)))
			return
		}
		caller, found, err := s.repo.GetAgent(r.Context(), event.CallerID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !found {
			writeError(w, domain.PermissionDenied("audit caller or target is unknown or outside authenticated admin scope"))
			return
		}
		if err := s.requireAgentManagementScope(r, caller); err != nil {
			writeError(w, err)
			return
		}
		target, found, err := s.repo.GetAgent(r.Context(), event.TargetID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !found {
			writeError(w, domain.PermissionDenied("audit caller or target is unknown or outside authenticated admin scope"))
			return
		}
		if err := s.requireAgentManagementScope(r, target); err != nil {
			writeError(w, err)
			return
		}
		// Imports from workspace-restricted operators require both concrete workspaces.
		if principal.Role != adminRolePlatformAdmin && principal.WorkspaceID != "" &&
			(caller.WorkspaceID != principal.WorkspaceID || target.WorkspaceID != principal.WorkspaceID) {
			writeError(w, domain.PermissionDenied("audit identities are outside authenticated workspace"))
			return
		}
		outcome := event.Outcome
		if event.Decision == "denied" && outcome == "" {
			outcome = "denied"
		}
		events = append(events, domain.TraceEvent{
			ID: event.ID, CallerID: caller.ID, TargetID: target.ID,
			TenantID: caller.TenantID, WorkspaceID: caller.WorkspaceID,
			RouteType: "mcp", RouteKey: event.Method, ToolName: event.Capability,
			Decision: domain.TraceDecision(event.Decision), Reason: event.Reason,
			RuleRef: event.RuleRef, ArgsDigest: event.ArgsDigest, RiskFlags: event.RiskFlags,
			DurationMs: event.DurationMs, CreatedAt: event.CreatedAt.UTC(), Source: "wrap", Outcome: outcome,
		})
	}
	if err := s.repo.IngestTraces(r.Context(), events); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"accepted": len(events)})
}

func validateWrapEvent(event wrap.Event) error {
	for _, field := range []struct {
		name, value string
		limit       int
	}{
		{"id", event.ID, 128}, {"callerAgentId", event.CallerID, 256},
		{"targetAgentId", event.TargetID, 256}, {"capabilityId", event.Capability, 256},
	} {
		if field.value == "" || len(field.value) > field.limit || strings.IndexFunc(field.value, unicode.IsControl) >= 0 {
			return fmt.Errorf("invalid %s", field.name)
		}
	}
	if event.Source != "wrap" || event.Method != "tools/call" {
		return fmt.Errorf("source must be wrap and routeKey must be tools/call")
	}
	if event.Decision != "allowed" && event.Decision != "denied" {
		return fmt.Errorf("invalid decision")
	}
	if event.Decision == "allowed" && event.Outcome != "ok" && event.Outcome != "error" {
		return fmt.Errorf("allowed event requires ok or error outcome")
	}
	if event.Decision == "denied" && event.Outcome != "" && event.Outcome != "denied" {
		return fmt.Errorf("denied event requires denied outcome")
	}
	if event.CreatedAt.IsZero() || event.CreatedAt.Year() < 1970 || event.CreatedAt.Year() > 9999 || event.DurationMs < 0 {
		return fmt.Errorf("invalid timestamp or duration")
	}
	if len(event.Reason) > 4096 || len(event.RuleRef) > 1024 || len(event.RiskFlags) > 16 {
		return fmt.Errorf("event metadata exceeds limits")
	}
	for _, flag := range event.RiskFlags {
		if flag == "" || len(flag) > 128 || strings.IndexFunc(flag, unicode.IsControl) >= 0 {
			return fmt.Errorf("invalid risk flag")
		}
	}
	if event.ArgsDigest != "" {
		if !strings.HasPrefix(event.ArgsDigest, "sha256:") || len(event.ArgsDigest) != 71 {
			return fmt.Errorf("argsDigest must be a SHA-256 digest")
		}
		if _, err := hex.DecodeString(event.ArgsDigest[7:]); err != nil {
			return fmt.Errorf("argsDigest must be a SHA-256 digest")
		}
	}
	return nil
}
