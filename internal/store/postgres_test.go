package store_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
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

	repo := store.NewPostgresWithCredentialKey(pool, []byte("0123456789abcdef0123456789abcdef"))
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
		ID:          security.NewID("agt"),
		TenantID:    "test",
		WorkspaceID: "ws-pg",
		Name:        "PG Target",
		ChannelType: "mcp",
		ChannelConfig: map[string]any{
			"endpoint": "https://api.example.com/mcp",
			"credentialHeaders": map[string]any{
				"Authorization": "apiToken",
			},
		},
		Credentials:       map[string]string{"apiToken": "Bearer pg-secret"},
		CredentialVersion: 1,
		Status:            domain.AgentStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if _, err := repo.CreateAgent(ctx, caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	if _, err := repo.CreateAgent(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	crossScopeTarget := domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      "other",
		WorkspaceID:   "ws-other",
		Name:          "PG Cross Scope Target",
		ChannelType:   "mcp",
		ChannelConfig: map[string]any{},
		Status:        domain.AgentStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := repo.CreateAgent(ctx, crossScopeTarget); err != nil {
		t.Fatalf("create cross-scope target: %v", err)
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
	persistedTarget, ok, err := repo.GetAgent(ctx, target.ID)
	if err != nil {
		t.Fatalf("get target: %v", err)
	}
	if !ok || persistedTarget.Credentials["apiToken"] != "Bearer pg-secret" || persistedTarget.CredentialVersion != 1 {
		t.Fatalf("expected credential round trip, ok=%v agent=%#v", ok, persistedTarget)
	}
	var credentialCiphertext []byte
	if err := pool.QueryRow(ctx, "select credentials_ciphertext from agents where id=$1", target.ID).Scan(&credentialCiphertext); err != nil {
		t.Fatalf("read credential ciphertext: %v", err)
	}
	if len(credentialCiphertext) == 0 || bytes.Contains(credentialCiphertext, []byte("pg-secret")) {
		t.Fatalf("expected encrypted credential ciphertext, got %x", credentialCiphertext)
	}
	target.Description = "updated before rotation"
	target.UpdatedAt = now.Add(time.Minute)
	updatedTarget, ok, err := repo.UpdateAgent(ctx, target)
	if err != nil {
		t.Fatalf("update target: %v", err)
	}
	if !ok || updatedTarget.Description != "updated before rotation" || updatedTarget.Credentials["apiToken"] != "Bearer pg-secret" || updatedTarget.CredentialVersion != 1 {
		t.Fatalf("expected updated target without credential version change, ok=%v agent=%#v", ok, updatedTarget)
	}
	rotatedTarget, ok, err := repo.RotateAgentCredentials(ctx, target.ID, map[string]string{"apiToken": "Bearer rotated-secret"}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("rotate target credentials: %v", err)
	}
	if !ok || rotatedTarget.Credentials["apiToken"] != "Bearer rotated-secret" || rotatedTarget.CredentialVersion != 2 {
		t.Fatalf("expected rotated target with version increment, ok=%v agent=%#v", ok, rotatedTarget)
	}
	if err := pool.QueryRow(ctx, "select credentials_ciphertext from agents where id=$1", target.ID).Scan(&credentialCiphertext); err != nil {
		t.Fatalf("read rotated credential ciphertext: %v", err)
	}
	if len(credentialCiphertext) == 0 || bytes.Contains(credentialCiphertext, []byte("rotated-secret")) {
		t.Fatalf("expected rotated credential ciphertext, got %x", credentialCiphertext)
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
	if repo.HasGrant(ctx, caller.ID, target.ID, "mcp", "TOOLS/CALL", now) {
		t.Fatalf("route key matching should be case-sensitive")
	}
	grants, err := repo.ListAccessGrants(ctx, store.ManagementScope{TenantID: "test", WorkspaceID: "ws-pg"})
	if err != nil {
		t.Fatalf("list grants: %v", err)
	}
	if len(grants) == 0 {
		t.Fatalf("expected grant rows")
	}
	denyPolicy := domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    caller.TenantID,
		WorkspaceID: caller.WorkspaceID,
		Name:        "PG deny call",
		CallerID:    caller.ID,
		TargetID:    target.ID,
		RouteType:   "mcp",
		RouteKey:    "tools/call",
		Effect:      domain.RoutePolicyEffectDeny,
		Status:      domain.RoutePolicyStatusEnabled,
		Priority:    100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := repo.CreateRoutePolicy(ctx, denyPolicy); err != nil {
		t.Fatalf("create route policy: %v", err)
	}
	decision, err := repo.EvaluateRouteAccess(ctx, caller.ID, target.ID, "mcp", "tools/call", now)
	if err != nil {
		t.Fatalf("evaluate denied route policy: %v", err)
	}
	if decision.Allowed || decision.Source != "route_policy" || decision.PolicyID != denyPolicy.ID {
		t.Fatalf("expected deny route policy to override grant, got %#v", decision)
	}
	policies, err := repo.ListRoutePolicies(ctx, store.ManagementScope{TenantID: "test", WorkspaceID: "ws-pg"})
	if err != nil {
		t.Fatalf("list route policies: %v", err)
	}
	if len(policies) != 1 || policies[0].ID != denyPolicy.ID {
		t.Fatalf("unexpected route policies: %#v", policies)
	}
	crossScopePolicy := domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    caller.TenantID,
		WorkspaceID: caller.WorkspaceID,
		Name:        "PG cross scope allow",
		CallerID:    caller.ID,
		TargetID:    crossScopeTarget.ID,
		RouteType:   "mcp",
		RouteKey:    "tools/call",
		Effect:      domain.RoutePolicyEffectAllow,
		Status:      domain.RoutePolicyStatusEnabled,
		Priority:    500,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := repo.CreateRoutePolicy(ctx, crossScopePolicy); err != nil {
		t.Fatalf("create cross-scope route policy: %v", err)
	}
	decision, err = repo.EvaluateRouteAccess(ctx, caller.ID, crossScopeTarget.ID, "mcp", "tools/call", now)
	if err != nil {
		t.Fatalf("evaluate cross-scope route policy: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("cross-scope route policy should be ignored, got %#v", decision)
	}
	if _, ok, err := repo.DisableRoutePolicy(ctx, denyPolicy.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("disable route policy: %v", err)
	} else if !ok {
		t.Fatalf("expected route policy to disable")
	}
	decision, err = repo.EvaluateRouteAccess(ctx, caller.ID, target.ID, "mcp", "tools/call", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("evaluate route access after disabled policy: %v", err)
	}
	if !decision.Allowed || decision.Source != "access_grant" {
		t.Fatalf("expected disabled route policy to fall back to access grant, got %#v", decision)
	}
	retryPolicy := domain.RoutePolicy{
		ID:          security.NewID("rpl"),
		TenantID:    caller.TenantID,
		WorkspaceID: caller.WorkspaceID,
		Name:        "PG allow call with retry",
		CallerID:    caller.ID,
		TargetID:    target.ID,
		RouteType:   "mcp",
		RouteKey:    "tools/call",
		Effect:      domain.RoutePolicyEffectAllow,
		Status:      domain.RoutePolicyStatusEnabled,
		Priority:    200,
		Retry: &domain.RoutePolicyRetry{
			MaxAttempts: 2,
			BackoffMs:   25,
			StatusCodes: []int{502, 503},
		},
		CreatedAt: now.Add(3 * time.Minute),
		UpdatedAt: now.Add(3 * time.Minute),
	}
	if _, err := repo.CreateRoutePolicy(ctx, retryPolicy); err != nil {
		t.Fatalf("create retry route policy: %v", err)
	}
	decision, err = repo.EvaluateRouteAccess(ctx, caller.ID, target.ID, "mcp", "tools/call", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("evaluate retry route policy: %v", err)
	}
	if !decision.Allowed || decision.PolicyID != retryPolicy.ID || decision.Retry == nil ||
		decision.Retry.MaxAttempts != 2 || decision.Retry.BackoffMs != 25 || len(decision.Retry.StatusCodes) != 2 || decision.Retry.StatusCodes[1] != 503 {
		t.Fatalf("expected retry policy decision, got %#v", decision)
	}

	trace := domain.TraceEvent{
		ID:               security.NewID("trc"),
		RunID:            "pg-run",
		CallerID:         caller.ID,
		TargetID:         target.ID,
		RouteType:        "mcp",
		RouteKey:         "tools/call",
		Decision:         domain.TraceDecisionAllowed,
		Reason:           "integration",
		DurationMs:       42,
		UpstreamAttempts: 2,
		UpstreamStatus:   202,
		UpstreamError:    "UPSTREAM_DNS_ERROR",
		CreatedAt:        now,
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
	if traces[0].DurationMs != trace.DurationMs ||
		traces[0].UpstreamAttempts != trace.UpstreamAttempts ||
		traces[0].UpstreamStatus != trace.UpstreamStatus ||
		traces[0].UpstreamError != trace.UpstreamError {
		t.Fatalf("unexpected trace metrics: %#v", traces[0])
	}
	audit := domain.AuditEvent{
		ID:           security.NewID("aud"),
		TenantID:     "test",
		WorkspaceID:  "ws-pg",
		Actor:        "integration-test",
		Action:       "agent.credentials_rotated",
		ResourceType: "agent",
		ResourceID:   target.ID,
		Summary:      "PG credential rotation",
		Metadata: map[string]any{
			"credentialVersion": 2,
			"credentialKeys":    []any{"apiToken"},
		},
		CreatedAt: now.Add(2 * time.Minute),
	}
	if _, err := repo.AppendAuditEvent(ctx, audit); err != nil {
		t.Fatalf("append audit event: %v", err)
	}
	audits, err := repo.ListAuditEvents(ctx, store.AuditEventFilter{
		ManagementScope: store.ManagementScope{TenantID: "test", WorkspaceID: "ws-pg"},
		Action:          "agent.credentials_rotated",
		ResourceType:    "agent",
		ResourceID:      target.ID,
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(audits) != 1 || audits[0].ID != audit.ID || audits[0].Metadata["credentialVersion"] != float64(2) {
		t.Fatalf("unexpected audit events: %#v", audits)
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

func TestPostgresCapabilityGovernanceRoundTrip(t *testing.T) {
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
	caller := domain.Agent{
		ID:          security.NewID("agt"),
		TenantID:    "tenant-pg-cap",
		WorkspaceID: "ws-pg-cap",
		Name:        "PG Capability Caller",
		ChannelType: "local",
		Status:      domain.AgentStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	target := domain.Agent{
		ID:          security.NewID("agt"),
		TenantID:    "tenant-pg-cap",
		WorkspaceID: "ws-pg-cap",
		Name:        "PG Capability Target",
		ChannelType: "mcp",
		ChannelConfig: map[string]any{
			"endpoint": "https://api.example.com/mcp",
		},
		Status:    domain.AgentStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := repo.CreateAgent(ctx, caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	if _, err := repo.CreateAgent(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	capability := domain.Capability{
		ID:           security.NewID("cap"),
		TargetID:     target.ID,
		Type:         domain.CapabilityTypeMCPTool,
		Key:          "search_customer",
		DisplayName:  "search_customer",
		Action:       domain.CapabilityActionRead,
		InputSchema:  map[string]any{"type": "object"},
		OutputSchema: map[string]any{},
		DataScopes: []domain.DataScope{{
			DataDomain:   "crm",
			Dataset:      "customers",
			TenantFilter: "tenant_id = 'tenant-pg-cap'",
		}},
		Sensitivity:     domain.CapabilitySensitivityInternal,
		RiskLevel:       domain.CapabilityRiskLow,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(ctx, capability); err != nil {
		t.Fatalf("upsert capability: %v", err)
	}
	caps, err := repo.ListCapabilities(ctx, store.CapabilityFilter{TargetID: target.ID})
	if err != nil {
		t.Fatalf("list capabilities: %v", err)
	}
	if len(caps) != 1 || caps[0].ID != capability.ID {
		t.Fatalf("unexpected capabilities: %#v", caps)
	}
	entitlement, err := repo.CreateTenantEntitlement(ctx, domain.TenantEntitlement{
		ID:           security.NewID("ent"),
		TenantID:     caller.TenantID,
		TargetID:     target.ID,
		CapabilityID: capability.ID,
		Effect:       domain.PolicyEffectAllow,
		Status:       domain.PolicyStatusEnabled,
		Priority:     100,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	workspaceAssignment, err := repo.CreateWorkspaceAssignment(ctx, domain.WorkspaceAssignment{
		ID:                  security.NewID("wsa"),
		TenantEntitlementID: entitlement.ID,
		TenantID:            caller.TenantID,
		WorkspaceID:         caller.WorkspaceID,
		Effect:              domain.PolicyEffectAllow,
		DataScopes: []domain.DataScope{{
			Region: "us-east",
		}},
		Status:    domain.PolicyStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create workspace assignment: %v", err)
	}
	instanceAssignment, err := repo.CreateInstanceAssignment(ctx, domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:*",
		Effect:                domain.PolicyEffectAllow,
		DataScopes: []domain.DataScope{{
			Table: "accounts",
		}},
		Status:    domain.PolicyStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create instance assignment: %v", err)
	}
	application, err := repo.CreatePermissionPackageApplication(ctx, domain.PermissionPackageApplication{
		ID:                     security.NewID("ppa"),
		DraftID:                "ppd_pg_cap",
		TemplateID:             "sales-readonly",
		TemplateVersion:        1,
		TenantID:               caller.TenantID,
		WorkspaceID:            caller.WorkspaceID,
		TargetID:               target.ID,
		CallerInstanceID:       caller.ID,
		SubjectSelector:        "user:*",
		RequestText:            "grant sales read access",
		Region:                 "us-east",
		DataScopes:             []domain.DataScope{{DataDomain: "crm", Region: "us-east", TenantFilter: "tenant_id = 'tenant-pg-cap'"}},
		AllowedCapabilityIDs:   []string{capability.ID},
		AllowedCapabilityKeys:  []string{capability.Key},
		TenantEntitlementIDs:   []string{entitlement.ID},
		WorkspaceAssignmentIDs: []string{workspaceAssignment.ID},
		InstanceAssignmentIDs:  []string{instanceAssignment.ID},
		AppliedAt:              now.Add(30 * time.Second),
	})
	if err != nil {
		t.Fatalf("create permission package application: %v", err)
	}
	applications, err := repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
		ManagementScope:  store.ManagementScope{TenantID: caller.TenantID, WorkspaceID: caller.WorkspaceID},
		TemplateID:       "sales-readonly",
		TargetID:         target.ID,
		CallerInstanceID: caller.ID,
		Limit:            1,
	})
	if err != nil {
		t.Fatalf("list permission package applications: %v", err)
	}
	if len(applications) != 1 || applications[0].ID != application.ID || applications[0].TemplateVersion != 1 ||
		len(applications[0].AllowedCapabilityIDs) != 1 || applications[0].AllowedCapabilityIDs[0] != capability.ID ||
		len(applications[0].DataScopes) != 1 || applications[0].DataScopes[0].Region != "us-east" {
		t.Fatalf("unexpected permission package applications: %#v", applications)
	}
	approval, err := repo.CreatePermissionPackageApprovalRequest(ctx, domain.PermissionPackageApprovalRequest{
		ID:                    security.NewID("ppar"),
		DraftID:               "ppd_pg_cap_pending",
		TemplateID:            "sales-readonly",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		TargetID:              target.ID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:*",
		RequestText:           "grant sales read access",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "crm", Region: "us-east", TenantFilter: "tenant_id = 'tenant-pg-cap'"}},
		AllowedCapabilityIDs:  []string{capability.ID},
		AllowedCapabilityKeys: []string{capability.Key},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
			Reasons: []domain.PermissionPackagePolicyReason{{
				ID:            "reason_high",
				CapabilityID:  capability.ID,
				CapabilityKey: capability.Key,
				Severity:      "high",
				Message:       "approval required",
				ReasonKey:     "risk",
				ReasonValues:  map[string]string{"risk": "high"},
			}},
			NextActions: []string{"request approval"},
		},
		Status:      domain.PermissionPackageApprovalStatusPending,
		RequestedBy: "admin",
		CreatedAt:   now.Add(40 * time.Second),
		UpdatedAt:   now.Add(40 * time.Second),
	})
	if err != nil {
		t.Fatalf("create permission package approval request: %v", err)
	}
	approvalRows, err := repo.ListPermissionPackageApprovalRequests(ctx, store.PermissionPackageApprovalRequestFilter{
		ManagementScope:  store.ManagementScope{TenantID: caller.TenantID, WorkspaceID: caller.WorkspaceID},
		TemplateID:       "sales-readonly",
		TargetID:         target.ID,
		CallerInstanceID: caller.ID,
		Status:           domain.PermissionPackageApprovalStatusPending,
		Limit:            1,
	})
	if err != nil {
		t.Fatalf("list permission package approval requests: %v", err)
	}
	if len(approvalRows) != 1 || approvalRows[0].ID != approval.ID ||
		len(approvalRows[0].PolicyGate.Reasons) != 1 || approvalRows[0].PolicyGate.Reasons[0].ReasonValues["risk"] != "high" ||
		len(approvalRows[0].DataScopes) != 1 || approvalRows[0].DataScopes[0].Region != "us-east" {
		t.Fatalf("unexpected permission package approval requests: %#v", approvalRows)
	}
	loadedApproval, ok, err := repo.GetPermissionPackageApprovalRequest(ctx, approval.ID)
	if err != nil {
		t.Fatalf("get permission package approval request: %v", err)
	}
	if !ok || loadedApproval.ID != approval.ID || loadedApproval.Status != domain.PermissionPackageApprovalStatusPending {
		t.Fatalf("unexpected permission package approval request get: ok=%v row=%#v", ok, loadedApproval)
	}
	loadedApproval.Status = domain.PermissionPackageApprovalStatusApproved
	loadedApproval.ReviewedBy = "security"
	loadedApproval.ReviewComment = "approved for pg test"
	loadedApproval.UpdatedAt = now.Add(50 * time.Second)
	loadedApproval.ResolvedAt = now.Add(50 * time.Second)
	updatedApproval, ok, err := repo.UpdatePermissionPackageApprovalRequest(ctx, loadedApproval)
	if err != nil {
		t.Fatalf("update permission package approval request: %v", err)
	}
	if !ok || updatedApproval.Status != domain.PermissionPackageApprovalStatusApproved ||
		updatedApproval.ReviewedBy != "security" || updatedApproval.ResolvedAt.IsZero() {
		t.Fatalf("unexpected updated permission package approval request: ok=%v row=%#v", ok, updatedApproval)
	}
	decision, err := repo.EvaluateCapabilityAccess(ctx, store.CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		SubjectID:        "user:pg",
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate capability access: %v", err)
	}
	if !decision.Allowed || decision.EntitlementID != entitlement.ID || decision.WorkspaceAssignmentID != workspaceAssignment.ID || decision.InstanceAssignmentID != instanceAssignment.ID {
		t.Fatalf("unexpected decision: %#v", decision)
	}
	wantScopes := []domain.DataScope{{
		DataDomain:   "crm",
		Dataset:      "customers",
		Table:        "accounts",
		Region:       "us-east",
		TenantFilter: "tenant_id = 'tenant-pg-cap'",
	}}
	if !reflect.DeepEqual(decision.DataScopes, wantScopes) {
		t.Fatalf("data scopes = %#v, want %#v", decision.DataScopes, wantScopes)
	}
	withoutSubject, err := repo.EvaluateCapabilityAccess(ctx, store.CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate without subject: %v", err)
	}
	if withoutSubject.Allowed {
		t.Fatalf("subject-scoped assignment should deny missing subject: %#v", withoutSubject)
	}
	denyAssignment, err := repo.CreateInstanceAssignment(ctx, domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:pg",
		Effect:                domain.PolicyEffectDeny,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now.Add(time.Minute),
		UpdatedAt:             now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("create deny instance assignment: %v", err)
	}
	deniedBySubject, err := repo.EvaluateCapabilityAccess(ctx, store.CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		SubjectID:        "user:pg",
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate subject deny: %v", err)
	}
	if deniedBySubject.Allowed || deniedBySubject.InstanceAssignmentID != denyAssignment.ID {
		t.Fatalf("exact deny assignment should take precedence over wildcard allow: %#v", deniedBySubject)
	}
}

func TestPostgresTenantHierarchyRoundTrip(t *testing.T) {
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
	rootID := security.NewID("tenant")
	childID := security.NewID("tenant")
	grandchildID := security.NewID("tenant")
	unrelatedID := security.NewID("tenant")

	for _, tenant := range []domain.Tenant{
		{ID: rootID, Name: "PG Root", Level: 1, Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: childID, ParentTenantID: rootID, Name: "PG Child", Level: 2, Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: grandchildID, ParentTenantID: childID, Name: "PG Grandchild", Level: 3, Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: unrelatedID, Name: "PG Unrelated", Level: 1, Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
	} {
		if _, err := repo.CreateTenant(ctx, tenant); err != nil {
			t.Fatalf("create tenant %s: %v", tenant.ID, err)
		}
	}
	for _, agent := range []domain.Agent{
		{ID: security.NewID("agt"), TenantID: rootID, WorkspaceID: "ws-pg-tree", Name: "Root Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: security.NewID("agt"), TenantID: childID, WorkspaceID: "ws-pg-tree", Name: "Child Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: security.NewID("agt"), TenantID: grandchildID, WorkspaceID: "ws-pg-tree", Name: "Grandchild Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: security.NewID("agt"), TenantID: unrelatedID, WorkspaceID: "ws-pg-tree", Name: "Unrelated Agent", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now},
	} {
		if _, err := repo.CreateAgent(ctx, agent); err != nil {
			t.Fatalf("create agent %s: %v", agent.ID, err)
		}
	}

	tenants, err := repo.ListTenants(ctx, store.TenantFilter{TenantID: rootID})
	if err != nil {
		t.Fatalf("list tenants: %v", err)
	}
	if got := len(tenants); got != 3 {
		t.Fatalf("tenant subtree length = %d, want 3; rows=%#v", got, tenants)
	}
	child, ok, err := repo.GetTenant(ctx, childID)
	if err != nil {
		t.Fatalf("get child tenant: %v", err)
	}
	if !ok || child.ParentTenantID != rootID || child.Level != 2 {
		t.Fatalf("unexpected child tenant: ok=%v row=%#v", ok, child)
	}
	agents, err := repo.ListAgents(ctx, store.AgentFilter{ManagementScope: store.ManagementScope{TenantID: rootID, WorkspaceID: "ws-pg-tree"}})
	if err != nil {
		t.Fatalf("list agents by tenant subtree: %v", err)
	}
	if got := len(agents); got != 3 {
		t.Fatalf("subtree agent length = %d, want 3; rows=%#v", got, agents)
	}
}

func TestPostgresAuditedCreateAgentRollsBackWhenAuditFails(t *testing.T) {
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
	agent := domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      "test",
		WorkspaceID:   "ws-pg-rollback",
		Name:          "Rollback Agent",
		ChannelType:   "local",
		ChannelConfig: map[string]any{},
		Status:        domain.AgentStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	audit := domain.AuditEvent{
		ID:           security.NewID("aud"),
		TenantID:     agent.TenantID,
		WorkspaceID:  agent.WorkspaceID,
		Actor:        "test",
		Action:       "agent.created",
		ResourceType: "agent",
		ResourceID:   agent.ID,
		Summary:      "Agent created",
		Metadata:     map[string]any{"source": "rollback-test"},
		CreatedAt:    now,
	}
	if _, err := pool.Exec(ctx, `
		insert into audit_events (
			id, tenant_id, workspace_id, actor, action, resource_type, resource_id, summary, metadata, created_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, audit.ID, audit.TenantID, audit.WorkspaceID, audit.Actor, audit.Action, audit.ResourceType,
		audit.ResourceID, audit.Summary, []byte(`{"source":"rollback-test"}`), audit.CreatedAt); err != nil {
		t.Fatalf("seed duplicate audit event: %v", err)
	}

	if _, err := repo.CreateAgentWithAudit(ctx, agent, func(domain.Agent) domain.AuditEvent {
		return audit
	}); err == nil {
		t.Fatalf("CreateAgentWithAudit should fail when audit insert conflicts")
	}
	if _, ok, err := repo.GetAgent(ctx, agent.ID); err != nil {
		t.Fatalf("get rollback agent: %v", err)
	} else if ok {
		t.Fatalf("agent persisted even though audit insert failed")
	}
	var auditCount int
	if err := pool.QueryRow(ctx, "select count(*) from audit_events where id=$1", audit.ID).Scan(&auditCount); err != nil {
		t.Fatalf("count rollback audit event: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected only seeded audit event after rollback, count=%d", auditCount)
	}
}
