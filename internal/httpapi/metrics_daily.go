package httpapi

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const (
	defaultDailyMetricsDays   = 7
	maxDailyMetricsDays       = 30
	minDailyMetricsTZOffset   = -720
	maxDailyMetricsTZOffset   = 840
	dailyMetricsTraceRowLimit = 20000
	dailyMetricsAuditRowLimit = 5000
	dailyMetricsDateLayout    = "2006-01-02"
)

type dailyMetricsBucket struct {
	Date         string   `json:"date"`
	Calls        int      `json:"calls"`
	AllowedCalls int      `json:"allowedCalls"`
	DeniedCalls  int      `json:"deniedCalls"`
	DenyRate     *float64 `json:"denyRate"`
	AuditEvents  int      `json:"auditEvents"`
}

type dailyMetricsTotals struct {
	Calls        int `json:"calls"`
	AllowedCalls int `json:"allowedCalls"`
	DeniedCalls  int `json:"deniedCalls"`
	AuditEvents  int `json:"auditEvents"`
}

type dailyMetricsResponse struct {
	Days            int                  `json:"days"`
	TZOffsetMinutes int                  `json:"tzOffsetMinutes"`
	From            string               `json:"from"`
	To              string               `json:"to"`
	GeneratedAt     time.Time            `json:"generatedAt"`
	Truncated       bool                 `json:"truncated"`
	Totals          dailyMetricsTotals   `json:"totals"`
	Buckets         []dailyMetricsBucket `json:"buckets"`
}

// dailyMetrics buckets gateway decisions and audit events per local day.
// Counting happens server-side over the same visibility rules as the list
// endpoints, so a chart never has to bucket a capped list in the browser.
func (s *Server) dailyMetrics(w http.ResponseWriter, r *http.Request) {
	scope, err := s.effectiveManagementScopeFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	days, err := boundedIntQueryParam(r, "days", defaultDailyMetricsDays, 1, maxDailyMetricsDays)
	if err != nil {
		writeError(w, err)
		return
	}
	offsetMinutes, err := boundedIntQueryParam(r, "tzOffsetMinutes", 0, minDailyMetricsTZOffset, maxDailyMetricsTZOffset)
	if err != nil {
		writeError(w, err)
		return
	}
	now := s.now()
	zone := time.FixedZone("", offsetMinutes*60)
	localNow := now.In(zone)
	lastDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, zone)
	firstDay := lastDay.AddDate(0, 0, -(days - 1))
	windowEnd := lastDay.AddDate(0, 0, 1)

	buckets := make([]dailyMetricsBucket, days)
	index := make(map[string]int, days)
	for i := range buckets {
		date := firstDay.AddDate(0, 0, i).Format(dailyMetricsDateLayout)
		buckets[i].Date = date
		index[date] = i
	}
	bucketFor := func(createdAt time.Time) (int, bool) {
		i, ok := index[createdAt.In(zone).Format(dailyMetricsDateLayout)]
		return i, ok
	}

	// One row past each cap tells a full window from a truncated one. Both
	// lists keep their newest rows, so truncation only thins the oldest days.
	traces, err := s.repo.ListTraces(r.Context(), store.TraceFilter{
		ManagementScope: scope,
		Since:           firstDay.UTC(),
		Until:           windowEnd.UTC(),
		Limit:           dailyMetricsTraceRowLimit + 1,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	events, err := s.repo.ListAuditEvents(r.Context(), store.AuditEventFilter{
		ManagementScope: scope,
		Since:           firstDay.UTC(),
		Until:           windowEnd.UTC(),
		Limit:           dailyMetricsAuditRowLimit + 1,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	truncated := false
	if len(traces) > dailyMetricsTraceRowLimit {
		traces = traces[len(traces)-dailyMetricsTraceRowLimit:]
		truncated = true
	}
	if len(events) > dailyMetricsAuditRowLimit {
		events = events[len(events)-dailyMetricsAuditRowLimit:]
		truncated = true
	}
	events, err = s.visibleAuditEvents(r.Context(), events, scope)
	if err != nil {
		writeError(w, err)
		return
	}

	var totals dailyMetricsTotals
	for _, trace := range traces {
		i, ok := bucketFor(trace.CreatedAt)
		if !ok {
			continue
		}
		buckets[i].Calls++
		totals.Calls++
		switch trace.Decision {
		case domain.TraceDecisionAllowed:
			buckets[i].AllowedCalls++
			totals.AllowedCalls++
		case domain.TraceDecisionDenied:
			buckets[i].DeniedCalls++
			totals.DeniedCalls++
		}
	}
	for _, event := range events {
		i, ok := bucketFor(event.CreatedAt)
		if !ok {
			continue
		}
		buckets[i].AuditEvents++
		totals.AuditEvents++
	}
	for i := range buckets {
		buckets[i].DenyRate = dailyDenyRate(buckets[i].DeniedCalls, buckets[i].Calls)
	}

	writeJSON(w, http.StatusOK, dailyMetricsResponse{
		Days:            days,
		TZOffsetMinutes: offsetMinutes,
		From:            buckets[0].Date,
		To:              buckets[len(buckets)-1].Date,
		GeneratedAt:     now,
		Truncated:       truncated,
		Totals:          totals,
		Buckets:         buckets,
	})
}

// dailyDenyRate is nil on days without calls so charts can show a gap instead
// of a misleading 0% deny rate.
func dailyDenyRate(denied int, calls int) *float64 {
	if calls == 0 {
		return nil
	}
	rate := math.Round(float64(denied)/float64(calls)*10000) / 10000
	return &rate
}

func boundedIntQueryParam(r *http.Request, name string, fallback int, minValue int, maxValue int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue || value > maxValue {
		return 0, domain.BadRequest("VALIDATION_FAILED", name+" must be between "+strconv.Itoa(minValue)+" and "+strconv.Itoa(maxValue))
	}
	return value, nil
}
