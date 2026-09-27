package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

const (
	targetProbeStatusOK    = "ok"
	targetProbeStatusError = "error"
	maxTargetProbeTimeout  = 5 * time.Second
)

type targetProbeResponse struct {
	TargetID   string    `json:"targetId"`
	Endpoint   string    `json:"endpoint"`
	Status     string    `json:"status"`
	ErrorCode  string    `json:"errorCode,omitempty"`
	Message    string    `json:"message,omitempty"`
	HTTPStatus int       `json:"httpStatus"`
	ToolCount  int       `json:"toolCount"`
	DurationMs int64     `json:"durationMs"`
	CheckedAt  time.Time `json:"checkedAt"`
}

// probeTarget sends the same tools/list call as a capability refresh but
// never writes capabilities or audit events. Reachability failures are part
// of the result (HTTP 200) so the console can show the classified cause —
// connection refused, DNS, TLS, timeout — instead of a generic fetch error.
func (s *Server) probeTarget(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "target probe currently supports mcp targets only"))
		return
	}
	endpoint := mcpTargetEndpoint(target)
	if endpoint == "" {
		writeError(w, domain.BadRequest("VALIDATION_FAILED", "mcp target requires channelConfig.endpoint for probing"))
		return
	}
	timeout, err := proxyTimeoutFromConfig(target.ChannelConfig)
	if err != nil {
		writeError(w, err)
		return
	}
	if timeout > maxTargetProbeTimeout {
		timeout = maxTargetProbeTimeout
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	req, err := newMCPToolsListRequest(ctx, target, endpoint, "target-probe")
	if err != nil && !errors.Is(err, errMCPToolsListRequestPrepare) {
		// Header and credential mapping errors are agent configuration
		// problems to fix on the record, not reachability results.
		writeError(w, err)
		return
	}

	result := targetProbeResponse{
		TargetID: target.ID,
		Endpoint: endpointWithoutUserinfo(endpoint),
		Status:   targetProbeStatusOK,
	}
	started := time.Now()
	var probeErr error
	if err != nil {
		probeErr = domain.UpstreamError("target probe request could not be prepared")
	} else {
		result.HTTPStatus, result.ToolCount, probeErr = probeMCPToolsList(ctx, req)
	}
	result.DurationMs = time.Since(started).Milliseconds()
	result.CheckedAt = s.now()
	if probeErr != nil {
		var appErr domain.AppError
		if !errors.As(probeErr, &appErr) {
			appErr = domain.UpstreamError(upstreamErrorWithCause("target probe failed", probeErr))
		}
		result.Status = targetProbeStatusError
		result.ErrorCode = appErr.Code
		result.Message = appErr.Message
	}
	writeJSON(w, http.StatusOK, result)
}

// probeMCPToolsList returns the upstream HTTP status (0 when no response
// arrived) and the advertised tool count. Every failure carries an UPSTREAM_*
// code so the console can localize it like a failed capability refresh.
func probeMCPToolsList(ctx context.Context, req *http.Request) (int, int, error) {
	resp, err := doUpstreamRequest(req)
	if err != nil {
		// The endpoint is reported separately without userinfo; unwrapping
		// keeps the request URL (and any username in it) out of the message.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return 0, 0, classifyUpstreamError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.StatusCode, 0, domain.UpstreamError("target probe upstream returned non-2xx status")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProxyBodyBytes+1))
	if err != nil {
		return resp.StatusCode, 0, classifyUpstreamError(ctx, err)
	}
	if len(body) > maxProxyBodyBytes {
		return resp.StatusCode, 0, domain.UpstreamError("target probe response exceeds 4MiB")
	}
	var tools mcpToolsListResponse
	if err := json.Unmarshal(body, &tools); err != nil {
		return resp.StatusCode, 0, domain.UpstreamError("target probe response must be valid MCP tools/list JSON")
	}
	return resp.StatusCode, len(tools.Result.Tools), nil
}

// endpointWithoutUserinfo drops userinfo from a registered URL: probe results
// are meant to be shown and shared, unlike the agent record itself.
func endpointWithoutUserinfo(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.User == nil {
		return endpoint
	}
	parsed.User = nil
	return parsed.String()
}
