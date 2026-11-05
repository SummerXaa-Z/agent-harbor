alter table trace_events
	add column if not exists args_digest text not null default '',
	add column if not exists risk_flags jsonb not null default '[]'::jsonb,
	add column if not exists rule_ref text not null default '',
	add column if not exists high_risk boolean not null default false;

create index if not exists trace_events_high_risk_idx
	on trace_events(created_at)
	where high_risk;
