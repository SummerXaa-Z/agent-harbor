# Console Session CSRF Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Protect browser console-session management writes with a session-bound CSRF token.

**Architecture:** Keep the protection in the HTTP/API boundary. The server derives a CSRF token from the signed session cookie and validates `X-AgentHarbor-CSRF` only when the console cookie is the credential used for an unsafe management request. The frontend stores the latest token in module memory and injects it centrally from `frontend/src/api.ts`.

**Tech Stack:** Go HTTP API, existing HMAC session signing, React/TypeScript frontend API client, Node test-source regression tests, Make release gates.

---

## Files

- Modify: `internal/httpapi/auth.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/tests/api.test.mjs`
- Modify: `CHANGELOG.md`
- Create: `docs/superpowers/specs/2026-06-17-console-session-csrf-guard-design.md`
- Create: `docs/superpowers/plans/2026-06-17-console-session-csrf-guard.md`

## Task 1: Backend CSRF Boundary

- [x] **Step 1: Add failing backend assertions**

Update `TestConsoleAuthSessionProtectsManagementEndpoints` in `internal/httpapi/server_test.go`:

```go
csrfToken, ok := session["csrfToken"].(string)
if !ok || csrfToken == "" {
	t.Fatalf("login session should include csrf token, got %#v", session)
}

blocked := requestWithCookie(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
	"name":        "Missing CSRF Caller",
	"workspaceId": "ws-1",
	"channelType": "local",
	"status":      "active",
}, cookies[0])
if blocked.Code != http.StatusForbidden || !strings.Contains(blocked.Body.String(), "csrf") {
	t.Fatalf("session cookie mutation without csrf should be forbidden, got %d body=%s", blocked.Code, blocked.Body.String())
}
```

Add a helper:

```go
func requestWithCookieAndCSRF(t *testing.T, router http.Handler, method string, path string, body any, cookie *http.Cookie, csrfToken string) *httptest.ResponseRecorder {
	t.Helper()
	rec, req := buildRequest(t, method, path, body, "", "", "")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrfToken != "" {
		req.Header.Set("X-AgentHarbor-CSRF", csrfToken)
	}
	router.ServeHTTP(rec, req)
	return rec
}
```

Use the helper for the successful create and logout calls.

- [x] **Step 2: Confirm backend RED**

Run:

```bash
go test ./internal/httpapi -run 'TestConsoleAuthSessionProtectsManagementEndpoints|TestLocalDevCORS' -count=1
```

Expected: fail because `csrfToken` and `X-AgentHarbor-CSRF` enforcement do not exist yet.

- [x] **Step 3: Implement backend CSRF helpers**

In `internal/httpapi/auth.go`, add:

```go
const consoleSessionCSRFHeader = "X-AgentHarbor-CSRF"
```

Add `CSRFToken string `json:"csrfToken,omitempty"` to `consoleSessionResponse`.

Add helpers:

```go
func consoleSessionTokenFromRequest(r *http.Request) (string, bool)
func (s *Server) consoleSessionCSRFToken(sessionToken string) string
func (s *Server) validateConsoleSessionCSRF(r *http.Request, sessionToken string) error
func requiresCSRFProtection(method string) bool
```

`consoleSessionCSRFToken` must HMAC `csrf:v1:` plus the session token with `s.consoleSessionSecret()`.

- [x] **Step 4: Wire backend enforcement**

In `getAuthSession`, return `csrfToken` when a valid cookie session exists.

In `login`, return `csrfToken` alongside the session response.

In `logout`, require the CSRF header when a valid session cookie is present.

In `requireAdmin`, when `X-Admin-Key` does not authenticate and a console session cookie does authenticate, validate CSRF before serving unsafe methods.

In `localDevCORS`, add `X-AgentHarbor-CSRF` to `Access-Control-Allow-Headers`.

- [x] **Step 5: Confirm backend GREEN**

Run:

```bash
gofmt -w internal/httpapi/auth.go internal/httpapi/server.go internal/httpapi/server_test.go
go test ./internal/httpapi -run 'TestConsoleAuthSessionProtectsManagementEndpoints|TestLocalDevCORS' -count=1
```

Expected: pass.

## Task 2: Frontend Token Plumbing

- [x] **Step 1: Add failing frontend assertions**

In `frontend/tests/api.test.mjs`, assert:

```js
assert.match(apiSource, /csrfToken\?: string/);
assert.match(apiSource, /X-AgentHarbor-CSRF/);
assert.match(apiSource, /setConsoleCsrfToken/);
```

In `frontend/src/types.ts`, add `csrfToken?: string` to `ConsoleSession`.

- [x] **Step 2: Confirm frontend RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/api.test.mjs
```

Expected: fail before `api.ts` implements token storage and header injection.

- [x] **Step 3: Implement frontend API token handling**

In `frontend/src/api.ts`, add module state:

```ts
let consoleCsrfToken = ''

function setConsoleCsrfToken(token?: string) {
  consoleCsrfToken = token?.trim() || ''
}

function requestMethod(options: RequestOptions) {
  return options.method ?? (options.body === undefined ? 'GET' : 'POST')
}

function shouldSendConsoleCsrf(method: string) {
  return method === 'POST' || method === 'PATCH' || method === 'DELETE'
}
```

Use `requestMethod(options)` once per request. When `shouldSendConsoleCsrf(method)` and `consoleCsrfToken` are true, set `headers['X-AgentHarbor-CSRF']`.

After `fetchConsoleSession` and `loginConsole`, call `setConsoleCsrfToken(session.csrfToken)`. After `logoutConsole`, call `setConsoleCsrfToken(session.csrfToken)`, which clears the token for unauthenticated sessions.

- [x] **Step 4: Confirm frontend GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/api.test.mjs
pnpm --dir frontend test
pnpm --dir frontend build
```

Expected: pass.

## Task 3: Docs, Changelog, and Release Gates

- [x] **Step 1: Update changelog**

Add bilingual Unreleased bullets:

```markdown
- Browser console sessions now require a session-bound CSRF header for management writes while API key automation remains unchanged.
- 浏览器控制台会话现在会在管理写操作中校验会话绑定的 CSRF 请求头，API key 自动化调用不受影响。
```

- [x] **Step 2: Run full gates**

Run:

```bash
go test ./internal/httpapi -run 'TestConsoleAuthSessionProtectsManagementEndpoints|TestLocalDevCORS' -count=1
pnpm --dir frontend exec node --test tests/api.test.mjs
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
make check
make release-check
```

Expected: all pass.

- [x] **Step 3: Ship**

Mark all checkboxes, commit, push, open PR, wait for CI, and merge when green.
