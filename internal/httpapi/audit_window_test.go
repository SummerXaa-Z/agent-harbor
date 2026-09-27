package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestAuditEventsAndTracesSupportNewestLimitAndTimeWindows(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepo(repo)
	base := time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)
	auditIDs := make([]string, 5)
	wantTraceIDs := make([]string, 5)
	for minute := range 5 {
		createdAt := base.Add(time.Duration(minute) * time.Minute)
		auditIDs[minute] = "aud_window_" + string(rune('a'+minute))
		if _, err := repo.AppendAuditEvent(t.Context(), domain.AuditEvent{
			ID:           auditIDs[minute],
			TenantID:     "tenant-window",
			WorkspaceID:  "ws-window",
			Actor:        "local-dev",
			Action:       "agent.updated",
			ResourceType: "agent",
			ResourceID:   "agt_window",
			CreatedAt:    createdAt,
		}); err != nil {
			t.Fatalf("append audit event: %v", err)
		}
		wantTraceIDs[minute] = "trc_window_" + string(rune('a'+minute))
		if _, err := repo.AppendTrace(t.Context(), domain.TraceEvent{
			ID:          wantTraceIDs[minute],
			RunID:       "run-window",
			TargetID:    "agt_window",
			RouteType:   "mcp",
			RouteKey:    "tools/call",
			TenantID:    "tenant-window",
			WorkspaceID: "ws-window",
			Decision:    domain.TraceDecisionAllowed,
			CreatedAt:   createdAt,
		}); err != nil {
			t.Fatalf("append trace: %v", err)
		}
	}
	at := func(minute int) string {
		return base.Add(time.Duration(minute) * time.Minute).Format(time.RFC3339)
	}

	newest := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?limit=2", nil, ""))
	if got := auditEventIDs(newest); !reflect.DeepEqual(got, auditIDs[3:]) {
		t.Fatalf("audit limit should return the newest events ascending, got %#v", got)
	}
	windowed := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?since="+at(1)+"&until="+at(3), nil, ""))
	if got := auditEventIDs(windowed); !reflect.DeepEqual(got, auditIDs[1:3]) {
		t.Fatalf("audit since is inclusive and until exclusive, got %#v", got)
	}
	// A "+08:00" offset must be percent-encoded; a raw "+" decodes to a space.
	offsetSince := url.QueryEscape(base.Add(time.Minute).In(time.FixedZone("", 8*3600)).Format(time.RFC3339))
	offsetWindowed := decodeData[[]auditEventResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/events?since="+offsetSince, nil, ""))
	if got := auditEventIDs(offsetWindowed); !reflect.DeepEqual(got, auditIDs[1:]) {
		t.Fatalf("audit since should accept RFC3339 offsets, got %#v", got)
	}

	allTraces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-window", nil, ""))
	if got := traceIDs(allTraces); !reflect.DeepEqual(got, wantTraceIDs) {
		t.Fatalf("trace listing without limit should stay unbounded, got %#v", got)
	}
	newestTraces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-window&limit=2", nil, ""))
	if got := traceIDs(newestTraces); !reflect.DeepEqual(got, wantTraceIDs[3:]) {
		t.Fatalf("trace limit should return the newest traces ascending, got %#v", got)
	}
	windowedTraces := decodeData[[]traceResponse](t, request(t, router, http.MethodGet, "/api/v1/audit/traces?runId=run-window&since="+at(1)+"&until="+at(4)+"&limit=2", nil, ""))
	if got := traceIDs(windowedTraces); !reflect.DeepEqual(got, wantTraceIDs[2:4]) {
		t.Fatalf("trace window with limit should keep the newest rows in the window, got %#v", got)
	}
}

func TestAuditEventsAndTracesRejectInvalidWindows(t *testing.T) {
	router := newRouter()
	for _, path := range []string{
		"/api/v1/audit/events?since=yesterday",
		"/api/v1/audit/events?until=2026-09-20",
		"/api/v1/audit/events?since=2026-09-20T08:00:00Z&until=2026-09-20T08:00:00Z",
		"/api/v1/audit/events?since=2026-09-20T09:00:00Z&until=2026-09-20T08:00:00Z",
		"/api/v1/audit/traces?since=2026-09-20T08:00:00",
		"/api/v1/audit/traces?since=2026-09-20T09:00:00Z&until=2026-09-20T08:00:00Z",
		"/api/v1/audit/traces?limit=0",
		"/api/v1/audit/traces?limit=501",
		"/api/v1/audit/traces?limit=many",
	} {
		resp := request(t, router, http.MethodGet, path, nil, "")
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s should be rejected, got %d body=%s", path, resp.Code, resp.Body.String())
		}
		var env apiEnvelope
		if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode error envelope for %s: %v", path, err)
		}
		if env.Error != "VALIDATION_FAILED" {
			t.Fatalf("%s should fail validation, got %#v", path, env)
		}
	}
}

func auditEventIDs(events []auditEventResponse) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}
