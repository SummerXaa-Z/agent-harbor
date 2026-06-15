# Connection Diagnostics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a production-oriented connection diagnostics action that checks login, API compatibility, live data, and MCP tool service readiness from the global Connection popover.

**Architecture:** Backend `system/info` exposes safe auth metadata. Frontend diagnostics rules live in a pure `connectionDiagnostics.ts` model, the async state lives in `useConnectionDiagnostics`, and `ConsoleController` only wires the hook into the existing Connection popover.

**Tech Stack:** Go HTTP API, React/TypeScript, existing Node source tests, Vitest, Make release gates.

---

### Task 1: Backend Auth Metadata

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`

- [x] **Step 1: Write failing backend assertions**

Add `AuthRequired bool` to the test response struct and assert:

```go
if !info.AuthRequired {
    t.Fatalf("system info should report auth required when admin auth is configured: %#v", info)
}
devInfo := decodeData[systemInfoResponse](t, request(t, newRouter(), http.MethodGet, "/api/v1/system/info", nil, ""))
if devInfo.AuthRequired {
    t.Fatalf("dev unauthenticated router should report authRequired=false: %#v", devInfo)
}
```

- [x] **Step 2: Verify backend test fails**

Run: `go test ./internal/httpapi -run TestSystemInfoIncludesConsoleCompatibilityContract -count=1`

Expected: FAIL because `authRequired` is not returned yet.

- [x] **Step 3: Implement `authRequired`**

Add `AuthRequired bool json:"authRequired"` to `systemInfoResponse` and set it from the existing development-bypass check in `systemInfo`.

- [x] **Step 4: Verify backend test passes**

Run: `go test ./internal/httpapi -run TestSystemInfoIncludesConsoleCompatibilityContract -count=1`

Expected: PASS.

### Task 2: Frontend Diagnostics Model and UI

**Files:**
- Create: `frontend/src/connectionDiagnostics.ts`
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/src/styles.css`
- Test: `frontend/tests/connectionDiagnostics.test.mjs`
- Test: `frontend/tests/styleTheme.test.mjs`
- Test: `frontend/tests/i18n.test.mjs`

- [x] **Step 1: Write failing frontend tests**

Add pure model tests for row statuses and summary status. Add source/i18n tests for the connection diagnostics action, list, and bilingual copy.

- [x] **Step 2: Verify frontend tests fail**

Run: `pnpm --dir frontend exec node --test tests/connectionDiagnostics.test.mjs tests/styleTheme.test.mjs tests/i18n.test.mjs`

Expected: FAIL because model, UI hooks, and copy do not exist yet.

- [x] **Step 3: Implement frontend diagnostics**

Create the pure diagnostics model, run `fetchConsoleSession`, `checkApiHealth`, and `checkMockMcpHealth` from `useConnectionDiagnostics`, then render a compact diagnostics list with status tones and last checked time in the connection popover.

- [x] **Step 4: Verify frontend tests pass**

Run: `pnpm --dir frontend exec node --test tests/connectionDiagnostics.test.mjs tests/styleTheme.test.mjs tests/i18n.test.mjs`

Expected: PASS.

### Task 3: Docs and Gates

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Document diagnostics**

Update README connection guidance to mention the Connection popover diagnostic action.

- [x] **Step 2: Update changelog**

Add English and Chinese bullets under Unreleased.

- [x] **Step 3: Run full verification**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
git diff --check
```

Expected: all commands exit 0.
