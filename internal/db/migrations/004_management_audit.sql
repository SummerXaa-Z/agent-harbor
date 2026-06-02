alter table agents
	add column if not exists credential_version integer not null default 0;

update agents
set credential_version = 1
where credential_version = 0
	and length(credentials_ciphertext) > 0;

create table if not exists audit_events (
	id text primary key,
	tenant_id text not null default '',
	workspace_id text not null default '',
	actor text not null default '',
	action text not null,
	resource_type text not null,
	resource_id text not null,
	summary text not null default '',
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null
);

create index if not exists audit_events_scope_idx on audit_events(tenant_id, workspace_id, created_at, id);
create index if not exists audit_events_action_idx on audit_events(action);
create index if not exists audit_events_resource_idx on audit_events(resource_type, resource_id, created_at, id);
create index if not exists audit_events_created_at_idx on audit_events(created_at);
