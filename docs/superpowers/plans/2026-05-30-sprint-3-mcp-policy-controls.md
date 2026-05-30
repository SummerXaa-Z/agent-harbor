# Sprint 3 MCP Policy Controls Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add MCP method-level authorization, safe upstream header injection, bounded proxy timeouts, and console route key presets.

**Architecture:** Keep `internal/httpapi` as the protocol edge and keep repository interfaces unchanged. MCP method parsing happens before authorization, channel config validation stays in `agentFromRequest`, and proxy execution uses a per-target timeout helper.

**Tech Stack:** Go 1.25, `chi`, standard `net/http`, React 19, Vite, TypeScript.

---

## File Structure

- `internal/domain/errors.go`: add timeout app error.
- `internal/httpapi/server.go`: parse MCP JSON-RPC method, validate headers/timeout config, forward target headers, enforce timeout.
- `internal/httpapi/server_test.go`: TDD coverage for method policy, config validation, header forwarding, and timeout response.
- `frontend/src/App.tsx`: add route key preset selector in create-grant form.
- `frontend/src/styles.css`: style compact preset controls if needed.
- `README.md`, `CHANGELOG.md`, `docs/sprints/sprint-3-brief.md`: document Sprint 3 behavior.

## Task 1: MCP Method-Level Authorization

- [x] Add tests in `internal/httpapi/server_test.go`:
  - `TestMCPMethodRouteKeyUsesJSONRPCMethod`
  - `TestMCPInvalidMethodReturnsValidationErrorWithoutTrace`
- [x] Run targeted tests and confirm they fail.
- [x] Implement MCP body buffering and method extraction in `mcpRPC`.
- [x] Preserve original body for upstream proxy after parsing.
- [x] Run targeted tests and `go test ./internal/httpapi -count=1`.

## Task 2: Upstream Headers and Timeout

- [x] Add tests in `internal/httpapi/server_test.go`:
  - `TestUpstreamProxyForwardsConfiguredHeaders`
  - `TestAgentRejectsSecretLikeHeaders`
  - `TestUpstreamTimeoutReturnsGatewayTimeout`
- [x] Run targeted tests and confirm they fail.
- [x] Validate `channelConfig.headers` and `channelConfig.timeoutMs` in `agentFromRequest`.
- [x] Forward configured headers after request headers so target config wins only for non-secret names.
- [x] Apply configured timeout around `http.DefaultClient.Do`.
- [x] Add `domain.UpstreamTimeout`.
- [x] Run targeted tests and `go test ./...`.

## Task 3: Console Route Key Presets

- [x] Add MCP route key preset UI in `GrantCreateForm`.
- [x] Keep free-form route key input editable.
- [x] Run `pnpm -C frontend build`.

## Task 4: Docs and Verification

- [x] Update README with Sprint 3 MCP policy and proxy config notes.
- [x] Update CHANGELOG.
- [x] Run:
  - `go test -count=1 ./...`
  - `go vet ./...`
  - `go build ./...`
  - `pnpm -C frontend build`
  - `bash -n scripts/demo-governance-loop.sh scripts/demo-sprint2-cleanup.sh`
  - Demo scripts against local API.
- [x] Final review, commit, push branch.
