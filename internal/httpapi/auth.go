package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
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
	CSRFToken     string `json:"csrfToken,omitempty"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	RequiresLogin bool   `json:"requiresLogin"`
}

const consoleSessionCSRFHeader = "X-AgentHarbor-CSRF"
const consoleLoginFailureWindow = 5 * time.Minute
const consoleLoginMaxFailures = 5

type consoleLoginFailure struct {
	Count      int
	WindowEnds time.Time
}

func (s *Server) getAuthSession(w http.ResponseWriter, r *http.Request) {
	setConsoleAuthNoStore(w)
	principal, expiresAt, ok := s.consoleSessionFromRequest(r)
	response := s.consoleSessionResponse(principal, expiresAt, ok)
	if ok {
		if sessionToken, tokenOK := consoleSessionTokenFromRequest(r); tokenOK {
			response.CSRFToken = s.consoleSessionCSRFToken(sessionToken)
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	setConsoleAuthNoStore(w)
	if s.developmentAdminBypassActive() {
		writeJSON(w, http.StatusOK, s.consoleSessionResponse(platformAdminPrincipal(developmentAdminActor), time.Time{}, true))
		return
	}

	var req consoleLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if retryAfterSeconds := s.consoleLoginRetryAfterSeconds(r); retryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
		writeError(w, domain.TooManyRequests("RATE_LIMITED", "too many failed console login attempts; retry later"))
		return
	}
	principal, ok := s.adminPrincipalForKey(r.Context(), req.AdminKey)
	if !ok {
		s.recordConsoleLoginFailure(r)
		writeError(w, domain.Unauthorized("missing or invalid admin key"))
		return
	}
	s.clearConsoleLoginFailures(r)
	expiresAt := s.now().Add(defaultConsoleSessionTTL).UTC()
	token, err := s.signConsoleSession(principal, expiresAt)
	if err != nil {
		writeError(w, err)
		return
	}
	http.SetCookie(w, s.consoleSessionCookie(token, expiresAt, false, r))
	response := s.consoleSessionResponse(principal, expiresAt, true)
	response.CSRFToken = s.consoleSessionCSRFToken(token)
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	setConsoleAuthNoStore(w)
	if sessionToken, ok := consoleSessionTokenFromRequest(r); ok {
		if _, _, valid := s.verifyConsoleSession(r.Context(), sessionToken); valid {
			if err := s.validateConsoleSessionCSRF(r, sessionToken); err != nil {
				writeError(w, err)
				return
			}
		}
	}
	http.SetCookie(w, s.consoleSessionCookie("", time.Unix(0, 0).UTC(), true, r))
	principal, expiresAt, ok := s.developmentSession()
	writeJSON(w, http.StatusOK, s.consoleSessionResponse(principal, expiresAt, ok))
}

func setConsoleAuthNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

func (s *Server) consoleSessionFromRequest(r *http.Request) (adminPrincipal, time.Time, bool) {
	if principal, expiresAt, ok := s.developmentSession(); ok {
		return principal, expiresAt, true
	}
	sessionToken, ok := consoleSessionTokenFromRequest(r)
	if !ok {
		return adminPrincipal{}, time.Time{}, false
	}
	return s.verifyConsoleSession(r.Context(), sessionToken)
}

func consoleSessionTokenFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(consoleSessionCookieName)
	if err != nil {
		return "", false
	}
	token := strings.TrimSpace(cookie.Value)
	if token == "" {
		return "", false
	}
	return token, true
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

func (s *Server) consoleSessionCSRFToken(sessionToken string) string {
	mac := hmac.New(sha256.New, s.consoleSessionSecret())
	_, _ = mac.Write([]byte("csrf:v1:"))
	_, _ = mac.Write([]byte(sessionToken))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) validateConsoleSessionCSRF(r *http.Request, sessionToken string) error {
	if !requiresCSRFProtection(r.Method) {
		return nil
	}
	expected := s.consoleSessionCSRFToken(sessionToken)
	provided := strings.TrimSpace(r.Header.Get(consoleSessionCSRFHeader))
	if provided == "" || !hmac.Equal([]byte(expected), []byte(provided)) {
		return domain.PermissionDenied("console session csrf token is required")
	}
	return nil
}

func requiresCSRFProtection(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (s *Server) consoleLoginClientKey(r *http.Request) string {
	remoteHost := consoleLoginRemoteHost(r)
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if trustedConsoleForwardedHeaderSource(remoteHost) {
		for _, part := range strings.Split(forwardedFor, ",") {
			if candidate := strings.TrimSpace(part); candidate != "" {
				return candidate
			}
		}
	}
	if remoteHost != "" {
		return remoteHost
	}
	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if remoteAddr == "" {
		return "unknown"
	}
	return remoteAddr
}

func consoleLoginRemoteHost(r *http.Request) string {
	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if remoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && strings.TrimSpace(host) != "" {
		return strings.TrimSpace(host)
	}
	return remoteAddr
}

func trustedConsoleForwardedHeaderSource(remoteHost string) bool {
	ip := net.ParseIP(strings.TrimSpace(remoteHost))
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

func (s *Server) consoleLoginRetryAfterSeconds(r *http.Request) int {
	key := s.consoleLoginClientKey(r)
	now := s.now()
	s.loginFailureMu.Lock()
	defer s.loginFailureMu.Unlock()
	if s.loginFailures == nil {
		return 0
	}
	failure, ok := s.loginFailures[key]
	if !ok {
		return 0
	}
	if !failure.WindowEnds.After(now) {
		delete(s.loginFailures, key)
		return 0
	}
	if failure.Count >= consoleLoginMaxFailures {
		return ceilDurationSeconds(failure.WindowEnds.Sub(now))
	}
	return 0
}

func (s *Server) recordConsoleLoginFailure(r *http.Request) {
	key := s.consoleLoginClientKey(r)
	now := s.now()
	s.loginFailureMu.Lock()
	defer s.loginFailureMu.Unlock()
	if s.loginFailures == nil {
		s.loginFailures = map[string]consoleLoginFailure{}
	}
	failure := s.loginFailures[key]
	if !failure.WindowEnds.After(now) {
		failure = consoleLoginFailure{WindowEnds: now.Add(consoleLoginFailureWindow)}
	}
	failure.Count++
	s.loginFailures[key] = failure
}

func (s *Server) clearConsoleLoginFailures(r *http.Request) {
	key := s.consoleLoginClientKey(r)
	s.loginFailureMu.Lock()
	defer s.loginFailureMu.Unlock()
	if s.loginFailures != nil {
		delete(s.loginFailures, key)
	}
}

func ceilDurationSeconds(duration time.Duration) int {
	if duration <= 0 {
		return 0
	}
	return int((duration + time.Second - 1) / time.Second)
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
	if !trustedConsoleForwardedHeaderSource(consoleLoginRemoteHost(r)) {
		return false
	}
	if forwardedProtoHeaderIsHTTPS(r.Header.Get("X-Forwarded-Proto")) {
		return true
	}
	return forwardedHeaderProtoIsHTTPS(r.Header.Get("Forwarded"))
}

func forwardedProtoHeaderIsHTTPS(header string) bool {
	for _, part := range strings.Split(header, ",") {
		if candidate := strings.TrimSpace(part); candidate != "" {
			return strings.EqualFold(candidate, "https")
		}
	}
	return false
}

func forwardedHeaderProtoIsHTTPS(header string) bool {
	for _, entry := range strings.Split(header, ",") {
		for _, param := range strings.Split(entry, ";") {
			key, value, ok := strings.Cut(strings.TrimSpace(param), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "proto") {
				continue
			}
			value = strings.Trim(strings.TrimSpace(value), `"`)
			return strings.EqualFold(value, "https")
		}
	}
	return false
}
