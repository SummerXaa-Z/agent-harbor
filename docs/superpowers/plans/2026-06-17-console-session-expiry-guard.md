# Console Session Expiry Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Return expired browser console sessions to the login flow before stale management writes.

**Architecture:** Preserve the existing signed-session protocol and use the existing `expiresAt` field. The backend marks auth responses no-store; the frontend adds a pure expiry-delay helper and a hook effect that refreshes the session at expiry and shows localized expired-session guidance.

**Tech Stack:** Go HTTP API, React TypeScript hook, existing i18n map, Node test runner, Make gates.

---

## Files

- Modify: `internal/httpapi/auth.go`
- Modify: `internal/httpapi/server_test.go`
- Create: `frontend/src/consoleSession.ts`
- Modify: `frontend/src/hooks/useConsoleAuth.ts`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/consoleAuth.test.mjs`
- Create: `frontend/tests/consoleSession.test.mjs`
- Modify: `CHANGELOG.md`
- Create: `docs/superpowers/specs/2026-06-17-console-session-expiry-guard-design.md`
- Create: `docs/superpowers/plans/2026-06-17-console-session-expiry-guard.md`

## Task 1: RED Tests

- [x] **Step 1: Add backend no-store expectations**

In `TestConsoleAuthSessionProtectsManagementEndpoints`, assert `Cache-Control: no-store` on session status, login, and logout responses.

- [x] **Step 2: Add frontend expiry tests**

Create `frontend/tests/consoleSession.test.mjs` to import `sessionExpiryDelayMs` and cover future, expired, missing, and invalid timestamps. Extend `frontend/tests/consoleAuth.test.mjs` to require hook scheduling with `window.setTimeout`, cleanup with `window.clearTimeout`, and bilingual `error.consoleSessionExpired` copy.

- [x] **Step 3: Confirm RED**

Run:

```bash
go test ./internal/httpapi -run 'TestConsoleAuthSessionProtectsManagementEndpoints' -count=1
pnpm --dir frontend test -- tests/consoleAuth.test.mjs tests/consoleSession.test.mjs
```

Expected: backend fails with empty `Cache-Control`; frontend fails because `consoleSession.ts`, expiry scheduling, and copy are missing.

## Task 2: Implement Backend and Frontend Guard

- [x] **Step 1: Add no-store auth response helper**

Add `setConsoleAuthNoStore(w)` and call it at the start of `getAuthSession`, `login`, and `logout`.

- [x] **Step 2: Add expiry delay pure function**

Create:

```ts
export function sessionExpiryDelayMs(expiresAt?: string, nowMs: number = Date.now()): number | null {
  if (!expiresAt) return null
  const expiresAtMs = Date.parse(expiresAt)
  if (!Number.isFinite(expiresAtMs)) return null
  return Math.max(0, expiresAtMs - nowMs)
}
```

- [x] **Step 3: Schedule session refresh in `useConsoleAuth`**

When `state.session` is authenticated, requires login, and has a valid `expiresAt`, schedule `fetchConsoleSession` at the computed delay. Clear the timeout and abort any in-flight refresh on cleanup.

- [x] **Step 4: Add bilingual copy and changelog**

Add `error.consoleSessionExpired` in English and Simplified Chinese, plus bilingual changelog bullets.

## Task 3: Verification and Ship

- [x] **Step 1: Confirm focused GREEN**

Run:

```bash
gofmt -w internal/httpapi/auth.go internal/httpapi/server_test.go
go test ./internal/httpapi -run 'TestConsoleAuthSessionProtectsManagementEndpoints' -count=1
pnpm --dir frontend test -- tests/consoleAuth.test.mjs tests/consoleSession.test.mjs
```

Expected: pass.

- [x] **Step 2: Run full gates**

Run:

```bash
git diff --check
make check
make release-check
```

Expected: all pass.

- [ ] **Step 3: Ship**

Review diff, commit, push, open PR, wait for CI, and merge when green.
