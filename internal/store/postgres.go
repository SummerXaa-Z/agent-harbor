package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) CreateAgent(ctx context.Context, agent domain.Agent) (domain.Agent, error) {
	config, err := json.Marshal(agent.ChannelConfig)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("marshal channel config: %w", err)
	}
	_, err = p.pool.Exec(ctx, `
		insert into agents (
			id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, status, created_at, updated_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, agent.ID, agent.TenantID, agent.WorkspaceID, agent.Name, agent.Description, agent.OwnerID,
		agent.ChannelType, config, string(agent.Status), agent.CreatedAt, agent.UpdatedAt)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("insert agent: %w", err)
	}
	return agent, nil
}

func (p *Postgres) ListAgents(ctx context.Context, filter AgentFilter) ([]domain.Agent, error) {
	query := `
		select id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, status, created_at, updated_at
		from agents
		where 1=1
	`
	args := []any{}
	add := func(sql string, value any) {
		args = append(args, value)
		query += fmt.Sprintf(" and "+sql, len(args))
	}
	if strings.TrimSpace(filter.TenantID) != "" {
		add("tenant_id=$%d", filter.TenantID)
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
	return scanAgents(rows)
}

func (p *Postgres) GetAgent(ctx context.Context, id string) (domain.Agent, bool, error) {
	row := p.pool.QueryRow(ctx, `
		select id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, status, created_at, updated_at
		from agents
		where id=$1
	`, id)
	agent, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, false, nil
	}
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("get agent: %w", err)
	}
	return agent, true, nil
}

func (p *Postgres) DisableAgent(ctx context.Context, id string, now time.Time) (domain.Agent, bool, error) {
	row := p.pool.QueryRow(ctx, `
		update agents
		set status=$2, updated_at=$3
		where id=$1
		returning id, tenant_id, workspace_id, name, description, owner_id,
			channel_type, channel_config, status, created_at, updated_at
	`, id, string(domain.AgentStatusDisabled), now)
	agent, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, false, nil
	}
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("disable agent: %w", err)
	}
	return agent, true, nil
}

func (p *Postgres) CreateAgentKey(ctx context.Context, key domain.AgentKey) (domain.AgentKey, error) {
	_, err := p.pool.Exec(ctx, `
		insert into agent_keys (id, agent_id, name, hash, prefix, created_at, expires_at, revoked_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8)
	`, key.ID, key.AgentID, key.Name, key.Hash, key.Prefix, key.CreatedAt, key.ExpiresAt, nullTime(key.RevokedAt))
	if err != nil {
		return domain.AgentKey{}, fmt.Errorf("insert agent key: %w", err)
	}
	return key, nil
}

func (p *Postgres) ListAgentKeys(ctx context.Context, scope ManagementScope) ([]domain.AgentKey, error) {
	query := `
		select k.id, k.agent_id, k.name, k.hash, k.prefix, k.created_at, k.expires_at, k.revoked_at
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
		add("a.tenant_id=$%d", scope.TenantID)
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
	row := p.pool.QueryRow(ctx, `
		update agent_keys
		set revoked_at=$2
		where id=$1
		returning id, agent_id, name, hash, prefix, created_at, expires_at, revoked_at
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
			a.channel_type, a.channel_config, a.status, a.created_at, a.updated_at
		from agent_keys k
		join agents a on a.id = k.agent_id
		where k.hash=$1 and k.revoked_at is null and k.expires_at > $2 and a.status=$3
	`, hash, now, string(domain.AgentStatusActive))
	agent, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, false, nil
	}
	if err != nil {
		return domain.Agent{}, false, fmt.Errorf("find agent by key: %w", err)
	}
	return agent, true, nil
}

func (p *Postgres) CreateAccessGrant(ctx context.Context, grant domain.AccessGrant) (domain.AccessGrant, error) {
	_, err := p.pool.Exec(ctx, `
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
	rows, err := p.pool.Query(ctx, `
		select g.id, g.caller_agent_id, g.target_agent_id, g.route_type, g.route_key, g.created_at, g.expires_at, g.revoked_at
		from access_grants g
		join agents c on c.id = g.caller_agent_id
		join agents t on t.id = g.target_agent_id
		where (
			($1 = '' and $2 = '')
			or (($1 = '' or c.tenant_id = $1) and ($2 = '' or c.workspace_id = $2))
			or (($1 = '' or t.tenant_id = $1) and ($2 = '' or t.workspace_id = $2))
		)
		order by g.created_at asc, g.id asc
	`, strings.TrimSpace(scope.TenantID), strings.TrimSpace(scope.WorkspaceID))
	if err != nil {
		return nil, fmt.Errorf("list access grants: %w", err)
	}
	defer rows.Close()
	return scanAccessGrants(rows)
}

func (p *Postgres) RevokeAccessGrant(ctx context.Context, id string, now time.Time) (domain.AccessGrant, bool, error) {
	row := p.pool.QueryRow(ctx, `
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
			where caller_agent_id=$1
				and target_agent_id=$2
				and revoked_at is null
				and (expires_at is null or expires_at > $5)
				and (route_type='' or route_type=$3)
				and (route_key='' or lower(route_key)=lower($4))
		)
	`, callerID, targetID, routeType, routeKey, now).Scan(&exists)
	return err == nil && exists
}

func (p *Postgres) AppendTrace(ctx context.Context, event domain.TraceEvent) (domain.TraceEvent, error) {
	_, err := p.pool.Exec(ctx, `
		insert into trace_events (
			id, run_id, caller_agent_id, target_agent_id, route_type, route_key, decision, reason, created_at
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, event.ID, event.RunID, event.CallerID, event.TargetID, event.RouteType,
		event.RouteKey, string(event.Decision), event.Reason, event.CreatedAt)
	if err != nil {
		return domain.TraceEvent{}, fmt.Errorf("insert trace event: %w", err)
	}
	return event, nil
}

func (p *Postgres) ListTraces(ctx context.Context, filter TraceFilter) ([]domain.TraceEvent, error) {
	query := `
		select id, run_id, caller_agent_id, target_agent_id, route_type, route_key, decision, reason, created_at
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
		args = append(args, strings.TrimSpace(filter.TenantID), strings.TrimSpace(filter.WorkspaceID))
		tenantIndex := len(args) - 1
		workspaceIndex := len(args)
		query += fmt.Sprintf(`
			and exists (
				select 1
				from agents scoped
				where scoped.id in (trace_events.caller_agent_id, trace_events.target_agent_id)
					and ($%d = '' or scoped.tenant_id = $%d)
					and ($%d = '' or scoped.workspace_id = $%d)
			)
		`, tenantIndex, tenantIndex, workspaceIndex, workspaceIndex)
	}
	query += " order by created_at asc, id asc"
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	defer rows.Close()
	return scanTraceEvents(rows)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAgents(rows pgx.Rows) ([]domain.Agent, error) {
	var out []domain.Agent
	for rows.Next() {
		agent, err := scanAgent(rows)
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

func scanAgent(row scanner) (domain.Agent, error) {
	var agent domain.Agent
	var status string
	var config []byte
	if err := row.Scan(&agent.ID, &agent.TenantID, &agent.WorkspaceID, &agent.Name, &agent.Description,
		&agent.OwnerID, &agent.ChannelType, &config, &status, &agent.CreatedAt, &agent.UpdatedAt); err != nil {
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

func scanTraceEvents(rows pgx.Rows) ([]domain.TraceEvent, error) {
	var out []domain.TraceEvent
	for rows.Next() {
		var trace domain.TraceEvent
		var decision string
		if err := rows.Scan(&trace.ID, &trace.RunID, &trace.CallerID, &trace.TargetID, &trace.RouteType,
			&trace.RouteKey, &decision, &trace.Reason, &trace.CreatedAt); err != nil {
			return nil, err
		}
		trace.Decision = domain.TraceDecision(decision)
		out = append(out, trace)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
