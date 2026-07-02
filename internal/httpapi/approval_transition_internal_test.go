package httpapi

import (
	"strings"
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

func TestResolvePermissionPackageApprovalRequestRecordRejectsStaleTransition(t *testing.T) {
	repo := store.NewMemory()
	server := New(repo, WithUnauthenticatedAdminAllowed(true))
	ctx := t.Context()
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	approval := domain.PermissionPackageApprovalRequest{
		ID:                   "ppar_stale_resolution",
		DraftID:              "ppd_stale_resolution",
		TemplateID:           "support-ticket-triage",
		TemplateVersion:      1,
		PolicyVersion:        1,
		TenantID:             "tenant-east",
		WorkspaceID:          "ws-support",
		TargetID:             "agt_mcp",
		CallerInstanceID:     "agt_caller",
		SubjectSelector:      "user:support-*",
		RequestText:          "grant support access",
		Region:               "us-east",
		AllowedCapabilityIDs: []string{"cap_update"},
		AllowedCapabilityKeys: []string{
			"update_ticket",
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "requester",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   now.Add(24 * time.Hour),
	}
	if _, err := repo.CreatePermissionPackageApprovalRequest(ctx, approval); err != nil {
		t.Fatalf("create approval request: %v", err)
	}

	approved := approval
	approved.Status = domain.PermissionPackageApprovalStatusApproved
	approved.ReviewedBy = "security-one"
	approved.UpdatedAt = now.Add(time.Minute)
	approved.ResolvedAt = now.Add(time.Minute)
	if _, ok, err := repo.UpdatePermissionPackageApprovalRequest(ctx, approved); err != nil || !ok {
		t.Fatalf("simulate first reviewer approval: ok=%v err=%v", ok, err)
	}

	_, err := server.resolvePermissionPackageApprovalRequestRecord(ctx, approval, domain.PermissionPackageApprovalStatusRejected, "security-two", "too late", now.Add(2*time.Minute))
	if err == nil || !strings.Contains(err.Error(), "already resolved") {
		t.Fatalf("expected stale resolution rejection, got %v", err)
	}
	loaded, ok, err := repo.GetPermissionPackageApprovalRequest(ctx, approval.ID)
	if err != nil || !ok {
		t.Fatalf("get approval request: ok=%v err=%v", ok, err)
	}
	if loaded.Status != domain.PermissionPackageApprovalStatusApproved || loaded.ReviewedBy != "security-one" {
		t.Fatalf("stale resolution overwrote first reviewer state: %#v", loaded)
	}
}
