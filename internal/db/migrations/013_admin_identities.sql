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
