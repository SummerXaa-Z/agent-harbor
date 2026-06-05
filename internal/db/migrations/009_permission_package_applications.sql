create table if not exists permission_package_applications (
	id text primary key,
	draft_id text not null,
	template_id text not null,
	template_version integer not null,
	tenant_id text not null,
	workspace_id text not null,
	target_agent_id text not null references agents(id) on delete cascade,
	caller_instance_id text not null references agents(id) on delete cascade,
	subject_selector text not null default '',
	request_text text not null default '',
	region text not null default '',
	data_scopes jsonb not null default '[]'::jsonb,
	allowed_capability_ids jsonb not null default '[]'::jsonb,
	allowed_capability_keys jsonb not null default '[]'::jsonb,
	tenant_entitlement_ids jsonb not null default '[]'::jsonb,
	workspace_assignment_ids jsonb not null default '[]'::jsonb,
	instance_assignment_ids jsonb not null default '[]'::jsonb,
	applied_at timestamptz not null
);

create index if not exists permission_package_applications_scope_idx
	on permission_package_applications(tenant_id, workspace_id, applied_at desc, id desc);

create index if not exists permission_package_applications_template_idx
	on permission_package_applications(template_id, template_version, applied_at desc, id desc);

create index if not exists permission_package_applications_target_idx
	on permission_package_applications(target_agent_id, applied_at desc, id desc);

create index if not exists permission_package_applications_caller_idx
	on permission_package_applications(caller_instance_id, applied_at desc, id desc);
