package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

type consoleLoginRequest struct {
	AdminKey string `json:"adminKey"`
}

type consoleSessionPayload struct {
	Actor       string `json:"actor"`
	Role        string `json:"role,omitempty"`
	TenantID    string `json:"tenantId,omitempty"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type consoleSessionResponse struct {
	Actor         string `json:"actor,omitempty"`
	Role          string `json:"role,omitempty"`
	TenantID      string `json:"tenantId,omitempty"`
	WorkspaceID   string `json:"workspaceId,omitempty"`
	Authenticated bool   `json:"authenticated"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	RequiresLogin bool   `json:"requiresLogin"`
}

func (s *Server) getAuthSession(w http.ResponseWriter, r *http.Request) {
	principal, expiresAt, ok := s.consoleSessionFromRequest(r)
	writeJSON(w, http.StatusOK, s.consoleSessionResponse(principal, expiresAt, ok))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.developmentAdminBypassActive() {
		writeJSON(w, http.StatusOK, s.consoleSessionResponse(platformAdminPrincipal(developmentAdminActor), time.Time{}, true))
		return
	}

	var req consoleLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	principal, ok := s.adminPrincipalForKey(r.Context(), req.AdminKey)
	if !ok {
		writeError(w, domain.Unauthorized("missing or invalid admin key"))
		return
	}
	expiresAt := s.now().Add(defaultConsoleSessionTTL).UTC()
	token, err := s.signConsoleSession(principal, expiresAt)
	if err != nil {
		writeError(w, err)
		return
	}
	http.SetCookie(w, s.consoleSessionCookie(token, expiresAt, false, r))
	writeJSON(w, http.StatusOK, s.consoleSessionResponse(principal, expiresAt, true))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, s.consoleSessionCookie("", time.Unix(0, 0).UTC(), true, r))
	principal, expiresAt, ok := s.developmentSession()
	writeJSON(w, http.StatusOK, s.consoleSessionResponse(principal, expiresAt, ok))
}

func (s *Server) consoleSessionFromRequest(r *http.Request) (adminPrincipal, time.Time, bool) {
	if principal, expiresAt, ok := s.developmentSession(); ok {
		return principal, expiresAt, true
	}
	cookie, err := r.Cookie(consoleSessionCookieName)
	if err != nil {
		return adminPrincipal{}, time.Time{}, false
	}
	return s.verifyConsoleSession(r.Context(), cookie.Value)
}

func (s *Server) developmentSession() (adminPrincipal, time.Time, bool) {
	if s.developmentAdminBypassActive() {
		return platformAdminPrincipal(developmentAdminActor), time.Time{}, true
	}
	return adminPrincipal{}, time.Time{}, false
}

func (s *Server) developmentAdminBypassActive() bool {
	return s.adminKey == "" && len(s.adminIdentities) == 0 && s.allowUnauthenticatedAdmin
}

func (s *Server) consoleSessionResponse(principal adminPrincipal, expiresAt time.Time, authenticated bool) consoleSessionResponse {
	principal = normalizeAdminPrincipal(principal)
	response := consoleSessionResponse{
		Actor:         principal.Actor,
		Role:          principal.Role,
		TenantID:      principal.TenantID,
		WorkspaceID:   principal.WorkspaceID,
		Authenticated: authenticated,
		RequiresLogin: !s.developmentAdminBypassActive(),
	}
	if !expiresAt.IsZero() {
		response.ExpiresAt = expiresAt.Format(time.RFC3339)
	}
	if !authenticated {
		response.Actor = ""
		response.Role = ""
		response.TenantID = ""
		response.WorkspaceID = ""
		response.ExpiresAt = ""
	}
	return response
}

func (s *Server) consoleSessionCookie(value string, expiresAt time.Time, clear bool, r *http.Request) *http.Cookie {
	cookie := &http.Cookie{
		Name:     consoleSessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		Secure:   isHTTPSRequest(r),
	}
	if clear {
		cookie.MaxAge = -1
	}
	return cookie
}

func (s *Server) signConsoleSession(principal adminPrincipal, expiresAt time.Time) (string, error) {
	principal = normalizeAdminPrincipal(principal)
	payload, err := json.Marshal(consoleSessionPayload{
		Actor:       principal.Actor,
		Role:        principal.Role,
		TenantID:    principal.TenantID,
		WorkspaceID: principal.WorkspaceID,
		ExpiresAt:   expiresAt.Unix(),
	})
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.consoleSessionSignature(encodedPayload)
	return "v1." + encodedPayload + "." + signature, nil
}

func (s *Server) verifyConsoleSession(ctx context.Context, token string) (adminPrincipal, time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return adminPrincipal{}, time.Time{}, false
	}
	expected := s.consoleSessionSignature(parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return adminPrincipal{}, time.Time{}, false
	}
	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return adminPrincipal{}, time.Time{}, false
	}
	var payload consoleSessionPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return adminPrincipal{}, time.Time{}, false
	}
	actor := strings.TrimSpace(payload.Actor)
	expiresAt := time.Unix(payload.ExpiresAt, 0).UTC()
	if actor == "" || !expiresAt.After(s.now()) {
		return adminPrincipal{}, time.Time{}, false
	}
	principal, ok := s.adminPrincipalForActor(ctx, actor)
	if !ok {
		return adminPrincipal{}, time.Time{}, false
	}
	return principal, expiresAt, true
}

func (s *Server) consoleSessionSignature(encodedPayload string) string {
	mac := hmac.New(sha256.New, s.consoleSessionSecret())
	_, _ = mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) consoleSessionSecret() []byte {
	if len(s.sessionSecret) > 0 {
		return s.sessionSecret
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("agent-harbor-console-session-v1"))
	if s.adminKey != "" {
		_, _ = hash.Write([]byte("|admin:"))
		_, _ = hash.Write([]byte(s.adminKey))
	}
	for _, identity := range s.adminIdentities {
		_, _ = hash.Write([]byte("|identity:"))
		_, _ = hash.Write([]byte(identity.Actor))
		_, _ = hash.Write([]byte("="))
		_, _ = hash.Write([]byte(identity.Key))
		_, _ = hash.Write([]byte("|role:"))
		_, _ = hash.Write([]byte(normalizeAdminRole(identity.Role)))
		_, _ = hash.Write([]byte("|tenant:"))
		_, _ = hash.Write([]byte(identity.TenantID))
		_, _ = hash.Write([]byte("|workspace:"))
		_, _ = hash.Write([]byte(identity.WorkspaceID))
	}
	return hash.Sum(nil)
}

func isHTTPSRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
