# PR Review Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add lightweight pull request and review governance so future AgentHarbor changes carry clear scope, verification evidence, and review boundaries.

**Architecture:** Add a GitHub pull request template for every future PR and a short engineering review guide for stacked sprint branches. Keep this documentation procedural, not policy-heavy, so it supports iteration without blocking clean-room development.

**Tech Stack:** GitHub pull request template Markdown, repository engineering docs.

---

### Task 1: Add PR Template

**Files:**
- Create: `.github/pull_request_template.md`

- [ ] **Step 1: Create template**

Create `.github/pull_request_template.md` with sections for summary, scope, review boundary, verification, data/security notes, and follow-ups.

- [ ] **Step 2: Verify template content**

Run:

```bash
sed -n '1,220p' .github/pull_request_template.md
```

Expected: template has concrete checklists and no placeholder-only instructions.

### Task 2: Add Review Guide

**Files:**
- Create: `docs/engineering/review-guidelines.md`

- [ ] **Step 1: Create guide**

Create `docs/engineering/review-guidelines.md` explaining review order for sprint stacks, required verification evidence, and when to split a PR.

- [ ] **Step 2: Verify guide content**

Run:

```bash
sed -n '1,240p' docs/engineering/review-guidelines.md
```

Expected: guide is specific to AgentHarbor and references backend, PostgreSQL, frontend, demo, and audit/security review surfaces.

### Task 3: Record and Publish

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Record changelog**

Prepend a session entry describing the PR template and review guide.

- [ ] **Step 2: Verify and commit**

Run:

```bash
git diff --check
git status --short
```

Expected: no whitespace errors and only intended docs/config files changed.

Commit and push:

```bash
git add .github/pull_request_template.md docs/engineering/review-guidelines.md docs/superpowers/plans/2026-05-31-pr-review-governance.md CHANGELOG.md
git commit -m "docs: add pull request review governance"
git push
```

Expected: Draft PR #1 updates and CI runs again.
