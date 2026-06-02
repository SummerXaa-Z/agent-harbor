# Tenant Permission Console Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Add a frontend `Access` console view that consumes the tenant access-profile API and explains tenant/workspace/instance permission chains, data scopes, invalid scope reasons, and recent trace evidence.

**Architecture:** Extend the existing React/Vite console in place. Add typed access-profile contracts, a focused API wrapper with fallback sample data, pure helpers for formatting/counting, one new view in `App.tsx`, and CSS that follows the current dense operational console patterns.

**Tech Stack:** React 19, Vite, TypeScript, lucide-react, Node built-in test runner, existing Go API.

---

### Task 1: Access Profile Types, Helpers, and Tests

**Files:**
- Modify: `frontend/src/types.ts`
- Create: `frontend/src/accessProfile.ts`
- Create: `frontend/tests/accessProfile.test.mjs`

- [x] **Step 1: Add frontend response types**

Add `Tenant`, `AccessProfileScopeStatus`, `AccessProfileFilters`, `AccessProfileSummary`, `TenantAccessProfile`, `TenantAccessProfileGrant`, `TenantAccessProfileWorkspace`, `TenantAccessProfileInstance`, and `TenantAccessProfileData` to `frontend/src/types.ts`.

- [x] **Step 2: Add pure helpers**

Create `frontend/src/accessProfile.ts` with helpers for:

- normalizing filters into query params
- validating UI trace limits from text input
- mapping `valid`/`invalid` scope status to UI tones
- summarizing `DataScope[]`
- counting invalid grant/workspace/instance rows

- [x] **Step 3: Add focused Node tests**

Create `frontend/tests/accessProfile.test.mjs` covering filter normalization, trace-limit validation, data-scope summaries, tone mapping, and invalid-row counting.

- [x] **Step 4: Run frontend tests**

Run:

```bash
pnpm --dir frontend test
```

Expected result: tests pass.

### Task 2: API Wrapper and Fallback Profile

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/data.ts`

- [x] **Step 1: Add fallback sample profile**

Create `sampleTenantAccessProfile` in `frontend/src/data.ts` using existing sample agents, capability, entitlement, workspace assignment, instance assignment, and traces.

- [x] **Step 2: Add API loader**

Add `fetchTenantAccessProfile` and `loadTenantAccessProfile` to `frontend/src/api.ts`.

Rules:

- Call `/api/v1/tenants/{tenantId}/access-profile`.
- Encode optional filters with the same query names as the backend.
- Fall back only for fetch/network failures.
- Return `TenantAccessProfileData` with `loadedFromApi` and `apiBase`.

- [x] **Step 3: Run frontend tests again**

Run:

```bash
pnpm --dir frontend test
```

Expected result: tests pass.

### Task 3: React Access View

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/styles.css`

- [x] **Step 1: Add nav and state**

Add `Access` to `navItems`, add profile state, profile filters, loading/message state, and load the profile whenever the view needs refresh.

- [x] **Step 2: Add the Access view**

Create `TenantAccessProfileView` in `App.tsx` with:

- tenant/filter toolbar
- summary strip
- tenant scope list
- effective grant chain table/rows
- recent trace evidence list

- [x] **Step 3: Wire refresh behavior**

The global refresh button should refresh the current console data and, when on `Access`, also refresh the tenant access profile.

- [x] **Step 4: Add responsive CSS**

Add access-specific CSS using existing colors, radius, spacing, and table/list patterns. Ensure mobile layouts collapse to one column.

### Task 4: Verification

**Files:**
- No new source files unless verification exposes defects.

- [x] **Step 1: Run frontend tests**

```bash
pnpm --dir frontend test
```

- [x] **Step 2: Run frontend build**

```bash
pnpm --dir frontend build
```

- [x] **Step 3: Run backend regression tests**

```bash
go test ./...
```

- [x] **Step 4: Inspect git diff**

```bash
git diff --stat
git diff -- frontend/src/App.tsx frontend/src/api.ts frontend/src/types.ts frontend/src/data.ts frontend/src/accessProfile.ts frontend/src/styles.css
```

Confirm the change is limited to the access-profile frontend journey and its docs/tests.
