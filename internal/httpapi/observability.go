package httpapi

import (
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) listTraces(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter := store.TraceFilter{
		ManagementScope: scope,
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
	filter.Since, filter.Until, err = timeWindowFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter.Limit, err = optionalTraceLimitFromRequest(r)
	if err != nil {
		writeError(w, err)
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
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter := store.AuditEventFilter{
		ManagementScope: scope,
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
	filter.Since, filter.Until, err = timeWindowFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListAuditEvents(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err = s.visibleAuditEvents(r.Context(), rows, scope)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) runtimeMetrics(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rows, err := s.repo.ListTraces(r.Context(), store.TraceFilter{ManagementScope: scope})
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

// optionalTraceLimitFromRequest keeps trace listing unbounded by default for
// existing callers; an explicit limit returns the newest traces.
func optionalTraceLimitFromRequest(r *http.Request) (int, error) {
	if strings.TrimSpace(r.URL.Query().Get("limit")) == "" {
		return 0, nil
	}
	return auditLimitFromRequest(r)
}

// timeWindowFromRequest parses the optional RFC3339 since (inclusive) and
// until (exclusive) bounds shared by audit event and trace listing.
func timeWindowFromRequest(r *http.Request) (time.Time, time.Time, error) {
	since, err := rfc3339QueryParam(r, "since")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	until, err := rfc3339QueryParam(r, "until")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !since.IsZero() && !until.IsZero() && !since.Before(until) {
		return time.Time{}, time.Time{}, domain.BadRequest("VALIDATION_FAILED", "since must be before until")
	}
	return since, until, nil
}

func rfc3339QueryParam(r *http.Request, name string) (time.Time, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return time.Time{}, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, domain.BadRequest("VALIDATION_FAILED", name+" must be an RFC3339 timestamp")
	}
	return value.UTC(), nil
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
