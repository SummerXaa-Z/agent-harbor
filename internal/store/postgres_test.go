package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SummerXaa-Z/agent-harbor/internal/db"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestPostgresRepositoryRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("AGENT_HARBOR_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set AGENT_HARBOR_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := store.NewPostgres(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	caller := domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      "test",
		WorkspaceID:   "ws-pg",
		Name:          "PG Caller",
		ChannelType:   "local",
		ChannelConfig: map[string]any{"description": "integration"},
		Status:        domain.AgentStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	target := domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      "test",
		WorkspaceID:   "ws-pg",
		Name:          "PG Target",
		ChannelType:   "mcp",
		ChannelConfig: map[string]any{"endpoint": "https://api.example.com/mcp"},
		Status:        domain.AgentStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := repo.CreateAgent(ctx, caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	if _, err := repo.CreateAgent(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	agents, err := repo.ListAgents(ctx, store.AgentFilter{ManagementScope: store.ManagementScope{
		TenantID:    "test",
		WorkspaceID: "ws-pg",
	}})
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(agents) < 2 {
		t.Fatalf("expected persisted agents, got %#v", agents)
	}

	plaintext, prefix := security.NewAgentKey()
	key := domain.AgentKey{
		ID:        security.NewID("key"),
		AgentID:   caller.ID,
		Name:      "pg-key",
		Hash:      security.HashSecret(plaintext),
		Prefix:    prefix,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}
	if _, err := repo.CreateAgentKey(ctx, key); err != nil {
		t.Fatalf("create key: %v", err)
	}
	keys, err := repo.ListAgentKeys(ctx, store.ManagementScope{TenantID: "test", WorkspaceID: "ws-pg"})
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	if len(keys) == 0 {
		t.Fatalf("expected scoped key rows")
	}
	found, ok, err := repo.FindAgentByKeyHash(ctx, key.Hash, now)
	if err != nil {
		t.Fatalf("find by key: %v", err)
	}
	if !ok || found.ID != caller.ID {
		t.Fatalf("expected caller by key, got ok=%v agent=%#v", ok, found)
	}

	grant := domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  caller.ID,
		TargetID:  target.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		CreatedAt: now,
	}
	if _, err := repo.CreateAccessGrant(ctx, grant); err != nil {
		t.Fatalf("create grant: %v", err)
	}
	if !repo.HasGrant(ctx, caller.ID, target.ID, "mcp", "tools/call", now) {
		t.Fatalf("expected grant match")
	}
	grants, err := repo.ListAccessGrants(ctx, store.ManagementScope{TenantID: "test", WorkspaceID: "ws-pg"})
	if err != nil {
		t.Fatalf("list grants: %v", err)
	}
	if len(grants) == 0 {
		t.Fatalf("expected grant rows")
	}

	trace := domain.TraceEvent{
		ID:        security.NewID("trc"),
		RunID:     "pg-run",
		CallerID:  caller.ID,
		TargetID:  target.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		Decision:  domain.TraceDecisionAllowed,
		Reason:    "integration",
		CreatedAt: now,
	}
	if _, err := repo.AppendTrace(ctx, trace); err != nil {
		t.Fatalf("append trace: %v", err)
	}
	traces, err := repo.ListTraces(ctx, store.TraceFilter{
		RunID:    "pg-run",
		Decision: domain.TraceDecisionAllowed,
		CallerID: caller.ID,
		TargetID: target.ID,
	})
	if err != nil {
		t.Fatalf("list traces: %v", err)
	}
	if len(traces) != 1 || traces[0].ID != trace.ID {
		t.Fatalf("unexpected traces: %#v", traces)
	}
	if _, ok, err := repo.RevokeAccessGrant(ctx, grant.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("revoke grant: %v", err)
	} else if !ok {
		t.Fatalf("expected grant to revoke")
	}
	if repo.HasGrant(ctx, caller.ID, target.ID, "mcp", "tools/call", now.Add(2*time.Minute)) {
		t.Fatalf("revoked grant should not match")
	}
	if _, ok, err := repo.DisableAgent(ctx, caller.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("disable agent: %v", err)
	} else if !ok {
		t.Fatalf("expected caller to disable")
	}
	if _, ok, err := repo.FindAgentByKeyHash(ctx, key.Hash, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("find disabled agent by key: %v", err)
	} else if ok {
		t.Fatalf("disabled agent key should no longer authenticate")
	}
}
