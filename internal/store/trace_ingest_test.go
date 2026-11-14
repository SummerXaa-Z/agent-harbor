package store_test

import (
	"context"
	"github.com/SummerXaa-Z/agent-harbor/internal/db"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"sync"
	"testing"
	"time"
)

type ingestRepo interface {
	traceRepo
	IngestTraces(context.Context, []domain.TraceEvent) error
	CreateTenant(context.Context, domain.Tenant) (domain.Tenant, error)
	CreateAgent(context.Context, domain.Agent) (domain.Agent, error)
}

func TestTraceIngestMemory(t *testing.T) { assertTraceIngest(t, store.NewMemory()) }
func TestTraceIngestPostgres(t *testing.T) {
	url := os.Getenv("AGENT_HARBOR_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set AGENT_HARBOR_TEST_DATABASE_URL for PostgreSQL integration")
	}
	pool, err := pgxpool.New(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	assertTraceIngest(t, store.NewPostgres(pool))
}
func assertTraceIngest(t *testing.T, repo ingestRepo) {
	t.Helper()
	ctx := t.Context()
	runID := security.NewID("run")
	event := domain.TraceEvent{ID: security.NewID("trc"), RunID: runID, CallerID: "caller", TargetID: "target", RouteType: "mcp_tool", RouteKey: "tools/call", ToolName: "search", Source: "wrap", Outcome: "error", Decision: domain.TraceDecisionAllowed, CreatedAt: time.Date(2026, 11, 13, 1, 2, 3, 123456789, time.UTC)}
	if err := repo.IngestTraces(ctx, []domain.TraceEvent{event, event}); err != nil {
		t.Fatal(err)
	}
	retry := event
	retry.CreatedAt = event.CreatedAt.In(time.FixedZone("test", 8*3600))
	retry.RiskFlags = []string{}
	retry.DataScopes = []domain.DataScope{}
	if err := repo.IngestTraces(ctx, []domain.TraceEvent{retry}); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ListTraces(ctx, store.TraceFilter{RunID: runID})
	if err != nil || len(rows) != 1 || rows[0].Source != "wrap" || rows[0].Outcome != "error" || rows[0].ToolName != "search" || rows[0].CapabilityID != "" {
		t.Fatalf("roundtrip: %+v %v", rows, err)
	}
	fresh := event
	fresh.ID = security.NewID("trc")
	conflict := event
	conflict.Outcome = "ok"
	if err := repo.IngestTraces(ctx, []domain.TraceEvent{fresh, conflict}); err == nil {
		t.Fatal("changed evidence accepted")
	}
	rows, err = repo.ListTraces(ctx, store.TraceFilter{RunID: runID})
	if err != nil || len(rows) != 1 || rows[0].Outcome != "error" {
		t.Fatalf("batch not rolled back: %+v %v", rows, err)
	}
	conflict = fresh
	conflict.Reason = "different"
	if err := repo.IngestTraces(ctx, []domain.TraceEvent{fresh, conflict}); err == nil {
		t.Fatal("intra-batch conflict accepted")
	}
	rows, _ = repo.ListTraces(ctx, store.TraceFilter{RunID: runID})
	if len(rows) != 1 {
		t.Fatal("intra-batch conflict wrote events")
	}
	gateway := event
	gateway.ID = security.NewID("trc")
	gateway.Source = ""
	gateway.Outcome = ""
	if _, err := repo.AppendTrace(ctx, gateway); err != nil {
		t.Fatal(err)
	}
	if err := repo.IngestTraces(ctx, []domain.TraceEvent{gateway}); err == nil {
		t.Fatal("gateway ID adopted")
	}
	rows, err = repo.ListTraces(ctx, store.TraceFilter{TraceID: gateway.ID})
	if err != nil || len(rows) != 1 || rows[0].Source != "gateway" || rows[0].Outcome != "" {
		t.Fatalf("legacy defaults: %+v %v", rows, err)
	}
	tenant := security.NewID("tenant")
	if _, err := repo.CreateTenant(ctx, domain.Tenant{ID: tenant, Name: tenant, Level: 1, Status: domain.TenantStatusActive, CreatedAt: event.CreatedAt, UpdatedAt: event.CreatedAt}); err != nil {
		t.Fatal(err)
	}
	scoped := event
	scoped.ID = security.NewID("trc")
	scoped.CallerID = security.NewID("agt")
	scoped.TargetID = security.NewID("agt")
	scoped.TenantID = tenant
	scoped.WorkspaceID = "wrap-scope"
	for _, id := range []string{scoped.CallerID, scoped.TargetID} {
		if _, err := repo.CreateAgent(ctx, domain.Agent{ID: id, Name: id, TenantID: tenant, WorkspaceID: scoped.WorkspaceID, Status: domain.AgentStatusActive, CreatedAt: event.CreatedAt, UpdatedAt: event.CreatedAt}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.IngestTraces(ctx, []domain.TraceEvent{scoped}); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []store.ManagementScope{{TenantID: tenant}, {TenantID: tenant, WorkspaceID: scoped.WorkspaceID}} {
		rows, err := repo.ListTraces(ctx, store.TraceFilter{TraceID: scoped.ID, ManagementScope: scope})
		if err != nil || len(rows) != 1 || rows[0].ToolName != "search" {
			t.Fatalf("scope visibility: %+v %v", rows, err)
		}
	}
	rows, err = repo.ListTraces(ctx, store.TraceFilter{TraceID: scoped.ID, ManagementScope: store.ManagementScope{WorkspaceID: "other"}})
	if err != nil || len(rows) != 0 {
		t.Fatalf("scope leakage: %+v %v", rows, err)
	}
	concurrent := event
	concurrent.ID = security.NewID("trc")
	var wg sync.WaitGroup
	errors := make(chan error, 8)
	for range 8 {
		wg.Go(func() { errors <- repo.IngestTraces(ctx, []domain.TraceEvent{concurrent}) })
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	rows, err = repo.ListTraces(ctx, store.TraceFilter{TraceID: concurrent.ID})
	if err != nil || len(rows) != 1 {
		t.Fatalf("concurrent retry duplicated: %+v %v", rows, err)
	}
}
