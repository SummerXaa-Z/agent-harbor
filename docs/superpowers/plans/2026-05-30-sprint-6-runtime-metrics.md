# Sprint 6 Runtime Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Back Runtime Signals with real trace-derived metrics and enrich data-plane traces with proxy attempts, status, errors, and duration.

**Architecture:** Keep trace storage append-only by recording allowed proxied traces after proxy completion, with proxy outcome fields included at insert time. Add an admin metrics endpoint that aggregates scoped traces in memory and returns the frontend's existing `SystemMetric` shape.

**Tech Stack:** Go `net/http`, existing `chi` HTTP API, memory/PostgreSQL repositories, Vite + React + TypeScript console.

---

### Task 1: Red Tests for Runtime Metrics API

**Files:**
- Modify: `internal/httpapi/server_test.go`

- [x] **Step 1: Write failing metrics test**

Add `TestRuntimeMetricsSummarizeDataPlaneTraces` that creates one denied MCP call and one allowed local-stub MCP call in the same workspace, then calls `GET /api/v1/metrics/runtime?workspaceId=ws-1` and expects:

```go
gateway_calls_total = 2
allowed_rate = 50
upstream_error_rate = 0
avg_latency_ms = 0
```

- [x] **Step 2: Verify red**

Run:

```bash
go test ./internal/httpapi -run TestRuntimeMetricsSummarizeDataPlaneTraces -count=1
```

Expected: `404 page not found` or missing metrics endpoint failure.

### Task 2: Red Tests for Proxy Trace Fields

**Files:**
- Modify: `internal/httpapi/server_test.go`

- [x] **Step 1: Write failing proxy trace test**

Add `TestProxyTraceMetricsRecordAttemptsStatusAndDuration` using an `httptest.Server` that returns `503` then `202`, target `channelConfig.retry.maxAttempts=2`, and run id `metrics-proxy`. After the call, read `/api/v1/audit/traces?runId=metrics-proxy` and expect the allowed trace to include:

```go
UpstreamAttempts == 2
UpstreamStatus == 202
UpstreamError == ""
DurationMs > 0
```

- [x] **Step 2: Verify red**

Run:

```bash
go test ./internal/httpapi -run TestProxyTraceMetricsRecordAttemptsStatusAndDuration -count=1
```

Expected: trace fields are zero because they do not exist yet.

### Task 3: Implement Trace Enrichment and Metrics API

**Files:**
- Modify: `internal/domain/types.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/store/memory.go`
- Modify: `internal/store/postgres.go`
- Create: `internal/db/migrations/003_sprint6_runtime_metrics.sql`

- [x] Add trace fields to `domain.TraceEvent`.
- [x] Add `SystemMetric` domain shape for the metrics API.
- [x] Add migration columns for `duration_ms`, `upstream_attempts`, `upstream_status`, and `upstream_error`.
- [x] Extend PostgreSQL trace insert/select/scan paths.
- [x] Refactor proxy helper to return proxy outcome fields before allowed trace insertion.
- [x] Add `GET /api/v1/metrics/runtime` under admin middleware.
- [x] Run targeted HTTP API tests and keep them green.

### Task 4: PostgreSQL Round Trip

**Files:**
- Modify: `internal/store/postgres_test.go`

- [x] Add trace field values to the existing round-trip test.
- [x] Assert `ListTraces` returns the same `durationMs`, `upstreamAttempts`, `upstreamStatus`, and `upstreamError`.
- [x] Run:

```bash
go test ./internal/store -run TestPostgresRepositoryRoundTrip -count=1
```

Expected locally without `AGENT_HARBOR_TEST_DATABASE_URL`: skip; final verification will run with a temporary PostgreSQL container.

### Task 5: Frontend Runtime Metrics Fetch

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/App.tsx`

- [x] Import `SystemMetric`.
- [x] Add `fetchRuntimeMetrics(scope, adminKey, signal)` calling `/api/v1/metrics/runtime`.
- [x] Add metrics fetch to `loadConsoleData()` with `systemMetrics` network fallback.
- [x] Change `SignalBoard` key from `metric.label` to `metric.id`.
- [x] Run:

```bash
pnpm -C frontend build
```

Expected: TypeScript and Vite build pass.

### Task 6: Docs, Demo, and Verification

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`
- Create: `scripts/demo-sprint6-runtime-metrics.sh`

- [x] Document `/api/v1/metrics/runtime` and Sprint 6 demo.
- [x] Add demo script that creates denied/allowed calls and verifies runtime metrics contain `gateway_calls_total >= 2` and `allowed_rate`.
- [x] Run:

```bash
go test -count=1 ./...
go vet ./...
go build ./...
pnpm -C frontend build
bash -n scripts/demo-governance-loop.sh scripts/demo-sprint2-cleanup.sh scripts/demo-sprint3-mcp-policy.sh scripts/demo-sprint4-credentials.sh scripts/demo-sprint5-retry-config.sh scripts/demo-sprint6-runtime-metrics.sh
```

Expected: all commands exit 0.
