package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/db"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMemoryAdminIdentityOwnedAgentsRoundtrip(t *testing.T) {
	repo := NewMemory()
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	identity := domain.AdminIdentity{
		ID:            "adm_owned_memory",
		Actor:         "owned-admin",
		Role:          domain.AdminIdentityRoleTenantAdmin,
		TenantID:      "tenant-east",
		Status:        domain.AdminIdentityStatusActive,
		Source:        domain.AdminIdentitySourceManaged,
		KeyHash:       security.HashSecret("owned-secret"),
		OwnedAgentIDs: []string{"agt_a", "agt_b"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if _, err := repo.CreateAdminIdentityWithAudit(ctx, identity, func(created domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: "aud_owned_create", Actor: "platform", Action: "admin_identity.created", ResourceType: "admin_identity", ResourceID: created.ID, CreatedAt: now}
	}); err != nil {
		t.Fatalf("create admin identity: %v", err)
	}
	byActor, ok, err := repo.GetAdminIdentityByActor(ctx, "owned-admin")
	if err != nil || !ok || len(byActor.OwnedAgentIDs) != 2 || byActor.OwnedAgentIDs[0] != "agt_a" || byActor.OwnedAgentIDs[1] != "agt_b" {
		t.Fatalf("owned agents should roundtrip, ok=%v identity=%#v err=%v", ok, byActor, err)
	}

	updatedAt := now.Add(time.Minute)
	updated, ok, err := repo.UpdateAdminIdentityOwnedAgentsWithAudit(ctx, identity.ID, []string{"agt_c"}, updatedAt, "platform", func(updated domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: "aud_owned_update", Actor: "platform", Action: "admin_identity.updated", ResourceType: "admin_identity", ResourceID: updated.ID, CreatedAt: updatedAt}
	})
	if err != nil || !ok {
		t.Fatalf("update owned agents: ok=%v identity=%#v err=%v", ok, updated, err)
	}
	if len(updated.OwnedAgentIDs) != 1 || updated.OwnedAgentIDs[0] != "agt_c" || !updated.UpdatedAt.Equal(updatedAt) || updated.UpdatedBy != "platform" {
		t.Fatalf("unexpected updated admin identity: %#v", updated)
	}

	missing, ok, err := repo.UpdateAdminIdentityOwnedAgentsWithAudit(ctx, "adm_missing", nil, updatedAt, "platform", func(updated domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: "aud_owned_missing", Action: "admin_identity.updated", ResourceType: "admin_identity", ResourceID: updated.ID, CreatedAt: updatedAt}
	})
	if err != nil || ok || missing.ID != "" {
		t.Fatalf("updating a missing identity should be a silent miss, ok=%v identity=%#v err=%v", ok, missing, err)
	}

	events, err := repo.ListAuditEvents(ctx, AuditEventFilter{ResourceType: "admin_identity", ResourceID: identity.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	actions := make([]string, 0, len(events))
	for _, event := range events {
		actions = append(actions, event.Action)
	}
	if len(actions) != 2 || actions[0] != "admin_identity.created" || actions[1] != "admin_identity.updated" {
		t.Fatalf("expected create+update audit events, got %#v", actions)
	}
}

func TestPostgresAdminIdentityOwnedAgents(t *testing.T) {
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

	repo := NewPostgresWithCredentialKey(pool, []byte("0123456789abcdef0123456789abcdef"))
	now := time.Now().UTC().Truncate(time.Microsecond)
	identityID := security.NewID("adm")
	actor := "pg-owned-admin-" + identityID
	identity := domain.AdminIdentity{
		ID:            identityID,
		Actor:         actor,
		DisplayName:   "PG Owned Admin",
		Role:          domain.AdminIdentityRoleTenantAdmin,
		TenantID:      "tenant-east",
		Status:        domain.AdminIdentityStatusActive,
		Source:        domain.AdminIdentitySourceManaged,
		KeyHash:       security.HashSecret("pg-owned-secret-" + identityID),
		KeyPrefix:     "ahadm_pg_owned",
		OwnedAgentIDs: []string{"agt_owned_a", "agt_owned_b"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	created, err := repo.CreateAdminIdentityWithAudit(ctx, identity, func(created domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: security.NewID("aud"), Actor: "platform", Action: "admin_identity.created", ResourceType: "admin_identity", ResourceID: created.ID, CreatedAt: now}
	})
	if err != nil {
		t.Fatalf("create admin identity: %v", err)
	}
	if len(created.OwnedAgentIDs) != 2 || created.OwnedAgentIDs[0] != "agt_owned_a" || created.OwnedAgentIDs[1] != "agt_owned_b" {
		t.Fatalf("created identity should carry owned agents: %#v", created.OwnedAgentIDs)
	}
	byActor, ok, err := repo.GetAdminIdentityByActor(ctx, actor)
	if err != nil || !ok || len(byActor.OwnedAgentIDs) != 2 {
		t.Fatalf("owned agents should roundtrip, ok=%v identity=%#v err=%v", ok, byActor, err)
	}

	updatedAt := now.Add(time.Minute)
	updated, ok, err := repo.UpdateAdminIdentityOwnedAgentsWithAudit(ctx, identityID, []string{"agt_owned_c"}, updatedAt, "platform", func(updated domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: security.NewID("aud"), Actor: "platform", Action: "admin_identity.updated", ResourceType: "admin_identity", ResourceID: updated.ID, CreatedAt: updatedAt}
	})
	if err != nil || !ok {
		t.Fatalf("update owned agents: ok=%v identity=%#v err=%v", ok, updated, err)
	}
	if len(updated.OwnedAgentIDs) != 1 || updated.OwnedAgentIDs[0] != "agt_owned_c" || !updated.UpdatedAt.Equal(updatedAt) || updated.UpdatedBy != "platform" {
		t.Fatalf("unexpected updated admin identity: %#v", updated)
	}
	reloaded, ok, err := repo.GetAdminIdentity(ctx, identityID)
	if err != nil || !ok || len(reloaded.OwnedAgentIDs) != 1 || reloaded.OwnedAgentIDs[0] != "agt_owned_c" {
		t.Fatalf("update should persist, ok=%v identity=%#v err=%v", ok, reloaded, err)
	}

	clearedAt := now.Add(2 * time.Minute)
	cleared, ok, err := repo.UpdateAdminIdentityOwnedAgentsWithAudit(ctx, identityID, []string{}, clearedAt, "platform", func(updated domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: security.NewID("aud"), Actor: "platform", Action: "admin_identity.updated", ResourceType: "admin_identity", ResourceID: updated.ID, CreatedAt: clearedAt}
	})
	if err != nil || !ok || len(cleared.OwnedAgentIDs) != 0 {
		t.Fatalf("clearing should persist an empty set, ok=%v identity=%#v err=%v", ok, cleared, err)
	}

	missing, ok, err := repo.UpdateAdminIdentityOwnedAgentsWithAudit(ctx, "adm_missing", nil, clearedAt, "platform", func(updated domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: security.NewID("aud"), Action: "admin_identity.updated", ResourceType: "admin_identity", ResourceID: updated.ID, CreatedAt: clearedAt}
	})
	if err != nil || ok || missing.ID != "" {
		t.Fatalf("updating a missing identity should be a silent miss, ok=%v identity=%#v err=%v", ok, missing, err)
	}
}
