alter table agents
	add column if not exists credentials_ciphertext bytea not null default decode('', 'hex');
