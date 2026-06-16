# Management Server Duplicate Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Block duplicate Resource Management mutations at the API layer without storing sensitive Agent key plaintext.

**Architecture:** Add server-side semantic duplicate guards for Agent creation, Agent key creation, credential rotation, and route policy creation. Return `409 DUPLICATE_RESOURCE_MUTATION` for duplicate create requests; make identical credential rotations a no-op. Surface the duplicate mutation error with bilingual Resource Management copy.

**Tech Stack:** Go HTTP API, existing memory/PostgreSQL repositories through current list/get methods, React i18n/tests, Make release gates.

---

## Files

- Create: `docs/superpowers/specs/2026-06-17-management-server-duplicate-guard-design.md`
- Create: `internal/httpapi/management_duplicate_guard.go`
- Modify: `internal/domain/errors.go`
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`
- Modify: `frontend/src/hooks/useManagementOperations.ts`
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/i18n.test.mjs`
- Modify: `frontend/tests/permissionJourneySafety.test.mjs`
- Modify: `CHANGELOG.md`

## Task 1: Backend Duplicate Semantics

- [x] **Step 1: Add failing HTTP API tests**

Add tests in `internal/httpapi/server_test.go` proving:

```go
func TestManagementMutationsRejectDuplicateAgentAndKey(t *testing.T) {
	router := newRouter()
	first := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"tenantId": "tenant-dup",
		"name": "Support Agent",
		"workspaceId": "ws-dup",
		"channelType": "local",
		"status": "active",
	}, "")
	if first.Code != http.StatusCreated {
		t.Fatalf("first agent create failed: %d body=%s", first.Code, first.Body.String())
	}
	created := decodeData[agentResponse](t, first)
	duplicate := request(t, router, http.MethodPost, "/api/v1/agents", map[string]any{
		"tenantId": "tenant-dup",
		"name": " support agent ",
		"workspaceId": "ws-dup",
		"channelType": "local",
		"status": "active",
	}, "")
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), "DUPLICATE_RESOURCE_MUTATION") {
		t.Fatalf("duplicate agent should be a conflict, got %d body=%s", duplicate.Code, duplicate.Body.String())
	}

	firstKey := request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": created.ID,
		"name": "console key",
		"expiresInSeconds": 900,
	}, "")
	if firstKey.Code != http.StatusCreated {
		t.Fatalf("first key create failed: %d body=%s", firstKey.Code, firstKey.Body.String())
	}
	duplicateKey := request(t, router, http.MethodPost, "/api/v1/agent-keys", map[string]any{
		"agentId": created.ID,
		"name": "console key",
		"expiresInSeconds": 900,
	}, "")
	if duplicateKey.Code != http.StatusConflict || !strings.Contains(duplicateKey.Body.String(), "DUPLICATE_RESOURCE_MUTATION") {
		t.Fatalf("duplicate key should be a conflict, got %d body=%s", duplicateKey.Code, duplicateKey.Body.String())
	}
}
```

Also add tests for identical credential rotation returning `200` with unchanged credential version and duplicate route policy returning `409 DUPLICATE_RESOURCE_MUTATION`.

- [x] **Step 2: Run backend test and confirm RED**

Run:

```bash
go test ./internal/httpapi -run 'TestManagementMutationsRejectDuplicate|TestManagementCredentialRotation|TestRoutePolicyCRUDAndAudit' -count=1
```

Expected: duplicate tests fail before guard implementation.

- [x] **Step 3: Add conflict error helper**

In `internal/domain/errors.go`, add:

```go
func Conflict(code, message string) AppError {
	return AppError{Status: 409, Code: code, Message: message}
}
```

- [x] **Step 4: Add duplicate guard helpers**

Create `internal/httpapi/management_duplicate_guard.go` with helpers:

```go
package httpapi

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/SummerXaa-Z/agent-harbor/internal/domain"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

const duplicateResourceMutationCode = "DUPLICATE_RESOURCE_MUTATION"
const duplicateAgentKeyWindow = 2 * time.Minute

func duplicateResourceMutation(message string) domain.AppError {
	return domain.Conflict(duplicateResourceMutationCode, message)
}
```

Implement `rejectDuplicateAgentCreate`, `rejectRecentDuplicateAgentKey`, `sameCredentials`, `rejectDuplicateRoutePolicy`, and `sameRoutePolicyRetry` using existing repository list methods.

- [x] **Step 5: Wire handlers**

In `internal/httpapi/server.go`:

- After `agentFromRequest` and scope validation, call `rejectDuplicateAgentCreate`.
- In `createAgentKey`, trim `req.Name`, compute `now` before generating plaintext, and call `rejectRecentDuplicateAgentKey` before `security.NewAgentKey()`.
- In `rotateAgentCredentials`, after validation and scope check, return current `agent` with `200` if `sameCredentials(agent.Credentials, credentials)`.
- In `createRoutePolicy`, call `rejectDuplicateRoutePolicy` before `CreateRoutePolicyWithAudit`.

- [x] **Step 6: Run backend tests and confirm GREEN**

Run:

```bash
go test ./internal/httpapi -run 'TestManagementMutationsRejectDuplicate|TestManagementCredentialRotation|TestRoutePolicyCRUDAndAudit' -count=1
```

Expected: pass.

## Task 2: Frontend Copy and Regression Guards

- [x] **Step 1: Add frontend failing assertions**

In `frontend/tests/permissionJourneySafety.test.mjs`, assert `useManagementOperations.ts` imports `ApiRequestError`, checks for `DUPLICATE_RESOURCE_MUTATION`, and maps it to `message.duplicateResourceMutation`.

In `frontend/tests/i18n.test.mjs`, include `message.duplicateResourceMutation` in a bilingual key list.

- [x] **Step 2: Implement frontend mapping**

In `frontend/src/hooks/useManagementOperations.ts`, import `ApiRequestError` and update `localizedErrorMessage`:

```ts
if (error instanceof ApiRequestError && error.code === "DUPLICATE_RESOURCE_MUTATION") {
  return t("message.duplicateResourceMutation");
}
```

- [x] **Step 3: Add bilingual copy**

In `frontend/src/i18n.ts`, add:

```ts
"message.duplicateResourceMutation": "This resource change appears to have already been submitted. Refresh the resource list before retrying.",
"message.duplicateResourceMutation": "这次资源变更看起来已经提交过。请先刷新资源列表，再决定是否需要重试。",
```

- [x] **Step 4: Run frontend focused tests**

Run:

```bash
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs tests/i18n.test.mjs
```

Expected: pass.

## Task 3: Docs, Changelog, and Release Gates

- [x] **Step 1: Update changelog**

Add one EN and one zh-CN Unreleased bullet:

```markdown
- Resource Management APIs now reject duplicate Agent, key, and route-policy mutations and treat identical credential rotations as no-op updates, reducing duplicate resources from retries.
- 资源管理 API 现在会拒绝重复的 Agent、密钥和路由规则变更，并将完全相同的凭据轮换视为无变更，降低重试导致的重复资源风险。
```

- [x] **Step 2: Run focused gates**

Run:

```bash
go test ./internal/httpapi -run 'TestManagementMutationsRejectDuplicate|TestManagementCredentialRotation|TestRoutePolicyCRUDAndAudit' -count=1
pnpm --dir frontend exec node --test tests/permissionJourneySafety.test.mjs tests/i18n.test.mjs
```

Expected: pass.

- [x] **Step 3: Run full gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
git diff --check
make check
make release-check
```

Expected: all pass.

- [x] **Step 4: Ship**

Mark all checkboxes, commit, push, open PR, wait for CI, and merge when green.
