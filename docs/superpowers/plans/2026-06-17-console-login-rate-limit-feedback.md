# Console Login Rate Limit Feedback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make browser console login lockouts understandable by exposing retry timing and showing localized retry guidance.

**Architecture:** Reuse the existing process-local failed-login counter. The login handler calculates the remaining lockout window and emits `Retry-After`; the frontend API client preserves that hint on `ApiRequestError`; the auth hook maps `RATE_LIMITED` to i18n copy.

**Tech Stack:** Go HTTP API, React TypeScript frontend, existing i18n map, existing Node test runner, Make gates.

---

## Files

- Modify: `internal/httpapi/auth.go`
- Modify: `internal/httpapi/server_test.go`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/hooks/useConsoleAuth.ts`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/api.test.mjs`
- Modify: `frontend/tests/consoleAuth.test.mjs`
- Modify: `CHANGELOG.md`
- Create: `docs/superpowers/specs/2026-06-17-console-login-rate-limit-feedback-design.md`
- Create: `docs/superpowers/plans/2026-06-17-console-login-rate-limit-feedback.md`

## Task 1: RED Tests

- [x] **Step 1: Add backend retry header expectation**

Add this assertion to `TestConsoleLoginRateLimit` after the 429 assertion:

```go
if got := limited.Header().Get("Retry-After"); got != "300" {
	t.Fatalf("rate limited login should expose retry window, got Retry-After=%q", got)
}
```

- [x] **Step 2: Add frontend API and auth wiring expectations**

Extend `frontend/tests/api.test.mjs` to require `retryAfterSeconds` and parsing of the `Retry-After` header. Add a new `frontend/tests/consoleAuth.test.mjs` test that requires `ApiRequestError`, `RATE_LIMITED`, `retryAfterSeconds`, and bilingual `error.consoleLoginRateLimited` copy.

- [x] **Step 3: Confirm RED**

Run:

```bash
go test ./internal/httpapi -run 'TestConsoleLoginRateLimit' -count=1
pnpm --dir frontend test -- tests/api.test.mjs tests/consoleAuth.test.mjs
```

Expected: backend fails because `Retry-After` is empty; frontend fails because retry metadata and localized lockout copy do not exist yet.

## Task 2: Backend Retry Hint

- [x] **Step 1: Calculate and emit retry window**

Replace the login pre-check with `consoleLoginRetryAfterSeconds`. When it returns a positive value, set `Retry-After` and write `RATE_LIMITED`.

```go
if retryAfterSeconds := s.consoleLoginRetryAfterSeconds(r); retryAfterSeconds > 0 {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	writeError(w, domain.TooManyRequests("RATE_LIMITED", "too many failed console login attempts; retry later"))
	return
}
```

- [x] **Step 2: Keep retry seconds bounded and deterministic**

Use ceiling seconds so a five-minute window reports `300` seconds immediately after the fifth failure.

```go
func ceilDurationSeconds(duration time.Duration) int {
	if duration <= 0 {
		return 0
	}
	return int((duration + time.Second - 1) / time.Second)
}
```

## Task 3: Frontend Guidance

- [x] **Step 1: Preserve retry metadata**

Add `retryAfterSeconds` to `ApiRequestError` and parse the `Retry-After` response header when throwing request errors.

- [x] **Step 2: Map `RATE_LIMITED` in the auth hook**

When `loginConsole` throws `ApiRequestError` with code `RATE_LIMITED`, set:

```ts
{ key: "error.consoleLoginRateLimited", params: { seconds: error.retryAfterSeconds ?? 300 } }
```

- [x] **Step 3: Add bilingual copy**

Add:

```ts
"error.consoleLoginRateLimited": "Too many failed sign-in attempts. Try again in about {seconds} seconds."
"error.consoleLoginRateLimited": "登录失败次数过多，请约 {seconds} 秒后再试。"
```

## Task 4: Verification and Ship

- [x] **Step 1: Confirm GREEN for focused tests**

Run:

```bash
gofmt -w internal/httpapi/auth.go internal/httpapi/server_test.go
go test ./internal/httpapi -run 'TestConsoleLoginRateLimit' -count=1
pnpm --dir frontend test -- tests/api.test.mjs tests/consoleAuth.test.mjs
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
