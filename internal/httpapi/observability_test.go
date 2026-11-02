package httpapi_test

import (
	"net/http"
	"strings"
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

func TestRuntimeMetricsSummarizeDataPlaneTraces(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	caller, key := createLocalCallerWithKey(t, router, "Runtime Metrics Caller")
	target := createDirectAgent(t, repo, "Runtime Metrics Target", "default", "ws-1", "mcp", domain.AgentStatusActive, nil)

	denied := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected denied call before grant, got %d body=%s", denied.Code, denied.Body.String())
	}
	grantRoute(t, router, caller.ID, target.ID, "mcp", "tools/call")
	allowed := request(t, router, http.MethodPost, "/api/v1/mcp/agents/"+target.ID+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
	}, key.Key)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected allowed call after grant, got %d body=%s", allowed.Code, allowed.Body.String())
	}

	metricsResp := request(t, router, http.MethodGet, "/api/v1/metrics/runtime?workspaceId=ws-1", nil, "")
	if metricsResp.Code != http.StatusOK {
		t.Fatalf("expected runtime metrics endpoint, got %d body=%s", metricsResp.Code, metricsResp.Body.String())
	}
	metrics := decodeData[[]metricResponse](t, metricsResp)
	if got := metricByID(t, metrics, "gateway_calls_total").Value; got != 2 {
		t.Fatalf("gateway_calls_total = %d, want 2; metrics=%#v", got, metrics)
	}
	if got := metricByID(t, metrics, "allowed_rate").Value; got != 50 {
		t.Fatalf("allowed_rate = %d, want 50; metrics=%#v", got, metrics)
	}
	if got := metricByID(t, metrics, "upstream_error_rate").Value; got != 0 {
		t.Fatalf("upstream_error_rate = %d, want 0; metrics=%#v", got, metrics)
	}
	if got := metricByID(t, metrics, "avg_latency_ms").Value; got != 0 {
		t.Fatalf("avg_latency_ms = %d, want 0 for local stub calls; metrics=%#v", got, metrics)
	}
}

func TestManagementAuditEventsFilterByScopeActionAndResource(t *testing.T) {
	router := newRouter()
	first := createAgent(t, router, map[string]any{
		"name":        "First Audited Agent",
		"tenantId":    "tenant-filter",
		"workspaceId": "ws-filter-a",
		"channelType": "local",
		"status":      "active",
	})
	second := createAgent(t, router, map[string]any{
		"name":        "Second Audited Agent",
		"tenantId":    "tenant-filter",
		"workspaceId": "ws-filter-b",
		"channelType": "local",
		"status":      "active",
	})
	patchResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+second.ID, map[string]any{
		"description": "only second agent was updated",
	}, "")
	if patchResp.Code != http.StatusOK {
		t.Fatalf("patch second audited agent failed: status=%d body=%s", patchResp.Code, patchResp.Body.String())
	}

	workspaceEvents := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?tenantId=tenant-filter&workspaceId=ws-filter-a", nil, ""))
	if len(workspaceEvents) != 1 || workspaceEvents[0].ResourceID != first.ID || workspaceEvents[0].Action != "agent.created" {
		t.Fatalf("workspace filter should return only first create event, got %#v", workspaceEvents)
	}

	updateEvents := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?action=agent.updated&resourceType=agent", nil, ""))
	if len(updateEvents) != 1 || updateEvents[0].ResourceID != second.ID {
		t.Fatalf("action/resourceType filter should return second update event, got %#v", updateEvents)
	}

	secondEvents := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?resourceId="+second.ID, nil, ""))
	if got := auditActions(secondEvents); strings.Join(got, ",") != "agent.created,agent.updated" {
		t.Fatalf("resourceId filter should return second lifecycle events, got %#v events=%#v", got, secondEvents)
	}

	limitedEvents := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?tenantId=tenant-filter&limit=1", nil, ""))
	if len(limitedEvents) != 1 {
		t.Fatalf("limit=1 should return one audit event, got %#v", limitedEvents)
	}
}
