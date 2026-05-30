# Sprint 7 Agent Update and Credential Rotation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add safe Agent partial updates and credential rotation that reuses existing validation and encrypted persistence.

**Architecture:** Extend the Agent repository with a full-row `UpdateAgent` operation. HTTP handlers load the existing Agent, build an effective updated Agent, validate it through shared helper logic, then persist it with updated timestamp and no credential echo.

**Tech Stack:** Go `net/http` + `chi`, existing memory/PostgreSQL repositories, AES-GCM credential helper, Vite + React console.

---

### Task 1: Red Tests for Agent Partial Update

**Files:**
- Modify: `internal/httpapi/server_test.go`

- [x] Add `TestPatchAgentUpdatesMutableFieldsAndValidatesConfig`.
- [x] Verify `PATCH /api/v1/agents/{id}` currently returns method not allowed.
- [x] Expected final assertions: name/status/channelConfig update, credentials omitted, unsafe endpoint update rejected.

### Task 2: Red Tests for Credential Rotation

**Files:**
- Modify: `internal/httpapi/server_test.go`

- [x] Add `TestRotateAgentCredentialsTakesEffectOnNextProxyCall`.
- [x] Use a mock RoundTripper with a credentialed MCP Agent.
- [x] Verify pre-rotate proxied call sends old `Authorization` header.
- [x] Verify rotate endpoint currently fails.
- [x] Expected final assertions: rotate response redacts plaintext, post-rotate proxied call sends new `Authorization` header.

### Task 3: Repository Update Support

**Files:**
- Modify: `internal/store/memory.go`
- Modify: `internal/store/postgres.go`
- Modify: `internal/store/postgres_test.go`

- [x] Add `UpdateAgent(context.Context, domain.Agent)` to `store.Repository`.
- [x] Implement memory replacement by ID.
- [x] Implement PostgreSQL update of mutable fields, `channel_config`, `credentials_ciphertext`, `status`, and `updated_at`.
- [x] Extend PostgreSQL round-trip test to update an Agent and verify rotated credentials round-trip encrypted.

### Task 4: HTTP Handlers and Validation

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [x] Add `UpdateAgentRequest` and `RotateAgentCredentialsRequest`.
- [x] Add `PATCH /api/v1/agents/{id}` and `POST /api/v1/agents/{id}/credentials:rotate`.
- [x] Extract shared Agent validation for create/update effective Agent.
- [x] Run targeted tests until green.

### Task 5: Frontend Controls

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/App.tsx`

- [x] Add `updateAgent` and `rotateAgentCredentials` API clients.
- [x] Add Registry action to activate/draft/disable via `PATCH`.
- [x] Add credential rotation form with Agent selector, credential key, secret value, and message state.
- [x] Keep secret values out of local data after submit.

### Task 6: Docs, Demo, and Verification

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`
- Create: `scripts/demo-sprint7-credential-rotation.sh`

- [x] Document new endpoints and Sprint 7 demo.
- [x] Add demo script proving PATCH update and credential rotation response redaction.
- [x] Run `go test -count=1 ./...`.
- [x] Run `go vet ./...`.
- [x] Run `go build ./...`.
- [x] Run `pnpm -C frontend build`.
- [x] Run all demo script syntax checks.
