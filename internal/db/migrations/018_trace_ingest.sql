alter table trace_events
	add column if not exists source text not null default 'gateway',
	add column if not exists outcome text not null default '',
	add column if not exists ingest_payload jsonb,
	add column if not exists tool_name text not null default '';
