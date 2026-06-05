alter table permission_package_approval_requests
	add column if not exists expires_at timestamptz;

update permission_package_approval_requests
set expires_at = created_at + interval '24 hours'
where expires_at is null;

alter table permission_package_approval_requests
	alter column expires_at set not null;

alter table permission_package_approval_requests
	add column if not exists consumed_at timestamptz,
	add column if not exists consumed_by_application_id text not null default '';

create index if not exists permission_package_approval_requests_consumed_idx
	on permission_package_approval_requests(consumed_at, created_at desc, id desc);
