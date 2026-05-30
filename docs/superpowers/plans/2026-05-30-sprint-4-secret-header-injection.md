# Sprint 4 Implementation Plan: Secret Header Injection

## Files

- `internal/domain/types.go`: add internal `Agent.Credentials` and request `credentials`.
- `internal/httpapi/server.go`: validate credentials, allow `credentialHeaders`, inject credential headers in proxy.
- `internal/httpapi/server_test.go`: cover response redaction, missing credential references, and upstream injection.
- `internal/security/credentials.go`: AES-GCM credential encryption and key parsing.
- `internal/store/postgres.go`: persist encrypted credential ciphertext and decrypt on read.
- `internal/db/migrations/002_sprint4_agent_credentials.sql`: add credential ciphertext column.
- `internal/app/app.go`: wire `AGENT_HARBOR_CREDENTIAL_KEY` into PostgreSQL repository.
- `frontend/src/App.tsx` / `frontend/src/types.ts`: add create-time credential fields.
- `scripts/demo-sprint4-credentials.sh`: management API redaction demo.
- `README.md`, `CHANGELOG.md`, `docs/clean-room-spec.md`, `docs/sprints/sprint-4-brief.md`: document behavior.

## Checklist

- [x] Add failing API test for credential create and response redaction.
- [x] Implement request/domain credential fields and redaction-by-JSON-tag.
- [x] Add failing proxy test for `Authorization` injection from credential mapping.
- [x] Implement `copyCredentialHeaders`.
- [x] Add Postgres credential round-trip expectation.
- [x] Add encrypted credential storage and migration.
- [x] Add frontend create form fields and request shaping.
- [x] Add demo script and docs.
- [x] Run full backend/frontend/script/PostgreSQL verification.
- [x] Request review and fix findings.
- [ ] Commit and push branch.
