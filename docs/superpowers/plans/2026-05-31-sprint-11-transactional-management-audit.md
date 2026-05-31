# Sprint 11 Transactional Management Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make covered management-plane mutations commit their audit event atomically with the business state change.

**Architecture:** Add explicit audited repository methods for each covered management mutation. Memory holds one write lock for mutation plus audit append; PostgreSQL runs the mutation and `audit_events` insert in one transaction. HTTP handlers build the same secret-free audit metadata as today, then call the audited methods instead of best-effort `recordManagementAudit`.

**Tech Stack:** Go 1.25, chi HTTP handlers, pgx PostgreSQL transactions, in-memory repository tests, shell demo script.

---

## File Structure

- Modify `internal/store/memory.go`: extend `Repository` with audited mutation methods and implement memory atomic writes.
- Modify `internal/store/postgres.go`: add transaction helpers, transaction-scoped mutation helpers, and audited mutation methods.
- Modify `internal/httpapi/server.go`: replace covered best-effort audit writes with audited repository calls.
- Modify `internal/httpapi/server_test.go`: add HTTP failure-injection coverage for create/update rollback behavior.
- Modify `internal/store/postgres_test.go`: add PostgreSQL rollback coverage for audit insert failure inside a transaction.
- Create `docs/sprints/sprint-11-brief.md`: short product/engineering brief for the sprint.
- Create `scripts/demo-sprint11-transactional-audit.sh`: public API demo for normal audit visibility and redaction.
- Modify `README.md`: document the Sprint 11 demo and strengthened audit contract.
- Modify `CHANGELOG.md`: record the Sprint 11 implementation and verification.

Keep staging exact Sprint 11 paths only. The worktree may already contain Sprint 10 files.

### Task 1: HTTP Red Tests for Audit Failure Blocking

**Files:**
- Modify: `internal/httpapi/server_test.go`

- [ ] **Step 1: Add a failing audited repository wrapper**

Add this type near `failingAllowedTraceRepository` at the bottom of `internal/httpapi/server_test.go`:

```go
type failingAuditedAgentRepository struct {
	store.Repository
}

func (r *failingAuditedAgentRepository) CreateAgentWithAudit(ctx context.Context, agent domain.Agent, audit domain.AuditEvent) (domain.Agent, error) {
	return domain.Agent{}, errors.New("audit insert failed")
}

func (r *failingAuditedAgentRepository) UpdateAgentWithAudit(ctx context.Context, agent domain.Agent, audit domain.AuditEvent) (domain.Agent, bool, error) {
	return domain.Agent{}, false, errors.New("audit insert failed")
}
```

- [ ] **Step 2: Add the HTTP regression test**

Add this test near the existing management audit tests:

```go
func TestManagementAuditFailureBlocksAgentCreateAndUpdate(t *testing.T) {
	base := store.NewMemory()
	router := newRouterWithRepo(&failingAuditedAgentRepository{Repository: base})

	createResp := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "Audit Required Agent",
		"tenantId":    "tenant-audit-failure",
		"workspaceId": "ws-audit-failure",
		"channelType": "local",
		"status":      "active",
	}, "")
	if createResp.Code != http.StatusInternalServerError {
		t.Fatalf("create should fail when audit persistence fails: status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	agents, err := base.ListAgents(t.Context(), store.AgentFilter{ManagementScope: store.ManagementScope{
		TenantID:    "tenant-audit-failure",
		WorkspaceID: "ws-audit-failure",
	}})
	if err != nil {
		t.Fatalf("list agents after failed create: %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("failed audited create should not persist agent: %#v", agents)
	}

	existing := domain.Agent{
		ID:            security.NewID("agt"),
		TenantID:      "tenant-audit-failure",
		WorkspaceID:   "ws-audit-failure",
		Name:          "Original Agent",
		ChannelType:   "local",
		ChannelConfig: map[string]any{},
		Status:        domain.AgentStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if _, err := base.CreateAgent(t.Context(), existing); err != nil {
		t.Fatalf("seed existing agent: %v", err)
	}

	updateResp := request(t, router, http.MethodPatch, "/api/v1/agents/"+existing.ID, map[string]any{
		"name": "Updated Agent",
	}, "")
	if updateResp.Code != http.StatusInternalServerError {
		t.Fatalf("update should fail when audit persistence fails: status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	after, ok, err := base.GetAgent(t.Context(), existing.ID)
	if err != nil {
		t.Fatalf("get agent after failed update: %v", err)
	}
	if !ok {
		t.Fatalf("seeded agent disappeared after failed update")
	}
	if after.Name != "Original Agent" {
		t.Fatalf("failed audited update should keep previous name, got %q", after.Name)
	}
}
```

- [ ] **Step 3: Run the red test**

Run:

```bash
go test ./internal/httpapi -run TestManagementAuditFailureBlocksAgentCreateAndUpdate -count=1
```

Expected: FAIL. The create path currently returns `201` because handlers still call `CreateAgent` and then ignore best-effort audit persistence.

- [ ] **Step 4: Commit the red test**

```bash
git add internal/httpapi/server_test.go
git commit -m "test: cover transactional audit failures"
```

### Task 2: PostgreSQL Red Test for Rollback Semantics

**Files:**
- Modify: `internal/store/postgres_test.go`

- [ ] **Step 1: Add the rollback integration test**

Add this test after `TestPostgresRepositoryRoundTrip` in `internal/store/postgres_test.go`:

```go
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

	if _, err := repo.CreateAgentWithAudit(ctx, agent, audit); err == nil {
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
```

- [ ] **Step 2: Run the red PostgreSQL test**

Run without PostgreSQL:

```bash
go test ./internal/store -run TestPostgresAuditedCreateAgentRollsBackWhenAuditFails -count=1
```

Expected without `AGENT_HARBOR_TEST_DATABASE_URL`: SKIP.

Run with PostgreSQL:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' go test ./internal/store -run TestPostgresAuditedCreateAgentRollsBackWhenAuditFails -count=1
```

Expected with PostgreSQL before implementation: compile FAIL with `repo.CreateAgentWithAudit undefined`. After implementation, the test should fail the audit insert with a duplicate audit id and prove the Agent row rolls back.

- [ ] **Step 3: Commit the red PostgreSQL test**

```bash
git add internal/store/postgres_test.go
git commit -m "test: cover postgres audited rollback"
```

### Task 3: Audited Repository Methods

**Files:**
- Modify: `internal/store/memory.go`
- Modify: `internal/store/postgres.go`

- [ ] **Step 1: Extend the repository interface**

In `internal/store/memory.go`, add these methods to `type Repository interface` next to their unaudited counterparts:

```go
CreateAgentWithAudit(context.Context, domain.Agent, domain.AuditEvent) (domain.Agent, error)
UpdateAgentWithAudit(context.Context, domain.Agent, domain.AuditEvent) (domain.Agent, bool, error)
RotateAgentCredentialsWithAudit(context.Context, string, map[string]string, time.Time, domain.AuditEvent) (domain.Agent, bool, error)
DisableAgentWithAudit(context.Context, string, time.Time, domain.AuditEvent) (domain.Agent, bool, error)
CreateAgentKeyWithAudit(context.Context, domain.AgentKey, domain.AuditEvent) (domain.AgentKey, error)
RevokeAgentKeyWithAudit(context.Context, string, time.Time, domain.AuditEvent) (domain.AgentKey, bool, error)
CreateAccessGrantWithAudit(context.Context, domain.AccessGrant, domain.AuditEvent) (domain.AccessGrant, error)
RevokeAccessGrantWithAudit(context.Context, string, time.Time, domain.AuditEvent) (domain.AccessGrant, bool, error)
CreateRoutePolicyWithAudit(context.Context, domain.RoutePolicy, domain.AuditEvent) (domain.RoutePolicy, error)
UpdateRoutePolicyWithAudit(context.Context, domain.RoutePolicy, domain.AuditEvent) (domain.RoutePolicy, bool, error)
DisableRoutePolicyWithAudit(context.Context, string, time.Time, domain.AuditEvent) (domain.RoutePolicy, bool, error)
```

- [ ] **Step 2: Implement memory audited methods**

Add this block to `internal/store/memory.go` after the unaudited mutation methods:

```go
func (m *Memory) CreateAgentWithAudit(_ context.Context, agent domain.Agent, audit domain.AuditEvent) (domain.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agent.ID] = agent
	m.audits = append(m.audits, audit)
	return agent, nil
}

func (m *Memory) UpdateAgentWithAudit(_ context.Context, agent domain.Agent, audit domain.AuditEvent) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.agents[agent.ID]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Credentials = existing.Credentials
	agent.CredentialVersion = existing.CredentialVersion
	m.agents[agent.ID] = agent
	m.audits = append(m.audits, audit)
	return agent, true, nil
}

func (m *Memory) RotateAgentCredentialsWithAudit(_ context.Context, id string, credentials map[string]string, now time.Time, audit domain.AuditEvent) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[id]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Credentials = credentials
	agent.CredentialVersion++
	agent.UpdatedAt = now
	m.agents[id] = agent
	m.audits = append(m.audits, audit)
	return agent, true, nil
}

func (m *Memory) DisableAgentWithAudit(_ context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.Agent, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.agents[id]
	if !ok {
		return domain.Agent{}, false, nil
	}
	agent.Status = domain.AgentStatusDisabled
	agent.UpdatedAt = now
	m.agents[id] = agent
	m.audits = append(m.audits, audit)
	return agent, true, nil
}

func (m *Memory) CreateAgentKeyWithAudit(_ context.Context, key domain.AgentKey, audit domain.AuditEvent) (domain.AgentKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[key.ID] = key
	m.audits = append(m.audits, audit)
	return key, nil
}

func (m *Memory) RevokeAgentKeyWithAudit(_ context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.AgentKey, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.keys[id]
	if !ok {
		return domain.AgentKey{}, false, nil
	}
	key.RevokedAt = now
	m.keys[id] = key
	m.audits = append(m.audits, audit)
	return key, true, nil
}

func (m *Memory) CreateAccessGrantWithAudit(_ context.Context, grant domain.AccessGrant, audit domain.AuditEvent) (domain.AccessGrant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grants[grant.ID] = grant
	m.audits = append(m.audits, audit)
	return grant, nil
}

func (m *Memory) RevokeAccessGrantWithAudit(_ context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.AccessGrant, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	grant, ok := m.grants[id]
	if !ok {
		return domain.AccessGrant{}, false, nil
	}
	grant.RevokedAt = now
	m.grants[id] = grant
	m.audits = append(m.audits, audit)
	return grant, true, nil
}

func (m *Memory) CreateRoutePolicyWithAudit(_ context.Context, policy domain.RoutePolicy, audit domain.AuditEvent) (domain.RoutePolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	policy.Retry = cloneRoutePolicyRetry(policy.Retry)
	m.policies[policy.ID] = policy
	m.audits = append(m.audits, audit)
	return policy, nil
}

func (m *Memory) UpdateRoutePolicyWithAudit(_ context.Context, policy domain.RoutePolicy, audit domain.AuditEvent) (domain.RoutePolicy, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.policies[policy.ID]
	if !ok {
		return domain.RoutePolicy{}, false, nil
	}
	policy.Retry = cloneRoutePolicyRetry(policy.Retry)
	policy.TenantID = existing.TenantID
	policy.WorkspaceID = existing.WorkspaceID
	policy.CallerID = existing.CallerID
	policy.TargetID = existing.TargetID
	policy.CreatedAt = existing.CreatedAt
	m.policies[policy.ID] = policy
	m.audits = append(m.audits, audit)
	return policy, true, nil
}

func (m *Memory) DisableRoutePolicyWithAudit(_ context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.RoutePolicy, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	policy, ok := m.policies[id]
	if !ok {
		return domain.RoutePolicy{}, false, nil
	}
	policy.Status = domain.RoutePolicyStatusDisabled
	policy.UpdatedAt = now
	m.policies[id] = policy
	m.audits = append(m.audits, audit)
	return policy, true, nil
}
```

- [ ] **Step 3: Add PostgreSQL transaction scaffolding**

In `internal/store/postgres.go`, add `github.com/jackc/pgx/v5/pgconn` to the imports and add these helpers near `type Postgres`:

```go
type sqlExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (p *Postgres) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Refactor PostgreSQL mutation helpers**

Change each existing mutation method to call a transaction-capable helper. The pattern for `CreateAgent` is:

```go
func (p *Postgres) CreateAgent(ctx context.Context, agent domain.Agent) (domain.Agent, error) {
	return p.createAgent(ctx, p.pool, agent)
}

func (p *Postgres) createAgent(ctx context.Context, exec sqlExecutor, agent domain.Agent) (domain.Agent, error) {
	config, err := json.Marshal(agent.ChannelConfig)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("marshal channel config: %w", err)
	}
	credentialsCiphertext, err := security.EncryptCredentials(agent.Credentials, p.credentialKey)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("encrypt credentials: %w", err)
	}
	if credentialsCiphertext == nil {
		credentialsCiphertext = []byte{}
	}
	_, err = exec.Exec(ctx, `
		insert into agents (
			id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, credentials_ciphertext, credential_version, status, created_at, updated_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, agent.ID, agent.TenantID, agent.WorkspaceID, agent.Name, agent.Description, agent.OwnerID,
		agent.ChannelType, config, credentialsCiphertext, agent.CredentialVersion, string(agent.Status), agent.CreatedAt, agent.UpdatedAt)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("insert agent: %w", err)
	}
	return agent, nil
}
```

Extract the remaining mutation helpers by moving each current public method body into the matching private method below. Each public method should become a wrapper that passes `p.pool` to its private helper. Keep every existing SQL statement, scan call, and `fmt.Errorf` message inside the matching private helper.

```go
func (p *Postgres) updateAgent(ctx context.Context, exec sqlExecutor, agent domain.Agent) (domain.Agent, bool, error)
func (p *Postgres) rotateAgentCredentials(ctx context.Context, exec sqlExecutor, id string, credentials map[string]string, now time.Time) (domain.Agent, bool, error)
func (p *Postgres) disableAgent(ctx context.Context, exec sqlExecutor, id string, now time.Time) (domain.Agent, bool, error)
func (p *Postgres) createAgentKey(ctx context.Context, exec sqlExecutor, key domain.AgentKey) (domain.AgentKey, error)
func (p *Postgres) revokeAgentKey(ctx context.Context, exec sqlExecutor, id string, now time.Time) (domain.AgentKey, bool, error)
func (p *Postgres) createAccessGrant(ctx context.Context, exec sqlExecutor, grant domain.AccessGrant) (domain.AccessGrant, error)
func (p *Postgres) revokeAccessGrant(ctx context.Context, exec sqlExecutor, id string, now time.Time) (domain.AccessGrant, bool, error)
func (p *Postgres) createRoutePolicy(ctx context.Context, exec sqlExecutor, policy domain.RoutePolicy) (domain.RoutePolicy, error)
func (p *Postgres) updateRoutePolicy(ctx context.Context, exec sqlExecutor, policy domain.RoutePolicy) (domain.RoutePolicy, bool, error)
func (p *Postgres) disableRoutePolicy(ctx context.Context, exec sqlExecutor, id string, now time.Time) (domain.RoutePolicy, bool, error)
func (p *Postgres) appendAuditEvent(ctx context.Context, exec sqlExecutor, event domain.AuditEvent) (domain.AuditEvent, error)
```

For example, `AppendAuditEvent` should become:

```go
func (p *Postgres) AppendAuditEvent(ctx context.Context, event domain.AuditEvent) (domain.AuditEvent, error) {
	return p.appendAuditEvent(ctx, p.pool, event)
}

func (p *Postgres) appendAuditEvent(ctx context.Context, exec sqlExecutor, event domain.AuditEvent) (domain.AuditEvent, error) {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return domain.AuditEvent{}, fmt.Errorf("marshal audit metadata: %w", err)
	}
	_, err = exec.Exec(ctx, `
		insert into audit_events (
			id, tenant_id, workspace_id, actor, action, resource_type, resource_id, summary, metadata, created_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, event.ID, event.TenantID, event.WorkspaceID, event.Actor, event.Action, event.ResourceType,
		event.ResourceID, event.Summary, metadata, event.CreatedAt)
	if err != nil {
		return domain.AuditEvent{}, fmt.Errorf("insert audit event: %w", err)
	}
	return event, nil
}
```

- [ ] **Step 5: Add PostgreSQL audited methods**

Add these audited methods in `internal/store/postgres.go` after the unaudited mutation methods:

For every method, the domain mutation helper must run before `appendAuditEvent` inside the same `withTx` callback. Reviewers will check this ordering because the duplicate-audit-id rollback test verifies final state, while code review verifies the business write is attempted before audit insert failure.

```go
func (p *Postgres) CreateAgentWithAudit(ctx context.Context, agent domain.Agent, audit domain.AuditEvent) (domain.Agent, error) {
	var created domain.Agent
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createAgent(ctx, tx, agent)
		if err != nil {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) UpdateAgentWithAudit(ctx context.Context, agent domain.Agent, audit domain.AuditEvent) (domain.Agent, bool, error) {
	var updated domain.Agent
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		updated, ok, err = p.updateAgent(ctx, tx, agent)
		if err != nil || !ok {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return updated, ok, err
}

func (p *Postgres) RotateAgentCredentialsWithAudit(ctx context.Context, id string, credentials map[string]string, now time.Time, audit domain.AuditEvent) (domain.Agent, bool, error) {
	var updated domain.Agent
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		updated, ok, err = p.rotateAgentCredentials(ctx, tx, id, credentials, now)
		if err != nil || !ok {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return updated, ok, err
}

func (p *Postgres) DisableAgentWithAudit(ctx context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.Agent, bool, error) {
	var agent domain.Agent
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		agent, ok, err = p.disableAgent(ctx, tx, id, now)
		if err != nil || !ok {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return agent, ok, err
}

func (p *Postgres) CreateAgentKeyWithAudit(ctx context.Context, key domain.AgentKey, audit domain.AuditEvent) (domain.AgentKey, error) {
	var created domain.AgentKey
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createAgentKey(ctx, tx, key)
		if err != nil {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) RevokeAgentKeyWithAudit(ctx context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.AgentKey, bool, error) {
	var key domain.AgentKey
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		key, ok, err = p.revokeAgentKey(ctx, tx, id, now)
		if err != nil || !ok {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return key, ok, err
}

func (p *Postgres) CreateAccessGrantWithAudit(ctx context.Context, grant domain.AccessGrant, audit domain.AuditEvent) (domain.AccessGrant, error) {
	var created domain.AccessGrant
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createAccessGrant(ctx, tx, grant)
		if err != nil {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) RevokeAccessGrantWithAudit(ctx context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.AccessGrant, bool, error) {
	var grant domain.AccessGrant
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		grant, ok, err = p.revokeAccessGrant(ctx, tx, id, now)
		if err != nil || !ok {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return grant, ok, err
}

func (p *Postgres) CreateRoutePolicyWithAudit(ctx context.Context, policy domain.RoutePolicy, audit domain.AuditEvent) (domain.RoutePolicy, error) {
	var created domain.RoutePolicy
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createRoutePolicy(ctx, tx, policy)
		if err != nil {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) UpdateRoutePolicyWithAudit(ctx context.Context, policy domain.RoutePolicy, audit domain.AuditEvent) (domain.RoutePolicy, bool, error) {
	var updated domain.RoutePolicy
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		updated, ok, err = p.updateRoutePolicy(ctx, tx, policy)
		if err != nil || !ok {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return updated, ok, err
}

func (p *Postgres) DisableRoutePolicyWithAudit(ctx context.Context, id string, now time.Time, audit domain.AuditEvent) (domain.RoutePolicy, bool, error) {
	var policy domain.RoutePolicy
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		policy, ok, err = p.disableRoutePolicy(ctx, tx, id, now)
		if err != nil || !ok {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return policy, ok, err
}
```

- [ ] **Step 6: Run store tests**

Run:

```bash
go test ./internal/store -count=1
```

Expected without PostgreSQL env: PASS with PostgreSQL integration tests skipped.

Run with PostgreSQL when available:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' go test ./internal/store -run 'TestPostgresRepositoryRoundTrip|TestPostgresAuditedCreateAgentRollsBackWhenAuditFails' -count=1
```

Expected with PostgreSQL env: PASS.

- [ ] **Step 7: Commit repository implementation**

```bash
git add internal/store/memory.go internal/store/postgres.go internal/store/postgres_test.go
git commit -m "feat: add transactional audited repository writes"
```

### Task 4: HTTP Handler Integration

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [ ] **Step 1: Add an audit event builder**

Replace `recordManagementAudit` with this event builder in `internal/httpapi/server.go`:

```go
func (s *Server) managementAuditEvent(r *http.Request, tenantID string, workspaceID string, action string, resourceType string, resourceID string, summary string, metadata map[string]any) domain.AuditEvent {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return domain.AuditEvent{
		ID:           security.NewID("aud"),
		TenantID:     tenantID,
		WorkspaceID:  workspaceID,
		Actor:        managementActor(r),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Summary:      summary,
		Metadata:     metadata,
		CreatedAt:    s.now(),
	}
}
```

- [ ] **Step 2: Update Agent handlers**

Change `createAgent`, `updateAgent`, `rotateAgentCredentials`, and `disableAgent` to build an audit event and call the audited repository method. Use this pattern for `createAgent`:

```go
audit := s.managementAuditEvent(r, agent.TenantID, agent.WorkspaceID, "agent.created", "agent", agent.ID, "Agent created", map[string]any{
	"channelType":        agent.ChannelType,
	"status":             string(agent.Status),
	"credentialVersion":  agent.CredentialVersion,
	"hasCredentials":     len(agent.Credentials) > 0,
	"credentialKeyCount": len(agent.Credentials),
})
created, err := s.repo.CreateAgentWithAudit(r.Context(), agent, audit)
if err != nil {
	writeError(w, err)
	return
}
writeJSON(w, http.StatusCreated, created)
```

Use this exact audit metadata for the other Agent paths:

```go
updateAudit := s.managementAuditEvent(r, updated.TenantID, updated.WorkspaceID, "agent.updated", "agent", updated.ID, "Agent updated", map[string]any{
	"fields":            agentPatchFields(req),
	"status":            string(updated.Status),
	"credentialVersion": updated.CredentialVersion,
})

rotateAudit := s.managementAuditEvent(r, agent.TenantID, agent.WorkspaceID, "agent.credentials_rotated", "agent", agent.ID, "Agent credentials rotated", map[string]any{
	"credentialKeys":    credentialKeyNames(credentials),
	"credentialVersion": agent.CredentialVersion + 1,
})

disableAudit := s.managementAuditEvent(r, agent.TenantID, agent.WorkspaceID, "agent.disabled", "agent", agent.ID, "Agent disabled", map[string]any{
	"status":            string(domain.AgentStatusDisabled),
	"credentialVersion": agent.CredentialVersion,
})
```

For `disableAgent`, load the existing Agent first with `GetAgent` so the audit event has scope and credential version before calling `DisableAgentWithAudit`.

- [ ] **Step 3: Update Agent Key and Access Grant handlers**

Use the audited methods in `createAgentKey`, `revokeAgentKey`, `createAccessGrant`, and `revokeAccessGrant`.

The `createAgentKey` metadata stays:

```go
map[string]any{
	"agentId":   key.AgentID,
	"name":      key.Name,
	"expiresAt": key.ExpiresAt,
}
```

The `revokeAgentKey` metadata stays:

```go
map[string]any{
	"agentId": key.AgentID,
	"name":    key.Name,
}
```

The access grant metadata stays:

```go
map[string]any{
	"callerAgentId": grant.CallerID,
	"targetAgentId": grant.TargetID,
	"routeType":     grant.RouteType,
	"routeKey":      grant.RouteKey,
}
```

- [ ] **Step 4: Update Route Policy handlers**

Use audited repository methods in `createRoutePolicy`, `updateRoutePolicy`, and `disableRoutePolicy`:

```go
audit := s.managementAuditEvent(r, policy.TenantID, policy.WorkspaceID, "route_policy.created", "route_policy", policy.ID, "Route policy created", routePolicyAuditMetadata(policy))
created, err := s.repo.CreateRoutePolicyWithAudit(r.Context(), policy, audit)
```

```go
audit := s.managementAuditEvent(r, policy.TenantID, policy.WorkspaceID, "route_policy.updated", "route_policy", policy.ID, "Route policy updated", routePolicyAuditMetadata(policy))
updated, ok, err := s.repo.UpdateRoutePolicyWithAudit(r.Context(), policy, audit)
```

```go
now := s.now()
auditPolicy := disabledRoutePolicyForAudit(existing, now)
audit := s.managementAuditEvent(r, existing.TenantID, existing.WorkspaceID, "route_policy.disabled", "route_policy", existing.ID, "Route policy disabled", routePolicyAuditMetadata(auditPolicy))
policy, ok, err := s.repo.DisableRoutePolicyWithAudit(r.Context(), existing.ID, now, audit)
```

If using the disabled audit helper, add this small function:

```go
func disabledRoutePolicyForAudit(policy domain.RoutePolicy, now time.Time) domain.RoutePolicy {
	policy.Status = domain.RoutePolicyStatusDisabled
	policy.UpdatedAt = now
	return policy
}
```

Store the `now := s.now()` value once in disable handlers so audit metadata and persisted update share the same timestamp.

- [ ] **Step 5: Run HTTP audit tests**

Run:

```bash
go test ./internal/httpapi -run 'TestManagementAuditFailureBlocksAgentCreateAndUpdate|TestManagementAuditEvents|TestRoutePolicyCRUDAndAudit' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit HTTP integration**

```bash
git add internal/httpapi/server.go internal/httpapi/server_test.go
git commit -m "feat: require transactional management audit"
```

### Task 5: Demo and Documentation

**Files:**
- Create: `docs/sprints/sprint-11-brief.md`
- Create: `scripts/demo-sprint11-transactional-audit.sh`
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Create the Sprint 11 brief**

Create `docs/sprints/sprint-11-brief.md`:

```markdown
# Sprint 11 Brief: Transactional Management Audit

Status: Planned after Sprint 10 closeout

## Goal

Make management audit events part of the write contract so successful control-plane mutations cannot commit without matching audit evidence.

## User Stories

- As a platform operator, every successful Agent, key, grant, or route policy mutation has durable audit evidence.
- As an engineer, audit persistence failures roll back covered PostgreSQL management mutations.
- As a developer, memory and PostgreSQL repositories expose the same audited mutation semantics.

## Acceptance

- Covered HTTP management handlers call audited repository methods.
- Memory store appends the audit event under the same write lock as the mutation.
- PostgreSQL store writes mutation and audit event in one transaction.
- Audit metadata remains secret-free.
- PostgreSQL rollback test proves duplicate audit insert failure prevents the Agent row from persisting.
- Existing audit listing filters continue to work.

## Non-goals

- No external outbox worker.
- No OpenTelemetry export.
- No data-plane trace transaction change.
- No route policy import/export.
```

- [ ] **Step 2: Add the Sprint 11 demo script**

Create `scripts/demo-sprint11-transactional-audit.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:9090}"
ADMIN_KEY="${ADMIN_KEY:-}"
RUN_ID="${RUN_ID:-sprint11-$(date +%s)}"

admin_args=()
if [[ -n "$ADMIN_KEY" ]]; then
  admin_args=(-H "X-Admin-Key: ${ADMIN_KEY}")
fi

curl_json() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  if [[ -n "$body" ]]; then
    curl -fsS -X "$method" "${BASE_URL}${path}" "${admin_args[@]}" -H 'Content-Type: application/json' -d "$body"
  else
    curl -fsS -X "$method" "${BASE_URL}${path}" "${admin_args[@]}" -H 'Content-Type: application/json'
  fi
}

extract_data_field() {
  local json="$1"
  local field="$2"
  python3 - "$field" "$json" <<'PY'
import json
import sys
field = sys.argv[1]
payload = json.loads(sys.argv[2])
print(payload["data"][field])
PY
}

echo "== Sprint 11 transactional audit demo (${RUN_ID}) =="

agent_payload=$(cat <<JSON
{
  "tenantId": "demo-sprint11",
  "workspaceId": "ws-sprint11",
  "name": "Sprint 11 Audited Target",
  "channelType": "mcp",
  "channelConfig": {
    "endpoint": "https://api.example.com/mcp",
    "credentialHeaders": {
      "Authorization": "apiToken"
    }
  },
  "credentials": {
    "apiToken": "Bearer sprint11-secret"
  },
  "status": "active"
}
JSON
)

agent_resp="$(curl_json POST /api/v1/agents "$agent_payload")"
agent_id="$(extract_data_field "$agent_resp" id)"
echo "Created audited agent: ${agent_id}"

rotate_resp="$(curl_json POST "/api/v1/agents/${agent_id}/credentials:rotate" '{"credentials":{"apiToken":"Bearer sprint11-rotated-secret"}}')"
echo "Rotated credentials: $(extract_data_field "$rotate_resp" credentialVersion)"

events="$(curl_json GET "/api/v1/audit/events?tenantId=demo-sprint11&workspaceId=ws-sprint11&resourceId=${agent_id}")"
python3 - "$events" <<'PY'
import json
import sys
payload = json.loads(sys.argv[1])
actions = [event["action"] for event in payload["data"]]
required = ["agent.created", "agent.credentials_rotated"]
missing = [action for action in required if action not in actions]
if missing:
    raise SystemExit(f"missing audit actions: {missing}; got {actions}")
text = json.dumps(payload)
if "sprint11-secret" in text or "sprint11-rotated-secret" in text:
    raise SystemExit("audit events leaked credential values")
print("Audit actions:", ", ".join(actions))
print("Credential values redacted from audit events")
PY
```

Run:

```bash
chmod +x scripts/demo-sprint11-transactional-audit.sh
bash -n scripts/demo-sprint11-transactional-audit.sh
```

Expected: shell syntax check passes.

- [ ] **Step 3: Update README**

Add this section after the Sprint 10 demo section:

````markdown
## Sprint 11 Transactional Audit Demo

Sprint 11 makes covered management audit writes transactional with the management mutation. Against a running API, this script creates and rotates a credentialed Agent, verifies audit events are visible, and checks credential values remain redacted:

```bash
bash scripts/demo-sprint11-transactional-audit.sh
```
````

Add the script to the Admin Key command block:

```bash
ADMIN_KEY=local-admin-key bash scripts/demo-sprint11-transactional-audit.sh
```

Add this paragraph to `## Proxy Controls` near the management audit paragraph:

```markdown
Management mutations that produce audit events commit the audit event with the business state change. If audit persistence fails, covered Agent, Agent Key, Access Grant, and Route Policy mutations fail instead of leaving unaudited state behind.
```

- [ ] **Step 4: Update CHANGELOG**

Prepend this entry to `CHANGELOG.md`:

```markdown
## [2026-05-31] Session: Sprint 11 Transactional Management Audit

### 完成
- 新增 audited repository mutation 方法，覆盖 Agent、Agent Key、Access Grant、Route Policy 管理写入。
- Memory store 在同一把写锁内完成业务变更和 audit append。
- PostgreSQL store 使用单事务提交业务变更和 `audit_events` 写入。
- HTTP 管理 mutation 从 best-effort audit 改为事务化 audit 写入。
- 新增 Sprint 11 demo，验证 audit 可见性和 credential redaction。

### 决策
- Sprint 11 不引入异步 outbox worker；`audit_events` 先作为本地事务化事件日志。
- data-plane trace append 保持非阻塞，不纳入本次事务化范围。

### 验证
- `go test ./internal/httpapi -run 'TestManagementAuditFailureBlocksAgentCreateAndUpdate|TestManagementAuditEvents|TestRoutePolicyCRUDAndAudit' -count=1`
- `go test ./internal/store -count=1`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `pnpm --dir frontend test`
- `pnpm --dir frontend build`
- `git diff --check`
- `bash -n scripts/demo-sprint11-transactional-audit.sh`

### 影响文件
- `internal/store/memory.go` / `internal/store/postgres.go`：事务化 audited mutations。
- `internal/httpapi/server.go` / `server_test.go`：HTTP 管理写入改用 audited repository 方法并补失败注入测试。
- `internal/store/postgres_test.go`：PostgreSQL rollback regression。
- `scripts/demo-sprint11-transactional-audit.sh`、`README.md`、`docs/sprints/`：Sprint 11 文档与 demo。
```

- [ ] **Step 5: Commit docs and demo**

```bash
git add docs/sprints/sprint-11-brief.md scripts/demo-sprint11-transactional-audit.sh README.md CHANGELOG.md
git commit -m "docs: add sprint 11 transactional audit demo"
```

### Task 6: Final Verification

**Files:**
- Verify all changed Sprint 11 files.

- [ ] **Step 1: Run focused backend tests**

```bash
go test ./internal/httpapi -run 'TestManagementAuditFailureBlocksAgentCreateAndUpdate|TestManagementAuditEvents|TestRoutePolicyCRUDAndAudit' -count=1
go test ./internal/store -count=1
```

Expected: PASS.

- [ ] **Step 2: Run full backend verification**

```bash
go test ./...
go vet ./...
go build ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend verification**

```bash
pnpm --dir frontend test
pnpm --dir frontend build
```

Expected: PASS.

- [ ] **Step 4: Run static checks**

```bash
git diff --check
bash -n scripts/demo-sprint11-transactional-audit.sh
```

Expected: PASS.

- [ ] **Step 5: Run the demo against a local API**

Start the API in another terminal:

```bash
AGENT_HARBOR_ADMIN_KEY=local-admin-key go run ./cmd/agent-harbor
```

Run:

```bash
ADMIN_KEY=local-admin-key bash scripts/demo-sprint11-transactional-audit.sh
```

Expected: the script prints the created Agent ID, the rotated credential version, the audit actions, and a redaction confirmation.

- [ ] **Step 6: Confirm exact staged files before any final commit**

```bash
git status --short
```

Expected: only intended Sprint 11 files are modified or staged by this sprint's commits. Existing Sprint 10 dirty files may still appear and must not be staged unless they were part of a Sprint 11 task.

- [ ] **Step 7: Final commit if verification changed docs or scripts**

```bash
git add CHANGELOG.md README.md docs/sprints/sprint-11-brief.md scripts/demo-sprint11-transactional-audit.sh
git commit -m "chore: record sprint 11 verification"
```

Expected: commit is needed only if verification required documentation changes after Task 5.
