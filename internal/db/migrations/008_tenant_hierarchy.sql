create table if not exists tenants (
	id text primary key,
	parent_tenant_id text references tenants(id) on delete restrict,
	level integer not null,
	name text not null,
	status text not null,
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists tenants_parent_idx on tenants(parent_tenant_id);
