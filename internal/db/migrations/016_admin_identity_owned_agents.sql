alter table admin_identities
	add column if not exists owned_agent_ids jsonb not null default '[]'::jsonb;
