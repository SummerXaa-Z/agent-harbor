# Public Readiness Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove and document that a first-time public evaluator can clone AgentHarbor, understand the product, run the local checks, and start the primary console journey without broken setup or misleading release claims.

**Architecture:** This is a release-readiness audit, not a new feature. Work stays in docs, GitHub issue state, and small copy fixes discovered by clean-checkout validation. Runtime behavior changes are out of scope unless the clean public path fails.

**Tech Stack:** GitHub CLI, Make, Go, pnpm/Vite, local shell smoke tests, and the existing AgentHarbor release gates.

---

### Task 1: Align The Public Release Gate

**Files:**
- External: GitHub issue `#43`
- Create: `docs/superpowers/plans/2026-06-12-public-readiness-audit.md`

- [x] **Step 1: Update GitHub issue #43**

Run:

```bash
gh issue edit 43 \
  --repo SummerXaa-Z/agent-harbor \
  --title "release: v0.2.0 public preview readiness gate" \
  --body-file /tmp/issue-43-public-preview.md
```

Expected: issue #43 points to v0.2.0 public preview readiness and no longer describes a stale v0.1.0 gate.

- [x] **Step 2: Add this implementation plan**

Create this file so the audit has a checked-in scope and non-goals before any fixes are made.

### Task 2: Run Fresh Clone Verification

**Files:**
- Modify only if failures are found: `README.md`, `docs/engineering/release-checklist.md`, `.env.example`, or narrow setup docs.
- Do not modify product code unless a clean-checkout failure proves the product is broken.

- [x] **Step 1: Create a temporary clean clone**

Run:

```bash
rm -rf /tmp/agent-harbor-public-readiness
git clone https://github.com/SummerXaa-Z/agent-harbor.git /tmp/agent-harbor-public-readiness
cd /tmp/agent-harbor-public-readiness
git status --short --branch
```

Expected: clean `main` checkout tracking `origin/main`.

- [x] **Step 2: Run the standard local gate**

Run:

```bash
make check
```

Expected: Go, frontend tests, frontend build, script lint, and GitHub config lint pass.

- [x] **Step 3: Run the release gate**

Run:

```bash
make release-check
```

Expected: the full release gate passes from the clean clone.

- [x] **Step 4: Record failures as concrete fixes**

If either gate fails, create a short findings section in the final audit notes with exact command, failure, suspected cause, and file to fix.

### Task 3: Smoke The Browser Demo Path

**Files:**
- Modify only if needed: `README.md`, `docs/engineering/release-checklist.md`, console copy files under `frontend/src/i18n.ts`.

- [x] **Step 1: Start the demo stack from the clean clone**

Run:

```bash
make demo
```

Expected: API, MCP demo service, and web console start using documented local ports.

- [x] **Step 2: Open the web console**

Open:

```text
http://127.0.0.1:5174/
```

Expected: the first screen is coherent for a public evaluator. It should not show stale v0.1.0 language, old evidence terminology, or broken setup prompts.

- [x] **Step 3: Verify the public evaluator message**

Check the visible UI for these claims:

```text
AgentHarbor is tenant-first access governance.
The first task is either Getting Started or Access Query.
Development-mode auth is distinguishable from production-style login/session auth.
```

Expected: a non-project insider can understand the next action without reading raw technical identifiers first.

### Task 4: Apply Narrow Readiness Fixes

**Files:**
- Modify: only files implicated by Task 2 or Task 3.
- Test: matching focused frontend or docs tests when copy/docs change.

- [x] **Step 1: Patch only verified blockers**

Use `apply_patch` for docs and copy fixes. Avoid broad UI refactors in this branch.

- [x] **Step 2: Run focused verification**

Run the smallest relevant command first:

```bash
pnpm --dir frontend test
```

or:

```bash
make github-config-lint
```

Expected: focused verification passes for the changed area.

- [x] **Step 3: Run full gates after fixes**

Run:

```bash
make check
make release-check
```

Expected: both pass locally after any fix.

### Task 5: Publish The Audit Result

**Files:**
- Create: `docs/engineering/public-readiness-audit-2026-06-12.md`
- Modify: `CHANGELOG.md` if user-facing copy or docs changed.

- [x] **Step 1: Write the audit report**

Include:

```markdown
# Public Readiness Audit - 2026-06-12

## Scope
## Fresh Clone Results
## Browser Demo Results
## Fixes Applied
## Remaining Non-Blocking Follow-Ups
## Release Recommendation
```

Expected: the report is factual, short, and tied to commands actually run.

- [x] **Step 2: Update issue #43 with evidence**

Run:

```bash
gh issue comment 43 --repo SummerXaa-Z/agent-harbor --body-file /tmp/public-readiness-evidence.md
```

Expected: #43 has evidence from the clean checkout and browser smoke.

- [x] **Step 3: Commit and open a PR**

Run:

```bash
git add docs/superpowers/plans/2026-06-12-public-readiness-audit.md docs/engineering/public-readiness-audit-2026-06-12.md
git commit -m "docs: add public readiness audit"
git push -u origin codex/public-readiness-audit
gh pr create --repo SummerXaa-Z/agent-harbor --base main --head codex/public-readiness-audit --title "Add public readiness audit" --body-file /tmp/public-readiness-pr.md
```

Expected: PR CI runs against `main` and the PR documents exactly what was verified.
