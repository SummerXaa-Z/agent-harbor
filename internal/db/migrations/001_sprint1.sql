create table if not exists agents (
	id text primary key,
	tenant_id text not null,
	workspace_id text not null,
	name text not null,
	description text not null default '',
	owner_id text not null default '',
	channel_type text not null,
	channel_config jsonb not null default '{}'::jsonb,
	status text not null,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists agents_workspace_id_idx on agents(workspace_id);

create table if not exists agent_keys (
	id text primary key,
	agent_id text not null references agents(id) on delete cascade,
	name text not null,
	hash text not null unique,
	prefix text not null,
	created_at timestamptz not null,
	expires_at timestamptz not null,
	revoked_at timestamptz
);

create index if not exists agent_keys_agent_id_idx on agent_keys(agent_id);
create index if not exists agent_keys_hash_idx on agent_keys(hash);

create table if not exists access_grants (
	id text primary key,
	caller_agent_id text not null references agents(id) on delete cascade,
	target_agent_id text not null references agents(id) on delete cascade,
	route_type text not null default '',
	route_key text not null default '',
	created_at timestamptz not null,
	expires_at timestamptz,
	revoked_at timestamptz
);

create index if not exists access_grants_caller_target_idx on access_grants(caller_agent_id, target_agent_id);

create table if not exists trace_events (
	id text primary key,
	run_id text not null default '',
	caller_agent_id text not null default '',
	target_agent_id text not null,
	route_type text not null,
	route_key text not null default '',
	decision text not null,
	reason text not null default '',
	created_at timestamptz not null
);

create index if not exists trace_events_run_id_idx on trace_events(run_id);
create index if not exists trace_events_decision_idx on trace_events(decision);
create index if not exists trace_events_caller_target_idx on trace_events(caller_agent_id, target_agent_id);
