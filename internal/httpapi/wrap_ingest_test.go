package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/SummerXaa-Z/agent-harbor/internal/wrap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func wrapIngestHarness(t *testing.T) (*store.Memory, http.Handler) {
	t.Helper()
	repo := store.NewMemory()
	for _, a := range []domain.Agent{{ID: "caller", TenantID: "t1", WorkspaceID: "w1"}, {ID: "target", TenantID: "t1", WorkspaceID: "w1"}, {ID: "outside", TenantID: "t2", WorkspaceID: "w2"}, {ID: "other-workspace", TenantID: "t1", WorkspaceID: "w2"}, {ID: "no-workspace", TenantID: "t1"}} {
		if _, err := repo.CreateAgent(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}
	return repo, httpapi.New(repo, httpapi.WithAdminKey("platform-key"), httpapi.WithAdminIdentities([]httpapi.AdminIdentity{{Actor: "operator", Key: "tenant-key", Role: "tenant_admin", TenantID: "t1", WorkspaceID: "w1"}, {Actor: "reviewer", Key: "review-key", Role: "security_reviewer", TenantID: "t1"}})).Router()
}
func wrapIngestEvent(id string) wrap.Event {
	return wrap.Event{ID: id, CallerID: "caller", TargetID: "target", Capability: "search_notes", Method: "tools/call", Decision: "allowed", Outcome: "error", Source: "wrap", CreatedAt: time.Date(2026, 11, 13, 1, 2, 3, 123456789, time.UTC), ArgsDigest: "sha256:" + strings.Repeat("a", 64)}
}
func postWrapBatch(t *testing.T, router http.Handler, key string, events []wrap.Event) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"events": events})
	if err != nil {
		t.Fatal(err)
	}
	return postWrapRaw(router, key, raw)
}
func postWrapRaw(router http.Handler, key string, raw []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/v1/audit/traces/ingest", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Key", key)
	out := httptest.NewRecorder()
	router.ServeHTTP(out, req)
	return out
}
func TestWrapIngestScopedRetryAndTimeline(t *testing.T) {
	repo, router := wrapIngestHarness(t)
	e := wrapIngestEvent("trc_wrap_1")
	for range 2 {
		out := postWrapBatch(t, router, "tenant-key", []wrap.Event{e})
		if out.Code != 200 {
			t.Fatalf("%d: %s", out.Code, out.Body)
		}
	}
	traces, err := repo.ListTraces(context.Background(), store.TraceFilter{ManagementScope: store.ManagementScope{TenantID: "t1", WorkspaceID: "w1"}})
	if err != nil || len(traces) != 1 {
		t.Fatalf("scoped timeline: %+v %v", traces, err)
	}
	got := traces[0]
	if got.Source != "wrap" || got.Outcome != "error" || got.ToolName != "search_notes" || got.CapabilityID != "" || got.TenantID != "t1" || got.WorkspaceID != "w1" {
		t.Fatalf("incorrect contract: %+v", got)
	}
	req := httptest.NewRequest("GET", "/api/v1/audit/traces?traceId="+e.ID, nil)
	req.Header.Set("X-Admin-Key", "tenant-key")
	out := httptest.NewRecorder()
	router.ServeHTTP(out, req)
	if out.Code != 200 || !strings.Contains(out.Body.String(), `"source":"wrap"`) || !strings.Contains(out.Body.String(), `"toolName":"search_notes"`) {
		t.Fatalf("audit lookup: %d %s", out.Code, out.Body)
	}
	e.Reason = "different evidence"
	out = postWrapBatch(t, router, "tenant-key", []wrap.Event{wrapIngestEvent("new-id"), e})
	if out.Code != 409 {
		t.Fatalf("collision: %d %s", out.Code, out.Body)
	}
	traces, _ = repo.ListTraces(context.Background(), store.TraceFilter{})
	if len(traces) != 1 {
		t.Fatal("partial commit on conflict")
	}
}
func TestWrapIngestRejectsAuthenticationAndScope(t *testing.T) {
	for _, tc := range []struct {
		name, key, caller, target string
		status                    int
	}{{"missing auth", "", "caller", "target", 401}, {"wrong auth", "bad", "caller", "target", 401}, {"review only", "review-key", "caller", "target", 403}, {"foreign caller", "tenant-key", "outside", "target", 403}, {"foreign target", "tenant-key", "caller", "outside", 403}, {"workspace target", "tenant-key", "caller", "other-workspace", 403}, {"empty workspace", "tenant-key", "caller", "no-workspace", 403}, {"unknown target", "platform-key", "caller", "missing", 403}} {
		t.Run(tc.name, func(t *testing.T) {
			repo, router := wrapIngestHarness(t)
			e := wrapIngestEvent("bad")
			e.CallerID = tc.caller
			e.TargetID = tc.target
			out := postWrapBatch(t, router, tc.key, []wrap.Event{wrapIngestEvent("valid-first"), e})
			if out.Code != tc.status {
				t.Fatalf("%d: %s", out.Code, out.Body)
			}
			rows, _ := repo.ListTraces(context.Background(), store.TraceFilter{})
			if len(rows) != 0 {
				t.Fatal("partial writes before validation")
			}
		})
	}
}
func TestWrapIngestValidatesMetadataAndNoPlaintextArguments(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*wrap.Event)
	}{{"source", func(e *wrap.Event) { e.Source = "gateway" }}, {"method", func(e *wrap.Event) { e.Method = "initialize" }}, {"missing outcome", func(e *wrap.Event) { e.Outcome = "" }}, {"decision", func(e *wrap.Event) { e.Decision = "yes" }}, {"duration", func(e *wrap.Event) { e.DurationMs = -1 }}, {"timestamp", func(e *wrap.Event) { e.CreatedAt = time.Time{} }}, {"raw digest", func(e *wrap.Event) { e.ArgsDigest = "secret argument" }}, {"control id", func(e *wrap.Event) { e.ID = "a\nb" }}} {
		t.Run(tc.name, func(t *testing.T) {
			_, router := wrapIngestHarness(t)
			e := wrapIngestEvent("bad")
			tc.change(&e)
			out := postWrapBatch(t, router, "platform-key", []wrap.Event{e})
			if out.Code != 400 {
				t.Fatalf("%d %s", out.Code, out.Body)
			}
		})
	}
	_, router := wrapIngestHarness(t)
	for _, raw := range []string{`{"events":[],"tenantId":"spoof"}`, `{"events":[{"arguments":{"secret":"value"}}]}`, `{"events":[],"events":[]}`} {
		if out := postWrapRaw(router, "platform-key", []byte(raw)); out.Code != 400 {
			t.Fatalf("accepted unknown/ambiguous payload: %d", out.Code)
		}
	}
	many := make([]wrap.Event, 101)
	for i := range many {
		many[i] = wrapIngestEvent("many")
	}
	if out := postWrapBatch(t, router, "platform-key", many); out.Code != 400 {
		t.Fatal("accepted unbounded batch")
	}
}
func TestWrapIngestLegacyDeniedOutcome(t *testing.T) {
	repo, router := wrapIngestHarness(t)
	e := wrapIngestEvent("denied")
	e.Decision = "denied"
	e.Outcome = ""
	out := postWrapBatch(t, router, "tenant-key", []wrap.Event{e})
	if out.Code != 200 {
		t.Fatalf("%d: %s", out.Code, out.Body)
	}
	rows, _ := repo.ListTraces(context.Background(), store.TraceFilter{})
	if len(rows) != 1 || rows[0].Outcome != "denied" {
		t.Fatalf("%+v", rows)
	}
}
