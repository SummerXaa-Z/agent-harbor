create table if not exists route_policies (
	id text primary key,
	tenant_id text not null,
	workspace_id text not null,
	name text not null,
	caller_agent_id text not null references agents(id) on delete cascade,
	target_agent_id text not null references agents(id) on delete cascade,
	route_type text not null default '',
	route_key text not null default '',
	effect text not null,
	status text not null,
	priority integer not null default 100,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists route_policies_scope_idx on route_policies(tenant_id, workspace_id, created_at, id);
create index if not exists route_policies_caller_target_idx on route_policies(caller_agent_id, target_agent_id, status, priority desc);
create index if not exists route_policies_route_idx on route_policies(route_type, route_key);
