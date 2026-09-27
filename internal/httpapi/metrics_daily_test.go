package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type dailyMetricsResult struct {
	Days            int       `json:"days"`
	TZOffsetMinutes int       `json:"tzOffsetMinutes"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	GeneratedAt     time.Time `json:"generatedAt"`
	Truncated       bool      `json:"truncated"`
	Totals          struct {
		Calls        int `json:"calls"`
		AllowedCalls int `json:"allowedCalls"`
		DeniedCalls  int `json:"deniedCalls"`
		AuditEvents  int `json:"auditEvents"`
	} `json:"totals"`
	Buckets []dailyMetricsBucketResult `json:"buckets"`
}

type dailyMetricsBucketResult struct {
	Date         string   `json:"date"`
	Calls        int      `json:"calls"`
	AllowedCalls int      `json:"allowedCalls"`
	DeniedCalls  int      `json:"deniedCalls"`
	DenyRate     *float64 `json:"denyRate"`
	AuditEvents  int      `json:"auditEvents"`
}

func TestDailyMetricsBucketsCallsAndAuditEventsByLocalDay(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 9, 25, 20, 30, 0, 0, time.UTC)
	router := httpapi.New(repo, httpapi.WithUnauthenticatedAdminAllowed(true), httpapi.WithClock(func() time.Time { return now })).Router()
	for i, trace := range []struct {
		createdAt string
		decision  domain.TraceDecision
	}{
		{createdAt: "2026-09-23T15:59:59Z", decision: domain.TraceDecisionAllowed},
		{createdAt: "2026-09-23T16:00:00Z", decision: domain.TraceDecisionAllowed},
		{createdAt: "2026-09-24T10:00:00Z", decision: domain.TraceDecisionDenied},
		{createdAt: "2026-09-24T11:00:00Z", decision: domain.TraceDecisionAllowed},
		{createdAt: "2026-09-25T17:00:00Z", decision: domain.TraceDecisionDenied},
	} {
		appendMetricsTrace(t, repo, fmt.Sprintf("trc_metrics_%d", i), "tenant-metrics", "ws-metrics", trace.decision, mustParseRFC3339(t, trace.createdAt))
	}
	appendMetricsTrace(t, repo, "trc_metrics_other", "tenant-other", "ws-other", domain.TraceDecisionAllowed, mustParseRFC3339(t, "2026-09-24T10:30:00Z"))
	appendMetricsAudit(t, repo, "aud_metrics_a", "tenant-metrics", "ws-metrics", "agent.updated", "agent", mustParseRFC3339(t, "2026-09-24T01:00:00Z"))
	appendMetricsAudit(t, repo, "aud_metrics_b", "tenant-metrics", "ws-metrics", "agent.updated", "agent", mustParseRFC3339(t, "2026-09-25T20:00:00Z"))
	// An applied event whose application is not visible stays hidden from
	// the audit list, so it must not be counted either.
	appendMetricsAudit(t, repo, "aud_metrics_hidden", "tenant-metrics", "ws-metrics", "permission_package.applied", "permission_package", mustParseRFC3339(t, "2026-09-24T02:00:00Z"))
	appendMetricsAudit(t, repo, "aud_metrics_other", "tenant-other", "ws-other", "agent.updated", "agent", mustParseRFC3339(t, "2026-09-24T03:00:00Z"))
	scope := "tenantId=tenant-metrics&workspaceId=ws-metrics"

	shanghaiResp := request(t, router, http.MethodGet, "/api/v1/metrics/daily?"+scope+"&days=3&tzOffsetMinutes=480", nil, "")
	shanghai := decodeData[dailyMetricsResult](t, shanghaiResp)
	if shanghai.Days != 3 || shanghai.TZOffsetMinutes != 480 || shanghai.From != "2026-09-24" || shanghai.To != "2026-09-26" || !shanghai.GeneratedAt.Equal(now) || shanghai.Truncated {
		t.Fatalf("unexpected UTC+8 window: %#v", shanghai)
	}
	assertDailyBuckets(t, shanghai.Buckets, []dailyMetricsBucketResult{
		{Date: "2026-09-24", Calls: 3, AllowedCalls: 2, DeniedCalls: 1, DenyRate: float64Ptr(0.3333), AuditEvents: 1},
		{Date: "2026-09-25"},
		{Date: "2026-09-26", Calls: 1, DeniedCalls: 1, DenyRate: float64Ptr(1), AuditEvents: 1},
	})
	if shanghai.Totals.Calls != 4 || shanghai.Totals.AllowedCalls != 2 || shanghai.Totals.DeniedCalls != 2 || shanghai.Totals.AuditEvents != 2 {
		t.Fatalf("unexpected UTC+8 totals: %#v", shanghai.Totals)
	}
	if !strings.Contains(shanghaiResp.Body.String(), `{"date":"2026-09-25","calls":0,"allowedCalls":0,"deniedCalls":0,"denyRate":null,"auditEvents":0}`) {
		t.Fatalf("days without calls should serialize denyRate as null: %s", shanghaiResp.Body.String())
	}

	utc := decodeData[dailyMetricsResult](t, request(t, router, http.MethodGet, "/api/v1/metrics/daily?"+scope, nil, ""))
	if utc.Days != 7 || utc.TZOffsetMinutes != 0 || utc.From != "2026-09-19" || utc.To != "2026-09-25" || len(utc.Buckets) != 7 {
		t.Fatalf("default window should be the last seven UTC days: %#v", utc)
	}
	assertDailyBuckets(t, utc.Buckets[4:], []dailyMetricsBucketResult{
		{Date: "2026-09-23", Calls: 2, AllowedCalls: 2, DenyRate: float64Ptr(0)},
		{Date: "2026-09-24", Calls: 2, AllowedCalls: 1, DeniedCalls: 1, DenyRate: float64Ptr(0.5), AuditEvents: 1},
		{Date: "2026-09-25", Calls: 1, DeniedCalls: 1, DenyRate: float64Ptr(1), AuditEvents: 1},
	})
	for _, bucket := range utc.Buckets[:4] {
		if bucket.Calls != 0 || bucket.AuditEvents != 0 || bucket.DenyRate != nil {
			t.Fatalf("days without activity should be zero-filled, got %#v", bucket)
		}
	}
	if utc.Totals.Calls != 5 || utc.Totals.AllowedCalls != 3 || utc.Totals.DeniedCalls != 2 || utc.Totals.AuditEvents != 2 {
		t.Fatalf("unexpected UTC totals: %#v", utc.Totals)
	}

	unscoped := decodeData[dailyMetricsResult](t, request(t, router, http.MethodGet, "/api/v1/metrics/daily", nil, ""))
	if unscoped.Totals.Calls != 6 || unscoped.Totals.AuditEvents != 3 {
		t.Fatalf("unscoped metrics should include every visible row, got %#v", unscoped.Totals)
	}
}

func TestDailyMetricsFollowsAdminScope(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	router := httpapi.New(
		repo,
		httpapi.WithAdminIdentities([]httpapi.AdminIdentity{
			{Actor: "support-admin", Key: "support-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
		}),
		httpapi.WithClock(func() time.Time { return now }),
	).Router()
	createdAt := now.Add(-time.Hour)
	appendMetricsTrace(t, repo, "trc_scope_east", "tenant-east", "ws-support", domain.TraceDecisionAllowed, createdAt)
	appendMetricsTrace(t, repo, "trc_scope_west", "tenant-west", "ws-finance", domain.TraceDecisionDenied, createdAt)
	appendMetricsAudit(t, repo, "aud_scope_east", "tenant-east", "ws-support", "agent.updated", "agent", createdAt)
	appendMetricsAudit(t, repo, "aud_scope_west", "tenant-west", "ws-finance", "agent.updated", "agent", createdAt)

	metrics := decodeData[dailyMetricsResult](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/metrics/daily?days=1", nil, "", "support-key"))
	if metrics.Totals.Calls != 1 || metrics.Totals.AllowedCalls != 1 || metrics.Totals.DeniedCalls != 0 || metrics.Totals.AuditEvents != 1 {
		t.Fatalf("scoped admin metrics should only count in-scope rows, got %#v", metrics.Totals)
	}
	if resp := requestWithAdmin(t, router, http.MethodGet, "/api/v1/metrics/daily?tenantId=tenant-west", nil, "", "support-key"); resp.Code != http.StatusForbidden {
		t.Fatalf("scoped admin should not read another tenant's metrics, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestDailyMetricsMarksTruncatedWindowsAndKeepsNewestRows(t *testing.T) {
	repo := store.NewMemory()
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	router := httpapi.New(repo, httpapi.WithUnauthenticatedAdminAllowed(true), httpapi.WithClock(func() time.Time { return now })).Router()
	appendMetricsTrace(t, repo, "trc_truncate_oldest", "tenant-cap", "ws-cap", domain.TraceDecisionDenied, now.AddDate(0, 0, -1))
	const traceRowLimit = 20000
	for i := range traceRowLimit {
		appendMetricsTrace(t, repo, fmt.Sprintf("trc_truncate_%05d", i), "tenant-cap", "ws-cap", domain.TraceDecisionAllowed, now.Add(-time.Duration(i)*time.Second))
	}

	metrics := decodeData[dailyMetricsResult](t, request(t, router, http.MethodGet, "/api/v1/metrics/daily?days=2", nil, ""))
	if !metrics.Truncated {
		t.Fatalf("a window over the row cap should be marked truncated: %#v", metrics.Totals)
	}
	if metrics.Totals.Calls != traceRowLimit || metrics.Totals.DeniedCalls != 0 {
		t.Fatalf("truncation should drop the oldest rows first, got %#v", metrics.Totals)
	}
	if metrics.Buckets[0].Calls != 0 || metrics.Buckets[1].Calls != traceRowLimit {
		t.Fatalf("truncation should only thin the oldest day, got %#v", metrics.Buckets)
	}
}

func TestDailyMetricsRejectsOutOfRangeParameters(t *testing.T) {
	router := newRouter()
	for _, query := range []string{
		"days=0",
		"days=31",
		"days=week",
		"tzOffsetMinutes=-721",
		"tzOffsetMinutes=841",
		"tzOffsetMinutes=90.5",
	} {
		resp := request(t, router, http.MethodGet, "/api/v1/metrics/daily?"+query, nil, "")
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s should be rejected, got %d body=%s", query, resp.Code, resp.Body.String())
		}
		var env apiEnvelope
		if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode error envelope for %s: %v", query, err)
		}
		if env.Error != "VALIDATION_FAILED" {
			t.Fatalf("%s should fail validation, got %#v", query, env)
		}
	}
	for _, query := range []string{"days=1", "days=30", "tzOffsetMinutes=-720", "tzOffsetMinutes=840"} {
		if resp := request(t, router, http.MethodGet, "/api/v1/metrics/daily?"+query, nil, ""); resp.Code != http.StatusOK {
			t.Fatalf("%s is inside the documented range, got %d body=%s", query, resp.Code, resp.Body.String())
		}
	}
}

func appendMetricsTrace(t *testing.T, repo store.Repository, id string, tenantID string, workspaceID string, decision domain.TraceDecision, createdAt time.Time) {
	t.Helper()
	if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
		ID:          id,
		RunID:       "run-daily-metrics",
		RouteType:   "mcp",
		RouteKey:    "tools/call",
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		Decision:    decision,
		CreatedAt:   createdAt,
	}); err != nil {
		t.Fatalf("append trace %s: %v", id, err)
	}
}

func appendMetricsAudit(t *testing.T, repo store.Repository, id string, tenantID string, workspaceID string, action string, resourceType string, createdAt time.Time) {
	t.Helper()
	if _, err := repo.AppendAuditEvent(t.Context(), domain.AuditEvent{
		ID:           id,
		TenantID:     tenantID,
		WorkspaceID:  workspaceID,
		Actor:        "local-dev",
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   id + "_resource",
		CreatedAt:    createdAt,
	}); err != nil {
		t.Fatalf("append audit event %s: %v", id, err)
	}
}

func assertDailyBuckets(t *testing.T, got []dailyMetricsBucketResult, want []dailyMetricsBucketResult) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected daily buckets:\n got: %s\nwant: %s", mustJSON(t, got), mustJSON(t, want))
	}
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse %s: %v", value, err)
	}
	return parsed
}

func float64Ptr(value float64) *float64 {
	return &value
}
