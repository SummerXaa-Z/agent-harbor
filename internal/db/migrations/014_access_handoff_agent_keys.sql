alter table agent_keys
	add column if not exists application_id text not null default '',
	add column if not exists template_id text not null default '',
	add column if not exists subject_selector text not null default '',
	add column if not exists created_for_handoff_id text not null default '';

create index if not exists agent_keys_access_handoff_idx
	on agent_keys(created_for_handoff_id, created_at)
	where created_for_handoff_id <> '';
