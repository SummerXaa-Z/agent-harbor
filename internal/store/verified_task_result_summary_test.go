package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SummerXaa-Z/agent-harbor/internal/db"
	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestMemoryVerifiedTaskResultSummaryUsesExactScopeAndUniqueSource(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemory()
	now := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	source := verifiedTaskResultSourceTrace(now)
	if _, err := repo.AppendTrace(ctx, source); err != nil {
		t.Fatalf("append source trace: %v", err)
	}
	summary := verifiedTaskResultSummaryFixture(source, now)
	created, err := repo.CreateVerifiedTaskResultSummaryWithAudit(ctx, summary, verifiedTaskResultAudit)
	if err != nil {
		t.Fatalf("create verified task result summary: %v", err)
	}
	loaded, found, err := repo.GetVerifiedTaskResultSummary(ctx, created.ID, verifiedTaskResultSummaryScope(summary))
	if err != nil || !found {
		t.Fatalf("get verified task result summary: found=%v err=%v", found, err)
	}
	if loaded.SourceTraceID != source.ID || loaded.Summary != summary.Summary || loaded.PayloadDigest != summary.PayloadDigest {
		t.Fatalf("summary round trip mismatch: %#v", loaded)
	}
	loaded.DataScopes[0].TenantFilter = "mutated"
	reloaded, found, err := repo.GetVerifiedTaskResultSummary(ctx, created.ID, verifiedTaskResultSummaryScope(summary))
	if err != nil || !found || reloaded.DataScopes[0].TenantFilter != "country=SG" {
		t.Fatalf("get must return a detached data-scope copy: found=%v summary=%#v err=%v", found, reloaded, err)
	}

	matching, err := repo.ListVerifiedTaskResultSummaries(ctx, store.VerifiedTaskResultSummaryFilter{VerifiedTaskResultSummaryScope: verifiedTaskResultSummaryScope(summary)})
	if err != nil || len(matching) != 1 || matching[0].ID != summary.ID {
		t.Fatalf("exact scope should return only the matching summary: rows=%#v err=%v", matching, err)
	}
	mismatched, err := repo.ListVerifiedTaskResultSummaries(ctx, store.VerifiedTaskResultSummaryFilter{VerifiedTaskResultSummaryScope: store.VerifiedTaskResultSummaryScope{
		TenantID:         "malaysia",
		WorkspaceID:      summary.WorkspaceID,
		CallerInstanceID: summary.CallerInstanceID,
		SubjectID:        summary.SubjectID,
	}})
	if err != nil || len(mismatched) != 0 {
		t.Fatalf("tenant mismatch must not return a summary: rows=%#v err=%v", mismatched, err)
	}
	if _, _, err := repo.GetVerifiedTaskResultSummary(ctx, created.ID, store.VerifiedTaskResultSummaryScope{}); !errors.Is(err, store.ErrVerifiedTaskResultSummaryScopeRequired) {
		t.Fatalf("unscoped get must be rejected, got %v", err)
	}
	if _, err := repo.ListVerifiedTaskResultSummaries(ctx, store.VerifiedTaskResultSummaryFilter{}); !errors.Is(err, store.ErrVerifiedTaskResultSummaryScopeRequired) {
		t.Fatalf("unscoped list must be rejected, got %v", err)
	}
	bySource, found, err := repo.FindVerifiedTaskResultSummaryBySource(ctx, source.ID)
	if err != nil || !found || bySource.ID != created.ID {
		t.Fatalf("find by source mismatch: found=%v summary=%#v err=%v", found, bySource, err)
	}

	duplicate := summary
	duplicate.ID = security.NewID("mem")
	if _, err := repo.CreateVerifiedTaskResultSummaryWithAudit(ctx, duplicate, verifiedTaskResultAudit); !errors.Is(err, store.ErrVerifiedTaskResultSummarySourceAlreadyUsed) {
		t.Fatalf("duplicate source trace should be rejected, got %v", err)
	}
	events, err := repo.ListAuditEvents(ctx, store.AuditEventFilter{Action: "memory_written", ResourceType: "verified_task_result_summary"})
	if err != nil || len(events) != 1 || events[0].ResourceID != summary.ID {
		t.Fatalf("summary write must have exactly one audit event: events=%#v err=%v", events, err)
	}
}

func TestPostgresVerifiedTaskResultSummaryUsesExactScopeAndUniqueSource(t *testing.T) {
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
	repo := store.NewPostgresWithCredentialKey(pool, []byte("0123456789abcdef0123456789abcdef"))
	now := time.Now().UTC().Truncate(time.Microsecond)
	source := verifiedTaskResultSourceTrace(now)
	if _, err := repo.AppendTrace(ctx, source); err != nil {
		t.Fatalf("append source trace: %v", err)
	}
	summary := verifiedTaskResultSummaryFixture(source, now)
	created, err := repo.CreateVerifiedTaskResultSummaryWithAudit(ctx, summary, verifiedTaskResultAudit)
	if err != nil {
		t.Fatalf("create verified task result summary: %v", err)
	}
	loaded, found, err := repo.GetVerifiedTaskResultSummary(ctx, created.ID, verifiedTaskResultSummaryScope(summary))
	if err != nil || !found {
		t.Fatalf("get verified task result summary: found=%v err=%v", found, err)
	}
	if loaded.SourceTraceID != source.ID || loaded.Summary != summary.Summary || loaded.PayloadDigest != summary.PayloadDigest || len(loaded.DataScopes) != 1 {
		t.Fatalf("summary round trip mismatch: %#v", loaded)
	}
	loadedTrace, found, err := repo.GetTrace(ctx, source.ID)
	if err != nil || !found || loadedTrace.ID != source.ID || loadedTrace.TenantID != source.TenantID || loadedTrace.CapabilityFingerprint != source.CapabilityFingerprint {
		t.Fatalf("get trace round trip mismatch: found=%v trace=%#v err=%v", found, loadedTrace, err)
	}

	matching, err := repo.ListVerifiedTaskResultSummaries(ctx, store.VerifiedTaskResultSummaryFilter{VerifiedTaskResultSummaryScope: verifiedTaskResultSummaryScope(summary)})
	if err != nil || len(matching) != 1 || matching[0].ID != summary.ID {
		t.Fatalf("exact scope should return the matching summary: rows=%#v err=%v", matching, err)
	}
	mismatched, err := repo.ListVerifiedTaskResultSummaries(ctx, store.VerifiedTaskResultSummaryFilter{VerifiedTaskResultSummaryScope: store.VerifiedTaskResultSummaryScope{
		TenantID:         "malaysia",
		WorkspaceID:      summary.WorkspaceID,
		CallerInstanceID: summary.CallerInstanceID,
		SubjectID:        summary.SubjectID,
	}})
	if err != nil || len(mismatched) != 0 {
		t.Fatalf("tenant mismatch must not return a summary: rows=%#v err=%v", mismatched, err)
	}
	if _, _, err := repo.GetVerifiedTaskResultSummary(ctx, created.ID, store.VerifiedTaskResultSummaryScope{}); !errors.Is(err, store.ErrVerifiedTaskResultSummaryScopeRequired) {
		t.Fatalf("unscoped get must be rejected, got %v", err)
	}
	if _, err := repo.ListVerifiedTaskResultSummaries(ctx, store.VerifiedTaskResultSummaryFilter{}); !errors.Is(err, store.ErrVerifiedTaskResultSummaryScopeRequired) {
		t.Fatalf("unscoped list must be rejected, got %v", err)
	}
	bySource, found, err := repo.FindVerifiedTaskResultSummaryBySource(ctx, source.ID)
	if err != nil || !found || bySource.ID != created.ID {
		t.Fatalf("find by source mismatch: found=%v summary=%#v err=%v", found, bySource, err)
	}

	duplicate := summary
	duplicate.ID = security.NewID("mem")
	if _, err := repo.CreateVerifiedTaskResultSummaryWithAudit(ctx, duplicate, verifiedTaskResultAudit); !errors.Is(err, store.ErrVerifiedTaskResultSummarySourceAlreadyUsed) {
		t.Fatalf("duplicate source trace should be rejected, got %v", err)
	}
	events, err := repo.ListAuditEvents(ctx, store.AuditEventFilter{Action: "memory_written", ResourceType: "verified_task_result_summary"})
	if err != nil || len(events) != 1 || events[0].ResourceID != summary.ID {
		t.Fatalf("summary write must have exactly one audit event: events=%#v err=%v", events, err)
	}
}

func verifiedTaskResultSourceTrace(now time.Time) domain.TraceEvent {
	return domain.TraceEvent{
		ID:                    security.NewID("trc"),
		RunID:                 "verified-task-result-store-test",
		CallerID:              "caller-sg",
		TargetID:              "target-sg",
		RouteType:             "mcp",
		RouteKey:              "tools/call",
		TenantID:              "singapore",
		WorkspaceID:           "support-sg",
		CallerInstanceID:      "caller-sg",
		SubjectID:             "user:support-001",
		CapabilityID:          "cap-ticket-read",
		CapabilityVersion:     1,
		CapabilityFingerprint: "cap-ticket-read:store-test",
		EntitlementID:         "ent-ticket-read",
		WorkspaceAssignmentID: "wsa-ticket-read",
		InstanceAssignmentID:  "ina-ticket-read",
		DataScopes:            []domain.DataScope{{DataDomain: "support", Dataset: "tickets", TenantFilter: "country=SG"}},
		Decision:              domain.TraceDecisionAllowed,
		UpstreamAttempts:      1,
		UpstreamStatus:        200,
		CreatedAt:             now,
	}
}

func verifiedTaskResultSummaryFixture(source domain.TraceEvent, now time.Time) domain.VerifiedTaskResultSummary {
	summary := "Synthetic ticket result was verified and redacted."
	return domain.VerifiedTaskResultSummary{
		ID:               security.NewID("mem"),
		TenantID:         source.TenantID,
		WorkspaceID:      source.WorkspaceID,
		CallerInstanceID: source.CallerInstanceID,
		SubjectID:        source.SubjectID,
		TargetID:         source.TargetID,
		CapabilityID:     source.CapabilityID,
		SourceTraceID:    source.ID,
		DataScopes:       domain.CloneDataScopes(source.DataScopes),
		Summary:          summary,
		PayloadDigest:    verifiedTaskResultDigest(summary),
		Verification:     domain.TaskResultSummaryVerificationHumanReviewedRedacted,
		VerifiedBy:       "store-test",
		VerifiedAt:       now,
		CreatedAt:        now,
		ExpiresAt:        now.Add(24 * time.Hour),
	}
}

func verifiedTaskResultSummaryScope(summary domain.VerifiedTaskResultSummary) store.VerifiedTaskResultSummaryScope {
	return store.VerifiedTaskResultSummaryScope{
		TenantID:         summary.TenantID,
		WorkspaceID:      summary.WorkspaceID,
		CallerInstanceID: summary.CallerInstanceID,
		SubjectID:        summary.SubjectID,
	}
}

func verifiedTaskResultAudit(summary domain.VerifiedTaskResultSummary) domain.AuditEvent {
	return domain.AuditEvent{
		ID:           security.NewID("aud"),
		TenantID:     summary.TenantID,
		WorkspaceID:  summary.WorkspaceID,
		Actor:        "store-test",
		Action:       "memory_written",
		ResourceType: "verified_task_result_summary",
		ResourceID:   summary.ID,
		Summary:      "Verified task result memory written",
		Metadata: map[string]any{
			"sourceTraceId": summary.SourceTraceID,
			"payloadDigest": summary.PayloadDigest,
		},
		CreatedAt: summary.CreatedAt,
	}
}

func verifiedTaskResultDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
