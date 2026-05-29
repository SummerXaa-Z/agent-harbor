# AI Nexus Go Rebirth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a clean-room Go Agent Gateway skeleton that proves registry, contracts, Agent Key auth, access grants, data-plane authorization, and audit traces.

**Architecture:** A new root-level Go module under `go-nexus/` contains all new runtime code. HTTP handlers depend on small domain services and an in-memory repository so the first milestone is testable without PostgreSQL. The API shape intentionally follows the product model, not the existing implementation internals.

**Tech Stack:** Go 1.25, `chi` router, standard `net/http/httptest`, in-memory store for Sprint 0, future `pgx/sqlc/goose` placeholders documented but not required for tests.

---

## File Structure

- `go-nexus/go.mod`: independent Go module for the rebirth service.
- `go-nexus/cmd/nexus-go/main.go`: process entrypoint.
- `go-nexus/internal/app/app.go`: server wiring and route registration.
- `go-nexus/internal/httpapi/`: JSON helpers, handlers, middleware.
- `go-nexus/internal/domain/`: clean-room domain types.
- `go-nexus/internal/store/`: repository interface and in-memory implementation.
- `go-nexus/internal/security/`: key generation, hashing, endpoint validation.
- `go-nexus/internal/contracts/`: provider/channel contract catalog.
- `go-nexus/internal/audit/`: trace event helpers.
- `go-nexus/internal/httpapi/server_test.go`: end-to-end HTTP tests.

## Tasks

### Task 1: Module and Domain Types

- [ ] Create `go-nexus/go.mod` with `chi` dependency.
- [ ] Create domain structs for Agent, AgentKey, AccessGrant, TraceEvent, contracts, and common API envelopes.
- [ ] Add repository interface and in-memory store with deterministic mutex protection.
- [ ] Run `go test ./...` from `go-nexus`.

### Task 2: HTTP Skeleton and Contracts

- [ ] Create app wiring with `chi.NewRouter`, request ID, recovery, timeout, JSON handlers.
- [ ] Implement `GET /healthz`, `GET /api/v1/contracts/providers`, and `GET /api/v1/contracts/channels`.
- [ ] Add httptest coverage for health and catalog responses.
- [ ] Run `go test ./...`.

### Task 3: Agent Registry and Validation

- [ ] Implement `POST /api/v1/agents`, `GET /api/v1/agents`, `GET /api/v1/agents/{id}`.
- [ ] Validate name, workspaceId, status, channelType, secret-like channel config keys, and unsafe endpoint values.
- [ ] Add tests for create/list/read plus validation failures.
- [ ] Run `go test ./...`.

### Task 4: Agent Key and Access Grant

- [ ] Implement `POST /api/v1/agent-keys`; return plaintext once, store only hash.
- [ ] Implement `POST /api/v1/access-grants`.
- [ ] Add bearer-auth middleware that resolves caller Agent from key hash.
- [ ] Add tests for key creation and unauthorized data-plane access.
- [ ] Run `go test ./...`.

### Task 5: Data-plane Simulation and Audit Traces

- [ ] Implement `POST /api/v1/mcp/agents/{targetId}/rpc`.
- [ ] Implement `POST /api/v1/openapi/agents/{targetId}/operations/{operationId}`.
- [ ] Enforce caller -> target grant, return `PERMISSION_DENIED` on deny, and write trace for both allowed and denied paths.
- [ ] Implement `GET /api/v1/audit/traces`.
- [ ] Add tests for allowed/denied trace evidence.
- [ ] Run `go test ./...` and `go build ./...`.

### Task 6: Clean-room Documentation and Push

- [ ] Add README for `go-nexus/` with clean-room warning and local commands.
- [ ] Run secret scan over new files.
- [ ] Commit with a clear clean-room message.
- [ ] Push branch `codex/ai-nexus-go-rebirth`.
