create table if not exists verified_task_result_summaries (
	id text primary key,
	tenant_id text not null,
	workspace_id text not null,
	caller_instance_id text not null,
	subject_id text not null,
	target_agent_id text not null,
	capability_id text not null,
	source_trace_id text not null references trace_events(id) on delete restrict,
	data_scopes jsonb not null default '[]'::jsonb,
	summary text not null,
	payload_digest text not null,
	verification text not null,
	verified_by text not null,
	verified_at timestamptz not null,
	created_at timestamptz not null,
	expires_at timestamptz not null,
	constraint verified_task_result_summaries_source_trace_id_key unique (source_trace_id)
);

create index if not exists verified_task_result_summaries_exact_scope_idx
	on verified_task_result_summaries (
		tenant_id, workspace_id, caller_instance_id, subject_id, expires_at, created_at
	);
