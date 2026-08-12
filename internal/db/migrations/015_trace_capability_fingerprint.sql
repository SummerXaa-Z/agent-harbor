-- A capability-aware trace carries a canonical policy witness so downstream
-- consumers can fail closed when capability policy changes without relying on
-- the legacy version counter alone.
alter table trace_events
	add column if not exists capability_fingerprint text not null default '';
