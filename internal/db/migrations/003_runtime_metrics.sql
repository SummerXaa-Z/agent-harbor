alter table trace_events
	add column if not exists duration_ms bigint not null default 0,
	add column if not exists upstream_attempts integer not null default 0,
	add column if not exists upstream_status integer not null default 0,
	add column if not exists upstream_error text not null default '';
