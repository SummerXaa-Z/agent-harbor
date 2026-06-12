# Vite Bundle Splitting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split stable frontend vendor code out of the production entry bundle so Vite no longer emits the current 611 KB single-entry chunk warning.

**Architecture:** Keep runtime UI code unchanged and make the fix at the Vite build boundary. Add a static regression test for the Vite config, then configure Vite 8 `build.rolldownOptions.output` chunk groups for React and lucide vendor dependencies. Verify with frontend tests, production build, and release gates.

**Tech Stack:** Vite 8, Rolldown build options, React 19, Node built-in test runner, pnpm, Make.

---

### Task 1: Add Build Config Regression Test

**Files:**
- Create: `frontend/tests/viteConfig.test.mjs`
- Read: `frontend/vite.config.ts`

- [x] **Step 1: Create the failing test**

Create `frontend/tests/viteConfig.test.mjs`:

```js
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const viteConfig = readFileSync(new URL("../vite.config.ts", import.meta.url), "utf8");

test("vite production build keeps the chunk warning budget visible", () => {
  assert.doesNotMatch(viteConfig, /chunkSizeWarningLimit\s*:/);
});

test("vite production build splits stable vendor dependencies", () => {
  assert.match(viteConfig, /rolldownOptions\s*:/);
  assert.match(viteConfig, /output\s*:/);
  assert.match(viteConfig, /react-vendor/);
  assert.match(viteConfig, /icons-vendor/);
  assert.match(viteConfig, /react\|react-dom/);
  assert.match(viteConfig, /lucide-react/);
});
```

- [x] **Step 2: Run the focused test and verify it fails**

Run:

```bash
pnpm --dir frontend exec node --test tests/viteConfig.test.mjs
```

Expected: the new test fails because `frontend/vite.config.ts` has no chunking strategy yet.

### Task 2: Configure Vite Vendor Chunking

**Files:**
- Modify: `frontend/vite.config.ts`
- Test: `frontend/tests/viteConfig.test.mjs`

- [x] **Step 1: Add explicit vendor chunk groups**

Change `frontend/vite.config.ts` to:

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "react-vendor",
              test: /node_modules\/(?:react|react-dom)\//
            },
            {
              name: "icons-vendor",
              test: /node_modules\/lucide-react\//
            }
          ]
        }
      }
    }
  },
  server: {
    host: "127.0.0.1",
    port: 5174
  },
  preview: {
    host: "127.0.0.1",
    port: 4174
  }
});
```

- [x] **Step 2: Run the focused test and verify it passes**

Run:

```bash
pnpm --dir frontend exec node --test tests/viteConfig.test.mjs
```

Expected: the new Vite config test passes.

### Task 3: Verify Production Build Output

**Files:**
- Inspect generated files under: `frontend/dist/assets`

- [x] **Step 1: Run the frontend production build**

Run:

```bash
pnpm --dir frontend build
```

Expected: build exits 0 and no longer emits the previous Vite chunk-size warning for a single 611 KB entry asset.

- [x] **Step 2: Inspect generated JavaScript assets**

Run:

```bash
find frontend/dist/assets -maxdepth 1 -type f -name '*.js' -print | sort | xargs ls -lh
```

Expected: more than one JavaScript asset is emitted, including vendor chunks, and no individual chunk exceeds the default 500 KB warning threshold.

### Task 4: Document The Readiness Follow-Up Closure

**Files:**
- Modify: `docs/engineering/public-readiness-audit-2026-06-12.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Update the public readiness audit follow-up**

In `docs/engineering/public-readiness-audit-2026-06-12.md`, update the remaining Vite warning note to state that the follow-up has been addressed by frontend vendor chunk splitting.

- [x] **Step 2: Update the changelog**

Add an unreleased bullet under the appropriate section:

```markdown
- Split frontend vendor dependencies into dedicated Vite production chunks so the public preview build no longer ships as one oversized JavaScript entry asset.
```

### Task 5: Run Full Verification

**Files:**
- All changed files.

- [x] **Step 1: Run frontend tests**

Run:

```bash
pnpm --dir frontend test
```

Expected: all frontend tests pass.

- [x] **Step 2: Run frontend build**

Run:

```bash
pnpm --dir frontend build
```

Expected: build passes and the previous chunk-size warning is absent.

- [x] **Step 3: Run repository check gate**

Run:

```bash
make check
```

Expected: repository check gate passes.

- [x] **Step 4: Run release gate**

Run:

```bash
make release-check
```

Expected: release gate passes.

- [x] **Step 5: Inspect final diff**

Run:

```bash
git diff --check
git status --short
```

Expected: no whitespace errors; only intended files are changed.
