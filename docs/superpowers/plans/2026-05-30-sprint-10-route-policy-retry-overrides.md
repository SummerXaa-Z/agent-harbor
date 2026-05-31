# Sprint 10 Route Policy Retry Overrides Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional retry overrides to RoutePolicy and make matched allow policies control proxy retry behavior.

**Architecture:** RoutePolicy stores a normalized retry object. Repository access decisions carry the matched policy retry override. HTTP proxy retry resolution prefers policy retry over target Agent `channelConfig.retry`, preserving legacy grant behavior.

**Tech Stack:** Go, chi HTTP API, in-memory repository, PostgreSQL/pgx, React/Vite TypeScript console, Bash demo scripts.

---

### Task 1: Backend Red Tests

**Files:**
- Modify: `internal/httpapi/server_test.go`
- Modify: `internal/store/postgres_test.go`

- [ ] **Step 1: Add failing HTTP test for policy retry override**

Add a test that creates an upstream returning 503 then 202, a target Agent without retry, and an allow RoutePolicy with `retry.maxAttempts=2`. The expected behavior is HTTP 202 and `X-AgentHarbor-Upstream-Attempts: 2`.

- [ ] **Step 2: Add failing validation tests**

Add create/patch tests where `retry.maxAttempts=5` and `retry.statusCodes=[429]` return 400.

- [ ] **Step 3: Add failing PostgreSQL round-trip assertion**

In `TestPostgresRepositoryRoundTrip`, create a policy with retry, list it, evaluate it, and assert retry survives both paths.

- [ ] **Step 4: Run tests and confirm RED**

Run: `go test ./internal/httpapi ./internal/store -count=1`

Expected: compile or assertion failure because RoutePolicy retry fields and proxy override behavior do not exist yet.

### Task 2: Domain, Store, and Migration

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/store/memory.go`
- Modify: `internal/store/postgres.go`
- Create: `internal/db/migrations/006_sprint10_route_policy_retry.sql`

- [ ] **Step 1: Add RoutePolicy retry types**

Add request and normalized domain types for route policy retry.

- [ ] **Step 2: Extend RouteAccessDecision**

Carry `Retry *RoutePolicyRetry` only for matched allow policies.

- [ ] **Step 3: Persist retry**

Memory stores retry directly. PostgreSQL adds a `retry jsonb` column and scans/marshals it.

- [ ] **Step 4: Run store tests**

Run: `go test ./internal/store -count=1`

Expected: store tests pass.

### Task 3: HTTP API and Proxy Integration

**Files:**
- Modify: `internal/httpapi/server.go`

- [ ] **Step 1: Validate route policy retry**

Reuse the same bounds as channelConfig retry. Missing fields inside a present retry object get defaults.

- [ ] **Step 2: Create and patch retry**

Create accepts optional retry. Patch can replace retry or clear it with null.

- [ ] **Step 3: Prefer policy retry in proxy**

Pass `decision.Retry` into proxy execution and use it before target Agent channelConfig retry.

- [ ] **Step 4: Run HTTP tests**

Run: `go test ./internal/httpapi -count=1`

Expected: HTTP tests pass.

### Task 4: Frontend, Demo, Docs

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/data.ts`
- Create: `scripts/demo-sprint10-route-policy-retry.sh`
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add frontend retry fields**

Create Policy form sends retry attempts/backoff when different from defaults.

- [ ] **Step 2: Show retry summary**

Policy table displays retry attempts when a policy has retry.

- [ ] **Step 3: Add demo script**

Demo creates a retry policy and proves the upstream is attempted twice.

- [ ] **Step 4: Run build and demo**

Run: `npm --prefix frontend run build` and the Sprint 10 demo against a local API.

### Task 5: Final Verification

**Files:**
- All touched files

- [ ] **Step 1: Run complete verification**

Run: `go test ./...`, `go vet ./...`, `go build ./...`, `git diff --check`, frontend build, demo, PostgreSQL round-trip, and browser smoke.

- [ ] **Step 2: Request review**

Ask reviewer to focus on retry precedence, validation parity, PostgreSQL JSON persistence, and legacy grant compatibility.
