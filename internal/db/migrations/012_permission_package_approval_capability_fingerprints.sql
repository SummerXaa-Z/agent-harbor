alter table permission_package_approval_requests
	add column if not exists allowed_capability_fingerprints jsonb not null default '[]'::jsonb;
