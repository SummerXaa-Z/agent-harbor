# Sprint 5 Proxy Retry and Error Classification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add bounded per-target proxy retry controls and classify upstream network failures with stable error codes.

**Architecture:** Keep retry as target `channelConfig` metadata so the data-plane proxy can evaluate it after authorization. The proxy will read the incoming request body once, rebuild a fresh upstream request per attempt, and return either the first non-retryable response or the last retryable response after attempts are exhausted.

**Tech Stack:** Go `net/http`, existing `chi` HTTP API, existing memory/PostgreSQL repositories, Vite + React console.

---

### Task 1: Red Tests for Proxy Retry

**Files:**
- Modify: `internal/httpapi/server_test.go`

- [x] Add `TestUpstreamProxyRetriesRetryableStatusThenReturnsSuccess` using an `httptest.Server` that returns `503` first and `202` second, with target `channelConfig.retry.maxAttempts=2`.
- [x] Run `go test ./internal/httpapi -run TestUpstreamProxyRetriesRetryableStatusThenReturnsSuccess -count=1` and verify it fails because only one attempt is made.
- [x] Add `TestUpstreamProxyReturnsLastRetryableStatusAfterAttemptsExhausted`.
- [x] Run `go test ./internal/httpapi -run 'TestUpstreamProxyRetriesRetryableStatusThenReturnsSuccess|TestUpstreamProxyReturnsLastRetryableStatusAfterAttemptsExhausted' -count=1`.

### Task 2: Implement Retry Policy

**Files:**
- Modify: `internal/httpapi/server.go`

- [x] Add `proxyRetryPolicy` parsing from `channelConfig.retry`.
- [x] Validate retry config in `agentFromRequest`.
- [x] Refactor `proxyUpstreamIfConfigured` to read the request body once and loop attempts.
- [x] Set `X-AgentHarbor-Upstream-Attempts` on proxied upstream responses.
- [x] Run the retry tests and the full `internal/httpapi` suite.

### Task 3: Red Tests for Error Classification

**Files:**
- Modify: `internal/httpapi/server_test.go`
- Modify: `internal/domain/errors.go`

- [x] Add a test for closed upstream connection returning `UPSTREAM_CONNECT_ERROR`.
- [x] Add a test for DNS failure returning `UPSTREAM_DNS_ERROR`.
- [x] Run targeted tests and verify they fail with the old `UPSTREAM_ERROR`.

### Task 4: Implement Error Classification

**Files:**
- Modify: `internal/domain/errors.go`
- Modify: `internal/httpapi/server.go`

- [x] Add domain helpers for `UPSTREAM_DNS_ERROR`, `UPSTREAM_TLS_ERROR`, and `UPSTREAM_CONNECT_ERROR`.
- [x] Add `classifyUpstreamError` that unwraps `url.Error`, checks timeout, DNS, TLS, and connect-style errors.
- [x] Use the classifier in the proxy failure path.
- [x] Run targeted and full Go tests.

### Task 5: Console and Docs

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/data.ts`
- Modify: `internal/contracts/catalog.go`
- Modify: `README.md`
- Modify: `CHANGELOG.md`
- Create: `scripts/demo-sprint5-retry-config.sh`

- [x] Add retry fields to Create Agent form: max attempts and backoff milliseconds.
- [x] Add retry contract fields for MCP/OpenAPI/A2A catalogs and sample data.
- [x] Add a demo script proving valid retry config is accepted and invalid retry config is rejected.
- [x] Update README, changelog, and next milestones.

### Task 6: Final Verification and Git

**Files:**
- All changed files

- [x] Run `go test -count=1 ./...`.
- [x] Run `go vet ./...`.
- [x] Run `go build ./...`.
- [x] Run `pnpm -C frontend build`.
- [x] Run `bash -n scripts/demo-governance-loop.sh scripts/demo-sprint2-cleanup.sh scripts/demo-sprint3-mcp-policy.sh scripts/demo-sprint4-credentials.sh scripts/demo-sprint5-retry-config.sh`.
- [x] Run local API demos for Sprint 1-5.
- [x] Run PostgreSQL integration test with a temporary PostgreSQL 16 container.
- [x] Request review and fix accepted findings.
- [ ] Commit and push `codex/sprint-5-proxy-retry-classification`.
