package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/security"
)

type sqlExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Postgres struct {
	pool          *pgxpool.Pool
	credentialKey []byte
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func NewPostgresWithCredentialKey(pool *pgxpool.Pool, key []byte) *Postgres {
	copied := make([]byte, len(key))
	copy(copied, key)
	return &Postgres{pool: pool, credentialKey: copied}
}

func (p *Postgres) CreateTenant(ctx context.Context, tenant domain.Tenant) (domain.Tenant, error) {
	return p.createTenant(ctx, p.pool, tenant)
}

func (p *Postgres) createTenant(ctx context.Context, exec sqlExecutor, tenant domain.Tenant) (domain.Tenant, error) {
	_, err := exec.Exec(ctx, `
		insert into tenants (
			id, parent_tenant_id, level, name, status, created_at, updated_at
		) values ($1,$2,$3,$4,$5,$6,$7)
	`, tenant.ID, nullString(tenant.ParentTenantID), tenant.Level, tenant.Name, string(tenant.Status), tenant.CreatedAt, tenant.UpdatedAt)
	if err != nil {
		return domain.Tenant{}, fmt.Errorf("insert tenant: %w", err)
	}
	return tenant, nil
}

func (p *Postgres) CreateTenantWithAudit(ctx context.Context, tenant domain.Tenant, build TenantAuditBuilder) (domain.Tenant, error) {
	var created domain.Tenant
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createTenant(ctx, tx, tenant)
		if err != nil {
			return err
		}
		audit := build(created)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) ListTenants(ctx context.Context, filter TenantFilter) ([]domain.Tenant, error) {
	query := `
		select id, coalesce(parent_tenant_id, ''), level, name, status, created_at, updated_at
		from tenants
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.ParentTenantID) != "" {
		add("parent_tenant_id=$%d", strings.TrimSpace(filter.ParentTenantID))
	}
	query += " order by level asc, created_at asc, id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()
	return scanTenants(rows)
}

func (p *Postgres) GetTenant(ctx context.Context, id string) (domain.Tenant, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, coalesce(parent_tenant_id, ''), level, name, status, created_at, updated_at
		from tenants
		where id=$1
	`, id)
	tenant, err := scanTenant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Tenant{}, false, nil
	}
	if err != nil {
		return domain.Tenant{}, false, fmt.Errorf("get tenant: %w", err)
	}
	return tenant, true, nil
}

func (p *Postgres) tenantIDsForScope(ctx context.Context, tenantID string) ([]string, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, nil
	}
	rows, err := p.pool.Query(ctx, `
		with recursive tenant_tree as (
			select id
			from tenants
			where id=$1
			union all
			select t.id
			from tenants t
			join tenant_tree parent on parent.id = t.parent_tenant_id
		)
		select id from tenant_tree
		order by id
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tenant subtree: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []string{tenantID}, nil
	}
	return ids, nil
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

func (p *Postgres) ListAgents(ctx context.Context, filter AgentFilter) ([]domain.Agent, error) {
	query := `
		select id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, credentials_ciphertext, credential_version, status, created_at, updated_at
		from agents
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.WorkspaceID) != "" {
		add("workspace_id=$%d", filter.WorkspaceID)
	}
	query += " order by created_at asc, id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()
	return p.scanAgents(rows)
}

func (p *Postgres) GetAgent(ctx context.Context, id string) (domain.Agent, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, credentials_ciphertext, credential_version, status, created_at, updated_at
		from agents
		where id=$1
	`, id)
	agent, err := p.scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, false, nil
	}
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("get agent: %w", err)
	}
	return agent, true, nil
}

func (p *Postgres) UpdateAgent(ctx context.Context, agent domain.Agent) (domain.Agent, bool, error) {
	return p.updateAgent(ctx, p.pool, agent)
}

func (p *Postgres) updateAgent(ctx context.Context, exec sqlExecutor, agent domain.Agent) (domain.Agent, bool, error) {
	config, err := json.Marshal(agent.ChannelConfig)
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("marshal channel config: %w", err)
	}
	row := exec.QueryRow(ctx, `
		update agents
		set name=$2, description=$3, owner_id=$4, channel_config=$5, status=$6, updated_at=$7
		where id=$1
		returning id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, credentials_ciphertext, credential_version, status, created_at, updated_at
	`, agent.ID, agent.Name, agent.Description, agent.OwnerID, config, string(agent.Status), agent.UpdatedAt)
	updated, err := p.scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, false, nil
	}
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("update agent: %w", err)
	}
	return updated, true, nil
}

func (p *Postgres) RotateAgentCredentials(ctx context.Context, id string, credentials map[string]string, now time.Time) (domain.Agent, bool, error) {
	return p.rotateAgentCredentials(ctx, p.pool, id, credentials, now)
}

func (p *Postgres) rotateAgentCredentials(ctx context.Context, exec sqlExecutor, id string, credentials map[string]string, now time.Time) (domain.Agent, bool, error) {
	credentialsCiphertext, err := security.EncryptCredentials(credentials, p.credentialKey)
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("encrypt credentials: %w", err)
	}
	if credentialsCiphertext == nil {
		credentialsCiphertext = []byte{}
	}
	row := exec.QueryRow(ctx, `
		update agents
		set credentials_ciphertext=$2,
			credential_version=credential_version + 1,
			updated_at=$3
		where id=$1
		returning id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, credentials_ciphertext, credential_version, status, created_at, updated_at
	`, id, credentialsCiphertext, now)
	updated, err := p.scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, false, nil
	}
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("rotate agent credentials: %w", err)
	}
	return updated, true, nil
}

func (p *Postgres) DisableAgent(ctx context.Context, id string, now time.Time) (domain.Agent, bool, error) {
	return p.disableAgent(ctx, p.pool, id, now)
}

func (p *Postgres) disableAgent(ctx context.Context, exec sqlExecutor, id string, now time.Time) (domain.Agent, bool, error) {
	row := exec.QueryRow(ctx, `
		update agents
		set status=$2, updated_at=$3
		where id=$1
		returning id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, credentials_ciphertext, credential_version, status, created_at, updated_at
	`, id, string(domain.AgentStatusDisabled), now)
	agent, err := p.scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, false, nil
	}
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("disable agent: %w", err)
	}
	return agent, true, nil
}

func (p *Postgres) CreateAgentKey(ctx context.Context, key domain.AgentKey) (domain.AgentKey, error) {
	return p.createAgentKey(ctx, p.pool, key)
}

func (p *Postgres) createAgentKey(ctx context.Context, exec sqlExecutor, key domain.AgentKey) (domain.AgentKey, error) {
	_, err := exec.Exec(ctx, `
		insert into agent_keys (
			id, agent_id, name, hash, prefix, application_id, template_id,
			subject_selector, created_for_handoff_id, created_at, expires_at, revoked_at
		)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, key.ID, key.AgentID, key.Name, key.Hash, key.Prefix, key.ApplicationID, key.TemplateID,
		key.SubjectSelector, key.CreatedForHandoffID, key.CreatedAt, key.ExpiresAt, nullTime(key.RevokedAt))
	if err != nil {
		return domain.AgentKey{}, fmt.Errorf("insert agent key: %w", err)
	}
	return key, nil
}

func (p *Postgres) ListAgentKeys(ctx context.Context, scope ManagementScope) ([]domain.AgentKey, error) {
	query := `
		select k.id, k.agent_id, k.name, k.hash, k.prefix, k.application_id, k.template_id,
			k.subject_selector, k.created_for_handoff_id, k.created_at, k.expires_at, k.revoked_at
		from agent_keys k
		join agents a on a.id = k.agent_id
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(scope.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, scope.TenantID)
		if err != nil {
			return nil, err
		}
		add("a.tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(scope.WorkspaceID) != "" {
		add("a.workspace_id=$%d", scope.WorkspaceID)
	}
	query += " order by k.created_at asc, k.id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agent keys: %w", err)
	}
	defer rows.Close()
	return scanAgentKeys(rows)
}

func (p *Postgres) RevokeAgentKey(ctx context.Context, id string, now time.Time) (domain.AgentKey, bool, error) {
	return p.revokeAgentKey(ctx, p.pool, id, now)
}

func (p *Postgres) revokeAgentKey(ctx context.Context, exec sqlExecutor, id string, now time.Time) (domain.AgentKey, bool, error) {
	row := exec.QueryRow(ctx, `
		update agent_keys
		set revoked_at=$2
		where id=$1
		returning id, agent_id, name, hash, prefix, application_id, template_id,
			subject_selector, created_for_handoff_id, created_at, expires_at, revoked_at
	`, id, now)
	key, err := scanAgentKey(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AgentKey{}, false, nil
	}
	if err != nil {
		return domain.AgentKey{}, false, fmt.Errorf("revoke agent key: %w", err)
	}
	return key, true, nil
}

func (p *Postgres) FindAgentByKeyHash(ctx context.Context, hash string, now time.Time) (domain.Agent, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select a.id, a.tenant_id, a.workspace_id, a.name, a.description, a.owner_id,
			a.channel_type, a.channel_config, a.credentials_ciphertext, a.credential_version, a.status, a.created_at, a.updated_at
		from agent_keys k
		join agents a on a.id = k.agent_id
		where k.hash=$1 and k.revoked_at is null and k.expires_at > $2 and a.status=$3
	`, hash, now, string(domain.AgentStatusActive))
	agent, err := p.scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, false, nil
	}
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("find agent by key: %w", err)
	}
	return agent, true, nil
}

func (p *Postgres) FindAgentKeyByHash(ctx context.Context, hash string, now time.Time) (domain.AgentKey, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, agent_id, name, hash, prefix, application_id, template_id,
			subject_selector, created_for_handoff_id, created_at, expires_at, revoked_at
		from agent_keys
		where hash=$1 and revoked_at is null and expires_at > $2
	`, hash, now)
	key, err := scanAgentKey(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AgentKey{}, false, nil
	}
	if err != nil {
		return domain.AgentKey{}, false, fmt.Errorf("find agent key by hash: %w", err)
	}
	return key, true, nil
}

func (p *Postgres) CreateAccessGrant(ctx context.Context, grant domain.AccessGrant) (domain.AccessGrant, error) {
	return p.createAccessGrant(ctx, p.pool, grant)
}

func (p *Postgres) createAccessGrant(ctx context.Context, exec sqlExecutor, grant domain.AccessGrant) (domain.AccessGrant, error) {
	_, err := exec.Exec(ctx, `
		insert into access_grants (
			id, caller_agent_id, target_agent_id, route_type, route_key, created_at, expires_at, revoked_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8)
	`, grant.ID, grant.CallerID, grant.TargetID, grant.RouteType, grant.RouteKey,
		grant.CreatedAt, nullTime(grant.ExpiresAt), nullTime(grant.RevokedAt))
	if err != nil {
		return domain.AccessGrant{}, fmt.Errorf("insert access grant: %w", err)
	}
	return grant, nil
}

func (p *Postgres) ListAccessGrants(ctx context.Context, scope ManagementScope) ([]domain.AccessGrant, error) {
	query := `
		select g.id, g.caller_agent_id, g.target_agent_id, g.route_type, g.route_key, g.created_at, g.expires_at, g.revoked_at
		from access_grants g
		join agents c on c.id = g.caller_agent_id
		join agents t on t.id = g.target_agent_id
		where 1=1
	`
	args := []any{}
	if strings.TrimSpace(scope.TenantID) != "" || strings.TrimSpace(scope.WorkspaceID) != "" {
		var tenantIDs []string
		if strings.TrimSpace(scope.TenantID) != "" {
			var err error
			tenantIDs, err = p.tenantIDsForScope(ctx, scope.TenantID)
			if err != nil {
				return nil, err
			}
		}
		workspaceID := strings.TrimSpace(scope.WorkspaceID)
		caller := []string{}
		target := []string{}
		if tenantIDs != nil {
			args = append(args, tenantIDs)
			idx := len(args)
			caller = append(caller, fmt.Sprintf("c.tenant_id = any($%d)", idx))
			target = append(target, fmt.Sprintf("t.tenant_id = any($%d)", idx))
		}
		if workspaceID != "" {
			args = append(args, workspaceID)
			idx := len(args)
			caller = append(caller, fmt.Sprintf("c.workspace_id=$%d", idx))
			target = append(target, fmt.Sprintf("t.workspace_id=$%d", idx))
		}
		query += fmt.Sprintf(" and (%s) and (%s)", strings.Join(caller, " and "), strings.Join(target, " and "))
	}
	query += " order by g.created_at asc, g.id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list access grants: %w", err)
	}
	defer rows.Close()
	return scanAccessGrants(rows)
}

func (p *Postgres) RevokeAccessGrant(ctx context.Context, id string, now time.Time) (domain.AccessGrant, bool, error) {
	return p.revokeAccessGrant(ctx, p.pool, id, now)
}

func (p *Postgres) revokeAccessGrant(ctx context.Context, exec sqlExecutor, id string, now time.Time) (domain.AccessGrant, bool, error) {
	row := exec.QueryRow(ctx, `
		update access_grants
		set revoked_at=$2
		where id=$1
		returning id, caller_agent_id, target_agent_id, route_type, route_key, created_at, expires_at, revoked_at
	`, id, now)
	grant, err := scanAccessGrant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AccessGrant{}, false, nil
	}
	if err != nil {
		return domain.AccessGrant{}, false, fmt.Errorf("revoke access grant: %w", err)
	}
	return grant, true, nil
}

func (p *Postgres) HasGrant(ctx context.Context, callerID string, targetID string, routeType string, routeKey string, now time.Time) bool {
	var exists bool
	err := p.pool.QueryRow(ctx, `
		select exists(
			select 1
			from access_grants
			join agents c on c.id = access_grants.caller_agent_id
			join agents t on t.id = access_grants.target_agent_id
			where access_grants.caller_agent_id=$1
				and access_grants.target_agent_id=$2
				and t.tenant_id = c.tenant_id
				and t.workspace_id = c.workspace_id
				and revoked_at is null
				and (expires_at is null or expires_at > $5)
				and (route_type='' or route_type=$3)
				and (route_key='' or route_key=$4)
		)
	`, callerID, targetID, routeType, routeKey, now).Scan(&exists)
	return err == nil && exists
}

func (p *Postgres) UpsertCapability(ctx context.Context, capability domain.Capability) (domain.Capability, error) {
	inputSchema, err := json.Marshal(capability.InputSchema)
	if err != nil {
		return domain.Capability{}, fmt.Errorf("marshal capability input schema: %w", err)
	}
	outputSchema, err := json.Marshal(capability.OutputSchema)
	if err != nil {
		return domain.Capability{}, fmt.Errorf("marshal capability output schema: %w", err)
	}
	nativeScopes, err := json.Marshal(capability.NativeScopes)
	if err != nil {
		return domain.Capability{}, fmt.Errorf("marshal capability native scopes: %w", err)
	}
	dataDomains, err := json.Marshal(capability.DataDomains)
	if err != nil {
		return domain.Capability{}, fmt.Errorf("marshal capability data domains: %w", err)
	}
	dataScopes, err := json.Marshal(capability.DataScopes)
	if err != nil {
		return domain.Capability{}, fmt.Errorf("marshal capability data scopes: %w", err)
	}
	row := p.pool.QueryRow(ctx, `
		insert into capabilities (
			id, target_agent_id, type, key, display_name, description, action,
			input_schema, output_schema, native_scopes, data_domains, data_scopes,
			sensitivity, risk_level, enforcement_mode, discovery_status, version, discovered_at, updated_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		on conflict(target_agent_id, type, key) do update
		set display_name=excluded.display_name,
			description=excluded.description,
			action=excluded.action,
			input_schema=excluded.input_schema,
			output_schema=excluded.output_schema,
			native_scopes=excluded.native_scopes,
			data_domains=excluded.data_domains,
			data_scopes=excluded.data_scopes,
			sensitivity=excluded.sensitivity,
			risk_level=excluded.risk_level,
			enforcement_mode=excluded.enforcement_mode,
			discovery_status=excluded.discovery_status,
			version=excluded.version,
			updated_at=excluded.updated_at
		returning id, target_agent_id, type, key, display_name, description, action,
			input_schema, output_schema, native_scopes, data_domains, data_scopes,
			sensitivity, risk_level, enforcement_mode, discovery_status, version, discovered_at, updated_at
	`, capability.ID, capability.TargetID, string(capability.Type), capability.Key, capability.DisplayName,
		capability.Description, string(capability.Action), inputSchema, outputSchema, nativeScopes,
		dataDomains, dataScopes, string(capability.Sensitivity), string(capability.RiskLevel),
		string(capability.EnforcementMode), string(capability.DiscoveryStatus), capability.Version,
		capability.DiscoveredAt, capability.UpdatedAt)
	created, err := scanCapability(row)
	if err != nil {
		return domain.Capability{}, fmt.Errorf("upsert capability: %w", err)
	}
	return created, nil
}

func (p *Postgres) ListCapabilities(ctx context.Context, filter CapabilityFilter) ([]domain.Capability, error) {
	query := `
		select c.id, c.target_agent_id, c.type, c.key, c.display_name, c.description, c.action,
			c.input_schema, c.output_schema, c.native_scopes, c.data_domains, c.data_scopes,
			c.sensitivity, c.risk_level, c.enforcement_mode, c.discovery_status, c.version, c.discovered_at, c.updated_at
		from capabilities c
		join agents target on target.id = c.target_agent_id
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TargetID) != "" {
		add("c.target_agent_id=$%d", strings.TrimSpace(filter.TargetID))
	}
	if strings.TrimSpace(string(filter.Status)) != "" {
		add("c.discovery_status=$%d", string(filter.Status))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("target.tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.WorkspaceID) != "" {
		add("target.workspace_id=$%d", strings.TrimSpace(filter.WorkspaceID))
	}
	query += " order by c.updated_at asc, c.id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list capabilities: %w", err)
	}
	defer rows.Close()
	return scanCapabilities(rows)
}

func (p *Postgres) GetCapability(ctx context.Context, id string) (domain.Capability, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, target_agent_id, type, key, display_name, description, action,
			input_schema, output_schema, native_scopes, data_domains, data_scopes,
			sensitivity, risk_level, enforcement_mode, discovery_status, version, discovered_at, updated_at
		from capabilities
		where id=$1
	`, id)
	capability, err := scanCapability(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Capability{}, false, nil
	}
	if err != nil {
		return domain.Capability{}, false, fmt.Errorf("get capability: %w", err)
	}
	return capability, true, nil
}

func (p *Postgres) UpdateCapability(ctx context.Context, capability domain.Capability) (domain.Capability, bool, error) {
	return p.updateCapability(ctx, p.pool, capability)
}

func (p *Postgres) updateCapability(ctx context.Context, exec sqlExecutor, capability domain.Capability) (domain.Capability, bool, error) {
	dataScopes, err := json.Marshal(capability.DataScopes)
	if err != nil {
		return domain.Capability{}, false, fmt.Errorf("marshal capability data scopes: %w", err)
	}
	dataDomains, err := json.Marshal(capability.DataDomains)
	if err != nil {
		return domain.Capability{}, false, fmt.Errorf("marshal capability data domains: %w", err)
	}
	row := exec.QueryRow(ctx, `
		update capabilities
		set display_name=$2, description=$3, sensitivity=$4, risk_level=$5, discovery_status=$6, data_scopes=$7, data_domains=$8, updated_at=$9
		where id=$1
		returning id, target_agent_id, type, key, display_name, description, action,
			input_schema, output_schema, native_scopes, data_domains, data_scopes,
			sensitivity, risk_level, enforcement_mode, discovery_status, version, discovered_at, updated_at
	`, capability.ID, capability.DisplayName, capability.Description, string(capability.Sensitivity),
		string(capability.RiskLevel), string(capability.DiscoveryStatus), dataScopes, dataDomains, capability.UpdatedAt)
	updated, err := scanCapability(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Capability{}, false, nil
	}
	if err != nil {
		return domain.Capability{}, false, fmt.Errorf("update capability: %w", err)
	}
	return updated, true, nil
}

func (p *Postgres) CreateTenantEntitlement(ctx context.Context, entitlement domain.TenantEntitlement) (domain.TenantEntitlement, error) {
	return p.createTenantEntitlement(ctx, p.pool, entitlement)
}

func (p *Postgres) createTenantEntitlement(ctx context.Context, exec sqlExecutor, entitlement domain.TenantEntitlement) (domain.TenantEntitlement, error) {
	dataScopes, err := json.Marshal(entitlement.DataScopes)
	if err != nil {
		return domain.TenantEntitlement{}, fmt.Errorf("marshal entitlement data scopes: %w", err)
	}
	_, err = exec.Exec(ctx, `
		insert into tenant_entitlements (
			id, tenant_id, target_agent_id, capability_id, effect, data_scopes, status, priority, created_at, updated_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, entitlement.ID, entitlement.TenantID, entitlement.TargetID, entitlement.CapabilityID,
		string(entitlement.Effect), dataScopes, string(entitlement.Status), entitlement.Priority,
		entitlement.CreatedAt, entitlement.UpdatedAt)
	if err != nil {
		return domain.TenantEntitlement{}, fmt.Errorf("insert tenant entitlement: %w", err)
	}
	return entitlement, nil
}

func (p *Postgres) ListTenantEntitlements(ctx context.Context, filter EntitlementFilter) ([]domain.TenantEntitlement, error) {
	query := `
		select id, tenant_id, target_agent_id, capability_id, effect, data_scopes, status, priority, created_at, updated_at
		from tenant_entitlements
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.TargetID) != "" {
		add("target_agent_id=$%d", strings.TrimSpace(filter.TargetID))
	}
	if strings.TrimSpace(filter.CapabilityID) != "" {
		add("capability_id=$%d", strings.TrimSpace(filter.CapabilityID))
	}
	query += " order by created_at asc, id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tenant entitlements: %w", err)
	}
	defer rows.Close()
	return scanTenantEntitlements(rows)
}

func (p *Postgres) CreateWorkspaceAssignment(ctx context.Context, assignment domain.WorkspaceAssignment) (domain.WorkspaceAssignment, error) {
	return p.createWorkspaceAssignment(ctx, p.pool, assignment)
}

func (p *Postgres) createWorkspaceAssignment(ctx context.Context, exec sqlExecutor, assignment domain.WorkspaceAssignment) (domain.WorkspaceAssignment, error) {
	dataScopes, err := json.Marshal(assignment.DataScopes)
	if err != nil {
		return domain.WorkspaceAssignment{}, fmt.Errorf("marshal workspace assignment data scopes: %w", err)
	}
	_, err = exec.Exec(ctx, `
		insert into workspace_assignments (
			id, tenant_entitlement_id, tenant_id, workspace_id, effect, data_scopes, status, created_at, updated_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, assignment.ID, assignment.TenantEntitlementID, assignment.TenantID, assignment.WorkspaceID,
		string(assignment.Effect), dataScopes, string(assignment.Status), assignment.CreatedAt, assignment.UpdatedAt)
	if err != nil {
		return domain.WorkspaceAssignment{}, fmt.Errorf("insert workspace assignment: %w", err)
	}
	return assignment, nil
}

func (p *Postgres) ListWorkspaceAssignments(ctx context.Context, filter AssignmentFilter) ([]domain.WorkspaceAssignment, error) {
	query := `
		select id, tenant_entitlement_id, tenant_id, workspace_id, effect, data_scopes, status, created_at, updated_at
		from workspace_assignments
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.WorkspaceID) != "" {
		add("workspace_id=$%d", strings.TrimSpace(filter.WorkspaceID))
	}
	if strings.TrimSpace(filter.EntitlementID) != "" {
		add("tenant_entitlement_id=$%d", strings.TrimSpace(filter.EntitlementID))
	}
	query += " order by created_at asc, id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list workspace assignments: %w", err)
	}
	defer rows.Close()
	return scanWorkspaceAssignments(rows)
}

func (p *Postgres) CreateInstanceAssignment(ctx context.Context, assignment domain.InstanceAssignment) (domain.InstanceAssignment, error) {
	return p.createInstanceAssignment(ctx, p.pool, assignment)
}

func (p *Postgres) createInstanceAssignment(ctx context.Context, exec sqlExecutor, assignment domain.InstanceAssignment) (domain.InstanceAssignment, error) {
	dataScopes, err := json.Marshal(assignment.DataScopes)
	if err != nil {
		return domain.InstanceAssignment{}, fmt.Errorf("marshal instance assignment data scopes: %w", err)
	}
	_, err = exec.Exec(ctx, `
		insert into instance_assignments (
			id, workspace_assignment_id, tenant_id, workspace_id, caller_instance_id, subject_selector,
			effect, data_scopes, status, created_at, updated_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, assignment.ID, assignment.WorkspaceAssignmentID, assignment.TenantID, assignment.WorkspaceID,
		assignment.CallerInstanceID, assignment.SubjectSelector, string(assignment.Effect),
		dataScopes, string(assignment.Status), assignment.CreatedAt, assignment.UpdatedAt)
	if err != nil {
		return domain.InstanceAssignment{}, fmt.Errorf("insert instance assignment: %w", err)
	}
	return assignment, nil
}

func (p *Postgres) ListInstanceAssignments(ctx context.Context, filter InstanceAssignmentFilter) ([]domain.InstanceAssignment, error) {
	query := `
		select ia.id, ia.workspace_assignment_id, ia.tenant_id, ia.workspace_id, ia.caller_instance_id,
			ia.subject_selector, ia.effect, ia.data_scopes, ia.status, ia.created_at, ia.updated_at
		from instance_assignments ia
	`
	args := []any{}
	if strings.TrimSpace(filter.CapabilityID) != "" {
		query += `
			join workspace_assignments wa on wa.id = ia.workspace_assignment_id
			join tenant_entitlements te on te.id = wa.tenant_entitlement_id
		`
	}
	query += " where 1=1"
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("ia.tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.WorkspaceID) != "" {
		add("ia.workspace_id=$%d", strings.TrimSpace(filter.WorkspaceID))
	}
	if strings.TrimSpace(filter.CallerInstanceID) != "" {
		add("ia.caller_instance_id=$%d", strings.TrimSpace(filter.CallerInstanceID))
	}
	if strings.TrimSpace(filter.CapabilityID) != "" {
		add("te.capability_id=$%d", strings.TrimSpace(filter.CapabilityID))
	}
	query += " order by ia.created_at asc, ia.id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list instance assignments: %w", err)
	}
	defer rows.Close()
	return scanInstanceAssignments(rows)
}

func (p *Postgres) CreatePermissionPackageApplication(ctx context.Context, application domain.PermissionPackageApplication) (domain.PermissionPackageApplication, error) {
	return p.createPermissionPackageApplication(ctx, p.pool, application)
}

func (p *Postgres) createPermissionPackageApplication(ctx context.Context, exec sqlExecutor, application domain.PermissionPackageApplication) (domain.PermissionPackageApplication, error) {
	dataScopes, err := json.Marshal(application.DataScopes)
	if err != nil {
		return domain.PermissionPackageApplication{}, fmt.Errorf("marshal permission package application data scopes: %w", err)
	}
	allowedCapabilityIDs, err := json.Marshal(application.AllowedCapabilityIDs)
	if err != nil {
		return domain.PermissionPackageApplication{}, fmt.Errorf("marshal permission package application capability ids: %w", err)
	}
	allowedCapabilityKeys, err := json.Marshal(application.AllowedCapabilityKeys)
	if err != nil {
		return domain.PermissionPackageApplication{}, fmt.Errorf("marshal permission package application capability keys: %w", err)
	}
	tenantEntitlementIDs, err := json.Marshal(application.TenantEntitlementIDs)
	if err != nil {
		return domain.PermissionPackageApplication{}, fmt.Errorf("marshal permission package application entitlement ids: %w", err)
	}
	workspaceAssignmentIDs, err := json.Marshal(application.WorkspaceAssignmentIDs)
	if err != nil {
		return domain.PermissionPackageApplication{}, fmt.Errorf("marshal permission package application workspace assignment ids: %w", err)
	}
	instanceAssignmentIDs, err := json.Marshal(application.InstanceAssignmentIDs)
	if err != nil {
		return domain.PermissionPackageApplication{}, fmt.Errorf("marshal permission package application instance assignment ids: %w", err)
	}
	_, err = exec.Exec(ctx, `
		insert into permission_package_applications (
			id, draft_id, template_id, template_version, tenant_id, workspace_id, target_agent_id,
			caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, tenant_entitlement_ids,
			workspace_assignment_ids, instance_assignment_ids, applied_at, requested_capability_id
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
	`, application.ID, application.DraftID, application.TemplateID, application.TemplateVersion,
		application.TenantID, application.WorkspaceID, application.TargetID, application.CallerInstanceID,
		application.SubjectSelector, application.RequestText, application.Region, dataScopes,
		allowedCapabilityIDs, allowedCapabilityKeys, tenantEntitlementIDs, workspaceAssignmentIDs,
		instanceAssignmentIDs, application.AppliedAt, application.RequestedCapabilityID)
	if err != nil {
		return domain.PermissionPackageApplication{}, fmt.Errorf("insert permission package application: %w", err)
	}
	return application, nil
}

func (p *Postgres) ListPermissionPackageApplications(ctx context.Context, filter PermissionPackageApplicationFilter) ([]domain.PermissionPackageApplication, error) {
	query := `
		select id, draft_id, template_id, template_version, tenant_id, workspace_id, target_agent_id,
			caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, tenant_entitlement_ids,
			workspace_assignment_ids, instance_assignment_ids, applied_at, requested_capability_id
		from permission_package_applications
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.ID) != "" {
		add("id=$%d", strings.TrimSpace(filter.ID))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.WorkspaceID) != "" {
		add("workspace_id=$%d", strings.TrimSpace(filter.WorkspaceID))
	}
	if strings.TrimSpace(filter.TemplateID) != "" {
		add("template_id=$%d", strings.TrimSpace(filter.TemplateID))
	}
	if strings.TrimSpace(filter.TargetID) != "" {
		add("target_agent_id=$%d", strings.TrimSpace(filter.TargetID))
	}
	if strings.TrimSpace(filter.CallerInstanceID) != "" {
		add("caller_instance_id=$%d", strings.TrimSpace(filter.CallerInstanceID))
	}
	if filter.MatchRequestedCapabilityID || strings.TrimSpace(filter.RequestedCapabilityID) != "" {
		add("requested_capability_id=$%d", strings.TrimSpace(filter.RequestedCapabilityID))
	}
	query += " order by applied_at desc, id desc"
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" limit $%d", len(args))
	}
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list permission package applications: %w", err)
	}
	defer rows.Close()
	return scanPermissionPackageApplications(rows)
}

func (p *Postgres) ApplyPermissionPackage(ctx context.Context, mutation PermissionPackageApplyMutation) (PermissionPackageApplyMutationResult, error) {
	var result PermissionPackageApplyMutationResult
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		result = PermissionPackageApplyMutationResult{
			Capabilities:         make([]domain.Capability, 0, len(mutation.Capabilities)),
			TenantEntitlements:   make([]domain.TenantEntitlement, 0, len(mutation.TenantEntitlements)),
			WorkspaceAssignments: make([]domain.WorkspaceAssignment, 0, len(mutation.WorkspaceAssignments)),
			InstanceAssignments:  make([]domain.InstanceAssignment, 0, len(mutation.InstanceAssignments)),
		}

		if err := p.rejectDuplicatePermissionPackageApplication(ctx, tx, mutation.Application); err != nil {
			return err
		}

		if mutation.ApprovalRequest != nil {
			approval, err := p.consumePermissionPackageApprovalRequest(ctx, tx, *mutation.ApprovalRequest)
			if err != nil {
				return err
			}
			if !permissionPackageApplyApprovalSnapshotMatches(mutation.ExpectedApproval, approval, mutation.Application) {
				return ErrPermissionPackageCapabilitySnapshotChanged
			}
			result.ApprovalRequest = &approval
			mutation.ApprovalRequest = &approval
		}
		capabilitiesByID := make(map[string]domain.Capability, len(mutation.Capabilities))
		capabilityIDs := make([]string, 0, len(mutation.Capabilities))
		for _, capabilityMutation := range mutation.Capabilities {
			capabilityIDs = append(capabilityIDs, capabilityMutation.Capability.ID)
		}
		sort.Strings(capabilityIDs)
		for _, capabilityID := range capabilityIDs {
			current, ok, err := p.getCapabilityForUpdate(ctx, tx, capabilityID)
			if err != nil {
				return err
			}
			if !ok {
				return domain.NotFound("capability not found")
			}
			capabilitiesByID[capabilityID] = current
		}
		if err := validatePermissionPackageApplyCapabilitySnapshots(mutation, func(id string) (domain.Capability, bool) {
			capability, ok := capabilitiesByID[id]
			return capability, ok
		}); err != nil {
			return err
		}
		for _, capabilityMutation := range mutation.Capabilities {
			capability := capabilityMutation.Capability
			updated, ok, err := p.updateCapability(ctx, tx, capability)
			if err != nil {
				return err
			}
			if !ok {
				return domain.NotFound("capability not found")
			}
			result.Capabilities = append(result.Capabilities, updated)
		}
		for _, entitlement := range mutation.TenantEntitlements {
			created, err := p.createTenantEntitlement(ctx, tx, entitlement)
			if err != nil {
				return err
			}
			result.TenantEntitlements = append(result.TenantEntitlements, created)
		}
		for _, assignment := range mutation.WorkspaceAssignments {
			created, err := p.createWorkspaceAssignment(ctx, tx, assignment)
			if err != nil {
				return err
			}
			result.WorkspaceAssignments = append(result.WorkspaceAssignments, created)
		}
		for _, assignment := range mutation.InstanceAssignments {
			created, err := p.createInstanceAssignment(ctx, tx, assignment)
			if err != nil {
				return err
			}
			result.InstanceAssignments = append(result.InstanceAssignments, created)
		}
		application, err := p.createPermissionPackageApplication(ctx, tx, mutation.Application)
		if err != nil {
			return err
		}
		result.Application = application
		audit, err := p.appendAuditEvent(ctx, tx, mutation.AuditEvent)
		if err != nil {
			return err
		}
		result.AuditEvent = audit
		return nil
	})
	return result, err
}

func (p *Postgres) getCapabilityForUpdate(ctx context.Context, tx pgx.Tx, id string) (domain.Capability, bool, error) {
	row := tx.QueryRow(ctx, `
		select id, target_agent_id, type, key, display_name, description, action,
			input_schema, output_schema, native_scopes, data_domains, data_scopes,
			sensitivity, risk_level, enforcement_mode, discovery_status, version, discovered_at, updated_at
		from capabilities
		where id=$1
		for update
	`, id)
	capability, err := scanCapability(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Capability{}, false, nil
	}
	if err != nil {
		return domain.Capability{}, false, fmt.Errorf("lock permission package capability: %w", err)
	}
	return capability, true, nil
}

func (p *Postgres) rejectDuplicatePermissionPackageApplication(ctx context.Context, tx pgx.Tx, application domain.PermissionPackageApplication) error {
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1::text, 0::bigint))`, permissionPackageApplicationDuplicateLockKey(application)); err != nil {
		return fmt.Errorf("lock duplicate permission package application: %w", err)
	}
	rows, err := tx.Query(ctx, `
		select id, draft_id, template_id, template_version, tenant_id, workspace_id, target_agent_id,
			caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, tenant_entitlement_ids,
			workspace_assignment_ids, instance_assignment_ids, applied_at, requested_capability_id
		from permission_package_applications
		where draft_id=$1
			and template_id=$2
			and template_version=$3
			and tenant_id=$4
			and workspace_id=$5
			and target_agent_id=$6
			and caller_instance_id=$7
			and subject_selector=$8
			and region=$9
	`, application.DraftID, application.TemplateID, application.TemplateVersion, application.TenantID,
		application.WorkspaceID, application.TargetID, application.CallerInstanceID, application.SubjectSelector,
		application.Region)
	if err != nil {
		return fmt.Errorf("list duplicate permission package applications: %w", err)
	}
	defer rows.Close()
	candidates, err := scanPermissionPackageApplications(rows)
	if err != nil {
		return fmt.Errorf("scan duplicate permission package applications: %w", err)
	}
	for _, candidate := range candidates {
		if permissionPackageApplicationsShareDuplicateKey(candidate, application) {
			return ErrPermissionPackageApplicationAlreadyApplied
		}
	}
	return nil
}

func (p *Postgres) CreatePermissionPackageApprovalRequest(ctx context.Context, request domain.PermissionPackageApprovalRequest) (domain.PermissionPackageApprovalRequest, error) {
	dataScopes, allowedCapabilityIDs, allowedCapabilityKeys, allowedCapabilityFingerprints, policyGate, err := marshalPermissionPackageApprovalRequestPayloads(request)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	err = p.withTx(ctx, func(tx pgx.Tx) error {
		if err := p.rejectDuplicateActivePermissionPackageApprovalRequest(ctx, tx, request); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
		insert into permission_package_approval_requests (
			id, draft_id, template_id, template_version, policy_version, tenant_id, workspace_id,
			target_agent_id, caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, allowed_capability_fingerprints,
			policy_gate, status, requested_by,
			reviewed_by, review_comment, created_at, updated_at, resolved_at, expires_at,
			consumed_at, consumed_by_application_id, requested_capability_id
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28)
	`, request.ID, request.DraftID, request.TemplateID, request.TemplateVersion, request.PolicyVersion,
			request.TenantID, request.WorkspaceID, request.TargetID, request.CallerInstanceID,
			request.SubjectSelector, request.RequestText, request.Region, dataScopes, allowedCapabilityIDs,
			allowedCapabilityKeys, allowedCapabilityFingerprints, policyGate, string(request.Status),
			request.RequestedBy, request.ReviewedBy, request.ReviewComment, request.CreatedAt, request.UpdatedAt,
			nullTime(request.ResolvedAt), request.ExpiresAt, nullTime(request.ConsumedAt),
			request.ConsumedByApplicationID, request.RequestedCapabilityID)
		return err
	})
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, fmt.Errorf("insert permission package approval request: %w", err)
	}
	return request, nil
}

func (p *Postgres) rejectDuplicateActivePermissionPackageApprovalRequest(ctx context.Context, tx pgx.Tx, request domain.PermissionPackageApprovalRequest) error {
	if request.Status != domain.PermissionPackageApprovalStatusPending {
		return nil
	}
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1::text, 0::bigint))`, permissionPackageApprovalRequestDuplicateLockKey(request)); err != nil {
		return fmt.Errorf("lock duplicate permission package approval request: %w", err)
	}
	referenceTime := permissionPackageApprovalRequestDuplicateReferenceTime(request)
	rows, err := tx.Query(ctx, `
		select id, draft_id, template_id, template_version, policy_version, tenant_id, workspace_id,
			target_agent_id, caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, allowed_capability_fingerprints,
			policy_gate, status, requested_by,
			reviewed_by, review_comment, created_at, updated_at, resolved_at, expires_at,
			consumed_at, consumed_by_application_id, requested_capability_id
		from permission_package_approval_requests
		where status=$1
			and draft_id=$2
			and template_id=$3
			and template_version=$4
			and policy_version=$5
			and tenant_id=$6
			and workspace_id=$7
			and target_agent_id=$8
			and caller_instance_id=$9
			and subject_selector=$10
			and request_text=$11
			and region=$12
			and requested_capability_id=$13
			and expires_at > $14
	`, string(domain.PermissionPackageApprovalStatusPending), request.DraftID, request.TemplateID,
		request.TemplateVersion, request.PolicyVersion, request.TenantID, request.WorkspaceID,
		request.TargetID, request.CallerInstanceID, request.SubjectSelector, request.RequestText,
		request.Region, request.RequestedCapabilityID, referenceTime)
	if err != nil {
		return fmt.Errorf("list duplicate permission package approval requests: %w", err)
	}
	defer rows.Close()
	candidates, err := scanPermissionPackageApprovalRequests(rows)
	if err != nil {
		return fmt.Errorf("scan duplicate permission package approval requests: %w", err)
	}
	for _, candidate := range candidates {
		if permissionPackageApprovalRequestsShareDuplicateKey(candidate, request) {
			return ErrPermissionPackageApprovalAlreadyPending
		}
	}
	return nil
}

func (p *Postgres) ListPermissionPackageApprovalRequests(ctx context.Context, filter PermissionPackageApprovalRequestFilter) ([]domain.PermissionPackageApprovalRequest, error) {
	query := `
		select id, draft_id, template_id, template_version, policy_version, tenant_id, workspace_id,
			target_agent_id, caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, allowed_capability_fingerprints,
			policy_gate, status, requested_by,
			reviewed_by, review_comment, created_at, updated_at, resolved_at, expires_at,
			consumed_at, consumed_by_application_id, requested_capability_id
		from permission_package_approval_requests
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.WorkspaceID) != "" {
		add("workspace_id=$%d", strings.TrimSpace(filter.WorkspaceID))
	}
	if strings.TrimSpace(filter.TemplateID) != "" {
		add("template_id=$%d", strings.TrimSpace(filter.TemplateID))
	}
	if strings.TrimSpace(filter.TargetID) != "" {
		add("target_agent_id=$%d", strings.TrimSpace(filter.TargetID))
	}
	if strings.TrimSpace(filter.CallerInstanceID) != "" {
		add("caller_instance_id=$%d", strings.TrimSpace(filter.CallerInstanceID))
	}
	if filter.MatchRequestedCapabilityID || strings.TrimSpace(filter.RequestedCapabilityID) != "" {
		add("requested_capability_id=$%d", strings.TrimSpace(filter.RequestedCapabilityID))
	}
	if strings.TrimSpace(string(filter.Status)) != "" {
		add("status=$%d", string(filter.Status))
	}
	query += " order by created_at desc, id desc"
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" limit $%d", len(args))
	}
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list permission package approval requests: %w", err)
	}
	defer rows.Close()
	return scanPermissionPackageApprovalRequests(rows)
}

func (p *Postgres) GetPermissionPackageApprovalRequest(ctx context.Context, id string) (domain.PermissionPackageApprovalRequest, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, draft_id, template_id, template_version, policy_version, tenant_id, workspace_id,
			target_agent_id, caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, allowed_capability_fingerprints,
			policy_gate, status, requested_by,
			reviewed_by, review_comment, created_at, updated_at, resolved_at, expires_at,
			consumed_at, consumed_by_application_id, requested_capability_id
		from permission_package_approval_requests
		where id=$1
	`, id)
	request, err := scanPermissionPackageApprovalRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PermissionPackageApprovalRequest{}, false, nil
	}
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, false, fmt.Errorf("get permission package approval request: %w", err)
	}
	return request, true, nil
}

func (p *Postgres) UpdatePermissionPackageApprovalRequest(ctx context.Context, request domain.PermissionPackageApprovalRequest) (domain.PermissionPackageApprovalRequest, bool, error) {
	dataScopes, allowedCapabilityIDs, allowedCapabilityKeys, allowedCapabilityFingerprints, policyGate, err := marshalPermissionPackageApprovalRequestPayloads(request)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, false, err
	}
	row := p.pool.QueryRow(ctx, `
		update permission_package_approval_requests
		set draft_id=$2, template_id=$3, template_version=$4, policy_version=$5, tenant_id=$6,
			workspace_id=$7, target_agent_id=$8, caller_instance_id=$9, subject_selector=$10,
			request_text=$11, region=$12, data_scopes=$13, allowed_capability_ids=$14,
			allowed_capability_keys=$15, allowed_capability_fingerprints=$16, policy_gate=$17,
			status=$18, requested_by=$19, reviewed_by=$20, review_comment=$21, created_at=$22,
			updated_at=$23, resolved_at=$24, expires_at=$25, consumed_at=$26,
			consumed_by_application_id=$27, requested_capability_id=$28
		where id=$1
		returning id, draft_id, template_id, template_version, policy_version, tenant_id, workspace_id,
			target_agent_id, caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, allowed_capability_fingerprints,
			policy_gate, status, requested_by,
			reviewed_by, review_comment, created_at, updated_at, resolved_at, expires_at,
			consumed_at, consumed_by_application_id, requested_capability_id
	`, request.ID, request.DraftID, request.TemplateID, request.TemplateVersion, request.PolicyVersion,
		request.TenantID, request.WorkspaceID, request.TargetID, request.CallerInstanceID,
		request.SubjectSelector, request.RequestText, request.Region, dataScopes, allowedCapabilityIDs,
		allowedCapabilityKeys, allowedCapabilityFingerprints, policyGate, string(request.Status),
		request.RequestedBy, request.ReviewedBy, request.ReviewComment, request.CreatedAt, request.UpdatedAt,
		nullTime(request.ResolvedAt), request.ExpiresAt, nullTime(request.ConsumedAt),
		request.ConsumedByApplicationID, request.RequestedCapabilityID)
	updated, err := scanPermissionPackageApprovalRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PermissionPackageApprovalRequest{}, false, nil
	}
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, false, fmt.Errorf("update permission package approval request: %w", err)
	}
	return updated, true, nil
}

func (p *Postgres) TransitionPermissionPackageApprovalRequest(ctx context.Context, request domain.PermissionPackageApprovalRequest, now time.Time) (domain.PermissionPackageApprovalRequest, bool, error) {
	dataScopes, allowedCapabilityIDs, allowedCapabilityKeys, allowedCapabilityFingerprints, policyGate, err := marshalPermissionPackageApprovalRequestPayloads(request)
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, false, err
	}
	row := p.pool.QueryRow(ctx, `
		update permission_package_approval_requests
		set draft_id=$2, template_id=$3, template_version=$4, policy_version=$5, tenant_id=$6,
			workspace_id=$7, target_agent_id=$8, caller_instance_id=$9, subject_selector=$10,
			request_text=$11, region=$12, data_scopes=$13, allowed_capability_ids=$14,
			allowed_capability_keys=$15, allowed_capability_fingerprints=$16, policy_gate=$17,
			status=$18, requested_by=$19, reviewed_by=$20, review_comment=$21, created_at=$22,
			updated_at=$23, resolved_at=$24, expires_at=$25, consumed_at=$26,
			consumed_by_application_id=$27, requested_capability_id=$30
		where id=$1
			and status=$28
			and consumed_at is null
			and consumed_by_application_id=''
			and expires_at > $29
		returning id, draft_id, template_id, template_version, policy_version, tenant_id, workspace_id,
			target_agent_id, caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, allowed_capability_fingerprints,
			policy_gate, status, requested_by,
			reviewed_by, review_comment, created_at, updated_at, resolved_at, expires_at,
			consumed_at, consumed_by_application_id, requested_capability_id
	`, request.ID, request.DraftID, request.TemplateID, request.TemplateVersion, request.PolicyVersion,
		request.TenantID, request.WorkspaceID, request.TargetID, request.CallerInstanceID,
		request.SubjectSelector, request.RequestText, request.Region, dataScopes, allowedCapabilityIDs,
		allowedCapabilityKeys, allowedCapabilityFingerprints, policyGate, string(request.Status),
		request.RequestedBy, request.ReviewedBy, request.ReviewComment, request.CreatedAt, request.UpdatedAt,
		nullTime(request.ResolvedAt), request.ExpiresAt, nullTime(request.ConsumedAt),
		request.ConsumedByApplicationID, string(domain.PermissionPackageApprovalStatusPending), now,
		request.RequestedCapabilityID)
	updated, err := scanPermissionPackageApprovalRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PermissionPackageApprovalRequest{}, false, nil
	}
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, false, fmt.Errorf("transition permission package approval request: %w", err)
	}
	return updated, true, nil
}

func (p *Postgres) consumePermissionPackageApprovalRequest(ctx context.Context, exec sqlExecutor, request domain.PermissionPackageApprovalRequest) (domain.PermissionPackageApprovalRequest, error) {
	row := exec.QueryRow(ctx, `
		update permission_package_approval_requests
		set consumed_at=$2, consumed_by_application_id=$3, updated_at=$4
		where id=$1
			and status=$5
			and consumed_at is null
			and expires_at > $2
		returning id, draft_id, template_id, template_version, policy_version, tenant_id, workspace_id,
			target_agent_id, caller_instance_id, subject_selector, request_text, region, data_scopes,
			allowed_capability_ids, allowed_capability_keys, allowed_capability_fingerprints,
			policy_gate, status, requested_by,
			reviewed_by, review_comment, created_at, updated_at, resolved_at, expires_at,
			consumed_at, consumed_by_application_id, requested_capability_id
	`, request.ID, nullTime(request.ConsumedAt), request.ConsumedByApplicationID, request.UpdatedAt,
		string(domain.PermissionPackageApprovalStatusApproved))
	updated, err := scanPermissionPackageApprovalRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PermissionPackageApprovalRequest{}, ErrPermissionPackageApprovalNotConsumable
	}
	if err != nil {
		return domain.PermissionPackageApprovalRequest{}, fmt.Errorf("consume permission package approval request: %w", err)
	}
	return updated, nil
}

func (p *Postgres) EvaluateCapabilityAccess(ctx context.Context, req CapabilityAccessRequest) (domain.CapabilityAccessDecision, error) {
	capability, ok, err := p.GetCapability(ctx, req.CapabilityID)
	if err != nil {
		return domain.CapabilityAccessDecision{}, err
	}
	if !ok || capability.TargetID != req.TargetID {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "capability", Reason: "capability is not registered for target"}, nil
	}
	if capability.DiscoveryStatus != domain.CapabilityDiscoveryApproved {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "capability", CapabilityID: capability.ID, Reason: "capability is not approved"}, nil
	}
	entitlement, ok, err := p.matchTenantEntitlement(ctx, req, capability.ID)
	if err != nil {
		return domain.CapabilityAccessDecision{}, err
	}
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "tenant_entitlement", CapabilityID: capability.ID, Reason: "tenant has no entitlement for capability"}, nil
	}
	if entitlement.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "tenant_entitlement", CapabilityID: capability.ID, EntitlementID: entitlement.ID, Reason: "tenant entitlement denies capability"}, nil
	}
	entitlementScopes, ok := domain.EffectiveDataScopes(capability.DataScopes, entitlement.DataScopes)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "tenant_entitlement", CapabilityID: capability.ID, EntitlementID: entitlement.ID, Reason: "tenant entitlement data scopes exceed capability boundary"}, nil
	}
	workspaceAssignment, ok, err := p.matchWorkspaceAssignment(ctx, req, entitlement.ID)
	if err != nil {
		return domain.CapabilityAccessDecision{}, err
	}
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "workspace_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, Reason: "workspace has no assignment for capability"}, nil
	}
	if workspaceAssignment.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "workspace_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, Reason: "workspace assignment denies capability"}, nil
	}
	workspaceScopes, ok := domain.EffectiveDataScopes(entitlementScopes, workspaceAssignment.DataScopes)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "workspace_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, Reason: "workspace assignment data scopes exceed tenant entitlement boundary"}, nil
	}
	instanceAssignment, ok, err := p.matchInstanceAssignment(ctx, req, workspaceAssignment.ID)
	if err != nil {
		return domain.CapabilityAccessDecision{}, err
	}
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "instance_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, Reason: "caller instance has no assignment for capability"}, nil
	}
	if instanceAssignment.Effect == domain.PolicyEffectDeny {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "instance_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, InstanceAssignmentID: instanceAssignment.ID, Reason: "caller instance assignment denies capability"}, nil
	}
	instanceScopes, ok := domain.EffectiveDataScopes(workspaceScopes, instanceAssignment.DataScopes)
	if !ok {
		return domain.CapabilityAccessDecision{Allowed: false, Source: "instance_assignment", CapabilityID: capability.ID, EntitlementID: entitlement.ID, WorkspaceAssignmentID: workspaceAssignment.ID, InstanceAssignmentID: instanceAssignment.ID, Reason: "caller instance assignment data scopes exceed workspace assignment boundary"}, nil
	}
	return domain.CapabilityAccessDecision{
		Allowed:               true,
		Source:                "capability_governance",
		CapabilityID:          capability.ID,
		EntitlementID:         entitlement.ID,
		WorkspaceAssignmentID: workspaceAssignment.ID,
		InstanceAssignmentID:  instanceAssignment.ID,
		Reason:                "capability assignment matched",
		DataScopes:            instanceScopes,
	}, nil
}

func (p *Postgres) matchTenantEntitlement(ctx context.Context, req CapabilityAccessRequest, capabilityID string) (domain.TenantEntitlement, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, tenant_id, target_agent_id, capability_id, effect, data_scopes, status, priority, created_at, updated_at
		from tenant_entitlements
		where tenant_id=$1
			and target_agent_id=$2
			and capability_id=$3
			and status=$4
		order by priority desc,
			case when effect=$5 then 0 else 1 end asc,
			created_at asc,
			id asc
		limit 1
	`, req.TenantID, req.TargetID, capabilityID, string(domain.PolicyStatusEnabled), string(domain.PolicyEffectDeny))
	entitlement, err := scanTenantEntitlement(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.TenantEntitlement{}, false, nil
	}
	if err != nil {
		return domain.TenantEntitlement{}, false, fmt.Errorf("match tenant entitlement: %w", err)
	}
	return entitlement, true, nil
}

func (p *Postgres) matchWorkspaceAssignment(ctx context.Context, req CapabilityAccessRequest, entitlementID string) (domain.WorkspaceAssignment, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, tenant_entitlement_id, tenant_id, workspace_id, effect, data_scopes, status, created_at, updated_at
		from workspace_assignments
		where tenant_entitlement_id=$1
			and tenant_id=$2
			and workspace_id=$3
			and status=$4
		order by case when effect=$5 then 0 else 1 end asc, created_at asc, id asc
		limit 1
	`, entitlementID, req.TenantID, req.WorkspaceID, string(domain.PolicyStatusEnabled), string(domain.PolicyEffectDeny))
	assignment, err := scanWorkspaceAssignment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WorkspaceAssignment{}, false, nil
	}
	if err != nil {
		return domain.WorkspaceAssignment{}, false, fmt.Errorf("match workspace assignment: %w", err)
	}
	return assignment, true, nil
}

func (p *Postgres) matchInstanceAssignment(ctx context.Context, req CapabilityAccessRequest, workspaceAssignmentID string) (domain.InstanceAssignment, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, workspace_assignment_id, tenant_id, workspace_id, caller_instance_id, subject_selector,
			effect, data_scopes, status, created_at, updated_at
		from instance_assignments
		where workspace_assignment_id=$1
			and tenant_id=$2
			and workspace_id=$3
			and caller_instance_id=$4
			and status=$5
				and (
					subject_selector=''
					or subject_selector=$6
					or (right(subject_selector, 1)='*' and $6 like left(subject_selector, length(subject_selector)-1) || '%')
				)
			order by case when effect=$7 then 0 else 1 end asc, created_at asc, id asc
		limit 1
	`, workspaceAssignmentID, req.TenantID, req.WorkspaceID, req.CallerInstanceID,
		string(domain.PolicyStatusEnabled), req.SubjectID, string(domain.PolicyEffectDeny))
	assignment, err := scanInstanceAssignment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.InstanceAssignment{}, false, nil
	}
	if err != nil {
		return domain.InstanceAssignment{}, false, fmt.Errorf("match instance assignment: %w", err)
	}
	return assignment, true, nil
}

func (p *Postgres) CreateRoutePolicy(ctx context.Context, policy domain.RoutePolicy) (domain.RoutePolicy, error) {
	return p.createRoutePolicy(ctx, p.pool, policy)
}

func (p *Postgres) createRoutePolicy(ctx context.Context, exec sqlExecutor, policy domain.RoutePolicy) (domain.RoutePolicy, error) {
	retry, err := marshalRoutePolicyRetry(policy.Retry)
	if err != nil {
		return domain.RoutePolicy{}, err
	}
	_, err = exec.Exec(ctx, `
		insert into route_policies (
			id, tenant_id, workspace_id, name, caller_agent_id, target_agent_id,
			route_type, route_key, effect, status, priority, retry, created_at, updated_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, policy.ID, policy.TenantID, policy.WorkspaceID, policy.Name, policy.CallerID, policy.TargetID,
		policy.RouteType, policy.RouteKey, string(policy.Effect), string(policy.Status), policy.Priority,
		retry, policy.CreatedAt, policy.UpdatedAt)
	if err != nil {
		return domain.RoutePolicy{}, fmt.Errorf("insert route policy: %w", err)
	}
	return policy, nil
}

func (p *Postgres) ListRoutePolicies(ctx context.Context, scope ManagementScope) ([]domain.RoutePolicy, error) {
	query := `
		select rp.id, rp.tenant_id, rp.workspace_id, rp.name, rp.caller_agent_id, rp.target_agent_id,
			rp.route_type, rp.route_key, rp.effect, rp.status, rp.priority, rp.retry, rp.created_at, rp.updated_at
		from route_policies rp
		join agents c on c.id = rp.caller_agent_id
		join agents t on t.id = rp.target_agent_id
		where 1=1
			and c.tenant_id = rp.tenant_id
			and t.tenant_id = rp.tenant_id
			and c.workspace_id = rp.workspace_id
			and t.workspace_id = rp.workspace_id
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(scope.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, scope.TenantID)
		if err != nil {
			return nil, err
		}
		add("rp.tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(scope.WorkspaceID) != "" {
		add("rp.workspace_id=$%d", strings.TrimSpace(scope.WorkspaceID))
	}
	query += " order by rp.created_at asc, rp.id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list route policies: %w", err)
	}
	defer rows.Close()
	return scanRoutePolicies(rows)
}

func (p *Postgres) GetRoutePolicy(ctx context.Context, id string) (domain.RoutePolicy, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, tenant_id, workspace_id, name, caller_agent_id, target_agent_id,
			route_type, route_key, effect, status, priority, retry, created_at, updated_at
		from route_policies
		where id=$1
	`, id)
	policy, err := scanRoutePolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RoutePolicy{}, false, nil
	}
	if err != nil {
		return domain.RoutePolicy{}, false, fmt.Errorf("get route policy: %w", err)
	}
	return policy, true, nil
}

func (p *Postgres) UpdateRoutePolicy(ctx context.Context, policy domain.RoutePolicy) (domain.RoutePolicy, bool, error) {
	return p.updateRoutePolicy(ctx, p.pool, policy)
}

func (p *Postgres) updateRoutePolicy(ctx context.Context, exec sqlExecutor, policy domain.RoutePolicy) (domain.RoutePolicy, bool, error) {
	retry, err := marshalRoutePolicyRetry(policy.Retry)
	if err != nil {
		return domain.RoutePolicy{}, false, err
	}
	row := exec.QueryRow(ctx, `
		update route_policies
		set name=$2, route_type=$3, route_key=$4, effect=$5, status=$6, priority=$7, retry=$8, updated_at=$9
		where id=$1
		returning id, tenant_id, workspace_id, name, caller_agent_id, target_agent_id,
			route_type, route_key, effect, status, priority, retry, created_at, updated_at
	`, policy.ID, policy.Name, policy.RouteType, policy.RouteKey, string(policy.Effect),
		string(policy.Status), policy.Priority, retry, policy.UpdatedAt)
	updated, err := scanRoutePolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RoutePolicy{}, false, nil
	}
	if err != nil {
		return domain.RoutePolicy{}, false, fmt.Errorf("update route policy: %w", err)
	}
	return updated, true, nil
}

func (p *Postgres) DisableRoutePolicy(ctx context.Context, id string, now time.Time) (domain.RoutePolicy, bool, error) {
	return p.disableRoutePolicy(ctx, p.pool, id, now)
}

func (p *Postgres) disableRoutePolicy(ctx context.Context, exec sqlExecutor, id string, now time.Time) (domain.RoutePolicy, bool, error) {
	row := exec.QueryRow(ctx, `
		update route_policies
		set status=$2, updated_at=$3
		where id=$1
		returning id, tenant_id, workspace_id, name, caller_agent_id, target_agent_id,
			route_type, route_key, effect, status, priority, retry, created_at, updated_at
	`, id, string(domain.RoutePolicyStatusDisabled), now)
	policy, err := scanRoutePolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RoutePolicy{}, false, nil
	}
	if err != nil {
		return domain.RoutePolicy{}, false, fmt.Errorf("disable route policy: %w", err)
	}
	return policy, true, nil
}

func (p *Postgres) EvaluateRouteAccess(ctx context.Context, callerID string, targetID string, routeType string, routeKey string, now time.Time) (domain.RouteAccessDecision, error) {
	row := p.pool.QueryRow(ctx, `
		select route_policies.id, route_policies.tenant_id, route_policies.workspace_id, route_policies.name,
			route_policies.caller_agent_id, route_policies.target_agent_id,
			route_policies.route_type, route_policies.route_key, route_policies.effect,
			route_policies.status, route_policies.priority, route_policies.retry,
			route_policies.created_at, route_policies.updated_at
		from route_policies
		join agents c on c.id = route_policies.caller_agent_id
		join agents t on t.id = route_policies.target_agent_id
		where route_policies.caller_agent_id=$1
			and route_policies.target_agent_id=$2
			and route_policies.tenant_id = c.tenant_id
			and route_policies.workspace_id = c.workspace_id
			and t.tenant_id = c.tenant_id
			and t.workspace_id = c.workspace_id
			and route_policies.status=$3
			and (route_policies.route_type='' or route_policies.route_type=$4)
			and (route_policies.route_key='' or route_policies.route_key=$5)
		order by route_policies.priority desc,
			case when route_policies.effect=$6 then 0 else 1 end asc,
			route_policies.created_at asc,
			route_policies.id asc
		limit 1
	`, callerID, targetID, string(domain.RoutePolicyStatusEnabled), routeType, routeKey, string(domain.RoutePolicyEffectDeny))
	policy, err := scanRoutePolicy(row)
	if err == nil {
		return routePolicyDecision(policy), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.RouteAccessDecision{}, fmt.Errorf("evaluate route policy: %w", err)
	}
	if p.HasGrant(ctx, callerID, targetID, routeType, routeKey, now) {
		return domain.RouteAccessDecision{
			Allowed: true,
			Source:  "access_grant",
			Reason:  "access grant matched",
		}, nil
	}
	return domain.RouteAccessDecision{
		Allowed: false,
		Source:  "none",
		Reason:  "caller has no route policy or access grant for target route",
	}, nil
}

func (p *Postgres) AppendTrace(ctx context.Context, event domain.TraceEvent) (domain.TraceEvent, error) {
	dataScopes, err := json.Marshal(event.DataScopes)
	if err != nil {
		return domain.TraceEvent{}, fmt.Errorf("marshal trace data scopes: %w", err)
	}
	_, err = p.pool.Exec(ctx, `
		insert into trace_events (
			id, run_id, caller_agent_id, target_agent_id, route_type, route_key, decision, reason,
			duration_ms, upstream_attempts, upstream_status, upstream_error,
			tenant_id, workspace_id, caller_instance_id, subject_id,
			capability_id, capability_version, entitlement_id, workspace_assignment_id,
			instance_assignment_id, data_scopes, created_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
	`, event.ID, event.RunID, event.CallerID, event.TargetID, event.RouteType,
		event.RouteKey, string(event.Decision), event.Reason, event.DurationMs,
		event.UpstreamAttempts, event.UpstreamStatus, event.UpstreamError,
		event.TenantID, event.WorkspaceID, event.CallerInstanceID, event.SubjectID,
		event.CapabilityID, event.CapabilityVersion, event.EntitlementID,
		event.WorkspaceAssignmentID, event.InstanceAssignmentID, dataScopes, event.CreatedAt)
	if err != nil {
		return domain.TraceEvent{}, fmt.Errorf("insert trace event: %w", err)
	}
	return event, nil
}

func (p *Postgres) ListTraces(ctx context.Context, filter TraceFilter) ([]domain.TraceEvent, error) {
	query := `
		select id, run_id, caller_agent_id, target_agent_id, route_type, route_key, decision, reason,
			duration_ms, upstream_attempts, upstream_status, upstream_error,
			tenant_id, workspace_id, caller_instance_id, subject_id,
			capability_id, capability_version, entitlement_id, workspace_assignment_id,
			instance_assignment_id, data_scopes, created_at
		from trace_events
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if filter.RunID != "" {
		add("run_id=$%d", filter.RunID)
	}
	if filter.Decision != "" {
		add("decision=$%d", string(filter.Decision))
	}
	if filter.CallerID != "" {
		add("caller_agent_id=$%d", filter.CallerID)
	}
	if filter.TargetID != "" {
		add("target_agent_id=$%d", filter.TargetID)
	}
	if strings.TrimSpace(filter.TenantID) != "" || strings.TrimSpace(filter.WorkspaceID) != "" {
		var tenantIDs []string
		if strings.TrimSpace(filter.TenantID) != "" {
			var err error
			tenantIDs, err = p.tenantIDsForScope(ctx, filter.TenantID)
			if err != nil {
				return nil, err
			}
		}
		workspaceID := strings.TrimSpace(filter.WorkspaceID)
		tenantArgIndex := 0
		workspaceArgIndex := 0
		direct := []string{"(trace_events.tenant_id <> '' or trace_events.workspace_id <> '')"}
		caller := []string{}
		target := []string{}
		callerInstance := []string{}
		if tenantIDs != nil {
			args = append(args, tenantIDs)
			idx := len(args)
			tenantArgIndex = idx
			direct = append(direct, fmt.Sprintf("trace_events.tenant_id = any($%d)", idx))
			caller = append(caller, fmt.Sprintf("c.tenant_id = any($%d)", idx))
			target = append(target, fmt.Sprintf("t.tenant_id = any($%d)", idx))
			callerInstance = append(callerInstance, fmt.Sprintf("ci.tenant_id = any($%d)", idx))
		}
		if workspaceID != "" {
			args = append(args, workspaceID)
			idx := len(args)
			workspaceArgIndex = idx
			direct = append(direct, fmt.Sprintf("trace_events.workspace_id=$%d", idx))
			caller = append(caller, fmt.Sprintf("c.workspace_id=$%d", idx))
			target = append(target, fmt.Sprintf("t.workspace_id=$%d", idx))
			callerInstance = append(callerInstance, fmt.Sprintf("ci.workspace_id=$%d", idx))
		}
		scopedAgent := func(alias string) string {
			conditions := []string{}
			if tenantArgIndex > 0 {
				conditions = append(conditions, fmt.Sprintf("%s.tenant_id = any($%d)", alias, tenantArgIndex))
			}
			if workspaceArgIndex > 0 {
				conditions = append(conditions, fmt.Sprintf("%s.workspace_id=$%d", alias, workspaceArgIndex))
			}
			return strings.Join(conditions, " and ")
		}
		scopedTenantWorkspace := func(tenantColumn string, workspaceColumn string) string {
			conditions := []string{}
			if tenantArgIndex > 0 {
				conditions = append(conditions, fmt.Sprintf("%s = any($%d)", tenantColumn, tenantArgIndex))
			}
			if workspaceArgIndex > 0 {
				conditions = append(conditions, fmt.Sprintf("%s=$%d", workspaceColumn, workspaceArgIndex))
			}
			return strings.Join(conditions, " and ")
		}
		scopedTenant := func(tenantColumn string) string {
			if tenantArgIndex == 0 {
				return "true"
			}
			return fmt.Sprintf("%s = any($%d)", tenantColumn, tenantArgIndex)
		}
		query += fmt.Sprintf(`
			and (
				exists (
					select 1
					from agents c
					join agents t on t.id = trace_events.target_agent_id
					where c.id = trace_events.caller_agent_id
						and %s
						and %s
				)
				or (
					%s
					and (
						trace_events.caller_instance_id = ''
						or exists (
							select 1
							from agents ci
							where ci.id = trace_events.caller_instance_id
								and %s
						)
					)
					and (
						trace_events.target_agent_id = ''
						or trace_events.entitlement_id <> ''
						or trace_events.workspace_assignment_id <> ''
						or trace_events.instance_assignment_id <> ''
						or exists (
							select 1
							from agents dt
							where dt.id = trace_events.target_agent_id
								and (%s or (trace_events.workspace_id <> '' and dt.workspace_id = trace_events.workspace_id))
						)
					)
					and (
						trace_events.capability_id = ''
						or exists (
							select 1
							from capabilities dc
							join agents dct on dct.id = dc.target_agent_id
							where dc.id = trace_events.capability_id
								and (trace_events.target_agent_id = '' or dc.target_agent_id = trace_events.target_agent_id)
								and (
									trace_events.entitlement_id <> ''
									or trace_events.workspace_assignment_id <> ''
									or trace_events.instance_assignment_id <> ''
									or %s
									or (trace_events.workspace_id <> '' and dct.workspace_id = trace_events.workspace_id)
								)
						)
					)
					and (
						trace_events.entitlement_id = ''
						or exists (
							select 1
							from tenant_entitlements de
							join agents det on det.id = de.target_agent_id
							where de.id = trace_events.entitlement_id
								and (trace_events.target_agent_id = '' or de.target_agent_id = trace_events.target_agent_id)
								and (trace_events.capability_id = '' or de.capability_id = trace_events.capability_id)
								and %s
						)
					)
					and (
						trace_events.workspace_assignment_id = ''
						or exists (
							select 1
							from workspace_assignments dwa
							join tenant_entitlements dwe on dwe.id = dwa.tenant_entitlement_id
							join agents dwt on dwt.id = dwe.target_agent_id
							where dwa.id = trace_events.workspace_assignment_id
								and (trace_events.target_agent_id = '' or dwe.target_agent_id = trace_events.target_agent_id)
								and (trace_events.capability_id = '' or dwe.capability_id = trace_events.capability_id)
								and %s
						)
					)
					and (
						trace_events.instance_assignment_id = ''
						or exists (
							select 1
							from instance_assignments dia
							join agents dici on dici.id = dia.caller_instance_id
							join workspace_assignments diw on diw.id = dia.workspace_assignment_id
							join tenant_entitlements die on die.id = diw.tenant_entitlement_id
							join agents dit on dit.id = die.target_agent_id
							where dia.id = trace_events.instance_assignment_id
								and (trace_events.caller_instance_id = '' or dia.caller_instance_id = trace_events.caller_instance_id)
								and (trace_events.target_agent_id = '' or die.target_agent_id = trace_events.target_agent_id)
								and (trace_events.capability_id = '' or die.capability_id = trace_events.capability_id)
								and %s
								and %s
						)
					)
					and (
						trace_events.capability_id <> ''
						or trace_events.entitlement_id <> ''
						or trace_events.workspace_assignment_id <> ''
						or trace_events.instance_assignment_id <> ''
						or (trace_events.caller_agent_id = '' and trace_events.target_agent_id = '')
					)
				)
			)
		`,
			strings.Join(caller, " and "),
			strings.Join(target, " and "),
			strings.Join(direct, " and "),
			strings.Join(callerInstance, " and "),
			scopedAgent("dt"),
			scopedAgent("dct"),
			scopedTenant("de.tenant_id"),
			scopedTenantWorkspace("dwa.tenant_id", "dwa.workspace_id"),
			scopedTenantWorkspace("dia.tenant_id", "dia.workspace_id"),
			scopedAgent("dici"),
		)
	}
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query = fmt.Sprintf(`
			select *
			from (%s order by created_at desc, id desc limit $%d) recent_trace_events
			order by created_at asc, id asc
		`, query, len(args))
	} else {
		query += " order by created_at asc, id asc"
	}
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	defer rows.Close()
	return scanTraceEvents(rows)
}

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

func (p *Postgres) CreateAgentWithAudit(ctx context.Context, agent domain.Agent, build AgentAuditBuilder) (domain.Agent, error) {
	var created domain.Agent
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createAgent(ctx, tx, agent)
		if err != nil {
			return err
		}
		audit := build(created)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) UpdateAgentWithAudit(ctx context.Context, agent domain.Agent, build AgentAuditBuilder) (domain.Agent, bool, error) {
	var updated domain.Agent
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		updated, ok, err = p.updateAgent(ctx, tx, agent)
		if err != nil || !ok {
			return err
		}
		audit := build(updated)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return updated, ok, err
}

func (p *Postgres) RotateAgentCredentialsWithAudit(ctx context.Context, id string, credentials map[string]string, now time.Time, build AgentAuditBuilder) (domain.Agent, bool, error) {
	var updated domain.Agent
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		updated, ok, err = p.rotateAgentCredentials(ctx, tx, id, credentials, now)
		if err != nil || !ok {
			return err
		}
		audit := build(updated)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return updated, ok, err
}

func (p *Postgres) DisableAgentWithAudit(ctx context.Context, id string, now time.Time, build AgentAuditBuilder) (domain.Agent, bool, error) {
	var disabled domain.Agent
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		disabled, ok, err = p.disableAgent(ctx, tx, id, now)
		if err != nil || !ok {
			return err
		}
		audit := build(disabled)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return disabled, ok, err
}

func (p *Postgres) CreateAgentKeyWithAudit(ctx context.Context, key domain.AgentKey, build AgentKeyAuditBuilder) (domain.AgentKey, error) {
	var created domain.AgentKey
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createAgentKey(ctx, tx, key)
		if err != nil {
			return err
		}
		audit := build(created)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) CreateAccessHandoffAgentKeyWithAudit(ctx context.Context, key domain.AgentKey, now time.Time, build AgentKeyAuditBuilder) (domain.AgentKey, error) {
	var created domain.AgentKey
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1, 0))`, key.CreatedForHandoffID); err != nil {
			return fmt.Errorf("lock access handoff token issuance: %w", err)
		}
		var active bool
		if err := tx.QueryRow(ctx, `
			select exists(
				select 1 from agent_keys
				where created_for_handoff_id=$1 and revoked_at is null and expires_at > $2
			)
		`, key.CreatedForHandoffID, now).Scan(&active); err != nil {
			return fmt.Errorf("check active access handoff token: %w", err)
		}
		if active {
			return ErrActiveAccessHandoffToken
		}
		var err error
		created, err = p.createAgentKey(ctx, tx, key)
		if err != nil {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, build(created))
		return err
	})
	return created, err
}

func (p *Postgres) RevokeAgentKeyWithAudit(ctx context.Context, id string, now time.Time, build AgentKeyAuditBuilder) (domain.AgentKey, bool, error) {
	var revoked domain.AgentKey
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		revoked, ok, err = p.revokeAgentKey(ctx, tx, id, now)
		if err != nil || !ok {
			return err
		}
		audit := build(revoked)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return revoked, ok, err
}

func (p *Postgres) CreateAccessGrantWithAudit(ctx context.Context, grant domain.AccessGrant, build AccessGrantAuditBuilder) (domain.AccessGrant, error) {
	var created domain.AccessGrant
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createAccessGrant(ctx, tx, grant)
		if err != nil {
			return err
		}
		audit := build(created)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) RevokeAccessGrantWithAudit(ctx context.Context, id string, now time.Time, build AccessGrantAuditBuilder) (domain.AccessGrant, bool, error) {
	var revoked domain.AccessGrant
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		revoked, ok, err = p.revokeAccessGrant(ctx, tx, id, now)
		if err != nil || !ok {
			return err
		}
		audit := build(revoked)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return revoked, ok, err
}

func (p *Postgres) CreateRoutePolicyWithAudit(ctx context.Context, policy domain.RoutePolicy, build RoutePolicyAuditBuilder) (domain.RoutePolicy, error) {
	var created domain.RoutePolicy
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createRoutePolicy(ctx, tx, policy)
		if err != nil {
			return err
		}
		audit := build(created)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return created, err
}

func (p *Postgres) UpdateRoutePolicyWithAudit(ctx context.Context, policy domain.RoutePolicy, build RoutePolicyAuditBuilder) (domain.RoutePolicy, bool, error) {
	var updated domain.RoutePolicy
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		updated, ok, err = p.updateRoutePolicy(ctx, tx, policy)
		if err != nil || !ok {
			return err
		}
		audit := build(updated)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return updated, ok, err
}

func (p *Postgres) DisableRoutePolicyWithAudit(ctx context.Context, id string, now time.Time, build RoutePolicyAuditBuilder) (domain.RoutePolicy, bool, error) {
	var disabled domain.RoutePolicy
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		disabled, ok, err = p.disableRoutePolicy(ctx, tx, id, now)
		if err != nil || !ok {
			return err
		}
		audit := build(disabled)
		_, err = p.appendAuditEvent(ctx, tx, audit)
		return err
	})
	return disabled, ok, err
}

func (p *Postgres) ListAuditEvents(ctx context.Context, filter AuditEventFilter) ([]domain.AuditEvent, error) {
	query := `
		select id, tenant_id, workspace_id, actor, action, resource_type, resource_id, summary, metadata, created_at
		from audit_events
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		tenantIDs, err := p.tenantIDsForScope(ctx, filter.TenantID)
		if err != nil {
			return nil, err
		}
		add("tenant_id = any($%d)", tenantIDs)
	}
	if strings.TrimSpace(filter.WorkspaceID) != "" {
		add("workspace_id=$%d", strings.TrimSpace(filter.WorkspaceID))
	}
	if strings.TrimSpace(filter.Action) != "" {
		add("action=$%d", strings.TrimSpace(filter.Action))
	}
	if strings.TrimSpace(filter.ResourceType) != "" {
		add("resource_type=$%d", strings.TrimSpace(filter.ResourceType))
	}
	if strings.TrimSpace(filter.ResourceID) != "" {
		add("resource_id=$%d", strings.TrimSpace(filter.ResourceID))
	}
	query += " order by created_at asc, id asc"
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" limit $%d", len(args))
	}
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()
	return scanAuditEvents(rows)
}

const adminIdentityColumns = `
	id, actor, display_name, role, tenant_id, workspace_id, status, source,
	key_hash, key_prefix, created_at, updated_at, last_used_at, rotated_at,
	disabled_at, created_by, updated_by, disabled_by
`

func (p *Postgres) ListAdminIdentities(ctx context.Context) ([]domain.AdminIdentity, error) {
	rows, err := p.pool.Query(ctx, `
		select `+adminIdentityColumns+`
		from admin_identities
		order by created_at asc, id asc
	`)
	if err != nil {
		return nil, fmt.Errorf("list admin identities: %w", err)
	}
	defer rows.Close()
	return scanAdminIdentities(rows)
}

func (p *Postgres) GetAdminIdentity(ctx context.Context, id string) (domain.AdminIdentity, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select `+adminIdentityColumns+`
		from admin_identities
		where id=$1
	`, strings.TrimSpace(id))
	identity, err := scanAdminIdentity(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AdminIdentity{}, false, nil
	}
	if err != nil {
		return domain.AdminIdentity{}, false, fmt.Errorf("get admin identity: %w", err)
	}
	return identity, true, nil
}

func (p *Postgres) GetAdminIdentityByActor(ctx context.Context, actor string) (domain.AdminIdentity, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select `+adminIdentityColumns+`
		from admin_identities
		where actor=$1
	`, strings.TrimSpace(actor))
	identity, err := scanAdminIdentity(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AdminIdentity{}, false, nil
	}
	if err != nil {
		return domain.AdminIdentity{}, false, fmt.Errorf("get admin identity by actor: %w", err)
	}
	return identity, true, nil
}

func (p *Postgres) FindAdminIdentityByKeyHash(ctx context.Context, hash string) (domain.AdminIdentity, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select `+adminIdentityColumns+`
		from admin_identities
		where key_hash=$1 and status=$2
	`, strings.TrimSpace(hash), string(domain.AdminIdentityStatusActive))
	identity, err := scanAdminIdentity(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AdminIdentity{}, false, nil
	}
	if err != nil {
		return domain.AdminIdentity{}, false, fmt.Errorf("find admin identity by key hash: %w", err)
	}
	return identity, true, nil
}

func (p *Postgres) CreateAdminIdentityWithAudit(ctx context.Context, identity domain.AdminIdentity, build AdminIdentityAuditBuilder) (domain.AdminIdentity, error) {
	var created domain.AdminIdentity
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		created, err = p.createAdminIdentity(ctx, tx, identity)
		if err != nil {
			return err
		}
		_, err = p.appendAuditEvent(ctx, tx, build(created))
		return err
	})
	return created, err
}

func (p *Postgres) createAdminIdentity(ctx context.Context, exec sqlExecutor, identity domain.AdminIdentity) (domain.AdminIdentity, error) {
	_, err := exec.Exec(ctx, `
		insert into admin_identities (
			id, actor, display_name, role, tenant_id, workspace_id, status, source,
			key_hash, key_prefix, created_at, updated_at, last_used_at, rotated_at,
			disabled_at, created_by, updated_by, disabled_by
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`, identity.ID, identity.Actor, identity.DisplayName, string(identity.Role), identity.TenantID, identity.WorkspaceID,
		string(identity.Status), string(identity.Source), identity.KeyHash, identity.KeyPrefix, identity.CreatedAt, identity.UpdatedAt,
		nullTime(identity.LastUsedAt), nullTime(identity.RotatedAt), nullTime(identity.DisabledAt),
		identity.CreatedBy, identity.UpdatedBy, identity.DisabledBy)
	if err != nil {
		if conflict := adminIdentityInsertConflict(err); conflict != nil {
			return domain.AdminIdentity{}, conflict
		}
		return domain.AdminIdentity{}, fmt.Errorf("insert admin identity: %w", err)
	}
	return identity, nil
}

func adminIdentityInsertConflict(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return nil
	}
	switch pgErr.ConstraintName {
	case "admin_identities_actor_key":
		return domain.Conflict("ADMIN_IDENTITY_ACTOR_EXISTS", "admin identity actor already exists")
	case "admin_identities_pkey":
		return domain.Conflict("ADMIN_IDENTITY_EXISTS", "admin identity already exists")
	default:
		return nil
	}
}

func (p *Postgres) RotateAdminIdentityKeyWithAudit(ctx context.Context, id string, keyHash string, keyPrefix string, now time.Time, actor string, build AdminIdentityAuditBuilder) (domain.AdminIdentity, bool, error) {
	var updated domain.AdminIdentity
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			update admin_identities
			set key_hash=$2, key_prefix=$3, rotated_at=$4, updated_at=$4, updated_by=$5
			where id=$1
			returning `+adminIdentityColumns+`
		`, strings.TrimSpace(id), strings.TrimSpace(keyHash), strings.TrimSpace(keyPrefix), now, strings.TrimSpace(actor))
		var err error
		updated, err = scanAdminIdentity(row)
		if errors.Is(err, pgx.ErrNoRows) {
			ok = false
			return nil
		}
		if err != nil {
			return fmt.Errorf("rotate admin identity key: %w", err)
		}
		ok = true
		_, err = p.appendAuditEvent(ctx, tx, build(updated))
		return err
	})
	return updated, ok, err
}

func (p *Postgres) DisableAdminIdentityWithAudit(ctx context.Context, id string, now time.Time, actor string, build AdminIdentityAuditBuilder) (domain.AdminIdentity, bool, error) {
	var disabled domain.AdminIdentity
	var ok bool
	err := p.withTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			update admin_identities
			set status=$2, disabled_at=$3, updated_at=$3, updated_by=$4, disabled_by=$4
			where id=$1
			returning `+adminIdentityColumns+`
		`, strings.TrimSpace(id), string(domain.AdminIdentityStatusDisabled), now, strings.TrimSpace(actor))
		var err error
		disabled, err = scanAdminIdentity(row)
		if errors.Is(err, pgx.ErrNoRows) {
			ok = false
			return nil
		}
		if err != nil {
			return fmt.Errorf("disable admin identity: %w", err)
		}
		ok = true
		_, err = p.appendAuditEvent(ctx, tx, build(disabled))
		return err
	})
	return disabled, ok, err
}

func (p *Postgres) TouchAdminIdentityLastUsed(ctx context.Context, id string, now time.Time) error {
	_, err := p.pool.Exec(ctx, `
		update admin_identities
		set last_used_at=$2, updated_at=$2
		where id=$1
	`, strings.TrimSpace(id), now)
	if err != nil {
		return fmt.Errorf("touch admin identity last used: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTenants(rows pgx.Rows) ([]domain.Tenant, error) {
	var out []domain.Tenant
	for rows.Next() {
		tenant, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, tenant)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanTenant(row scanner) (domain.Tenant, error) {
	var tenant domain.Tenant
	var status string
	if err := row.Scan(&tenant.ID, &tenant.ParentTenantID, &tenant.Level, &tenant.Name, &status, &tenant.CreatedAt, &tenant.UpdatedAt); err != nil {
		return domain.Tenant{}, err
	}
	tenant.Status = domain.TenantStatus(status)
	return tenant, nil
}

func (p *Postgres) scanAgents(rows pgx.Rows) ([]domain.Agent, error) {
	var out []domain.Agent
	for rows.Next() {
		agent, err := p.scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, agent)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (p *Postgres) scanAgent(row scanner) (domain.Agent, error) {
	var agent domain.Agent
	var status string
	var config []byte
	var credentialsCiphertext []byte
	if err := row.Scan(&agent.ID, &agent.TenantID, &agent.WorkspaceID, &agent.Name, &agent.Description,
		&agent.OwnerID, &agent.ChannelType, &config, &credentialsCiphertext, &agent.CredentialVersion, &status, &agent.CreatedAt, &agent.UpdatedAt); err != nil {
		return domain.Agent{}, err
	}
	if len(config) > 0 {
		if err := json.Unmarshal(config, &agent.ChannelConfig); err != nil {
			return domain.Agent{}, fmt.Errorf("unmarshal channel config: %w", err)
		}
	}
	if agent.ChannelConfig == nil {
		agent.ChannelConfig = map[string]any{}
	}
	credentials, err := security.DecryptCredentials(credentialsCiphertext, p.credentialKey)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("decrypt credentials: %w", err)
	}
	agent.Credentials = credentials
	agent.Status = domain.AgentStatus(status)
	return agent, nil
}

func scanAgentKeys(rows pgx.Rows) ([]domain.AgentKey, error) {
	var out []domain.AgentKey
	for rows.Next() {
		key, err := scanAgentKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanAgentKey(row scanner) (domain.AgentKey, error) {
	var key domain.AgentKey
	var revokedAt *time.Time
	if err := row.Scan(&key.ID, &key.AgentID, &key.Name, &key.Hash, &key.Prefix,
		&key.ApplicationID, &key.TemplateID, &key.SubjectSelector, &key.CreatedForHandoffID,
		&key.CreatedAt, &key.ExpiresAt, &revokedAt); err != nil {
		return domain.AgentKey{}, err
	}
	if revokedAt != nil {
		key.RevokedAt = *revokedAt
	}
	return key, nil
}

func scanAccessGrants(rows pgx.Rows) ([]domain.AccessGrant, error) {
	var out []domain.AccessGrant
	for rows.Next() {
		grant, err := scanAccessGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanAccessGrant(row scanner) (domain.AccessGrant, error) {
	var grant domain.AccessGrant
	var expiresAt *time.Time
	var revokedAt *time.Time
	if err := row.Scan(&grant.ID, &grant.CallerID, &grant.TargetID, &grant.RouteType, &grant.RouteKey,
		&grant.CreatedAt, &expiresAt, &revokedAt); err != nil {
		return domain.AccessGrant{}, err
	}
	if expiresAt != nil {
		grant.ExpiresAt = *expiresAt
	}
	if revokedAt != nil {
		grant.RevokedAt = *revokedAt
	}
	return grant, nil
}

func scanCapabilities(rows pgx.Rows) ([]domain.Capability, error) {
	var out []domain.Capability
	for rows.Next() {
		capability, err := scanCapability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, capability)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanCapability(row scanner) (domain.Capability, error) {
	var capability domain.Capability
	var typ string
	var action string
	var inputSchema []byte
	var outputSchema []byte
	var nativeScopes []byte
	var dataDomains []byte
	var dataScopes []byte
	var sensitivity string
	var riskLevel string
	var enforcementMode string
	var discoveryStatus string
	if err := row.Scan(&capability.ID, &capability.TargetID, &typ, &capability.Key, &capability.DisplayName,
		&capability.Description, &action, &inputSchema, &outputSchema, &nativeScopes, &dataDomains,
		&dataScopes, &sensitivity, &riskLevel, &enforcementMode, &discoveryStatus,
		&capability.Version, &capability.DiscoveredAt, &capability.UpdatedAt); err != nil {
		return domain.Capability{}, err
	}
	capability.Type = domain.CapabilityType(typ)
	capability.Action = domain.CapabilityAction(action)
	capability.Sensitivity = domain.CapabilitySensitivity(sensitivity)
	capability.RiskLevel = domain.CapabilityRisk(riskLevel)
	capability.EnforcementMode = domain.CapabilityEnforcementMode(enforcementMode)
	capability.DiscoveryStatus = domain.CapabilityDiscoveryStatus(discoveryStatus)
	if err := unmarshalJSON(inputSchema, &capability.InputSchema, "capability input schema"); err != nil {
		return domain.Capability{}, err
	}
	if err := unmarshalJSON(outputSchema, &capability.OutputSchema, "capability output schema"); err != nil {
		return domain.Capability{}, err
	}
	if err := unmarshalJSON(nativeScopes, &capability.NativeScopes, "capability native scopes"); err != nil {
		return domain.Capability{}, err
	}
	if err := unmarshalJSON(dataDomains, &capability.DataDomains, "capability data domains"); err != nil {
		return domain.Capability{}, err
	}
	if err := unmarshalJSON(dataScopes, &capability.DataScopes, "capability data scopes"); err != nil {
		return domain.Capability{}, err
	}
	if capability.InputSchema == nil {
		capability.InputSchema = map[string]any{}
	}
	if capability.OutputSchema == nil {
		capability.OutputSchema = map[string]any{}
	}
	return capability, nil
}

func scanTenantEntitlements(rows pgx.Rows) ([]domain.TenantEntitlement, error) {
	var out []domain.TenantEntitlement
	for rows.Next() {
		entitlement, err := scanTenantEntitlement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, entitlement)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanTenantEntitlement(row scanner) (domain.TenantEntitlement, error) {
	var entitlement domain.TenantEntitlement
	var effect string
	var dataScopes []byte
	var status string
	if err := row.Scan(&entitlement.ID, &entitlement.TenantID, &entitlement.TargetID,
		&entitlement.CapabilityID, &effect, &dataScopes, &status, &entitlement.Priority,
		&entitlement.CreatedAt, &entitlement.UpdatedAt); err != nil {
		return domain.TenantEntitlement{}, err
	}
	entitlement.Effect = domain.PolicyEffect(effect)
	entitlement.Status = domain.PolicyStatus(status)
	if err := unmarshalJSON(dataScopes, &entitlement.DataScopes, "tenant entitlement data scopes"); err != nil {
		return domain.TenantEntitlement{}, err
	}
	return entitlement, nil
}

func scanWorkspaceAssignments(rows pgx.Rows) ([]domain.WorkspaceAssignment, error) {
	var out []domain.WorkspaceAssignment
	for rows.Next() {
		assignment, err := scanWorkspaceAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanWorkspaceAssignment(row scanner) (domain.WorkspaceAssignment, error) {
	var assignment domain.WorkspaceAssignment
	var effect string
	var dataScopes []byte
	var status string
	if err := row.Scan(&assignment.ID, &assignment.TenantEntitlementID, &assignment.TenantID,
		&assignment.WorkspaceID, &effect, &dataScopes, &status, &assignment.CreatedAt,
		&assignment.UpdatedAt); err != nil {
		return domain.WorkspaceAssignment{}, err
	}
	assignment.Effect = domain.PolicyEffect(effect)
	assignment.Status = domain.PolicyStatus(status)
	if err := unmarshalJSON(dataScopes, &assignment.DataScopes, "workspace assignment data scopes"); err != nil {
		return domain.WorkspaceAssignment{}, err
	}
	return assignment, nil
}

func scanInstanceAssignments(rows pgx.Rows) ([]domain.InstanceAssignment, error) {
	var out []domain.InstanceAssignment
	for rows.Next() {
		assignment, err := scanInstanceAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanInstanceAssignment(row scanner) (domain.InstanceAssignment, error) {
	var assignment domain.InstanceAssignment
	var effect string
	var dataScopes []byte
	var status string
	if err := row.Scan(&assignment.ID, &assignment.WorkspaceAssignmentID, &assignment.TenantID,
		&assignment.WorkspaceID, &assignment.CallerInstanceID, &assignment.SubjectSelector,
		&effect, &dataScopes, &status, &assignment.CreatedAt, &assignment.UpdatedAt); err != nil {
		return domain.InstanceAssignment{}, err
	}
	assignment.Effect = domain.PolicyEffect(effect)
	assignment.Status = domain.PolicyStatus(status)
	if err := unmarshalJSON(dataScopes, &assignment.DataScopes, "instance assignment data scopes"); err != nil {
		return domain.InstanceAssignment{}, err
	}
	return assignment, nil
}

func scanPermissionPackageApplications(rows pgx.Rows) ([]domain.PermissionPackageApplication, error) {
	var out []domain.PermissionPackageApplication
	for rows.Next() {
		application, err := scanPermissionPackageApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, application)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanPermissionPackageApplication(row scanner) (domain.PermissionPackageApplication, error) {
	var application domain.PermissionPackageApplication
	var dataScopes []byte
	var allowedCapabilityIDs []byte
	var allowedCapabilityKeys []byte
	var tenantEntitlementIDs []byte
	var workspaceAssignmentIDs []byte
	var instanceAssignmentIDs []byte
	if err := row.Scan(&application.ID, &application.DraftID, &application.TemplateID,
		&application.TemplateVersion, &application.TenantID, &application.WorkspaceID,
		&application.TargetID, &application.CallerInstanceID, &application.SubjectSelector,
		&application.RequestText, &application.Region, &dataScopes, &allowedCapabilityIDs,
		&allowedCapabilityKeys, &tenantEntitlementIDs, &workspaceAssignmentIDs,
		&instanceAssignmentIDs, &application.AppliedAt, &application.RequestedCapabilityID); err != nil {
		return domain.PermissionPackageApplication{}, err
	}
	if err := unmarshalJSON(dataScopes, &application.DataScopes, "permission package application data scopes"); err != nil {
		return domain.PermissionPackageApplication{}, err
	}
	if err := unmarshalJSON(allowedCapabilityIDs, &application.AllowedCapabilityIDs, "permission package application capability ids"); err != nil {
		return domain.PermissionPackageApplication{}, err
	}
	if err := unmarshalJSON(allowedCapabilityKeys, &application.AllowedCapabilityKeys, "permission package application capability keys"); err != nil {
		return domain.PermissionPackageApplication{}, err
	}
	if err := unmarshalJSON(tenantEntitlementIDs, &application.TenantEntitlementIDs, "permission package application entitlement ids"); err != nil {
		return domain.PermissionPackageApplication{}, err
	}
	if err := unmarshalJSON(workspaceAssignmentIDs, &application.WorkspaceAssignmentIDs, "permission package application workspace assignment ids"); err != nil {
		return domain.PermissionPackageApplication{}, err
	}
	if err := unmarshalJSON(instanceAssignmentIDs, &application.InstanceAssignmentIDs, "permission package application instance assignment ids"); err != nil {
		return domain.PermissionPackageApplication{}, err
	}
	return application, nil
}

func scanPermissionPackageApprovalRequests(rows pgx.Rows) ([]domain.PermissionPackageApprovalRequest, error) {
	var out []domain.PermissionPackageApprovalRequest
	for rows.Next() {
		request, err := scanPermissionPackageApprovalRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, request)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanPermissionPackageApprovalRequest(row scanner) (domain.PermissionPackageApprovalRequest, error) {
	var request domain.PermissionPackageApprovalRequest
	var dataScopes []byte
	var allowedCapabilityIDs []byte
	var allowedCapabilityKeys []byte
	var allowedCapabilityFingerprints []byte
	var policyGate []byte
	var status string
	var resolvedAt *time.Time
	var consumedAt *time.Time
	if err := row.Scan(&request.ID, &request.DraftID, &request.TemplateID,
		&request.TemplateVersion, &request.PolicyVersion, &request.TenantID,
		&request.WorkspaceID, &request.TargetID, &request.CallerInstanceID,
		&request.SubjectSelector, &request.RequestText, &request.Region, &dataScopes,
		&allowedCapabilityIDs, &allowedCapabilityKeys, &allowedCapabilityFingerprints, &policyGate, &status,
		&request.RequestedBy, &request.ReviewedBy, &request.ReviewComment,
		&request.CreatedAt, &request.UpdatedAt, &resolvedAt, &request.ExpiresAt,
		&consumedAt, &request.ConsumedByApplicationID, &request.RequestedCapabilityID); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	request.Status = domain.PermissionPackageApprovalStatus(status)
	if resolvedAt != nil {
		request.ResolvedAt = *resolvedAt
	}
	if consumedAt != nil {
		request.ConsumedAt = *consumedAt
	}
	if err := unmarshalJSON(dataScopes, &request.DataScopes, "permission package approval request data scopes"); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if err := unmarshalJSON(allowedCapabilityIDs, &request.AllowedCapabilityIDs, "permission package approval request capability ids"); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if err := unmarshalJSON(allowedCapabilityKeys, &request.AllowedCapabilityKeys, "permission package approval request capability keys"); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if err := unmarshalJSON(allowedCapabilityFingerprints, &request.AllowedCapabilityFingerprints, "permission package approval request capability fingerprints"); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	if err := unmarshalJSON(policyGate, &request.PolicyGate, "permission package approval request policy gate"); err != nil {
		return domain.PermissionPackageApprovalRequest{}, err
	}
	return request, nil
}

func marshalPermissionPackageApprovalRequestPayloads(request domain.PermissionPackageApprovalRequest) ([]byte, []byte, []byte, []byte, []byte, error) {
	dataScopes, err := json.Marshal(request.DataScopes)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("marshal permission package approval request data scopes: %w", err)
	}
	allowedCapabilityIDs, err := json.Marshal(request.AllowedCapabilityIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("marshal permission package approval request capability ids: %w", err)
	}
	allowedCapabilityKeys, err := json.Marshal(request.AllowedCapabilityKeys)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("marshal permission package approval request capability keys: %w", err)
	}
	allowedCapabilityFingerprints, err := json.Marshal(request.AllowedCapabilityFingerprints)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("marshal permission package approval request capability fingerprints: %w", err)
	}
	policyGate, err := json.Marshal(request.PolicyGate)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("marshal permission package approval request policy gate: %w", err)
	}
	return dataScopes, allowedCapabilityIDs, allowedCapabilityKeys, allowedCapabilityFingerprints, policyGate, nil
}

func scanRoutePolicies(rows pgx.Rows) ([]domain.RoutePolicy, error) {
	var out []domain.RoutePolicy
	for rows.Next() {
		policy, err := scanRoutePolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanRoutePolicy(row scanner) (domain.RoutePolicy, error) {
	var policy domain.RoutePolicy
	var effect string
	var status string
	var retry []byte
	if err := row.Scan(&policy.ID, &policy.TenantID, &policy.WorkspaceID, &policy.Name,
		&policy.CallerID, &policy.TargetID, &policy.RouteType, &policy.RouteKey, &effect,
		&status, &policy.Priority, &retry, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
		return domain.RoutePolicy{}, err
	}
	policy.Effect = domain.RoutePolicyEffect(effect)
	policy.Status = domain.RoutePolicyStatus(status)
	if len(retry) > 0 {
		var parsed domain.RoutePolicyRetry
		if err := json.Unmarshal(retry, &parsed); err != nil {
			return domain.RoutePolicy{}, fmt.Errorf("unmarshal route policy retry: %w", err)
		}
		policy.Retry = &parsed
	}
	return policy, nil
}

func marshalRoutePolicyRetry(retry *domain.RoutePolicyRetry) ([]byte, error) {
	if retry == nil {
		return nil, nil
	}
	data, err := json.Marshal(retry)
	if err != nil {
		return nil, fmt.Errorf("marshal route policy retry: %w", err)
	}
	return data, nil
}

func scanTraceEvents(rows pgx.Rows) ([]domain.TraceEvent, error) {
	var out []domain.TraceEvent
	for rows.Next() {
		var trace domain.TraceEvent
		var decision string
		var dataScopes []byte
		if err := rows.Scan(&trace.ID, &trace.RunID, &trace.CallerID, &trace.TargetID, &trace.RouteType,
			&trace.RouteKey, &decision, &trace.Reason, &trace.DurationMs, &trace.UpstreamAttempts,
			&trace.UpstreamStatus, &trace.UpstreamError, &trace.TenantID, &trace.WorkspaceID,
			&trace.CallerInstanceID, &trace.SubjectID, &trace.CapabilityID, &trace.CapabilityVersion,
			&trace.EntitlementID, &trace.WorkspaceAssignmentID, &trace.InstanceAssignmentID,
			&dataScopes, &trace.CreatedAt); err != nil {
			return nil, err
		}
		trace.Decision = domain.TraceDecision(decision)
		if err := unmarshalJSON(dataScopes, &trace.DataScopes, "trace data scopes"); err != nil {
			return nil, err
		}
		out = append(out, trace)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanAuditEvents(rows pgx.Rows) ([]domain.AuditEvent, error) {
	var out []domain.AuditEvent
	for rows.Next() {
		var event domain.AuditEvent
		var metadata []byte
		if err := rows.Scan(&event.ID, &event.TenantID, &event.WorkspaceID, &event.Actor, &event.Action,
			&event.ResourceType, &event.ResourceID, &event.Summary, &metadata, &event.CreatedAt); err != nil {
			return nil, err
		}
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal audit metadata: %w", err)
			}
		}
		if event.Metadata == nil {
			event.Metadata = map[string]any{}
		}
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanAdminIdentities(rows pgx.Rows) ([]domain.AdminIdentity, error) {
	var out []domain.AdminIdentity
	for rows.Next() {
		identity, err := scanAdminIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, identity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanAdminIdentity(row scanner) (domain.AdminIdentity, error) {
	var identity domain.AdminIdentity
	var role string
	var status string
	var source string
	var lastUsedAt *time.Time
	var rotatedAt *time.Time
	var disabledAt *time.Time
	if err := row.Scan(
		&identity.ID,
		&identity.Actor,
		&identity.DisplayName,
		&role,
		&identity.TenantID,
		&identity.WorkspaceID,
		&status,
		&source,
		&identity.KeyHash,
		&identity.KeyPrefix,
		&identity.CreatedAt,
		&identity.UpdatedAt,
		&lastUsedAt,
		&rotatedAt,
		&disabledAt,
		&identity.CreatedBy,
		&identity.UpdatedBy,
		&identity.DisabledBy,
	); err != nil {
		return domain.AdminIdentity{}, err
	}
	identity.Role = domain.AdminIdentityRole(role)
	identity.Status = domain.AdminIdentityStatus(status)
	identity.Source = domain.AdminIdentitySource(source)
	if lastUsedAt != nil {
		identity.LastUsedAt = *lastUsedAt
	}
	if rotatedAt != nil {
		identity.RotatedAt = *rotatedAt
	}
	if disabledAt != nil {
		identity.DisabledAt = *disabledAt
	}
	return identity, nil
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func unmarshalJSON(data []byte, target any, label string) error {
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal %s: %w", label, err)
	}
	return nil
}
