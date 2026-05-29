# AgentHarbor Sprint 1 Governance Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first usable AgentHarbor governance loop with PostgreSQL persistence, minimal admin-key protection, API-backed console workflows, and an E2E demo path.

**Architecture:** Keep the existing `store.Repository` boundary and add a PostgreSQL implementation beside the in-memory store. Keep HTTP endpoint paths stable, but add admin-key middleware around management endpoints and read APIs needed by the console. The frontend remains a Vite React app and grows from cockpit-only into a small operational console with create-agent, create-key, create-grant, and trace refresh flows.

**Tech Stack:** Go 1.25, `chi`, `pgx/v5/pgxpool`, SQL migrations run at startup, React 19, Vite, TypeScript, browser `fetch`.

---

## File Structure

- `internal/store/memory.go`: keep the existing test-friendly repository and extend interface methods for grants and trace filters.
- `internal/store/postgres.go`: PostgreSQL repository implementation.
- `internal/store/postgres_test.go`: repository integration tests gated by `AGENT_HARBOR_TEST_DATABASE_URL`.
- `internal/app/app.go`: choose memory or PostgreSQL repository from environment.
- `internal/db/migrate.go`: startup migration runner for PostgreSQL.
- `internal/db/migrations/001_sprint1.sql`: schema for agents, agent_keys, access_grants, trace_events.
- `internal/httpapi/server.go`: admin-key middleware, grant list endpoint, trace filters.
- `internal/httpapi/server_test.go`: admin auth and HTTP behavior coverage.
- `cmd/agent-harbor/main.go`: environment handling remains small; app wiring owns repository choice.
- `frontend/src/types.ts`: add AccessGrant, AdminState, Create forms, richer ConsoleData.
- `frontend/src/api.ts`: add admin-key aware management API calls.
- `frontend/src/App.tsx`: add create workflows and API-backed grants/traces.
- `frontend/src/styles.css`: form, drawer/panel, empty-state, and alert styles.
- `docs/sprints/sprint-1-brief.md`: Sprint 1 product brief.
- `scripts/demo-governance-loop.sh`: local E2E demo script.

## Task 1: Backend Repository Contract and HTTP Admin Auth

- [x] Extend `store.Repository` with `ListAccessGrants(ctx)` and trace filter support.
- [x] Update `Memory` implementation and existing tests.
- [x] Add `AGENT_HARBOR_ADMIN_KEY` config to app/server wiring.
- [x] Protect management write endpoints and management read endpoints used by the console with `X-Admin-Key` when the env var is set.
- [x] Keep `/healthz`, `/contracts/*`, and data-plane Agent Key auth public/unchanged.
- [x] Add tests proving management writes reject missing/wrong admin key and allow correct key.

## Task 2: PostgreSQL Persistence

- [x] Add `github.com/jackc/pgx/v5/pgxpool`.
- [x] Add SQL migration for agents, agent_keys, access_grants, trace_events.
- [x] Implement `internal/store/postgres.go` with the same repository semantics as memory.
- [x] Add startup repository selection:
  - no `AGENT_HARBOR_DATABASE_URL`: use memory
  - with `AGENT_HARBOR_DATABASE_URL`: migrate then use PostgreSQL
- [x] Add integration tests gated by `AGENT_HARBOR_TEST_DATABASE_URL`.

## Task 3: API Read Models for Console

- [x] Add `GET /api/v1/access-grants`.
- [x] Add trace filters: `runId`, `decision`, `callerAgentId`, `targetAgentId`.
- [x] Ensure listed agent keys never include plaintext key material.
- [x] Update README API list and local env docs.

## Task 4: Frontend Operational Workflows

- [x] Add admin key input/state in the console header or status strip.
- [x] Add create-agent workflow for local/mcp/openapi/a2a/webhook channels.
- [x] Add create-agent-key workflow for active local caller agents and display plaintext once.
- [x] Add create-access-grant workflow with caller/target/route fields.
- [x] Replace Route Governance and Audit Traces with API-backed grants/traces.
- [x] Preserve fallback sample mode only when API is unreachable.
- [x] Add polished empty states for live API with no data.

## Task 5: E2E Demo and Verification

- [x] Add `scripts/demo-governance-loop.sh`.
- [x] Script flow:
  - start from a running service
  - create local caller
  - create target
  - create agent key
  - call data-plane and observe denied trace
  - create access grant
  - call data-plane again and observe allowed trace
- [x] Verify:
  - `go test ./...`
  - `go vet ./...`
  - `go build ./...`
  - `pnpm -C frontend build`
  - Playwright smoke for live API and fallback mode
