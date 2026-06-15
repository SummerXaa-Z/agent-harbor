# API Compatibility Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a public API compatibility contract so the console can fail early when connected to an old or incomplete backend.

**Architecture:** The backend owns the source of truth via `GET /api/v1/system/info`. The frontend health check reads that contract after `/healthz` and maps missing endpoint/capability failures to localized readiness messages.

**Tech Stack:** Go `net/http` + chi router, TypeScript frontend API helpers, Node source tests, Vitest, existing Make gates.

---

### Task 1: Backend Contract Endpoint

**Files:**
- Modify: `internal/httpapi/server.go`
- Test: `internal/httpapi/server_test.go`

- [x] **Step 1: Write the failing backend test**

Add a test that requests `GET /api/v1/system/info` without `X-Admin-Key`, decodes the envelope, and asserts `apiVersion` plus every required capability is present.

- [x] **Step 2: Run the focused backend test**

Run: `go test ./internal/httpapi -run TestSystemInfoIncludesConsoleCompatibilityContract -count=1`

Expected: FAIL with 404 or missing route before implementation.

- [x] **Step 3: Implement the public endpoint**

Add a `systemInfoResponse` type, a stable `systemAPIVersion`, a `systemCapabilities` slice, `s.systemInfo`, and route `GET /api/v1/system/info` outside the admin group.

- [x] **Step 4: Re-run the focused backend test**

Run: `go test ./internal/httpapi -run TestSystemInfoIncludesConsoleCompatibilityContract -count=1`

Expected: PASS.

### Task 2: Frontend Health Compatibility Check

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/ConsoleController.tsx`
- Test: `frontend/tests/api.test.mjs`
- Test: `frontend/tests/i18n.test.mjs`

- [x] **Step 1: Write failing frontend tests**

Add source-level tests that require `/api/v1/system/info`, `requiredConsoleCapabilities`, `api_contract_unavailable`, and `api_contract_incompatible`. Add i18n assertions for both Chinese and English messages.

- [x] **Step 2: Run the focused frontend tests**

Run: `pnpm --dir frontend exec node --test tests/api.test.mjs tests/i18n.test.mjs`

Expected: FAIL before frontend implementation and i18n keys exist.

- [x] **Step 3: Implement frontend contract checks**

Add `SystemInfo`, `fetchSystemInfo`, `missingConsoleCapabilities`, `requiredConsoleCapabilities`, and richer `HealthCheckResult.code`. Update `checkApiHealth` and the readiness detail builder to use localized messages.

- [x] **Step 4: Re-run the focused frontend tests**

Run: `pnpm --dir frontend exec node --test tests/api.test.mjs tests/i18n.test.mjs`

Expected: PASS.

### Task 3: Docs and Gates

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Document the compatibility endpoint**

Update the Health and Contracts README section with `GET /api/v1/system/info` and a bilingual note that the console uses it to detect incompatible APIs before running permission changes.

- [x] **Step 2: Update the changelog**

Add English and Chinese bullets for the API compatibility contract and console readiness behavior.

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
