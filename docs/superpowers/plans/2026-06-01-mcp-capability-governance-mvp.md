# MCP Capability Governance MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first tenant-scoped capability governance loop for MCP targets: discover tools, grant one tool to a tenant/workspace/caller instance, enforce `tools/list` and `tools/call`, and prove decisions in trace evidence.

**Architecture:** Extend the current Go service instead of replacing it. Existing `Agent` records remain both target records and caller instances; the new capability governance layer sits in front of legacy route-policy/access-grant fallback for MCP tool calls. The frontend adds a focused capability governance path while the existing cockpit stays navigable.

**Tech Stack:** Go 1.25, `chi`, in-memory repository, PostgreSQL migrations through the existing embed migrator, React + TypeScript + Vite frontend, existing shell demo pattern.

---

## File Structure

- Modify `internal/domain/types.go`: add capability, entitlement, workspace assignment, instance assignment, data-scope, and enriched trace types.
- Modify `internal/store/memory.go`: add repository methods and in-memory maps for capabilities and assignments; add capability-based evaluation.
- Modify `internal/store/postgres.go`: add PostgreSQL CRUD and evaluation methods matching the memory repository.
- Add `internal/db/migrations/007_capability_governance.sql`: create capability governance tables and trace enrichment columns.
- Modify `internal/httpapi/server.go`: add management endpoints, MCP discovery, capability enforcement, filtered `tools/list`, enriched traces, and audit metadata.
- Modify `internal/httpapi/server_test.go`: add end-to-end tests for discovery, grant/assignment, allowed call, denied call, filtered list, and trace evidence.
- Modify `internal/store/postgres_test.go`: add PostgreSQL persistence and evaluation coverage for capability governance.
- Modify `frontend/src/types.ts`: add capability, entitlement, workspace assignment, and instance assignment types.
- Modify `frontend/src/api.ts`: add API calls for discovery, list, grants, and assignments.
- Modify `frontend/src/data.ts`: add sample capability governance data.
- Modify `frontend/src/App.tsx`: add the minimum console path for discovery, entitlement, assignment, and test evidence.
- Modify `frontend/src/styles.css`: add compact table/form states for the new console path using the existing visual language.
- Add `scripts/demo-sprint12-mcp-capability-governance.sh`: prove the MVP loop against a running API.
- Modify `README.md`: document the new API surface and demo.

## API Shape

Add these management endpoints behind the existing admin gate:

```text
POST /api/v1/targets/{targetId}/capabilities:refresh
GET  /api/v1/capabilities?tenantId=&workspaceId=&targetId=
PATCH /api/v1/capabilities/{id}
POST /api/v1/tenant-entitlements
GET  /api/v1/tenant-entitlements?tenantId=&workspaceId=&targetId=&capabilityId=
POST /api/v1/workspace-assignments
GET  /api/v1/workspace-assignments?tenantId=&workspaceId=&entitlementId=
POST /api/v1/instance-assignments
GET  /api/v1/instance-assignments?tenantId=&workspaceId=&callerInstanceId=
```

Keep existing routes unchanged. Capability enforcement applies to:

```text
POST /api/v1/mcp/agents/{targetId}
POST /api/v1/mcp/agents/{targetId}/rpc
```

## Task 1: Domain Model And Memory Store

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/store/memory.go`
- Test: `internal/store/memory_test.go`

- [ ] **Step 1: Write failing memory-store tests**

Create `internal/store/memory_test.go` with tests that prove the new model can store and evaluate one MCP tool assignment:

```go
package store

import (
	"testing"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

func TestMemoryCapabilityAssignmentEvaluation(t *testing.T) {
	repo := NewMemory()
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	ctx := t.Context()

	caller := domain.Agent{ID: "agt_caller", TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	target := domain.Agent{ID: "agt_mcp", TenantID: "tenant-a", WorkspaceID: "ws-sales", Name: "MCP", ChannelType: "mcp", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(ctx, caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	if _, err := repo.CreateAgent(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	capability := domain.Capability{
		ID:              "cap_search",
		TargetID:        target.ID,
		Type:            domain.CapabilityTypeMCPTool,
		Key:             "search_customer",
		DisplayName:     "search_customer",
		Action:          domain.CapabilityActionRead,
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

	denied, err := repo.EvaluateCapabilityAccess(ctx, CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate denied: %v", err)
	}
	if denied.Allowed {
		t.Fatalf("capability should be denied before entitlement and assignments: %#v", denied)
	}

	entitlement, err := repo.CreateTenantEntitlement(ctx, domain.TenantEntitlement{
		ID:           "ent_search",
		TenantID:     caller.TenantID,
		TargetID:     target.ID,
		CapabilityID: capability.ID,
		Effect:       domain.PolicyEffectAllow,
		Status:       domain.PolicyStatusEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	workspaceAssignment, err := repo.CreateWorkspaceAssignment(ctx, domain.WorkspaceAssignment{
		ID:                  "wsa_search",
		TenantEntitlementID: entitlement.ID,
		TenantID:            caller.TenantID,
		WorkspaceID:         caller.WorkspaceID,
		Effect:              domain.PolicyEffectAllow,
		Status:              domain.PolicyStatusEnabled,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		t.Fatalf("create workspace assignment: %v", err)
	}
	if _, err := repo.CreateInstanceAssignment(ctx, domain.InstanceAssignment{
		ID:                    "ina_search",
		WorkspaceAssignmentID: workspaceAssignment.ID,
		TenantID:              caller.TenantID,
		WorkspaceID:           caller.WorkspaceID,
		CallerInstanceID:      caller.ID,
		Effect:                domain.PolicyEffectAllow,
		Status:                domain.PolicyStatusEnabled,
		CreatedAt:             now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatalf("create instance assignment: %v", err)
	}

	allowed, err := repo.EvaluateCapabilityAccess(ctx, CapabilityAccessRequest{
		TenantID:         caller.TenantID,
		WorkspaceID:      caller.WorkspaceID,
		CallerInstanceID: caller.ID,
		TargetID:         target.ID,
		CapabilityID:     capability.ID,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("evaluate allowed: %v", err)
	}
	if !allowed.Allowed || allowed.EntitlementID != entitlement.ID || allowed.WorkspaceAssignmentID != workspaceAssignment.ID {
		t.Fatalf("unexpected allowed decision: %#v", allowed)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/store -run TestMemoryCapabilityAssignmentEvaluation -count=1
```

Expected: compile failure for missing domain and repository symbols.

- [ ] **Step 3: Add domain types**

In `internal/domain/types.go`, add these types after `RouteAccessDecision`:

```go
type CapabilityType string

const (
	CapabilityTypeMCPTool CapabilityType = "mcp_tool"
	CapabilityTypeMCPMethod CapabilityType = "mcp_method"
)

type CapabilityAction string

const (
	CapabilityActionRead CapabilityAction = "read"
	CapabilityActionWrite CapabilityAction = "write"
	CapabilityActionDelete CapabilityAction = "delete"
	CapabilityActionExecute CapabilityAction = "execute"
	CapabilityActionExport CapabilityAction = "export"
	CapabilityActionAdmin CapabilityAction = "admin"
)

type CapabilitySensitivity string

const (
	CapabilitySensitivityPublic CapabilitySensitivity = "public"
	CapabilitySensitivityInternal CapabilitySensitivity = "internal"
	CapabilitySensitivityConfidential CapabilitySensitivity = "confidential"
	CapabilitySensitivityRestricted CapabilitySensitivity = "restricted"
)

type CapabilityRisk string

const (
	CapabilityRiskLow CapabilityRisk = "low"
	CapabilityRiskMedium CapabilityRisk = "medium"
	CapabilityRiskHigh CapabilityRisk = "high"
	CapabilityRiskCritical CapabilityRisk = "critical"
)

type CapabilityEnforcementMode string

const (
	CapabilityEnforcementGateway CapabilityEnforcementMode = "gateway"
	CapabilityEnforcementContextForwarded CapabilityEnforcementMode = "context_forwarded"
	CapabilityEnforcementDownstreamNative CapabilityEnforcementMode = "downstream_native"
	CapabilityEnforcementAdvisory CapabilityEnforcementMode = "advisory"
)

type CapabilityDiscoveryStatus string

const (
	CapabilityDiscoveryPendingReview CapabilityDiscoveryStatus = "pending_review"
	CapabilityDiscoveryApproved CapabilityDiscoveryStatus = "approved"
	CapabilityDiscoveryDeprecated CapabilityDiscoveryStatus = "deprecated"
	CapabilityDiscoveryRemoved CapabilityDiscoveryStatus = "removed"
)

type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny PolicyEffect = "deny"
)

type PolicyStatus string

const (
	PolicyStatusEnabled PolicyStatus = "enabled"
	PolicyStatusDisabled PolicyStatus = "disabled"
)

type DataScope struct {
	DataDomain string `json:"dataDomain,omitempty"`
	Dataset string `json:"dataset,omitempty"`
	Schema string `json:"schema,omitempty"`
	Table string `json:"table,omitempty"`
	Field string `json:"field,omitempty"`
	Classification string `json:"classification,omitempty"`
	Region string `json:"region,omitempty"`
	TenantFilter string `json:"tenantFilter,omitempty"`
	MaskingPolicy string `json:"maskingPolicy,omitempty"`
	RowFilter string `json:"rowFilter,omitempty"`
}

type Capability struct {
	ID string `json:"id"`
	TargetID string `json:"targetId"`
	Type CapabilityType `json:"type"`
	Key string `json:"key"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
	Action CapabilityAction `json:"action"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
	NativeScopes []string `json:"nativeScopes,omitempty"`
	DataDomains []string `json:"dataDomains,omitempty"`
	DataScopes []DataScope `json:"dataScopes,omitempty"`
	Sensitivity CapabilitySensitivity `json:"sensitivity"`
	RiskLevel CapabilityRisk `json:"riskLevel"`
	EnforcementMode CapabilityEnforcementMode `json:"enforcementMode"`
	DiscoveryStatus CapabilityDiscoveryStatus `json:"discoveryStatus"`
	Version int `json:"version"`
	DiscoveredAt time.Time `json:"discoveredAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TenantEntitlement struct {
	ID string `json:"id"`
	TenantID string `json:"tenantId"`
	TargetID string `json:"targetId"`
	CapabilityID string `json:"capabilityId"`
	Effect PolicyEffect `json:"effect"`
	DataScopes []DataScope `json:"dataScopes,omitempty"`
	Status PolicyStatus `json:"status"`
	Priority int `json:"priority"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type WorkspaceAssignment struct {
	ID string `json:"id"`
	TenantEntitlementID string `json:"tenantEntitlementId"`
	TenantID string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
	Effect PolicyEffect `json:"effect"`
	DataScopes []DataScope `json:"dataScopes,omitempty"`
	Status PolicyStatus `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type InstanceAssignment struct {
	ID string `json:"id"`
	WorkspaceAssignmentID string `json:"workspaceAssignmentId"`
	TenantID string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
	CallerInstanceID string `json:"callerInstanceId"`
	SubjectSelector string `json:"subjectSelector,omitempty"`
	Effect PolicyEffect `json:"effect"`
	DataScopes []DataScope `json:"dataScopes,omitempty"`
	Status PolicyStatus `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
```

- [ ] **Step 4: Extend repository interface**

In `internal/store/memory.go`, add methods to `Repository`:

```go
UpsertCapability(context.Context, domain.Capability) (domain.Capability, error)
ListCapabilities(context.Context, CapabilityFilter) ([]domain.Capability, error)
GetCapability(context.Context, string) (domain.Capability, bool, error)
UpdateCapability(context.Context, domain.Capability) (domain.Capability, bool, error)
CreateTenantEntitlement(context.Context, domain.TenantEntitlement) (domain.TenantEntitlement, error)
ListTenantEntitlements(context.Context, EntitlementFilter) ([]domain.TenantEntitlement, error)
CreateWorkspaceAssignment(context.Context, domain.WorkspaceAssignment) (domain.WorkspaceAssignment, error)
ListWorkspaceAssignments(context.Context, AssignmentFilter) ([]domain.WorkspaceAssignment, error)
CreateInstanceAssignment(context.Context, domain.InstanceAssignment) (domain.InstanceAssignment, error)
ListInstanceAssignments(context.Context, InstanceAssignmentFilter) ([]domain.InstanceAssignment, error)
EvaluateCapabilityAccess(context.Context, CapabilityAccessRequest) (domain.CapabilityAccessDecision, error)
```

Add filter/request types near existing filters:

```go
type CapabilityFilter struct {
	ManagementScope
	TargetID string
	Status domain.CapabilityDiscoveryStatus
}

type EntitlementFilter struct {
	ManagementScope
	TargetID string
	CapabilityID string
}

type AssignmentFilter struct {
	ManagementScope
	EntitlementID string
}

type InstanceAssignmentFilter struct {
	ManagementScope
	CallerInstanceID string
	CapabilityID string
}

type CapabilityAccessRequest struct {
	TenantID string
	WorkspaceID string
	CallerInstanceID string
	SubjectID string
	TargetID string
	CapabilityID string
	Now time.Time
}
```

Add `CapabilityAccessDecision` to `internal/domain/types.go`:

```go
type CapabilityAccessDecision struct {
	Allowed bool `json:"allowed"`
	Source string `json:"source"`
	CapabilityID string `json:"capabilityId,omitempty"`
	EntitlementID string `json:"entitlementId,omitempty"`
	WorkspaceAssignmentID string `json:"workspaceAssignmentId,omitempty"`
	InstanceAssignmentID string `json:"instanceAssignmentId,omitempty"`
	Reason string `json:"reason"`
	DataScopes []DataScope `json:"dataScopes,omitempty"`
}
```

- [ ] **Step 5: Implement memory storage and evaluation**

Add maps to `Memory`:

```go
capabilities map[string]domain.Capability
entitlements map[string]domain.TenantEntitlement
workspaceAssignments map[string]domain.WorkspaceAssignment
instanceAssignments map[string]domain.InstanceAssignment
```

Initialize them in `NewMemory`.

Implement evaluation with strict narrowing:

```go
func (m *Memory) EvaluateCapabilityAccess(_ context.Context, req CapabilityAccessRequest) (domain.CapabilityAccessDecision, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	capability, ok := m.capabilities[req.CapabilityID]
	if !ok || capability.TargetID != req.TargetID {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "capability", Reason: "capability is not registered for target"}, nil
	}
	if capability.DiscoveryStatus != domain.CapabilityDiscoveryApproved {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "capability", CapabilityID: capability.ID, Reason: "capability is not approved"}, nil
	}

	entitlement, ok := m.matchTenantEntitlementLocked(req, capability.ID)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "tenant_entitlement", CapabilityID: capability.ID, Reason: "tenant has no entitlement for capability"}, nil
	}
	if entitlement.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "tenant_entitlement", CapabilityID: capability.ID, EntitlementID: entitlement.ID, Reason: "tenant entitlement denies capability"}, nil
	}

	workspaceAssignment, ok := m.matchWorkspaceAssignmentLocked(req, entitlement.ID)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "workspace_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, Reason: "workspace has no assignment for capability"}, nil
	}
	if workspaceAssignment.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "workspace_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, Reason: "workspace assignment denies capability"}, nil
	}

	instanceAssignment, ok := m.matchInstanceAssignmentLocked(req, workspaceAssignment.ID)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "instance_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, Reason: "caller instance has no assignment for capability"}, nil
	}
	if instanceAssignment.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "instance_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, InstanceAssignmentID: instanceAssignment.ID, Reason: "caller instance assignment denies capability"}, nil
	}

	return domain.CapabilityAccessDecision{
		Allowed: true,
		Source: "capability_governance",
		CapabilityID: capability.ID,
		EntitlementID: entitlement.ID,
		WorkspaceAssignmentID: workspaceAssignment.ID,
		InstanceAssignmentID: instanceAssignment.ID,
		Reason: "capability assignment matched",
		DataScopes: instanceAssignment.DataScopes,
	}, nil
}
```

The helper matchers must require matching `tenantId`, `workspaceId`, `callerInstanceId`, enabled status, and the correct parent id.

- [ ] **Step 6: Run store test**

Run:

```bash
go test ./internal/store -run TestMemoryCapabilityAssignmentEvaluation -count=1
```

Expected: pass.

- [ ] **Step 7: Run full Go tests**

Run:

```bash
go test ./...
```

Expected: fail until PostgreSQL implements the new repository interface. That failure confirms Task 2 is required.

## Task 2: PostgreSQL Migration And Repository

**Files:**
- Add: `internal/db/migrations/007_capability_governance.sql`
- Modify: `internal/store/postgres.go`
- Modify: `internal/store/postgres_test.go`

- [ ] **Step 1: Write failing PostgreSQL test**

In `internal/store/postgres_test.go`, add a test in the existing PostgreSQL integration style:

```go
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
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	caller := domain.Agent{ID: "agt_pg_cap_caller", TenantID: "tenant-pg", WorkspaceID: "ws-pg-cap", Name: "PG Caller", ChannelType: "local", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	target := domain.Agent{ID: "agt_pg_cap_target", TenantID: "tenant-pg", WorkspaceID: "ws-pg-cap", Name: "PG MCP", ChannelType: "mcp", Status: domain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.CreateAgent(ctx, caller); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	if _, err := repo.CreateAgent(ctx, target); err != nil {
		t.Fatalf("create target: %v", err)
	}
	capability := domain.Capability{ID: "cap_pg_search", TargetID: target.ID, Type: domain.CapabilityTypeMCPTool, Key: "search_customer", DisplayName: "search_customer", Action: domain.CapabilityActionRead, Sensitivity: domain.CapabilitySensitivityInternal, RiskLevel: domain.CapabilityRiskLow, EnforcementMode: domain.CapabilityEnforcementGateway, DiscoveryStatus: domain.CapabilityDiscoveryApproved, Version: 1, DiscoveredAt: now, UpdatedAt: now}
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
}
```

- [ ] **Step 2: Run test to verify it fails**

Run with a configured PostgreSQL test database:

```bash
AGENT_HARBOR_TEST_DATABASE_URL='postgres://agent_harbor:agent_harbor@127.0.0.1:5432/agent_harbor?sslmode=disable' go test ./internal/store -run TestPostgresCapabilityGovernanceRoundTrip -count=1
```

Expected: compile or SQL failure until migration and repository code are implemented. If the database URL is unavailable locally, record that and run `go test ./...` after repository compile errors are fixed.

- [ ] **Step 3: Add migration**

Create `internal/db/migrations/007_capability_governance.sql`:

```sql
create table if not exists capabilities (
	id text primary key,
	target_agent_id text not null references agents(id) on delete cascade,
	type text not null,
	key text not null,
	display_name text not null,
	description text not null default '',
	action text not null,
	input_schema jsonb not null default '{}'::jsonb,
	output_schema jsonb not null default '{}'::jsonb,
	native_scopes jsonb not null default '[]'::jsonb,
	data_domains jsonb not null default '[]'::jsonb,
	data_scopes jsonb not null default '[]'::jsonb,
	sensitivity text not null,
	risk_level text not null,
	enforcement_mode text not null,
	discovery_status text not null,
	version integer not null,
	discovered_at timestamptz not null,
	updated_at timestamptz not null,
	unique(target_agent_id, type, key)
);

create index if not exists capabilities_target_idx on capabilities(target_agent_id, discovery_status, key);

create table if not exists tenant_entitlements (
	id text primary key,
	tenant_id text not null,
	target_agent_id text not null references agents(id) on delete cascade,
	capability_id text not null references capabilities(id) on delete cascade,
	effect text not null,
	data_scopes jsonb not null default '[]'::jsonb,
	status text not null,
	priority integer not null default 100,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists tenant_entitlements_scope_idx on tenant_entitlements(tenant_id, target_agent_id, capability_id, status, priority desc);

create table if not exists workspace_assignments (
	id text primary key,
	tenant_entitlement_id text not null references tenant_entitlements(id) on delete cascade,
	tenant_id text not null,
	workspace_id text not null,
	effect text not null,
	data_scopes jsonb not null default '[]'::jsonb,
	status text not null,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists workspace_assignments_scope_idx on workspace_assignments(tenant_id, workspace_id, tenant_entitlement_id, status);

create table if not exists instance_assignments (
	id text primary key,
	workspace_assignment_id text not null references workspace_assignments(id) on delete cascade,
	tenant_id text not null,
	workspace_id text not null,
	caller_instance_id text not null references agents(id) on delete cascade,
	subject_selector text not null default '',
	effect text not null,
	data_scopes jsonb not null default '[]'::jsonb,
	status text not null,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists instance_assignments_scope_idx on instance_assignments(tenant_id, workspace_id, caller_instance_id, status);

alter table trace_events
	add column if not exists tenant_id text not null default '',
	add column if not exists workspace_id text not null default '',
	add column if not exists caller_instance_id text not null default '',
	add column if not exists subject_id text not null default '',
	add column if not exists capability_id text not null default '',
	add column if not exists capability_version integer not null default 0,
	add column if not exists entitlement_id text not null default '',
	add column if not exists workspace_assignment_id text not null default '',
	add column if not exists instance_assignment_id text not null default '',
	add column if not exists data_scopes jsonb not null default '[]'::jsonb;
```

- [ ] **Step 4: Implement PostgreSQL methods**

Add CRUD methods in `internal/store/postgres.go` using the same pattern as route policies:

- marshal and unmarshal `input_schema`, `output_schema`, `native_scopes`, `data_domains`, and `data_scopes`.
- `UpsertCapability` uses `insert ... on conflict(target_agent_id, type, key) do update`.
- list methods order by `created_at, id` or `updated_at, id` consistently.
- evaluation queries in this order: capability, top tenant entitlement, matching workspace assignment, matching instance assignment.

For MVP, a capability must have `discovery_status='approved'` before it can be allowed.

- [ ] **Step 5: Run compile and store tests**

Run:

```bash
go test ./internal/store -run TestMemoryCapabilityAssignmentEvaluation -count=1
go test ./...
```

Expected: repository interface compiles. PostgreSQL-specific test may be skipped if `AGENT_HARBOR_TEST_DATABASE_URL` is unset.

## Task 3: Management APIs For Discovery And Grants

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [ ] **Step 1: Write failing HTTP test for discovery-to-assignment**

Add `TestMCPCapabilityDiscoveryAndAssignmentManagement` to `internal/httpapi/server_test.go`. It should:

1. Start an `httptest.Server` that responds to JSON-RPC `tools/list` with two tools: `search_customer` and `export_contracts`.
2. Create an MCP target agent with endpoint pointing at the test server.
3. Call `POST /api/v1/targets/{targetId}/capabilities:refresh`.
4. Assert two capabilities are returned with `pending_review`.
5. Patch `search_customer` to `approved`.
6. Create tenant entitlement, workspace assignment, and instance assignment for a caller Agent.

Use request bodies like:

```json
{
  "discoveryStatus": "approved"
}
```

```json
{
  "tenantId": "tenant-a",
  "targetId": "agt_mcp",
  "capabilityId": "cap_x",
  "effect": "allow",
  "status": "enabled"
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/httpapi -run TestMCPCapabilityDiscoveryAndAssignmentManagement -count=1
```

Expected: 404 for new endpoints.

- [ ] **Step 3: Add request types**

In `internal/domain/types.go`, add:

```go
type UpdateCapabilityRequest struct {
	DiscoveryStatus *CapabilityDiscoveryStatus `json:"discoveryStatus"`
	Sensitivity *CapabilitySensitivity `json:"sensitivity"`
	RiskLevel *CapabilityRisk `json:"riskLevel"`
	DataScopes []DataScope `json:"dataScopes"`
}

type CreateTenantEntitlementRequest struct {
	TenantID string `json:"tenantId"`
	TargetID string `json:"targetId"`
	CapabilityID string `json:"capabilityId"`
	Effect PolicyEffect `json:"effect"`
	DataScopes []DataScope `json:"dataScopes"`
	Status PolicyStatus `json:"status"`
	Priority *int `json:"priority"`
}

type CreateWorkspaceAssignmentRequest struct {
	TenantEntitlementID string `json:"tenantEntitlementId"`
	WorkspaceID string `json:"workspaceId"`
	Effect PolicyEffect `json:"effect"`
	DataScopes []DataScope `json:"dataScopes"`
	Status PolicyStatus `json:"status"`
}

type CreateInstanceAssignmentRequest struct {
	WorkspaceAssignmentID string `json:"workspaceAssignmentId"`
	CallerInstanceID string `json:"callerInstanceId"`
	SubjectSelector string `json:"subjectSelector"`
	Effect PolicyEffect `json:"effect"`
	DataScopes []DataScope `json:"dataScopes"`
	Status PolicyStatus `json:"status"`
}
```

- [ ] **Step 4: Register management routes**

In `Router()`, inside the admin group, add:

```go
r.Post("/targets/{targetId}/capabilities:refresh", s.refreshTargetCapabilities)
r.Get("/capabilities", s.listCapabilities)
r.Patch("/capabilities/{id}", s.updateCapability)
r.Post("/tenant-entitlements", s.createTenantEntitlement)
r.Get("/tenant-entitlements", s.listTenantEntitlements)
r.Post("/workspace-assignments", s.createWorkspaceAssignment)
r.Get("/workspace-assignments", s.listWorkspaceAssignments)
r.Post("/instance-assignments", s.createInstanceAssignment)
r.Get("/instance-assignments", s.listInstanceAssignments)
```

- [ ] **Step 5: Implement MCP discovery handler**

`refreshTargetCapabilities` should:

1. Load target by `targetId`.
2. Require `channelType == "mcp"`.
3. Require active or draft target with `channelConfig.endpoint`.
4. Send JSON-RPC request to endpoint:

```json
{"jsonrpc":"2.0","id":"capability-discovery","method":"tools/list","params":{}}
```

5. Parse `result.tools`.
6. Upsert one `Capability` per tool with:

```text
type: mcp_tool
key: tool.name
displayName: tool.title or tool.name
description: tool.description
action: read for names starting with search/list/get/query, export for names starting with export, execute otherwise
sensitivity: internal
riskLevel: low for read, high for export, medium otherwise
enforcementMode: gateway
discoveryStatus: pending_review for new tools
version: increment if input schema or description changes
```

Do not send plaintext credentials in the response.

- [ ] **Step 6: Implement grant and assignment handlers**

Handlers should validate:

- capability exists and belongs to target.
- entitlement tenant matches target tenant for MVP.
- workspace assignment references an existing entitlement.
- caller instance exists and has matching tenant/workspace.
- status is `enabled` or `disabled`.
- effect is `allow` or `deny`.

Use existing `security.NewID` prefixes:

```text
cap
ent
wsa
ina
```

- [ ] **Step 7: Add management audit events**

For MVP, append audit events with actions:

```text
capabilities.refreshed
capability.updated
tenant_entitlement.created
workspace_assignment.created
instance_assignment.created
```

Metadata must include ids, target id, capability key, status/effect, and data-scope keys. It must not include credentials or raw tool output payloads.

- [ ] **Step 8: Run HTTP tests**

Run:

```bash
go test ./internal/httpapi -run TestMCPCapabilityDiscoveryAndAssignmentManagement -count=1
go test ./...
```

Expected: pass except optional PostgreSQL integration when database env is unset.

## Task 4: Runtime MCP Capability Enforcement

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/domain/types.go`
- Modify: `internal/store/memory.go`
- Modify: `internal/store/postgres.go`
- Test: `internal/httpapi/server_test.go`

- [ ] **Step 1: Write failing runtime tests**

Add `TestMCPCapabilityGovernanceFiltersToolsListAndDeniesUnassignedTool`:

1. Upstream MCP server returns tools `search_customer` and `export_contracts` for `tools/list`.
2. Caller is assigned only `search_customer`.
3. `tools/list` through AgentHarbor returns only `search_customer`.
4. `tools/call` with `params.name="export_contracts"` returns `403 PERMISSION_DENIED`.
5. `tools/call` with `params.name="search_customer"` reaches upstream or returns accepted.

Request shape:

```json
{
  "jsonrpc": "2.0",
  "id": "run-1",
  "method": "tools/call",
  "params": {
    "name": "search_customer",
    "arguments": {
      "query": "Acme"
    }
  }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/httpapi -run TestMCPCapabilityGovernanceFiltersToolsListAndDeniesUnassignedTool -count=1
```

Expected: current route-policy behavior does not filter by capability.

- [ ] **Step 3: Parse MCP capability keys**

Replace `mcpRouteKeyFromRequest` with a richer parser:

```go
type mcpRequestInfo struct {
	Method string
	ToolName string
	Body []byte
}
```

Parsing rules:

- `method == "tools/list"` maps to method capability plus list filtering.
- `method == "tools/call"` requires `params.name` and maps to `mcp_tool` capability by tool name.
- Other methods keep legacy route key behavior.

- [ ] **Step 4: Resolve runtime identity**

Add:

```go
type runtimeIdentity struct {
	PlatformID string
	TenantID string
	WorkspaceID string
	CallerInstanceID string
	SubjectID string
}
```

For MVP:

- `PlatformID` is `"default"`.
- `TenantID` and `WorkspaceID` come from the authenticated caller Agent.
- `CallerInstanceID` is the caller Agent id.
- `SubjectID` comes from `X-AgentHarbor-Subject-Id` and defaults to empty.

- [ ] **Step 5: Evaluate capability before legacy route fallback**

In MCP data-plane handling:

- For `tools/call`, find a capability where `targetId`, `type=mcp_tool`, and `key=params.name`.
- If no capability exists, record denied trace with reason `capability is not registered for target`.
- If capability exists, call `EvaluateCapabilityAccess`.
- If denied, record denied trace and return `PERMISSION_DENIED`.
- If allowed, proxy upstream or accepted response.
- Only fall back to existing `EvaluateRouteAccess` for MCP methods that are not capability-aware in MVP.

- [ ] **Step 6: Filter `tools/list`**

For `tools/list`:

1. Proxy to upstream as today.
2. Parse JSON response.
3. Remove tools where the caller lacks an allowed capability decision.
4. Return the filtered JSON-RPC response.
5. Record allowed trace with reason `filtered tools/list by capability assignments`.

If upstream is unavailable, keep existing upstream error behavior and record trace with upstream error code.

- [ ] **Step 7: Run runtime tests**

Run:

```bash
go test ./internal/httpapi -run 'TestMCPCapabilityGovernance|TestRoutePolicy|TestProxy' -count=1
go test ./...
```

Expected: pass with legacy route policy and proxy tests unchanged.

## Task 5: Trace Enrichment And Audit Evidence

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/store/memory.go`
- Modify: `internal/store/postgres.go`
- Test: `internal/httpapi/server_test.go`

- [ ] **Step 1: Write failing trace evidence test**

Add `TestMCPCapabilityTraceIncludesScopeAndPolicyEvidence`:

1. Assign `search_customer`.
2. Call `tools/call` with `X-Run-Id: cap-run-1`.
3. Fetch `/api/v1/audit/traces?runId=cap-run-1`.
4. Assert trace contains:

```text
tenantId = tenant-a
workspaceId = ws-sales
callerInstanceId = caller id
targetAgentId = target id
capabilityId = discovered capability id
capabilityVersion = 1
entitlementId = entitlement id
workspaceAssignmentId = workspace assignment id
instanceAssignmentId = instance assignment id
decision = allowed
reason = capability assignment matched
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/httpapi -run TestMCPCapabilityTraceIncludesScopeAndPolicyEvidence -count=1
```

Expected: trace response lacks new fields.

- [ ] **Step 3: Extend trace domain and persistence**

Add fields to `TraceEvent`:

```go
TenantID string `json:"tenantId,omitempty"`
WorkspaceID string `json:"workspaceId,omitempty"`
CallerInstanceID string `json:"callerInstanceId,omitempty"`
SubjectID string `json:"subjectId,omitempty"`
CapabilityID string `json:"capabilityId,omitempty"`
CapabilityVersion int `json:"capabilityVersion,omitempty"`
EntitlementID string `json:"entitlementId,omitempty"`
WorkspaceAssignmentID string `json:"workspaceAssignmentId,omitempty"`
InstanceAssignmentID string `json:"instanceAssignmentId,omitempty"`
DataScopes []DataScope `json:"dataScopes,omitempty"`
```

Update memory append/list, PostgreSQL insert/select, and scan helpers.

- [ ] **Step 4: Pass evidence through data-plane recording**

Replace the current `recordDataPlaneTrace` argument list with a small input struct:

```go
type traceRecordInput struct {
	Identity runtimeIdentity
	CallerID string
	TargetID string
	RouteType string
	RouteKey string
	Decision domain.TraceDecision
	Reason string
	Capability domain.Capability
	CapabilityDecision domain.CapabilityAccessDecision
	ProxyResult proxyTraceResult
}
```

Legacy callers can pass empty capability fields.

- [ ] **Step 5: Run trace tests**

Run:

```bash
go test ./internal/httpapi -run 'TestMCPCapabilityTrace|TestTrace|TestRuntimeMetrics' -count=1
go test ./...
```

Expected: pass.

## Task 6: Frontend Capability Governance Path

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/data.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/styles.css`
- Test: `frontend/tests/retryForm.test.mjs`

- [ ] **Step 1: Run existing frontend checks**

Run:

```bash
cd frontend && pnpm test && pnpm build
```

Expected: current tests and build pass before edits.

- [ ] **Step 2: Add frontend types and API calls**

Add TypeScript interfaces matching the Go JSON shapes:

```ts
export interface DataScope {
  dataDomain?: string
  dataset?: string
  schema?: string
  table?: string
  field?: string
  classification?: string
  region?: string
  tenantFilter?: string
  maskingPolicy?: string
  rowFilter?: string
}

export interface Capability {
  id: string
  targetId: string
  type: 'mcp_tool' | 'mcp_method'
  key: string
  displayName: string
  description?: string
  action: 'read' | 'write' | 'delete' | 'execute' | 'export' | 'admin'
  inputSchema?: JsonObject
  outputSchema?: JsonObject
  nativeScopes?: string[]
  dataDomains?: string[]
  dataScopes?: DataScope[]
  sensitivity: 'public' | 'internal' | 'confidential' | 'restricted'
  riskLevel: 'low' | 'medium' | 'high' | 'critical'
  enforcementMode: 'gateway' | 'context_forwarded' | 'downstream_native' | 'advisory'
  discoveryStatus: 'pending_review' | 'approved' | 'deprecated' | 'removed'
  version: number
  discoveredAt: string
  updatedAt: string
}
```

Add `refreshTargetCapabilities`, `fetchCapabilities`, `updateCapability`, `createTenantEntitlement`, `createWorkspaceAssignment`, and `createInstanceAssignment` in `frontend/src/api.ts`.

- [ ] **Step 3: Add sample data**

Add two sample capabilities:

- `search_customer`: approved, read, low risk.
- `export_contracts`: pending review, export, high risk.

Add one entitlement/assignment chain for `search_customer`.

- [ ] **Step 4: Add minimal UI path**

In `App.tsx`:

- Add nav item `Capabilities`.
- Add a `Capability Governance` panel showing target, capability key, status, action, risk, and assignment state.
- Add a refresh button that calls `refreshTargetCapabilities` for the selected MCP target.
- Add approve action for pending capabilities.
- Add a compact form to grant selected capability to tenant/workspace/caller instance.
- Add trace panel columns for capability id and assignment ids.

Keep the UI dense and operational; do not add marketing text or a landing page.

- [ ] **Step 5: Run frontend checks**

Run:

```bash
cd frontend && pnpm test && pnpm build
```

Expected: pass.

## Task 7: Demo Script And Documentation

**Files:**
- Add: `scripts/demo-sprint12-mcp-capability-governance.sh`
- Modify: `scripts/demo-all.sh`
- Modify: `README.md`

- [ ] **Step 1: Write demo script**

Create `scripts/demo-sprint12-mcp-capability-governance.sh` that:

1. Creates a local caller Agent.
2. Starts or expects a test MCP endpoint URL from `MCP_ENDPOINT`.
3. Creates an MCP target Agent.
4. Refreshes capabilities.
5. Approves `search_customer`.
6. Creates tenant entitlement, workspace assignment, and instance assignment.
7. Creates an Agent Key.
8. Calls `tools/list` and verifies `search_customer` is present.
9. Calls `tools/call` for `export_contracts` and verifies `PERMISSION_DENIED`.
10. Calls `tools/call` for `search_customer` and verifies allowed.
11. Fetches trace evidence and verifies `capabilityId` is present.

- [ ] **Step 2: Add script lint expectation**

Run:

```bash
make demo-scripts-lint
```

Expected: pass.

- [ ] **Step 3: Update README**

Add a section:

```markdown
## Sprint 12 MCP Capability Governance Demo

Sprint 12 introduces tenant-scoped MCP capability governance. It discovers MCP tools, keeps new tools denied until approval, grants an approved tool to a tenant/workspace/caller instance chain, filters `tools/list`, denies unassigned `tools/call`, and records capability evidence in traces.
```

Add the new API endpoints to the Current API list.

- [ ] **Step 4: Run full verification**

Run:

```bash
make check
```

Expected: Go tests, vet, build, frontend tests/build, and demo script lint pass.

## Task 8: Final Review And Compatibility Check

**Files:**
- Review all touched files.

- [ ] **Step 1: Verify legacy demos remain compatible**

Run against a local API if feasible:

```bash
make run
```

In another shell:

```bash
bash scripts/demo-governance-loop.sh
bash scripts/demo-sprint9-route-policies.sh
bash scripts/demo-sprint10-route-policy-retry.sh
```

Expected: existing route-policy and retry demos continue to pass.

- [ ] **Step 2: Verify new MVP demo**

Run:

```bash
bash scripts/demo-sprint12-mcp-capability-governance.sh
```

Expected: discovery, approval, entitlement, assignment, filtered list, denied unassigned call, allowed assigned call, and trace evidence checks all pass.

- [ ] **Step 3: Final code review**

Review for these issues:

- capability grants do not bypass tenant/workspace matching.
- new tools remain denied until approved.
- data scopes are copied or cloned when stored to avoid mutation leaks.
- traces do not include credentials or raw secret-bearing payloads.
- existing `RoutePolicy` and `AccessGrant` fallback remains available for non-capability-aware routes.
- frontend mock fallback remains navigable when backend is down.

- [ ] **Step 4: Commit**

Commit after all verification passes:

```bash
git add internal frontend scripts README.md docs/superpowers/specs/2026-06-01-tenant-scoped-capability-governance-design.md docs/superpowers/plans/2026-06-01-mcp-capability-governance-mvp.md
git commit -m "feat: add mcp capability governance"
```
