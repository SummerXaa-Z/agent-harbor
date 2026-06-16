# Console Login Rate Limit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-client throttling for repeated failed console admin-key login attempts.

**Architecture:** Store process-local failure counters on `Server` protected by a mutex. The login handler checks the counter before verifying the admin key, records failed key checks, and clears the counter after a successful login.

**Tech Stack:** Go HTTP API, existing domain error helpers, existing HTTP test helpers, Make release gates.

---

## Files

- Modify: `internal/domain/errors.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/auth.go`
- Modify: `internal/httpapi/server_test.go`
- Modify: `CHANGELOG.md`
- Create: `docs/superpowers/specs/2026-06-17-console-login-rate-limit-design.md`
- Create: `docs/superpowers/plans/2026-06-17-console-login-rate-limit.md`

## Task 1: Backend Login Throttle

- [x] **Step 1: Add failing backend tests**

Add `TestConsoleLoginRateLimit` near the existing console auth tests:

```go
func TestConsoleLoginRateLimit(t *testing.T) {
	router := newRouterWithAdmin("test-admin")

	for i := 0; i < 5; i++ {
		resp := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "")
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("failed login %d should be unauthorized before throttle, got %d body=%s", i+1, resp.Code, resp.Body.String())
		}
	}
	limited := request(t, router, http.MethodPost, "/api/v1/auth/login", map[string]any{"adminKey": "wrong-admin"}, "")
	if limited.Code != http.StatusTooManyRequests || !strings.Contains(limited.Body.String(), "RATE_LIMITED") {
		t.Fatalf("sixth failed login should be rate limited, got %d body=%s", limited.Code, limited.Body.String())
	}

	apiKeyCreate := decodeData[agentResponse](t, requestWithAdmin(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"name":        "API Key Still Works",
		"workspaceId": "ws-login-limit",
		"channelType": "local",
		"status":      "active",
	}, "", "test-admin"))
	if apiKeyCreate.ID == "" {
		t.Fatalf("api key management request should bypass login throttle: %#v", apiKeyCreate)
	}
}
```

Add `TestConsoleLoginRateLimitClearsAfterSuccess` with a router, 4 failed logins, one successful login, then another wrong login that should still return `401` rather than `429`.

- [x] **Step 2: Confirm backend RED**

Run:

```bash
go test ./internal/httpapi -run 'TestConsoleLoginRateLimit|TestConsoleAuthSessionProtectsManagementEndpoints' -count=1
```

Expected: fail because the sixth wrong login is still `401`.

- [x] **Step 3: Add error helper and server state**

In `internal/domain/errors.go`, add:

```go
func TooManyRequests(code, message string) AppError {
	return AppError{Status: 429, Code: code, Message: message}
}
```

In `internal/httpapi/server.go`, import `sync` and add fields to `Server`:

```go
loginFailureMu sync.Mutex
loginFailures  map[string]consoleLoginFailure
```

- [x] **Step 4: Implement auth helpers**

In `internal/httpapi/auth.go`, add constants:

```go
const consoleLoginFailureWindow = 5 * time.Minute
const consoleLoginMaxFailures = 5
```

Add:

```go
type consoleLoginFailure struct {
	Count       int
	WindowEnds time.Time
}

func (s *Server) consoleLoginClientKey(r *http.Request) string
func (s *Server) requireConsoleLoginAllowed(r *http.Request) error
func (s *Server) recordConsoleLoginFailure(r *http.Request)
func (s *Server) clearConsoleLoginFailures(r *http.Request)
```

Use `net.SplitHostPort(r.RemoteAddr)` and fall back to `r.RemoteAddr` when needed. If `X-Forwarded-For` is present, use the first comma-separated IP after trimming.

- [x] **Step 5: Wire login handler**

In `login`, after decoding JSON and before `adminPrincipalForKey`, call `requireConsoleLoginAllowed`.

If `adminPrincipalForKey` fails, call `recordConsoleLoginFailure` before returning `401`.

On successful login, call `clearConsoleLoginFailures` before setting the cookie.

- [x] **Step 6: Confirm backend GREEN**

Run:

```bash
gofmt -w internal/domain/errors.go internal/httpapi/auth.go internal/httpapi/server.go internal/httpapi/server_test.go
go test ./internal/httpapi -run 'TestConsoleLoginRateLimit|TestConsoleAuthSessionProtectsManagementEndpoints' -count=1
```

Expected: pass.

## Task 2: Changelog and Release Gates

- [x] **Step 1: Update changelog**

Add bilingual bullets:

```markdown
- Console login now rate-limits repeated failed admin-key attempts per client while leaving authenticated API key automation unchanged.
- 控制台登录现在会按客户端限制连续失败的管理员密钥尝试，已认证的 API key 自动化调用不受影响。
```

- [x] **Step 2: Run full gates**

Run:

```bash
go test ./internal/httpapi -run 'TestConsoleLoginRateLimit|TestConsoleAuthSessionProtectsManagementEndpoints' -count=1
git diff --check
make check
make release-check
```

Expected: all pass.

- [x] **Step 3: Ship**

Mark all checkboxes, commit, push, open PR, wait for CI, and merge when green.
