# Permission Change Command Sheet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Move the Permission Changes draft form out of the always-visible main path and into an explicit command sheet, while keeping the current permission-change status, context, approval, apply, and go-live flow visible.

**Architecture:** Keep backend and permission mutation semantics unchanged. Split the permission draft editor into a focused presentational component in `PermissionWorkbenchParts.tsx`, let `AiAdminPermissionWorkbench` own only sheet open/close state and high-level journey rendering, and guard the shell with existing layout tests plus new source-level tests.

**Tech Stack:** React, TypeScript, existing `ApprovalDropdown`, existing i18n translator, Node test runner, Vite.

---

### Task 1: Add Regression Guard For Sheet-Based Permission Draft Editing

**Files:**
- Modify: `frontend/tests/permissionFlowLayout.test.mjs`

- [x] **Step 1: Write the source-level failing test**

Add a test that requires the permission draft form to be rendered through a dedicated sheet component instead of inline inside the main flow.

```js
test("permission change draft editor opens from a command sheet", () => {
  assert.match(workbench, /useState<\"closed\" \| \"edit\">\\(\"closed\"\\)/);
  assert.match(workbench, /<PermissionChangeDraftSheet/);
  assert.match(workbench, /isOpen=\\{permissionDraftSheet === \"edit\"\\}/);
  assert.match(workbench, /onOpenDraftSheet=\\{\\(\\) => setPermissionDraftSheet\\(\"edit\"\\)\\}/);
  assert.match(workbench, /onClose=\\{\\(\\) => setPermissionDraftSheet\\(\"closed\"\\)\\}/);
  assert.doesNotMatch(workbench, /className=\\{`approval-section approval-request-form-section/);
});
```

- [x] **Step 2: Run focused test and verify it fails**

Run:

```bash
node --test frontend/tests/permissionFlowLayout.test.mjs
```

Expected: FAIL because `PermissionChangeDraftSheet` and `permissionDraftSheet` do not exist yet.

### Task 2: Extract The Draft Form Into A Reusable Sheet Component

**Files:**
- Modify: `frontend/src/components/PermissionWorkbenchParts.tsx`
- Modify: `frontend/src/components/AiAdminPermissionWorkbench.tsx`

- [x] **Step 1: Add the component API**

In `PermissionWorkbenchParts.tsx`, add a `PermissionChangeDraftSheet` component that receives:

```ts
export interface PermissionChangeDraftSheetProps {
  accessSubjectCatalog: AccessSubjectOption[];
  accessSubjectDropdownOptions: Array<{ label: string; value: string }>;
  callerDropdownOptions: Array<{ label: string; value: string }>;
  draft: PermissionPackageDraft;
  form: PermissionPackageDraftInput;
  isLocked: boolean;
  isOpen: boolean;
  onChange: (form: PermissionPackageDraftInput) => void;
  onClose: () => void;
  onOpenDraftSheet: () => void;
  selectedAccessSubject: AccessSubjectOption;
  selectedCaller: Agent | undefined;
  selectedTarget: Agent | undefined;
  templateDropdownOptions: Array<{ label: string; value: string }>;
  targetDropdownOptions: Array<{ label: string; value: string }>;
  tenantDropdownOptions: Array<{ label: string; value: string }>;
  tenantPath: { path: string; primary: string };
  t: Translator;
  workspaceName: string;
}
```

The component renders:
- A compact `.permission-draft-command` button-like summary in the main flow.
- A `.permission-draft-sheet-backdrop` only when `isOpen` is true.
- The existing request text, tenant, workspace, caller, target, access subject, region, permission package, package preview, and technical overrides inside `.permission-draft-sheet`.
- No submit button and no network action.

- [x] **Step 2: Replace the inline form section**

In `AiAdminPermissionWorkbench.tsx`:
- Import `useState` from React if it is not already imported.
- Import `PermissionChangeDraftSheet` from `PermissionWorkbenchParts`.
- Add `const [permissionDraftSheet, setPermissionDraftSheet] = useState<"closed" | "edit">("closed");`.
- Replace the inline `approval-request-form-section` JSX with `<PermissionChangeDraftSheet ... />`.
- Change locked “新建权限变更” actions to call `onStartNewPermissionChange(); setPermissionDraftSheet("edit");`.
- Keep all approval/apply/go-live actions unchanged.

- [x] **Step 3: Run focused test and verify it passes**

Run:

```bash
node --test frontend/tests/permissionFlowLayout.test.mjs
```

Expected: PASS.

### Task 3: Style The Command Summary And Sheet

**Files:**
- Modify: `frontend/src/styles/permission-workbench.css`
- Modify: `frontend/tests/styleTheme.test.mjs`

- [x] **Step 1: Add style guards**

In `frontend/tests/styleTheme.test.mjs`, add checks under the permission workbench layout test:

```js
assert.match(workbench, /className="permission-draft-command"/);
assert.match(workbench, /className="permission-draft-sheet-backdrop"/);
assert.match(workbench, /className="permission-draft-sheet"/);
assert.match(permissionWorkbenchStyles, /\\.permission-draft-command\\s*\\{/);
assert.match(permissionWorkbenchStyles, /\\.permission-draft-sheet-backdrop\\s*\\{/);
assert.match(permissionWorkbenchStyles, /\\.permission-draft-sheet\\s*\\{/);
```

- [x] **Step 2: Run style test and verify it fails**

Run:

```bash
node --test frontend/tests/styleTheme.test.mjs
```

Expected: FAIL until styles and class names exist.

- [x] **Step 3: Add production B-side styling**

Add styles that make the summary visibly actionable:
- `.permission-draft-command` is a full-width button with left context, right action, compact height, brand border on hover, and no card-in-card nesting.
- `.permission-draft-sheet-backdrop` is a fixed overlay aligned to the right.
- `.permission-draft-sheet` is a right sheet with max width around 760px, white surface, scrollable body, header, content, and footer close action.
- Mobile uses full-screen bottom sheet behavior.

- [x] **Step 4: Run style test and verify it passes**

Run:

```bash
node --test frontend/tests/styleTheme.test.mjs
```

Expected: PASS.

### Task 4: Update Copy, Documentation, And Changelog

**Files:**
- Modify: `frontend/src/i18n.ts`
- Modify: `frontend/tests/i18n.test.mjs`
- Add: `docs/engineering/0.2.0-permission-change-command-sheet.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Add bilingual copy**

Add keys:
- `action.editPermissionChangeDraft`
- `action.closePermissionChangeDraft`
- `text.permissionDraftCommandTitle`
- `text.permissionDraftCommandDetail`
- `text.permissionDraftSheetTitle`
- `text.permissionDraftSheetDetail`

- [x] **Step 2: Add i18n assertions**

Assert Chinese labels for the new keys in `frontend/tests/i18n.test.mjs`.

- [x] **Step 3: Write engineering note**

Create `docs/engineering/0.2.0-permission-change-command-sheet.md` with objective, scope, changes, and verification.

- [x] **Step 4: Update changelog**

Add EN + zh-CN Unreleased entries describing the Permission Changes command sheet.

### Task 5: Verify In Browser And Run Gates

**Files:**
- No implementation files unless verification exposes a defect.

- [x] **Step 1: Browser verify the main path**

Open the local console at `http://127.0.0.1:15184/#ai-admin` or the current demo URL. Confirm:
- Main page shows status/context/process without the full draft form open.
- A visible edit/new permission change command opens the sheet.
- Closing the sheet returns to the status page without submitting.

- [x] **Step 2: Run focused tests**

Run:

```bash
node --test frontend/tests/permissionFlowLayout.test.mjs frontend/tests/styleTheme.test.mjs frontend/tests/i18n.test.mjs
```

Expected: PASS.

- [x] **Step 3: Run full frontend and release gates**

Run:

```bash
pnpm --dir frontend test
pnpm --dir frontend build
make check
make release-check
```

Expected: all pass; existing Vite chunk-size warning is acceptable.

- [x] **Step 4: Commit, push, and open PR**

Commit message:

```bash
git commit -m "Move permission change draft into command sheet"
```

Open the PR against `codex/capability-handoff-context`.
