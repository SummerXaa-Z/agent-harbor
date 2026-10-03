package httpapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/permissionpack"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/go-chi/chi/v5"
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

type upstreamRequestMutator func(*http.Request) error

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
			writeError(w, s.agentKeyAuthenticationError(r.Context(), token))
			return
		}
		key, ok, err := s.repo.FindAgentKeyByHash(r.Context(), security.HashSecret(token), s.now())
		if err != nil {
			writeError(w, err)
			return
		}
		if !ok || key.AgentID != caller.ID {
			writeError(w, domain.Unauthorized("invalid or expired bearer token"))
			return
		}
		ctx := context.WithValue(r.Context(), callerContextKey{}, caller)
		ctx = context.WithValue(ctx, agentKeyContextKey{}, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// agentKeyAuthenticationError distinguishes revoked and expired tokens from
// unknown ones. A hash match proves the caller held the token, so naming the
// concrete reason leaks nothing to guessed tokens, which stay on the generic
// invalid message.
func (s *Server) agentKeyAuthenticationError(ctx context.Context, token string) error {
	key, ok, err := s.repo.LookupAgentKeyByHash(ctx, security.HashSecret(token))
	if err != nil {
		return err
	}
	if !ok {
		return domain.Unauthorized("invalid or expired bearer token")
	}
	if !key.RevokedAt.IsZero() {
		return domain.Unauthorized("bearer token has been revoked")
	}
	if s.now().After(key.ExpiresAt) {
		return domain.Unauthorized("bearer token has expired")
	}
	return domain.Unauthorized("caller agent is not active")
}

func callerFromContext(ctx context.Context) domain.Agent {
	caller, _ := ctx.Value(callerContextKey{}).(domain.Agent)
	return caller
}

func agentKeyFromContext(ctx context.Context) domain.AgentKey {
	key, _ := ctx.Value(agentKeyContextKey{}).(domain.AgentKey)
	return key
}

func (s *Server) validateAccessHandoffKey(ctx context.Context, key domain.AgentKey, capabilityID string, subjectID string) error {
	application, err := s.accessHandoffApplicationForKey(ctx, key, subjectID)
	if err != nil || application == nil {
		return err
	}
	if !stringSliceContains(application.AllowedCapabilityIDs, strings.TrimSpace(capabilityID)) {
		return domain.PermissionDenied("access handoff token does not allow this capability")
	}
	return nil
}

func (s *Server) accessHandoffApplicationForKey(ctx context.Context, key domain.AgentKey, subjectID string) (*domain.PermissionPackageApplication, error) {
	if strings.TrimSpace(key.CreatedForHandoffID) == "" {
		return nil, nil
	}
	if !subjectSelectorMatchesRuntime(key.SubjectSelector, subjectID) {
		return nil, domain.PermissionDenied("access handoff token does not allow this subject")
	}
	applications, err := s.repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
		ID:    strings.TrimSpace(key.ApplicationID),
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(applications) != 1 {
		return nil, domain.PermissionDenied("access handoff token does not allow this capability")
	}
	application := applications[0]
	templateVersion, ok := currentPermissionPackageTemplateVersion(application.TemplateID)
	if !ok || application.CallerInstanceID != key.AgentID ||
		application.TemplateID != key.TemplateID ||
		application.TemplateVersion != templateVersion ||
		application.SubjectSelector != key.SubjectSelector {
		return nil, domain.PermissionDenied("access handoff token does not allow this capability")
	}
	capabilitiesCurrent, err := s.permissionPackageApplicationCapabilitiesCurrent(ctx, application)
	if err != nil {
		return nil, err
	}
	if !capabilitiesCurrent {
		return nil, domain.PermissionDenied("access handoff token references a stale permission package application")
	}
	return &application, nil
}

func currentPermissionPackageTemplateVersion(templateID string) (int, bool) {
	for _, template := range permissionpack.Templates() {
		if template.ID == templateID {
			return template.Version, true
		}
	}
	return 0, false
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func subjectSelectorMatchesRuntime(selector string, subjectID string) bool {
	selector = strings.TrimSpace(selector)
	subjectID = strings.TrimSpace(subjectID)
	if selector == "" || subjectID == "" {
		return false
	}
	if selector == subjectID {
		return true
	}
	return strings.HasSuffix(selector, "*") && strings.HasPrefix(subjectID, strings.TrimSuffix(selector, "*"))
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
	if isMCPProtocolLifecycleMethod(info.Method) {
		if s.handleMCPProtocolLifecycle(w, r, info) {
			return
		}
	}
	s.handleDataPlane(w, r, "mcp", info.Method)
}

// mcpSynthesizedProtocolVersion is answered for initialize requests that do
// not carry a client-requested protocolVersion.
const mcpSynthesizedProtocolVersion = "2025-03-26"

// isMCPProtocolLifecycleMethod reports whether the method is an MCP protocol
// handshake or lifecycle method rather than a capability invocation.
func isMCPProtocolLifecycleMethod(method string) bool {
	return method == "initialize" || method == "ping" || strings.HasPrefix(method, "notifications/")
}

// handleMCPProtocolLifecycle answers MCP protocol lifecycle methods so
// standards-compliant clients can complete their handshake against governed
// targets without an upstream round trip. Synthesized responses are local and
// never forwarded upstream. Explicit route-policy decisions still win: an
// allow falls through to the existing proxied path and an explicit deny falls
// through to the regular denied path. Access-handoff keys are bounded by their
// application binding (subject and target) instead of route policies, matching
// the other handoff-governed methods.
func (s *Server) handleMCPProtocolLifecycle(w http.ResponseWriter, r *http.Request, info mcpRequestInfo) bool {
	caller := callerFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	identity := identityFromRequest(r, caller)
	key := agentKeyFromContext(r.Context())
	handoffApplication, err := s.accessHandoffApplicationForKey(r.Context(), key, identity.SubjectID)
	if err == nil && handoffApplication != nil && handoffApplication.TargetID != targetID {
		err = domain.PermissionDenied("access handoff token does not allow this capability")
	}
	if err != nil {
		if _, traceErr := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:  identity,
			CallerID:  caller.ID,
			TargetID:  targetID,
			RouteType: "mcp",
			RouteKey:  info.Method,
			Decision:  domain.TraceDecisionDenied,
			Reason:    err.Error(),
		}); traceErr != nil {
			writeError(w, traceErr)
			return true
		}
		writeError(w, err)
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
	if handoffApplication == nil {
		decision, err := s.repo.EvaluateRouteAccess(r.Context(), caller.ID, targetID, "mcp", info.Method, s.now())
		if err != nil {
			writeError(w, err)
			return true
		}
		if decision.Allowed || decision.Source == "route_policy" {
			return false
		}
	}
	if info.Method == "initialize" {
		if _, traceErr := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:  identity,
			CallerID:  caller.ID,
			TargetID:  targetID,
			RouteType: "mcp",
			RouteKey:  info.Method,
			Decision:  domain.TraceDecisionAllowed,
			Reason:    "synthesized mcp initialize response",
		}); traceErr != nil {
			writeError(w, traceErr)
			return true
		}
	}
	writeMCPSynthesizedProtocolResponse(w, info)
	return true
}

func writeMCPSynthesizedProtocolResponse(w http.ResponseWriter, info mcpRequestInfo) {
	if strings.HasPrefix(info.Method, "notifications/") {
		setJSONResponseHeaders(w)
		w.WriteHeader(http.StatusAccepted)
		return
	}
	var payload struct {
		Params struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"params"`
	}
	if err := json.Unmarshal(info.Body, &payload); err != nil {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "mcp request body must be valid JSON"))
		return
	}
	var result map[string]any
	if info.Method == "initialize" {
		protocolVersion := strings.TrimSpace(payload.Params.ProtocolVersion)
		if protocolVersion == "" {
			protocolVersion = mcpSynthesizedProtocolVersion
		}
		result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "agent-harbor", "version": systemAPIVersion},
			"instructions":    "AgentHarbor governed access: only authorized tool capabilities are available on this endpoint.",
		}
	} else {
		result = map[string]any{}
	}
	// The JSON-RPC envelope is written raw, matching the proxied tools/list
	// path, so standards-compliant MCP clients can parse the response.
	setJSONResponseHeaders(w)
	w.WriteHeader(http.StatusOK)
	response := map[string]any{"jsonrpc": "2.0", "id": requestJSONRPCID(info.Body), "result": result}
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) openapiOperation(w http.ResponseWriter, r *http.Request) {
	s.handleDataPlane(w, r, "openapi", chi.URLParam(r, "operationId"))
}

func (s *Server) openapiRelativePath(w http.ResponseWriter, r *http.Request) {
	relativePath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if err := validateOpenAPIRelativePath(relativePath); err != nil {
		writeError(w, err)
		return
	}
	s.handleDataPlane(w, r, "openapi", relativePath)
}

func (s *Server) handleDataPlane(w http.ResponseWriter, r *http.Request, routeType string, routeKey string) {
	caller := callerFromContext(r.Context())
	if strings.TrimSpace(agentKeyFromContext(r.Context()).CreatedForHandoffID) != "" {
		writeError(w, domain.PermissionDenied("access handoff token only allows listed MCP tool capabilities"))
		return
	}
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
	if err := s.validateAccessHandoffKey(r.Context(), agentKeyFromContext(r.Context()), capability.ID, identity.SubjectID); err != nil {
		if _, traceErr := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:   identity,
			CallerID:   caller.ID,
			TargetID:   targetID,
			RouteType:  "mcp",
			RouteKey:   info.Method,
			Decision:   domain.TraceDecisionDenied,
			Reason:     err.Error(),
			Capability: capability,
		}); traceErr != nil {
			writeError(w, traceErr)
			return true
		}
		writeError(w, err)
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
	identity := identityFromRequest(r, caller)
	key := agentKeyFromContext(r.Context())
	handoffApplication, err := s.accessHandoffApplicationForKey(r.Context(), key, identity.SubjectID)
	if err == nil && handoffApplication != nil && handoffApplication.TargetID != targetID {
		err = domain.PermissionDenied("access handoff token does not allow this capability")
	}
	if err != nil {
		if _, traceErr := s.recordCapabilityTrace(r, traceRecordInput{
			Identity:  identity,
			CallerID:  caller.ID,
			TargetID:  targetID,
			RouteType: "mcp",
			RouteKey:  info.Method,
			Decision:  domain.TraceDecisionDenied,
			Reason:    err.Error(),
		}); traceErr != nil {
			writeError(w, traceErr)
			return true
		}
		writeError(w, err)
		return true
	}
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
	allowedTools := map[string]domain.Capability{}
	for _, capability := range capabilities {
		if capability.Type != domain.CapabilityTypeMCPTool {
			continue
		}
		if handoffApplication != nil && !stringSliceContains(handoffApplication.AllowedCapabilityIDs, capability.ID) {
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
		resp, err := doUpstreamRequest(req)
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
	if err := validateOpenAPIRelativePath(relativePath); err != nil {
		return "", err
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

func validateOpenAPIRelativePath(relativePath string) error {
	if relativePath == "" || strings.Contains(relativePath, "://") {
		return domain.BadRequest("VALIDATION_FAILED", "openapi relative path is invalid")
	}
	decoded := relativePath
	for range 3 {
		next, err := url.PathUnescape(decoded)
		if err != nil {
			return domain.BadRequest("VALIDATION_FAILED", "openapi relative path is invalid")
		}
		if next == decoded {
			break
		}
		decoded = next
	}
	if strings.Contains(decoded, "..") || strings.Contains(decoded, "://") {
		return domain.BadRequest("VALIDATION_FAILED", "openapi relative path is invalid")
	}
	return nil
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
	if err := requireJSONContentType(r); err != nil {
		return mcpRequestInfo{}, err
	}
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
	resp, err := doUpstreamRequest(req)
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
		return domain.UpstreamTimeout(upstreamErrorWithCause("upstream request timed out", err))
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return domain.UpstreamError(upstreamErrorWithCause("upstream request canceled", err))
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return domain.UpstreamDNSError(upstreamErrorWithCause("upstream DNS lookup failed", err))
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
		return domain.UpstreamTLSError(upstreamErrorWithCause("upstream TLS handshake failed", err))
	}
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE) ||
		strings.Contains(strings.ToLower(err.Error()), "connection refused") ||
		strings.Contains(strings.ToLower(err.Error()), "connection reset") {
		return domain.UpstreamConnectError(upstreamErrorWithCause("upstream connection failed", err))
	}
	return domain.UpstreamError(upstreamErrorWithCause("upstream request failed", err))
}

// upstreamErrorWithCause keeps the stable classification prefix while appending
// the underlying transport error (dial/refused/DNS host), which is what lets an
// administrator tell a wrong URL from a wrong port or a credential problem.
func upstreamErrorWithCause(message string, err error) string {
	if err == nil {
		return message
	}
	cause := strings.TrimSpace(err.Error())
	if cause == "" || cause == message {
		return message
	}
	return message + ": " + cause
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
