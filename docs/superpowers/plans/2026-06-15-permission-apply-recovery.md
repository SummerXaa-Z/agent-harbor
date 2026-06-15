# Permission Apply Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make permission package apply retries and post-apply refresh failures recoverable for production operators without weakening one-time approval consumption.

**Architecture:** Preserve the backend store transaction and approval-consumption semantics. Add a stable HTTP error code for consumed approval retries, preserve that code in the frontend API client, and map it to localized recovery behavior in the permission change controller. Extend tests and the approval scenario to lock the contract.

**Tech Stack:** Go HTTP API and tests, TypeScript React controller/API client, Node test runner, Bash scenario gate, Markdown changelog.

---

## File Structure

- Modify `internal/httpapi/server.go`
  - Return `PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED` when an approval request cannot be reused because it is already consumed.
- Modify `internal/httpapi/server_test.go`
  - Assert the stable error code in the existing consumed approval retry path.
- Modify `frontend/src/api.ts`
  - Add `code` to `ApiRequestError` and populate it from envelope `error`.
- Modify `frontend/src/ConsoleController.tsx`
  - Add a small helper to recognize consumed-approval retry errors.
  - In `applyAiAdminPermissionPackage`, show a localized recovery message and best-effort refresh health/readiness when that error occurs.
- Modify `frontend/src/i18n.ts`
  - Add `message.permissionApprovalAlreadyConsumedRecovery` in English and zh-CN.
- Modify frontend source tests
  - `frontend/tests/api.test.mjs`
  - `frontend/tests/permissionJourneySafety.test.mjs`
  - `frontend/tests/i18n.test.mjs`
- Modify `scripts/scenario-permission-package-approval.sh`
  - Assert the stable backend error code after the retry.
- Modify `CHANGELOG.md`
  - Document the production recovery behavior in English and zh-CN.
- Modify this plan while executing
  - Check off each step after verification.

---

### Task 1: Backend Stable Error Contract

**Files:**
- Modify: `internal/httpapi/server_test.go`
- Modify: `internal/httpapi/server.go`

- [ ] **Step 1: Write the failing backend assertion**

In the existing consumed approval retry assertion in `internal/httpapi/server_test.go`, require the stable error code:

```go
if reusedApply.Code != http.StatusBadRequest ||
	!strings.Contains(reusedApply.Body.String(), "PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED") ||
	!strings.Contains(reusedApply.Body.String(), "already consumed") {
	t.Fatalf("consumed approval request should not authorize apply, status=%d body=%s", reusedApply.Code, reusedApply.Body.String())
}
```

- [ ] **Step 2: Run focused backend test and confirm RED**

Run:

```bash
go test ./internal/httpapi -run 'TestPermissionPackage.*Approval|TestPermissionPackageApproval' -count=1
```

Expected: FAIL because consumed approval retry currently returns `VALIDATION_FAILED`.

- [ ] **Step 3: Implement the stable error code**

In `permissionPackageApprovalNotConsumableError`, return:

```go
return domain.BadRequest("PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED", "permission package approval request is already consumed")
```

for the consumed-approval path, while preserving existing validation behavior for not found, status mismatch, expiration, and draft mismatch.

- [ ] **Step 4: Run focused backend test and confirm GREEN**

Run:

```bash
go test ./internal/httpapi -run 'TestPermissionPackage.*Approval|TestPermissionPackageApproval' -count=1
```

Expected: PASS.

---

### Task 2: Scenario Regression Contract

**Files:**
- Modify: `scripts/scenario-permission-package-approval.sh`

- [ ] **Step 1: Add stable code assertion to consumed retry**

After the consumed approval retry request, assert both the code and the human message:

```bash
assert_body_contains "PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED" "consumed approval retry error code"
assert_body_contains "already consumed" "consumed approval retry"
```

- [ ] **Step 2: Run the focused scenario if services are available**

Run:

```bash
make permission-package-approval
```

Expected: PASS when the local API and mock MCP scenario services start correctly.

If local port contention prevents the scenario from starting, record the blocker and rely on `make check` / `make release-check` later because those gates include the scenario scripts in normal project flow.

---

### Task 3: Frontend API Error Code Preservation

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/tests/api.test.mjs`

- [ ] **Step 1: Write source-level test**

Append to `frontend/tests/api.test.mjs`:

```js
test("API request errors preserve backend error codes", () => {
  assert.match(apiSource, /readonly code\\?: string/);
  assert.match(apiSource, /constructor\\(status: number, message: string, code\\?: string\\)/);
  assert.match(apiSource, /new ApiRequestError\\(response\\.status, message \\|\\| `Request failed with status \\$\\{response\\.status\\}`, isEnvelope<.*>\\(payload\\) \\? payload\\.error : undefined\\)/s);
});
```

- [ ] **Step 2: Run frontend API test and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/api.test.mjs
```

Expected: FAIL because `ApiRequestError` does not yet store `code`.

- [ ] **Step 3: Implement code preservation**

Update `ApiRequestError` in `frontend/src/api.ts`:

```ts
export class ApiRequestError extends Error {
  readonly code?: string
  readonly status: number

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = 'ApiRequestError'
    this.code = code
    this.status = status
  }
}
```

Update both non-OK request paths to pass the envelope error code:

```ts
const message = isEnvelope<T>(payload) ? payload.message || payload.error : response.statusText
throw new ApiRequestError(
  response.status,
  message || `Request failed with status ${response.status}`,
  isEnvelope<T>(payload) ? payload.error : undefined,
)
```

and:

```ts
const message = isEnvelope<unknown>(payload) ? payload.message || payload.error : response.statusText
throw new ApiRequestError(
  response.status,
  message || `Request failed with status ${response.status}`,
  isEnvelope<unknown>(payload) ? payload.error : undefined,
)
```

- [ ] **Step 4: Run frontend API test and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/api.test.mjs
```

Expected: PASS.

---

### Task 4: Frontend Recovery Message and Refresh

**Files:**
- Modify: `frontend/src/ConsoleController.tsx`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/permissionJourneySafety.test.mjs`
- Modify: `frontend/tests/i18n.test.mjs`

- [ ] **Step 1: Write frontend safety tests**

Add a test to `frontend/tests/permissionJourneySafety.test.mjs` that checks `applyAiAdminPermissionPackage` detects consumed approval retry before falling back to generic errors:

```js
test("permission apply consumed approval retry shows recovery guidance", () => {
  const block = functionBlock("applyAiAdminPermissionPackage");

  assert.match(app, /function isConsumedApprovalRetryError\\(error: unknown\\)/);
  assert.match(app, /PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED/);
  assert.match(block, /if \\(isConsumedApprovalRetryError\\(error\\)\\)/);
  assert.match(block, /refreshAiAdminApplicationHealth\\(aiAdminForm, \\{ requireLiveApi: false \\}\\)/);
  assert.match(block, /refreshAiAdminProductionReadiness\\(aiAdminForm, \\{ requireLiveApi: false \\}\\)/);
  assert.match(block, /message\\.permissionApprovalAlreadyConsumedRecovery/);
});
```

Add assertions to `frontend/tests/i18n.test.mjs`:

```js
assert.equal(
  t("message.permissionApprovalAlreadyConsumedRecovery"),
  "审批已被使用。请先刷新状态检查或查看当前权限变更，确认是否已应用；不要重复提交。"
);
```

and in the English section:

```js
assert.equal(
  t("message.permissionApprovalAlreadyConsumedRecovery"),
  "This approval has already been used. Refresh status checks or review the current permission change before retrying."
);
```

- [ ] **Step 2: Run focused frontend tests and confirm RED**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs tests/i18n.test.mjs
```

Expected: FAIL because the helper and copy do not exist yet.

- [ ] **Step 3: Implement recovery helper and catch branch**

In `frontend/src/ConsoleController.tsx`, add:

```ts
function isConsumedApprovalRetryError(error: unknown) {
  return error instanceof ApiRequestError && error.code === "PERMISSION_PACKAGE_APPROVAL_ALREADY_CONSUMED";
}
```

In the `catch` block of `applyAiAdminPermissionPackage`, branch before the generic message:

```ts
if (isConsumedApprovalRetryError(error)) {
  await Promise.allSettled([
    refreshAiAdminApplicationHealth(aiAdminForm, { requireLiveApi: false }),
    refreshAiAdminProductionReadiness(aiAdminForm, { requireLiveApi: false }),
    loadAiAdminApprovalRequestsForDraft(aiAdminDraft)
  ]);
  setAiAdminMessage({ key: "message.permissionApprovalAlreadyConsumedRecovery" });
  return;
}
```

Then keep:

```ts
setAiAdminMessage(localizedErrorMessageState(error, "error.applyPermissionPackage"));
```

Add i18n entries in both language maps.

- [ ] **Step 4: Run focused frontend tests and confirm GREEN**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs tests/i18n.test.mjs
```

Expected: PASS.

---

### Task 5: Documentation and Final Verification

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `docs/superpowers/plans/2026-06-15-permission-apply-recovery.md`

- [ ] **Step 1: Update CHANGELOG**

Add a 0.2.0 unreleased bullet in English and zh-CN describing the recoverable consumed-approval retry behavior.

- [ ] **Step 2: Run focused checks**

Run:

```bash
go test ./internal/httpapi -run 'TestPermissionPackage.*Approval|TestPermissionPackageApproval' -count=1
pnpm --dir frontend exec node --test tests/api.test.mjs tests/permissionJourneySafety.test.mjs tests/i18n.test.mjs
```

Expected: PASS.

- [ ] **Step 3: Run full frontend and release gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
git diff --check
```

Expected: PASS.

- [ ] **Step 4: Commit, push, and open PR**

Run:

```bash
git status --short
git add docs/superpowers/specs/2026-06-15-permission-apply-recovery-design.md docs/superpowers/plans/2026-06-15-permission-apply-recovery.md internal/httpapi/server.go internal/httpapi/server_test.go frontend/src/api.ts frontend/src/ConsoleController.tsx frontend/src/i18n.ts frontend/tests/api.test.mjs frontend/tests/permissionJourneySafety.test.mjs frontend/tests/i18n.test.mjs scripts/scenario-permission-package-approval.sh CHANGELOG.md
git commit -m "Improve permission apply recovery"
git push -u origin codex/permission-apply-recovery
gh pr create --title "Improve permission apply recovery" --body "..."
```

Expected: PR created with focused verification evidence.
