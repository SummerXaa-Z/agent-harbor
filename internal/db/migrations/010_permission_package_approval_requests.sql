create table if not exists permission_package_approval_requests (
	id text primary key,
	draft_id text not null,
	template_id text not null,
	template_version integer not null,
	policy_version integer not null,
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
	policy_gate jsonb not null default '{}'::jsonb,
	status text not null,
	requested_by text not null default '',
	reviewed_by text not null default '',
	review_comment text not null default '',
	created_at timestamptz not null,
	updated_at timestamptz not null,
	resolved_at timestamptz
);

create index if not exists permission_package_approval_requests_scope_idx
	on permission_package_approval_requests(tenant_id, workspace_id, created_at desc, id desc);

create index if not exists permission_package_approval_requests_template_idx
	on permission_package_approval_requests(template_id, template_version, created_at desc, id desc);

create index if not exists permission_package_approval_requests_status_idx
	on permission_package_approval_requests(status, created_at desc, id desc);

create index if not exists permission_package_approval_requests_target_idx
	on permission_package_approval_requests(target_agent_id, created_at desc, id desc);

create index if not exists permission_package_approval_requests_caller_idx
	on permission_package_approval_requests(caller_instance_id, created_at desc, id desc);
