package store_test

import (
	"bytes"
	"context"
	"errors"
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
	policies, err = repo.ListRoutePolicies(ctx, store.ManagementScope{TenantID: caller.TenantID, WorkspaceID: caller.WorkspaceID})
	if err != nil {
		t.Fatalf("list route policies after cross-scope policy: %v", err)
	}
	if len(policies) != 1 || policies[0].ID != denyPolicy.ID {
		t.Fatalf("cross-scope route policy should be hidden from scoped list, got %#v", policies)
	}
	decision, err = repo.EvaluateRouteAccess(ctx, caller.ID, crossScopeTarget.ID, "mcp", "tools/call", now)
	if err != nil {
		t.Fatalf("evaluate cross-scope route policy: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("cross-scope route policy should be ignored, got %#v", decision)
	}
	crossScopeGrant := domain.AccessGrant{
		ID:        security.NewID("grt"),
		CallerID:  caller.ID,
		TargetID:  crossScopeTarget.ID,
		RouteType: "mcp",
		RouteKey:  "tools/call",
		CreatedAt: now,
	}
	if _, err := repo.CreateAccessGrant(ctx, crossScopeGrant); err != nil {
		t.Fatalf("create cross-scope legacy grant: %v", err)
	}
	if repo.HasGrant(ctx, caller.ID, crossScopeTarget.ID, "mcp", "tools/call", now) {
		t.Fatalf("cross-scope legacy grant should not authorize direct grant checks")
	}
	decision, err = repo.EvaluateRouteAccess(ctx, caller.ID, crossScopeTarget.ID, "mcp", "tools/call", now)
	if err != nil {
		t.Fatalf("evaluate cross-scope legacy grant: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("cross-scope legacy grant should not authorize data-plane access, got %#v", decision)
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
	crossScopeTrace := trace
	crossScopeTrace.ID = security.NewID("trc")
	crossScopeTrace.TargetID = crossScopeTarget.ID
	crossScopeTrace.TenantID = caller.TenantID
	crossScopeTrace.WorkspaceID = caller.WorkspaceID
	crossScopeTrace.CreatedAt = now.Add(time.Second)
	if _, err := repo.AppendTrace(ctx, crossScopeTrace); err != nil {
		t.Fatalf("append cross-scope trace: %v", err)
	}
	crossScopeCapability := domain.Capability{
		ID:              security.NewID("cap"),
		TargetID:        crossScopeTarget.ID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             "export_finance",
		DisplayName:     "Export finance",
		Description:     "Export finance",
		Action:          domain.CapabilityActionExport,
		Sensitivity:     domain.CapabilitySensitivityConfidential,
		RiskLevel:       domain.CapabilityRiskHigh,
		EnforcementMode: domain.CapabilityEnforcementGateway,
		DiscoveryStatus: domain.CapabilityDiscoveryApproved,
		Version:         1,
		DiscoveredAt:    now,
		UpdatedAt:       now,
	}
	if _, err := repo.UpsertCapability(ctx, crossScopeCapability); err != nil {
		t.Fatalf("create cross-scope capability: %v", err)
	}
	crossScopeCapabilityTrace := trace
	crossScopeCapabilityTrace.ID = security.NewID("trc")
	crossScopeCapabilityTrace.TargetID = crossScopeTarget.ID
	crossScopeCapabilityTrace.TenantID = caller.TenantID
	crossScopeCapabilityTrace.WorkspaceID = caller.WorkspaceID
	crossScopeCapabilityTrace.CallerInstanceID = caller.ID
	crossScopeCapabilityTrace.CapabilityID = crossScopeCapability.ID
	crossScopeCapabilityTrace.CapabilityVersion = crossScopeCapability.Version
	crossScopeCapabilityTrace.CreatedAt = now.Add(2 * time.Second)
	if _, err := repo.AppendTrace(ctx, crossScopeCapabilityTrace); err != nil {
		t.Fatalf("append cross-scope capability trace: %v", err)
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
	scopedTraces, err := repo.ListTraces(ctx, store.TraceFilter{
		RunID: "pg-run",
		ManagementScope: store.ManagementScope{
			TenantID:    caller.TenantID,
			WorkspaceID: caller.WorkspaceID,
		},
	})
	if err != nil {
		t.Fatalf("list scoped traces: %v", err)
	}
	if len(scopedTraces) != 1 || scopedTraces[0].ID != trace.ID {
		t.Fatalf("scoped traces should require caller and target in scope, got %#v", scopedTraces)
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

func TestPostgresAdminIdentityLifecycle(t *testing.T) {
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
	identityID := security.NewID("adm")
	actor := "pg-admin-" + identityID
	oldHash := security.HashSecret("pg-admin-secret-old-" + identityID)
	newHash := security.HashSecret("pg-admin-secret-new-" + identityID)
	identity := domain.AdminIdentity{
		ID:          identityID,
		Actor:       actor,
		DisplayName: "PG Tenant Admin",
		Role:        domain.AdminIdentityRoleTenantAdmin,
		TenantID:    "tenant-east",
		WorkspaceID: "ws-support",
		Status:      domain.AdminIdentityStatusActive,
		Source:      domain.AdminIdentitySourceManaged,
		KeyHash:     oldHash,
		KeyPrefix:   "ahadm_pg_old",
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "platform",
		UpdatedBy:   "platform",
	}

	created, err := repo.CreateAdminIdentityWithAudit(ctx, identity, func(created domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: security.NewID("aud"), Actor: "platform", Action: "admin_identity.created", ResourceType: "admin_identity", ResourceID: created.ID, CreatedAt: now}
	})
	if err != nil {
		t.Fatalf("create admin identity: %v", err)
	}
	if created.ID != identity.ID || created.Actor != actor || created.KeyHash != oldHash {
		t.Fatalf("unexpected created admin identity: %#v", created)
	}

	rows, err := repo.ListAdminIdentities(ctx)
	if err != nil {
		t.Fatalf("list admin identities: %v", err)
	}
	if !adminIdentityIDsContain(rows, identity.ID) {
		t.Fatalf("created admin identity missing from list: %#v", rows)
	}
	byActor, ok, err := repo.GetAdminIdentityByActor(ctx, actor)
	if err != nil || !ok || byActor.ID != identity.ID {
		t.Fatalf("get admin identity by actor: ok=%v identity=%#v err=%v", ok, byActor, err)
	}
	byHash, ok, err := repo.FindAdminIdentityByKeyHash(ctx, oldHash)
	if err != nil || !ok || byHash.ID != identity.ID {
		t.Fatalf("find admin identity by key hash: ok=%v identity=%#v err=%v", ok, byHash, err)
	}

	rotatedAt := now.Add(time.Minute)
	rotated, ok, err := repo.RotateAdminIdentityKeyWithAudit(ctx, identity.ID, newHash, "ahadm_pg_new", rotatedAt, "platform", func(rotated domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: security.NewID("aud"), Actor: "platform", Action: "admin_identity.key_rotated", ResourceType: "admin_identity", ResourceID: rotated.ID, CreatedAt: rotatedAt}
	})
	if err != nil || !ok {
		t.Fatalf("rotate admin identity key: ok=%v identity=%#v err=%v", ok, rotated, err)
	}
	if rotated.KeyHash != newHash || rotated.KeyPrefix != "ahadm_pg_new" || !rotated.RotatedAt.Equal(rotatedAt) {
		t.Fatalf("unexpected rotated admin identity: %#v", rotated)
	}
	if _, ok, err := repo.FindAdminIdentityByKeyHash(ctx, oldHash); err != nil || ok {
		t.Fatalf("old admin hash should not authenticate after rotation: ok=%v err=%v", ok, err)
	}

	lastUsedAt := now.Add(2 * time.Minute)
	if err := repo.TouchAdminIdentityLastUsed(ctx, identity.ID, lastUsedAt); err != nil {
		t.Fatalf("touch last used: %v", err)
	}
	touched, ok, err := repo.GetAdminIdentity(ctx, identity.ID)
	if err != nil || !ok || !touched.LastUsedAt.Equal(lastUsedAt) {
		t.Fatalf("expected touched last used, ok=%v identity=%#v err=%v", ok, touched, err)
	}

	disabledAt := now.Add(3 * time.Minute)
	disabled, ok, err := repo.DisableAdminIdentityWithAudit(ctx, identity.ID, disabledAt, "platform", func(disabled domain.AdminIdentity) domain.AuditEvent {
		return domain.AuditEvent{ID: security.NewID("aud"), Actor: "platform", Action: "admin_identity.disabled", ResourceType: "admin_identity", ResourceID: disabled.ID, CreatedAt: disabledAt}
	})
	if err != nil || !ok {
		t.Fatalf("disable admin identity: ok=%v identity=%#v err=%v", ok, disabled, err)
	}
	if disabled.Status != domain.AdminIdentityStatusDisabled || !disabled.DisabledAt.Equal(disabledAt) || disabled.DisabledBy != "platform" {
		t.Fatalf("unexpected disabled admin identity: %#v", disabled)
	}
	if _, ok, err := repo.FindAdminIdentityByKeyHash(ctx, newHash); err != nil || ok {
		t.Fatalf("disabled admin hash should not authenticate: ok=%v err=%v", ok, err)
	}

	events, err := repo.ListAuditEvents(ctx, store.AuditEventFilter{ResourceType: "admin_identity", ResourceID: identity.ID})
	if err != nil {
		t.Fatalf("list admin identity audit events: %v", err)
	}
	if got := postgresAuditActions(events); !reflect.DeepEqual(got, []string{"admin_identity.created", "admin_identity.key_rotated", "admin_identity.disabled"}) {
		t.Fatalf("unexpected admin identity audit actions: %#v", got)
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
	byID, err := repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
		ID:              application.ID,
		ManagementScope: store.ManagementScope{TenantID: caller.TenantID, WorkspaceID: caller.WorkspaceID},
	})
	if err != nil {
		t.Fatalf("list permission package application by id: %v", err)
	}
	if len(byID) != 1 || byID[0].ID != application.ID || byID[0].DraftID != application.DraftID {
		t.Fatalf("expected exact permission package application by id, got %#v", byID)
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
		ExpiresAt:   now.Add(24 * time.Hour),
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
		len(approvalRows[0].DataScopes) != 1 || approvalRows[0].DataScopes[0].Region != "us-east" ||
		!approvalRows[0].ExpiresAt.Equal(now.Add(24*time.Hour)) {
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
	loadedApproval.ConsumedAt = now.Add(60 * time.Second)
	loadedApproval.ConsumedByApplicationID = "ppa_pg"
	updatedApproval, ok, err := repo.UpdatePermissionPackageApprovalRequest(ctx, loadedApproval)
	if err != nil {
		t.Fatalf("update permission package approval request: %v", err)
	}
	if !ok || updatedApproval.Status != domain.PermissionPackageApprovalStatusApproved ||
		updatedApproval.ReviewedBy != "security" || updatedApproval.ResolvedAt.IsZero() ||
		updatedApproval.ConsumedByApplicationID != "ppa_pg" || updatedApproval.ConsumedAt.IsZero() ||
		!updatedApproval.ExpiresAt.Equal(now.Add(24*time.Hour)) {
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

	applyApproval, err := repo.CreatePermissionPackageApprovalRequest(ctx, domain.PermissionPackageApprovalRequest{
		ID:                    security.NewID("ppar"),
		DraftID:               "ppd_pg_apply",
		TemplateID:            "support-write",
		TemplateVersion:       1,
		PolicyVersion:         1,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		TargetID:              target.ID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:pg-apply",
		RequestText:           "grant support write access",
		Region:                "us-east",
		DataScopes:            []domain.DataScope{{DataDomain: "crm", Region: "us-east", TenantFilter: "tenant_id = 'tenant-pg-cap'"}},
		AllowedCapabilityIDs:  []string{capability.ID},
		AllowedCapabilityKeys: []string{capability.Key},
		PolicyGate: domain.PermissionPackagePolicyGate{
			Decision:         domain.PermissionPackagePolicyDecisionApprovalRequired,
			CanApplyDirectly: false,
			PolicyVersion:    1,
		},
		Status:      domain.PermissionPackageApprovalStatusApproved,
		RequestedBy: "admin",
		ReviewedBy:  "security",
		CreatedAt:   now.Add(70 * time.Second),
		UpdatedAt:   now.Add(70 * time.Second),
		ResolvedAt:  now.Add(70 * time.Second),
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create apply approval request: %v", err)
	}
	applyTime := now.Add(80 * time.Second)
	applyEntitlement := domain.TenantEntitlement{
		ID:           security.NewID("ent"),
		TenantID:     caller.TenantID,
		TargetID:     target.ID,
		CapabilityID: capability.ID,
		Effect:       domain.PolicyEffectAllow,
		Status:       domain.PolicyStatusEnabled,
		Priority:     40,
		CreatedAt:    applyTime,
		UpdatedAt:    applyTime,
	}
	applyWorkspaceAssignment := domain.WorkspaceAssignment{
		ID:                  security.NewID("wsa"),
		TenantEntitlementID: applyEntitlement.ID,
		TenantID:            caller.TenantID,
		WorkspaceID:         caller.WorkspaceID,
		Effect:              domain.PolicyEffectAllow,
		Status:              domain.PolicyStatusEnabled,
		CreatedAt:           applyTime,
		UpdatedAt:           applyTime,
	}
	applyInstanceAssignment := domain.InstanceAssignment{
		ID:                    security.NewID("ina"),
		WorkspaceAssignmentID: applyWorkspaceAssignment.ID,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		CallerInstanceID:      caller.ID,
		SubjectSelector:       "user:pg-apply",
		Effect:                domain.PolicyEffectAllow,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             applyTime,
		UpdatedAt:             applyTime,
	}
	applyApplication := domain.PermissionPackageApplication{
		ID:                     security.NewID("ppa"),
		DraftID:                applyApproval.DraftID,
		TemplateID:             applyApproval.TemplateID,
		TemplateVersion:        applyApproval.TemplateVersion,
		TenantID:               caller.TenantID,
		WorkspaceID:            caller.WorkspaceID,
		TargetID:               target.ID,
		CallerInstanceID:       caller.ID,
		SubjectSelector:        "user:pg-apply",
		RequestText:            applyApproval.RequestText,
		Region:                 applyApproval.Region,
		DataScopes:             applyApproval.DataScopes,
		AllowedCapabilityIDs:   []string{capability.ID},
		AllowedCapabilityKeys:  []string{capability.Key},
		TenantEntitlementIDs:   []string{applyEntitlement.ID},
		WorkspaceAssignmentIDs: []string{applyWorkspaceAssignment.ID},
		InstanceAssignmentIDs:  []string{applyInstanceAssignment.ID},
		AppliedAt:              applyTime,
	}
	consumedApproval := applyApproval
	consumedApproval.ConsumedAt = applyTime
	consumedApproval.ConsumedByApplicationID = applyApplication.ID
	consumedApproval.UpdatedAt = applyTime
	applyMutation := store.PermissionPackageApplyMutation{
		Capabilities:         []domain.Capability{capability},
		TenantEntitlements:   []domain.TenantEntitlement{applyEntitlement},
		WorkspaceAssignments: []domain.WorkspaceAssignment{applyWorkspaceAssignment},
		InstanceAssignments:  []domain.InstanceAssignment{applyInstanceAssignment},
		Application:          applyApplication,
		ApprovalRequest:      &consumedApproval,
		AuditEvent: domain.AuditEvent{
			ID:           security.NewID("aud"),
			TenantID:     caller.TenantID,
			WorkspaceID:  caller.WorkspaceID,
			Actor:        "admin",
			Action:       "permission_package.applied",
			ResourceType: "permission_package",
			ResourceID:   applyApplication.ID,
			Summary:      "Permission package applied",
			Metadata:     map[string]any{"approvalRequestId": applyApproval.ID},
			CreatedAt:    applyTime,
		},
	}
	applyResult, err := repo.ApplyPermissionPackage(ctx, applyMutation)
	if err != nil {
		t.Fatalf("apply permission package mutation: %v", err)
	}
	if applyResult.Application.ID != applyApplication.ID || applyResult.ApprovalRequest == nil ||
		applyResult.ApprovalRequest.ConsumedByApplicationID != applyApplication.ID || applyResult.ApprovalRequest.ConsumedAt.IsZero() {
		t.Fatalf("unexpected apply mutation result: %#v", applyResult)
	}
	loadedApplyApproval, ok, err := repo.GetPermissionPackageApprovalRequest(ctx, applyApproval.ID)
	if err != nil || !ok {
		t.Fatalf("get applied approval request: ok=%v err=%v", ok, err)
	}
	if loadedApplyApproval.ConsumedByApplicationID != applyApplication.ID || loadedApplyApproval.ConsumedAt.IsZero() {
		t.Fatalf("approval request should be consumed by application %s: %#v", applyApplication.ID, loadedApplyApproval)
	}
	if _, err := repo.ApplyPermissionPackage(ctx, applyMutation); !errors.Is(err, store.ErrPermissionPackageApprovalNotConsumable) {
		t.Fatalf("expected consumed approval error on retry, got %v", err)
	}
	applyApplications, err := repo.ListPermissionPackageApplications(ctx, store.PermissionPackageApplicationFilter{
		ManagementScope:  store.ManagementScope{TenantID: caller.TenantID, WorkspaceID: caller.WorkspaceID},
		TemplateID:       "support-write",
		TargetID:         target.ID,
		CallerInstanceID: caller.ID,
	})
	if err != nil || len(applyApplications) != 1 || applyApplications[0].ID != applyApplication.ID {
		t.Fatalf("retry should not create duplicate applications, applications=%#v err=%v", applyApplications, err)
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

func adminIdentityIDsContain(rows []domain.AdminIdentity, id string) bool {
	for _, row := range rows {
		if row.ID == id {
			return true
		}
	}
	return false
}

func postgresAuditActions(events []domain.AuditEvent) []string {
	actions := make([]string, 0, len(events))
	for _, event := range events {
		actions = append(actions, event.Action)
	}
	return actions
}
