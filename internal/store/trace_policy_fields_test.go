package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/db"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The v2 policy fields (args digest, risk flags, rule ref) must round-trip
// through both stores and feed the high-risk filter that powers
// `harbor audit --high-risk` and the console red markers.
func TestTracePolicyFieldsRoundTripMemory(t *testing.T) {
	repo := store.NewMemory()
	assertTracePolicyFields(t, repo)
}

func TestTracePolicyFieldsRoundTripPostgres(t *testing.T) {
	databaseURL := os.Getenv("AGENT_HARBOR_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set AGENT_HARBOR_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	assertTracePolicyFields(t, store.NewPostgres(pool))
}

type traceRepo interface {
	AppendTrace(context.Context, domain.TraceEvent) (domain.TraceEvent, error)
	ListTraces(context.Context, store.TraceFilter) ([]domain.TraceEvent, error)
}

func assertTracePolicyFields(t *testing.T, repo traceRepo) {
	t.Helper()
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Microsecond)
	base := domain.TraceEvent{
		RunID:     security.NewID("run"),
		TargetID:  security.NewID("agt"),
		RouteType: "mcp_tool",
		Decision:  domain.TraceDecisionAllowed,
		Reason:    "default-open",
		CreatedAt: now,
	}
	flagged := base
	flagged.ID = security.NewID("trace")
	flagged.CapabilityID = security.NewID("cap")
	flagged.ArgsDigest = "sha256:9f3e"
	flagged.RiskFlags = []string{"delete"}
	flagged.RuleRef = "harbor.yaml:18"
	plain := base
	plain.ID = security.NewID("trace")
	plain.CreatedAt = now.Add(time.Millisecond)

	if _, err := repo.AppendTrace(ctx, flagged); err != nil {
		t.Fatalf("append flagged trace: %v", err)
	}
	if _, err := repo.AppendTrace(ctx, plain); err != nil {
		t.Fatalf("append plain trace: %v", err)
	}

	// The suite shares one database (schema resets once), so scope both
	// queries to this run's unique id instead of asserting global counts.
	highRisk, err := repo.ListTraces(ctx, store.TraceFilter{RunID: base.RunID, HighRiskOnly: true})
	if err != nil {
		t.Fatalf("list high-risk traces: %v", err)
	}
	if len(highRisk) != 1 || highRisk[0].ID != flagged.ID {
		t.Fatalf("high-risk filter should return only the flagged trace: %+v", highRisk)
	}
	got := highRisk[0]
	if got.ArgsDigest != flagged.ArgsDigest || got.RuleRef != flagged.RuleRef {
		t.Fatalf("policy fields did not round-trip: %+v", got)
	}
	if len(got.RiskFlags) != 1 || got.RiskFlags[0] != "delete" {
		t.Fatalf("risk flags did not round-trip: %+v", got.RiskFlags)
	}

	all, err := repo.ListTraces(ctx, store.TraceFilter{RunID: base.RunID})
	if err != nil {
		t.Fatalf("list all traces: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("unfiltered list should return both traces: %d", len(all))
	}
}
