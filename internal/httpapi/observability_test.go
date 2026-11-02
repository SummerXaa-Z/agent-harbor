package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type runtimeMetricResult struct {
	ID     string `json:"id"`
	Value  int    `json:"value"`
	Unit   string `json:"unit"`
	Status string `json:"status"`
}

func TestRuntimeMetricsSummarizeTrailingWindow(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	router := httpapi.New(repo, httpapi.WithUnauthenticatedAdminAllowed(true), httpapi.WithClock(func() time.Time { return now })).Router()
	appendMetricsTrace(t, repo, "trc_rt_denied", "tenant-rt", "ws-rt", domain.TraceDecisionDenied, now.Add(-2*time.Hour))
	appendMetricsTrace(t, repo, "trc_rt_allowed", "tenant-rt", "ws-rt", domain.TraceDecisionAllowed, now.Add(-1*time.Hour))
	appendMetricsTrace(t, repo, "trc_rt_stale_denied", "tenant-rt", "ws-rt", domain.TraceDecisionDenied, now.Add(-25*time.Hour))

	metrics := decodeData[[]runtimeMetricResult](t, request(t, router, http.MethodGet, "/api/v1/metrics/runtime", nil, ""))
	values := map[string]runtimeMetricResult{}
	for _, metric := range metrics {
		values[metric.ID] = metric
	}
	if got := values["gateway_calls_total"].Value; got != 2 {
		t.Fatalf("gateway_calls_total should count only traces inside the trailing window, got %d", got)
	}
	if got := values["allowed_rate"].Value; got != 50 {
		t.Fatalf("allowed_rate should be computed over the trailing window, got %d", got)
	}
	if got := values["avg_latency_ms"].Value; got != 0 {
		t.Fatalf("avg_latency_ms should stay zero when no fresh trace carries latency, got %d", got)
	}
}
