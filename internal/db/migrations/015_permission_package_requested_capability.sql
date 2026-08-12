alter table permission_package_approval_requests
	add column if not exists requested_capability_id text not null default '';

alter table permission_package_applications
	add column if not exists requested_capability_id text not null default '';

create index if not exists permission_package_approval_requests_requested_capability_idx
	on permission_package_approval_requests(requested_capability_id, created_at desc, id desc)
	where requested_capability_id <> '';

create index if not exists permission_package_applications_requested_capability_idx
	on permission_package_applications(requested_capability_id, applied_at desc, id desc)
	where requested_capability_id <> '';
