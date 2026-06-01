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
