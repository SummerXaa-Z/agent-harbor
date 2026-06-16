# Admin Tenant Access Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add production-safe in-product management for administrator identities and their tenant/workspace boundaries.

**Architecture:** Persist managed administrator identities separately from bootstrap environment identities, authenticate by hashed one-time administrator keys, and route all lifecycle operations through a small admin-identity service shared by REST and management MCP. The frontend adds a focused Administrators & Boundaries workspace with modal create/rotate/disable flows, while existing tenant/workspace scope enforcement continues to protect resource operations after login.

**Tech Stack:** Go HTTP API, in-memory repository, PostgreSQL migrations, management MCP JSON-RPC tools, React/TypeScript console, pnpm/Vitest-style source tests, shell release scenarios.

---

## File Map

- Create: `internal/httpapi/admin_identities.go` — REST request/response types, lifecycle handlers, role guards, one-time key generation, audit metadata.
- Modify: `internal/domain/types.go` — persisted administrator identity model and create/rotate response types.
- Modify: `internal/store/memory.go` — repository interface plus memory implementation for managed admin identity CRUD-like lifecycle.
- Modify: `internal/store/postgres.go` — PostgreSQL implementation and scan helpers for managed admin identities.
- Create: `internal/db/migrations/013_admin_identities.sql` — persistent `admin_identities` table and indexes.
- Modify: `internal/httpapi/server.go` — route registration and authentication lookup delegation to managed identities.
- Modify: `internal/httpapi/auth.go` — session actor re-resolution recognizes active managed identities and updates `lastUsedAt`.
- Modify: `internal/httpapi/management_mcp.go` — AI-friendly administrator lifecycle tools.
- Modify: `internal/httpapi/server_test.go` — REST lifecycle, auth, session invalidation, scope, MCP, and secret-redaction tests.
- Modify: `internal/store/memory_test.go` — memory repository lifecycle and audit tests.
- Modify: `internal/store/postgres_test.go` — PostgreSQL lifecycle, uniqueness, and audit tests.
- Modify: `frontend/src/types.ts` — admin identity types and request/response payloads.
- Modify: `frontend/src/api.ts` — admin identity REST client functions.
- Modify: `frontend/src/consoleNavigation.ts` — new `admin-access` configuration navigation item.
- Create: `frontend/src/adminAccess.ts` — summary and presenter helpers for administrator identity rows.
- Create: `frontend/src/hooks/useAdminAccessController.ts` — frontend controller state and mutation handlers.
- Create: `frontend/src/components/AdminAccessManagementView.tsx` — production console page and modal forms.
- Modify: `frontend/src/components/ConsoleViews.tsx` — render the new workspace.
- Modify: `frontend/src/ConsoleController.tsx` — wire API, hook, panel, and current-session link.
- Modify: `frontend/src/i18n.ts` — bilingual copy.
- Modify: `frontend/src/styles.css` — modal/action/list styling using existing tokens.
- Modify: `frontend/tests/*.mjs` — source tests for navigation, i18n, modal flow, and no secret rendering in list path.
- Create: `scripts/scenario-admin-access-management.sh` — release scenario covering managed admin create/login/rotate/disable/guards.
- Modify: `Makefile`, `tests/makefile_targets_test.sh`, `README.md`, `CHANGELOG.md` — release wiring and docs.

---

### Task 1: Red Backend Tests For Managed Administrator Lifecycle

**Files:**
- Modify: `internal/httpapi/server_test.go`

- [x] **Step 1: Add REST lifecycle test skeleton**

Add these response structs next to existing test response structs:

```go
type adminIdentityResponse struct {
	ID          string         `json:"id"`
	Actor       string         `json:"actor"`
	DisplayName string         `json:"displayName"`
	Role        string         `json:"role"`
	TenantID    string         `json:"tenantId"`
	WorkspaceID string         `json:"workspaceId"`
	Status      string         `json:"status"`
	Source      string         `json:"source"`
	KeyPrefix   string         `json:"keyPrefix"`
	CreatedBy   string         `json:"createdBy"`
	UpdatedBy   string         `json:"updatedBy"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type createAdminIdentityResponse struct {
	Identity adminIdentityResponse `json:"identity"`
	Key      string                `json:"key"`
}

type rotateAdminIdentityKeyResponse struct {
	Identity adminIdentityResponse `json:"identity"`
	Key      string                `json:"key"`
}
```

- [x] **Step 2: Add platform lifecycle test**

Add `TestManagedAdminIdentityLifecycleAndScopedLogin` after scoped admin tests:

```go
func TestManagedAdminIdentityLifecycleAndScopedLogin(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	create := decodeData[createAdminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":       "east-admin",
		"displayName": "East Administrator",
		"role":        "tenant_admin",
		"tenantId":    "tenant-east",
		"workspaceId": "ws-support",
	}, "", "platform-key"))
	if create.Identity.Actor != "east-admin" || create.Identity.Source != "managed" || create.Identity.Status != "active" {
		t.Fatalf("unexpected created managed admin: %#v", create)
	}
	if create.Key == "" || strings.Contains(create.Key, create.Identity.KeyPrefix) == false {
		t.Fatalf("expected one-time key with visible prefix, got response=%#v", create)
	}

	list := decodeData[[]adminIdentityResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/admin-identities", nil, "", "platform-key"))
	if len(list) != 2 {
		t.Fatalf("expected bootstrap plus managed identity, got %#v", list)
	}
	for _, row := range list {
		if bytes.Contains(mustJSON(t, row), []byte(create.Key)) {
			t.Fatalf("list response must not expose one-time admin key: %#v", row)
		}
	}

	login := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": create.Key}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("managed admin key should log in, got %d body=%s", login.Code, login.Body.String())
	}
	session := decodeData[map[string]any](t, login)
	if session["actor"] != "east-admin" || session["role"] != "tenant_admin" || session["tenantId"] != "tenant-east" || session["workspaceId"] != "ws-support" {
		t.Fatalf("unexpected managed session: %#v", session)
	}

	rotate := decodeData[rotateAdminIdentityKeyResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+create.Identity.ID+"/key:rotate", nil, "", "platform-key"))
	if rotate.Key == "" || rotate.Key == create.Key || rotate.Identity.KeyPrefix == create.Identity.KeyPrefix {
		t.Fatalf("expected rotated key and prefix, got %#v", rotate)
	}
	oldLogin := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": create.Key}, "")
	if oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old key must be invalid after rotation, got %d body=%s", oldLogin.Code, oldLogin.Body.String())
	}
	newLogin := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": rotate.Key}, "")
	if newLogin.Code != http.StatusOK {
		t.Fatalf("rotated key should log in, got %d body=%s", newLogin.Code, newLogin.Body.String())
	}

	disabled := decodeData[adminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+create.Identity.ID+":disable", nil, "", "platform-key"))
	if disabled.Status != "disabled" {
		t.Fatalf("expected disabled managed admin, got %#v", disabled)
	}
	disabledLogin := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": rotate.Key}, "")
	if disabledLogin.Code != http.StatusUnauthorized {
		t.Fatalf("disabled admin key must be invalid, got %d body=%s", disabledLogin.Code, disabledLogin.Body.String())
	}

	events := decodeData[[]auditEventResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/audit/events?resourceType=admin_identity", nil, "", "platform-key"))
	if got := auditActions(events); !reflect.DeepEqual(got, []string{"admin_identity.created", "admin_identity.key_rotated", "admin_identity.disabled"}) {
		t.Fatalf("unexpected admin identity audit actions: %#v", got)
	}
	for _, event := range events {
		raw := mustJSON(t, event)
		if bytes.Contains(raw, []byte(create.Key)) || bytes.Contains(raw, []byte(rotate.Key)) || bytes.Contains(raw, []byte("keyHash")) {
			t.Fatalf("audit event must not expose admin key material: %s", raw)
		}
	}
}
```

- [x] **Step 3: Add forbidden scoped-admin management test**

Add:

```go
func TestScopedAdminCannotManageAdminIdentities(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east", WorkspaceID: "ws-support"},
	})

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{name: "list", method: http.MethodGet, path: "/api/v1/admin-identities"},
		{name: "create", method: http.MethodPost, path: "/api/v1/admin-identities", body: map[string]any{"actor": "bad", "role": "tenant_admin", "tenantId": "tenant-east"}},
		{name: "rotate", method: http.MethodPost, path: "/api/v1/admin-identities/adm_missing/key:rotate"},
		{name: "disable", method: http.MethodPost, path: "/api/v1/admin-identities/adm_missing:disable"},
	} {
		resp := requestWithAdmin(t, router, tc.method, tc.path, tc.body, "", "east-key")
		if resp.Code != http.StatusForbidden {
			t.Fatalf("%s should be forbidden for scoped admin, got %d body=%s", tc.name, resp.Code, resp.Body.String())
		}
	}
}
```

- [x] **Step 4: Add guard tests for bootstrap and last platform admin**

Add:

```go
func TestAdminIdentityLifecycleRejectsBootstrapAndLastPlatformMutation(t *testing.T) {
	repo := store.NewMemory()
	router := newRouterWithRepoAndAdminIdentities(repo, []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
	})

	list := decodeData[[]adminIdentityResponse](t, requestWithAdmin(t, router, http.MethodGet, "/api/v1/admin-identities", nil, "", "platform-key"))
	if len(list) != 1 || list[0].Source != "bootstrap" {
		t.Fatalf("expected one bootstrap identity, got %#v", list)
	}
	rotateBootstrap := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+list[0].ID+"/key:rotate", nil, "", "platform-key")
	if rotateBootstrap.Code != http.StatusBadRequest {
		t.Fatalf("bootstrap identity rotation should be rejected, got %d body=%s", rotateBootstrap.Code, rotateBootstrap.Body.String())
	}

	created := decodeData[createAdminIdentityResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities", map[string]any{
		"actor":       "managed-platform",
		"displayName": "Managed Platform",
		"role":        "platform_admin",
	}, "", "platform-key"))
	disableManagedPlatform := requestWithAdmin(t, router, http.MethodPost, "/api/v1/admin-identities/"+created.Identity.ID+":disable", nil, "", created.Key)
	if disableManagedPlatform.Code != http.StatusForbidden {
		t.Fatalf("self-disable should be rejected, got %d body=%s", disableManagedPlatform.Code, disableManagedPlatform.Body.String())
	}
}
```

- [x] **Step 5: Add helper for JSON secret scans**

Add near test helpers:

```go
func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test value: %v", err)
	}
	return raw
}
```

- [x] **Step 6: Run red tests**

Run:

```bash
go test ./internal/httpapi -run 'TestManagedAdminIdentityLifecycleAndScopedLogin|TestScopedAdminCannotManageAdminIdentities|TestAdminIdentityLifecycleRejectsBootstrapAndLastPlatformMutation' -count=1
```

Expected: FAIL because `/api/v1/admin-identities` routes and managed identity store methods do not exist.

- [x] **Step 7: Commit red tests**

```bash
git add internal/httpapi/server_test.go
git commit -m "test: cover managed admin identity lifecycle"
```

---

### Task 2: Domain Model, Store Interface, Memory Store, And PostgreSQL Store

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/store/memory.go`
- Modify: `internal/store/memory_test.go`
- Modify: `internal/store/postgres.go`
- Modify: `internal/store/postgres_test.go`
- Create: `internal/db/migrations/013_admin_identities.sql`

- [x] **Step 1: Add domain types**

Append after `AuditEvent` in `internal/domain/types.go`:

```go
type AdminIdentityRole string

const (
	AdminIdentityRolePlatformAdmin    AdminIdentityRole = "platform_admin"
	AdminIdentityRoleTenantAdmin      AdminIdentityRole = "tenant_admin"
	AdminIdentityRoleSecurityReviewer AdminIdentityRole = "security_reviewer"
)

type AdminIdentityStatus string

const (
	AdminIdentityStatusActive   AdminIdentityStatus = "active"
	AdminIdentityStatusDisabled AdminIdentityStatus = "disabled"
)

type AdminIdentitySource string

const (
	AdminIdentitySourceBootstrap AdminIdentitySource = "bootstrap"
	AdminIdentitySourceManaged   AdminIdentitySource = "managed"
)

type AdminIdentity struct {
	ID          string              `json:"id"`
	Actor       string              `json:"actor"`
	DisplayName string              `json:"displayName"`
	Role        AdminIdentityRole   `json:"role"`
	TenantID    string              `json:"tenantId,omitempty"`
	WorkspaceID string              `json:"workspaceId,omitempty"`
	Status      AdminIdentityStatus `json:"status"`
	Source      AdminIdentitySource `json:"source"`
	KeyHash     string              `json:"-"`
	KeyPrefix   string              `json:"keyPrefix,omitempty"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	LastUsedAt  time.Time           `json:"lastUsedAt,omitempty,omitzero"`
	RotatedAt   time.Time           `json:"rotatedAt,omitempty,omitzero"`
	DisabledAt  time.Time           `json:"disabledAt,omitempty,omitzero"`
	CreatedBy   string              `json:"createdBy,omitempty"`
	UpdatedBy   string              `json:"updatedBy,omitempty"`
	DisabledBy  string              `json:"disabledBy,omitempty"`
}

type CreateAdminIdentityRequest struct {
	Actor       string            `json:"actor"`
	DisplayName string            `json:"displayName"`
	Role        AdminIdentityRole `json:"role"`
	TenantID    string            `json:"tenantId"`
	WorkspaceID string            `json:"workspaceId"`
}

type CreateAdminIdentityResponse struct {
	Identity AdminIdentity `json:"identity"`
	Key      string        `json:"key"`
}

type RotateAdminIdentityKeyResponse struct {
	Identity AdminIdentity `json:"identity"`
	Key      string        `json:"key"`
}
```

- [x] **Step 2: Extend repository interface**

Add these methods to `store.Repository` in `internal/store/memory.go` after audit methods:

```go
	ListAdminIdentities(context.Context) ([]domain.AdminIdentity, error)
	GetAdminIdentity(context.Context, string) (domain.AdminIdentity, bool, error)
	GetAdminIdentityByActor(context.Context, string) (domain.AdminIdentity, bool, error)
	FindAdminIdentityByKeyHash(context.Context, string) (domain.AdminIdentity, bool, error)
	CreateAdminIdentityWithAudit(context.Context, domain.AdminIdentity, AdminIdentityAuditBuilder) (domain.AdminIdentity, error)
	RotateAdminIdentityKeyWithAudit(context.Context, string, string, string, time.Time, string, AdminIdentityAuditBuilder) (domain.AdminIdentity, bool, error)
	DisableAdminIdentityWithAudit(context.Context, string, time.Time, string, AdminIdentityAuditBuilder) (domain.AdminIdentity, bool, error)
	TouchAdminIdentityLastUsed(context.Context, string, time.Time) error
```

Add the builder:

```go
type AdminIdentityAuditBuilder func(domain.AdminIdentity) domain.AuditEvent
```

- [x] **Step 3: Add memory fields**

Add to `Memory`:

```go
	adminIdentities      map[string]domain.AdminIdentity
	adminIdentityActorID map[string]string
```

Initialize in `NewMemory()`:

```go
adminIdentities:      make(map[string]domain.AdminIdentity),
adminIdentityActorID: make(map[string]string),
```

- [x] **Step 4: Implement memory methods**

Add after audit methods:

```go
func (m *Memory) ListAdminIdentities(_ context.Context) ([]domain.AdminIdentity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rows := make([]domain.AdminIdentity, 0, len(m.adminIdentities))
	for _, identity := range m.adminIdentities {
		rows = append(rows, identity)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	return rows, nil
}

func (m *Memory) GetAdminIdentity(_ context.Context, id string) (domain.AdminIdentity, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	identity, ok := m.adminIdentities[strings.TrimSpace(id)]
	return identity, ok, nil
}

func (m *Memory) GetAdminIdentityByActor(_ context.Context, actor string) (domain.AdminIdentity, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.adminIdentityActorID[strings.TrimSpace(actor)]
	if !ok {
		return domain.AdminIdentity{}, false, nil
	}
	identity, ok := m.adminIdentities[id]
	return identity, ok, nil
}

func (m *Memory) FindAdminIdentityByKeyHash(_ context.Context, hash string) (domain.AdminIdentity, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hash = strings.TrimSpace(hash)
	for _, identity := range m.adminIdentities {
		if identity.Status == domain.AdminIdentityStatusActive && identity.KeyHash == hash {
			return identity, true, nil
		}
	}
	return domain.AdminIdentity{}, false, nil
}

func (m *Memory) CreateAdminIdentityWithAudit(_ context.Context, identity domain.AdminIdentity, build AdminIdentityAuditBuilder) (domain.AdminIdentity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.adminIdentities[identity.ID]; exists {
		return domain.AdminIdentity{}, domain.Conflict("admin identity already exists")
	}
	if _, exists := m.adminIdentityActorID[identity.Actor]; exists {
		return domain.AdminIdentity{}, domain.Conflict("admin identity actor already exists")
	}
	m.adminIdentities[identity.ID] = identity
	m.adminIdentityActorID[identity.Actor] = identity.ID
	m.audits = append(m.audits, build(identity))
	return identity, nil
}

func (m *Memory) RotateAdminIdentityKeyWithAudit(_ context.Context, id string, keyHash string, keyPrefix string, now time.Time, actor string, build AdminIdentityAuditBuilder) (domain.AdminIdentity, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	identity, ok := m.adminIdentities[strings.TrimSpace(id)]
	if !ok {
		return domain.AdminIdentity{}, false, nil
	}
	identity.KeyHash = strings.TrimSpace(keyHash)
	identity.KeyPrefix = strings.TrimSpace(keyPrefix)
	identity.RotatedAt = now
	identity.UpdatedAt = now
	identity.UpdatedBy = strings.TrimSpace(actor)
	m.adminIdentities[identity.ID] = identity
	m.audits = append(m.audits, build(identity))
	return identity, true, nil
}

func (m *Memory) DisableAdminIdentityWithAudit(_ context.Context, id string, now time.Time, actor string, build AdminIdentityAuditBuilder) (domain.AdminIdentity, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	identity, ok := m.adminIdentities[strings.TrimSpace(id)]
	if !ok {
		return domain.AdminIdentity{}, false, nil
	}
	identity.Status = domain.AdminIdentityStatusDisabled
	identity.DisabledAt = now
	identity.UpdatedAt = now
	identity.UpdatedBy = strings.TrimSpace(actor)
	identity.DisabledBy = strings.TrimSpace(actor)
	m.adminIdentities[identity.ID] = identity
	m.audits = append(m.audits, build(identity))
	return identity, true, nil
}

func (m *Memory) TouchAdminIdentityLastUsed(_ context.Context, id string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	identity, ok := m.adminIdentities[strings.TrimSpace(id)]
	if !ok {
		return nil
	}
	identity.LastUsedAt = now
	identity.UpdatedAt = now
	m.adminIdentities[identity.ID] = identity
	return nil
}
```

- [x] **Step 5: Add PostgreSQL migration**

Create `internal/db/migrations/013_admin_identities.sql`:

```sql
create table if not exists admin_identities (
	id text primary key,
	actor text not null unique,
	display_name text not null default '',
	role text not null,
	tenant_id text not null default '',
	workspace_id text not null default '',
	status text not null,
	source text not null,
	key_hash text not null,
	key_prefix text not null default '',
	created_at timestamptz not null,
	updated_at timestamptz not null,
	last_used_at timestamptz,
	rotated_at timestamptz,
	disabled_at timestamptz,
	created_by text not null default '',
	updated_by text not null default '',
	disabled_by text not null default ''
);

create index if not exists admin_identities_status_idx on admin_identities(status, role, created_at, id);
create index if not exists admin_identities_key_hash_idx on admin_identities(key_hash) where status = 'active';
create index if not exists admin_identities_scope_idx on admin_identities(tenant_id, workspace_id, role);
```

- [x] **Step 6: Implement PostgreSQL methods**

Add methods matching memory signatures. Use `nullTime()` for optional timestamps and `scanAdminIdentity(row scanner)`.

The `insert` statement must write every field except optional zero timestamps as SQL nulls:

```go
_, err := exec.Exec(ctx, `
	insert into admin_identities (
		id, actor, display_name, role, tenant_id, workspace_id, status, source,
		key_hash, key_prefix, created_at, updated_at, last_used_at, rotated_at,
		disabled_at, created_by, updated_by, disabled_by
	) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
`, identity.ID, identity.Actor, identity.DisplayName, identity.Role, identity.TenantID, identity.WorkspaceID,
	identity.Status, identity.Source, identity.KeyHash, identity.KeyPrefix, identity.CreatedAt, identity.UpdatedAt,
	nullTime(identity.LastUsedAt), nullTime(identity.RotatedAt), nullTime(identity.DisabledAt),
	identity.CreatedBy, identity.UpdatedBy, identity.DisabledBy)
```

- [x] **Step 7: Add focused store tests**

Add `TestMemoryAdminIdentityLifecycle` and `TestPostgresAdminIdentityLifecycle` with the same assertions:

```go
identity := domain.AdminIdentity{
	ID: "adm_test",
	Actor: "tenant-admin",
	DisplayName: "Tenant Admin",
	Role: domain.AdminIdentityRoleTenantAdmin,
	TenantID: "tenant-east",
	WorkspaceID: "ws-support",
	Status: domain.AdminIdentityStatusActive,
	Source: domain.AdminIdentitySourceManaged,
	KeyHash: security.HashSecret("admin-secret"),
	KeyPrefix: "ahadm_test",
	CreatedAt: now,
	UpdatedAt: now,
	CreatedBy: "platform",
	UpdatedBy: "platform",
}
```

Assert create, list, get by actor, find by key hash, rotate, old hash not found, touch last used, disable, disabled hash not found, and three audit events.

- [x] **Step 8: Run store tests**

```bash
go test ./internal/store -run 'TestMemoryAdminIdentityLifecycle|TestPostgresAdminIdentityLifecycle' -count=1
```

Expected: memory passes; PostgreSQL test runs only when `AGENT_HARBOR_TEST_DATABASE_URL` is configured, matching existing test conventions.

- [x] **Step 9: Commit store layer**

```bash
git add internal/domain/types.go internal/store/memory.go internal/store/memory_test.go internal/store/postgres.go internal/store/postgres_test.go internal/db/migrations/013_admin_identities.sql
git commit -m "feat: persist managed admin identities"
```

---

### Task 3: Authentication And REST Lifecycle Handlers

**Files:**
- Create: `internal/httpapi/admin_identities.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/auth.go`

- [x] **Step 1: Add admin key generator helper**

In `internal/security/key.go`, add:

```go
func NewAdminKey() (plaintext string, prefix string) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	return "ahadm_" + encoded, "ahadm_" + encoded[:8]
}
```

- [x] **Step 2: Add platform guard and bootstrap projection**

Create `internal/httpapi/admin_identities.go` with:

```go
func (s *Server) requirePlatformAdmin(r *http.Request) (adminPrincipal, error) {
	principal, ok := requestAdminPrincipal(r)
	if !ok {
		return adminPrincipal{}, domain.Unauthorized("admin authentication is required")
	}
	if principal.Role != adminRolePlatformAdmin {
		return adminPrincipal{}, domain.PermissionDenied("platform administrator is required")
	}
	return principal, nil
}

func (s *Server) bootstrapAdminIdentities() []domain.AdminIdentity {
	rows := make([]domain.AdminIdentity, 0, len(s.adminIdentities)+1)
	now := s.now()
	for _, identity := range s.adminIdentities {
		principal := identity.principal()
		rows = append(rows, domain.AdminIdentity{
			ID:          "bootstrap:" + principal.Actor,
			Actor:       principal.Actor,
			DisplayName: principal.Actor,
			Role:        domain.AdminIdentityRole(principal.Role),
			TenantID:    principal.TenantID,
			WorkspaceID: principal.WorkspaceID,
			Status:      domain.AdminIdentityStatusActive,
			Source:      domain.AdminIdentitySourceBootstrap,
			KeyPrefix:   "bootstrap",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	if s.adminKey != "" {
		rows = append(rows, domain.AdminIdentity{
			ID:          "bootstrap:admin-key",
			Actor:       "admin-key",
			DisplayName: "Bootstrap administrator",
			Role:        domain.AdminIdentityRolePlatformAdmin,
			Status:      domain.AdminIdentityStatusActive,
			Source:      domain.AdminIdentitySourceBootstrap,
			KeyPrefix:   "bootstrap",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return rows
}
```

- [x] **Step 3: Add validation helpers**

Add:

```go
func normalizeManagedAdminIdentityRequest(req domain.CreateAdminIdentityRequest) (domain.CreateAdminIdentityRequest, error) {
	req.Actor = strings.TrimSpace(req.Actor)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	if req.Actor == "" {
		return req, domain.BadRequest("VALIDATION_FAILED", "actor is required")
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Actor
	}
	switch req.Role {
	case "", domain.AdminIdentityRolePlatformAdmin:
		req.Role = domain.AdminIdentityRolePlatformAdmin
		req.TenantID = ""
		req.WorkspaceID = ""
	case domain.AdminIdentityRoleTenantAdmin, domain.AdminIdentityRoleSecurityReviewer:
		if req.TenantID == "" {
			return req, domain.BadRequest("VALIDATION_FAILED", "tenantId is required for scoped administrator roles")
		}
	default:
		return req, domain.BadRequest("VALIDATION_FAILED", "role must be platform_admin, tenant_admin, or security_reviewer")
	}
	return req, nil
}
```

- [x] **Step 4: Add REST handlers**

Add handlers:

```go
func (s *Server) listAdminIdentities(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requirePlatformAdmin(r); err != nil {
		writeError(w, err)
		return
	}
	managed, err := s.repo.ListAdminIdentities(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	rows := append(s.bootstrapAdminIdentities(), managed...)
	writeJSON(w, http.StatusOK, rows)
}
```

`createAdminIdentity` must:

1. Require platform admin.
2. Decode and normalize request.
3. Reject actor collision with `s.adminPrincipalForActor(req.Actor)` and `repo.GetAdminIdentityByActor`.
4. Generate key with `security.NewAdminKey()`.
5. Store `security.HashSecret(key)` and key prefix.
6. Return `domain.CreateAdminIdentityResponse{Identity: created, Key: plaintext}`.

`rotateAdminIdentityKey` and `disableAdminIdentity` must:

1. Require platform admin.
2. Reject ids prefixed with `bootstrap:`.
3. Load managed identity by id.
4. Reject missing with `domain.NotFound("admin identity not found")`.
5. Reject self-disable by comparing `identity.Actor` to current principal actor.
6. Call `s.ensureAnotherPlatformAdmin(ctx, identity)` before disabling or rotating a platform admin.
7. Write audit with `adminIdentityAuditMetadata(identity)`.

- [x] **Step 5: Add last-platform-admin guard**

Add:

```go
func (s *Server) activePlatformAdminCount(ctx context.Context) (int, error) {
	count := 0
	for _, identity := range s.bootstrapAdminIdentities() {
		if identity.Role == domain.AdminIdentityRolePlatformAdmin && identity.Status == domain.AdminIdentityStatusActive {
			count++
		}
	}
	managed, err := s.repo.ListAdminIdentities(ctx)
	if err != nil {
		return 0, err
	}
	for _, identity := range managed {
		if identity.Role == domain.AdminIdentityRolePlatformAdmin && identity.Status == domain.AdminIdentityStatusActive {
			count++
		}
	}
	return count, nil
}
```

When disabling an active managed platform admin, reject if count is less than or equal to 1.

- [x] **Step 6: Register REST routes**

In `Router()` inside the `requireAdmin` group:

```go
r.Get("/admin-identities", s.listAdminIdentities)
r.Post("/admin-identities", s.createAdminIdentity)
r.Post("/admin-identities/{id}/key:rotate", s.rotateAdminIdentityKey)
r.Post("/admin-identities/{id}:disable", s.disableAdminIdentity)
```

- [x] **Step 7: Extend authentication lookup**

In `adminPrincipalForKey`, before configured identities:

```go
if strings.TrimSpace(provided) != "" {
	managed, ok, err := s.repo.FindAdminIdentityByKeyHash(context.Background(), security.HashSecret(provided))
	if err == nil && ok && managed.Status == domain.AdminIdentityStatusActive {
		_ = s.repo.TouchAdminIdentityLastUsed(context.Background(), managed.ID, s.now())
		return adminPrincipalFromManagedIdentity(managed), true
	}
}
```

In `adminPrincipalForActor`, before configured identities:

```go
managed, ok, err := s.repo.GetAdminIdentityByActor(context.Background(), actor)
if err == nil && ok && managed.Status == domain.AdminIdentityStatusActive {
	return adminPrincipalFromManagedIdentity(managed), true
}
```

Use a helper:

```go
func adminPrincipalFromManagedIdentity(identity domain.AdminIdentity) adminPrincipal {
	return adminPrincipal{
		Actor:       strings.TrimSpace(identity.Actor),
		Role:        normalizeAdminRole(string(identity.Role)),
		TenantID:    strings.TrimSpace(identity.TenantID),
		WorkspaceID: strings.TrimSpace(identity.WorkspaceID),
	}
}
```

- [x] **Step 8: Run backend tests**

```bash
go test ./internal/httpapi -run 'TestManagedAdminIdentityLifecycleAndScopedLogin|TestScopedAdminCannotManageAdminIdentities|TestAdminIdentityLifecycleRejectsBootstrapAndLastPlatformMutation|TestConsoleAuthSessionReportsScopedAdminIdentity' -count=1
```

Expected: PASS.

- [x] **Step 9: Commit REST/auth**

```bash
git add internal/security/key.go internal/httpapi/admin_identities.go internal/httpapi/server.go internal/httpapi/auth.go internal/httpapi/server_test.go
git commit -m "feat: add managed admin identity API"
```

---

### Task 4: Management MCP Tools

**Files:**
- Modify: `internal/httpapi/management_mcp.go`
- Modify: `internal/httpapi/server_test.go`

- [x] **Step 1: Add MCP argument types**

Add:

```go
type managementMCPCreateAdminIdentityArgs struct {
	Actor       string `json:"actor"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	TenantID    string `json:"tenantId"`
	WorkspaceID string `json:"workspaceId"`
}

type managementMCPAdminIdentityIDArgs struct {
	ID string `json:"id"`
}
```

- [x] **Step 2: Add MCP tools/list entries**

Append to `managementMCPTools()`:

```go
{
	Name:        "list_admin_identities",
	Description: "List bootstrap and managed administrator identities, including roles, status, and tenant/workspace boundaries. Requires platform administrator.",
	InputSchema: objectSchema(map[string]any{}, []string{}),
},
{
	Name:        "create_admin_identity",
	Description: "Create a managed administrator identity and return its one-time key. Requires platform administrator.",
	InputSchema: objectSchema(map[string]any{
		"actor":       stringSchema("Unique administrator actor."),
		"displayName": stringSchema("Business-readable administrator name."),
		"role":        map[string]any{"type": "string", "enum": []string{"platform_admin", "tenant_admin", "security_reviewer"}},
		"tenantId":    stringSchema("Required for tenant_admin or security_reviewer."),
		"workspaceId": stringSchema("Optional scoped workspace."),
	}, []string{"actor", "role"}),
},
{
	Name:        "rotate_admin_identity_key",
	Description: "Rotate a managed administrator key and return the new one-time key. Requires platform administrator.",
	InputSchema: objectSchema(map[string]any{"id": stringSchema("Managed administrator identity id.")}, []string{"id"}),
},
{
	Name:        "disable_admin_identity",
	Description: "Disable a managed administrator identity. Requires platform administrator.",
	InputSchema: objectSchema(map[string]any{"id": stringSchema("Managed administrator identity id.")}, []string{"id"}),
},
```

- [x] **Step 3: Route MCP calls through REST-equivalent helpers**

In `callManagementMCPTool`, add cases that call service helpers shared with REST handlers, not copied authorization logic:

```go
case "list_admin_identities":
	rows, err := s.adminIdentityRowsForPlatform(r)
	if err != nil {
		return managementMCPCallResult{}, err
	}
	return managementMCPResult(rows), nil
case "create_admin_identity":
	args, err := decodeManagementMCPArguments[managementMCPCreateAdminIdentityArgs](req.Params.Arguments)
	if err != nil {
		return managementMCPCallResult{}, err
	}
	created, err := s.createManagedAdminIdentity(r, domain.CreateAdminIdentityRequest{
		Actor: args.Actor, DisplayName: args.DisplayName, Role: domain.AdminIdentityRole(args.Role), TenantID: args.TenantID, WorkspaceID: args.WorkspaceID,
	})
	if err != nil {
		return managementMCPCallResult{}, err
	}
	return managementMCPResult(created), nil
```

Repeat for rotate and disable using helper functions.

- [x] **Step 4: Add MCP tests**

Add `TestManagementMCPAdminIdentityTools`:

```go
func TestManagementMCPAdminIdentityTools(t *testing.T) {
	router := newRouterWithRepoAndAdminIdentities(store.NewMemory(), []httpapi.AdminIdentity{
		{Actor: "platform", Key: "platform-key", Role: "platform_admin"},
		{Actor: "east-admin", Key: "east-key", Role: "tenant_admin", TenantID: "tenant-east"},
	})
	tools := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0", "id": "tools", "method": "tools/list",
	}, "", "platform-key"))
	if !managementMCPToolNamesContain(tools, "list_admin_identities", "create_admin_identity", "rotate_admin_identity_key", "disable_admin_identity") {
		t.Fatalf("admin identity tools missing from tools/list: %#v", tools)
	}

	create := decodeMCPResult(t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id": "create-admin",
		"method": "tools/call",
		"params": map[string]any{"name": "create_admin_identity", "arguments": map[string]any{"actor": "mcp-east", "role": "tenant_admin", "tenantId": "tenant-east"}},
	}, "", "platform-key"))
	if !bytes.Contains(mustJSON(t, create), []byte("ahadm_")) {
		t.Fatalf("create_admin_identity should return one-time key: %#v", create)
	}

	forbidden := requestWithAdmin(t, router, http.MethodPost, "/api/v1/management/mcp", map[string]any{
		"jsonrpc": "2.0",
		"id": "forbidden",
		"method": "tools/call",
		"params": map[string]any{"name": "list_admin_identities", "arguments": map[string]any{}},
	}, "", "east-key")
	if forbidden.Code != http.StatusOK || !strings.Contains(forbidden.Body.String(), "platform administrator is required") {
		t.Fatalf("scoped admin MCP call should be rejected, got %d body=%s", forbidden.Code, forbidden.Body.String())
	}
}
```

- [x] **Step 5: Run MCP tests**

```bash
go test ./internal/httpapi -run 'TestManagementMCPAdminIdentityTools' -count=1
```

Expected: PASS.

- [x] **Step 6: Commit MCP tools**

```bash
git add internal/httpapi/management_mcp.go internal/httpapi/server_test.go
git commit -m "feat: expose admin identity MCP tools"
```

---

### Task 5: Frontend API, Controller Hook, Workspace, And i18n

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/consoleNavigation.ts`
- Create: `frontend/src/adminAccess.ts`
- Create: `frontend/src/hooks/useAdminAccessController.ts`
- Create: `frontend/src/components/AdminAccessManagementView.tsx`
- Modify: `frontend/src/components/ConsoleViews.tsx`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/src/styles.css`
- Modify: `frontend/tests/i18n.test.mjs`
- Modify: `frontend/tests/permissionFlowLayout.test.mjs`

- [x] **Step 1: Add frontend types**

Add to `frontend/src/types.ts`:

```ts
export type AdminIdentityRole = "platform_admin" | "tenant_admin" | "security_reviewer"
export type AdminIdentityStatus = "active" | "disabled"
export type AdminIdentitySource = "bootstrap" | "managed"

export interface AdminIdentity {
  id: string
  actor: string
  displayName: string
  role: AdminIdentityRole
  tenantId?: string
  workspaceId?: string
  status: AdminIdentityStatus
  source: AdminIdentitySource
  keyPrefix?: string
  createdAt: string
  updatedAt: string
  lastUsedAt?: string
  rotatedAt?: string
  disabledAt?: string
  createdBy?: string
  updatedBy?: string
  disabledBy?: string
}

export interface CreateAdminIdentityRequest {
  actor: string
  displayName?: string
  role: AdminIdentityRole
  tenantId?: string
  workspaceId?: string
}

export interface CreateAdminIdentityResponse {
  identity: AdminIdentity
  key: string
}

export interface RotateAdminIdentityKeyResponse {
  identity: AdminIdentity
  key: string
}
```

- [x] **Step 2: Add API client functions**

In `frontend/src/api.ts`, import the new types and add:

```ts
export async function fetchAdminIdentities(adminKey?: string, signal?: AbortSignal): Promise<AdminIdentity[]> {
  return request<AdminIdentity[]>('/api/v1/admin-identities', { adminKey, signal })
}

export async function createAdminIdentity(body: CreateAdminIdentityRequest, adminKey?: string): Promise<CreateAdminIdentityResponse> {
  return request<CreateAdminIdentityResponse>('/api/v1/admin-identities', { adminKey, body })
}

export async function rotateAdminIdentityKey(id: string, adminKey?: string): Promise<RotateAdminIdentityKeyResponse> {
  return request<RotateAdminIdentityKeyResponse>(`/api/v1/admin-identities/${encodeURIComponent(id)}/key:rotate`, { adminKey, body: {} })
}

export async function disableAdminIdentity(id: string, adminKey?: string): Promise<AdminIdentity> {
  return request<AdminIdentity>(`/api/v1/admin-identities/${encodeURIComponent(id)}:disable`, { adminKey, body: {} })
}
```

- [x] **Step 3: Add navigation item**

In `frontend/src/consoleNavigation.ts`, add nav key `"admin-access"` and item:

```ts
{ detailKey: "navDetail.adminAccess", groupKey: "configuration", key: "admin-access", label: "Administrators & Boundaries" },
```

Add view:

```ts
"admin-access": {
  key: "admin-access",
  primaryPanelKey: "adminAccess",
  titleKey: "page.adminAccess",
},
```

- [x] **Step 4: Add presenter helpers**

Create `frontend/src/adminAccess.ts`:

```ts
import type { AdminIdentity } from "./types"

export interface AdminAccessSummary {
  active: number
  bootstrap: number
  disabled: number
  scoped: number
}

export function summarizeAdminIdentities(rows: AdminIdentity[]): AdminAccessSummary {
  return rows.reduce(
    (summary, row) => {
      if (row.status === "active") summary.active += 1
      if (row.status === "disabled") summary.disabled += 1
      if (row.source === "bootstrap") summary.bootstrap += 1
      if (row.tenantId || row.workspaceId) summary.scoped += 1
      return summary
    },
    { active: 0, bootstrap: 0, disabled: 0, scoped: 0 },
  )
}
```

- [x] **Step 5: Add controller hook**

Create `frontend/src/hooks/useAdminAccessController.ts` with state:

```ts
interface AdminAccessState {
  creating: boolean
  identities: AdminIdentity[]
  loading: boolean
  message: { key: string; params?: Record<string, string | number> } | null
  modal: "create" | "rotate" | "disable" | null
  selected: AdminIdentity | null
  oneTimeKey: string
}
```

Expose:

```ts
loadAdminIdentities()
openCreate()
openRotate(identity)
openDisable(identity)
closeModal()
submitCreate(body)
submitRotate()
submitDisable()
clearOneTimeKey()
```

All messages must store `{ key, params }`, not translated strings.

- [x] **Step 6: Add workspace component**

Create `frontend/src/components/AdminAccessManagementView.tsx` with:

- Summary metric row.
- Primary blue button `adminAccess.create`.
- Table/list columns for administrator, role, scope, source, status, key prefix, last used, actions.
- Managed rows show rotate/disable buttons.
- Bootstrap rows show read-only label.
- Modal form for create.
- Confirmation modal for rotate and disable.
- One-time key panel after create/rotate with copy button and no persistence after close.

Use existing class patterns: `primary-button`, `secondary-button`, `danger-button`, `content-panel`, `form-grid`, and token colors.

- [x] **Step 7: Wire ConsoleController**

Import hook and view, create `adminAccessPanel`, and add a case:

```tsx
case "admin-access":
  return <AdminAccessView adminAccessPanel={adminAccessPanel} />;
```

Add `AdminAccessView` to `ConsoleViews.tsx`:

```tsx
export function AdminAccessView({ adminAccessPanel }: { adminAccessPanel: ReactNode }) {
  return <section className="content-grid">{adminAccessPanel}</section>;
}
```

- [x] **Step 8: Add i18n keys**

Add English and Simplified Chinese keys:

```ts
"page.adminAccess": "Administrators & Boundaries",
"navDetail.adminAccess": "Manage administrator login keys and tenant boundaries.",
"adminAccess.create": "Create administrator",
"adminAccess.title": "Administrators & Boundaries",
"adminAccess.subtitle": "Manage who can operate the control plane and which tenant/workspace they can reach.",
"adminAccess.oneTimeKey": "One-time administrator key",
"adminAccess.oneTimeKeyDetail": "Copy it now. It will not be shown again.",
```

Chinese:

```ts
"page.adminAccess": "管理员与边界",
"navDetail.adminAccess": "管理管理员登录密钥和租户边界。",
"adminAccess.create": "创建管理员",
"adminAccess.title": "管理员与边界",
"adminAccess.subtitle": "管理谁可以操作控制台，以及可访问的租户和工作区范围。",
"adminAccess.oneTimeKey": "一次性管理员密钥",
"adminAccess.oneTimeKeyDetail": "请立即复制，关闭后不会再次展示。",
```

- [x] **Step 9: Add frontend tests**

Add source tests that assert:

- `admin-access` exists in navigation.
- EN and zh-CN i18n both include all `adminAccess.*` keys.
- `AdminAccessManagementView.tsx` contains `oneTimeKey` panel and does not render `keyHash`.
- Modal actions use `primary-button` for create/rotate and `danger-button` for disable.

Run:

```bash
pnpm --dir frontend test
```

Expected: PASS.

- [x] **Step 10: Commit frontend**

```bash
git add frontend/src frontend/tests
git commit -m "feat: add admin boundary management workspace"
```

---

### Task 6: Release Scenario And Documentation

**Files:**
- Create: `scripts/scenario-admin-access-management.sh`
- Modify: `Makefile`
- Modify: `tests/makefile_targets_test.sh`
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Create scenario script**

Create `scripts/scenario-admin-access-management.sh` with the same shell conventions used by existing scenario scripts: `set -euo pipefail`, `BASE_URL`, `RUN_ID`, `cleanup`, `wait_http`, `request`, `expect_status`, and Python JSON assertions. Do not source another scenario script; keep this scenario self-contained so `bash -n` and `make scenario-admin-access-management` can validate it independently.

The scenario must:

1. Start API with a bootstrap platform identity.
2. Create managed tenant admin through `/api/v1/admin-identities`.
3. Log in with returned key.
4. Prove tenant admin cannot call `/api/v1/admin-identities`.
5. Create tenant-scoped data and prove tenant admin sees only its scope.
6. Rotate managed key and prove old key fails/new key succeeds.
7. Disable managed identity and prove key login fails.
8. Query `/api/v1/audit/events?resourceType=admin_identity` and assert three lifecycle actions.
9. Assert returned list and audit JSON do not contain generated plaintext keys.

- [x] **Step 2: Add Makefile target**

Add to `SCENARIO_SCRIPTS`:

```make
scripts/scenario-admin-access-management.sh \
```

Add phony target:

```make
scenario-admin-access-management:
	bash scripts/scenario-admin-access-management.sh
```

Add to `release-check` after `scenario-admin-tenant-boundary`:

```make
scenario-admin-access-management
```

- [x] **Step 3: Update makefile target test**

In `tests/makefile_targets_test.sh`, add the target to the expected release gate list.

- [x] **Step 4: Update README**

Document:

- Bootstrap env identities are read-only in product.
- Managed administrator identities are created in the console.
- Managed keys are shown once.
- Tenant admins cannot manage administrators.
- Recovery path: keep at least one bootstrap platform administrator configured for production break-glass.

- [x] **Step 5: Update CHANGELOG**

Add bilingual entries:

```markdown
- Web console now includes Administrators & Boundaries, where platform administrators can create scoped managed administrators, rotate one-time keys, disable identities, and audit lifecycle changes without editing deployment configuration.
- 控制台新增“管理员与边界”，平台管理员可以在产品内创建范围化管理员、轮换一次性密钥、禁用身份并审计生命周期变更，不再依赖修改部署配置完成日常管理。
```

- [x] **Step 6: Run focused gates**

```bash
bash tests/makefile_targets_test.sh
bash -n scripts/scenario-admin-access-management.sh
make scenario-admin-access-management
```

Expected: PASS.

- [x] **Step 7: Commit docs and scenario**

```bash
git add scripts/scenario-admin-access-management.sh Makefile tests/makefile_targets_test.sh README.md CHANGELOG.md
git commit -m "test: gate managed admin identity lifecycle"
```

---

### Task 7: Full Verification And PR Preparation

**Files:**
- Modify: `docs/superpowers/plans/2026-06-16-admin-tenant-access-management.md`

- [x] **Step 1: Run focused backend tests**

```bash
go test ./internal/store ./internal/httpapi -run 'AdminIdentity|ManagedAdmin|ManagementMCPAdminIdentity|ConsoleAuthSession' -count=1
```

Expected: PASS.

- [x] **Step 2: Run frontend tests and build**

```bash
pnpm --dir frontend test
pnpm --dir frontend build
```

Expected: PASS.

- [x] **Step 3: Run repository gates**

```bash
make check
make release-check
```

Expected: PASS.

- [x] **Step 4: Browser smoke**

Start local demo stack and open the console:

```bash
make demo
```

Verify:

- The navigation contains **管理员与边界**.
- A platform administrator can open the create modal.
- A tenant administrator sees a forbidden or read-only state for administrator management.
- One-time key panel is prominent and disappears when cleared.

Observed on the authenticated local stack:

- Platform administrator login with `platform-key` opened **管理员与边界** and the create modal.
- Created `smoke-tenant-admin`, confirmed the one-time key panel was shown once, and confirmed it disappeared after clearing.
- Tenant administrator login with the generated one-time key loaded the console without the global loading shell, hid create actions, and showed the localized platform-administrator-required state.

- [x] **Step 5: Check plan boxes**

Update this plan so completed steps are marked `[x]`.

- [x] **Step 6: Final commit if plan checkboxes changed**

```bash
git add docs/superpowers/plans/2026-06-16-admin-tenant-access-management.md
git commit -m "docs: mark admin access management plan complete"
```

- [ ] **Step 7: Push and open PR**

```bash
git push -u origin codex/admin-tenant-access-management
gh pr create --base main --head codex/admin-tenant-access-management --title "Add managed admin access management" --body-file /tmp/agent-harbor-admin-access-pr.md
```

PR body must include:

- Summary.
- Security guardrails.
- Verification results.
- Browser smoke notes.
- Explicit non-goals: no SSO, no password login, no delegated tenant-admin identity management.
