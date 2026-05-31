# Sprint 8 Management Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Add management audit events and non-secret Agent credential versions.

**Architecture:** Extend the domain model with `AuditEvent` and `Agent.CredentialVersion`, then add repository methods for appending and listing management events. HTTP handlers record audit events after successful control-plane mutations and expose them through an admin list endpoint. The console consumes the new endpoint in a read-only audit panel.

**Tech Stack:** Go `net/http` + `chi`, memory/PostgreSQL repositories, pgx migrations, Vite + React + TypeScript.

---

## File Structure

- Modify `internal/domain/types.go`: add `Agent.CredentialVersion`, `AuditEvent`, and filter request-facing types.
- Modify `internal/store/memory.go`: add `AppendAuditEvent` / `ListAuditEvents` and filtering.
- Modify `internal/store/postgres.go`: persist and scan credential version plus audit events.
- Create `internal/db/migrations/004_sprint8_management_audit.sql`: schema change.
- Modify `internal/httpapi/server.go`: add route, handler, and audit recording helpers.
- Modify `internal/httpapi/server_test.go`: red tests for audit API and credential version.
- Modify `internal/store/postgres_test.go`: red test for PostgreSQL round trip.
- Modify `frontend/src/types.ts`, `frontend/src/api.ts`, `frontend/src/data.ts`, `frontend/src/App.tsx`, `frontend/src/styles.css`: audit event display.
- Create `scripts/demo-sprint8-management-audit.sh`: curl demo and redaction checks.
- Modify `README.md` and `CHANGELOG.md`: document endpoint and sprint.

## Tasks

### Task 1: Backend API Red Tests

- [x] Add `auditEventResponse` test helper mirroring the JSON response.
- [x] Add `TestManagementAuditEventsRecordAgentLifecycleWithoutSecrets`.
- [x] Assert create returns `credentialVersion=1`, patch keeps it at `1`, rotation returns `2`, and audit list contains `agent.created`, `agent.updated`, `agent.credentials_rotated`.
- [x] Assert response bodies do not contain old/new secret values or `credentials`.
- [x] Add `TestManagementAuditEventsFilterByScopeActionAndResource` for tenant/workspace/action/resource filters.
- [x] Run `go test ./internal/httpapi -run 'TestManagementAuditEvents|TestRotateAgentCredentials' -count=1` and verify the new tests fail because the endpoint and fields do not exist.

### Task 2: Domain, Memory Store, and HTTP Implementation

- [x] Add `CredentialVersion int` to `domain.Agent`.
- [x] Add `domain.AuditEvent` with fields from the design.
- [x] Add `store.AuditEventFilter`, `AppendAuditEvent`, and `ListAuditEvents`.
- [x] Implement memory append/list filtering and stable ordering.
- [x] Set create-time credential version to `1` only when normalized credentials are non-empty.
- [x] Increment credential version on rotation.
- [x] Add `GET /api/v1/audit/events` and audit recording helpers.
- [x] Record management events for Agent create/update/disable, Agent Key create/revoke, and Access Grant create/revoke.
- [x] Run the focused HTTP tests and verify they pass.

### Task 3: PostgreSQL Persistence

- [x] Create migration `004_sprint8_management_audit.sql`.
- [x] Include `credential_version` in all Agent insert/select/update/disable/find-key paths.
- [x] Implement Postgres `AppendAuditEvent` and `ListAuditEvents`.
- [x] Extend `internal/store/postgres_test.go` to assert credential version and audit event round trip.
- [x] Run `go test ./internal/store -count=1`.

### Task 4: Frontend Console

- [x] Add `AuditEvent` and `AuditEventFilters` types.
- [x] Add `fetchAuditEvents` and include audit events in `loadConsoleData`.
- [x] Add sample audit events to fallback data.
- [x] Render a compact Management Audit table in `App.tsx`.
- [x] Run `npm --prefix frontend run build`.

### Task 5: Demo and Docs

- [x] Add `scripts/demo-sprint8-management-audit.sh` that creates, patches, rotates, lists audit events, and checks redaction.
- [x] Update README API list, demo section, and next milestones.
- [x] Update CHANGELOG with Sprint 8.
- [x] Run `bash -n scripts/demo-sprint8-management-audit.sh`.

### Task 6: Verification and Review

- [x] Run `go test ./...`.
- [x] Run `npm --prefix frontend run build`.
- [x] Run frontend smoke with local API and Vite preview.
- [x] Request code review.
- [x] Address review findings.
- [x] Commit and push `codex/sprint-8-management-audit`.
